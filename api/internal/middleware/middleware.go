// Package middleware holds the HTTP middleware chain shared by every route:
//
//	RequestID → AccessLog → Recovery → SecurityHeaders → CORS → BodyLimit → Errors
//	→ /api/v1: RateLimit → (auth middleware, added by the auth sessions) → handler
package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
)

const requestIDHeader = "X-Request-ID"

var validRequestID = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

// RequestID accepts a well-formed client X-Request-ID or generates one, and echoes it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(requestIDHeader)
		if !validRequestID.MatchString(id) {
			id = newID()
		}
		c.Set(httpx.RequestIDKey, id)
		c.Header(requestIDHeader, id)
		c.Next()
	}
}

// AccessLog writes one structured line per request. It logs the route template, never
// query strings, headers or bodies (they can carry tokens and personal data).
func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := c.Writer.Status()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}
		log.LogAttrs(c.Request.Context(), level, "http request",
			slog.String("request_id", c.GetString(httpx.RequestIDKey)),
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

// Recovery turns a panic into a logged INTERNAL_ERROR response.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
					panic(rec)
				}
				log.Error("panic recovered",
					slog.String("request_id", c.GetString(httpx.RequestIDKey)),
					slog.Any("panic", rec),
					slog.String("stack", string(debug.Stack())),
				)
				httpx.WriteError(c, apperr.From(nil))
			}
		}()
		c.Next()
	}
}

// Errors renders the first error a handler passed to httpx.Fail. Client errors keep
// their code and message; anything else becomes INTERNAL_ERROR and its cause is logged.
func Errors(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}
		e := apperr.From(c.Errors[0].Err)
		if e.Status() >= 500 {
			log.Error("request failed",
				slog.String("request_id", c.GetString(httpx.RequestIDKey)),
				slog.String("route", c.FullPath()),
				slog.String("error", e.Error()),
			)
		}
		httpx.WriteError(c, e)
	}
}

// SecurityHeaders sets conservative headers for a JSON API. HSTS only in production,
// where traffic is HTTPS.
func SecurityHeaders(production bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		h.Set("Cross-Origin-Resource-Policy", "same-site")
		if production {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		c.Next()
	}
}

// CORS allows only the configured browser origins (exact match). Native mobile apps send
// no Origin header and are unaffected; CORS is a browser protection, not authentication.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowHeaders := "Authorization, Content-Type, X-Request-ID, X-Client-Platform"
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		h := c.Writer.Header()
		if origin != "" {
			h.Add("Vary", "Origin")
		}
		allowed := origin != "" && slices.Contains(allowedOrigins, origin)
		if allowed {
			h.Set("Access-Control-Allow-Origin", origin)
			h.Set("Access-Control-Allow-Credentials", "true")
			h.Set("Access-Control-Expose-Headers", "X-Request-ID, Retry-After, X-RateLimit-Limit, X-RateLimit-Remaining")
		}
		if c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != "" {
			if !allowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			h.Set("Access-Control-Allow-Headers", allowHeaders)
			h.Set("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// BodyLimit caps request bodies. Upload routes raise their own limit when they exist.
func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			httpx.WriteError(c, apperr.New(apperr.CodePayloadTooLarge, "Request body is too large"))
			return
		}
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

// RateLimit applies limit requests per window to each key (default: client IP).
func RateLimit(l ratelimit.Limiter, scope string, limit int, window time.Duration, key func(*gin.Context) string) gin.HandlerFunc {
	if key == nil {
		key = func(c *gin.Context) string { return c.ClientIP() }
	}
	return func(c *gin.Context) {
		res := l.Allow(c.Request.Context(), scope+":"+key(c), limit, window)
		c.Header("X-RateLimit-Limit", strconv.Itoa(res.Limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
		if !res.Allowed {
			secs := int(res.RetryAfter.Seconds() + 0.999)
			if secs < 1 {
				secs = 1
			}
			c.Header("Retry-After", strconv.Itoa(secs))
			httpx.WriteError(c, apperr.New(apperr.CodeRateLimited, "Too many requests. Please try again later.").
				WithDetails(map[string]any{"retry_after_seconds": secs}))
			return
		}
		c.Next()
	}
}

// NotFound and MethodNotAllowed keep unmatched requests in the standard envelope.
func NotFound() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpx.WriteError(c, apperr.NotFound("Route not found"))
	}
}

func MethodNotAllowed() gin.HandlerFunc {
	return func(c *gin.Context) {
		httpx.WriteError(c, apperr.New(apperr.CodeMethodNotAllowed, "Method not allowed"))
	}
}

func newID() string { return randomHex(16) }
