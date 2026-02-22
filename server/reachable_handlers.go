package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func (s *Server) reachableHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	icao := r.URL.Query().Get("icao")
	rangeStr := r.URL.Query().Get("range")

	if icao == "" {
		http.Error(w, "icao parameter is required", http.StatusBadRequest)
		return
	}

	if rangeStr == "" {
		http.Error(w, "range parameter is required", http.StatusBadRequest)
		return
	}

	if !isValidICAOCode(icao) {
		http.Error(w, "Invalid ICAO code - must be 4 letters", http.StatusBadRequest)
		return
	}

	rangeNM, ok := isValidRange(rangeStr)
	if !ok {
		http.Error(w, "Invalid range - must be a positive number up to 10800 NM", http.StatusBadRequest)
		return
	}

	typesStr := r.URL.Query().Get("type")
	types, validTypes := parseAirportTypes(typesStr)
	if !validTypes {
		http.Error(w, "Invalid airport type - valid types are: large_airport, medium_airport, small_airport, heliport, seaplane_base, closed, balloonport", http.StatusBadRequest)
		return
	}

	origin, err := s.getAirportByICAO(icao)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Airport not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error retrieving airport", http.StatusInternalServerError)
		return
	}

	airports, err := s.getAirportsInRange(origin, rangeNM, types)
	if err != nil {
		http.Error(w, "Error querying reachable airports", http.StatusInternalServerError)
		return
	}

	response := ReachableResponse{
		OriginAirport: *origin,
		RangeNM:       rangeNM,
		Airports:      airports,
		Count:         len(airports),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
