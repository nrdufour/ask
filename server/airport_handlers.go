package server

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) airportSearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	name := strings.TrimSpace(r.URL.Query().Get("name"))
	country := r.URL.Query().Get("country")

	if name == "" {
		http.Error(w, "name parameter is required", http.StatusBadRequest)
		return
	}

	if !isValidSearchQuery(name) {
		http.Error(w, "Invalid name parameter - must be 2 to 100 printable characters", http.StatusBadRequest)
		return
	}

	if country != "" && !isValidCountryCode(country) {
		http.Error(w, "Invalid country parameter - must be a 2-letter ISO country code", http.StatusBadRequest)
		return
	}

	limit, ok := parseSearchLimit(r.URL.Query().Get("limit"))
	if !ok {
		http.Error(w, "Invalid limit parameter - must be an integer from 1 to 500", http.StatusBadRequest)
		return
	}

	airports, truncated, err := s.searchAirports(name, country, limit)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}

	response := SearchResponse{
		Airports:  airports,
		Count:     len(airports),
		Truncated: truncated,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
