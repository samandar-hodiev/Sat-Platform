// Package httpx holds the API's HTTP conventions: the response envelope, request binding
// and validation, and pagination. Every endpoint responds through these helpers so web,
// iOS and Android clients can rely on one shape (see docs/API.md).
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
)

// Envelope is the single response shape:
//
//	{ "success": true,  "data": {...}, "meta": {...} }
//	{ "success": false, "error": { "code": "...", "message": "...", "details": {...}, "request_id": "..." } }
type Envelope struct {
	Success bool       `json:"success"`
	Data    any        `json:"data,omitempty"`
	Error   *ErrorBody `json:"error,omitempty"`
	Meta    *Meta      `json:"meta,omitempty"`
}

type ErrorBody struct {
	Code      apperr.Code    `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

// Meta carries pagination information for list responses.
type Meta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// RequestIDKey is the gin context key holding the request ID.
const RequestIDKey = "request_id"

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data})
}

func OKWithMeta(c *gin.Context, data any, meta Meta) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data, Meta: &meta})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Fail hands err to the error middleware and stops the handler chain:
//
//	if err != nil { httpx.Fail(c, err); return }
func Fail(c *gin.Context, err error) {
	_ = c.Error(err)
	c.Abort()
}

// WriteError renders a client-safe error immediately. Handlers use Fail; only middleware
// that runs outside the error middleware (recovery, rate limiting, 404) calls this.
func WriteError(c *gin.Context, e *apperr.Error) {
	c.AbortWithStatusJSON(e.Status(), Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:      e.Code,
			Message:   e.Message,
			Details:   e.Details,
			RequestID: c.GetString(RequestIDKey),
		},
	})
}
