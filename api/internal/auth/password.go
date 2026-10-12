package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Password length limits, counted in characters. The upper bound keeps hashing cost bounded
// and is far above any real passphrase.
const (
	PasswordMinLength = 8
	PasswordMaxLength = 128
)

var (
	ErrPasswordTooShort = fmt.Errorf("password must be at least %d characters", PasswordMinLength)
	ErrPasswordTooLong  = fmt.Errorf("password must be at most %d characters", PasswordMaxLength)
	// ErrInvalidHash means a stored hash could not be parsed. It is an internal error,
	// never a "wrong password".
	ErrInvalidHash = errors.New("invalid password hash")
)

// Argon2Params are the Argon2id cost parameters. They are encoded into every hash, so they can
// be raised later: old hashes keep verifying and NeedsRehash reports them for upgrade on the
// next successful sign-in.
type Argon2Params struct {
	MemoryKiB   uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params follow the OWASP Password Storage Cheat Sheet minimum for Argon2id
// (19 MiB, 2 iterations, 1 lane). Memory per sign-in stays small enough that the
// rate-limited sign-in endpoint cannot be used to exhaust server memory.
var DefaultArgon2Params = Argon2Params{
	MemoryKiB:   19 * 1024,
	Iterations:  2,
	Parallelism: 1,
	SaltLength:  16,
	KeyLength:   32,
}

// ValidatePassword checks the password policy. Only length is enforced (NIST SP 800-63B):
// composition rules push users to predictable patterns.
func ValidatePassword(password string) error {
	n := utf8.RuneCountInString(password)
	switch {
	case n < PasswordMinLength:
		return ErrPasswordTooShort
	case n > PasswordMaxLength:
		return ErrPasswordTooLong
	}
	return nil
}

// HashPassword returns a PHC-format Argon2id hash:
// $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash> (unpadded standard base64).
func HashPassword(password string, p Argon2Params) (string, error) {
	salt := make([]byte, p.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, p.Iterations, p.MemoryKiB, p.Parallelism, p.KeyLength)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.MemoryKiB, p.Iterations, p.Parallelism,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword reports whether password matches the encoded hash, in constant time.
// A malformed hash returns ErrInvalidHash.
func VerifyPassword(password, encoded string) (bool, error) {
	p, salt, key, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}
	other := argon2.IDKey([]byte(password), salt, p.Iterations, p.MemoryKiB, p.Parallelism, p.KeyLength)
	return subtle.ConstantTimeCompare(key, other) == 1, nil
}

// NeedsRehash reports whether encoded was made with parameters other than p, so the caller
// can store a fresh hash after a successful sign-in.
func NeedsRehash(encoded string, p Argon2Params) bool {
	got, salt, key, err := decodeHash(encoded)
	if err != nil {
		return true
	}
	return got.MemoryKiB != p.MemoryKiB || got.Iterations != p.Iterations ||
		got.Parallelism != p.Parallelism || uint32(len(salt)) != p.SaltLength ||
		uint32(len(key)) != p.KeyLength
}

// DummyPasswordCheck spends the same time as a real VerifyPassword. Sign-in calls it when the
// email is unknown or has no password (Google-only account), so response time does not reveal
// which emails are registered.
type DummyPasswordCheck struct{ hash string }

func NewDummyPasswordCheck(p Argon2Params) (*DummyPasswordCheck, error) {
	h, err := HashPassword("dummy-password-for-timing", p)
	if err != nil {
		return nil, err
	}
	return &DummyPasswordCheck{hash: h}, nil
}

func (d *DummyPasswordCheck) Run(password string) {
	_, _ = VerifyPassword(password, d.hash)
}

var b64 = base64.RawStdEncoding

func decodeHash(encoded string) (Argon2Params, []byte, []byte, error) {
	// "", "argon2id", "v=19", "m=..,t=..,p=..", salt, hash
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	var p Argon2Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Iterations, &p.Parallelism); err != nil {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	// Bounds stop a corrupted or hostile row from making verification allocate gigabytes.
	if p.MemoryKiB < 8 || p.MemoryKiB > 1<<20 || p.Iterations < 1 || p.Iterations > 20 || p.Parallelism < 1 {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	key, err := b64.DecodeString(parts[5])
	if err != nil || len(key) < 16 {
		return Argon2Params{}, nil, nil, ErrInvalidHash
	}
	p.SaltLength = uint32(len(salt))
	p.KeyLength = uint32(len(key))
	return p, salt, key, nil
}
