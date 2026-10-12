package onboarding

import "github.com/gin-gonic/gin"

// Module wires onboarding. internal/server registers it after the auth and profile modules:
//
//	m, err := onboarding.NewModule(onboarding.Deps{Store: ..., Profiles: onboarding.ProfileService{...}})
//	m.RegisterRoutes(v1, authModule.Authenticate())
//
// Other modules (study plan 2.05, dashboard 2.04, settings 2.11) use m.Service.
type Module struct {
	Service *Service
	handler *Handler
}

type Deps struct {
	Store    Store
	Profiles Profiles
}

func NewModule(d Deps) (*Module, error) {
	svc, err := NewService(ServiceDeps{Store: d.Store, Profiles: d.Profiles})
	if err != nil {
		return nil, err
	}
	return &Module{Service: svc, handler: NewHandler(svc)}, nil
}

func (m *Module) RegisterRoutes(v1 *gin.RouterGroup, authenticate gin.HandlerFunc) {
	m.handler.RegisterRoutes(v1, authenticate)
}
