package auth

import (
	"errors"
	"regexp"
	"testing"
	"time"
)

func TestNewCode(t *testing.T) {
	re := regexp.MustCompile(`^\d{6}$`)
	seen := map[string]bool{}
	for range 200 {
		c, err := NewCode()
		if err != nil {
			t.Fatal(err)
		}
		if !re.MatchString(c) {
			t.Fatalf("bad code %q", c)
		}
		seen[c] = true
	}
	if len(seen) < 190 {
		t.Fatalf("codes are not random enough: %d unique of 200", len(seen))
	}
}

func TestCodeHasher(t *testing.T) {
	if _, err := NewCodeHasher("short"); !errors.Is(err, ErrWeakSecret) {
		t.Fatalf("want ErrWeakSecret, got %v", err)
	}
	h, _ := NewCodeHasher(testSecret)
	a := h.Hash("a@example.com", "123456")
	if a != h.Hash("a@example.com", "123456") {
		t.Fatal("hash must be deterministic")
	}
	if a == h.Hash("b@example.com", "123456") {
		t.Fatal("hash must depend on email")
	}
	if a == h.Hash("a@example.com", "123457") {
		t.Fatal("hash must depend on code")
	}
	other, _ := NewCodeHasher(testSecret + "x")
	if a == other.Hash("a@example.com", "123456") {
		t.Fatal("hash must depend on secret")
	}
}

func TestCheckCanSend(t *testing.T) {
	p := DefaultCodePolicy
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)

	if _, err := p.CheckCanSend(now, nil, 0); err != nil {
		t.Fatalf("first code: %v", err)
	}
	recent := now.Add(-20 * time.Second)
	wait, err := p.CheckCanSend(now, &recent, 1)
	if !errors.Is(err, ErrCodeResendTooSoon) || wait != 40*time.Second {
		t.Fatalf("cooldown: wait=%v err=%v", wait, err)
	}
	old := now.Add(-61 * time.Second)
	if _, err := p.CheckCanSend(now, &old, 1); err != nil {
		t.Fatalf("after cooldown: %v", err)
	}
	if _, err := p.CheckCanSend(now, &old, p.DailyCap); !errors.Is(err, ErrCodeDailyCapExceeded) {
		t.Fatalf("daily cap: %v", err)
	}
}

func TestVerifyCode(t *testing.T) {
	p := DefaultCodePolicy
	h, _ := NewCodeHasher(testSecret)
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	email := "a@example.com"
	state := CodeState{Hash: h.Hash(email, "123456"), ExpiresAt: now.Add(p.TTL)}

	if err := p.Verify(now, state, h, email, "123456"); err != nil {
		t.Fatalf("correct code: %v", err)
	}
	if err := p.Verify(now, state, h, email, "654321"); !errors.Is(err, ErrCodeMismatch) {
		t.Fatalf("wrong code: %v", err)
	}
	if err := p.Verify(now, state, h, "b@example.com", "123456"); !errors.Is(err, ErrCodeMismatch) {
		t.Fatalf("other email: %v", err)
	}
	if err := p.Verify(state.ExpiresAt, state, h, email, "123456"); !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("expired: %v", err)
	}

	locked := state
	locked.Attempts = p.MaxAttempts
	// Even the correct code is refused once the attempts are used up.
	if err := p.Verify(now, locked, h, email, "123456"); !errors.Is(err, ErrCodeTooManyAttempts) {
		t.Fatalf("locked: %v", err)
	}

	used := state
	usedAt := now.Add(-time.Minute)
	used.UsedAt = &usedAt
	if err := p.Verify(now, used, h, email, "123456"); !errors.Is(err, ErrCodeAlreadyUsed) {
		t.Fatalf("used: %v", err)
	}
}
