package preferences

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
)

var ctx = context.Background()

func ptr[T any](v T) *T { return &v }

func wantCode(t *testing.T, err error, code apperr.Code) *apperr.Error {
	t.Helper()
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != code {
		t.Fatalf("got %v, want %s", err, code)
	}
	return ae
}

func wantFields(t *testing.T, err error, fields ...string) {
	t.Helper()
	ae := wantCode(t, err, apperr.CodeValidation)
	got, _ := ae.Details["fields"].(map[string]string)
	if len(got) != len(fields) {
		t.Fatalf("fields %v, want %v", got, fields)
	}
	for _, f := range fields {
		if got[f] == "" {
			t.Fatalf("fields %v, missing %q", got, f)
		}
	}
}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

var keyPattern = regexp.MustCompile(`^wallpapers/` + userA + `/[0-9a-f]{32}\.jpg$`)

func fileExists(t *testing.T, root, key string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(key)))
	return err == nil
}

func TestGetDefaultsWhenNothingStored(t *testing.T) {
	f := newFixture(t)
	v, err := f.svc.Get(ctx, userA)
	if err != nil {
		t.Fatal(err)
	}
	want := Defaults()
	if v.Stored || v.Theme != "light" || v.Accent != "neutral" || v.Density != "comfortable" ||
		v.Motion != "system" || v.SidebarCollapsed || v.Light != want.Light || v.Dark != want.Dark ||
		v.Opacity != DefaultOpacity || v.Blur != 0 {
		t.Fatalf("%+v", v)
	}
}

func TestUpdateAppearancePartially(t *testing.T) {
	f := newFixture(t)
	v, err := f.svc.Update(ctx, userA, PatchInput{Theme: ptr("dark"), Accent: ptr("teal")})
	if err != nil {
		t.Fatal(err)
	}
	if !v.Stored || v.Theme != "dark" || v.Accent != "teal" || v.Density != "comfortable" {
		t.Fatalf("%+v", v)
	}
	// A later patch keeps the earlier fields.
	v, err = f.svc.Update(ctx, userA, PatchInput{Density: ptr("compact"), Motion: ptr("reduced"), SidebarCollapsed: ptr(true)})
	if err != nil {
		t.Fatal(err)
	}
	if v.Theme != "dark" || v.Accent != "teal" || v.Density != "compact" || v.Motion != "reduced" || !v.SidebarCollapsed {
		t.Fatalf("%+v", v)
	}
	got, _ := f.svc.Get(ctx, userA)
	if !got.Stored || got.Prefs.Theme != "dark" || !got.SidebarCollapsed {
		t.Fatalf("%+v", got)
	}
	// Another learner is unaffected.
	other, _ := f.svc.Get(ctx, userB)
	if other.Stored || other.Theme != "light" {
		t.Fatalf("%+v", other)
	}
}

func TestEmptyUpdateCreatesNothing(t *testing.T) {
	f := newFixture(t)
	v, err := f.svc.Update(ctx, userA, PatchInput{})
	if err != nil || v.Stored {
		t.Fatalf("%+v %v", v, err)
	}
}

func TestUpdateValidation(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Update(ctx, userA, PatchInput{
		Theme: ptr("sepia"), Accent: ptr("green"), Density: ptr("cozy"), Motion: ptr("fast"),
		WallpaperOpacity: ptr(9), WallpaperBlur: ptr(21),
		WallpaperLight: &ChoiceInput{Source: "photo"},
		WallpaperDark:  &ChoiceInput{Source: SourceGallery, GalleryID: ptr("unknown")},
	})
	wantFields(t, err, "theme", "accent", "density", "motion", "wallpaper.opacity", "wallpaper.blur",
		"wallpaper.light.source", "wallpaper.dark.gallery_id")

	_, err = f.svc.Update(ctx, userA, PatchInput{
		WallpaperLight:   &ChoiceInput{Source: SourceGallery},
		WallpaperDark:    &ChoiceInput{Source: SourceNone, GalleryID: ptr("dots")},
		WallpaperOpacity: ptr(81),
	})
	wantFields(t, err, "wallpaper.light.gallery_id", "wallpaper.dark.gallery_id", "wallpaper.opacity")

	if v, _ := f.svc.Get(ctx, userA); v.Stored {
		t.Fatal("invalid update must not write")
	}
	// Bounds are inclusive.
	v, err := f.svc.Update(ctx, userA, PatchInput{WallpaperOpacity: ptr(MaxOpacity), WallpaperBlur: ptr(MaxBlur)})
	if err != nil || v.Opacity != MaxOpacity || v.Blur != MaxBlur {
		t.Fatalf("%+v %v", v, err)
	}
	v, err = f.svc.Update(ctx, userA, PatchInput{WallpaperOpacity: ptr(MinOpacity), WallpaperBlur: ptr(0)})
	if err != nil || v.Opacity != MinOpacity || v.Blur != 0 {
		t.Fatalf("%+v %v", v, err)
	}
}

func TestGalleryChoicePerMode(t *testing.T) {
	f := newFixture(t)
	v, err := f.svc.Update(ctx, userA, PatchInput{
		WallpaperLight: &ChoiceInput{Source: SourceGallery, GalleryID: ptr("dots")},
		WallpaperDark:  &ChoiceInput{Source: SourceGallery, GalleryID: ptr("aurora")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if v.Light != (Wallpaper{Source: SourceGallery, GalleryID: "dots"}) || v.Dark != (Wallpaper{Source: SourceGallery, GalleryID: "aurora"}) {
		t.Fatalf("%+v", v)
	}
	v, err = f.svc.Update(ctx, userA, PatchInput{WallpaperLight: &ChoiceInput{Source: SourceNone}})
	if err != nil {
		t.Fatal(err)
	}
	if v.Light != (Wallpaper{Source: SourceNone}) || v.Dark.GalleryID != "aurora" {
		t.Fatalf("%+v", v)
	}
}

func TestChooseCustomWithoutUploadConflicts(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Update(ctx, userA, PatchInput{WallpaperDark: &ChoiceInput{Source: SourceCustom}, Theme: ptr("dark")})
	ae := wantCode(t, err, apperr.CodeConflict)
	if ae.Details["reason"] != "wallpaper_not_uploaded" {
		t.Fatalf("%+v", ae.Details)
	}
	if v, _ := f.svc.Get(ctx, userA); v.Stored {
		t.Fatal("nothing must be written")
	}
}

func TestUploadReplaceSwitchAndRemove(t *testing.T) {
	f := newFixture(t)
	v, err := f.svc.UploadWallpaper(ctx, userA, ModeLight, pngBytes(t, 300, 200))
	if err != nil {
		t.Fatal(err)
	}
	first := v.Light.Key
	if v.Light.Source != SourceCustom || !keyPattern.MatchString(first) || !fileExists(t, f.root, first) || v.Dark.Source != SourceNone {
		t.Fatalf("%+v", v)
	}

	// A new upload replaces the old file.
	v, err = f.svc.UploadWallpaper(ctx, userA, ModeLight, pngBytes(t, 300, 200))
	if err != nil {
		t.Fatal(err)
	}
	if v.Light.Key == first || fileExists(t, f.root, first) || !fileExists(t, f.root, v.Light.Key) {
		t.Fatalf("old file kept or new missing: %+v", v)
	}
	second := v.Light.Key

	// Switching to a gallery pattern keeps the upload, so the learner can switch back.
	v, err = f.svc.Update(ctx, userA, PatchInput{WallpaperLight: &ChoiceInput{Source: SourceGallery, GalleryID: ptr("grid")}})
	if err != nil {
		t.Fatal(err)
	}
	if v.Light.Source != SourceGallery || v.Light.Key != second {
		t.Fatalf("%+v", v)
	}
	v, err = f.svc.Update(ctx, userA, PatchInput{WallpaperLight: &ChoiceInput{Source: SourceCustom}})
	if err != nil || v.Light.Source != SourceCustom {
		t.Fatalf("%+v %v", v, err)
	}

	// Removing the chosen upload falls back to no wallpaper and deletes the file.
	v, err = f.svc.RemoveWallpaper(ctx, userA, ModeLight)
	if err != nil {
		t.Fatal(err)
	}
	if v.Light != (Wallpaper{Source: SourceNone}) || fileExists(t, f.root, second) {
		t.Fatalf("%+v", v)
	}
	// Removing again is fine.
	if _, err := f.svc.RemoveWallpaper(ctx, userA, ModeLight); err != nil {
		t.Fatal(err)
	}
}

func TestRemoveUploadKeepsGalleryChoice(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.UploadWallpaper(ctx, userA, ModeDark, pngBytes(t, 40, 40)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Update(ctx, userA, PatchInput{WallpaperDark: &ChoiceInput{Source: SourceGallery, GalleryID: ptr("waves")}}); err != nil {
		t.Fatal(err)
	}
	v, err := f.svc.RemoveWallpaper(ctx, userA, ModeDark)
	if err != nil {
		t.Fatal(err)
	}
	if v.Dark != (Wallpaper{Source: SourceGallery, GalleryID: "waves"}) {
		t.Fatalf("%+v", v)
	}
}

func TestUploadRejectsBadFiles(t *testing.T) {
	f := newFixture(t)
	for name, data := range map[string][]byte{
		"text":      []byte("hello, not an image"),
		"svg":       []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`),
		"truncated": pngBytes(t, 50, 50)[:40],
		"too big":   bytes.Repeat([]byte{0}, WallpaperMaxBytes+1),
	} {
		_, err := f.svc.UploadWallpaper(ctx, userA, ModeLight, data)
		if err == nil {
			t.Fatalf("%s: accepted", name)
		}
		wantFields(t, err, "file")
	}
	if v, _ := f.svc.Get(ctx, userA); v.Stored {
		t.Fatal("rejected upload must not write")
	}
	entries, _ := os.ReadDir(f.root)
	if len(entries) != 0 {
		t.Fatalf("files written: %v", entries)
	}
}

func TestWritesForUnknownAccount(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Update(ctx, userGone, PatchInput{Theme: ptr("dark")})
	wantCode(t, err, apperr.CodeUnauthorized)
	_, err = f.svc.UploadWallpaper(ctx, userGone, ModeLight, pngBytes(t, 10, 10))
	wantCode(t, err, apperr.CodeUnauthorized)
	// The stored file is cleaned up when the row cannot be written.
	if orphans, _ := filepath.Glob(filepath.Join(f.root, "wallpapers", userGone, "*")); len(orphans) != 0 {
		t.Fatalf("orphan files: %v", orphans)
	}
	_, err = f.svc.RemoveWallpaper(ctx, userGone, ModeLight)
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestConcurrentUploadsLeaveOneFile(t *testing.T) {
	f := newFixture(t)
	data := pngBytes(t, 64, 64)
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.svc.UploadWallpaper(ctx, userA, ModeLight, data); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	v, _ := f.svc.Get(ctx, userA)
	images, _ := filepath.Glob(filepath.Join(f.root, "wallpapers", userA, "*.jpg"))
	if len(images) != 1 || "wallpapers/"+userA+"/"+filepath.Base(images[0]) != v.Light.Key {
		t.Fatalf("files %v, key %s", images, v.Light.Key)
	}
}

func TestParseMode(t *testing.T) {
	for in, ok := range map[string]bool{"light": true, "dark": true, "Light": false, "system": false, "": false} {
		if _, got := ParseMode(in); got != ok {
			t.Errorf("%q: %v", in, got)
		}
	}
}
