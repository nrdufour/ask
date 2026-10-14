package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"ask/db"

	_ "github.com/mattn/go-sqlite3"
)

// Real coordinates, trimmed to the columns the code reads. Empty fields
// become NULL, as in a real import.
const fixtureAirports = `id,ident,type,name,latitude_deg,longitude_deg,elevation_ft,iso_country,municipality,icao_code,iata_code,gps_code
1,LFPG,large_airport,Charles de Gaulle International Airport,49.0097,2.5479,392,FR,Paris,LFPG,CDG,LFPG
2,LFPO,large_airport,Paris-Orly Airport,48.7233,2.3794,291,FR,Paris,LFPO,ORY,LFPO
3,LSZH,large_airport,Zürich Airport,47.4581,8.5481,1416,CH,Zürich,LSZH,ZRH,LSZH
4,FVHA,large_airport,Robert Gabriel Mugabe International Airport,-17.9318,31.0928,4887,ZW,Harare,,HRE,FVHA
5,ZW-0001,closed,Old Harare Field,-17.8,31.0,,ZW,Harare,,,FVHA
6,NFFN,large_airport,Nadi International Airport,-17.7554,177.4431,59,FJ,Nadi,NFFN,NAN,NFFN
7,NSFA,medium_airport,Faleolo International Airport,-13.83,-172.008,58,WS,Apia,NSFA,APW,NSFA
8,ENSB,medium_airport,Svalbard Airport Longyear,78.2461,15.4656,88,NO,Longyearbyen,ENSB,LYR,ENSB
9,CYLT,small_airport,Alert Airport,82.5178,-62.2806,100,CA,Alert,CYLT,YLT,CYLT
10,BGTL,medium_airport,Thule Air Base,76.5312,-68.7032,251,GL,Pituffik,BGTL,THU,BGTL
11,XX-0001,heliport,100% Pure_Heliport,49.0,2.5,,FR,Paris,,,
`

func newTestServer(t *testing.T) *Server {
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
	if _, err := db.ImportCSV(database, "airports", strings.NewReader(fixtureAirports)); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateIndexes(database); err != nil {
		t.Fatal(err)
	}
	return &Server{db: database}
}

func idents(airports []Airport) []string {
	var out []string
	for _, a := range airports {
		out = append(out, a.Ident)
	}
	return out
}

func TestGetAirportByCode(t *testing.T) {
	s := newTestServer(t)

	tests := []struct{ code, want string }{
		{"LFPG", "LFPG"},
		{"lfpg", "LFPG"},
		// No icao_code: found by ident, ahead of the closed duplicate that
		// shares its GPS code.
		{"FVHA", "FVHA"},
	}
	for _, tt := range tests {
		a, err := s.getAirportByCode(tt.code)
		if err != nil {
			t.Errorf("%s: %v", tt.code, err)
			continue
		}
		if a.Ident != tt.want {
			t.Errorf("%s: got %s, want %s", tt.code, a.Ident, tt.want)
		}
	}

	if _, err := s.getAirportByCode("ZZZZ"); err != sql.ErrNoRows {
		t.Errorf("ZZZZ: err = %v, want sql.ErrNoRows", err)
	}
}

func TestSearchAirports(t *testing.T) {
	s := newTestServer(t)

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
		got, truncated, err := s.searchAirports(tt.query, tt.country, 10)
		if err != nil {
			t.Errorf("%s: %v", tt.name, err)
			continue
		}
		if truncated {
			t.Errorf("%s: unexpectedly truncated", tt.name)
		}
		if !slices.Equal(idents(got), tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, idents(got), tt.want)
		}
	}
}

func TestSearchAirportsLimit(t *testing.T) {
	s := newTestServer(t)

	got, truncated, err := s.searchAirports("paris", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !truncated {
		t.Errorf("got %d airports, truncated=%v; want 2, true", len(got), truncated)
	}

	_, truncated, _ = s.searchAirports("paris", "", 3)
	if truncated {
		t.Error("exactly 3 matches with limit 3 should not be truncated")
	}
}

// getAirportsInRange must agree with a brute-force distance check over the
// whole table, including across the antimeridian and over the pole.
func TestGetAirportsInRangeMatchesBruteForce(t *testing.T) {
	s := newTestServer(t)

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
		origin, err := s.getAirportByCode(tt.origin)
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.getAirportsInRange(origin, tt.rangeNM, nil)
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

func TestSearchHandler(t *testing.T) {
	s := newTestServer(t)

	tests := []struct {
		query string
		code  int
	}{
		{"name=Z%C3%BCrich", http.StatusOK},
		{"name=nowhere", http.StatusOK},
		{"", http.StatusBadRequest},
		{"name=a", http.StatusBadRequest},
		{"name=paris&country=FRA", http.StatusBadRequest},
		{"name=paris&limit=0", http.StatusBadRequest},
		{"name=paris&limit=501", http.StatusBadRequest},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		s.airportSearchHandler(rec, httptest.NewRequest("GET", "/api/airport/search?"+tt.query, nil))
		if rec.Code != tt.code {
			t.Errorf("%q: status %d, want %d (%s)", tt.query, rec.Code, tt.code, rec.Body)
		}
	}

	// No match is an empty array, not null.
	rec := httptest.NewRecorder()
	s.airportSearchHandler(rec, httptest.NewRequest("GET", "/api/airport/search?name=nowhere", nil))
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if string(resp["airports"]) != "[]" {
		t.Errorf("airports = %s, want []", resp["airports"])
	}
}
