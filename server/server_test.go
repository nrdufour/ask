package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"ask/db"
	"ask/store"

	_ "github.com/mattn/go-sqlite3"
)

// newTestServer serves the store fixture through the real routes.
func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	database, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { database.Close() })

	if err := db.CreateSchema(database); err != nil {
		t.Fatal(err)
	}
	fixture, err := os.Open("../store/testdata/airports.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	if _, err := db.ImportCSV(database, "airports", fixture); err != nil {
		t.Fatal(err)
	}

	s := &Server{mux: http.NewServeMux(), store: store.New(database)}
	if s.pages, err = parsePages(); err != nil {
		t.Fatal(err)
	}
	if s.assetHash, err = assetHash(); err != nil {
		t.Fatal(err)
	}
	s.setupRoutes()
	return s.mux
}

func get(t *testing.T, h http.Handler, url string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
	return rec
}

func TestRoutes(t *testing.T) {
	h := newTestServer(t)

	tests := []struct {
		url  string
		code int
	}{
		{"/", 200},
		{"/airports", 200},
		{"/distance", 200},
		{"/reachable", 200},
		{"/static/app.css", 200},
		{"/static/vendor/leaflet/leaflet.js", 200},
		{"/nope", 404},
		{"/health", 200},
		{"/version", 200},
		{"/api/country", 200},
		{"/api/import/status", 200},

		{"/api/airport/search?name=Z%C3%BCrich", 200},
		{"/api/airport/search?name=nowhere", 200},
		{"/api/airport/search", 400},
		{"/api/airport/search?name=a", 400},
		{"/api/airport/search?name=paris&country=FRA", 400},
		{"/api/airport/search?name=paris&limit=0", 400},
		{"/api/airport/search?name=paris&limit=many", 400},

		{"/api/airport/distance?departure=LFPG&destination=FVHA", 200},
		{"/api/airport/distance?departure=LFPG&destination=ZZZZ", 404},
		{"/api/airport/distance?departure=LFPG", 400},
		{"/api/airport/distance?departure=LFPG&destination=EGL", 400},

		{"/api/airport/reachable?icao=LFPG&range=500", 200},
		{"/api/airport/reachable?icao=LFPG&range=500&type=large_airport,%20heliport", 200},
		{"/api/airport/reachable?icao=LFPG", 400},
		{"/api/airport/reachable?icao=LFPG&range=-5", 400},
		{"/api/airport/reachable?icao=LFPG&range=20000", 400},
		{"/api/airport/reachable?icao=LFPG&range=500&type=spaceport", 400},
		{"/api/airport/reachable?icao=ZZZZ&range=500", 404},

		{"/api/airport/time?icao=FVHA", 200},
		{"/api/airport/time?icao=ZZZZ", 404},
		{"/api/airport/time", 400},
	}
	for _, tt := range tests {
		if rec := get(t, h, tt.url); rec.Code != tt.code {
			t.Errorf("%s: status %d, want %d (%s)", tt.url, rec.Code, tt.code, strings.TrimSpace(rec.Body.String()))
		}
	}
}

func TestJSONShapes(t *testing.T) {
	h := newTestServer(t)

	// No match is an empty array, not null.
	var search map[string]json.RawMessage
	if err := json.Unmarshal(get(t, h, "/api/airport/search?name=nowhere").Body.Bytes(), &search); err != nil {
		t.Fatal(err)
	}
	if string(search["airports"]) != "[]" {
		t.Errorf("airports = %s, want []", search["airports"])
	}

	var route store.Route
	if err := json.Unmarshal(get(t, h, "/api/airport/distance?departure=nffn&destination=NSFA").Body.Bytes(), &route); err != nil {
		t.Fatal(err)
	}
	if route.DepartureAirport.Ident != "NFFN" || route.DistanceNM < 500 || route.DistanceNM > 700 {
		t.Errorf("NFFN-NSFA = %+v", route)
	}

	var inRange store.InRangeResult
	if err := json.Unmarshal(get(t, h, "/api/airport/reachable?icao=NFFN&range=700").Body.Bytes(), &inRange); err != nil {
		t.Fatal(err)
	}
	if inRange.Count != 1 || inRange.Airports[0].Airport.Ident != "NSFA" {
		t.Errorf("700 NM from NFFN = %+v, want NSFA across the antimeridian", inRange.Airports)
	}
}

func TestIndexLinksHashedAssets(t *testing.T) {
	h := newTestServer(t)
	body := get(t, h, "/").Body.String()
	if !strings.Contains(body, "/static/app.css?v=") {
		t.Error("index page does not link the hashed stylesheet")
	}
}
