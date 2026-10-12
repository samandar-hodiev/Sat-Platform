package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// CodeLength is the number of digits in an email verification code.
const CodeLength = 6

// CodePolicy limits verification codes (PRD §6.1: expiry, attempt limit, resend limit).
type CodePolicy struct {
	TTL            time.Duration // a code expires this long after it is sent
	MaxAttempts    int           // wrong guesses allowed per code; then a new code is needed
	ResendCooldown time.Duration // minimum time between two codes to the same email
	DailyCap       int           // codes per email per rolling 24 hours
}

var DefaultCodePolicy = CodePolicy{
	TTL:            10 * time.Minute,
	MaxAttempts:    5,
	ResendCooldown: 60 * time.Second,
	DailyCap:       10,
}

var (
	ErrCodeExpired          = errors.New("verification code expired")
	ErrCodeTooManyAttempts  = errors.New("too many wrong attempts for this code")
	ErrCodeMismatch         = errors.New("verification code is incorrect")
	ErrCodeAlreadyUsed      = errors.New("verification code already used")
	ErrCodeResendTooSoon    = errors.New("a code was sent recently")
	ErrCodeDailyCapExceeded = errors.New("too many codes requested today")
)

// CodeState is what the repository stores for one issued code.
type CodeState struct {
	Hash      string
	ExpiresAt time.Time
	Attempts  int
	UsedAt    *time.Time
}

// CodeHasher hashes codes with HMAC-SHA256 under a server secret. A plain hash would not be
// enough: a 6-digit code has only 10^6 values, so a leaked table could be reversed instantly.
type CodeHasher struct{ key []byte }

func NewCodeHasher(secret string) (*CodeHasher, error) {
	if len(secret) < MinJWTSecretLength {
		return nil, ErrWeakSecret
	}
	// Domain separation: the JWT secret is reused, but never as the same key.
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte("sat-platform/email-verification-code/v1"))
	return &CodeHasher{key: mac.Sum(nil)}, nil
}

// Hash binds the code to the email, so a code sent to one address cannot verify another.
func (h *CodeHasher) Hash(email, code string) string {
	mac := hmac.New(sha256.New, h.key)
	mac.Write([]byte(email))
	mac.Write([]byte{0})
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

// NewCode returns a uniformly random numeric code of CodeLength digits (leading zeros kept).
func NewCode() (string, error) {
	max := big.NewInt(1)
	for range CodeLength {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("generate code: %w", err)
	}
	return fmt.Sprintf("%0*d", CodeLength, n), nil
}

// CheckCanSend decides whether a new code may be sent. lastSentAt is the newest code for the
// email (nil if none) and sentLast24h counts codes in the last 24 hours. retryAfter tells the
// client how long to wait when sending is refused.
func (p CodePolicy) CheckCanSend(now time.Time, lastSentAt *time.Time, sentLast24h int) (retryAfter time.Duration, err error) {
	if sentLast24h >= p.DailyCap {
		return 0, ErrCodeDailyCapExceeded
	}
	if lastSentAt != nil {
		if wait := lastSentAt.Add(p.ResendCooldown).Sub(now); wait > 0 {
			return wait.Round(time.Second), ErrCodeResendTooSoon
		}
	}
	return 0, nil
}

// Verify checks a submitted code against the stored state. The caller must persist
// state.Attempts+1 whenever the result is ErrCodeMismatch, and mark the code used on success,
// in the same transaction that reads it.
func (p CodePolicy) Verify(now time.Time, state CodeState, hasher *CodeHasher, email, code string) error {
	switch {
	case state.UsedAt != nil:
		return ErrCodeAlreadyUsed
	case !now.Before(state.ExpiresAt):
		return ErrCodeExpired
	case state.Attempts >= p.MaxAttempts:
		return ErrCodeTooManyAttempts
	}
	if !hmac.Equal([]byte(hasher.Hash(email, code)), []byte(state.Hash)) {
		return ErrCodeMismatch
	}
	return nil
}
