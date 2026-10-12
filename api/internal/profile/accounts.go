package profile

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/samandar-hodiev/sat-platform/api/internal/auth"
)

// AuthAccounts adapts the auth service to Accounts. Auth errors (session ended, password
// policy, password already set) are API errors and pass through unchanged.
type AuthAccounts struct{ Auth *auth.Service }

func (a AuthAccounts) Account(ctx context.Context, userID string) (Account, error) {
	acc, err := a.Auth.Account(ctx, userID)
	if err != nil {
		return Account{}, err
	}
	return Account{Email: acc.Email, HasPassword: acc.HasPassword(), HasGoogle: acc.HasGoogle}, nil
}

func (a AuthAccounts) SetInitialPasswordTx(ctx context.Context, tx pgx.Tx, userID, password string) error {
	return a.Auth.SetInitialPasswordTx(ctx, auth.PgxTx(tx), userID, password)
}

// GooglePrefiller implements auth.ProfilePrefiller: a new Google learner's name is filled in
// before Profile Setup (PRD §7). It needs only the store, so auth can be built before this
// module.
type GooglePrefiller struct{ Store Store }

// PrefillFromGoogle fills empty name fields. Names that do not pass the Profile Setup rules
// are skipped; the learner types them in.
func (g GooglePrefiller) PrefillFromGoogle(ctx context.Context, userID string, p auth.GoogleProfile) error {
	first, msg := normalizeName(p.FirstName)
	if msg != "" {
		first = ""
	}
	last, msg := normalizeName(p.LastName)
	if msg != "" {
		last = ""
	}
	if first == "" && last == "" {
		return nil
	}
	return g.Store.Prefill(ctx, userID, first, last)
}

var _ auth.ProfilePrefiller = GooglePrefiller{}
