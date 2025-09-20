package server

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type IndexPageData struct {
	Version      string
	ImportStatus []ImportStatus
}

func (s *Server) airportsPageHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the template
	tmplPath := filepath.Join("templates", "airports.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// Set content type to HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute the template
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (s *Server) distancePageHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the template
	tmplPath := filepath.Join("templates", "distance.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// Set content type to HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute the template
	if err := tmpl.Execute(w, nil); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (s *Server) indexPageHandler(w http.ResponseWriter, r *http.Request) {
	// Parse the template
	tmplPath := filepath.Join("templates", "index.html")
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}

	// Set content type to HTML
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Fetch import status data - handle case where database or table doesn't exist
	var importStatus []ImportStatus
	if s.db != nil {
		query := `SELECT table_name, last_import_date, git_commit_hash, git_commit_date, record_count 
				  FROM import_status 
				  ORDER BY table_name`

		rows, err := s.db.Query(query)
		if err != nil {
			// If table doesn't exist, continue with empty status - don't error
			// This happens when database hasn't been initialized yet
			importStatus = []ImportStatus{}
		} else {
			defer rows.Close()

			for rows.Next() {
				var status ImportStatus
				err := rows.Scan(&status.TableName, &status.LastImportDate, &status.GitCommitHash, &status.GitCommitDate, &status.RecordCount)
				if err != nil {
					http.Error(w, "Failed to scan row", http.StatusInternalServerError)
					return
				}
				importStatus = append(importStatus, status)
			}

			if err = rows.Err(); err != nil {
				http.Error(w, "Row iteration failed", http.StatusInternalServerError)
				return
			}
		}
	}

	// Prepare template data with version and import status
	data := IndexPageData{
		Version:      VERSION,
		ImportStatus: importStatus,
	}

	// Execute the template
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}
