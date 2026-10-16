package profile

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
)

// My Profile (2.10): editing the profile and deleting the account. The same scenarios run on
// PostgreSQL (repository_integration_test.go → account scenarios).

// ---- memory store and fake auth additions ----

func (s *memStore) Update(_ context.Context, userID string, in Update, now time.Time) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[userID]
	if !ok || p.CompletedAt == nil {
		return Profile{}, ErrNotCompleted
	}
	if in.FirstName != nil {
		p.FirstName = *in.FirstName
	}
	if in.LastName != nil {
		p.LastName = *in.LastName
	}
	if in.Phone != nil {
		p.Phone = *in.Phone
	}
	if in.DateOfBirth != nil {
		dob := *in.DateOfBirth
		p.DateOfBirth = &dob
	}
	if in.Timezone != nil {
		p.Timezone = *in.Timezone
	}
	p.UpdatedAt = now
	s.profiles[userID] = p
	return p, nil
}

func (s *memStore) DeleteAccount(_ context.Context, userID string, also func(pgx.Tx) error) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.users[userID] {
		return "", ErrNotFound
	}
	if also != nil {
		if err := also(nil); err != nil {
			return "", err // rolled back: the profile stays
		}
	}
	p := s.profiles[userID]
	delete(s.profiles, userID)
	return p.AvatarKey, nil
}

var errAuthDown = errors.New("database is down")

// fakeReauthCode is the code the fake auth accepts for accounts without a password.
const fakeReauthCode = "123456"

func (f *fakeAccounts) Reauthenticate(_ context.Context, userID string, r auth.Reauth) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[userID]
	if !ok {
		return apperr.Unauthorized("Sign in to continue.")
	}
	if a.HasPassword {
		if want, set := f.password[userID]; r.Password == "" || (set && r.Password != want) {
			return apperr.New(apperr.CodeValidation, "Some fields are invalid").
				WithDetails(map[string]any{"fields": map[string]string{"current_password": "This password is incorrect."}})
		}
		return nil
	}
	if r.Code != fakeReauthCode {
		return apperr.New(apperr.CodeValidation, "Some fields are invalid").
			WithDetails(map[string]any{"fields": map[string]string{"code": "The code is incorrect."}})
	}
	return nil
}

// DeleteAccountTx soft-deletes the users row inside the profile transaction when there is one
// (PostgreSQL test), like the auth module does.
func (f *fakeAccounts) DeleteAccountTx(ctx context.Context, tx pgx.Tx, userID string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.failNext; err != nil {
		f.failNext = nil
		return err
	}
	if tx != nil {
		if _, err := tx.Exec(ctx, `UPDATE users SET deleted_at = now() WHERE id = $1`, userID); err != nil {
			return err
		}
	}
	delete(f.accounts, userID)
	return nil
}

func ptr[T any](v T) *T { return &v }

func (f *fixture) completed(t *testing.T, userID string) View {
	t.Helper()
	in := validSetup()
	if f.accounts.accounts[userID].HasPassword || f.accounts.accounts[userID].HasGoogle {
		in.Password = ""
	}
	v, err := f.svc.CompleteSetup(ctx, userID, in)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// ---- Update ----

func TestUpdateProfile(t *testing.T) {
	f := newFixture(t)
	f.completed(t, userPwd)

	v, err := f.svc.Update(ctx, userPwd, UpdateInput{
		FirstName: ptr("  Aziza "), LastName: ptr("Karimova"), Phone: ptr("00998 91 765 43 21"),
		DateOfBirth: ptr("2007-12-31"), Timezone: ptr("Europe/London"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if v.FirstName != "Aziza" || v.LastName != "Karimova" || v.Phone != "+998917654321" ||
		v.DateOfBirth.Format(time.DateOnly) != "2007-12-31" || v.Timezone != "Europe/London" || v.CompletedAt == nil {
		t.Fatalf("%+v", v)
	}

	// Only the fields sent change.
	v, err = f.svc.Update(ctx, userPwd, UpdateInput{LastName: ptr("Valiyeva")})
	if err != nil {
		t.Fatal(err)
	}
	if v.FirstName != "Aziza" || v.LastName != "Valiyeva" || v.Phone != "+998917654321" || v.Timezone != "Europe/London" {
		t.Fatalf("%+v", v)
	}
	got, _ := f.svc.Get(ctx, userPwd)
	if got.LastName != "Valiyeva" {
		t.Fatalf("not saved: %+v", got)
	}

	// An empty phone removes it.
	v, err = f.svc.Update(ctx, userPwd, UpdateInput{Phone: ptr("")})
	if err != nil || v.Phone != "" {
		t.Fatalf("%+v %v", v, err)
	}

	// Nothing sent: nothing changes.
	v, err = f.svc.Update(ctx, userPwd, UpdateInput{})
	if err != nil || v.LastName != "Valiyeva" {
		t.Fatalf("%+v %v", v, err)
	}
}

func TestUpdateProfileReportsEveryInvalidField(t *testing.T) {
	f := newFixture(t)
	f.completed(t, userPwd)
	_, err := f.svc.Update(ctx, userPwd, UpdateInput{
		FirstName: ptr(" "), LastName: ptr("x1"), Phone: ptr("12"), DateOfBirth: ptr("2020-01-01"), Timezone: ptr(""),
	})
	wantFields(t, err, "first_name", "last_name", "phone", "date_of_birth", "timezone")
	// Nothing was saved.
	v, _ := f.svc.Get(ctx, userPwd)
	if v.FirstName != "Ali" {
		t.Fatalf("%+v", v)
	}
}

func TestUpdateProfileNeedsFinishedSetup(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Update(ctx, userEmail, UpdateInput{FirstName: ptr("Ali")})
	ae := wantCode(t, err, apperr.CodeConflict)
	if ae.Details["reason"] != "profile_setup_required" {
		t.Fatal(ae.Details)
	}
	_, err = f.svc.Update(ctx, "44444444-4444-4444-8444-444444444444", UpdateInput{FirstName: ptr("Ali")})
	wantCode(t, err, apperr.CodeUnauthorized)
}

// ---- DeleteAccount ----

func TestDeleteAccountWithPassword(t *testing.T) {
	f := newFixture(t)
	f.accounts.password[userPwd] = "correct-password"
	f.completed(t, userPwd)
	av, err := f.svc.UploadAvatar(ctx, userPwd, encodeJPEG(t, halves(200, 200)))
	if err != nil {
		t.Fatal(err)
	}

	err = f.svc.DeleteAccount(ctx, userPwd, DeleteAccountInput{Confirmation: "delete", Reauth: auth.Reauth{Password: "correct-password"}})
	wantFields(t, err, "confirmation")
	err = f.svc.DeleteAccount(ctx, userPwd, DeleteAccountInput{Confirmation: "DELETE", Reauth: auth.Reauth{Password: "wrong"}})
	wantFields(t, err, "current_password")
	if _, err := f.svc.Get(ctx, userPwd); err != nil {
		t.Fatal("nothing may be deleted before the learner is confirmed")
	}

	if err := f.svc.DeleteAccount(ctx, userPwd, DeleteAccountInput{Confirmation: "DELETE", Reauth: auth.Reauth{Password: "correct-password"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Get(ctx, userPwd); !errors.Is(err, ErrNotFound) {
		t.Fatalf("profile row must be gone: %v", err)
	}
	if f.fileExists(av.AvatarKey) {
		t.Fatal("avatar file must be deleted")
	}
	_, err = f.svc.Get(ctx, userPwd)
	wantCode(t, err, apperr.CodeUnauthorized)
	err = f.svc.DeleteAccount(ctx, userPwd, DeleteAccountInput{Confirmation: "DELETE", Reauth: auth.Reauth{Password: "correct-password"}})
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestDeleteAccountWithCodeAndWithoutProfile(t *testing.T) {
	f := newFixture(t)
	// A learner who never finished Profile Setup can delete the account too.
	err := f.svc.DeleteAccount(ctx, userEmail, DeleteAccountInput{Confirmation: "DELETE", Reauth: auth.Reauth{Code: "000000"}})
	wantFields(t, err, "code")
	if err := f.svc.DeleteAccount(ctx, userEmail, DeleteAccountInput{Confirmation: "DELETE", Reauth: auth.Reauth{Code: fakeReauthCode}}); err != nil {
		t.Fatal(err)
	}
	_, err = f.svc.Get(ctx, userEmail)
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestDeleteAccountAuthFailureKeepsProfile(t *testing.T) {
	f := newFixture(t)
	f.completed(t, userGoogle)
	av, err := f.svc.UploadAvatar(ctx, userGoogle, encodeJPEG(t, halves(200, 200)))
	if err != nil {
		t.Fatal(err)
	}
	f.accounts.failNext = errAuthDown
	err = f.svc.DeleteAccount(ctx, userGoogle, DeleteAccountInput{Confirmation: "DELETE", Reauth: auth.Reauth{Code: fakeReauthCode}})
	if err == nil {
		t.Fatal("want the auth error")
	}
	v, err := f.svc.Get(ctx, userGoogle)
	if err != nil || v.FirstName != "Ali" || v.AvatarKey != av.AvatarKey || !f.fileExists(av.AvatarKey) {
		t.Fatalf("profile and avatar must stay: %+v %v", v, err)
	}
}
