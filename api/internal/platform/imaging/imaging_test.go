package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

var lim = Limits{MaxSide: 4000, MaxPixels: 4_000_000}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withOrientation inserts an EXIF APP1 segment with the given Orientation after SOI.
func withOrientation(jpg []byte, o uint16, littleEndian bool) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1,
		0x01, 0x12, 0, 3, 0, 0, 0, 1, byte(o >> 8), byte(o), 0, 0,
		0, 0, 0, 0}
	if littleEndian {
		tiff = []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0,
			0x12, 0x01, 3, 0, 1, 0, 0, 0, byte(o), byte(o >> 8), 0, 0,
			0, 0, 0, 0}
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	size := len(payload) + 2
	seg := append([]byte{0xFF, 0xE1, byte(size >> 8), byte(size)}, payload...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func TestDecodeDetectsTypeFromBytes(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 30, 20))
	for name, data := range map[string][]byte{"png": encodePNG(t, src), "jpeg": encodeJPEG(t, src)} {
		img, o, err := Decode(data, lim)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if img.Bounds().Dx() != 30 || img.Bounds().Dy() != 20 || o != 1 {
			t.Errorf("%s: got %v orientation %d", name, img.Bounds(), o)
		}
	}
}

func TestDecodeRejects(t *testing.T) {
	var gifBuf bytes.Buffer
	if err := gif.Encode(&gifBuf, image.NewPaletted(image.Rect(0, 0, 4, 4), []color.Color{color.Black}), nil); err != nil {
		t.Fatal(err)
	}
	jpg := encodeJPEG(t, image.NewRGBA(image.Rect(0, 0, 64, 64)))
	cases := map[string]struct {
		data []byte
		want error
	}{
		"gif":       {gifBuf.Bytes(), ErrUnsupportedType},
		"svg":       {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), ErrUnsupportedType},
		"empty":     {nil, ErrUnsupportedType},
		"truncated": {jpg[:len(jpg)/3], ErrCorrupt},
		"too wide":  {encodePNG(t, image.NewGray(image.Rect(0, 0, lim.MaxSide+1, 1))), ErrTooLarge},
		"too many":  {encodePNG(t, image.NewGray(image.Rect(0, 0, 2001, 2000))), ErrTooLarge},
	}
	for name, tc := range cases {
		if _, _, err := Decode(tc.data, lim); !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", name, err, tc.want)
		}
	}
}

func TestDecodeReadsOrientation(t *testing.T) {
	jpg := encodeJPEG(t, image.NewRGBA(image.Rect(0, 0, 8, 8)))
	for _, le := range []bool{false, true} {
		_, o, err := Decode(withOrientation(jpg, 6, le), lim)
		if err != nil || o != 6 {
			t.Errorf("littleEndian=%v: orientation %d, err %v", le, o, err)
		}
	}
	if JPEGOrientation(withOrientation(jpg, 9, false)) != 1 {
		t.Error("out-of-range orientation should be 1")
	}
	if JPEGOrientation([]byte{0xFF, 0xD8, 0xFF}) != 1 {
		t.Error("short data should be 1")
	}
}

func TestOrientNonSquare(t *testing.T) {
	// 3×2 image; mark the top-left pixel and check where it lands and the output size.
	src := image.NewRGBA(image.Rect(0, 0, 3, 2))
	mark := color.RGBA{1, 2, 3, 255}
	src.SetRGBA(0, 0, mark)
	type want struct {
		w, h int
		at   image.Point
	}
	cases := map[int]want{
		1: {3, 2, image.Pt(0, 0)}, 2: {3, 2, image.Pt(2, 0)}, 3: {3, 2, image.Pt(2, 1)},
		4: {3, 2, image.Pt(0, 1)}, 5: {2, 3, image.Pt(0, 0)}, 6: {2, 3, image.Pt(1, 0)},
		7: {2, 3, image.Pt(1, 2)}, 8: {2, 3, image.Pt(0, 2)},
	}
	for o, w := range cases {
		got := Orient(src, o)
		if got.Bounds().Dx() != w.w || got.Bounds().Dy() != w.h {
			t.Errorf("orientation %d: size %v", o, got.Bounds())
			continue
		}
		if got.RGBAAt(w.at.X, w.at.Y) != mark {
			t.Errorf("orientation %d: top-left pixel not at %v", o, w.at)
		}
	}
}

func TestEncodeJPEGDropsMetadata(t *testing.T) {
	jpg := withOrientation(encodeJPEG(t, image.NewRGBA(image.Rect(0, 0, 8, 8))), 6, false)
	img, _, err := Decode(jpg, lim)
	if err != nil {
		t.Fatal(err)
	}
	out, err := EncodeJPEG(img, 80)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("Exif")) {
		t.Error("EXIF kept")
	}
}
