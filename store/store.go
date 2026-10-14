// Package store answers airport questions from the SQLite database: lookups,
// search, distances, airports in range and local times. The web server and
// the CLI both use it, and its result types are what the JSON API returns.
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"unicode"
	"unicode/utf8"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ringsaturn/tzf/v2"
)

const (
	DefaultSearchLimit = 100
	MaxSearchLimit     = 500
	// MaxRangeNM is half the Earth's circumference: every point is closer.
	MaxRangeNM = 10800
)

// ErrNotFound is returned, wrapped, when no airport has the requested code.
var ErrNotFound = errors.New("airport not found")

// InputError reports a request the store cannot answer as asked: a malformed
// code, an out-of-bounds range. Its message is fit to show the user.
type InputError struct{ msg string }

func (e *InputError) Error() string { return e.msg }

func inputErrorf(format string, args ...any) error {
	return &InputError{fmt.Sprintf(format, args...)}
}

type Store struct {
	db       *sql.DB
	tzFinder func() (tzf.F, error)
}

// Open opens the database at path read-only.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", "file:"+path+"?mode=ro")
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return New(db), nil
}

// New wraps an open database.
func New(db *sql.DB) *Store {
	return &Store{
		db:       db,
		tzFinder: sync.OnceValues(tzf.NewDefaultFinder),
	}
}

func (s *Store) Close() error { return s.db.Close() }

type Airport struct {
	ID               int     `json:"id"`
	Ident            string  `json:"ident"`
	Type             string  `json:"type"`
	Name             string  `json:"name"`
	LatitudeDeg      float64 `json:"latitude_deg"`
	LongitudeDeg     float64 `json:"longitude_deg"`
	ElevationFt      *int    `json:"elevation_ft"`
	Continent        string  `json:"continent"`
	IsoCountry       string  `json:"iso_country"`
	IsoRegion        string  `json:"iso_region"`
	Municipality     string  `json:"municipality"`
	ScheduledService string  `json:"scheduled_service"`
	IcaoCode         string  `json:"icao_code"`
	IataCode         string  `json:"iata_code"`
	GpsCode          string  `json:"gps_code"`
	LocalCode        string  `json:"local_code"`
	HomeLink         string  `json:"home_link"`
	WikipediaLink    string  `json:"wikipedia_link"`
	Keywords         string  `json:"keywords"`
}

// Code is the code to show for the airport: its ICAO code, or its
// OurAirports ident when it has none.
func (a Airport) Code() string {
	if a.IcaoCode != "" {
		return a.IcaoCode
	}
	return a.Ident
}

type Country struct {
	ID            int    `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Continent     string `json:"continent"`
	WikipediaLink string `json:"wikipedia_link"`
	Keywords      string `json:"keywords"`
}

type ImportStatus struct {
	TableName      string `json:"table_name"`
	LastImportDate string `json:"last_import_date"`
	GitCommitHash  string `json:"git_commit_hash"`
	GitCommitDate  string `json:"git_commit_date"`
	RecordCount    int    `json:"record_count"`
}

type SearchResult struct {
	Airports []Airport `json:"airports"`
	Count    int       `json:"count"`
	// Truncated is set when more airports matched than the limit allowed.
	Truncated bool `json:"truncated"`
}

type Route struct {
	DepartureAirport   Airport `json:"departure_airport"`
	DestinationAirport Airport `json:"destination_airport"`
	DistanceNM         float64 `json:"distance_nm"`
	DistanceKM         float64 `json:"distance_km"`
	DistanceSM         float64 `json:"distance_sm"`
	// InitialBearingDeg is the true course at departure along the great circle.
	InitialBearingDeg float64 `json:"initial_bearing_deg"`
}

type ReachableAirport struct {
	Airport    Airport `json:"airport"`
	DistanceNM float64 `json:"distance_nm"`
}

type InRangeResult struct {
	OriginAirport Airport            `json:"origin_airport"`
	RangeNM       float64            `json:"range_nm"`
	Airports      []ReachableAirport `json:"airports"`
	Count         int                `json:"count"`
}

type AirportTime struct {
	ICAO      string `json:"icao"`
	Name      string `json:"name"`
	Timezone  string `json:"timezone"`
	LocalTime string `json:"local_time"`
	UTCOffset string `json:"utc_offset"`
}

// AirportTypes are the OurAirports airport types, largest first.
var AirportTypes = []string{
	"large_airport", "medium_airport", "small_airport",
	"heliport", "seaplane_base", "balloonport", "closed",
}

var (
	icaoCodeRe    = regexp.MustCompile(`^[a-zA-Z]{4}$`)
	countryCodeRe = regexp.MustCompile(`^[a-zA-Z]{2}$`)
)

func checkCode(code string) error {
	if code == "" {
		return inputErrorf("an ICAO code is required")
	}
	if !icaoCodeRe.MatchString(code) {
		return inputErrorf("%q is not a 4-letter ICAO code", code)
	}
	return nil
}

// checkQuery accepts 2 to 100 printable characters in any script. Queries are
// bound as SQL parameters, so no character needs filtering for safety; the
// bounds only keep the scan meaningful and cheap.
func checkQuery(q string) error {
	n := utf8.RuneCountInString(q)
	if n < 2 || n > 100 {
		return inputErrorf("the search must be 2 to 100 characters long")
	}
	for _, r := range q {
		if !unicode.IsPrint(r) {
			return inputErrorf("the search contains unprintable characters")
		}
	}
	return nil
}

func checkCountry(code string) error {
	if code != "" && !countryCodeRe.MatchString(code) {
		return inputErrorf("%q is not a 2-letter ISO country code", code)
	}
	return nil
}

func checkLimit(limit int) error {
	if limit < 1 || limit > MaxSearchLimit {
		return inputErrorf("the limit must be from 1 to %d", MaxSearchLimit)
	}
	return nil
}

func checkRange(rangeNM float64) error {
	if !(rangeNM > 0 && rangeNM <= MaxRangeNM) {
		return inputErrorf("the range must be more than 0 and at most %d NM", MaxRangeNM)
	}
	return nil
}

func checkTypes(types []string) error {
	for _, t := range types {
		known := false
		for _, k := range AirportTypes {
			known = known || t == k
		}
		if !known {
			return inputErrorf("unknown airport type %q - valid types are %v", t, AirportTypes)
		}
	}
	return nil
}
