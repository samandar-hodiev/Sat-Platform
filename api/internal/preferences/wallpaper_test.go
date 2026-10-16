package preferences

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func decodeJPEG(t *testing.T, b []byte) image.Image {
	t.Helper()
	img, err := jpeg.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("output is not a JPEG: %v", err)
	}
	return img
}

func TestProcessWallpaperScalesDownKeepingAspect(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4000, 2000))
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, nil); err != nil {
		t.Fatal(err)
	}
	out, err := processWallpaper(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeJPEG(t, out).Bounds(); b.Dx() != WallpaperMaxSide || b.Dy() != WallpaperMaxSide/2 {
		t.Fatalf("got %v", b)
	}
}

func TestProcessWallpaperKeepsSmallImagesSize(t *testing.T) {
	out, err := processWallpaper(pngBytes(t, 640, 360))
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeJPEG(t, out).Bounds(); b.Dx() != 640 || b.Dy() != 360 {
		t.Fatalf("got %v", b)
	}
}

func TestProcessWallpaperAppliesOrientationAndStripsExif(t *testing.T) {
	// A 40×20 landscape JPEG tagged "rotate 90° clockwise" is shown as 20×40 portrait.
	src := image.NewRGBA(image.Rect(0, 0, 40, 20))
	for y := 0; y < 20; y++ {
		for x := 0; x < 40; x++ {
			src.Set(x, y, color.RGBA{200, 30, 30, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	jpg := buf.Bytes()
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, 6, 0, 0, 0, 0, 0, 0}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := append([]byte{0xFF, 0xE1, byte((len(payload) + 2) >> 8), byte(len(payload) + 2)}, payload...)
	tagged := append(append(append([]byte{}, jpg[:2]...), seg...), jpg[2:]...)

	out, err := processWallpaper(tagged)
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeJPEG(t, out).Bounds(); b.Dx() != 20 || b.Dy() != 40 {
		t.Fatalf("got %v", b)
	}
	if bytes.Contains(out, []byte("Exif")) {
		t.Fatal("EXIF kept")
	}
}

// The web draws the gallery patterns; both sides must list the same ids.
func TestGalleryMatchesWeb(t *testing.T) {
	path := filepath.Join("..", "..", "..", "web", "src", "features", "settings", "wallpaper", "gallery.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("web gallery not found: %v", err)
	}
	var web []string
	if err := json.Unmarshal(raw, &web); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(web, GalleryIDs) {
		t.Fatalf("web %v, api %v", web, GalleryIDs)
	}
}
