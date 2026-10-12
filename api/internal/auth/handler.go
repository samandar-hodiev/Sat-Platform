package auth

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
	"github.com/samandar-hodiev/sat-platform/api/internal/middleware"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
)

// Per-IP budgets of the auth routes, on top of the global /api/v1 limit. The service adds
// per-email limits where an attacker could rotate IPs (sign-in, password reset).
var routeLimits = map[string]struct {
	n      int
	window time.Duration
}{
	"auth:sign-up-start":  {5, time.Minute},
	"auth:sign-up-verify": {10, time.Minute},
	"auth:sign-in":        {10, time.Minute},
	"auth:google":         {10, time.Minute},
	"auth:refresh":        {30, time.Minute},
	"auth:forgot":         {5, time.Minute},
	"auth:reset":          {10, time.Minute},
}

type Handler struct {
	svc     *Service
	tokens  *AccessTokens
	limiter ratelimit.Limiter
}

func NewHandler(svc *Service, tokens *AccessTokens, limiter ratelimit.Limiter) *Handler {
	return &Handler{svc: svc, tokens: tokens, limiter: limiter}
}

// RegisterRoutes mounts /auth/* on the versioned group.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup) {
	limit := func(scope string) gin.HandlerFunc {
		l := routeLimits[scope]
		return middleware.RateLimit(h.limiter, scope, l.n, l.window, nil)
	}
	g := v1.Group("/auth")
	g.POST("/sign-up/start", limit("auth:sign-up-start"), h.startSignUp)
	g.POST("/sign-up/verify", limit("auth:sign-up-verify"), h.verifySignUp)
	g.POST("/sign-in", limit("auth:sign-in"), h.signIn)
	g.POST("/google", limit("auth:google"), h.google)
	g.POST("/refresh", limit("auth:refresh"), h.refresh)
	g.POST("/logout", h.logout)
	g.POST("/password/forgot", limit("auth:forgot"), h.forgotPassword)
	g.POST("/password/reset", limit("auth:reset"), h.resetPassword)
	g.GET("/me", h.Authenticate(), h.me)
}

// ---------------------------------------------------------------------------------------------
// Request and response shapes (docs/API.md, openapi.yaml)

type emailRequest struct {
	Email string `json:"email" binding:"required,max=254"`
}

type verifyRequest struct {
	Email string `json:"email" binding:"required,max=254"`
	Code  string `json:"code" binding:"required,len=6,numeric"`
}

type signInRequest struct {
	Email    string `json:"email" binding:"required,max=254"`
	Password string `json:"password" binding:"required,max=512"`
	Remember bool   `json:"remember"`
}

type googleRequest struct {
	IDToken string `json:"id_token" binding:"required,max=8192"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required,max=256"`
}

type resetRequest struct {
	Token    string `json:"token" binding:"required,max=256"`
	Password string `json:"password" binding:"required,max=512"`
}

type userResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	Role          Role      `json:"role"`
	EmailVerified bool      `json:"email_verified"`
	HasPassword   bool      `json:"has_password"`
	CreatedAt     time.Time `json:"created_at"`
}

type sessionResponse struct {
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	TokenType             string       `json:"token_type"`
	Remember              bool         `json:"remember"`
	User                  userResponse `json:"user"`
	IsNewUser             bool         `json:"is_new_user"`
}

type codeSentResponse struct {
	ExpiresAt         time.Time `json:"expires_at"`
	ResendAvailableAt time.Time `json:"resend_available_at"`
}

func toUser(u User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, Role: u.Role, EmailVerified: u.EmailVerifiedAt != nil,
		HasPassword: u.PasswordHash != "", CreatedAt: u.CreatedAt}
}

func toSession(r SessionResult) sessionResponse {
	return sessionResponse{
		AccessToken: r.AccessToken, AccessTokenExpiresAt: r.AccessTokenExpiresAt,
		RefreshToken: r.RefreshToken, RefreshTokenExpiresAt: r.RefreshTokenExpiresAt,
		TokenType: "Bearer", Remember: r.Remember, User: toUser(r.User), IsNewUser: r.IsNewUser,
	}
}

func clientInfo(c *gin.Context) ClientInfo {
	platform := c.GetHeader("X-Client-Platform")
	switch platform {
	case "web", "android", "ios":
	default:
		platform = "unknown"
	}
	ua := c.Request.UserAgent()
	if len(ua) > 512 {
		ua = ua[:512]
	}
	return ClientInfo{Platform: platform, UserAgent: ua, IP: c.ClientIP()}
}

// ---------------------------------------------------------------------------------------------

func (h *Handler) startSignUp(c *gin.Context) {
	var req emailRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	sent, err := h.svc.StartSignUp(c.Request.Context(), req.Email)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, codeSentResponse{ExpiresAt: sent.ExpiresAt, ResendAvailableAt: sent.ResendAvailableAt})
}

func (h *Handler) verifySignUp(c *gin.Context) {
	var req verifyRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, codeBindError(err))
		return
	}
	res, err := h.svc.VerifySignUp(c.Request.Context(), req.Email, req.Code, clientInfo(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toSession(res))
}

func (h *Handler) signIn(c *gin.Context) {
	var req signInRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	res, err := h.svc.SignIn(c.Request.Context(), req.Email, req.Password, req.Remember, clientInfo(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toSession(res))
}

func (h *Handler) google(c *gin.Context) {
	var req googleRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	res, err := h.svc.SignInWithGoogle(c.Request.Context(), req.IDToken, clientInfo(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toSession(res))
}

func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, errSessionEnded)
		return
	}
	res, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toSession(res))
}

func (h *Handler) logout(c *gin.Context) {
	var req refreshRequest
	// A missing or malformed body still logs out locally; there is nothing to revoke.
	_ = c.ShouldBindJSON(&req)
	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) forgotPassword(c *gin.Context) {
	var req emailRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

func (h *Handler) resetPassword(c *gin.Context) {
	var req resetRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.NoContent(c)
}

type meUserResponse struct {
	userResponse
	HasGoogle bool `json:"has_google"`
}

func (h *Handler) me(c *gin.Context) {
	acc, err := h.svc.Account(c.Request.Context(), CurrentUser(c).UserID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"user": meUserResponse{userResponse: toUser(acc.User), HasGoogle: acc.HasGoogle}})
}

// codeBindError gives the code field a learner-friendly message instead of "is invalid".
func codeBindError(err error) error {
	var e *apperr.Error
	if errors.As(err, &e) && e.Code == apperr.CodeValidation {
		if fields, ok := e.Details["fields"].(map[string]string); ok {
			if _, bad := fields["code"]; bad {
				fields["code"] = "Enter the 6-digit code."
			}
		}
	}
	return err
}

// ---------------------------------------------------------------------------------------------
// Authentication middleware

const currentUserKey = "auth.current_user"

// Authenticate requires a valid access token (Authorization: Bearer …) and stores its claims
// for CurrentUser. Expired tokens answer 401 with details.reason "token_expired", so clients
// refresh and retry; any other failure is plain 401.
func (h *Handler) Authenticate() gin.HandlerFunc {
	return Authenticate(h.tokens)
}

func Authenticate(tokens *AccessTokens) gin.HandlerFunc {
	return func(c *gin.Context) {
		const prefix = "Bearer "
		header := c.GetHeader("Authorization")
		if len(header) <= len(prefix) || header[:len(prefix)] != prefix {
			httpx.Fail(c, apperr.Unauthorized("Sign in to continue."))
			return
		}
		claims, err := tokens.Verify(header[len(prefix):])
		if errors.Is(err, ErrExpiredToken) {
			httpx.Fail(c, apperr.Unauthorized("Your session has expired.").
				WithDetails(map[string]any{"reason": "token_expired"}))
			return
		}
		if err != nil {
			httpx.Fail(c, apperr.Unauthorized("Sign in to continue."))
			return
		}
		c.Set(currentUserKey, claims)
		c.Next()
	}
}

// CurrentUser returns the claims set by Authenticate. Handlers behind Authenticate only.
func CurrentUser(c *gin.Context) AccessClaims {
	claims, _ := c.Get(currentUserKey)
	ac, _ := claims.(AccessClaims)
	return ac
}
