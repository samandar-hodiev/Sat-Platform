package onboarding

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
)

// memStore is the in-memory Store for unit tests. The PostgreSQL store runs the same scenarios
// in repository_integration_test.go.
type memStore struct {
	mu      sync.Mutex
	users   map[string]bool
	records map[string]Record
}

func newMemStore(users ...string) *memStore {
	s := &memStore{users: map[string]bool{}, records: map[string]Record{}}
	for _, u := range users {
		s.users[u] = true
	}
	return s
}

func (s *memStore) Get(_ context.Context, userID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[userID]
	if !ok {
		return Record{}, ErrNotFound
	}
	return r, nil
}

func (s *memStore) write(userID string, now time.Time, f func(*Record)) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.users[userID] {
		return Record{}, ErrNotFound
	}
	r, ok := s.records[userID]
	if !ok {
		r = Record{UserID: userID}
	}
	f(&r)
	r.UpdatedAt = now
	s.records[userID] = r
	return r, nil
}

func (s *memStore) SaveGoal(_ context.Context, userID string, g Goal, now time.Time) (Record, error) {
	return s.write(userID, now, func(r *Record) { r.Goal, r.GoalSavedAt = g, &now })
}

func (s *memStore) SaveSchedule(_ context.Context, userID string, sc Schedule, now time.Time) (Record, error) {
	sc.StudyDays = slices.Clone(sc.StudyDays)
	return s.write(userID, now, func(r *Record) { r.Schedule, r.ScheduleSavedAt = sc, &now })
}

func (s *memStore) SaveBackground(_ context.Context, userID string, b Background, now time.Time) (Record, error) {
	return s.write(userID, now, func(r *Record) { r.Background, r.BackgroundSavedAt = b, &now })
}

func (s *memStore) Complete(_ context.Context, userID string, now time.Time) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.records[userID]
	if !ok {
		return Record{}, ErrNotFound
	}
	if len(r.Missing()) > 0 {
		return Record{}, ErrIncomplete
	}
	if r.CompletedAt == nil {
		r.CompletedAt, r.UpdatedAt = &now, now
		s.records[userID] = r
	}
	return r, nil
}

// Learners in the fixture: userReady finished Profile Setup (Tashkent, UTC+5), userNew has not.
const (
	userReady = "11111111-1111-4111-8111-111111111111"
	userNew   = "22222222-2222-4222-8222-222222222222"
	userGone  = "44444444-4444-4444-8444-444444444444"
)

// fixedNow is 12 Oct 2026, 20:30 UTC: already 13 Oct in Tashkent.
var fixedNow = time.Date(2026, time.October, 12, 20, 30, 0, 0, time.UTC)

type fakeProfiles map[string]ProfileStatus

func (f fakeProfiles) Status(_ context.Context, userID string) (ProfileStatus, error) {
	st, ok := f[userID]
	if !ok {
		return ProfileStatus{}, apperr.Unauthorized("Sign in to continue.")
	}
	return st, nil
}

// newStore builds the fixture's store; the integration test swaps in PostgreSQL.
var newStore = func(t *testing.T, users ...string) Store { return newMemStore(users...) }

type fixture struct {
	store Store
	svc   *Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	store := newStore(t, userReady, userNew)
	profiles := fakeProfiles{
		userReady: {Completed: true, Timezone: "Asia/Tashkent"},
		userNew:   {Completed: false, Timezone: "UTC"},
	}
	svc, err := NewService(ServiceDeps{Store: store, Profiles: profiles, Now: func() time.Time { return fixedNow }})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{store: store, svc: svc}
}
