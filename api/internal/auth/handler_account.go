package auth

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
	"github.com/samandar-hodiev/sat-platform/api/internal/middleware"
)

// HTTP side of the account changes of My Profile (2.10, account.go; docs/API.md → My Profile).
// Account deletion is DELETE /me/account in the profile module, which deletes the profile in
// the same transaction.

// Per-user budgets of the account routes. Codes have their own cooldown and daily cap, and
// password confirmations their own budget (account.go).
var accountRouteLimits = map[string]struct {
	n      int
	window time.Duration
}{
	"account:reauth-code":   {5, 10 * time.Minute},
	"account:email-change":  {10, 10 * time.Minute},
	"account:email-confirm": {10, 10 * time.Minute},
	"account:password":      {10, 10 * time.Minute},
	"account:google":        {10, 10 * time.Minute},
}

func (h *Handler) registerAccountRoutes(v1 *gin.RouterGroup) {
	byUser := func(c *gin.Context) string { return CurrentUser(c).UserID }
	limit := func(scope string) gin.HandlerFunc {
		l := accountRouteLimits[scope]
		return middleware.RateLimit(h.limiter, scope, l.n, l.window, byUser)
	}
	me := v1.Group("/me", h.Authenticate(), LearnerOnly)
	me.GET("/sign-in-methods", h.signInMethods)
	me.POST("/reauth/code", limit("account:reauth-code"), h.sendReauthCode)
	me.POST("/email/change", limit("account:email-change"), h.startEmailChange)
	me.POST("/email/change/confirm", limit("account:email-confirm"), h.confirmEmailChange)
	me.PUT("/password", limit("account:password"), h.changePassword)
	me.POST("/sign-in-methods/google", limit("account:google"), h.linkGoogle)
	me.DELETE("/sign-in-methods/google", h.unlinkGoogle)
}

// LearnerOnly rejects owner tokens: these routes manage a learner account (owner accounts are
// 1.02's). Use after Authenticate.
func LearnerOnly(c *gin.Context) {
	if CurrentUser(c).Role != RoleLearner {
		httpx.Fail(c, apperr.Forbidden("This is available to learners only."))
		return
	}
	c.Next()
}

// ReauthRequest is the confirmation part of a sensitive request (see Reauth). Embedded in
// request bodies, also by the profile module (DELETE /me/account).
type ReauthRequest struct {
	CurrentPassword string `json:"current_password" binding:"max=512"`
	Code            string `json:"code" binding:"omitempty,len=6,numeric"`
}

func (r ReauthRequest) Reauth() Reauth { return Reauth{Password: r.CurrentPassword, Code: r.Code} }

type identityResponse struct {
	Email    *string   `json:"email"`
	LinkedAt time.Time `json:"linked_at"`
}

type signInMethodsResponse struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	HasPassword   bool   `json:"has_password"`
	// EmailCode: signing in with a code sent to the email always works.
	EmailCode bool `json:"email_code"`
	// ReauthMethod: what sensitive changes ask for, "password" or "email_code".
	ReauthMethod string            `json:"reauth_method"`
	Google       *identityResponse `json:"google"`
}

func toSignInMethods(m SignInMethods) signInMethodsResponse {
	res := signInMethodsResponse{
		Email: m.Email, EmailVerified: m.EmailVerified, HasPassword: m.HasPassword,
		EmailCode: true, ReauthMethod: m.ReauthMethod(),
	}
	if m.Google != nil {
		res.Google = &identityResponse{LinkedAt: m.Google.LinkedAt}
		if m.Google.Email != "" {
			res.Google.Email = &m.Google.Email
		}
	}
	return res
}

func (h *Handler) signInMethods(c *gin.Context) {
	m, err := h.svc.SignInMethods(c.Request.Context(), CurrentUser(c).UserID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toSignInMethods(m))
}

func (h *Handler) sendReauthCode(c *gin.Context) {
	sent, err := h.svc.SendReauthCode(c.Request.Context(), CurrentUser(c).UserID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, codeSentResponse{ExpiresAt: sent.ExpiresAt, ResendAvailableAt: sent.ResendAvailableAt})
}

type emailChangeRequest struct {
	NewEmail string `json:"new_email" binding:"required,max=254"`
	ReauthRequest
}

type emailChangeResponse struct {
	NewEmail string `json:"new_email"`
	codeSentResponse
}

func (h *Handler) startEmailChange(c *gin.Context) {
	var req emailChangeRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, codeBindError(err))
		return
	}
	started, err := h.svc.StartEmailChange(c.Request.Context(), CurrentUser(c).UserID, req.NewEmail, req.Reauth())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, emailChangeResponse{
		NewEmail:         started.Email,
		codeSentResponse: codeSentResponse{ExpiresAt: started.ExpiresAt, ResendAvailableAt: started.ResendAvailableAt},
	})
}

type codeRequest struct {
	Code string `json:"code" binding:"required,len=6,numeric"`
}

func (h *Handler) confirmEmailChange(c *gin.Context) {
	var req codeRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, codeBindError(err))
		return
	}
	user := CurrentUser(c)
	m, err := h.svc.ConfirmEmailChange(c.Request.Context(), user.UserID, user.SessionID, req.Code)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toSignInMethods(m))
}

type passwordRequest struct {
	NewPassword string `json:"new_password" binding:"required,max=512"`
	ReauthRequest
}

func (h *Handler) changePassword(c *gin.Context) {
	var req passwordRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, codeBindError(err))
		return
	}
	user := CurrentUser(c)
	if err := h.svc.ChangePassword(c.Request.Context(), user.UserID, user.SessionID, req.Reauth(), req.NewPassword); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

type linkGoogleRequest struct {
	IDToken string `json:"id_token" binding:"required,max=8192"`
	ReauthRequest
}

func (h *Handler) linkGoogle(c *gin.Context) {
	var req linkGoogleRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, codeBindError(err))
		return
	}
	m, err := h.svc.LinkGoogle(c.Request.Context(), CurrentUser(c).UserID, req.IDToken, req.Reauth())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toSignInMethods(m))
}

func (h *Handler) unlinkGoogle(c *gin.Context) {
	m, err := h.svc.UnlinkGoogle(c.Request.Context(), CurrentUser(c).UserID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toSignInMethods(m))
}
