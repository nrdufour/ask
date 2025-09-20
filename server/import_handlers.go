package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) importStatusHandler(w http.ResponseWriter, r *http.Request) {
	var tables []ImportStatus

	if s.db != nil {
		query := `SELECT table_name, last_import_date, git_commit_hash, git_commit_date, record_count 
				  FROM import_status 
				  ORDER BY table_name`

		rows, err := s.db.Query(query)
		if err != nil {
			// If table doesn't exist, return empty array - don't error
			tables = []ImportStatus{}
		} else {
			defer rows.Close()

			for rows.Next() {
				var status ImportStatus
				err := rows.Scan(&status.TableName, &status.LastImportDate, &status.GitCommitHash, &status.GitCommitDate, &status.RecordCount)
				if err != nil {
					http.Error(w, "Failed to scan row", http.StatusInternalServerError)
					return
				}
				tables = append(tables, status)
			}

			if err = rows.Err(); err != nil {
				http.Error(w, "Row iteration failed", http.StatusInternalServerError)
				return
			}
		}
	}

	response := ImportStatusResponse{
		Tables: tables,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
