package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
)

func (s *Server) airportSearchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get query parameters
	name := r.URL.Query().Get("name")
	country := r.URL.Query().Get("country")

	// Validate that name parameter is provided
	if name == "" {
		http.Error(w, "Name parameter is required", http.StatusBadRequest)
		return
	}

	// Sanitize parameters - only accept letters, spaces, hyphens, apostrophes
	if !isValidSearchParameter(name) {
		http.Error(w, "Invalid name parameter - only letters, spaces, hyphens, and apostrophes are allowed", http.StatusBadRequest)
		return
	}

	if country != "" && !isValidCountryCode(country) {
		http.Error(w, "Invalid country parameter - only letters are allowed", http.StatusBadRequest)
		return
	}

	// Build SQL query
	var query strings.Builder
	var args []interface{}

	query.WriteString("SELECT id, ident, type, name, latitude_deg, longitude_deg, COALESCE(NULLIF(elevation_ft, ''), 0) as elevation_ft, continent, iso_country, iso_region, municipality, scheduled_service, icao_code, iata_code, gps_code, local_code, home_link, wikipedia_link, keywords FROM airports WHERE LOWER(name) LIKE LOWER(?)")
	args = append(args, "%"+name+"%")

	if country != "" {
		query.WriteString(" AND LOWER(iso_country) = LOWER(?)")
		args = append(args, country)
	}

	// Execute query
	rows, err := s.db.Query(query.String(), args...)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var airports []Airport
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

		err := rows.Scan(
			&id,
			&ident,
			&airportType,
			&name,
			&latitudeDeg,
			&longitudeDeg,
			&elevationFt,
			&continent,
			&isoCountry,
			&isoRegion,
			&municipality,
			&scheduledService,
			&icaoCode,
			&iataCode,
			&gpsCode,
			&localCode,
			&homeLink,
			&wikipediaLink,
			&keywords,
		)
		if err != nil {
			http.Error(w, "Error scanning database results", http.StatusInternalServerError)
			return
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

		airports = append(airports, airport)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Error processing database results", http.StatusInternalServerError)
		return
	}

	response := SearchResponse{
		Airports: airports,
		Count:    len(airports),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
