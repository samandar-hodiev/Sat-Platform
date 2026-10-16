package preferences

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrNotFound: the learner has no preferences row yet (all defaults), or the account is gone.
	ErrNotFound = errors.New("preferences: not found")
	// ErrNoCustomWallpaper: "custom" was chosen for a mode without an uploaded image.
	ErrNoCustomWallpaper = errors.New("preferences: no uploaded wallpaper for this mode")
)

// Wallpaper sources.
const (
	SourceNone    = "none"
	SourceGallery = "gallery"
	SourceCustom  = "custom"
)

// Mode is the colour mode a wallpaper belongs to (PRD §4.5: separate Light and Dark wallpapers).
type Mode string

const (
	ModeLight Mode = "light"
	ModeDark  Mode = "dark"
)

// Wallpaper is the choice for one mode. Key is the uploaded image, kept while another source is
// chosen so the learner can switch back; GalleryID is set only for the gallery source.
type Wallpaper struct {
	Source    string
	GalleryID string
	Key       string
}

// Prefs is one user_preferences row.
type Prefs struct {
	Theme            string
	Accent           string
	Density          string
	Motion           string
	SidebarCollapsed bool
	Light            Wallpaper
	Dark             Wallpaper
	Opacity          int
	Blur             int
	UpdatedAt        time.Time
}

// Defaults are the column defaults: a missing row means exactly this (PRD §4.2 neutral Light).
func Defaults() Prefs {
	return Prefs{
		Theme: "light", Accent: "neutral", Density: "comfortable", Motion: "system",
		Light: Wallpaper{Source: SourceNone}, Dark: Wallpaper{Source: SourceNone},
		Opacity: DefaultOpacity, Blur: 0,
	}
}

// Choice selects a wallpaper source for a mode (the uploaded image itself is set by upload).
type Choice struct {
	Source    string
	GalleryID string
}

// Patch is a validated partial update; nil fields keep their value.
type Patch struct {
	Theme            *string
	Accent           *string
	Density          *string
	Motion           *string
	SidebarCollapsed *bool
	Light            *Choice
	Dark             *Choice
	Opacity          *int
	Blur             *int
}

func (p Patch) empty() bool {
	return p.Theme == nil && p.Accent == nil && p.Density == nil && p.Motion == nil &&
		p.SidebarCollapsed == nil && p.Light == nil && p.Dark == nil && p.Opacity == nil && p.Blur == nil
}

// Apply returns current with the patch applied. Both stores call it under their row lock.
func (p Patch) Apply(current Prefs) (Prefs, error) {
	next := current
	set := func(dst *string, v *string) {
		if v != nil {
			*dst = *v
		}
	}
	set(&next.Theme, p.Theme)
	set(&next.Accent, p.Accent)
	set(&next.Density, p.Density)
	set(&next.Motion, p.Motion)
	if p.SidebarCollapsed != nil {
		next.SidebarCollapsed = *p.SidebarCollapsed
	}
	if p.Opacity != nil {
		next.Opacity = *p.Opacity
	}
	if p.Blur != nil {
		next.Blur = *p.Blur
	}
	for _, c := range []struct {
		choice *Choice
		w      *Wallpaper
	}{{p.Light, &next.Light}, {p.Dark, &next.Dark}} {
		if c.choice == nil {
			continue
		}
		if c.choice.Source == SourceCustom && c.w.Key == "" {
			return Prefs{}, ErrNoCustomWallpaper
		}
		c.w.Source = c.choice.Source
		c.w.GalleryID = ""
		if c.choice.Source == SourceGallery {
			c.w.GalleryID = c.choice.GalleryID
		}
	}
	return next, nil
}

// Store persists preferences (PostgreSQL in production, memory in unit tests).
type Store interface {
	// Get returns the row, or ErrNotFound when the learner has none yet.
	Get(ctx context.Context, userID string) (Prefs, error)
	// Update applies the patch under a row lock, creating the row from the defaults if needed.
	// ErrNoCustomWallpaper (nothing written) or ErrNotFound when the user does not exist.
	Update(ctx context.Context, userID string, patch Patch) (Prefs, error)
	// SetWallpaperKey stores a newly uploaded image for mode and selects it; it returns the
	// replaced key ("" if none) so the caller deletes that file.
	SetWallpaperKey(ctx context.Context, userID string, mode Mode, key string) (Prefs, string, error)
	// ClearWallpaperKey removes the uploaded image of mode (selecting "none" if it was chosen)
	// and returns the removed key ("" if none).
	ClearWallpaperKey(ctx context.Context, userID string, mode Mode) (Prefs, string, error)
}
