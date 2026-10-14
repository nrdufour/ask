package server

import (
	"encoding/json"
	"net/http"
)

func (s *Server) importStatus() ([]ImportStatus, error) {
	rows, err := s.db.Query(`SELECT table_name, last_import_date, git_commit_hash, git_commit_date, record_count
		FROM import_status
		ORDER BY table_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := []ImportStatus{}
	for rows.Next() {
		var status ImportStatus
		if err := rows.Scan(&status.TableName, &status.LastImportDate, &status.GitCommitHash, &status.GitCommitDate, &status.RecordCount); err != nil {
			return nil, err
		}
		tables = append(tables, status)
	}
	return tables, rows.Err()
}

func (s *Server) importStatusHandler(w http.ResponseWriter, r *http.Request) {
	tables, err := s.importStatus()
	if err != nil {
		http.Error(w, "Failed to read import status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ImportStatusResponse{Tables: tables}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
