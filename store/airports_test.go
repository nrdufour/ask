package store

import (
	"database/sql"
	"errors"
	"os"
	"slices"
	"testing"

	"ask/db"

	_ "github.com/mattn/go-sqlite3"
)

// testdata/airports.csv holds real coordinates, trimmed to the columns the
// code reads. Empty fields become NULL, as in a real import.

// newTestStore returns a store over an in-memory copy of the fixture.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	// One connection: each new :memory: connection would be a fresh database.
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { database.Close() })

	if err := db.CreateSchema(database); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.Open("testdata/airports.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	if _, err := db.ImportCSV(database, "airports", fixture); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateIndexes(database); err != nil {
		t.Fatal(err)
	}
	return New(database)
}

func idents(airports []Airport) []string {
	var out []string
	for _, a := range airports {
		out = append(out, a.Ident)
	}
	return out
}

func TestGetAirportByCode(t *testing.T) {
	s := newTestStore(t)

	tests := []struct{ code, want string }{
		{"LFPG", "LFPG"},
		{"lfpg", "LFPG"},
		// No icao_code: found by ident, ahead of the closed duplicate that
		// shares its GPS code.
		{"FVHA", "FVHA"},
	}
	for _, tt := range tests {
		a, err := s.ByCode(tt.code)
		if err != nil {
			t.Errorf("%s: %v", tt.code, err)
			continue
		}
		if a.Ident != tt.want {
			t.Errorf("%s: got %s, want %s", tt.code, a.Ident, tt.want)
		}
	}

	if _, err := s.ByCode("ZZZZ"); !errors.Is(err, ErrNotFound) {
		t.Errorf("ZZZZ: err = %v, want ErrNotFound", err)
	}

	var inputErr *InputError
	for _, code := range []string{"", "LFP", "LFPG1", "12AB"} {
		if _, err := s.ByCode(code); !errors.As(err, &inputErr) {
			t.Errorf("%q: err = %v, want an InputError", code, err)
		}
	}
}

func TestSearchAirports(t *testing.T) {
	s := newTestStore(t)

	tests := []struct {
		name, query, country string
		want                 []string
	}{
		{"non-ASCII name", "Zürich", "", []string{"LSZH"}},
		{"city", "paris", "", []string{"LFPG", "LFPO", "XX-0001"}},
		{"country filter", "paris", "fr", []string{"LFPG", "LFPO", "XX-0001"}},
		{"country excludes", "paris", "CH", nil},
		{"ICAO code", "lfpo", "", []string{"LFPO"}},
		{"IATA code", "HRE", "", []string{"FVHA"}},
		// The exact code match ranks ahead of name matches.
		{"code match first", "FVHA", "", []string{"FVHA", "ZW-0001"}},
		{"digits and punctuation", "100% Pure", "", []string{"XX-0001"}},
		// LIKE wildcards in the query are literal.
		{"literal underscore", "e_h", "", []string{"XX-0001"}},
		{"literal percent", "%%", "", nil},
	}
	for _, tt := range tests {
		got, err := s.Search(tt.query, tt.country, 10)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if got.Truncated {
			t.Errorf("%s: unexpectedly truncated", tt.name)
		}
		if !slices.Equal(idents(got.Airports), tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, idents(got.Airports), tt.want)
		}
	}
}

func TestSearchAirportsLimit(t *testing.T) {
	s := newTestStore(t)

	got, err := s.Search("paris", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.Count != 2 || !got.Truncated {
		t.Errorf("got %d airports, truncated=%v; want 2, true", got.Count, got.Truncated)
	}

	got, _ = s.Search("paris", "", 3)
	if got.Truncated {
		t.Error("exactly 3 matches with limit 3 should not be truncated")
	}
}

// getAirportsInRange must agree with a brute-force distance check over the
// whole table, including across the antimeridian and over the pole.
func TestGetAirportsInRangeMatchesBruteForce(t *testing.T) {
	s := newTestStore(t)

	rows, err := s.db.Query(`SELECT ` + airportColumns + ` FROM airports`)
	if err != nil {
		t.Fatal(err)
	}
	all, err := scanAirports(rows)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		origin  string
		rangeNM float64
	}{
		{"LFPG", 50},
		{"LFPG", 500},
		{"NFFN", 700},  // reaches NSFA across the antimeridian
		{"NSFA", 700},  // and back
		{"ENSB", 1000}, // over the pole to CYLT and BGTL
		{"CYLT", 300},
		{"FVHA", 10800},
	}
	for _, tt := range tests {
		origin, err := s.ByCode(tt.origin)
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.airportsInRange(origin, tt.rangeNM, nil)
		if err != nil {
			t.Fatal(err)
		}

		var want []string
		for _, a := range all {
			if a.ID != origin.ID && calculateDistance(origin.LatitudeDeg, origin.LongitudeDeg, a.LatitudeDeg, a.LongitudeDeg) <= tt.rangeNM {
				want = append(want, a.Ident)
			}
		}

		var gotIdents []string
		for _, r := range got {
			gotIdents = append(gotIdents, r.Airport.Ident)
		}
		slices.Sort(gotIdents)
		slices.Sort(want)
		if !slices.Equal(gotIdents, want) {
			t.Errorf("%s within %.0f NM: got %v, want %v", tt.origin, tt.rangeNM, gotIdents, want)
		}
	}
}

func TestInputValidation(t *testing.T) {
	s := newTestStore(t)

	calls := map[string]func() error{
		"one-letter search": func() error { _, err := s.Search("a", "", 10); return err },
		"3-letter country":  func() error { _, err := s.Search("paris", "FRA", 10); return err },
		"zero limit":        func() error { _, err := s.Search("paris", "", 0); return err },
		"limit too high":    func() error { _, err := s.Search("paris", "", MaxSearchLimit+1); return err },
		"zero range":        func() error { _, err := s.InRange("LFPG", 0, nil); return err },
		"range too long":    func() error { _, err := s.InRange("LFPG", MaxRangeNM+1, nil); return err },
		"unknown type":      func() error { _, err := s.InRange("LFPG", 100, []string{"spaceport"}); return err },
		"bad route code":    func() error { _, err := s.Route("LFPG", "KJF"); return err },
	}
	for name, call := range calls {
		var inputErr *InputError
		if err := call(); !errors.As(err, &inputErr) {
			t.Errorf("%s: err = %v, want an InputError", name, err)
		}
	}
}

func TestRoute(t *testing.T) {
	s := newTestStore(t)

	r, err := s.Route("lfpg", "FVHA")
	if err != nil {
		t.Fatal(err)
	}
	if r.DepartureAirport.Ident != "LFPG" || r.DestinationAirport.Ident != "FVHA" {
		t.Errorf("route %s -> %s", r.DepartureAirport.Ident, r.DestinationAirport.Ident)
	}
	if r.DistanceKM != r.DistanceNM*1.852 {
		t.Errorf("km = %f for %f NM", r.DistanceKM, r.DistanceNM)
	}
	// Paris to Harare heads south-south-east.
	if r.InitialBearingDeg < 140 || r.InitialBearingDeg > 160 {
		t.Errorf("initial bearing = %.1f, want ~151", r.InitialBearingDeg)
	}

	if _, err := s.Route("LFPG", "ZZZZ"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown destination: err = %v, want ErrNotFound", err)
	}
}

func TestLocalTime(t *testing.T) {
	s := newTestStore(t)

	// NFFN sits right by the antimeridian; ENSB is in the high Arctic.
	for code, zone := range map[string]string{"LFPG": "Europe/Paris", "NFFN": "Pacific/Fiji", "ENSB": "Arctic/Longyearbyen", "FVHA": "Africa/Harare"} {
		got, err := s.LocalTime(code)
		if err != nil {
			t.Errorf("%s: %v", code, err)
			continue
		}
		if got.Timezone != zone {
			t.Errorf("%s: timezone %s, want %s", code, got.Timezone, zone)
		}
	}
}
