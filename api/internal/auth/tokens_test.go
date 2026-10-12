package auth

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "0123456789abcdef0123456789abcdef" // 32 bytes

func newTestTokens(t *testing.T, now time.Time) *AccessTokens {
	t.Helper()
	a, err := NewAccessTokens(testSecret, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	a.now = func() time.Time { return now }
	return a
}

func TestNewAccessTokensValidates(t *testing.T) {
	if _, err := NewAccessTokens("short", time.Minute); !errors.Is(err, ErrWeakSecret) {
		t.Fatalf("want ErrWeakSecret, got %v", err)
	}
	if _, err := NewAccessTokens(testSecret, 0); err == nil {
		t.Fatal("want error for zero ttl")
	}
}

func TestIssueAndVerify(t *testing.T) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	a := newTestTokens(t, now)

	tok, exp, err := a.Issue("user-1", RoleLearner, "session-1")
	if err != nil {
		t.Fatal(err)
	}
	if !exp.Equal(now.Add(15 * time.Minute)) {
		t.Fatalf("exp = %v", exp)
	}
	c, err := a.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.UserID != "user-1" || c.Role != RoleLearner || c.SessionID != "session-1" || !c.ExpiresAt.Equal(exp) {
		t.Fatalf("claims = %+v", c)
	}
}

func TestVerifyExpired(t *testing.T) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	a := newTestTokens(t, now)
	tok, _, _ := a.Issue("u", RoleLearner, "s")

	a.now = func() time.Time { return now.Add(16 * time.Minute) }
	if _, err := a.Verify(tok); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("want ErrExpiredToken, got %v", err)
	}
}

func TestVerifyRejectsForgedTokens(t *testing.T) {
	now := time.Now()
	a := newTestTokens(t, now)
	good, _, _ := a.Issue("u", RoleLearner, "s")

	sign := func(method jwt.SigningMethod, key any, claims jwt.Claims) string {
		s, err := jwt.NewWithClaims(method, claims).SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	base := func() jwtClaims {
		return jwtClaims{Role: RoleLearner, SessionID: "s", RegisteredClaims: jwt.RegisteredClaims{
			Issuer: TokenIssuer, Subject: "u", Audience: jwt.ClaimStrings{TokenAudience},
			IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
		}}
	}
	other := base()
	wrongIss := base()
	wrongIss.Issuer = "someone-else"
	wrongAud := base()
	wrongAud.Audience = jwt.ClaimStrings{"other"}
	noSid := base()
	noSid.SessionID = ""
	noExp := base()
	noExp.ExpiresAt = nil

	cases := map[string]string{
		"garbage":       "not.a.jwt",
		"tampered":      good[:len(good)-2] + "xx",
		"wrong secret":  sign(jwt.SigningMethodHS256, []byte(strings.Repeat("z", 32)), other),
		"alg none":      sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, other),
		"hs512":         sign(jwt.SigningMethodHS512, []byte(testSecret), other),
		"wrong issuer":  sign(jwt.SigningMethodHS256, []byte(testSecret), wrongIss),
		"wrong aud":     sign(jwt.SigningMethodHS256, []byte(testSecret), wrongAud),
		"no session id": sign(jwt.SigningMethodHS256, []byte(testSecret), noSid),
		"no exp":        sign(jwt.SigningMethodHS256, []byte(testSecret), noExp),
	}
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := a.Verify(tok); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("want ErrInvalidToken, got %v", err)
			}
		})
	}
}

func TestOpaqueToken(t *testing.T) {
	tok, hash, err := NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 43 { // 32 bytes, unpadded base64url
		t.Fatalf("token length = %d", len(tok))
	}
	if hash != HashOpaqueToken(tok) || len(hash) != 64 {
		t.Fatalf("hash mismatch: %s", hash)
	}
	tok2, _, _ := NewOpaqueToken()
	if tok == tok2 {
		t.Fatal("tokens must be random")
	}
}
