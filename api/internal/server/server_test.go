package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/config"
	"github.com/samandar-hodiev/sat-platform/api/internal/health"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
)

func testRouter(t *testing.T, env map[string]string, dbUp bool) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	base := map[string]string{"DATABASE_URL": "postgres://localhost/sat_test"}
	for k, v := range env {
		base[k] = v
	}
	cfg, err := config.FromLookup(func(k string) (string, bool) { v, ok := base[k]; return v, ok })
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewRouter(Deps{
		Config:  cfg,
		Log:     slog.New(slog.NewTextHandler(io.Discard, nil)),
		Limiter: ratelimit.NewMemory(),
		Checks: map[string]health.Checker{"database": func(context.Context) error {
			if dbUp {
				return nil
			}
			return errors.New("down")
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func do(r http.Handler, req *http.Request) (*httptest.ResponseRecorder, map[string]any) {
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w, body
}

func TestProbes(t *testing.T) {
	r := testRouter(t, nil, false)
	if w, _ := do(r, httptest.NewRequest(http.MethodGet, "/health", nil)); w.Code != http.StatusOK {
		t.Errorf("/health = %d", w.Code)
	}
	if w, _ := do(r, httptest.NewRequest(http.MethodGet, "/ready", nil)); w.Code != http.StatusServiceUnavailable {
		t.Errorf("/ready with database down = %d, want 503", w.Code)
	}
}

func TestUnknownRouteUsesEnvelope(t *testing.T) {
	w, body := do(testRouter(t, nil, true), httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil))
	if w.Code != http.StatusNotFound || body["success"] != false {
		t.Fatalf("%d %v", w.Code, body)
	}
	e := body["error"].(map[string]any)
	if e["code"] != "NOT_FOUND" || e["request_id"] == "" {
		t.Errorf("error = %v", e)
	}
	if w.Header().Get("X-Request-ID") == "" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("request id and security headers must be set on every response")
	}
}

func TestCORSAllowsOnlyConfiguredOrigins(t *testing.T) {
	r := testRouter(t, map[string]string{"CORS_ALLOWED_ORIGINS": "http://localhost:3000"}, true)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	if w, _ := do(r, req); w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("configured origin not allowed")
	}

	req = httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://evil.example")
	if w, _ := do(r, req); w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("foreign origin allowed")
	}

	req = httptest.NewRequest(http.MethodOptions, "/api/v1/openapi.yaml", nil)
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	if w, _ := do(r, req); w.Code != http.StatusForbidden {
		t.Errorf("foreign preflight = %d, want 403", w.Code)
	}

	// Native apps send no Origin: requests work, no CORS headers.
	if w, _ := do(r, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)); w.Code != http.StatusOK {
		t.Errorf("request without Origin = %d", w.Code)
	}
}

func TestRateLimitOnAPI(t *testing.T) {
	r := testRouter(t, map[string]string{"RATE_LIMIT_PER_MINUTE": "2"}, true)
	for i := 0; i < 2; i++ {
		if w, _ := do(r, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil)); w.Code != http.StatusOK {
			t.Fatalf("request %d = %d", i+1, w.Code)
		}
	}
	w, body := do(r, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil))
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Fatalf("3rd request = %d (Retry-After %q)", w.Code, w.Header().Get("Retry-After"))
	}
	if body["error"].(map[string]any)["code"] != "RATE_LIMITED" {
		t.Errorf("error = %v", body["error"])
	}
	// Probes are not rate limited: orchestrators poll them constantly.
	if w, _ := do(r, httptest.NewRequest(http.MethodGet, "/health", nil)); w.Code != http.StatusOK {
		t.Errorf("/health rate limited: %d", w.Code)
	}
}

func TestBodyLimit(t *testing.T) {
	r := testRouter(t, map[string]string{"HTTP_MAX_BODY_BYTES": "10"}, true)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/anything", strings.NewReader(strings.Repeat("x", 100)))
	w, body := do(r, req)
	if w.Code != http.StatusRequestEntityTooLarge || body["error"].(map[string]any)["code"] != "PAYLOAD_TOO_LARGE" {
		t.Errorf("%d %v", w.Code, body)
	}
}
