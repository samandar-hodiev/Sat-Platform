package preferences

import (
	"crypto/rand"
	"encoding/hex"
	"image"
	"image/color"

	"golang.org/x/image/draw"

	"github.com/samandar-hodiev/sat-platform/api/internal/platform/imaging"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/storage"
)

const (
	// WallpaperMaxBytes is the largest accepted upload (decided for 2.11: 8 MB).
	WallpaperMaxBytes = 8 << 20
	// WallpaperMaxSide is the longest side of the stored image; larger uploads are scaled down
	// (never up). 2560 px covers a QHD screen; the wallpaper is decoration, often blurred.
	WallpaperMaxSide = 2560
	// Same decompression-bomb guard as avatars (2.01): 40 MP covers every phone camera.
	wallpaperMaxPixels = 40_000_000
	wallpaperMaxSide   = 12_000
	wallpaperQuality   = 82
	// WallpaperContentType: uploads are re-encoded, never kept as is.
	WallpaperContentType = "image/jpeg"
)

// processWallpaper turns an uploaded JPEG, PNG or WebP into the stored wallpaper with the
// shared upload pipeline (platform/imaging, also used for avatars): type from the bytes,
// dimension limits, EXIF orientation applied. The whole image is kept (no crop: the page covers
// it with CSS), scaled to at most WallpaperMaxSide and re-encoded as JPEG, which drops every
// metadata block (GPS position, camera details).
func processWallpaper(data []byte) ([]byte, error) {
	src, orientation, err := imaging.Decode(data, imaging.Limits{MaxSide: wallpaperMaxSide, MaxPixels: wallpaperMaxPixels})
	if err != nil {
		return nil, err
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if longest := max(w, h); longest > WallpaperMaxSide {
		w = max(1, w*WallpaperMaxSide/longest)
		h = max(1, h*WallpaperMaxSide/longest)
	}
	// White background: JPEG has no transparency.
	scaled := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(scaled, scaled.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), src, b, draw.Over, nil)
	return imaging.EncodeJPEG(imaging.Orient(scaled, orientation), wallpaperQuality)
}

// Wallpaper keys: wallpapers/<user-id>/<32 random hex>.jpg. Every upload is a new URL (safe to
// cache forever) and the URL is unguessable.
func newWallpaperKey(userID string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	key := "wallpapers/" + userID + "/" + hex.EncodeToString(b) + ".jpg"
	if err := storage.ValidateKey(key); err != nil {
		return "", err
	}
	return key, nil
}
