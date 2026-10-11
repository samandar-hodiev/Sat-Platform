// Package ratelimit provides the request limiter used by the rate-limit middleware.
//
// Limiter is an interface so a shared store (Redis or PostgreSQL) can replace the
// in-memory implementation when the API runs on more than one instance. Feature modules
// (sign-in, verification codes, AI) reuse it with their own keys and budgets instead of
// writing their own limiter.
package ratelimit

import (
	"context"
	"sync"
	"time"
)

// Result describes the decision for one request.
type Result struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
}

type Limiter interface {
	// Allow counts one hit for key against limit per window.
	Allow(ctx context.Context, key string, limit int, window time.Duration) Result
}

// Memory is a fixed-window limiter kept in process memory. Counts are per instance and
// reset on restart.
type Memory struct {
	mu      sync.Mutex
	windows map[string]*window
	now     func() time.Time
	lastGC  time.Time
}

type window struct {
	start time.Time
	ttl   time.Duration
	count int
}

func NewMemory() *Memory {
	return &Memory{windows: make(map[string]*window), now: time.Now}
}

func (m *Memory) Allow(_ context.Context, key string, limit int, win time.Duration) Result {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	m.gc(now)

	w, ok := m.windows[key]
	if !ok || now.Sub(w.start) >= win {
		w = &window{start: now, ttl: win}
		m.windows[key] = w
	}
	w.count++

	remaining := limit - w.count
	if remaining < 0 {
		remaining = 0
	}
	res := Result{Allowed: w.count <= limit, Limit: limit, Remaining: remaining}
	if !res.Allowed {
		res.RetryAfter = w.start.Add(win).Sub(now)
	}
	return res
}

// gc drops expired windows at most once a minute so memory stays bounded.
func (m *Memory) gc(now time.Time) {
	if now.Sub(m.lastGC) < time.Minute {
		return
	}
	m.lastGC = now
	for k, w := range m.windows {
		if now.Sub(w.start) >= w.ttl {
			delete(m.windows, k)
		}
	}
}
