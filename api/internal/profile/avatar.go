package profile

import (
	"image"
	"image/color"

	"golang.org/x/image/draw"

	"github.com/samandar-hodiev/sat-platform/api/internal/platform/imaging"
)

const (
	// AvatarMaxBytes is the largest accepted upload (decided for 2.01: 5 MB).
	AvatarMaxBytes = 5 << 20
	// AvatarSize is the side of the stored square avatar, in pixels.
	AvatarSize = 512
	// avatarMaxPixels guards against decompression bombs: a small file that decodes into a
	// huge bitmap. 40 MP covers every phone camera.
	avatarMaxPixels = 40_000_000
	avatarMaxSide   = 12_000
	avatarQuality   = 85
	// AvatarContentType is what every stored avatar is: uploads are re-encoded, never kept as is.
	AvatarContentType = "image/jpeg"
)

var (
	errAvatarType    = imaging.ErrUnsupportedType
	errAvatarCorrupt = imaging.ErrCorrupt
	errAvatarTooBig  = imaging.ErrTooLarge
)

// jpegOrientation: the shared EXIF reader (platform/imaging).
var jpegOrientation = imaging.JPEGOrientation

// processAvatar turns an uploaded JPEG, PNG or WebP into the stored avatar: the type is
// detected from the bytes (never the file name or the client's Content-Type), the camera
// orientation is applied, the centre square is cropped and scaled to at most AvatarSize, and
// the result is re-encoded as JPEG. Re-encoding drops every metadata block (EXIF GPS position,
// camera details) and anything hidden after the image data.
func processAvatar(data []byte) ([]byte, error) {
	src, orientation, err := imaging.Decode(data, imaging.Limits{MaxSide: avatarMaxSide, MaxPixels: avatarMaxPixels})
	if err != nil {
		return nil, err
	}

	// Crop the centre square of the image as it is displayed (after orientation). Rotations
	// by 90° swap width and height, so the square is taken from the stored pixels first and
	// rotated after; a square stays a square.
	b := src.Bounds()
	side := min(b.Dx(), b.Dy())
	crop := image.Rect(0, 0, side, side).Add(image.Pt(b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2))
	out := min(side, AvatarSize)

	// White background: JPEG has no transparency, so transparent PNG/WebP pixels become white.
	scaled := image.NewRGBA(image.Rect(0, 0, out, out))
	draw.Draw(scaled, scaled.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), src, crop, draw.Over, nil)

	return imaging.EncodeJPEG(orient(scaled, orientation), avatarQuality)
}

// orient applies an EXIF orientation (1–8) to a square image.
func orient(img *image.RGBA, o int) image.Image { return imaging.Orient(img, o) }
