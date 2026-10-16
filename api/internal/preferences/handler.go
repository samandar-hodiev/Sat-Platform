package preferences

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
	// WallpaperRoute is the upload route template; server raises the body limit for it to
	// WallpaperRequestMaxBytes.
	WallpaperRoute = "/api/v1/me/wallpapers/:mode"
	// WallpaperRequestMaxBytes is the image plus room for the multipart framing.
	WallpaperRequestMaxBytes = WallpaperMaxBytes + 64<<10
	mediaPrefix              = "/api/v1/media/"
)

// Per-learner upload budget: re-encoding images costs CPU.
const (
	wallpaperUploadLimit  = 20
	wallpaperUploadWindow = time.Hour
)

type Handler struct {
	svc     *Service
	limiter ratelimit.Limiter
}

func NewHandler(svc *Service, limiter ratelimit.Limiter) *Handler {
	return &Handler{svc: svc, limiter: limiter}
}

// RegisterRoutes mounts /me/preferences, /me/wallpapers/:mode and the public wallpaper media
// route. authenticate is the auth module's middleware.
func (h *Handler) RegisterRoutes(v1 *gin.RouterGroup, authenticate gin.HandlerFunc) {
	me := v1.Group("/me", authenticate, learnerOnly)
	me.GET("/preferences", h.get)
	me.PATCH("/preferences", h.update)
	byUser := func(c *gin.Context) string { return auth.CurrentUser(c).UserID }
	me.POST("/wallpapers/:mode", middleware.RateLimit(h.limiter, "preferences:wallpaper", wallpaperUploadLimit, wallpaperUploadWindow, byUser), h.upload)
	me.DELETE("/wallpapers/:mode", h.remove)

	v1.GET("/media/wallpapers/:user/:file", h.serve)
}

// learnerOnly: these are the learner app's settings; the Owner's appearance is 4.01's.
func learnerOnly(c *gin.Context) {
	if auth.CurrentUser(c).Role != auth.RoleLearner {
		httpx.Fail(c, apperr.Forbidden("This is available to learners only."))
		return
	}
	c.Next()
}

type wallpaperResponse struct {
	Source    string  `json:"source"`
	GalleryID *string `json:"gallery_id"`
	// CustomURL is the uploaded image of this mode (also while another source is chosen).
	CustomURL *string `json:"custom_url"`
}

type wallpapersResponse struct {
	Light   wallpaperResponse `json:"light"`
	Dark    wallpaperResponse `json:"dark"`
	Opacity int               `json:"opacity"`
	Blur    int               `json:"blur"`
}

type preferencesResponse struct {
	Theme            string             `json:"theme"`
	Accent           string             `json:"accent"`
	Density          string             `json:"density"`
	Motion           string             `json:"motion"`
	SidebarCollapsed bool               `json:"sidebar_collapsed"`
	Wallpaper        wallpapersResponse `json:"wallpaper"`
	// Stored is false while nothing was ever saved: the values are the defaults.
	Stored    bool       `json:"stored"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// WallpaperURL is the path (relative to the API origin) a stored wallpaper is served at.
func WallpaperURL(key string) string { return mediaPrefix + key }

func toWallpaper(w Wallpaper) wallpaperResponse {
	r := wallpaperResponse{Source: w.Source}
	if w.GalleryID != "" {
		id := w.GalleryID
		r.GalleryID = &id
	}
	if w.Key != "" {
		u := WallpaperURL(w.Key)
		r.CustomURL = &u
	}
	return r
}

func toResponse(v View) preferencesResponse {
	r := preferencesResponse{
		Theme: v.Theme, Accent: v.Accent, Density: v.Density, Motion: v.Motion,
		SidebarCollapsed: v.SidebarCollapsed,
		Wallpaper: wallpapersResponse{
			Light: toWallpaper(v.Light), Dark: toWallpaper(v.Dark), Opacity: v.Opacity, Blur: v.Blur,
		},
		Stored: v.Stored,
	}
	if v.Stored {
		t := v.UpdatedAt
		r.UpdatedAt = &t
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

type choiceRequest struct {
	Source    string  `json:"source"`
	GalleryID *string `json:"gallery_id"`
}

type wallpaperRequest struct {
	Light   *choiceRequest `json:"light"`
	Dark    *choiceRequest `json:"dark"`
	Opacity *int           `json:"opacity"`
	Blur    *int           `json:"blur"`
}

// updateRequest: every field is optional; only the fields sent change.
type updateRequest struct {
	Theme            *string           `json:"theme"`
	Accent           *string           `json:"accent"`
	Density          *string           `json:"density"`
	Motion           *string           `json:"motion"`
	SidebarCollapsed *bool             `json:"sidebar_collapsed"`
	Wallpaper        *wallpaperRequest `json:"wallpaper"`
}

func (h *Handler) update(c *gin.Context) {
	var req updateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	in := PatchInput{Theme: req.Theme, Accent: req.Accent, Density: req.Density, Motion: req.Motion,
		SidebarCollapsed: req.SidebarCollapsed}
	if w := req.Wallpaper; w != nil {
		in.WallpaperOpacity, in.WallpaperBlur = w.Opacity, w.Blur
		if w.Light != nil {
			in.WallpaperLight = &ChoiceInput{Source: w.Light.Source, GalleryID: w.Light.GalleryID}
		}
		if w.Dark != nil {
			in.WallpaperDark = &ChoiceInput{Source: w.Dark.Source, GalleryID: w.Dark.GalleryID}
		}
	}
	v, err := h.svc.Update(c.Request.Context(), auth.CurrentUser(c).UserID, in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(v))
}

func modeParam(c *gin.Context) (Mode, bool) {
	m, ok := ParseMode(c.Param("mode"))
	if !ok {
		httpx.Fail(c, apperr.NotFound("Wallpapers exist for the light and dark modes only."))
	}
	return m, ok
}

// upload reads the "file" part of a multipart/form-data body, streaming: nothing is written to
// disk before validation.
func (h *Handler) upload(c *gin.Context) {
	mode, ok := modeParam(c)
	if !ok {
		return
	}
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
		data, err = io.ReadAll(io.LimitReader(part, WallpaperMaxBytes+1))
		_ = part.Close()
		if err != nil {
			httpx.Fail(c, readError(err))
			return
		}
		break
	}
	if len(data) == 0 {
		httpx.Fail(c, wallpaperError("Choose an image to upload."))
		return
	}
	v, err := h.svc.UploadWallpaper(c.Request.Context(), auth.CurrentUser(c).UserID, mode, data)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(v))
}

func readError(err error) error {
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return apperr.New(apperr.CodePayloadTooLarge, "The image is larger than 8 MB.")
	}
	return apperr.BadRequest("The upload could not be read. Please try again.")
}

func (h *Handler) remove(c *gin.Context) {
	mode, ok := modeParam(c)
	if !ok {
		return
	}
	v, err := h.svc.RemoveWallpaper(c.Request.Context(), auth.CurrentUser(c).UserID, mode)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, toResponse(v))
}

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	filePattern = regexp.MustCompile(`^[0-9a-f]{32}\.jpg$`)
)

// serve is public: CSS background images cannot send the bearer token, and wallpaper URLs carry
// 128 random bits. Every upload gets a new URL, so responses are cached for a year.
func (h *Handler) serve(c *gin.Context) {
	user, file := c.Param("user"), c.Param("file")
	if !uuidPattern.MatchString(user) || !filePattern.MatchString(file) {
		httpx.Fail(c, apperr.NotFound("Image not found."))
		return
	}
	f, err := h.svc.OpenWallpaper(c.Request.Context(), "wallpapers/"+user+"/"+file)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	defer f.Body.Close()
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.Header("Content-Disposition", "inline")
	ct := f.ContentType
	if !strings.HasPrefix(ct, "image/") {
		ct = WallpaperContentType
	}
	c.DataFromReader(http.StatusOK, f.Size, ct, f.Body, nil)
}
