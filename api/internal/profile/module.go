package profile

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/storage"
)

// Module wires Profile Setup. internal/server registers it after the auth module:
//
//	m, err := profile.NewModule(profile.Deps{..., Accounts: <auth adapter>})
//	m.RegisterRoutes(v1, authModule.Authenticate())
type Module struct {
	Service *Service
	handler *Handler
}

type Deps struct {
	Log      *slog.Logger
	Limiter  ratelimit.Limiter
	Store    Store
	Accounts Accounts
	Storage  storage.ObjectStorage
}

func NewModule(d Deps) (*Module, error) {
	svc, err := NewService(ServiceDeps{Store: d.Store, Accounts: d.Accounts, Storage: d.Storage, Log: d.Log})
	if err != nil {
		return nil, err
	}
	return &Module{Service: svc, handler: NewHandler(svc, d.Limiter)}, nil
}

func (m *Module) RegisterRoutes(v1 *gin.RouterGroup, authenticate gin.HandlerFunc) {
	m.handler.RegisterRoutes(v1, authenticate)
}
