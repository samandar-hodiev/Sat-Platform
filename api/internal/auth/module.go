package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/config"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/mailer"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
)

// Module wires the auth service from configuration. internal/server registers it once the
// PostgreSQL store exists (after session 0.01):
//
//	m, err := auth.NewModule(auth.Deps{Config: cfg, Log: log, Limiter: limiter, Store: auth.NewPostgresStore(db)})
//	m.RegisterRoutes(v1)
//
// Other modules protect their routes with m.Authenticate() and read auth.CurrentUser(c).
type Module struct {
	Service *Service
	Tokens  *AccessTokens
	handler *Handler
}

type Deps struct {
	Config  *config.Config
	Log     *slog.Logger
	Limiter ratelimit.Limiter
	Store   Store
	// Optional: the profile module (2.01) fills a new Google learner's name.
	ProfilePrefiller ProfilePrefiller
}

func NewModule(d Deps) (*Module, error) {
	ac := d.Config.Auth
	secret := ac.JWTSecret
	if secret == "" {
		if d.Config.IsProduction() {
			return nil, fmt.Errorf("auth: JWT_SECRET is required in production")
		}
		// Development convenience: a random secret per process. Every token dies on restart.
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		secret = hex.EncodeToString(b)
		d.Log.Warn("JWT_SECRET is not set; using a random secret, sessions end when the API restarts")
	}

	tokens, err := NewAccessTokens(secret, ac.AccessTokenTTL)
	if err != nil {
		return nil, err
	}
	codes, err := NewCodeHasher(secret)
	if err != nil {
		return nil, err
	}

	var google *GoogleVerifier
	if len(ac.GoogleClientIDs) > 0 {
		if google, err = NewGoogleVerifier(ac.GoogleClientIDs, nil); err != nil {
			return nil, err
		}
	}

	var mail mailer.Mailer
	switch ac.MailDriver {
	case "log":
		if mail, err = mailer.NewLog(d.Log, d.Config.IsProduction()); err != nil {
			return nil, err
		}
	case "disabled":
		// nil: sign-up codes and password reset answer SERVICE_UNAVAILABLE.
	default:
		return nil, fmt.Errorf("auth: unsupported MAIL_DRIVER %q", ac.MailDriver)
	}

	cfg := DefaultServiceConfig(d.Config.WebURL)
	cfg.RefreshTokenTTL = ac.RefreshTokenTTL
	cfg.ShortRefreshTokenTTL = ac.ShortRefreshTokenTTL

	svc, err := NewService(ServiceDeps{
		Config: cfg, Store: d.Store, Tokens: tokens, Codes: codes, Google: google,
		Mail: mail, Limiter: d.Limiter, Log: d.Log, ProfilePrefiller: d.ProfilePrefiller,
	})
	if err != nil {
		return nil, err
	}
	return &Module{Service: svc, Tokens: tokens, handler: NewHandler(svc, tokens, d.Limiter)}, nil
}

func (m *Module) RegisterRoutes(v1 *gin.RouterGroup) { m.handler.RegisterRoutes(v1) }

// Authenticate is the middleware other modules put in front of learner routes.
func (m *Module) Authenticate() gin.HandlerFunc { return Authenticate(m.Tokens) }
