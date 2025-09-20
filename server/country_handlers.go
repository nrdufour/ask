package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
)

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
