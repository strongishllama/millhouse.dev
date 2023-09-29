package app

import (
	log "log/slog"
	"net/http"
	"time"
)

func NewRequestDuration(handler http.Handler) *RequestDuration {
	return &RequestDuration{
		handler: handler,
	}
}

type RequestDuration struct {
	handler http.Handler
}

func (rd *RequestDuration) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	rw := &responseWriter{
		ResponseWriter: w,
		StatusCode:     http.StatusOK,
	}

	rd.handler.ServeHTTP(rw, r)

	log.Info(
		"incoming request",
		"duration (ns)", time.Since(startTime),
		"method", r.Method,
		"statusCode", rw.StatusCode,
		"url", r.URL.Path,
	)
}

type responseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (r *responseWriter) WriteHeader(statusCode int) {
	r.StatusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func NewMethodCheck(handler http.Handler) *MethodCheck {
	return &MethodCheck{
		handler: handler,
	}
}

type MethodCheck struct {
	handler http.Handler
}

func (m *MethodCheck) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	m.handler.ServeHTTP(w, r)
}
