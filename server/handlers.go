package server

import (
	"encoding/json"
	"net/http"
)

const VERSION = "v0.1"

type VersionResponse struct {
	Version string `json:"version"`
}

type HealthResponse struct {
	Status string `json:"status"`
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