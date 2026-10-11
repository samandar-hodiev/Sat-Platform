// Package health exposes the infrastructure probes:
//
//	GET /health  the process is running (no dependency checks) — liveness
//	GET /ready   required dependencies answer (PostgreSQL) — readiness; 503 when not
package health

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
)

// Checker returns nil when the dependency is usable.
type Checker func(ctx context.Context) error

type CheckResult struct {
	Status    string `json:"status"` // up | down
	LatencyMs int64  `json:"latency_ms"`
}

type Report struct {
	Status  string                 `json:"status"` // ok | unavailable
	Version string                 `json:"version"`
	Time    time.Time              `json:"time"`
	Checks  map[string]CheckResult `json:"checks"`
}

type Handler struct {
	version string
	checks  map[string]Checker
	timeout time.Duration
}

func NewHandler(version string, checks map[string]Checker) *Handler {
	return &Handler{version: version, checks: checks, timeout: 2 * time.Second}
}

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/health", h.live)
	r.GET("/ready", h.ready)
}

func (h *Handler) live(c *gin.Context) {
	httpx.OK(c, gin.H{"status": "ok", "version": h.version})
}

func (h *Handler) ready(c *gin.Context) {
	report := h.Run(c.Request.Context())
	if report.Status == "ok" {
		httpx.OK(c, report)
		return
	}
	// Dependency errors are not returned: they can reveal hosts and driver details.
	c.JSON(http.StatusServiceUnavailable, httpx.Envelope{
		Success: false,
		Data:    report,
		Error: &httpx.ErrorBody{
			Code:      apperr.CodeUnavailable,
			Message:   "The service is not ready",
			RequestID: c.GetString(httpx.RequestIDKey),
		},
	})
}

// Run executes all checks concurrently with a shared timeout.
func (h *Handler) Run(ctx context.Context) Report {
	ctx, cancel := context.WithTimeout(ctx, h.timeout)
	defer cancel()

	names := make([]string, 0, len(h.checks))
	for name := range h.checks {
		names = append(names, name)
	}
	sort.Strings(names)

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		results = make(map[string]CheckResult, len(names))
		status  = "ok"
	)
	for _, name := range names {
		wg.Add(1)
		go func(name string, check Checker) {
			defer wg.Done()
			start := time.Now()
			err := check(ctx)
			res := CheckResult{Status: "up", LatencyMs: time.Since(start).Milliseconds()}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				res.Status = "down"
				status = "unavailable"
			}
			results[name] = res
		}(name, h.checks[name])
	}
	wg.Wait()

	return Report{Status: status, Version: h.version, Time: time.Now().UTC(), Checks: results}
}
