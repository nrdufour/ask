package server

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"ask/store"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encoding response: %v", err)
	}
}

// writeError answers with the status matching err: 400 for bad input, 404
// for an unknown airport, 500 for anything else, whose detail stays in the log.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var inputErr *store.InputError
	switch {
	case errors.As(err, &inputErr):
		http.Error(w, inputErr.Error(), http.StatusBadRequest)
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		log.Printf("%s %s: %v", r.Method, r.URL, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (s *Server) versionHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, VersionResponse{Version: VERSION})
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, HealthResponse{Status: "OK"})
}

func (s *Server) airportSearchHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := store.DefaultSearchLimit
	if l := q.Get("limit"); l != "" {
		n, err := strconv.Atoi(l)
		if err != nil {
			http.Error(w, "the limit must be a whole number", http.StatusBadRequest)
			return
		}
		limit = n
	}

	result, err := s.store.Search(q.Get("name"), q.Get("country"), limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, result)
}

func (s *Server) distanceHandler(w http.ResponseWriter, r *http.Request) {
	route, err := s.store.Route(r.URL.Query().Get("departure"), r.URL.Query().Get("destination"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, route)
}

func (s *Server) reachableHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	rangeNM, err := strconv.ParseFloat(q.Get("range"), 64)
	if err != nil {
		http.Error(w, "the range must be a number of nautical miles", http.StatusBadRequest)
		return
	}

	var types []string
	for _, t := range strings.Split(q.Get("type"), ",") {
		if t = strings.TrimSpace(t); t != "" {
			types = append(types, t)
		}
	}

	result, err := s.store.InRange(q.Get("icao"), rangeNM, types)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, result)
}

func (s *Server) airportTimeHandler(w http.ResponseWriter, r *http.Request) {
	t, err := s.store.LocalTime(r.URL.Query().Get("icao"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, t)
}

func (s *Server) countryListHandler(w http.ResponseWriter, r *http.Request) {
	countries, err := s.store.Countries()
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, CountryListResponse{Countries: countries, Count: len(countries)})
}

func (s *Server) importStatusHandler(w http.ResponseWriter, r *http.Request) {
	tables, err := s.store.ImportStatus()
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, ImportStatusResponse{Tables: tables})
}
