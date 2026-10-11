package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func serve(checks map[string]Checker, path string) (*httptest.ResponseRecorder, map[string]any) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	NewHandler("test", checks).RegisterRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w, body
}

func up(context.Context) error   { return nil }
func down(context.Context) error { return errors.New("dial tcp 10.0.0.5:5432: connection refused") }

func TestHealthIgnoresDependencies(t *testing.T) {
	w, body := serve(map[string]Checker{"database": down}, "/health")
	if w.Code != http.StatusOK || body["success"] != true {
		t.Fatalf("liveness must not depend on the database: %d %v", w.Code, body)
	}
}

func TestReadyWhenDatabaseIsUp(t *testing.T) {
	w, body := serve(map[string]Checker{"database": up}, "/ready")
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	data := body["data"].(map[string]any)
	if data["status"] != "ok" || data["checks"].(map[string]any)["database"].(map[string]any)["status"] != "up" {
		t.Errorf("data = %v", data)
	}
}

func TestNotReadyWhenDatabaseIsDown(t *testing.T) {
	w, body := serve(map[string]Checker{"database": down}, "/ready")
	if w.Code != http.StatusServiceUnavailable || body["success"] != false {
		t.Fatalf("status %d, body %v", w.Code, body)
	}
	if body["error"].(map[string]any)["code"] != "SERVICE_UNAVAILABLE" {
		t.Errorf("error = %v", body["error"])
	}
	if s := w.Body.String(); strings.Contains(s, "10.0.0.5") || strings.Contains(s, "refused") {
		t.Errorf("dependency error leaked: %s", s)
	}
}
