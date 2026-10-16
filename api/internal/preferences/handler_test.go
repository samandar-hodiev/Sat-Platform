package preferences

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
	"github.com/samandar-hodiev/sat-platform/api/internal/middleware"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
)

type testAPI struct {
	t      *testing.T
	f      *fixture
	r      *gin.Engine
	tokens *auth.AccessTokens
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	gin.SetMode(gin.TestMode)
	httpx.ConfigureValidator()
	f := newFixture(t)
	tokens, err := auth.NewAccessTokens(strings.Repeat("k", 32), 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	r.Use(
		middleware.BodyLimitWithRoutes(1<<20, map[string]int64{http.MethodPost + " " + WallpaperRoute: WallpaperRequestMaxBytes}),
		middleware.Errors(slog.New(slog.NewTextHandler(io.Discard, nil))),
	)
	NewHandler(f.svc, ratelimit.NewMemory()).RegisterRoutes(r.Group("/api/v1"), auth.Authenticate(tokens))
	return &testAPI{t: t, f: f, r: r, tokens: tokens}
}

func (a *testAPI) token(userID string, role auth.Role) string {
	a.t.Helper()
	tok, _, err := a.tokens.Issue(userID, role, "33333333-0000-4000-8000-000000000000")
	if err != nil {
		a.t.Fatal(err)
	}
	return tok
}

type wallpaperJSON struct {
	Source    string  `json:"source"`
	GalleryID *string `json:"gallery_id"`
	CustomURL *string `json:"custom_url"`
}

type apiResponse struct {
	Data struct {
		Theme            string `json:"theme"`
		Accent           string `json:"accent"`
		Density          string `json:"density"`
		Motion           string `json:"motion"`
		SidebarCollapsed bool   `json:"sidebar_collapsed"`
		Wallpaper        struct {
			Light   wallpaperJSON `json:"light"`
			Dark    wallpaperJSON `json:"dark"`
			Opacity int           `json:"opacity"`
			Blur    int           `json:"blur"`
		} `json:"wallpaper"`
		Stored    bool    `json:"stored"`
		UpdatedAt *string `json:"updated_at"`
	} `json:"data"`
	Error struct {
		Code    string         `json:"code"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

func (a *testAPI) send(req *http.Request, token string) (*httptest.ResponseRecorder, apiResponse) {
	a.t.Helper()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	a.r.ServeHTTP(w, req)
	var res apiResponse
	if strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			a.t.Fatalf("invalid JSON %q", w.Body.String())
		}
	}
	return w, res
}

func (a *testAPI) do(method, path, body, token string) (*httptest.ResponseRecorder, apiResponse) {
	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, r)
	req.Header.Set("Content-Type", "application/json")
	return a.send(req, token)
}

func (a *testAPI) upload(mode, field string, data []byte, token string) (*httptest.ResponseRecorder, apiResponse) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("note", "ignored")
	fw, _ := mw.CreateFormFile(field, "../../evil.svg") // name and extension are ignored
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/wallpapers/"+mode, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return a.send(req, token)
}

func TestHTTPRequiresLearnerToken(t *testing.T) {
	a := newTestAPI(t)
	if w, _ := a.do(http.MethodGet, "/api/v1/me/preferences", "", ""); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
	owner := a.token(userA, auth.Role("owner"))
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/me/preferences"}, {http.MethodPatch, "/api/v1/me/preferences"},
		{http.MethodPost, "/api/v1/me/wallpapers/light"}, {http.MethodDelete, "/api/v1/me/wallpapers/dark"},
	} {
		if w, _ := a.do(c.method, c.path, "{}", owner); w.Code != http.StatusForbidden {
			t.Errorf("%s %s: %d", c.method, c.path, w.Code)
		}
	}
}

func TestHTTPGetAndPatch(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userA, auth.RoleLearner)

	w, res := a.do(http.MethodGet, "/api/v1/me/preferences", "", tok)
	d := res.Data
	if w.Code != http.StatusOK || d.Stored || d.UpdatedAt != nil || d.Theme != "light" || d.Wallpaper.Light.Source != "none" ||
		d.Wallpaper.Light.GalleryID != nil || d.Wallpaper.Light.CustomURL != nil || d.Wallpaper.Opacity != DefaultOpacity {
		t.Fatalf("%d %+v", w.Code, d)
	}

	w, res = a.do(http.MethodPatch, "/api/v1/me/preferences", `{"theme":"system","accent":"violet","sidebar_collapsed":true,
		"wallpaper":{"light":{"source":"gallery","gallery_id":"peaks"},"opacity":55,"blur":6}}`, tok)
	d = res.Data
	if w.Code != http.StatusOK || !d.Stored || d.UpdatedAt == nil || d.Theme != "system" || d.Accent != "violet" ||
		!d.SidebarCollapsed || d.Wallpaper.Light.Source != "gallery" || *d.Wallpaper.Light.GalleryID != "peaks" ||
		d.Wallpaper.Dark.Source != "none" || d.Wallpaper.Opacity != 55 || d.Wallpaper.Blur != 6 {
		t.Fatalf("%d %+v %+v", w.Code, d, res.Error)
	}

	// Only the fields sent change.
	_, res = a.do(http.MethodPatch, "/api/v1/me/preferences", `{"density":"compact"}`, tok)
	if res.Data.Accent != "violet" || res.Data.Density != "compact" || res.Data.Wallpaper.Opacity != 55 {
		t.Fatalf("%+v", res.Data)
	}
}

func TestHTTPPatchErrors(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userA, auth.RoleLearner)

	w, res := a.do(http.MethodPatch, "/api/v1/me/preferences", `{"theme":"neon","wallpaper":{"dark":{"source":"gallery","gallery_id":"nope"},"blur":-1}}`, tok)
	fields, _ := res.Error.Details["fields"].(map[string]any)
	if w.Code != http.StatusUnprocessableEntity || fields["theme"] == nil || fields["wallpaper.dark.gallery_id"] == nil || fields["wallpaper.blur"] == nil {
		t.Fatalf("%d %+v", w.Code, res.Error)
	}
	if w, _ := a.do(http.MethodPatch, "/api/v1/me/preferences", `{"theme":1}`, tok); w.Code != http.StatusBadRequest {
		t.Fatal("wrong type:", w.Code)
	}
	if w, _ := a.do(http.MethodPatch, "/api/v1/me/preferences", `not json`, tok); w.Code != http.StatusBadRequest {
		t.Fatal("bad json:", w.Code)
	}
	w, res = a.do(http.MethodPatch, "/api/v1/me/preferences", `{"wallpaper":{"light":{"source":"custom"}}}`, tok)
	if w.Code != http.StatusConflict || res.Error.Details["reason"] != "wallpaper_not_uploaded" {
		t.Fatalf("%d %+v", w.Code, res.Error)
	}
}

func TestHTTPWallpaperUploadServeRemove(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userA, auth.RoleLearner)

	w, res := a.upload("dark", "file", pngBytes(t, 320, 180), tok)
	dark := res.Data.Wallpaper.Dark
	if w.Code != http.StatusOK || dark.Source != "custom" || dark.CustomURL == nil || res.Data.Wallpaper.Light.Source != "none" {
		t.Fatalf("%d %+v %+v", w.Code, res.Data, res.Error)
	}
	url := *dark.CustomURL
	if !strings.HasPrefix(url, "/api/v1/media/wallpapers/"+userA+"/") || !strings.HasSuffix(url, ".jpg") {
		t.Fatal(url)
	}

	// Public, no token; cached forever because every upload has a new URL.
	img := httptest.NewRecorder()
	a.r.ServeHTTP(img, httptest.NewRequest(http.MethodGet, url, nil))
	if img.Code != http.StatusOK || img.Header().Get("Content-Type") != "image/jpeg" ||
		!strings.Contains(img.Header().Get("Cache-Control"), "immutable") || !bytes.HasPrefix(img.Body.Bytes(), []byte{0xFF, 0xD8}) {
		t.Fatalf("%d %v", img.Code, img.Header())
	}

	w, res = a.do(http.MethodDelete, "/api/v1/me/wallpapers/dark", "", tok)
	if w.Code != http.StatusOK || res.Data.Wallpaper.Dark.Source != "none" || res.Data.Wallpaper.Dark.CustomURL != nil {
		t.Fatal(w.Code, res.Data)
	}
	gone := httptest.NewRecorder()
	a.r.ServeHTTP(gone, httptest.NewRequest(http.MethodGet, url, nil))
	if gone.Code != http.StatusNotFound {
		t.Fatal(gone.Code)
	}
}

func TestHTTPWallpaperUploadErrors(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userA, auth.RoleLearner)

	if w, _ := a.upload("system", "file", pngBytes(t, 10, 10), tok); w.Code != http.StatusNotFound {
		t.Fatal("unknown mode:", w.Code)
	}
	if w, _ := a.do(http.MethodDelete, "/api/v1/me/wallpapers/sepia", "", tok); w.Code != http.StatusNotFound {
		t.Fatal("unknown mode delete:", w.Code)
	}
	if w, res := a.upload("light", "file", []byte("GIF89a..."), tok); w.Code != http.StatusUnprocessableEntity ||
		res.Error.Details["fields"].(map[string]any)["file"] == nil {
		t.Fatal(w.Code, res.Error)
	}
	if w, _ := a.upload("light", "image", pngBytes(t, 10, 10), tok); w.Code != http.StatusUnprocessableEntity {
		t.Fatal("missing file field:", w.Code)
	}
	if w, _ := a.upload("light", "file", bytes.Repeat([]byte{0xFF}, WallpaperMaxBytes+1), tok); w.Code != http.StatusUnprocessableEntity {
		t.Fatal("over 8 MB:", w.Code)
	}
	if w, _ := a.upload("light", "file", bytes.Repeat([]byte{0xFF}, WallpaperRequestMaxBytes), tok); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatal("over request limit:", w.Code)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/me/wallpapers/light", strings.NewReader(`{"file":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	if w, _ := a.send(req, tok); w.Code != http.StatusBadRequest {
		t.Fatal("not multipart:", w.Code)
	}
}

func TestHTTPWallpaperUploadRateLimitedPerLearner(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userA, auth.RoleLearner)
	img := pngBytes(t, 8, 8)
	for i := 0; i < wallpaperUploadLimit; i++ {
		if w, _ := a.upload("light", "file", img, tok); w.Code != http.StatusOK {
			t.Fatalf("upload %d: %d", i, w.Code)
		}
	}
	if w, _ := a.upload("dark", "file", img, tok); w.Code != http.StatusTooManyRequests {
		t.Fatal(w.Code)
	}
	if w, _ := a.upload("light", "file", img, a.token(userB, auth.RoleLearner)); w.Code != http.StatusOK {
		t.Fatal("another learner has their own budget:", w.Code)
	}
}

func TestHTTPServeWallpaperRejectsOtherPaths(t *testing.T) {
	a := newTestAPI(t)
	for _, p := range []string{
		"/api/v1/media/wallpapers/not-a-uuid/00000000000000000000000000000000.jpg",
		"/api/v1/media/wallpapers/" + userA + "/..%2F..%2Fetc%2Fpasswd",
		"/api/v1/media/wallpapers/" + userA + "/00000000000000000000000000000000.jpg.meta",
		"/api/v1/media/wallpapers/" + userA + "/00000000000000000000000000000000.jpg",
	} {
		w := httptest.NewRecorder()
		a.r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
}
