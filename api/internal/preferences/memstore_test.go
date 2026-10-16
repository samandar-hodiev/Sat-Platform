package preferences

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/samandar-hodiev/sat-platform/api/internal/platform/storage"
)

const (
	userA = "11111111-1111-4111-8111-111111111111"
	userB = "22222222-2222-4222-8222-222222222222"
	// userGone has no users row: writes fail like the FK does in PostgreSQL.
	userGone = "44444444-4444-4444-8444-444444444444"
)

// memStore mirrors PostgresStore: rows exist only for known users, writes go through Patch.Apply.
type memStore struct {
	mu    sync.Mutex
	users map[string]bool
	rows  map[string]Prefs
}

func newMemStore(users ...string) *memStore {
	s := &memStore{users: map[string]bool{}, rows: map[string]Prefs{}}
	for _, u := range users {
		s.users[u] = true
	}
	return s
}

func (s *memStore) Get(_ context.Context, userID string) (Prefs, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.rows[userID]
	if !ok {
		return Prefs{}, ErrNotFound
	}
	return p, nil
}

func (s *memStore) locked(userID string, fn func(Prefs) (Prefs, error)) (Prefs, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.users[userID] {
		return Prefs{}, ErrNotFound
	}
	current, ok := s.rows[userID]
	if !ok {
		current = Defaults()
	}
	next, err := fn(current)
	if err != nil {
		return Prefs{}, err
	}
	next.UpdatedAt = time.Now()
	s.rows[userID] = next
	return next, nil
}

func (s *memStore) Update(_ context.Context, userID string, patch Patch) (Prefs, error) {
	return s.locked(userID, patch.Apply)
}

func (s *memStore) SetWallpaperKey(_ context.Context, userID string, mode Mode, key string) (Prefs, string, error) {
	var previous string
	p, err := s.locked(userID, func(p Prefs) (Prefs, error) {
		previous = setKey(&p, mode, key)
		return p, nil
	})
	return p, previous, err
}

func (s *memStore) ClearWallpaperKey(_ context.Context, userID string, mode Mode) (Prefs, string, error) {
	var previous string
	p, err := s.locked(userID, func(p Prefs) (Prefs, error) {
		previous = setKey(&p, mode, "")
		return p, nil
	})
	return p, previous, err
}

type fixture struct {
	svc     *Service
	store   Store
	storage *storage.Local
	root    string
}

// newStore builds the Store under test; the PostgreSQL test swaps it.
var newStore = func(t *testing.T, users ...string) Store { return newMemStore(users...) }

func newFixture(t *testing.T) *fixture {
	t.Helper()
	root := t.TempDir()
	st, err := storage.NewLocal(root)
	if err != nil {
		t.Fatal(err)
	}
	store := newStore(t, userA, userB)
	svc, err := NewService(ServiceDeps{Store: store, Storage: st, Log: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{svc: svc, store: store, storage: st, root: root}
}
