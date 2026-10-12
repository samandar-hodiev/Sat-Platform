package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GoogleCertsURL is Google's JWKS endpoint for ID-token signing keys.
const GoogleCertsURL = "https://www.googleapis.com/oauth2/v3/certs"

var googleIssuers = []string{"accounts.google.com", "https://accounts.google.com"}

var (
	// ErrInvalidGoogleToken covers every rejected ID token. The cause is wrapped for logs.
	ErrInvalidGoogleToken = errors.New("invalid Google ID token")
	// ErrGoogleEmailNotVerified: the Google account's email is not verified, so it cannot be
	// trusted to identify (or be linked to) a platform account.
	ErrGoogleEmailNotVerified = errors.New("Google account email is not verified")
)

// GoogleIdentity is the verified identity from a Google ID token.
type GoogleIdentity struct {
	Subject    string // stable Google account id ("sub"); the link key, not the email
	Email      string // normalized
	Name       string
	GivenName  string
	FamilyName string
	PictureURL string
}

// GoogleVerifier verifies Google ID tokens sent by the web, Android and iOS apps
// (POST /auth/google { id_token }). It checks the RS256 signature against Google's published
// keys, issuer, expiry, that the audience is one of our client IDs and email_verified.
type GoogleVerifier struct {
	clientIDs []string
	keys      *jwksCache
	now       func() time.Time
}

func NewGoogleVerifier(clientIDs []string, client *http.Client) (*GoogleVerifier, error) {
	if len(clientIDs) == 0 {
		return nil, errors.New("at least one Google client ID is required")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &GoogleVerifier{
		clientIDs: clientIDs,
		keys:      newJWKSCache(GoogleCertsURL, client),
		now:       time.Now,
	}, nil
}

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"` // bool, or "true" in older tokens
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	jwt.RegisteredClaims
}

func (v *GoogleVerifier) Verify(ctx context.Context, idToken string) (GoogleIdentity, error) {
	var c googleClaims
	_, err := jwt.ParseWithClaims(idToken, &c, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("missing kid")
		}
		return v.keys.key(ctx, kid)
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(30*time.Second),
		jwt.WithTimeFunc(v.now),
	)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("%w: %w", ErrInvalidGoogleToken, err)
	}
	if !contains(googleIssuers, c.Issuer) {
		return GoogleIdentity{}, fmt.Errorf("%w: unexpected issuer", ErrInvalidGoogleToken)
	}
	if !audienceMatches(c.Audience, v.clientIDs) {
		return GoogleIdentity{}, fmt.Errorf("%w: unexpected audience", ErrInvalidGoogleToken)
	}
	if c.Subject == "" || c.Email == "" {
		return GoogleIdentity{}, fmt.Errorf("%w: missing sub or email", ErrInvalidGoogleToken)
	}
	if !isTrue(c.EmailVerified) {
		return GoogleIdentity{}, ErrGoogleEmailNotVerified
	}
	email, err := NormalizeEmail(c.Email)
	if err != nil {
		return GoogleIdentity{}, fmt.Errorf("%w: %w", ErrInvalidGoogleToken, err)
	}
	return GoogleIdentity{Subject: c.Subject, Email: email, Name: c.Name, GivenName: c.GivenName,
		FamilyName: c.FamilyName, PictureURL: c.Picture}, nil
}

func audienceMatches(aud jwt.ClaimStrings, clientIDs []string) bool {
	for _, a := range aud {
		if contains(clientIDs, a) {
			return true
		}
	}
	return false
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func isTrue(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true"
	}
	return false
}

// jwksCache keeps Google's RSA keys in memory for as long as the response's Cache-Control
// max-age allows. An unknown kid (Google rotated keys) triggers a refetch, but at most once per
// minRefetch so forged kids cannot make us hammer Google.
type jwksCache struct {
	url    string
	client *http.Client
	now    func() time.Time

	mu          sync.Mutex
	keys        map[string]*rsa.PublicKey
	expiresAt   time.Time
	lastFetchAt time.Time
}

const (
	jwksDefaultTTL = time.Hour
	jwksMinRefetch = time.Minute
	jwksMaxBytes   = 1 << 20
)

func newJWKSCache(url string, client *http.Client) *jwksCache {
	return &jwksCache{url: url, client: client, now: time.Now}
}

func (c *jwksCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.now()
	k, ok := c.keys[kid]
	fresh := now.Before(c.expiresAt)
	if ok && fresh {
		return k, nil
	}
	if !fresh || now.Sub(c.lastFetchAt) >= jwksMinRefetch {
		if err := c.fetch(ctx, now); err != nil {
			if ok {
				// Google unreachable: a key we already trust is still better than failing
				// every Google sign-in.
				return k, nil
			}
			return nil, err
		}
		if k, ok = c.keys[kid]; ok {
			return k, nil
		}
	}
	return nil, fmt.Errorf("unknown signing key %q", kid)
}

func (c *jwksCache) fetch(ctx context.Context, now time.Time) error {
	c.lastFetchAt = now
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch Google keys: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch Google keys: status %d", resp.StatusCode)
	}
	var body struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, jwksMaxBytes)).Decode(&body); err != nil {
		return fmt.Errorf("decode Google keys: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(body.Keys))
	for _, k := range body.Keys {
		if k.Kty != "RSA" || k.Kid == "" {
			continue
		}
		pub, err := rsaKey(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return errors.New("Google keys response has no RSA keys")
	}
	c.keys = keys
	c.expiresAt = now.Add(maxAge(resp.Header.Get("Cache-Control")))
	return nil
}

func rsaKey(n, e string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil || len(eb) == 0 || len(eb) > 4 {
		return nil, errors.New("bad exponent")
	}
	exp := int(new(big.Int).SetBytes(eb).Int64())
	if exp < 3 {
		return nil, errors.New("bad exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: exp}, nil
}

var maxAgeRe = regexp.MustCompile(`max-age=(\d+)`)

func maxAge(cacheControl string) time.Duration {
	m := maxAgeRe.FindStringSubmatch(cacheControl)
	if m == nil {
		return jwksDefaultTTL
	}
	s, err := strconv.Atoi(m[1])
	if err != nil || s <= 0 {
		return jwksDefaultTTL
	}
	// Never trust a cache lifetime longer than a day.
	return min(time.Duration(s)*time.Second, 24*time.Hour)
}
