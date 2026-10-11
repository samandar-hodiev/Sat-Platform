// Package server builds the HTTP router and runs it with graceful shutdown.
//
// Versioning: business routes live under /api/v1 and are registered by modules
// (api/internal/<module>) in NewRouter. A future /api/v2 is a second group with new
// handlers over the same services, so older mobile app versions keep working. Only the
// infrastructure probes (/health, /ready) sit outside the versioned prefix.
package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/config"
	"github.com/samandar-hodiev/sat-platform/api/internal/health"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
	"github.com/samandar-hodiev/sat-platform/api/internal/middleware"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
	"github.com/samandar-hodiev/sat-platform/api/openapi"
)

// Deps are the shared services modules are built from. Modules receive what they need
// from here; they never reach into each other's tables.
type Deps struct {
	Config  *config.Config
	Log     *slog.Logger
	Limiter ratelimit.Limiter
	// Checks are the readiness checks, e.g. {"database": ping}.
	Checks map[string]health.Checker
}

func NewRouter(d Deps) (*gin.Engine, error) {
	cfg := d.Config
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	httpx.ConfigureValidator()

	r := gin.New()
	// Without trusted proxies, ClientIP is the TCP peer; X-Forwarded-For is honoured only
	// from proxies listed in TRUSTED_PROXIES (rate limiting depends on this).
	if err := r.SetTrustedProxies(cfg.HTTP.TrustedProxies); err != nil {
		return nil, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	r.HandleMethodNotAllowed = true
	r.NoRoute(middleware.NotFound())
	r.NoMethod(middleware.MethodNotAllowed())

	r.Use(
		middleware.RequestID(),
		middleware.AccessLog(d.Log),
		middleware.Recovery(d.Log),
		middleware.SecurityHeaders(cfg.IsProduction()),
		middleware.CORS(cfg.HTTP.CORSAllowedOrigins),
		middleware.BodyLimit(cfg.HTTP.MaxBodyBytes),
		middleware.Errors(d.Log),
	)

	health.NewHandler(cfg.Version, d.Checks).RegisterRoutes(r)

	v1 := r.Group("/api/v1",
		middleware.RateLimit(d.Limiter, "api", cfg.RateLimit.PerMinute, time.Minute, nil))
	openapi.RegisterRoutes(v1)
	// Modules register their routes here as they are built, e.g.
	//   auth.NewModule(auth.Deps{...}).RegisterRoutes(v1)

	return r, nil
}

// Run serves until ctx is cancelled (SIGINT/SIGTERM), then stops accepting connections
// and waits up to HTTP_SHUTDOWN_TIMEOUT for in-flight requests to finish.
func Run(ctx context.Context, cfg *config.Config, log *slog.Logger, handler http.Handler) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       60 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("api listening", slog.String("addr", srv.Addr), slog.String("env", string(cfg.Env)), slog.String("version", cfg.Version))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
		close(serveErr)
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down api", slog.Duration("timeout", cfg.HTTP.ShutdownTimeout))
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	log.Info("api stopped")
	return nil
}
