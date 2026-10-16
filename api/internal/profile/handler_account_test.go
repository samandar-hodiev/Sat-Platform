package profile

import (
	"net/http"
	"testing"

	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
)

func TestHTTPMyProfileRoutesNeedLearner(t *testing.T) {
	a := newTestAPI(t)
	for _, r := range []struct{ method, path string }{
		{http.MethodPatch, "/api/v1/me/profile"},
		{http.MethodDelete, "/api/v1/me/account"},
	} {
		if w, _ := a.json(r.method, r.path, map[string]string{}, ""); w.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: %d", r.method, r.path, w.Code)
		}
		if w, _ := a.json(r.method, r.path, map[string]string{}, a.token(userPwd, auth.Role("owner"))); w.Code != http.StatusForbidden {
			t.Errorf("%s %s as owner: %d", r.method, r.path, w.Code)
		}
	}
}

func TestHTTPUpdateProfile(t *testing.T) {
	a := newTestAPI(t)
	tok := a.token(userPwd, auth.RoleLearner)

	w, res := a.json(http.MethodPatch, "/api/v1/me/profile", map[string]string{"first_name": "Ali"}, tok)
	if w.Code != http.StatusConflict || res.Error.Details["reason"] != "profile_setup_required" {
		t.Fatalf("before setup: %d %+v", w.Code, res.Error)
	}
	a.f.completed(t, userPwd)

	// null is "not sent"; phone "" removes the phone.
	w, res = a.json(http.MethodPatch, "/api/v1/me/profile",
		map[string]any{"first_name": "Aziza", "last_name": nil, "phone": "", "date_of_birth": "2007-01-02"}, tok)
	if w.Code != http.StatusOK || *res.Data.FirstName != "Aziza" || res.Data.Phone != nil ||
		*res.Data.DateOfBirth != "2007-01-02" || !res.Data.ProfileCompleted {
		t.Fatalf("%d %+v %+v", w.Code, res.Data, res.Error)
	}
	got, _ := a.f.svc.Get(ctx, userPwd)
	if got.LastName != "Valiyev" {
		t.Fatalf("last name must stay: %+v", got)
	}

	w, res = a.json(http.MethodPatch, "/api/v1/me/profile", map[string]any{"first_name": "", "timezone": "Mars/Base"}, tok)
	fields, _ := res.Error.Details["fields"].(map[string]any)
	if w.Code != http.StatusUnprocessableEntity || fields["first_name"] == nil || fields["timezone"] == nil {
		t.Fatalf("%d %+v", w.Code, res.Error)
	}
	if w, _ := a.json(http.MethodPatch, "/api/v1/me/profile", map[string]any{"first_name": 5}, tok); w.Code != http.StatusBadRequest {
		t.Fatalf("wrong JSON type: %d", w.Code)
	}
}

func TestHTTPDeleteAccount(t *testing.T) {
	a := newTestAPI(t)
	a.f.accounts.password[userPwd] = "correct-password"
	a.f.completed(t, userPwd)
	tok := a.token(userPwd, auth.RoleLearner)

	w, res := a.json(http.MethodDelete, "/api/v1/me/account", map[string]string{"current_password": "correct-password"}, tok)
	if fields, _ := res.Error.Details["fields"].(map[string]any); w.Code != http.StatusUnprocessableEntity || fields["confirmation"] == nil {
		t.Fatalf("%d %+v", w.Code, res.Error)
	}
	w, res = a.json(http.MethodDelete, "/api/v1/me/account",
		map[string]string{"confirmation": "DELETE", "current_password": "wrong"}, tok)
	if fields, _ := res.Error.Details["fields"].(map[string]any); w.Code != http.StatusUnprocessableEntity || fields["current_password"] == nil {
		t.Fatalf("%d %+v", w.Code, res.Error)
	}

	w, _ = a.json(http.MethodDelete, "/api/v1/me/account",
		map[string]string{"confirmation": "DELETE", "current_password": "correct-password"}, tok)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: %d %s", w.Code, w.Body.String())
	}
	// The (still unexpired) access token no longer reaches the profile.
	if w, _ := a.json(http.MethodGet, "/api/v1/me/profile", nil, tok); w.Code != http.StatusUnauthorized {
		t.Fatalf("after delete: %d", w.Code)
	}
}
