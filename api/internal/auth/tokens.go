package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Token issuer and audience. Both are checked on every request, so a token minted for another
// service with the same secret is rejected.
const (
	TokenIssuer   = "sat-platform-api"
	TokenAudience = "sat-platform"
)

// MinJWTSecretLength is the minimum JWT_SECRET length in bytes (HS256 needs a 256-bit key).
const MinJWTSecretLength = 32

var (
	ErrWeakSecret = fmt.Errorf("JWT secret must be at least %d bytes", MinJWTSecretLength)
	// ErrInvalidToken covers every rejected access token: bad signature, wrong algorithm,
	// issuer or audience, malformed claims. Callers answer 401 without saying which.
	ErrInvalidToken = errors.New("invalid access token")
	// ErrExpiredToken lets clients tell "refresh and retry" apart from "sign in again".
	ErrExpiredToken = errors.New("access token expired")
)

// Role is the user's role as carried in the access token. Owner roles are added by 1.02.
type Role string

const RoleLearner Role = "learner"

// AccessClaims are the claims of an access token.
type AccessClaims struct {
	UserID    string    // sub
	Role      Role      // role
	SessionID string    // sid: the session row the refresh token belongs to
	IssuedAt  time.Time // iat
	ExpiresAt time.Time // exp
}

type jwtClaims struct {
	Role      Role   `json:"role"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

// AccessTokens issues and verifies access tokens.
type AccessTokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewAccessTokens(secret string, ttl time.Duration) (*AccessTokens, error) {
	if len(secret) < MinJWTSecretLength {
		return nil, ErrWeakSecret
	}
	if ttl <= 0 {
		return nil, errors.New("access token lifetime must be positive")
	}
	return &AccessTokens{secret: []byte(secret), ttl: ttl, now: time.Now}, nil
}

// Issue signs an access token for the user and session and returns it with its expiry.
func (a *AccessTokens) Issue(userID string, role Role, sessionID string) (string, time.Time, error) {
	now := a.now().UTC().Truncate(time.Second)
	exp := now.Add(a.ttl)
	claims := jwtClaims{
		Role:      role,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    TokenIssuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{TokenAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}
	return signed, exp, nil
}

// Verify checks signature (HS256 only), issuer, audience and expiry, and returns the claims.
func (a *AccessTokens) Verify(token string) (AccessClaims, error) {
	var c jwtClaims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return a.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(TokenIssuer),
		jwt.WithAudience(TokenAudience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(a.now),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return AccessClaims{}, ErrExpiredToken
		}
		return AccessClaims{}, ErrInvalidToken
	}
	if c.Subject == "" || c.SessionID == "" || c.Role == "" {
		return AccessClaims{}, ErrInvalidToken
	}
	return AccessClaims{
		UserID:    c.Subject,
		Role:      c.Role,
		SessionID: c.SessionID,
		IssuedAt:  c.IssuedAt.Time,
		ExpiresAt: c.ExpiresAt.Time,
	}, nil
}

// opaqueTokenBytes is the entropy of refresh and password-reset tokens (256 bits).
const opaqueTokenBytes = 32

// NewOpaqueToken returns a random URL-safe token for the client and the hash to store.
// Refresh and password-reset tokens use it; the plain token is never stored.
func NewOpaqueToken() (token, hash string, err error) {
	b := make([]byte, opaqueTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashOpaqueToken(token), nil
}

// HashOpaqueToken is the lookup key stored in the database. A plain SHA-256 is enough: the
// token has 256 bits of entropy, so it cannot be brute-forced from a leaked hash.
func HashOpaqueToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
