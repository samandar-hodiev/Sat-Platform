package auth

import (
	"errors"
	"strings"
	"testing"
)

// fastParams keep tests quick; production uses DefaultArgon2Params.
var fastParams = Argon2Params{MemoryKiB: 64, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery", fastParams)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Fatalf("unexpected hash format: %s", hash)
	}
	if ok, err := VerifyPassword("correct horse battery", hash); err != nil || !ok {
		t.Fatalf("correct password: ok=%v err=%v", ok, err)
	}
	if ok, err := VerifyPassword("wrong horse battery", hash); err != nil || ok {
		t.Fatalf("wrong password: ok=%v err=%v", ok, err)
	}
}

func TestHashPasswordUsesRandomSalt(t *testing.T) {
	a, _ := HashPassword("same password", fastParams)
	b, _ := HashPassword("same password", fastParams)
	if a == b {
		t.Fatal("two hashes of the same password must differ")
	}
}

func TestDefaultParamsRoundTrip(t *testing.T) {
	hash, err := HashPassword("p@ssw0rd-123", DefaultArgon2Params)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected hash: %s", hash)
	}
	if ok, _ := VerifyPassword("p@ssw0rd-123", hash); !ok {
		t.Fatal("default params must verify")
	}
}

func TestVerifyPasswordRejectsMalformedHashes(t *testing.T) {
	valid, _ := HashPassword("x", fastParams)
	parts := strings.Split(valid, "$")
	cases := map[string]string{
		"empty":         "",
		"bcrypt":        "$2a$10$abcdefghijklmnopqrstuuJ1yQzQ8m1fM6l9QhQe2G1Y0bJ7Wb8m2",
		"argon2i":       strings.Replace(valid, "argon2id", "argon2i", 1),
		"wrong version": strings.Replace(valid, "v=19", "v=16", 1),
		"bad params":    strings.Replace(valid, "m=64,t=1,p=1", "m=x", 1),
		"huge memory":   strings.Replace(valid, "m=64", "m=99999999", 1),
		"zero threads":  strings.Replace(valid, "p=1", "p=0", 1),
		"bad salt":      strings.Join([]string{"", parts[1], parts[2], parts[3], "!!", parts[5]}, "$"),
		"short key":     strings.Join([]string{"", parts[1], parts[2], parts[3], parts[4], "AAAA"}, "$"),
		"extra field":   valid + "$x",
	}
	for name, h := range cases {
		t.Run(name, func(t *testing.T) {
			ok, err := VerifyPassword("x", h)
			if ok || !errors.Is(err, ErrInvalidHash) {
				t.Fatalf("want ErrInvalidHash, got ok=%v err=%v", ok, err)
			}
		})
	}
}

func TestNeedsRehash(t *testing.T) {
	hash, _ := HashPassword("x", fastParams)
	if NeedsRehash(hash, fastParams) {
		t.Fatal("same params must not need rehash")
	}
	stronger := fastParams
	stronger.Iterations = 2
	if !NeedsRehash(hash, stronger) {
		t.Fatal("changed params must need rehash")
	}
	if !NeedsRehash("garbage", fastParams) {
		t.Fatal("malformed hash must need rehash")
	}
}

func TestValidatePassword(t *testing.T) {
	cases := []struct {
		in   string
		want error
	}{
		{"", ErrPasswordTooShort},
		{"1234567", ErrPasswordTooShort},
		{"12345678", nil},
		{strings.Repeat("ж", 8), nil}, // counted in characters, not bytes
		{strings.Repeat("a", PasswordMaxLength), nil},
		{strings.Repeat("a", PasswordMaxLength+1), ErrPasswordTooLong},
	}
	for _, c := range cases {
		if got := ValidatePassword(c.in); !errors.Is(got, c.want) {
			t.Errorf("ValidatePassword(len %d) = %v, want %v", len(c.in), got, c.want)
		}
	}
}

func TestDummyPasswordCheck(t *testing.T) {
	d, err := NewDummyPasswordCheck(fastParams)
	if err != nil {
		t.Fatal(err)
	}
	d.Run("anything") // must not panic
}
