package auth

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// memStore is an in-memory Store for tests only. InTx works on a copy and commits it only when
// fn succeeds, so tests see real rollback behaviour. Production uses PostgreSQL (repository.go).
type memStore struct {
	mu sync.Mutex
	d  *memData
}

type memData struct {
	seq        int
	users      map[string]User
	lastLogin  map[string]time.Time
	identities map[string]string // provider|subject → user id
	codes      []memCode
	sessions   map[string]Session
	refresh    map[string]RefreshToken
	resets     map[string]memReset
}

type memCode struct {
	StoredCode
	Email     string
	CreatedAt time.Time
}

type memReset struct {
	ResetToken
	CreatedAt time.Time
}

func newMemStore() *memStore {
	return &memStore{d: &memData{
		users: map[string]User{}, lastLogin: map[string]time.Time{}, identities: map[string]string{},
		sessions: map[string]Session{}, refresh: map[string]RefreshToken{}, resets: map[string]memReset{},
	}}
}

func (d *memData) clone() *memData {
	c := *d
	c.users = cloneMap(d.users)
	c.lastLogin = cloneMap(d.lastLogin)
	c.identities = cloneMap(d.identities)
	c.codes = append([]memCode(nil), d.codes...)
	c.sessions = cloneMap(d.sessions)
	c.refresh = cloneMap(d.refresh)
	c.resets = cloneMap(d.resets)
	return &c
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

type memTx struct{ d *memData }

func (s *memStore) InTx(ctx context.Context, fn func(tx Tx) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	work := s.d.clone()
	if err := fn(memTx{work}); err != nil {
		return err
	}
	s.d = work
	return nil
}

// Store's own (non-transactional) methods run as single-statement transactions.
func (s *memStore) tx(fn func(memTx) error) error {
	return s.InTx(context.Background(), func(tx Tx) error { return fn(tx.(memTx)) })
}

func (s *memStore) UserByID(ctx context.Context, id string) (u User, err error) {
	err = s.tx(func(t memTx) error { u, err = t.UserByID(ctx, id); return err })
	return
}
func (s *memStore) UserByEmail(ctx context.Context, email string) (u User, err error) {
	err = s.tx(func(t memTx) error { u, err = t.UserByEmail(ctx, email); return err })
	return
}
func (s *memStore) CreateUser(ctx context.Context, n NewUser) (u User, err error) {
	err = s.tx(func(t memTx) error { u, err = t.CreateUser(ctx, n); return err })
	return
}
func (s *memStore) SetPassword(ctx context.Context, id, h string) error {
	return s.tx(func(t memTx) error { return t.SetPassword(ctx, id, h) })
}
func (s *memStore) SetPasswordIfUnset(ctx context.Context, id, h string) error {
	return s.tx(func(t memTx) error { return t.SetPasswordIfUnset(ctx, id, h) })
}
func (s *memStore) MarkEmailVerified(ctx context.Context, id string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.MarkEmailVerified(ctx, id, at) })
}
func (s *memStore) TouchLastLogin(ctx context.Context, id string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.TouchLastLogin(ctx, id, at) })
}
func (s *memStore) UserIDByIdentity(ctx context.Context, p, sub string) (id string, err error) {
	err = s.tx(func(t memTx) error { id, err = t.UserIDByIdentity(ctx, p, sub); return err })
	return
}
func (s *memStore) LinkIdentity(ctx context.Context, id, p, sub, email string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.LinkIdentity(ctx, id, p, sub, email, at) })
}
func (s *memStore) IdentityProviders(ctx context.Context, id string) (p []string, err error) {
	err = s.tx(func(t memTx) error { p, err = t.IdentityProviders(ctx, id); return err })
	return
}
func (s *memStore) CodeStats(ctx context.Context, email string, since time.Time) (last *time.Time, n int, err error) {
	err = s.tx(func(t memTx) error { last, n, err = t.CodeStats(ctx, email, since); return err })
	return
}
func (s *memStore) CreateCode(ctx context.Context, c CodeRecord) error {
	return s.tx(func(t memTx) error { return t.CreateCode(ctx, c) })
}
func (s *memStore) LatestCode(ctx context.Context, email string) (c StoredCode, err error) {
	err = s.tx(func(t memTx) error { c, err = t.LatestCode(ctx, email); return err })
	return
}
func (s *memStore) IncrementCodeAttempts(ctx context.Context, id string) error {
	return s.tx(func(t memTx) error { return t.IncrementCodeAttempts(ctx, id) })
}
func (s *memStore) MarkCodeUsed(ctx context.Context, id string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.MarkCodeUsed(ctx, id, at) })
}
func (s *memStore) CreateSession(ctx context.Context, n NewSession) (ss Session, err error) {
	err = s.tx(func(t memTx) error { ss, err = t.CreateSession(ctx, n); return err })
	return
}
func (s *memStore) SessionByID(ctx context.Context, id string) (ss Session, err error) {
	err = s.tx(func(t memTx) error { ss, err = t.SessionByID(ctx, id); return err })
	return
}
func (s *memStore) RefreshTokenByHash(ctx context.Context, h string) (r RefreshToken, err error) {
	err = s.tx(func(t memTx) error { r, err = t.RefreshTokenByHash(ctx, h); return err })
	return
}
func (s *memStore) AddRefreshToken(ctx context.Context, sid, h string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.AddRefreshToken(ctx, sid, h, at) })
}
func (s *memStore) MarkRefreshTokenUsed(ctx context.Context, h string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.MarkRefreshTokenUsed(ctx, h, at) })
}
func (s *memStore) TouchSession(ctx context.Context, id string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.TouchSession(ctx, id, at) })
}
func (s *memStore) RevokeSession(ctx context.Context, id string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.RevokeSession(ctx, id, at) })
}
func (s *memStore) RevokeUserSessions(ctx context.Context, id string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.RevokeUserSessions(ctx, id, at) })
}
func (s *memStore) CreateResetToken(ctx context.Context, id, h string, exp, at time.Time) error {
	return s.tx(func(t memTx) error { return t.CreateResetToken(ctx, id, h, exp, at) })
}
func (s *memStore) ResetTokenByHash(ctx context.Context, h string) (r ResetToken, err error) {
	err = s.tx(func(t memTx) error { r, err = t.ResetTokenByHash(ctx, h); return err })
	return
}
func (s *memStore) MarkResetTokenUsed(ctx context.Context, h string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.MarkResetTokenUsed(ctx, h, at) })
}

// --- memTx -----------------------------------------------------------------------------------

func (t memTx) id(prefix string) string {
	t.d.seq++
	return fmt.Sprintf("%s-%d", prefix, t.d.seq)
}

func (t memTx) UserByID(_ context.Context, id string) (User, error) {
	u, ok := t.d.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (t memTx) UserByEmail(_ context.Context, email string) (User, error) {
	for _, u := range t.d.users {
		if u.Email == email {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (t memTx) CreateUser(_ context.Context, n NewUser) (User, error) {
	for _, u := range t.d.users {
		if u.Email == n.Email {
			return User{}, fmt.Errorf("duplicate email %s", n.Email)
		}
	}
	u := User{ID: t.id("user"), Email: n.Email, Role: n.Role, Status: UserStatusActive, EmailVerifiedAt: n.EmailVerifiedAt}
	t.d.users[u.ID] = u
	return u, nil
}

func (t memTx) update(id string, f func(*User)) error {
	u, ok := t.d.users[id]
	if !ok {
		return ErrNotFound
	}
	f(&u)
	t.d.users[id] = u
	return nil
}

func (t memTx) SetPassword(_ context.Context, id, hash string) error {
	return t.update(id, func(u *User) { u.PasswordHash = hash })
}
func (t memTx) SetPasswordIfUnset(_ context.Context, id, hash string) error {
	u, ok := t.d.users[id]
	if !ok {
		return ErrNotFound
	}
	if u.PasswordHash != "" {
		return ErrPasswordAlreadySet
	}
	u.PasswordHash = hash
	t.d.users[id] = u
	return nil
}

func (t memTx) MarkEmailVerified(_ context.Context, id string, at time.Time) error {
	return t.update(id, func(u *User) { u.EmailVerifiedAt = &at })
}
func (t memTx) TouchLastLogin(_ context.Context, id string, at time.Time) error {
	t.d.lastLogin[id] = at
	return nil
}

func (t memTx) UserIDByIdentity(_ context.Context, provider, subject string) (string, error) {
	id, ok := t.d.identities[provider+"|"+subject]
	if !ok {
		return "", ErrNotFound
	}
	return id, nil
}

func (t memTx) LinkIdentity(_ context.Context, userID, provider, subject, _ string, _ time.Time) error {
	t.d.identities[provider+"|"+subject] = userID
	return nil
}

func (t memTx) IdentityProviders(_ context.Context, userID string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for key, id := range t.d.identities {
		provider, _, _ := strings.Cut(key, "|")
		if id == userID && !seen[provider] {
			seen[provider] = true
			out = append(out, provider)
		}
	}
	return out, nil
}

func (t memTx) CodeStats(_ context.Context, email string, since time.Time) (*time.Time, int, error) {
	var last *time.Time
	n := 0
	for _, c := range t.d.codes {
		if c.Email != email {
			continue
		}
		if !c.CreatedAt.Before(since) {
			n++
		}
		if last == nil || c.CreatedAt.After(*last) {
			at := c.CreatedAt
			last = &at
		}
	}
	return last, n, nil
}

func (t memTx) CreateCode(_ context.Context, n CodeRecord) error {
	t.d.codes = append(t.d.codes, memCode{
		StoredCode: StoredCode{ID: t.id("code"), CodeState: CodeState{Hash: n.Hash, ExpiresAt: n.ExpiresAt}},
		Email:      n.Email, CreatedAt: n.CreatedAt,
	})
	return nil
}

func (t memTx) LatestCode(_ context.Context, email string) (StoredCode, error) {
	for i := len(t.d.codes) - 1; i >= 0; i-- {
		if c := t.d.codes[i]; c.Email == email && c.UsedAt == nil {
			return c.StoredCode, nil
		}
	}
	return StoredCode{}, ErrNotFound
}

func (t memTx) code(id string, f func(*memCode)) error {
	for i := range t.d.codes {
		if t.d.codes[i].ID == id {
			f(&t.d.codes[i])
			return nil
		}
	}
	return ErrNotFound
}

func (t memTx) IncrementCodeAttempts(_ context.Context, id string) error {
	return t.code(id, func(c *memCode) { c.Attempts++ })
}
func (t memTx) MarkCodeUsed(_ context.Context, id string, at time.Time) error {
	return t.code(id, func(c *memCode) { c.UsedAt = &at })
}

func (t memTx) CreateSession(_ context.Context, n NewSession) (Session, error) {
	s := Session{ID: t.id("session"), UserID: n.UserID, ExpiresAt: n.ExpiresAt, Remember: n.Remember, LastUsedAt: n.CreatedAt}
	t.d.sessions[s.ID] = s
	return s, nil
}

func (t memTx) SessionByID(_ context.Context, id string) (Session, error) {
	s, ok := t.d.sessions[id]
	if !ok {
		return Session{}, ErrNotFound
	}
	return s, nil
}

func (t memTx) RefreshTokenByHash(_ context.Context, hash string) (RefreshToken, error) {
	r, ok := t.d.refresh[hash]
	if !ok {
		return RefreshToken{}, ErrNotFound
	}
	return r, nil
}

func (t memTx) AddRefreshToken(_ context.Context, sid, hash string, _ time.Time) error {
	t.d.refresh[hash] = RefreshToken{SessionID: sid}
	return nil
}

func (t memTx) MarkRefreshTokenUsed(_ context.Context, hash string, at time.Time) error {
	r := t.d.refresh[hash]
	r.UsedAt = &at
	t.d.refresh[hash] = r
	return nil
}

func (t memTx) TouchSession(_ context.Context, id string, at time.Time) error {
	s := t.d.sessions[id]
	s.LastUsedAt = at
	t.d.sessions[id] = s
	return nil
}

func (t memTx) RevokeSession(_ context.Context, id string, at time.Time) error {
	if s, ok := t.d.sessions[id]; ok && s.RevokedAt == nil {
		s.RevokedAt = &at
		t.d.sessions[id] = s
	}
	return nil
}

func (t memTx) RevokeUserSessions(ctx context.Context, userID string, at time.Time) error {
	for id, s := range t.d.sessions {
		if s.UserID == userID {
			_ = t.RevokeSession(ctx, id, at)
		}
	}
	return nil
}

func (t memTx) CreateResetToken(_ context.Context, userID, hash string, exp, at time.Time) error {
	for h, r := range t.d.resets {
		if r.UserID == userID && r.UsedAt == nil {
			r.UsedAt = &at // superseded
			t.d.resets[h] = r
		}
	}
	t.d.resets[hash] = memReset{ResetToken: ResetToken{UserID: userID, ExpiresAt: exp}, CreatedAt: at}
	return nil
}

func (t memTx) ResetTokenByHash(_ context.Context, hash string) (ResetToken, error) {
	r, ok := t.d.resets[hash]
	if !ok {
		return ResetToken{}, ErrNotFound
	}
	return r.ResetToken, nil
}

func (t memTx) MarkResetTokenUsed(_ context.Context, hash string, at time.Time) error {
	r := t.d.resets[hash]
	r.UsedAt = &at
	t.d.resets[hash] = r
	return nil
}
