package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
)

var configureOnce sync.Once

// ConfigureValidator makes validation errors name fields by their json/form tag, so
// `details.fields` keys match what the client sent. Called once when the router is built.
func ConfigureValidator() {
	configureOnce.Do(func() {
		v, ok := binding.Validator.Engine().(*validator.Validate)
		if !ok {
			return
		}
		v.RegisterTagNameFunc(func(f reflect.StructField) string {
			for _, tag := range []string{"json", "form", "uri"} {
				name := strings.Split(f.Tag.Get(tag), ",")[0]
				if name == "-" {
					return ""
				}
				if name != "" {
					return name
				}
			}
			return f.Name
		})
	})
}

// BindJSON decodes and validates a JSON body (validator tags on dst). Malformed JSON is
// BAD_REQUEST; failed rules are VALIDATION_ERROR with per-field messages.
func BindJSON(c *gin.Context, dst any) error {
	if err := c.ShouldBindJSON(dst); err != nil {
		return bindError(err)
	}
	return nil
}

// BindQuery decodes and validates query parameters (form tags on dst).
func BindQuery(c *gin.Context, dst any) error {
	if err := c.ShouldBindQuery(dst); err != nil {
		return bindError(err)
	}
	return nil
}

func bindError(err error) error {
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		fields := make(map[string]string, len(verrs))
		for _, fe := range verrs {
			fields[fe.Field()] = fieldMessage(fe)
		}
		return apperr.New(apperr.CodeValidation, "Some fields are invalid").
			WithDetails(map[string]any{"fields": fields})
	}
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return apperr.New(apperr.CodePayloadTooLarge, "Request body is too large")
	}
	var syntaxErr *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	if errors.Is(err, io.EOF) || errors.As(err, &syntaxErr) || errors.As(err, &typeErr) {
		return apperr.New(apperr.CodeBadRequest, "Request body must be valid JSON")
	}
	return apperr.New(apperr.CodeBadRequest, "Invalid request")
}

func fieldMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return "must be at least " + fe.Param()
	case "max":
		return "must be at most " + fe.Param()
	case "oneof":
		return "must be one of: " + fe.Param()
	case "uuid", "uuid4":
		return "must be a valid UUID"
	default:
		return "is invalid"
	}
}

// Pagination is the shared ?page=&page_size= contract. Defaults: page 1, 20 per page, max 100.
type Pagination struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

func (p *Pagination) Normalize() {
	if p.Page == 0 {
		p.Page = 1
	}
	if p.PageSize == 0 {
		p.PageSize = 20
	}
}

func (p Pagination) Offset() int { return (p.Page - 1) * p.PageSize }

func (p Pagination) Meta(total int64) Meta {
	return Meta{Page: p.Page, PageSize: p.PageSize, Total: total}
}
