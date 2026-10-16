package preferences

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"slices"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/imaging"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/storage"
)

// Allowed values. Appearance values match the web theme attributes (0.02) and the CHECK
// constraints of user_preferences.
var (
	Themes    = []string{"light", "dark", "system"}
	Accents   = []string{"neutral", "blue", "indigo", "violet", "teal", "rose"}
	Densities = []string{"comfortable", "compact"}
	Motions   = []string{"system", "reduced", "full"}
	Sources   = []string{SourceNone, SourceGallery, SourceCustom}
	// GalleryIDs are the built-in wallpaper patterns. The web draws them
	// (web/src/features/settings/wallpaper/gallery.json lists the same ids; a test compares).
	GalleryIDs = []string{"aurora", "dots", "grid", "waves", "contours", "peaks"}
)

const (
	MinOpacity     = 10
	MaxOpacity     = 80
	DefaultOpacity = 40
	MaxBlur        = 20
)

// View is the learner's preferences as the API returns them. Stored is false while the
// learner has never saved anything (the values are then the defaults): the web uses it to
// upload the choices made on this device before signing in (decided for 2.11: the account
// wins, except on the very first sync).
type View struct {
	Prefs
	Stored bool
}

// PatchInput is the raw PATCH body; nil means "not sent".
type PatchInput struct {
	Theme            *string
	Accent           *string
	Density          *string
	Motion           *string
	SidebarCollapsed *bool
	WallpaperLight   *ChoiceInput
	WallpaperDark    *ChoiceInput
	WallpaperOpacity *int
	WallpaperBlur    *int
}

type ChoiceInput struct {
	Source    string
	GalleryID *string
}

type ServiceDeps struct {
	Store   Store
	Storage storage.ObjectStorage
	Log     *slog.Logger
}

type Service struct {
	store   Store
	storage storage.ObjectStorage
	log     *slog.Logger
}

func NewService(d ServiceDeps) (*Service, error) {
	if d.Store == nil || d.Storage == nil || d.Log == nil {
		return nil, errors.New("preferences: Store, Storage and Log are required")
	}
	return &Service{store: d.Store, storage: d.Storage, log: d.Log}, nil
}

var errAccountGone = apperr.Unauthorized("Sign in to continue.")

// Get returns the stored preferences, or the defaults with Stored=false.
func (s *Service) Get(ctx context.Context, userID string) (View, error) {
	p, err := s.store.Get(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return View{Prefs: Defaults()}, nil
	}
	if err != nil {
		return View{}, err
	}
	return View{Prefs: p, Stored: true}, nil
}

// Update validates and applies a partial update. Every invalid field is reported at once.
// An empty update changes nothing (and creates no row).
func (s *Service) Update(ctx context.Context, userID string, in PatchInput) (View, error) {
	patch, err := validatePatch(in)
	if err != nil {
		return View{}, err
	}
	if patch.empty() {
		return s.Get(ctx, userID)
	}
	p, err := s.store.Update(ctx, userID, patch)
	switch {
	case errors.Is(err, ErrNoCustomWallpaper):
		return View{}, apperr.Conflict("Upload an image for this mode before choosing it.").
			WithDetails(map[string]any{"reason": "wallpaper_not_uploaded"})
	case errors.Is(err, ErrNotFound):
		return View{}, errAccountGone
	case err != nil:
		return View{}, err
	}
	return View{Prefs: p, Stored: true}, nil
}

func validatePatch(in PatchInput) (Patch, error) {
	fields := map[string]string{}
	oneOf := func(field string, v *string, allowed []string) {
		if v != nil && !slices.Contains(allowed, *v) {
			fields[field] = "must be one of " + joinOr(allowed)
		}
	}
	oneOf("theme", in.Theme, Themes)
	oneOf("accent", in.Accent, Accents)
	oneOf("density", in.Density, Densities)
	oneOf("motion", in.Motion, Motions)

	p := Patch{Theme: in.Theme, Accent: in.Accent, Density: in.Density, Motion: in.Motion,
		SidebarCollapsed: in.SidebarCollapsed, Opacity: in.WallpaperOpacity, Blur: in.WallpaperBlur}
	if v := in.WallpaperOpacity; v != nil && (*v < MinOpacity || *v > MaxOpacity) {
		fields["wallpaper.opacity"] = "must be between 10 and 80"
	}
	if v := in.WallpaperBlur; v != nil && (*v < 0 || *v > MaxBlur) {
		fields["wallpaper.blur"] = "must be between 0 and 20"
	}
	choice := func(field string, c *ChoiceInput) *Choice {
		if c == nil {
			return nil
		}
		if !slices.Contains(Sources, c.Source) {
			fields[field+".source"] = "must be one of " + joinOr(Sources)
			return nil
		}
		out := &Choice{Source: c.Source}
		switch {
		case c.Source == SourceGallery && (c.GalleryID == nil || !slices.Contains(GalleryIDs, *c.GalleryID)):
			fields[field+".gallery_id"] = "must be one of " + joinOr(GalleryIDs)
		case c.Source == SourceGallery:
			out.GalleryID = *c.GalleryID
		case c.GalleryID != nil:
			fields[field+".gallery_id"] = "is only allowed with source gallery"
		}
		return out
	}
	p.Light = choice("wallpaper.light", in.WallpaperLight)
	p.Dark = choice("wallpaper.dark", in.WallpaperDark)

	if len(fields) > 0 {
		return Patch{}, validation(fields)
	}
	return p, nil
}

func joinOr(values []string) string {
	out := ""
	for i, v := range values {
		switch {
		case i == 0:
		case i == len(values)-1:
			out += " or "
		default:
			out += ", "
		}
		out += v
	}
	return out
}

func validation(fields map[string]string) error {
	return apperr.New(apperr.CodeValidation, "Some fields are invalid").
		WithDetails(map[string]any{"fields": fields})
}

func wallpaperError(message string) error {
	return validation(map[string]string{"file": message})
}

// ParseMode accepts "light" or "dark" (the path segment of the upload routes).
func ParseMode(s string) (Mode, bool) {
	switch Mode(s) {
	case ModeLight, ModeDark:
		return Mode(s), true
	}
	return "", false
}

// UploadWallpaper validates and re-encodes the image, stores it under a fresh key, makes it the
// wallpaper of mode and deletes the image it replaced.
func (s *Service) UploadWallpaper(ctx context.Context, userID string, mode Mode, data []byte) (View, error) {
	if len(data) > WallpaperMaxBytes {
		return View{}, wallpaperError("The image is larger than 8 MB.")
	}
	img, err := processWallpaper(data)
	switch {
	case errors.Is(err, imaging.ErrUnsupportedType):
		return View{}, wallpaperError("Upload a JPG, PNG or WebP image.")
	case errors.Is(err, imaging.ErrCorrupt):
		return View{}, wallpaperError("This image can't be read. Try another file.")
	case errors.Is(err, imaging.ErrTooLarge):
		return View{}, wallpaperError("This image is too large. Use one under 40 megapixels.")
	case err != nil:
		return View{}, err
	}

	key, err := newWallpaperKey(userID)
	if err != nil {
		return View{}, err
	}
	if _, err := s.storage.Put(ctx, key, bytes.NewReader(img), WallpaperContentType); err != nil {
		return View{}, err
	}
	p, previous, err := s.store.SetWallpaperKey(ctx, userID, mode, key)
	if err != nil {
		s.deleteFile(ctx, key)
		if errors.Is(err, ErrNotFound) {
			return View{}, errAccountGone
		}
		return View{}, err
	}
	s.deleteFile(ctx, previous)
	return View{Prefs: p, Stored: true}, nil
}

// RemoveWallpaper deletes the uploaded image of mode; if it was the chosen wallpaper, the mode
// falls back to no wallpaper.
func (s *Service) RemoveWallpaper(ctx context.Context, userID string, mode Mode) (View, error) {
	p, previous, err := s.store.ClearWallpaperKey(ctx, userID, mode)
	if errors.Is(err, ErrNotFound) {
		return View{}, errAccountGone
	}
	if err != nil {
		return View{}, err
	}
	s.deleteFile(ctx, previous)
	return View{Prefs: p, Stored: true}, nil
}

// deleteFile removes a replaced image. A failure leaves an orphan file, not broken preferences,
// so it is logged and not returned.
func (s *Service) deleteFile(ctx context.Context, key string) {
	if key == "" {
		return
	}
	if err := s.storage.Delete(ctx, key); err != nil {
		s.log.WarnContext(ctx, "wallpaper file not deleted", slog.String("key", key), slog.String("error", err.Error()))
	}
}

// WallpaperFile is an open stored wallpaper; the caller closes Body.
type WallpaperFile struct {
	Body io.ReadCloser
	storage.Object
}

// OpenWallpaper returns a stored wallpaper for the public media route.
func (s *Service) OpenWallpaper(ctx context.Context, key string) (WallpaperFile, error) {
	rc, obj, err := s.storage.Open(ctx, key)
	if errors.Is(err, storage.ErrNotFound) || errors.Is(err, storage.ErrInvalidKey) {
		return WallpaperFile{}, apperr.NotFound("Image not found.")
	}
	if err != nil {
		return WallpaperFile{}, err
	}
	return WallpaperFile{Body: rc, Object: obj}, nil
}
