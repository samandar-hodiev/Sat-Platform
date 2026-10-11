package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
)

type signUp struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"display_name" binding:"required,max=5"`
}

func bind(body string) error {
	gin.SetMode(gin.TestMode)
	ConfigureValidator()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	var in signUp
	return BindJSON(c, &in)
}

func TestBindJSONValidationUsesJSONFieldNames(t *testing.T) {
	err := bind(`{"email":"nope","display_name":"too long name"}`)
	var e *apperr.Error
	if !errors.As(err, &e) || e.Code != apperr.CodeValidation {
		t.Fatalf("want VALIDATION_ERROR, got %v", err)
	}
	fields := e.Details["fields"].(map[string]string)
	if fields["email"] != "must be a valid email address" || fields["display_name"] != "must be at most 5" {
		t.Errorf("fields = %v", fields)
	}
}

func TestBindJSONMalformed(t *testing.T) {
	for _, body := range []string{`{`, ``, `{"email": 5}`} {
		var e *apperr.Error
		if err := bind(body); !errors.As(err, &e) || e.Code != apperr.CodeBadRequest {
			t.Errorf("%q: want BAD_REQUEST, got %v", body, err)
		}
	}
}

func TestPagination(t *testing.T) {
	p := Pagination{}
	p.Normalize()
	if p.Page != 1 || p.PageSize != 20 || p.Offset() != 0 {
		t.Errorf("defaults = %+v", p)
	}
	p = Pagination{Page: 3, PageSize: 10}
	if p.Offset() != 20 || p.Meta(42).Total != 42 {
		t.Errorf("offset/meta wrong: %d %+v", p.Offset(), p.Meta(42))
	}
}
