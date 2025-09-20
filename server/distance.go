package server

import (
	"math"
)

const (
	// Earth's radius in nautical miles
	earthRadiusNM = 3440.065
)

// calculateDistance computes the great circle distance between two points using the haversine formula
// Returns distance in nautical miles
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	dLat := lat2Rad - lat1Rad
	dLon := lon2Rad - lon1Rad

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := earthRadiusNM * c
	return distance
}

// getAirportByICAO retrieves airport information by ICAO code
func (s *Server) getAirportByICAO(icao string) (*Airport, error) {
	query := `SELECT id, ident, type, name, latitude_deg, longitude_deg, 
			  COALESCE(NULLIF(elevation_ft, ''), 0) as elevation_ft, continent, 
			  iso_country, iso_region, municipality, scheduled_service, 
			  icao_code, iata_code, gps_code, local_code, home_link, 
			  wikipedia_link, keywords 
			  FROM airports WHERE UPPER(icao_code) = UPPER(?)`

	var airport Airport
	var (
		id               int
		ident            string
		airportType      string
		name             string
		latitudeDeg      float64
		longitudeDeg     float64
		elevationFt      int
		continent        string
		isoCountry       string
		isoRegion        string
		municipality     string
		scheduledService string
		icaoCode         string
		iataCode         string
		gpsCode          string
		localCode        string
		homeLink         string
		wikipediaLink    string
		keywords         string
	)

	err := s.db.QueryRow(query, icao).Scan(
		&id, &ident, &airportType, &name, &latitudeDeg, &longitudeDeg,
		&elevationFt, &continent, &isoCountry, &isoRegion, &municipality,
		&scheduledService, &icaoCode, &iataCode, &gpsCode, &localCode,
		&homeLink, &wikipediaLink, &keywords,
	)

	if err != nil {
		return nil, err
	}

	airport = Airport{
		ID:               id,
		Ident:            ident,
		Type:             airportType,
		Name:             name,
		LatitudeDeg:      latitudeDeg,
		LongitudeDeg:     longitudeDeg,
		ElevationFt:      elevationFt,
		Continent:        continent,
		IsoCountry:       isoCountry,
		IsoRegion:        isoRegion,
		Municipality:     municipality,
		ScheduledService: scheduledService,
		IcaoCode:         icaoCode,
		IataCode:         iataCode,
		GpsCode:          gpsCode,
		LocalCode:        localCode,
		HomeLink:         homeLink,
		WikipediaLink:    wikipediaLink,
		Keywords:         keywords,
	}

	return &airport, nil
}
