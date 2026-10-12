package profile

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

var (
	red  = color.RGBA{220, 30, 30, 255}
	blue = color.RGBA{30, 30, 220, 255}
)

// halves draws a w×h image: left half red, right half blue.
func halves(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w/2 {
				img.Set(x, y, red)
			} else {
				img.Set(x, y, blue)
			}
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// withOrientation inserts an EXIF APP1 segment with the given Orientation after SOI.
func withOrientation(jpg []byte, o uint16) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, // header, IFD at 8, one entry
		0x01, 0x12, 0, 3, 0, 0, 0, 1, byte(o >> 8), byte(o), 0, 0, // Orientation SHORT
		0, 0, 0, 0} // no next IFD
	payload := append([]byte("Exif\x00\x00"), tiff...)
	size := len(payload) + 2
	seg := append([]byte{0xFF, 0xE1, byte(size >> 8), byte(size)}, payload...)
	out := append([]byte{}, jpg[:2]...)
	out = append(out, seg...)
	return append(out, jpg[2:]...)
}

func decodeOut(t *testing.T, b []byte) image.Image {
	t.Helper()
	if !bytes.HasPrefix(b, []byte{0xFF, 0xD8}) {
		t.Fatal("output is not a JPEG")
	}
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func near(c color.Color, want color.RGBA) bool {
	r, g, b, _ := c.RGBA()
	d := func(a uint32, w uint8) int { return int(a>>8) - int(w) }
	abs := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	return abs(d(r, want.R)) < 40 && abs(d(g, want.G)) < 40 && abs(d(b, want.B)) < 40
}

func TestProcessAvatarCropsCentreSquareAndScales(t *testing.T) {
	out, err := processAvatar(encodeJPEG(t, halves(1600, 800)))
	if err != nil {
		t.Fatal(err)
	}
	img := decodeOut(t, out)
	if b := img.Bounds(); b.Dx() != AvatarSize || b.Dy() != AvatarSize {
		t.Fatalf("size %v", b)
	}
	// Centre 800×800 of the 1600 wide image: left half red, right half blue.
	if !near(img.At(50, 256), red) || !near(img.At(460, 256), blue) {
		t.Fatalf("unexpected crop: %v %v", img.At(50, 256), img.At(460, 256))
	}
}

func TestProcessAvatarKeepsSmallImagesSmall(t *testing.T) {
	out, err := processAvatar(encodePNG(t, halves(120, 200)))
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeOut(t, out).Bounds(); b.Dx() != 120 || b.Dy() != 120 {
		t.Fatalf("size %v", b)
	}
}

func TestProcessAvatarAppliesExifOrientationAndStripsMetadata(t *testing.T) {
	src := withOrientation(encodeJPEG(t, halves(200, 100)), 6) // rotate 90° clockwise to display
	if jpegOrientation(src) != 6 {
		t.Fatal("test JPEG has no orientation")
	}
	out, err := processAvatar(src)
	if err != nil {
		t.Fatal(err)
	}
	img := decodeOut(t, out)
	// Left (red) half rotated clockwise ends up on top.
	if !near(img.At(50, 10), red) || !near(img.At(50, 90), blue) {
		t.Fatalf("orientation not applied: top %v bottom %v", img.At(50, 10), img.At(50, 90))
	}
	if bytes.Contains(out, []byte("Exif")) {
		t.Fatal("EXIF kept in output")
	}
}

func TestOrientAllValues(t *testing.T) {
	// 2×2 image with distinct pixels; check where the top-left pixel ends up.
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	tl := color.RGBA{1, 2, 3, 255}
	src.SetRGBA(0, 0, tl)
	want := map[int]image.Point{1: {0, 0}, 2: {1, 0}, 3: {1, 1}, 4: {0, 1}, 5: {0, 0}, 6: {1, 0}, 7: {1, 1}, 8: {0, 1}}
	for o, p := range want {
		got := orient(src, o).(interface{ RGBAAt(x, y int) color.RGBA })
		if got.RGBAAt(p.X, p.Y) != tl {
			t.Errorf("orientation %d: top-left pixel not at %v", o, p)
		}
	}
}

func TestProcessAvatarFlattensTransparencyOnWhite(t *testing.T) {
	out, err := processAvatar(encodePNG(t, image.NewNRGBA(image.Rect(0, 0, 64, 64))))
	if err != nil {
		t.Fatal(err)
	}
	if !near(decodeOut(t, out).At(32, 32), color.RGBA{255, 255, 255, 255}) {
		t.Fatal("transparent pixels are not white")
	}
}

func TestProcessAvatarWebP(t *testing.T) {
	// 1×1 lossless WebP.
	webp, _ := base64.StdEncoding.DecodeString("UklGRhoAAABXRUJQVlA4TA0AAAAvAAAAEAcQERGIiP4HAA==")
	out, err := processAvatar(webp)
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeOut(t, out).Bounds(); b.Dx() != 1 {
		t.Fatalf("size %v", b)
	}
}

func TestProcessAvatarRejects(t *testing.T) {
	var gifBuf bytes.Buffer
	_ = gif.Encode(&gifBuf, image.NewPaletted(image.Rect(0, 0, 4, 4), []color.Color{color.White}), nil)
	jpg := encodeJPEG(t, halves(40, 40))

	cases := map[string]struct {
		data []byte
		want error
	}{
		"gif":         {gifBuf.Bytes(), errAvatarType},
		"svg":         {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), errAvatarType},
		"html":        {[]byte("<html><body>hi</body></html>"), errAvatarType},
		"empty":       {nil, errAvatarType},
		"truncated":   {jpg[:len(jpg)/3], errAvatarCorrupt},
		"header only": {[]byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 16, 'J', 'F', 'I', 'F', 0}, errAvatarCorrupt},
		"too wide":    {encodePNG(t, image.NewGray(image.Rect(0, 0, avatarMaxSide+1, 1))), errAvatarTooBig},
		"too many px": {encodePNG(t, image.NewGray(image.Rect(0, 0, 8000, 6000))), errAvatarTooBig},
	}
	for name, tc := range cases {
		if _, err := processAvatar(tc.data); !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", name, err, tc.want)
		}
	}
}

func TestJPEGOrientationMalformed(t *testing.T) {
	for _, b := range [][]byte{
		{0xFF, 0xD8},
		{0xFF, 0xD8, 0xFF, 0xE1, 0xFF, 0xFF}, // segment longer than data
		{0xFF, 0xD8, 0xFF, 0xE1, 0, 10, 'E', 'x', 'i', 'f', 0, 0, 'X', 'X'}, // bad byte order
		{0xFF, 0xD8, 0x00},
	} {
		if o := jpegOrientation(b); o != 1 {
			t.Errorf("%v: %d", b, o)
		}
	}
}
