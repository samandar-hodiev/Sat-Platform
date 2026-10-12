package profile

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
)

var ctx = context.Background()

func validSetup() SetupInput {
	return SetupInput{
		FirstName: "  Ali ", LastName: "Valiyev", Phone: "+998 90 123 45 67",
		DateOfBirth: "2008-05-01", Timezone: "Asia/Tashkent", Password: "correct horse battery",
	}
}

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

func TestGetFreshProfile(t *testing.T) {
	f := newFixture(t)
	v, err := f.svc.Get(ctx, userEmail)
	if err != nil {
		t.Fatal(err)
	}
	if v.Email != "email@example.com" || v.CompletedAt != nil || !v.PasswordRequired || v.HasPassword || v.Timezone != "UTC" {
		t.Fatalf("%+v", v)
	}
	g, _ := f.svc.Get(ctx, userGoogle)
	if g.PasswordRequired {
		t.Fatal("Google account must not require a password")
	}
	p, _ := f.svc.Get(ctx, userPwd)
	if p.PasswordRequired || !p.HasPassword {
		t.Fatalf("%+v", p)
	}
}

func TestUnknownAccountIsUnauthorized(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.Get(ctx, "44444444-4444-4444-8444-444444444444")
	wantCode(t, err, apperr.CodeUnauthorized)
}

func TestCompleteSetupEmailAccount(t *testing.T) {
	f := newFixture(t)
	v, err := f.svc.CompleteSetup(ctx, userEmail, validSetup())
	if err != nil {
		t.Fatal(err)
	}
	if v.FirstName != "Ali" || v.Phone != "+998901234567" || v.Timezone != "Asia/Tashkent" ||
		v.DateOfBirth.Format("2006-01-02") != "2008-05-01" || v.CompletedAt == nil || !v.HasPassword || v.PasswordRequired {
		t.Fatalf("%+v", v)
	}
	if f.accounts.password[userEmail] != "correct horse battery" {
		t.Fatal("password not set")
	}
	// A second submit (double click, retry) does not overwrite anything.
	_, err = f.svc.CompleteSetup(ctx, userEmail, validSetup())
	ae := wantCode(t, err, apperr.CodeConflict)
	if ae.Details["reason"] != "profile_already_completed" {
		t.Fatal(ae.Details)
	}
}

func TestCompleteSetupReportsEveryInvalidField(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.CompleteSetup(ctx, userEmail, SetupInput{
		FirstName: "", LastName: "x1", Phone: "12", DateOfBirth: "2020-01-01", Timezone: "Nowhere/City",
	})
	wantFields(t, err, "first_name", "last_name", "phone", "date_of_birth", "timezone", "password")
}

func TestCompleteSetupUnder13(t *testing.T) {
	f := newFixture(t)
	in := validSetup()
	in.DateOfBirth = "2013-10-13" // 13th birthday is tomorrow
	_, err := f.svc.CompleteSetup(ctx, userEmail, in)
	wantFields(t, err, "date_of_birth")
}

func TestCompleteSetupPhoneOptional(t *testing.T) {
	f := newFixture(t)
	in := validSetup()
	in.Phone = ""
	v, err := f.svc.CompleteSetup(ctx, userEmail, in)
	if err != nil || v.Phone != "" {
		t.Fatal(v, err)
	}
}

func TestCompleteSetupPasswordPolicyRollsBack(t *testing.T) {
	f := newFixture(t)
	in := validSetup()
	in.Password = "short"
	_, err := f.svc.CompleteSetup(ctx, userEmail, in)
	wantFields(t, err, "password")
	v, _ := f.svc.Get(ctx, userEmail)
	if v.CompletedAt != nil || v.FirstName != "" {
		t.Fatalf("profile written although the password failed: %+v", v)
	}
}

func TestCompleteSetupPasswordStepErrorRollsBack(t *testing.T) {
	f := newFixture(t)
	f.accounts.failNext = errors.New("database is down")
	if _, err := f.svc.CompleteSetup(ctx, userEmail, validSetup()); err == nil {
		t.Fatal("expected error")
	}
	if v, _ := f.svc.Get(ctx, userEmail); v.CompletedAt != nil {
		t.Fatal("profile completed although the password step failed")
	}
	// Retrying works.
	if _, err := f.svc.CompleteSetup(ctx, userEmail, validSetup()); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteSetupGooglePasswordOptional(t *testing.T) {
	f := newFixture(t)
	in := validSetup()
	in.Password = ""
	v, err := f.svc.CompleteSetup(ctx, userGoogle, in)
	if err != nil || v.HasPassword || v.CompletedAt == nil {
		t.Fatal(v, err)
	}

	f = newFixture(t)
	v, err = f.svc.CompleteSetup(ctx, userGoogle, validSetup())
	if err != nil || !v.HasPassword {
		t.Fatal(v, err)
	}
}

func TestCompleteSetupRejectsPasswordWhenOneExists(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.CompleteSetup(ctx, userPwd, validSetup())
	wantFields(t, err, "password")
	in := validSetup()
	in.Password = ""
	if _, err := f.svc.CompleteSetup(ctx, userPwd, in); err != nil {
		t.Fatal(err)
	}
}

func TestPrefillFromGoogle(t *testing.T) {
	f := newFixture(t)
	if err := (GooglePrefiller{f.store}).PrefillFromGoogle(ctx, userGoogle, auth.GoogleProfile{FirstName: " Ali ", LastName: "Valiyev"}); err != nil {
		t.Fatal(err)
	}
	v, _ := f.svc.Get(ctx, userGoogle)
	if v.FirstName != "Ali" || v.LastName != "Valiyev" || v.CompletedAt != nil {
		t.Fatalf("%+v", v)
	}
	// Existing values are kept; names that break the rules are skipped.
	if err := (GooglePrefiller{f.store}).PrefillFromGoogle(ctx, userGoogle, auth.GoogleProfile{FirstName: "Other", LastName: "Name"}); err != nil {
		t.Fatal(err)
	}
	if err := (GooglePrefiller{f.store}).PrefillFromGoogle(ctx, userEmail, auth.GoogleProfile{FirstName: "Ali 😀", LastName: "Karimov"}); err != nil {
		t.Fatal(err)
	}
	v, _ = f.svc.Get(ctx, userGoogle)
	e, _ := f.svc.Get(ctx, userEmail)
	if v.FirstName != "Ali" || e.FirstName != "" || e.LastName != "Karimov" {
		t.Fatalf("%+v %+v", v, e)
	}
}

func TestPrefillDoesNotTouchCompletedProfile(t *testing.T) {
	f := newFixture(t)
	in := validSetup()
	in.Password = ""
	if _, err := f.svc.CompleteSetup(ctx, userGoogle, in); err != nil {
		t.Fatal(err)
	}
	if err := (GooglePrefiller{f.store}).PrefillFromGoogle(ctx, userGoogle, auth.GoogleProfile{FirstName: "Other", LastName: "Name"}); err != nil {
		t.Fatal(err)
	}
	if v, _ := f.svc.Get(ctx, userGoogle); v.FirstName != "Ali" {
		t.Fatalf("%+v", v)
	}
}

var avatarKeyPattern = regexp.MustCompile(`^avatars/` + userEmail + `/[0-9a-f]{32}\.jpg$`)

func (f *fixture) fileExists(key string) bool {
	_, err := os.Stat(filepath.Join(f.root, filepath.FromSlash(key)))
	return err == nil
}

func TestAvatarUploadReplaceRemove(t *testing.T) {
	f := newFixture(t)
	img := encodeJPEG(t, halves(300, 300))

	v1, err := f.svc.UploadAvatar(ctx, userEmail, img)
	if err != nil {
		t.Fatal(err)
	}
	if !avatarKeyPattern.MatchString(v1.AvatarKey) || !f.fileExists(v1.AvatarKey) {
		t.Fatalf("key %q", v1.AvatarKey)
	}
	file, err := f.svc.OpenAvatar(ctx, v1.AvatarKey)
	if err != nil {
		t.Fatal(err)
	}
	_ = file.Body.Close()
	if file.ContentType != AvatarContentType {
		t.Fatal(file.ContentType)
	}

	v2, err := f.svc.UploadAvatar(ctx, userEmail, img)
	if err != nil {
		t.Fatal(err)
	}
	if v2.AvatarKey == v1.AvatarKey || f.fileExists(v1.AvatarKey) || !f.fileExists(v2.AvatarKey) {
		t.Fatal("old avatar not replaced")
	}

	v3, err := f.svc.RemoveAvatar(ctx, userEmail)
	if err != nil {
		t.Fatal(err)
	}
	if v3.AvatarKey != "" || f.fileExists(v2.AvatarKey) {
		t.Fatal("avatar not removed")
	}
	// Removing again is fine.
	if _, err := f.svc.RemoveAvatar(ctx, userEmail); err != nil {
		t.Fatal(err)
	}
}

func TestAvatarUploadKeepsSetupState(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.UploadAvatar(ctx, userEmail, encodeJPEG(t, halves(64, 64))); err != nil {
		t.Fatal(err)
	}
	in := validSetup()
	v, err := f.svc.CompleteSetup(ctx, userEmail, in)
	if err != nil || v.AvatarKey == "" {
		t.Fatalf("avatar lost by setup: %+v %v", v, err)
	}
}

func TestAvatarUploadRejectsBadFiles(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.UploadAvatar(ctx, userEmail, []byte("<svg/>"))
	wantFields(t, err, "file")
	_, err = f.svc.UploadAvatar(ctx, userEmail, make([]byte, AvatarMaxBytes+1))
	wantFields(t, err, "file")
	entries, _ := os.ReadDir(f.root)
	if len(entries) != 0 {
		t.Fatal("rejected upload left files behind")
	}
}

func TestOpenAvatarMissing(t *testing.T) {
	f := newFixture(t)
	_, err := f.svc.OpenAvatar(ctx, "avatars/"+userEmail+"/"+"00000000000000000000000000000000.jpg")
	wantCode(t, err, apperr.CodeNotFound)
}
