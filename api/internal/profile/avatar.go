package profile

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
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
	errAvatarType    = errors.New("unsupported image type")
	errAvatarCorrupt = errors.New("image cannot be read")
	errAvatarTooBig  = errors.New("image dimensions too large")
)

// processAvatar turns an uploaded JPEG, PNG or WebP into the stored avatar: the type is
// detected from the bytes (never the file name or the client's Content-Type), the camera
// orientation is applied, the centre square is cropped and scaled to at most AvatarSize, and
// the result is re-encoded as JPEG. Re-encoding drops every metadata block (EXIF GPS position,
// camera details) and anything hidden after the image data.
func processAvatar(data []byte) ([]byte, error) {
	var decode func([]byte) (image.Image, error)
	var config func([]byte) (image.Config, error)
	switch http.DetectContentType(data) {
	case "image/jpeg":
		decode = func(b []byte) (image.Image, error) { return jpeg.Decode(bytes.NewReader(b)) }
		config = func(b []byte) (image.Config, error) { return jpeg.DecodeConfig(bytes.NewReader(b)) }
	case "image/png":
		decode = func(b []byte) (image.Image, error) { return png.Decode(bytes.NewReader(b)) }
		config = func(b []byte) (image.Config, error) { return png.DecodeConfig(bytes.NewReader(b)) }
	case "image/webp":
		decode = func(b []byte) (image.Image, error) { return webp.Decode(bytes.NewReader(b)) }
		config = func(b []byte) (image.Config, error) { return webp.DecodeConfig(bytes.NewReader(b)) }
	default:
		return nil, errAvatarType
	}

	cfg, err := config(data)
	if err != nil {
		return nil, errAvatarCorrupt
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > avatarMaxSide || cfg.Height > avatarMaxSide ||
		cfg.Width*cfg.Height > avatarMaxPixels {
		return nil, errAvatarTooBig
	}
	src, err := decode(data)
	if err != nil {
		return nil, errAvatarCorrupt
	}

	orientation := 1
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8}) {
		orientation = jpegOrientation(data)
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

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, orient(scaled, orientation), &jpeg.Options{Quality: avatarQuality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// orient applies an EXIF orientation (1–8) to a square image.
func orient(img *image.RGBA, o int) image.Image {
	if o < 2 || o > 8 {
		return img
	}
	n := img.Bounds().Dx()
	dst := image.NewRGBA(img.Bounds())
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			var dx, dy int
			switch o {
			case 2: // mirror horizontal
				dx, dy = n-1-x, y
			case 3: // rotate 180
				dx, dy = n-1-x, n-1-y
			case 4: // mirror vertical
				dx, dy = x, n-1-y
			case 5: // transpose
				dx, dy = y, x
			case 6: // rotate 90 clockwise
				dx, dy = n-1-y, x
			case 7: // transverse
				dx, dy = n-1-y, n-1-x
			case 8: // rotate 90 counter-clockwise
				dx, dy = y, n-1-x
			}
			dst.SetRGBA(dx, dy, img.RGBAAt(x, y))
		}
	}
	return dst
}

// jpegOrientation reads the EXIF Orientation tag (0x0112) from a JPEG's APP1 segment. It returns
// 1 (as stored) when there is no EXIF block or it cannot be read.
func jpegOrientation(data []byte) int {
	i := 2 // after SOI
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xD8 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			i += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 { // start of scan / end: no more metadata
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2:]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		seg := data[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) > 6 && string(seg[:6]) == "Exif\x00\x00" {
			return tiffOrientation(seg[6:])
		}
		i += 2 + size
	}
	return 1
}

func tiffOrientation(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	ifd := int(bo.Uint32(t[4:]))
	if ifd < 8 || ifd+2 > len(t) {
		return 1
	}
	count := int(bo.Uint16(t[ifd:]))
	for k := 0; k < count; k++ {
		e := ifd + 2 + k*12
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 && bo.Uint16(t[e+2:]) == 3 { // SHORT
			if v := int(bo.Uint16(t[e+8:])); v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}
