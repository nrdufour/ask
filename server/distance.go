package server

import (
	"database/sql"
	"math"
	"sort"
	"strings"
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

// getAirportsInRange finds all airports within rangeNM nautical miles of the origin airport.
// If types is non-empty, only airports matching those types are returned.
func (s *Server) getAirportsInRange(origin *Airport, rangeNM float64, types []string) ([]ReachableAirport, error) {
	// Compute bounding box: 1 deg lat ~ 60 NM, 1 deg lon ~ 60*cos(lat) NM
	latDelta := rangeNM / 60.0
	cosLat := math.Cos(origin.LatitudeDeg * math.Pi / 180)
	if cosLat < 0.01 {
		cosLat = 0.01 // avoid division by zero near poles
	}
	lonDelta := rangeNM / (60.0 * cosLat)

	minLat := origin.LatitudeDeg - latDelta
	maxLat := origin.LatitudeDeg + latDelta
	minLon := origin.LongitudeDeg - lonDelta
	maxLon := origin.LongitudeDeg + lonDelta

	var query string
	var args []interface{}

	selectCols := `SELECT id, ident, type, name, latitude_deg, longitude_deg,
		COALESCE(NULLIF(elevation_ft, ''), 0) as elevation_ft, continent,
		iso_country, iso_region, municipality, scheduled_service,
		icao_code, iata_code, gps_code, local_code, home_link,
		wikipedia_link, keywords FROM airports WHERE latitude_deg BETWEEN ? AND ?`

	if minLon < -180 || maxLon > 180 {
		// Antimeridian crossing: split into two longitude ranges
		wrappedMinLon := minLon
		wrappedMaxLon := maxLon
		if minLon < -180 {
			wrappedMinLon = minLon + 360
		}
		if maxLon > 180 {
			wrappedMaxLon = maxLon - 360
		}
		query = selectCols + " AND (longitude_deg >= ? OR longitude_deg <= ?)"
		if minLon < -180 {
			args = []interface{}{minLat, maxLat, wrappedMinLon, maxLon}
		} else {
			args = []interface{}{minLat, maxLat, minLon, wrappedMaxLon}
		}
	} else {
		query = selectCols + " AND longitude_deg BETWEEN ? AND ?"
		args = []interface{}{minLat, maxLat, minLon, maxLon}
	}

	if len(types) > 0 {
		placeholders := strings.Repeat("?,", len(types))
		placeholders = placeholders[:len(placeholders)-1] // trim trailing comma
		query += " AND type IN (" + placeholders + ")"
		for _, t := range types {
			args = append(args, t)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ReachableAirport
	for rows.Next() {
		var (
			id               sql.NullInt64
			ident            sql.NullString
			airportType      sql.NullString
			name             sql.NullString
			latitudeDeg      sql.NullFloat64
			longitudeDeg     sql.NullFloat64
			elevationFt      sql.NullInt64
			continent        sql.NullString
			isoCountry       sql.NullString
			isoRegion        sql.NullString
			municipality     sql.NullString
			scheduledService sql.NullString
			icaoCode         sql.NullString
			iataCode         sql.NullString
			gpsCode          sql.NullString
			localCode        sql.NullString
			homeLink         sql.NullString
			wikipediaLink    sql.NullString
			keywords         sql.NullString
		)

		if err := rows.Scan(
			&id, &ident, &airportType, &name, &latitudeDeg, &longitudeDeg,
			&elevationFt, &continent, &isoCountry, &isoRegion, &municipality,
			&scheduledService, &icaoCode, &iataCode, &gpsCode, &localCode,
			&homeLink, &wikipediaLink, &keywords,
		); err != nil {
			return nil, err
		}

		airport := Airport{
			ID:               int(id.Int64),
			Ident:            ident.String,
			Type:             airportType.String,
			Name:             name.String,
			LatitudeDeg:      latitudeDeg.Float64,
			LongitudeDeg:     longitudeDeg.Float64,
			ElevationFt:      int(elevationFt.Int64),
			Continent:        continent.String,
			IsoCountry:       isoCountry.String,
			IsoRegion:        isoRegion.String,
			Municipality:     municipality.String,
			ScheduledService: scheduledService.String,
			IcaoCode:         icaoCode.String,
			IataCode:         iataCode.String,
			GpsCode:          gpsCode.String,
			LocalCode:        localCode.String,
			HomeLink:         homeLink.String,
			WikipediaLink:    wikipediaLink.String,
			Keywords:         keywords.String,
		}

		dist := calculateDistance(origin.LatitudeDeg, origin.LongitudeDeg, airport.LatitudeDeg, airport.LongitudeDeg)
		if dist > 0.01 && dist <= rangeNM {
			// Round to 1 decimal place
			dist = math.Round(dist*10) / 10
			results = append(results, ReachableAirport{
				Airport:    airport,
				DistanceNM: dist,
			})
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].DistanceNM < results[j].DistanceNM
	})

	return results, nil
}
