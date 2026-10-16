// Package imaging holds the shared, safe part of image uploads (avatars 2.01, wallpapers 2.11):
// the type is detected from the bytes (never the file name or the client's Content-Type), the
// dimensions are checked before decoding (decompression bombs), and the EXIF orientation of
// JPEG camera photos is read and applied. Each module decides its own crop, size and quality and
// re-encodes the result, which drops every metadata block.
package imaging

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"

	"golang.org/x/image/webp"
)

var (
	ErrUnsupportedType = errors.New("unsupported image type")
	ErrCorrupt         = errors.New("image cannot be read")
	ErrTooLarge        = errors.New("image dimensions too large")
)

// Limits bound the decoded bitmap: a small file can decode into a huge one.
type Limits struct {
	MaxSide   int
	MaxPixels int
}

// Decode reads a JPEG, PNG or WebP upload. It returns the image as stored and its EXIF
// orientation (1–8; 1 for anything but a JPEG with an Orientation tag).
func Decode(data []byte, lim Limits) (image.Image, int, error) {
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
		return nil, 0, ErrUnsupportedType
	}

	cfg, err := config(data)
	if err != nil {
		return nil, 0, ErrCorrupt
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > lim.MaxSide || cfg.Height > lim.MaxSide ||
		cfg.Width*cfg.Height > lim.MaxPixels {
		return nil, 0, ErrTooLarge
	}
	img, err := decode(data)
	if err != nil {
		return nil, 0, ErrCorrupt
	}
	orientation := 1
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8}) {
		orientation = JPEGOrientation(data)
	}
	return img, orientation, nil
}

// EncodeJPEG re-encodes img; the output carries no metadata.
func EncodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Orient applies an EXIF orientation (1–8) to img. Orientations 5–8 swap width and height.
// Any other value returns img unchanged.
func Orient(img *image.RGBA, o int) *image.RGBA {
	if o < 2 || o > 8 {
		return img
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2: // mirror horizontal
				dx, dy = w-1-x, y
			case 3: // rotate 180
				dx, dy = w-1-x, h-1-y
			case 4: // mirror vertical
				dx, dy = x, h-1-y
			case 5: // transpose
				dx, dy = y, x
			case 6: // rotate 90 clockwise
				dx, dy = h-1-y, x
			case 7: // transverse
				dx, dy = h-1-y, w-1-x
			case 8: // rotate 90 counter-clockwise
				dx, dy = y, w-1-x
			}
			dst.SetRGBA(dx, dy, img.RGBAAt(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}

// JPEGOrientation reads the EXIF Orientation tag (0x0112) from a JPEG's APP1 segment. It returns
// 1 (as stored) when there is no EXIF block or it cannot be read.
func JPEGOrientation(data []byte) int {
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
