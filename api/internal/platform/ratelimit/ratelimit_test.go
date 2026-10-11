package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryFixedWindow(t *testing.T) {
	now := time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)
	m := NewMemory()
	m.now = func() time.Time { return now }
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		if r := m.Allow(ctx, "ip:1", 3, time.Minute); !r.Allowed || r.Remaining != 3-i {
			t.Fatalf("hit %d: %+v", i, r)
		}
	}
	r := m.Allow(ctx, "ip:1", 3, time.Minute)
	if r.Allowed || r.RetryAfter != time.Minute {
		t.Fatalf("4th hit should be blocked for a minute: %+v", r)
	}
	if r := m.Allow(ctx, "ip:2", 3, time.Minute); !r.Allowed {
		t.Fatal("keys must be independent")
	}

	now = now.Add(time.Minute)
	if r := m.Allow(ctx, "ip:1", 3, time.Minute); !r.Allowed || r.Remaining != 2 {
		t.Fatalf("new window should reset the count: %+v", r)
	}
}

func TestMemoryDropsExpiredWindows(t *testing.T) {
	now := time.Now()
	m := NewMemory()
	m.now = func() time.Time { return now }
	m.Allow(context.Background(), "old", 1, time.Second)

	now = now.Add(2 * time.Minute)
	m.Allow(context.Background(), "new", 1, time.Second)
	if _, ok := m.windows["old"]; ok {
		t.Error("expired window was not collected")
	}
}
