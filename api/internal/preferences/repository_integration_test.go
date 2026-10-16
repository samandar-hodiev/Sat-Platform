package preferences

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/samandar-hodiev/sat-platform/api/internal/platform/migrate"
	"github.com/samandar-hodiev/sat-platform/api/migrations"
)

// TestPostgresStore runs the service and HTTP scenarios against PostgreSQL. It needs a
// disposable database (TEST_DATABASE_URL) and deletes every user in it between scenarios.
func TestPostgresStore(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	mg, err := migrate.New(url, migrations.FS, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := mg.Up(); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	_ = mg.Close()

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	saved := newStore
	newStore = func(t *testing.T, users ...string) Store {
		t.Helper()
		// DELETE, not TRUNCATE: audit_logs is append-only and only allows its FK to go NULL.
		if _, err := pool.Exec(ctx, "DELETE FROM users"); err != nil {
			t.Fatal(err)
		}
		for i, id := range users {
			if _, err := pool.Exec(ctx, `INSERT INTO users (id, email, email_verified_at) VALUES ($1, $2, now())`,
				id, "learner"+string(rune('a'+i))+"@example.com"); err != nil {
				t.Fatal(err)
			}
		}
		return NewPostgresStore(pool)
	}
	t.Cleanup(func() { newStore = saved })

	scenarios := map[string]func(*testing.T){
		"defaults":                 TestGetDefaultsWhenNothingStored,
		"partial update":           TestUpdateAppearancePartially,
		"empty update":             TestEmptyUpdateCreatesNothing,
		"validation":               TestUpdateValidation,
		"gallery per mode":         TestGalleryChoicePerMode,
		"custom without upload":    TestChooseCustomWithoutUploadConflicts,
		"upload replace remove":    TestUploadReplaceSwitchAndRemove,
		"remove keeps gallery":     TestRemoveUploadKeepsGalleryChoice,
		"unknown account":          TestWritesForUnknownAccount,
		"concurrent uploads":       TestConcurrentUploadsLeaveOneFile,
		"http get and patch":       TestHTTPGetAndPatch,
		"http errors":              TestHTTPPatchErrors,
		"http upload serve remove": TestHTTPWallpaperUploadServeRemove,
		"concurrent patches":       testConcurrentPatches,
		"constraints":              func(t *testing.T) { testConstraints(t, pool) },
	}
	for name, fn := range scenarios {
		t.Run(name, fn)
	}
}

// Concurrent partial updates of different fields all survive (row lock, no lost update).
func testConcurrentPatches(t *testing.T) {
	f := newFixture(t)
	patches := []PatchInput{
		{Theme: ptr("dark")}, {Accent: ptr("rose")}, {Density: ptr("compact")}, {Motion: ptr("full")},
		{SidebarCollapsed: ptr(true)}, {WallpaperOpacity: ptr(70)}, {WallpaperBlur: ptr(12)},
		{WallpaperLight: &ChoiceInput{Source: SourceGallery, GalleryID: ptr("contours")}},
	}
	var wg sync.WaitGroup
	for _, p := range patches {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.svc.Update(ctx, userA, p); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	v, err := f.svc.Get(ctx, userA)
	if err != nil {
		t.Fatal(err)
	}
	if v.Theme != "dark" || v.Accent != "rose" || v.Density != "compact" || v.Motion != "full" || !v.SidebarCollapsed ||
		v.Opacity != 70 || v.Blur != 12 || v.Light.GalleryID != "contours" {
		t.Fatalf("lost update: %+v", v)
	}
}

// The CHECK constraints hold the same rules as the service, for writes from anywhere else.
func testConstraints(t *testing.T, pool *pgxpool.Pool) {
	newFixture(t)
	key := "wallpapers/" + userA + "/0123456789abcdef0123456789abcdef.jpg"
	for name, stmt := range map[string]string{
		"source":                `INSERT INTO user_preferences (user_id, wallpaper_light_source) VALUES ($1, 'photo')`,
		"gallery without id":    `INSERT INTO user_preferences (user_id, wallpaper_dark_source) VALUES ($1, 'gallery')`,
		"id without gallery":    `INSERT INTO user_preferences (user_id, wallpaper_light_gallery_id) VALUES ($1, 'dots')`,
		"gallery id format":     `INSERT INTO user_preferences (user_id, wallpaper_light_source, wallpaper_light_gallery_id) VALUES ($1, 'gallery', '../x')`,
		"custom without key":    `INSERT INTO user_preferences (user_id, wallpaper_dark_source) VALUES ($1, 'custom')`,
		"key format":            `INSERT INTO user_preferences (user_id, wallpaper_light_key) VALUES ($1, 'avatars/x.jpg')`,
		"opacity low":           `INSERT INTO user_preferences (user_id, wallpaper_opacity) VALUES ($1, 9)`,
		"opacity high":          `INSERT INTO user_preferences (user_id, wallpaper_opacity) VALUES ($1, 81)`,
		"blur":                  `INSERT INTO user_preferences (user_id, wallpaper_blur) VALUES ($1, 21)`,
		"theme (0.01 contract)": `INSERT INTO user_preferences (user_id, theme) VALUES ($1, 'sepia')`,
	} {
		if _, err := pool.Exec(ctx, stmt, userA); err == nil {
			t.Errorf("%s: accepted", name)
			_, _ = pool.Exec(ctx, `DELETE FROM user_preferences WHERE user_id = $1`, userA)
		}
	}
	// A valid custom row is accepted.
	if _, err := pool.Exec(ctx, `INSERT INTO user_preferences (user_id, wallpaper_light_source, wallpaper_light_key) VALUES ($1, 'custom', $2)`, userA, key); err != nil {
		t.Fatal(err)
	}
	// Deleting the user removes the preferences.
	if _, err := pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userA); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM user_preferences WHERE user_id = $1`, userA).Scan(&n); err != nil || n != 0 {
		t.Fatal(n, err)
	}
}
