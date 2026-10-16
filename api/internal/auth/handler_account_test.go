package auth

import (
	"encoding/json"
	"testing"
	"time"
)

func (a *testAPI) bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func TestHTTPAccountRoutesNeedLearner(t *testing.T) {
	a := newTestAPI(t)
	routes := []struct{ method, path string }{
		{"GET", "/api/v1/me/sign-in-methods"},
		{"POST", "/api/v1/me/reauth/code"},
		{"POST", "/api/v1/me/email/change"},
		{"POST", "/api/v1/me/email/change/confirm"},
		{"PUT", "/api/v1/me/password"},
		{"POST", "/api/v1/me/sign-in-methods/google"},
		{"DELETE", "/api/v1/me/sign-in-methods/google"},
	}
	owner, _, _ := a.f.svc.tokens.Issue("00000000-0000-0000-0000-000000000001", Role("owner"), "s")
	for _, r := range routes {
		if code, _ := a.do(r.method, r.path, map[string]string{}, nil); code != 401 {
			t.Errorf("%s %s without token: %d", r.method, r.path, code)
		}
		if code, _ := a.do(r.method, r.path, map[string]string{}, a.bearer(owner)); code != 403 {
			t.Errorf("%s %s as owner: %d", r.method, r.path, code)
		}
	}
}

func TestHTTPAccountFlow(t *testing.T) {
	a := newTestAPI(t)
	res := a.f.signUp(t, "learner@example.com")
	auth := a.bearer(res.AccessToken)

	code, body := a.do("GET", "/api/v1/me/sign-in-methods", nil, auth)
	if code != 200 {
		t.Fatalf("methods: %d %+v", code, body.Error)
	}
	var m map[string]any
	_ = json.Unmarshal(body.Data, &m)
	if m["email"] != "learner@example.com" || m["has_password"] != false || m["email_code"] != true ||
		m["reauth_method"] != "email_code" || m["google"] != nil || m["email_verified"] != true {
		t.Fatalf("methods = %s", body.Data)
	}

	// Set the first password with a code.
	a.f.advance(2 * time.Minute)
	if code, body = a.do("POST", "/api/v1/me/reauth/code", nil, auth); code != 200 {
		t.Fatalf("reauth code: %d %+v", code, body.Error)
	}
	code, body = a.do("PUT", "/api/v1/me/password", map[string]string{"new_password": "first-password", "code": "12"}, auth)
	if code != 422 || body.Error.Details["fields"].(map[string]any)["code"] != "Enter the 6-digit code." {
		t.Fatalf("bad code shape: %d %+v", code, body.Error)
	}
	code, _ = a.do("PUT", "/api/v1/me/password", map[string]string{"new_password": "first-password", "code": a.f.mail.code(t)}, auth)
	if code != 204 {
		t.Fatalf("password: %d", code)
	}

	// Change the email, confirmed with that password.
	code, body = a.do("POST", "/api/v1/me/email/change",
		map[string]string{"new_email": "New@Example.com", "current_password": "first-password"}, auth)
	if code != 200 {
		t.Fatalf("email change: %d %+v", code, body.Error)
	}
	var started map[string]string
	_ = json.Unmarshal(body.Data, &started)
	if started["new_email"] != "new@example.com" || started["expires_at"] == "" || started["resend_available_at"] == "" {
		t.Fatalf("started = %s", body.Data)
	}
	code, body = a.do("POST", "/api/v1/me/email/change/confirm", map[string]string{"code": a.f.mail.code(t)}, auth)
	if code != 200 {
		t.Fatalf("confirm: %d %+v", code, body.Error)
	}
	_ = json.Unmarshal(body.Data, &m)
	if m["email"] != "new@example.com" || m["has_password"] != true || m["reauth_method"] != "password" {
		t.Fatalf("after change = %s", body.Data)
	}

	// The access token still works (the current session is kept).
	if code, _ = a.do("GET", "/api/v1/auth/me", nil, auth); code != 200 {
		t.Fatalf("me after change: %d", code)
	}

	// Unlinking Google when none is linked.
	if code, _ = a.do("DELETE", "/api/v1/me/sign-in-methods/google", nil, auth); code != 404 {
		t.Fatalf("unlink: %d", code)
	}
	// Google disabled in this test API.
	code, _ = a.do("POST", "/api/v1/me/sign-in-methods/google",
		map[string]string{"id_token": "x", "current_password": "first-password"}, auth)
	if code != 503 {
		t.Fatalf("link without Google: %d", code)
	}
}

func TestHTTPReauthCodeRouteLimit(t *testing.T) {
	a := newTestAPI(t)
	res := a.f.signUp(t, "learner@example.com")
	auth := a.bearer(res.AccessToken)
	for i := range 5 {
		a.f.advance(2 * time.Minute)
		if code, body := a.do("POST", "/api/v1/me/reauth/code", nil, auth); code != 200 {
			t.Fatalf("send %d: %d %+v", i, code, body.Error)
		}
	}
	if code, _ := a.do("POST", "/api/v1/me/reauth/code", nil, auth); code != 429 {
		t.Fatalf("6th send in 10 minutes: %d", code)
	}
}
