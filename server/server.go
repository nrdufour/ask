package server

import (
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"time"

	"ask/db"
	"ask/store"
	"ask/web"
)

type Server struct {
	port      int
	mux       *http.ServeMux
	server    *http.Server
	store     *store.Store
	tiles     *tiles
	pages     map[string]*template.Template
	assetHash string
}

// NewServer serves the database; tilesPath, when set, is where the basemap
// lives (see tiles.go).
func NewServer(port int, tilesPath string) (*Server, error) {
	s := &Server{
		port:  port,
		mux:   http.NewServeMux(),
		tiles: newTiles(tilesPath),
	}

	st, err := store.Open(db.Path())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	s.store = st

	s.pages, err = parsePages()
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	s.assetHash, err = assetHash()
	if err != nil {
		return nil, fmt.Errorf("failed to hash static assets: %w", err)
	}

	s.setupRoutes()

	s.server = &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      LoggingMiddleware(s.mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("GET /version", s.versionHandler)
	s.mux.HandleFunc("GET /health", s.healthHandler)
	s.mux.HandleFunc("GET /api/airport/search", s.airportSearchHandler)
	s.mux.HandleFunc("GET /api/airport/distance", s.distanceHandler)
	s.mux.HandleFunc("GET /api/airport/time", s.airportTimeHandler)
	s.mux.HandleFunc("GET /api/airport/reachable", s.reachableHandler)
	s.mux.HandleFunc("GET /api/airport/{code}", s.airportDetailHandler)
	s.mux.HandleFunc("GET /api/country", s.countryListHandler)
	s.mux.HandleFunc("GET /api/import/status", s.importStatusHandler)

	s.mux.HandleFunc("GET /tiles/basemap.pmtiles", s.tilesHandler)

	// Static files
	static, _ := fs.Sub(web.FS, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(static)))

	// Web routes
	s.mux.HandleFunc("GET /{$}", s.pageHandler("index"))
	s.mux.HandleFunc("GET /airports", s.pageHandler("airports"))
	s.mux.HandleFunc("GET /distance", s.pageHandler("distance"))
	s.mux.HandleFunc("GET /reachable", s.pageHandler("reachable"))
	s.mux.HandleFunc("GET /airport/{code}", s.airportPageHandler)
}

func (s *Server) Start() error {
	log.Printf("Starting server on port %d", s.port)
	s.tiles.refreshIfStale(context.Background())
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	log.Println("Stopping server...")
	err := s.server.Shutdown(ctx)
	s.store.Close()
	return err
}
