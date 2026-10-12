package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
)

func router(logs *bytes.Buffer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(logs, nil))
	r := gin.New()
	r.Use(RequestID(), Recovery(log), Errors(log))
	r.GET("/panic", func(*gin.Context) { panic("boom: secret detail") })
	r.GET("/internal", func(c *gin.Context) {
		httpx.Fail(c, errors.New("pq: password authentication failed for user admin"))
	})
	r.GET("/conflict", func(c *gin.Context) { httpx.Fail(c, apperr.Conflict("Email is already registered")) })
	return r
}

func get(r http.Handler, path string, header map[string]string) (*httptest.ResponseRecorder, map[string]any) {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w, body
}

func TestRecoveryHidesPanicDetails(t *testing.T) {
	var logs bytes.Buffer
	w, body := get(router(&logs), "/panic", nil)
	if w.Code != http.StatusInternalServerError || body["error"].(map[string]any)["code"] != "INTERNAL_ERROR" {
		t.Fatalf("%d %v", w.Code, body)
	}
	if strings.Contains(w.Body.String(), "secret detail") {
		t.Error("panic value leaked to the client")
	}
	if !strings.Contains(logs.String(), "secret detail") {
		t.Error("panic value must be logged")
	}
}

func TestErrorsMapsUnknownErrorsToInternal(t *testing.T) {
	var logs bytes.Buffer
	w, body := get(router(&logs), "/internal", nil)
	if w.Code != http.StatusInternalServerError || strings.Contains(w.Body.String(), "password") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if body["error"].(map[string]any)["request_id"] == "" {
		t.Error("error body must carry the request id")
	}
	if !strings.Contains(logs.String(), "password authentication failed") {
		t.Error("internal cause must be logged")
	}
}

func TestErrorsKeepsClientErrors(t *testing.T) {
	var logs bytes.Buffer
	w, body := get(router(&logs), "/conflict", nil)
	e := body["error"].(map[string]any)
	if w.Code != http.StatusConflict || e["code"] != "CONFLICT" || e["message"] != "Email is already registered" {
		t.Fatalf("%d %v", w.Code, e)
	}
}

func TestRequestIDAcceptsOnlyWellFormedIDs(t *testing.T) {
	var logs bytes.Buffer
	r := router(&logs)
	if w, _ := get(r, "/conflict", map[string]string{"X-Request-ID": "client-trace-123"}); w.Header().Get("X-Request-ID") != "client-trace-123" {
		t.Error("valid client id not echoed")
	}
	w, _ := get(r, "/conflict", map[string]string{"X-Request-ID": "bad id\nwith newline"})
	if id := w.Header().Get("X-Request-ID"); len(id) != 32 {
		t.Errorf("malformed id must be replaced, got %q", id)
	}
}

func TestBodyLimitWithRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(BodyLimitWithRoutes(10, map[string]int64{"POST /upload": 100}))
	read := func(c *gin.Context) {
		if _, err := c.GetRawData(); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	}
	r.POST("/upload", read)
	r.POST("/other", read)
	r.PUT("/upload", read)

	cases := []struct {
		method, path string
		size, want   int
	}{
		{http.MethodPost, "/upload", 100, http.StatusOK},
		{http.MethodPost, "/upload", 101, http.StatusRequestEntityTooLarge},
		{http.MethodPost, "/other", 11, http.StatusRequestEntityTooLarge},
		{http.MethodPut, "/upload", 11, http.StatusRequestEntityTooLarge}, // limit is per method
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(strings.Repeat("x", tc.size)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.want {
			t.Errorf("%s %s %d bytes: %d, want %d", tc.method, tc.path, tc.size, w.Code, tc.want)
		}
	}
	// Without Content-Length the reader still stops at the route's limit.
	req := httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader(strings.Repeat("x", 101)))
	req.ContentLength = -1
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("chunked: %d", w.Code)
	}
}
