package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

const VERSION = "v0.1"

type VersionResponse struct {
	Version string `json:"version"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type Airport struct {
	ID               int     `json:"id"`
	Ident            string  `json:"ident"`
	Type             string  `json:"type"`
	Name             string  `json:"name"`
	LatitudeDeg      float64 `json:"latitude_deg"`
	LongitudeDeg     float64 `json:"longitude_deg"`
	ElevationFt      int     `json:"elevation_ft"`
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

type Country struct {
	ID            int    `json:"id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Continent     string `json:"continent"`
	WikipediaLink string `json:"wikipedia_link"`
	Keywords      string `json:"keywords"`
}

type SearchResponse struct {
	Airports []Airport `json:"airports"`
	Count    int       `json:"count"`
}

type CountryListResponse struct {
	Countries []Country `json:"countries"`
	Count     int       `json:"count"`
}

func (s *Server) versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := VersionResponse{
		Version: VERSION,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := HealthResponse{
		Status: "OK",
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

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

// isValidSearchParameter validates that the search parameter contains only allowed characters
func isValidSearchParameter(param string) bool {
	// Allow letters, spaces, hyphens, apostrophes, and common punctuation for airport names
	matched, _ := regexp.MatchString(`^[a-zA-Z\s\-'\.]+$`, param)
	return matched
}

// isValidCountryCode validates that the country code contains only letters
func isValidCountryCode(code string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z]+$`, code)
	return matched
}

func (s *Server) countryListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Query to get all countries from the countries table
	query := "SELECT id, code, name, continent, wikipedia_link, keywords FROM countries ORDER BY name"

	rows, err := s.db.Query(query)
	if err != nil {
		http.Error(w, "Database query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var countries []Country
	for rows.Next() {
		var (
			id            sql.NullInt64
			code          sql.NullString
			name          sql.NullString
			continent     sql.NullString
			wikipediaLink sql.NullString
			keywords      sql.NullString
		)

		err := rows.Scan(&id, &code, &name, &continent, &wikipediaLink, &keywords)
		if err != nil {
			http.Error(w, "Error scanning database results", http.StatusInternalServerError)
			return
		}

		country := Country{
			ID:            int(id.Int64),
			Code:          code.String,
			Name:          name.String,
			Continent:     continent.String,
			WikipediaLink: wikipediaLink.String,
			Keywords:      keywords.String,
		}

		countries = append(countries, country)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Error processing database results", http.StatusInternalServerError)
		return
	}

	response := CountryListResponse{
		Countries: countries,
		Count:     len(countries),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}
