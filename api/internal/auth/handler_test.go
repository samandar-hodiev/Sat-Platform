package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/samandar-hodiev/sat-platform/api/internal/config"
	"github.com/samandar-hodiev/sat-platform/api/internal/httpx"
	"github.com/samandar-hodiev/sat-platform/api/internal/middleware"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/ratelimit"
)

type apiResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	} `json:"error"`
}

type testAPI struct {
	t   *testing.T
	f   *fixture
	r   *gin.Engine
	lim *ratelimit.Memory
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	gin.SetMode(gin.TestMode)
	httpx.ConfigureValidator()
	f := newFixture(t)
	lim := ratelimit.NewMemory()
	r := gin.New()
	r.Use(middleware.Errors(slog.New(slog.NewTextHandler(io.Discard, nil))))
	NewHandler(f.svc, f.svc.tokens, lim).RegisterRoutes(r.Group("/api/v1"))
	return &testAPI{t: t, f: f, r: r, lim: lim}
}

func (a *testAPI) do(method, path string, body any, header map[string]string) (int, apiResponse) {
	a.t.Helper()
	var buf io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-Platform", "web")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	a.r.ServeHTTP(w, req)
	var res apiResponse
	if w.Code != http.StatusNoContent {
		if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
			a.t.Fatalf("%s %s: invalid JSON %q", method, path, w.Body.String())
		}
	}
	return w.Code, res
}

func TestHTTPSignUpSignInRefreshMeLogout(t *testing.T) {
	a := newTestAPI(t)

	code, res := a.do("POST", "/api/v1/auth/sign-up/start", map[string]string{"email": "Learner@Example.com"}, nil)
	if code != 200 {
		t.Fatalf("start: %d %+v", code, res.Error)
	}
	var sent map[string]string
	_ = json.Unmarshal(res.Data, &sent)
	if sent["expires_at"] == "" || sent["resend_available_at"] == "" {
		t.Fatalf("sent = %s", res.Data)
	}

	code, res = a.do("POST", "/api/v1/auth/sign-up/verify",
		map[string]string{"email": "learner@example.com", "code": a.f.mail.code(t)}, nil)
	if code != 200 {
		t.Fatalf("verify: %d %+v", code, res.Error)
	}
	var session struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		Remember     bool   `json:"remember"`
		IsNewUser    bool   `json:"is_new_user"`
		User         struct {
			ID            string `json:"id"`
			Email         string `json:"email"`
			Role          string `json:"role"`
			EmailVerified bool   `json:"email_verified"`
			HasPassword   bool   `json:"has_password"`
		} `json:"user"`
	}
	_ = json.Unmarshal(res.Data, &session)
	if session.TokenType != "Bearer" || !session.IsNewUser || session.User.Email != "learner@example.com" ||
		session.User.Role != "learner" || !session.User.EmailVerified || session.User.HasPassword || !session.Remember {
		t.Fatalf("session = %s", res.Data)
	}

	code, res = a.do("GET", "/api/v1/auth/me", nil, map[string]string{"Authorization": "Bearer " + session.AccessToken})
	if code != 200 || !strings.Contains(string(res.Data), `"email":"learner@example.com"`) ||
		!strings.Contains(string(res.Data), `"has_google":false`) {
		t.Fatalf("me: %d %s", code, res.Data)
	}

	code, res = a.do("POST", "/api/v1/auth/refresh", map[string]string{"refresh_token": session.RefreshToken}, nil)
	if code != 200 {
		t.Fatalf("refresh: %d %+v", code, res.Error)
	}
	var rotated struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.Unmarshal(res.Data, &rotated)

	if code, _ := a.do("POST", "/api/v1/auth/logout", map[string]string{"refresh_token": rotated.RefreshToken}, nil); code != 204 {
		t.Fatalf("logout: %d", code)
	}
	if code, _ := a.do("POST", "/api/v1/auth/refresh", map[string]string{"refresh_token": rotated.RefreshToken}, nil); code != 401 {
		t.Fatalf("refresh after logout: %d", code)
	}
}

func TestHTTPMeRequiresToken(t *testing.T) {
	a := newTestAPI(t)
	code, res := a.do("GET", "/api/v1/auth/me", nil, nil)
	if code != 401 || res.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("no token: %d %+v", code, res.Error)
	}
	code, _ = a.do("GET", "/api/v1/auth/me", nil, map[string]string{"Authorization": "Bearer garbage"})
	if code != 401 {
		t.Fatalf("garbage token: %d", code)
	}

	tok, _, _ := a.f.svc.tokens.Issue("user-x", RoleLearner, "session-x")
	a.f.advance(16 * time.Minute)
	code, res = a.do("GET", "/api/v1/auth/me", nil, map[string]string{"Authorization": "Bearer " + tok})
	if code != 401 || res.Error.Details["reason"] != "token_expired" {
		t.Fatalf("expired: %d %+v", code, res.Error)
	}
}

func TestHTTPValidation(t *testing.T) {
	a := newTestAPI(t)
	code, res := a.do("POST", "/api/v1/auth/sign-up/verify", map[string]string{"email": "a@example.com", "code": "12ab"}, nil)
	if code != 422 {
		t.Fatalf("code: %d", code)
	}
	fields, _ := res.Error.Details["fields"].(map[string]any)
	if fields["code"] != "Enter the 6-digit code." {
		t.Fatalf("fields = %v", fields)
	}

	code, _ = a.do("POST", "/api/v1/auth/sign-in", map[string]string{"email": "a@example.com"}, nil)
	if code != 422 {
		t.Fatalf("missing password: %d", code)
	}
	code, res = a.do("POST", "/api/v1/auth/sign-in", map[string]any{"email": "a@example.com", "password": "x"}, nil)
	if code != 401 || res.Error.Message != "Email or password is incorrect." {
		t.Fatalf("unknown user: %d %+v", code, res.Error)
	}
	if code, _ := a.do("POST", "/api/v1/auth/password/forgot", map[string]string{"email": "nobody@example.com"}, nil); code != 204 {
		t.Fatalf("forgot: %d", code)
	}
	code, res = a.do("POST", "/api/v1/auth/password/reset", map[string]string{"token": "nope", "password": "long-enough"}, nil)
	if code != 400 || res.Error.Message != "This reset link is invalid or has expired." {
		t.Fatalf("reset: %d %+v", code, res.Error)
	}
	if code, _ := a.do("POST", "/api/v1/auth/logout", nil, nil); code != 204 {
		t.Fatalf("logout without body: %d", code)
	}
	code, res = a.do("POST", "/api/v1/auth/google", map[string]string{"id_token": "x"}, nil)
	if code != 503 {
		t.Fatalf("google disabled: %d %+v", code, res.Error)
	}
}

func TestHTTPSignInRouteIsRateLimitedPerIP(t *testing.T) {
	a := newTestAPI(t)
	var last int
	for i := range 11 {
		last, _ = a.do("POST", "/api/v1/auth/sign-in",
			map[string]any{"email": "user" + string(rune('a'+i)) + "@example.com", "password": "x"}, nil)
	}
	if last != 429 {
		t.Fatalf("11th sign-in from one IP: %d, want 429", last)
	}
}

func TestNewModule(t *testing.T) {
	base := func() *config.Config {
		cfg, err := config.FromLookup(func(k string) (string, bool) {
			if k == "DATABASE_URL" {
				return "postgres://localhost/x", true
			}
			return "", false
		})
		if err != nil {
			t.Fatal(err)
		}
		return cfg
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	m, err := NewModule(Deps{Config: base(), Log: log, Limiter: ratelimit.NewMemory(), Store: newMemStore()})
	if err != nil {
		t.Fatalf("development without JWT_SECRET should work: %v", err)
	}
	if m.Service.google != nil || m.Service.mail == nil {
		t.Fatal("google must be off without client IDs; log mail on in development")
	}

	cfg := base()
	cfg.Env = config.EnvProduction
	if _, err := NewModule(Deps{Config: cfg, Log: log, Limiter: ratelimit.NewMemory(), Store: newMemStore()}); err == nil {
		t.Fatal("production without JWT_SECRET must fail")
	}

	cfg = base()
	cfg.Auth.JWTSecret = testSecret
	cfg.Auth.MailDriver = "disabled"
	cfg.Auth.GoogleClientIDs = []string{"web-client"}
	m, err = NewModule(Deps{Config: cfg, Log: log, Limiter: ratelimit.NewMemory(), Store: newMemStore()})
	if err != nil || m.Service.mail != nil || m.Service.google == nil {
		t.Fatalf("module = %+v %v", m, err)
	}
}
