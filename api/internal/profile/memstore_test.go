package profile

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/storage"
)

// memStore is the in-memory Store for unit tests. The PostgreSQL store runs the same
// scenarios in repository_integration_test.go.
type memStore struct {
	mu       sync.Mutex
	users    map[string]bool
	profiles map[string]Profile
}

func newMemStore(users ...string) *memStore {
	s := &memStore{users: map[string]bool{}, profiles: map[string]Profile{}}
	for _, u := range users {
		s.users[u] = true
	}
	return s
}

func (s *memStore) Get(_ context.Context, userID string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.profiles[userID]
	if !ok {
		return Profile{}, ErrNotFound
	}
	return p, nil
}

func (s *memStore) row(userID string) (Profile, bool) {
	if !s.users[userID] {
		return Profile{}, false
	}
	p, ok := s.profiles[userID]
	if !ok {
		p = Profile{UserID: userID, Timezone: "UTC"}
	}
	return p, true
}

func (s *memStore) CompleteSetup(_ context.Context, userID string, in Setup, now time.Time, also func(pgx.Tx) error) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.row(userID)
	if !ok {
		return Profile{}, ErrNotFound
	}
	if p.CompletedAt != nil {
		return Profile{}, ErrAlreadyCompleted
	}
	if also != nil {
		if err := also(nil); err != nil {
			return Profile{}, err // rolled back: nothing written
		}
	}
	dob := in.DateOfBirth
	p.FirstName, p.LastName, p.Phone, p.DateOfBirth = in.FirstName, in.LastName, in.Phone, &dob
	if in.Timezone != "" {
		p.Timezone = in.Timezone
	}
	p.CompletedAt, p.UpdatedAt = &now, now
	s.profiles[userID] = p
	return p, nil
}

func (s *memStore) Prefill(_ context.Context, userID, first, last string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.row(userID)
	if !ok {
		return ErrNotFound
	}
	if p.CompletedAt != nil {
		return nil
	}
	if p.FirstName == "" {
		p.FirstName = first
	}
	if p.LastName == "" {
		p.LastName = last
	}
	s.profiles[userID] = p
	return nil
}

func (s *memStore) swap(userID, key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.row(userID)
	if !ok {
		return "", ErrNotFound
	}
	prev := p.AvatarKey
	p.AvatarKey = key
	s.profiles[userID] = p
	return prev, nil
}

func (s *memStore) SetAvatar(_ context.Context, userID, key string) (string, error) {
	return s.swap(userID, key)
}

func (s *memStore) ClearAvatar(_ context.Context, userID string) (string, error) {
	return s.swap(userID, "")
}

// fakeAccounts stands in for the auth module.
type fakeAccounts struct {
	mu       sync.Mutex
	accounts map[string]*Account
	password map[string]string
	failNext error
}

func newFakeAccounts() *fakeAccounts {
	return &fakeAccounts{accounts: map[string]*Account{}, password: map[string]string{}}
}

func (f *fakeAccounts) add(userID string, a Account) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.accounts[userID] = &a
}

func (f *fakeAccounts) Account(_ context.Context, userID string) (Account, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.accounts[userID]
	if !ok {
		return Account{}, ErrNotFound
	}
	return *a, nil
}

func (f *fakeAccounts) SetInitialPasswordTx(_ context.Context, _ pgx.Tx, userID, password string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.failNext; err != nil {
		f.failNext = nil
		return err
	}
	if len(password) < 8 {
		return apperr.New(apperr.CodeValidation, "Some fields are invalid").
			WithDetails(map[string]any{"fields": map[string]string{"password": "Use at least 8 characters."}})
	}
	a := f.accounts[userID]
	if a.HasPassword {
		return apperr.Conflict("A password is already set for this account.")
	}
	a.HasPassword = true
	f.password[userID] = password
	return nil
}

const (
	userEmail  = "11111111-1111-4111-8111-111111111111"
	userGoogle = "22222222-2222-4222-8222-222222222222"
	userPwd    = "33333333-3333-4333-8333-333333333333"
)

var fixedNow = time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)

type fixture struct {
	svc      *Service
	store    Store
	accounts *fakeAccounts
	storage  *storage.Local
	root     string
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
	accounts := newFakeAccounts()
	accounts.add(userEmail, Account{Email: "email@example.com"})
	accounts.add(userGoogle, Account{Email: "google@example.com", HasGoogle: true})
	accounts.add(userPwd, Account{Email: "pwd@example.com", HasPassword: true})
	store := newStore(t, userEmail, userGoogle, userPwd)
	svc, err := NewService(ServiceDeps{
		Store: store, Accounts: accounts, Storage: st,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Now: func() time.Time { return fixedNow },
	})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{svc: svc, store: store, accounts: accounts, storage: st, root: root}
}
