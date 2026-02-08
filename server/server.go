package server

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/viper"
)

type Server struct {
	port   int
	router *mux.Router
	server *http.Server
	db     *sql.DB
}

func NewServer(port int) *Server {
	s := &Server{
		port:   port,
		router: mux.NewRouter(),
	}

	// Initialize database connection
	if err := s.initDatabase(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	s.setupRoutes()

	s.server = &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      LoggingMiddleware(s.router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) setupRoutes() {
	// API routes
	s.router.HandleFunc("/version", s.versionHandler).Methods("GET")
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
	s.router.HandleFunc("/api/airport/search", s.airportSearchHandler).Methods("GET")
	s.router.HandleFunc("/api/airport/distance", s.distanceHandler).Methods("GET")
	s.router.HandleFunc("/api/airport/time", s.airportTimeHandler).Methods("GET")
	s.router.HandleFunc("/api/country", s.countryListHandler).Methods("GET")
	s.router.HandleFunc("/api/import/status", s.importStatusHandler).Methods("GET")

	// Web routes
	s.router.HandleFunc("/", s.indexPageHandler).Methods("GET")
	s.router.HandleFunc("/airports", s.airportsPageHandler).Methods("GET")
	s.router.HandleFunc("/distance", s.distancePageHandler).Methods("GET")
}

func (s *Server) Start() error {
	log.Printf("Starting server on port %d", s.port)
	return s.server.ListenAndServe()
}

func (s *Server) initDatabase() error {
	repoDir := viper.GetString("repository")
	dbDir := filepath.Join(repoDir, viper.GetString("db"))
	dbPath := filepath.Join(dbDir, "ask.db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return err
	}

	s.db = db
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	log.Println("Stopping server...")
	if s.db != nil {
		s.db.Close()
	}
	return s.server.Shutdown(ctx)
}
