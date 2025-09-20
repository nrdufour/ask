package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

func (s *Server) distanceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get query parameters
	departureICAO := r.URL.Query().Get("departure")
	destinationICAO := r.URL.Query().Get("destination")

	// Validate parameters
	if departureICAO == "" {
		http.Error(w, "departure parameter is required", http.StatusBadRequest)
		return
	}

	if destinationICAO == "" {
		http.Error(w, "destination parameter is required", http.StatusBadRequest)
		return
	}

	// Validate ICAO code format (4 letters)
	if !isValidICAOCode(departureICAO) {
		http.Error(w, "Invalid departure ICAO code - must be 4 letters", http.StatusBadRequest)
		return
	}

	if !isValidICAOCode(destinationICAO) {
		http.Error(w, "Invalid destination ICAO code - must be 4 letters", http.StatusBadRequest)
		return
	}

	// Get departure airport
	departureAirport, err := s.getAirportByICAO(departureICAO)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Departure airport not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error retrieving departure airport", http.StatusInternalServerError)
		return
	}

	// Get destination airport
	destinationAirport, err := s.getAirportByICAO(destinationICAO)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Destination airport not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error retrieving destination airport", http.StatusInternalServerError)
		return
	}

	// Calculate distance
	distance := calculateDistance(
		departureAirport.LatitudeDeg, departureAirport.LongitudeDeg,
		destinationAirport.LatitudeDeg, destinationAirport.LongitudeDeg,
	)

	// Create response
	response := DistanceResponse{
		DepartureAirport:   *departureAirport,
		DestinationAirport: *destinationAirport,
		DistanceNM:         distance,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
