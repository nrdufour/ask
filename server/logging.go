package server

import (
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware logs HTTP request details including method, path, query parameters, and response time
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process the request
		next.ServeHTTP(wrapped, r)

		// Log the request details with structured format
		duration := time.Since(start)
		queryString := r.URL.RawQuery
		if queryString == "" {
			queryString = "-"
		}

		log.Printf("HTTP Request: method=%s path=%s qs=%s remote_addr=%s status=%d duration=%v user_agent=%s",
			r.Method,
			r.URL.Path,
			queryString,
			r.RemoteAddr,
			wrapped.statusCode,
			duration,
			r.UserAgent(),
		)
	})
}

// responseWriter is a wrapper around http.ResponseWriter to capture the status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
