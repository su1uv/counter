package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestRouter() http.Handler {
	return NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
}

func TestRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{"healthz", http.MethodGet, "/healthz", http.StatusOK, "ok"},
		{"unknown path", http.MethodGet, "/nope", http.StatusNotFound, ""},
		{"wrong method", http.MethodPost, "/healthz", http.StatusMethodNotAllowed, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newTestRouter().ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantBody != "" && rec.Body.String() != tt.wantBody {
				t.Fatalf("body = %q, want %q", rec.Body.String(), tt.wantBody)
			}
		})
	}
}

func TestRecoversFromPanic(t *testing.T) {
	r := NewRouter(slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	r.Get("/boom", func(http.ResponseWriter, *http.Request) { panic("boom") })
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/boom", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) (int32, error) { return 1, f.err }

func TestReadyz(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantLog    string
	}{
		{"database up", nil, http.StatusOK, ""},
		{"database down", errors.New("connection refused"), http.StatusServiceUnavailable, "connection refused"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			r := NewRouter(slog.New(slog.NewTextHandler(&logs, nil)), fakePinger{tt.err})
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantLog != "" && !bytes.Contains(logs.Bytes(), []byte(tt.wantLog)) {
				t.Fatalf("logs %q lack %q", logs.String(), tt.wantLog)
			}
		})
	}
}
