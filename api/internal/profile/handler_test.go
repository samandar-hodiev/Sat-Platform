package profile

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
		middleware.BodyLimitWithRoutes(1<<20, map[string]int64{http.MethodPost + " " + AvatarRoute: AvatarRequestMaxBytes}),
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

type apiResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Email              string  `json:"email"`
		FirstName          *string `json:"first_name"`
		Phone              *string `json:"phone"`
		DateOfBirth        *string `json:"date_of_birth"`
		AvatarURL          *string `json:"avatar_url"`
		HasPassword        bool    `json:"has_password"`
		PasswordRequired   bool    `json:"password_required"`
		ProfileCompleted   bool    `json:"profile_completed"`
		ProfileCompletedAt *string `json:"profile_completed_at"`
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

func (a *testAPI) json(method, path string, body any, token string) (*httptest.ResponseRecorder, apiResponse) {
	var buf io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	return a.send(req, token)
}

func (a *testAPI) upload(field string, data []byte, token string) (*httptest.ResponseRecorder, apiResponse) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("note", "ignored")
	fw, _ := mw.CreateFormFile(field, "photo.png") // name and extension are ignored
	_, _ = fw.Write(data)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, AvatarRoute, &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return a.send(req, token)
}

func TestHTTPProfileRequiresLearnerToken(t *testing.T) {
	a := newTestAPI(t)
	if w, _ := a.json(http.MethodGet, "/api/v1/me/profile", nil, ""); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
	w, res := a.json(http.MethodGet, "/api/v1/me/profile", nil, a.token(userEmail, auth.Role("owner")))
	if w.Code != http.StatusForbidden || res.Error.Code != "FORBIDDEN" {
		t.Fatal(w.Code, res.Error)
	}
	if w, _ := a.upload("file", []byte("x"), ""); w.Code != http.StatusUnauthorized {
		t.Fatal(w.Code)
	}
}

func TestHTTPSetupFlow(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userEmail, auth.RoleLearner)

	w, res := a.json(http.MethodGet, "/api/v1/me/profile", nil, tok)
	if w.Code != http.StatusOK || res.Data.Email != "email@example.com" || res.Data.ProfileCompleted ||
		!res.Data.PasswordRequired || res.Data.FirstName != nil || res.Data.AvatarURL != nil {
		t.Fatalf("%d %+v", w.Code, res.Data)
	}

	body := map[string]string{
		"first_name": "Ali", "last_name": "Valiyev", "phone": "+998 90 123 45 67",
		"date_of_birth": "2008-05-01", "timezone": "Asia/Tashkent", "password": "correct horse battery",
	}
	w, res = a.json(http.MethodPut, "/api/v1/me/profile/setup", body, tok)
	if w.Code != http.StatusOK || !res.Data.ProfileCompleted || *res.Data.Phone != "+998901234567" ||
		*res.Data.DateOfBirth != "2008-05-01" || !res.Data.HasPassword || res.Data.ProfileCompletedAt == nil {
		t.Fatalf("%d %+v %+v", w.Code, res.Data, res.Error)
	}

	w, res = a.json(http.MethodPut, "/api/v1/me/profile/setup", body, tok)
	if w.Code != http.StatusConflict || res.Error.Details["reason"] != "profile_already_completed" {
		t.Fatal(w.Code, res.Error)
	}
}

func TestHTTPSetupValidation(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userEmail, auth.RoleLearner)
	w, res := a.json(http.MethodPut, "/api/v1/me/profile/setup", map[string]string{"first_name": "Ali"}, tok)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatal(w.Code)
	}
	fields, _ := res.Error.Details["fields"].(map[string]any)
	for _, f := range []string{"last_name", "date_of_birth", "password"} {
		if fields[f] == nil {
			t.Errorf("missing field error %q: %v", f, fields)
		}
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/me/profile/setup", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	if w, _ := a.send(req, tok); w.Code != http.StatusBadRequest {
		t.Fatal(w.Code)
	}
}

func TestHTTPAvatarUploadServeRemove(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userEmail, auth.RoleLearner)

	w, res := a.upload("file", encodePNG(t, halves(200, 100)), tok)
	if w.Code != http.StatusOK || res.Data.AvatarURL == nil {
		t.Fatalf("%d %+v", w.Code, res.Error)
	}
	url := *res.Data.AvatarURL
	if !strings.HasPrefix(url, "/api/v1/media/avatars/"+userEmail+"/") || !strings.HasSuffix(url, ".jpg") {
		t.Fatal(url)
	}

	// Public, no token; cached forever because every upload has a new URL.
	img := httptest.NewRecorder()
	a.r.ServeHTTP(img, httptest.NewRequest(http.MethodGet, url, nil))
	if img.Code != http.StatusOK || img.Header().Get("Content-Type") != "image/jpeg" ||
		!strings.Contains(img.Header().Get("Cache-Control"), "immutable") || !bytes.HasPrefix(img.Body.Bytes(), []byte{0xFF, 0xD8}) {
		t.Fatalf("%d %v", img.Code, img.Header())
	}

	w, res = a.json(http.MethodDelete, AvatarRoute, nil, tok)
	if w.Code != http.StatusOK || res.Data.AvatarURL != nil {
		t.Fatal(w.Code, res.Data)
	}
	gone := httptest.NewRecorder()
	a.r.ServeHTTP(gone, httptest.NewRequest(http.MethodGet, url, nil))
	if gone.Code != http.StatusNotFound {
		t.Fatal(gone.Code)
	}
}

func TestHTTPAvatarUploadErrors(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userEmail, auth.RoleLearner)

	if w, res := a.upload("file", []byte("GIF89a..."), tok); w.Code != http.StatusUnprocessableEntity ||
		res.Error.Details["fields"].(map[string]any)["file"] == nil {
		t.Fatal(w.Code, res.Error)
	}
	if w, _ := a.upload("photo", encodePNG(t, halves(10, 10)), tok); w.Code != http.StatusUnprocessableEntity {
		t.Fatal("missing file field:", w.Code)
	}
	if w, _ := a.upload("file", bytes.Repeat([]byte{0xFF}, AvatarMaxBytes+1), tok); w.Code != http.StatusUnprocessableEntity {
		t.Fatal("over 5 MB:", w.Code)
	}
	if w, res := a.upload("file", bytes.Repeat([]byte{0xFF}, AvatarRequestMaxBytes), tok); w.Code != http.StatusRequestEntityTooLarge {
		t.Fatal("over request limit:", w.Code, res.Error)
	}
	req := httptest.NewRequest(http.MethodPost, AvatarRoute, strings.NewReader(`{"file":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	if w, _ := a.send(req, tok); w.Code != http.StatusBadRequest {
		t.Fatal("not multipart:", w.Code)
	}
}

func TestHTTPAvatarUploadRateLimitedPerLearner(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userEmail, auth.RoleLearner)
	img := encodePNG(t, halves(8, 8))
	for i := 0; i < avatarUploadLimit; i++ {
		if w, _ := a.upload("file", img, tok); w.Code != http.StatusOK {
			t.Fatalf("upload %d: %d", i, w.Code)
		}
	}
	if w, _ := a.upload("file", img, tok); w.Code != http.StatusTooManyRequests {
		t.Fatal(w.Code)
	}
	// Another learner has their own budget.
	if w, _ := a.upload("file", img, a.token(userGoogle, auth.RoleLearner)); w.Code != http.StatusOK {
		t.Fatal(w.Code)
	}
}

func TestHTTPServeAvatarRejectsOtherPaths(t *testing.T) {
	a := newTestAPI(t)
	for _, p := range []string{
		"/api/v1/media/avatars/not-a-uuid/00000000000000000000000000000000.jpg",
		"/api/v1/media/avatars/" + userEmail + "/..%2F..%2Fetc%2Fpasswd",
		"/api/v1/media/avatars/" + userEmail + "/00000000000000000000000000000000.jpg.meta",
		"/api/v1/media/avatars/" + userEmail + "/00000000000000000000000000000000.jpg",
	} {
		w := httptest.NewRecorder()
		a.r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("%s: %d", p, w.Code)
		}
	}
}
