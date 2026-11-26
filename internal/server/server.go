// Package server wires the HTTP router and handlers.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Pinger reports whether the database is reachable.
type Pinger interface {
	Ping(ctx context.Context) (int32, error)
}

func NewRouter(logger *slog.Logger, db Pinger) chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.Recoverer, requestLogger(logger))
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/readyz", func(w http.ResponseWriter, req *http.Request) {
		if _, err := db.Ping(req.Context()); err != nil {
			logger.Error("readiness check failed", "error", err)
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	return r
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, req.ProtoMajor)
			next.ServeHTTP(ww, req)
			logger.Info("request",
				"method", req.Method,
				"path", req.URL.Path,
				"status", ww.Status(),
				"duration", time.Since(start),
				"request_id", middleware.GetReqID(req.Context()),
			)
		})
	}
}
