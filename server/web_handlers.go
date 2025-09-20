package server

import (
	"html/template"
	"net/http"
	"path/filepath"
)

type IndexPageData struct {
	Version string
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

	// Prepare template data with version
	data := IndexPageData{
		Version: VERSION,
	}

	// Execute the template
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}
