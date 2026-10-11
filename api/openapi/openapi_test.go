package openapi

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/goccy/go-yaml"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
)

type document struct {
	OpenAPI string                               `yaml:"openapi"`
	Info    struct{ Title, Version string }      `yaml:"info"`
	Paths   map[string]map[string]map[string]any `yaml:"paths"`
	Comps   struct {
		Schemas struct {
			ErrorCode struct {
				Enum []string `yaml:"enum"`
			} `yaml:"ErrorCode"`
		} `yaml:"schemas"`
	} `yaml:"components"`
}

func parse(t *testing.T) document {
	t.Helper()
	var doc document
	if err := yaml.Unmarshal(Spec, &doc); err != nil {
		t.Fatalf("openapi.yaml does not parse: %v", err)
	}
	return doc
}

func TestSpecIsValid(t *testing.T) {
	doc := parse(t)
	if !strings.HasPrefix(doc.OpenAPI, "3.1.") {
		t.Errorf("openapi = %q, want 3.1.x", doc.OpenAPI)
	}
	if doc.Info.Title == "" || doc.Info.Version == "" {
		t.Error("info.title and info.version are required")
	}
	for path, ops := range doc.Paths {
		if !strings.HasPrefix(path, "/api/v1/") && path != "/health" && path != "/ready" {
			t.Errorf("%s: business endpoints must live under /api/v1", path)
		}
		for method, op := range ops {
			if op["operationId"] == nil {
				t.Errorf("%s %s: operationId is required (client generators name methods from it)", method, path)
			}
			if op["responses"] == nil {
				t.Errorf("%s %s: responses are required", method, path)
			}
		}
	}
}

// The documented error codes must match the ones the API can return.
func TestErrorCodesMatchCode(t *testing.T) {
	documented := parse(t).Comps.Schemas.ErrorCode.Enum
	for _, c := range []apperr.Code{
		apperr.CodeBadRequest, apperr.CodeValidation, apperr.CodeUnauthorized, apperr.CodeForbidden,
		apperr.CodeNotFound, apperr.CodeMethodNotAllowed, apperr.CodeConflict, apperr.CodePayloadTooLarge,
		apperr.CodeRateLimited, apperr.CodeInternal, apperr.CodeUnavailable,
	} {
		if !slices.Contains(documented, string(c)) {
			t.Errorf("error code %s is not documented in openapi.yaml", c)
		}
	}
}

func TestSpecIsServed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/openapi.yaml", nil))
	if w.Code != http.StatusOK || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/yaml") {
		t.Fatalf("status %d, content type %q", w.Code, w.Header().Get("Content-Type"))
	}
	if w.Body.Len() != len(Spec) {
		t.Error("served document differs from the embedded spec")
	}
}
