package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ringsaturn/tzf"
	pb "github.com/ringsaturn/tzf/gen/go/tzf/v1"
	tzfrellite "github.com/ringsaturn/tzf-rel-lite"
	"google.golang.org/protobuf/proto"
)

var finder tzf.F

func init() {
	input := &pb.PreindexTimezones{}
	if err := proto.Unmarshal(tzfrellite.PreindexData, input); err != nil {
		panic("failed to unmarshal preindex data: " + err.Error())
	}
	var err error
	finder, err = tzf.NewFuzzyFinderFromPB(input)
	if err != nil {
		panic("failed to initialize timezone finder: " + err.Error())
	}
}

func (s *Server) airportTimeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	icao := r.URL.Query().Get("icao")

	if icao == "" {
		http.Error(w, "icao parameter is required", http.StatusBadRequest)
		return
	}

	if !isValidICAOCode(icao) {
		http.Error(w, "Invalid ICAO code - must be 4 letters", http.StatusBadRequest)
		return
	}

	airport, err := s.getAirportByICAO(icao)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Airport not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Error retrieving airport", http.StatusInternalServerError)
		return
	}

	timezoneName := finder.GetTimezoneName(airport.LongitudeDeg, airport.LatitudeDeg)
	if timezoneName == "" {
		http.Error(w, "Could not determine timezone for airport location", http.StatusInternalServerError)
		return
	}

	loc, err := time.LoadLocation(timezoneName)
	if err != nil {
		http.Error(w, "Error loading timezone", http.StatusInternalServerError)
		return
	}

	now := time.Now().In(loc)

	response := AirportTimeResponse{
		ICAO:      airport.IcaoCode,
		Name:      airport.Name,
		Timezone:  timezoneName,
		LocalTime: now.Format(time.RFC3339),
		UTCOffset: now.Format("-07:00"),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
