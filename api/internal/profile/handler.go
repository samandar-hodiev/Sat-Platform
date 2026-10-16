package profile

import (
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
	"github.com/samandar-hodiev/sat-platform/api/internal/middleware"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
)

const (
	// AvatarRoute is the upload route; server raises the body limit for it to AvatarRequestMaxBytes.
	AvatarRoute = "/api/v1/me/avatar"
	// AvatarRequestMaxBytes is the image plus room for the multipart framing.
	AvatarRequestMaxBytes = AvatarMaxBytes + 64<<10
	// avatarMediaPrefix is where stored avatars are served (public, unguessable URLs).
	avatarMediaPrefix = "/api/v1/media/"
)

// Per-learner upload budget: re-encoding images costs CPU.
const (
	avatarUploadLimit  = 20
	avatarUploadWindow = time.Hour
)

type Handler struct {
	svc     *Service
	limiter ratelimit.Limiter
}

func NewHandler(svc *Service, limiter ratelimit.Limiter) *Handler {
	return &Handler{svc: svc, limiter: limiter}
}

// RegisterRoutes mounts /me/profile*, /me/avatar and the public avatar media route.
// authenticate is the auth module's middleware.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup, authenticate gin.HandlerFunc) {
	me := v1.Group("/me", authenticate, learnerOnly)
	me.GET("/profile", h.get)
	me.PUT("/profile/setup", h.setup)
	byUser := func(c *gin.Context) string { return auth.CurrentUser(c).UserID }
	me.POST("/avatar", middleware.RateLimit(h.limiter, "profile:avatar", avatarUploadLimit, avatarUploadWindow, byUser), h.uploadAvatar)
	me.DELETE("/avatar", h.removeAvatar)
	// My Profile (2.10), handler_account.go.
	me.PATCH("/profile", h.update)
	me.DELETE("/account", middleware.RateLimit(h.limiter, "profile:delete-account", deleteAccountLimit, deleteAccountWindow, byUser), h.deleteAccount)

	v1.GET("/media/avatars/:user/:file", h.serveAvatar)
}

// learnerOnly: the Owner has no learner profile (owner accounts are 1.02).
func learnerOnly(c *gin.Context) {
	if auth.CurrentUser(c).Role != auth.RoleLearner {
		httpx.Fail(c, apperr.Forbidden("This is available to learners only."))
		return
	}
	c.Next()
}

type profileResponse struct {
	Email              string     `json:"email"`
	FirstName          *string    `json:"first_name"`
	LastName           *string    `json:"last_name"`
	Phone              *string    `json:"phone"`
	DateOfBirth        *string    `json:"date_of_birth"`
	AvatarURL          *string    `json:"avatar_url"`
	Timezone           string     `json:"timezone"`
	HasPassword        bool       `json:"has_password"`
	PasswordRequired   bool       `json:"password_required"`
	ProfileCompleted   bool       `json:"profile_completed"`
	ProfileCompletedAt *time.Time `json:"profile_completed_at"`
}

func strOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// AvatarURL is the path (relative to the API origin) a stored avatar is served at.
func AvatarURL(key string) string { return avatarMediaPrefix + key }

func toResponse(v View) profileResponse {
	r := profileResponse{
		Email: v.Email, FirstName: strOrNil(v.FirstName), LastName: strOrNil(v.LastName),
		Phone: strOrNil(v.Phone), Timezone: v.Timezone, HasPassword: v.HasPassword,
		PasswordRequired: v.PasswordRequired, ProfileCompleted: v.CompletedAt != nil,
		ProfileCompletedAt: v.CompletedAt,
	}
	if v.DateOfBirth != nil {
		d := v.DateOfBirth.Format(time.DateOnly)
		r.DateOfBirth = &d
	}
	if v.AvatarKey != "" {
		u := AvatarURL(v.AvatarKey)
		r.AvatarURL = &u
	}
	return r
}

func (h *Handler) get(c *gin.Context) {
	v, err := h.svc.Get(c.Request.Context(), auth.CurrentUser(c).UserID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(v))
}

type setupRequest struct {
	FirstName   string `json:"first_name" binding:"max=400"`
	LastName    string `json:"last_name" binding:"max=400"`
	Phone       string `json:"phone" binding:"max=64"`
	DateOfBirth string `json:"date_of_birth" binding:"max=32"`
	Timezone    string `json:"timezone" binding:"max=64"`
	Password    string `json:"password" binding:"max=1024"`
}

func (h *Handler) setup(c *gin.Context) {
	var req setupRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	v, err := h.svc.CompleteSetup(c.Request.Context(), auth.CurrentUser(c).UserID, SetupInput(req))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(v))
}

// uploadAvatar reads the "file" part of a multipart/form-data body, streaming: nothing is
// written to disk before validation.
func (h *Handler) uploadAvatar(c *gin.Context) {
	mr, err := c.Request.MultipartReader()
	if err != nil {
		httpx.Fail(c, apperr.BadRequest("Send the image as multipart/form-data in the \"file\" field."))
		return
	}
	var data []byte
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			httpx.Fail(c, readError(err))
			return
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		data, err = io.ReadAll(io.LimitReader(part, AvatarMaxBytes+1))
		_ = part.Close()
		if err != nil {
			httpx.Fail(c, readError(err))
			return
		}
		break
	}
	if len(data) == 0 {
		httpx.Fail(c, avatarError("Choose an image to upload."))
		return
	}
	v, err := h.svc.UploadAvatar(c.Request.Context(), auth.CurrentUser(c).UserID, data)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(v))
}

func readError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return apperr.New(apperr.CodePayloadTooLarge, "The image is larger than 5 MB.")
	}
	return apperr.BadRequest("The upload could not be read. Please try again.")
}

func (h *Handler) removeAvatar(c *gin.Context) {
	v, err := h.svc.RemoveAvatar(c.Request.Context(), auth.CurrentUser(c).UserID)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(v))
}

var (
	uuidPattern       = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	avatarFilePattern = regexp.MustCompile(`^[0-9a-f]{32}\.jpg$`)
)

// serveAvatar is public: <img> tags cannot send the bearer token, and avatar URLs carry 128
// random bits. Every upload gets a new URL, so responses are cached for a year.
func (h *Handler) serveAvatar(c *gin.Context) {
	user, file := c.Param("user"), c.Param("file")
	if !uuidPattern.MatchString(user) || !avatarFilePattern.MatchString(file) {
		httpx.Fail(c, apperr.NotFound("Image not found."))
		return
	}
	f, err := h.svc.OpenAvatar(c.Request.Context(), "avatars/"+user+"/"+file)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	defer f.Body.Close()
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Content-Disposition", "inline")
	ct := f.ContentType
	if !strings.HasPrefix(ct, "image/") {
		ct = AvatarContentType
	}
	c.DataFromReader(http.StatusOK, f.Size, ct, f.Body, nil)
}
