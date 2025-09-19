package server

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type Server struct {
	port   int
	router *mux.Router
	server *http.Server
}

func NewServer(port int) *Server {
	s := &Server{
		port:   port,
		router: mux.NewRouter(),
	}
	
	s.setupRoutes()
	
	s.server = &http.Server{
		Addr:         ":" + strconv.Itoa(port),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	return s
}

func (s *Server) setupRoutes() {
	s.router.HandleFunc("/version", s.versionHandler).Methods("GET")
	s.router.HandleFunc("/health", s.healthHandler).Methods("GET")
}

func (s *Server) Start() error {
	log.Printf("Starting server on port %d", s.port)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	log.Println("Stopping server...")
	return s.server.Shutdown(ctx)
}