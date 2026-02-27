// Package middleware provides reusable HTTP middleware for the API server.
package middleware

import (
	"log"
	"net/http"
	"time"
)

// wrappedWriter captures the status code written by downstream handlers.
type wrappedWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *wrappedWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Logger logs method, path, status code, and duration for every request.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &wrappedWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, ww.statusCode, time.Since(start))
	})
}
