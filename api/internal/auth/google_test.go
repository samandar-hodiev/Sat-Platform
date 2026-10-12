package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testClientID = "web-client.apps.googleusercontent.com"

type fakeGoogle struct {
	t       *testing.T
	key     *rsa.PrivateKey
	kid     string
	fetches atomic.Int32
	status  atomic.Int32
	srv     *httptest.Server
}

func newFakeGoogle(t *testing.T) *fakeGoogle {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	g := &fakeGoogle{t: t, key: key, kid: "kid-1"}
	g.status.Store(http.StatusOK)
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		g.fetches.Add(1)
		if s := int(g.status.Load()); s != http.StatusOK {
			w.WriteHeader(s)
			return
		}
		pub := g.key.PublicKey
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{{
			"kid": g.kid, "kty": "RSA", "alg": "RS256", "use": "sig",
			"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}}})
	}))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *fakeGoogle) verifier(now time.Time) *GoogleVerifier {
	v, err := NewGoogleVerifier([]string{"android-client", testClientID}, g.srv.Client())
	if err != nil {
		g.t.Fatal(err)
	}
	v.keys = newJWKSCache(g.srv.URL, g.srv.Client())
	v.keys.now = func() time.Time { return now }
	v.now = func() time.Time { return now }
	return v
}

func (g *fakeGoogle) claims(now time.Time) jwt.MapClaims {
	return jwt.MapClaims{
		"iss": "https://accounts.google.com", "aud": testClientID, "sub": "1234567890",
		"email": "Learner@Gmail.com", "email_verified": true, "name": "Ali Valiyev",
		"given_name": "Ali", "family_name": "Valiyev",
		"picture": "https://lh3.googleusercontent.com/a/x",
		"iat":     now.Unix(), "exp": now.Add(time.Hour).Unix(),
	}
}

func (g *fakeGoogle) sign(claims jwt.MapClaims, kid string, key *rsa.PrivateKey) string {
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	if err != nil {
		g.t.Fatal(err)
	}
	return s
}

func TestGoogleVerifyValidToken(t *testing.T) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	g := newFakeGoogle(t)
	v := g.verifier(now)

	id, err := v.Verify(context.Background(), g.sign(g.claims(now), g.kid, g.key))
	if err != nil {
		t.Fatal(err)
	}
	want := GoogleIdentity{Subject: "1234567890", Email: "learner@gmail.com", Name: "Ali Valiyev",
		GivenName: "Ali", FamilyName: "Valiyev",
		PictureURL: "https://lh3.googleusercontent.com/a/x"}
	if id != want {
		t.Fatalf("identity = %+v", id)
	}

	// Keys are cached: a second verification does not refetch.
	if _, err := v.Verify(context.Background(), g.sign(g.claims(now), g.kid, g.key)); err != nil {
		t.Fatal(err)
	}
	if n := g.fetches.Load(); n != 1 {
		t.Fatalf("fetches = %d, want 1", n)
	}
}

func TestGoogleVerifyRejects(t *testing.T) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	g := newFakeGoogle(t)
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	mod := func(f func(jwt.MapClaims)) jwt.MapClaims {
		c := g.claims(now)
		f(c)
		return c
	}
	cases := map[string]struct {
		token string
		want  error
	}{
		"wrong key":          {g.sign(g.claims(now), g.kid, otherKey), ErrInvalidGoogleToken},
		"unknown kid":        {g.sign(g.claims(now), "kid-2", g.key), ErrInvalidGoogleToken},
		"wrong audience":     {g.sign(mod(func(c jwt.MapClaims) { c["aud"] = "someone-else" }), g.kid, g.key), ErrInvalidGoogleToken},
		"wrong issuer":       {g.sign(mod(func(c jwt.MapClaims) { c["iss"] = "https://evil.example" }), g.kid, g.key), ErrInvalidGoogleToken},
		"expired":            {g.sign(mod(func(c jwt.MapClaims) { c["exp"] = now.Add(-time.Hour).Unix() }), g.kid, g.key), ErrInvalidGoogleToken},
		"no sub":             {g.sign(mod(func(c jwt.MapClaims) { delete(c, "sub") }), g.kid, g.key), ErrInvalidGoogleToken},
		"email not verified": {g.sign(mod(func(c jwt.MapClaims) { c["email_verified"] = false }), g.kid, g.key), ErrGoogleEmailNotVerified},
		"verified missing":   {g.sign(mod(func(c jwt.MapClaims) { delete(c, "email_verified") }), g.kid, g.key), ErrGoogleEmailNotVerified},
		"garbage":            {"abc.def.ghi", ErrInvalidGoogleToken},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			v := g.verifier(now)
			if _, err := v.Verify(context.Background(), c.token); !errors.Is(err, c.want) {
				t.Fatalf("want %v, got %v", c.want, err)
			}
		})
	}

	t.Run("hs256 with public key as secret", func(t *testing.T) {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, g.claims(now))
		tok.Header["kid"] = g.kid
		s, _ := tok.SignedString(g.key.PublicKey.N.Bytes())
		if _, err := g.verifier(now).Verify(context.Background(), s); !errors.Is(err, ErrInvalidGoogleToken) {
			t.Fatalf("want ErrInvalidGoogleToken, got %v", err)
		}
	})

	t.Run("email_verified as string", func(t *testing.T) {
		s := g.sign(mod(func(c jwt.MapClaims) { c["email_verified"] = "true" }), g.kid, g.key)
		if _, err := g.verifier(now).Verify(context.Background(), s); err != nil {
			t.Fatalf("want ok, got %v", err)
		}
	})
}

func TestJWKSUnknownKidRefetchIsThrottled(t *testing.T) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	g := newFakeGoogle(t)
	v := g.verifier(now)
	ctx := context.Background()

	if _, err := v.Verify(ctx, g.sign(g.claims(now), g.kid, g.key)); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		_, _ = v.Verify(ctx, g.sign(g.claims(now), "forged-kid", g.key))
	}
	if n := g.fetches.Load(); n != 1 {
		t.Fatalf("unknown kids within a minute must not refetch, fetches = %d", n)
	}

	// Google rotates keys: after the throttle window the new kid is fetched.
	later := now.Add(2 * time.Minute)
	v.now = func() time.Time { return later }
	v.keys.now = func() time.Time { return later }
	g.kid = "kid-rotated"
	if _, err := v.Verify(ctx, g.sign(g.claims(later), "kid-rotated", g.key)); err != nil {
		t.Fatalf("rotated key: %v", err)
	}
	if n := g.fetches.Load(); n != 2 {
		t.Fatalf("fetches = %d, want 2", n)
	}
}

func TestJWKSKeepsKnownKeyWhenGoogleIsDown(t *testing.T) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	g := newFakeGoogle(t)
	v := g.verifier(now)
	ctx := context.Background()
	if _, err := v.Verify(ctx, g.sign(g.claims(now), g.kid, g.key)); err != nil {
		t.Fatal(err)
	}

	later := now.Add(2 * time.Hour) // cache expired
	v.now = func() time.Time { return later }
	v.keys.now = func() time.Time { return later }
	g.status.Store(http.StatusServiceUnavailable)
	if _, err := v.Verify(ctx, g.sign(g.claims(later), g.kid, g.key)); err != nil {
		t.Fatalf("known key must still verify while Google is down: %v", err)
	}
}

func TestNewGoogleVerifierRequiresClientID(t *testing.T) {
	if _, err := NewGoogleVerifier(nil, nil); err == nil {
		t.Fatal("want error")
	}
}

func TestMaxAge(t *testing.T) {
	cases := map[string]time.Duration{
		"":                            jwksDefaultTTL,
		"public, max-age=19800":       19800 * time.Second,
		"max-age=0":                   jwksDefaultTTL,
		"max-age=9999999, must-reval": 24 * time.Hour,
	}
	for in, want := range cases {
		if got := maxAge(in); got != want {
			t.Errorf("maxAge(%q) = %v, want %v", in, got, want)
		}
	}
}
