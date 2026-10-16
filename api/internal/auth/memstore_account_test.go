package auth

import (
	"context"
	"strings"
	"time"
)

// In-memory side of the My Profile (2.10) store methods. A deleted user leaves t.d.users (auth
// never sees deleted accounts) and is kept in t.d.deleted for assertions.

type memActionCode struct {
	StoredActionCode
	UserID    string
	Purpose   string
	CreatedAt time.Time
}

func (t memTx) ActionCodeStats(_ context.Context, userID, purpose string, since time.Time) (*time.Time, int, error) {
	var last *time.Time
	n := 0
	for _, c := range t.d.actionCodes {
		if c.UserID != userID || c.Purpose != purpose {
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

func (t memTx) CreateActionCode(_ context.Context, r ActionCodeRecord) error {
	t.d.actionCodes = append(t.d.actionCodes, memActionCode{
		StoredActionCode: StoredActionCode{ID: t.id("action"), Email: r.Email, CodeState: CodeState{Hash: r.Hash, ExpiresAt: r.ExpiresAt}},
		UserID:           r.UserID, Purpose: r.Purpose, CreatedAt: r.CreatedAt,
	})
	return nil
}

func (t memTx) LatestActionCode(_ context.Context, userID, purpose string) (StoredActionCode, error) {
	for i := len(t.d.actionCodes) - 1; i >= 0; i-- {
		if c := t.d.actionCodes[i]; c.UserID == userID && c.Purpose == purpose && c.UsedAt == nil {
			return c.StoredActionCode, nil
		}
	}
	return StoredActionCode{}, ErrNotFound
}

func (t memTx) actionCode(id string, f func(*memActionCode)) error {
	for i := range t.d.actionCodes {
		if t.d.actionCodes[i].ID == id {
			f(&t.d.actionCodes[i])
			return nil
		}
	}
	return ErrNotFound
}

func (t memTx) IncrementActionCodeAttempts(_ context.Context, id string) error {
	return t.actionCode(id, func(c *memActionCode) { c.Attempts++ })
}

func (t memTx) MarkActionCodeUsed(_ context.Context, id string, at time.Time) error {
	return t.actionCode(id, func(c *memActionCode) { c.UsedAt = &at })
}

func (t memTx) ChangeEmail(_ context.Context, userID, email string, at time.Time) error {
	for id, u := range t.d.users {
		if u.Email == email && id != userID {
			return ErrEmailTaken
		}
	}
	return t.update(userID, func(u *User) { u.Email, u.EmailVerifiedAt = email, &at })
}

func (t memTx) Identity(_ context.Context, userID, provider string) (Identity, error) {
	for key, id := range t.d.identities {
		if p, _, _ := strings.Cut(key, "|"); p == provider && id == userID {
			return Identity{Provider: provider}, nil
		}
	}
	return Identity{}, ErrNotFound
}

func (t memTx) UnlinkIdentity(_ context.Context, userID, provider string) error {
	found := false
	for key, id := range t.d.identities {
		if p, _, _ := strings.Cut(key, "|"); p == provider && id == userID {
			delete(t.d.identities, key)
			found = true
		}
	}
	if !found {
		return ErrNotFound
	}
	return nil
}

func (t memTx) RevokeOtherSessions(ctx context.Context, userID, keep string, at time.Time) error {
	for id, s := range t.d.sessions {
		if s.UserID == userID && id != keep {
			_ = t.RevokeSession(ctx, id, at)
		}
	}
	return nil
}

func (t memTx) DeleteUser(ctx context.Context, userID string, at time.Time) error {
	u, ok := t.d.users[userID]
	if !ok {
		return ErrNotFound
	}
	delete(t.d.users, userID)
	u.Email, u.PasswordHash, u.EmailVerifiedAt = "deleted-"+userID+"@"+DeletedEmailDomain, "", nil
	t.d.deleted[userID] = u
	for key, id := range t.d.identities {
		if id == userID {
			delete(t.d.identities, key)
		}
	}
	for h, r := range t.d.resets {
		if r.UserID == userID {
			delete(t.d.resets, h)
		}
	}
	kept := t.d.actionCodes[:0]
	for _, c := range t.d.actionCodes {
		if c.UserID != userID {
			kept = append(kept, c)
		}
	}
	t.d.actionCodes = kept
	return t.RevokeUserSessions(ctx, userID, at)
}

// Non-transactional wrappers (Store embeds Tx).

func (s *memStore) ActionCodeStats(ctx context.Context, userID, purpose string, since time.Time) (last *time.Time, n int, err error) {
	err = s.tx(func(t memTx) error { last, n, err = t.ActionCodeStats(ctx, userID, purpose, since); return err })
	return
}
func (s *memStore) CreateActionCode(ctx context.Context, r ActionCodeRecord) error {
	return s.tx(func(t memTx) error { return t.CreateActionCode(ctx, r) })
}
func (s *memStore) LatestActionCode(ctx context.Context, userID, purpose string) (c StoredActionCode, err error) {
	err = s.tx(func(t memTx) error { c, err = t.LatestActionCode(ctx, userID, purpose); return err })
	return
}
func (s *memStore) IncrementActionCodeAttempts(ctx context.Context, id string) error {
	return s.tx(func(t memTx) error { return t.IncrementActionCodeAttempts(ctx, id) })
}
func (s *memStore) MarkActionCodeUsed(ctx context.Context, id string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.MarkActionCodeUsed(ctx, id, at) })
}
func (s *memStore) ChangeEmail(ctx context.Context, userID, email string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.ChangeEmail(ctx, userID, email, at) })
}
func (s *memStore) Identity(ctx context.Context, userID, provider string) (id Identity, err error) {
	err = s.tx(func(t memTx) error { id, err = t.Identity(ctx, userID, provider); return err })
	return
}
func (s *memStore) UnlinkIdentity(ctx context.Context, userID, provider string) error {
	return s.tx(func(t memTx) error { return t.UnlinkIdentity(ctx, userID, provider) })
}
func (s *memStore) RevokeOtherSessions(ctx context.Context, userID, keep string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.RevokeOtherSessions(ctx, userID, keep, at) })
}
func (s *memStore) DeleteUser(ctx context.Context, userID string, at time.Time) error {
	return s.tx(func(t memTx) error { return t.DeleteUser(ctx, userID, at) })
}
