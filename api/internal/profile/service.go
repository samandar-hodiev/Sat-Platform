package profile

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/samandar-hodiev/sat-platform/api/internal/apperr"
	"github.com/samandar-hodiev/sat-platform/api/internal/platform/storage"
)

// View is the profile as the API returns it.
type View struct {
	Email       string
	FirstName   string
	LastName    string
	Phone       string
	DateOfBirth *time.Time
	AvatarKey   string
	Timezone    string
	HasPassword bool
	// PasswordRequired: Profile Setup must set a password (email accounts without one).
	PasswordRequired bool
	CompletedAt      *time.Time
}

// SetupInput is the raw Profile Setup form.
type SetupInput struct {
	FirstName   string
	LastName    string
	Phone       string
	DateOfBirth string
	Timezone    string
	Password    string
}

type ServiceDeps struct {
	Store    Store
	Accounts Accounts
	Storage  storage.ObjectStorage
	Log      *slog.Logger
	Now      func() time.Time
}

type Service struct {
	store    Store
	accounts Accounts
	storage  storage.ObjectStorage
	log      *slog.Logger
	now      func() time.Time
}

func NewService(d ServiceDeps) (*Service, error) {
	if d.Store == nil || d.Accounts == nil || d.Storage == nil || d.Log == nil {
		return nil, errors.New("profile: Store, Accounts, Storage and Log are required")
	}
	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Service{store: d.Store, accounts: d.Accounts, storage: d.Storage, log: d.Log, now: now}, nil
}

var errAccountGone = apperr.Unauthorized("Sign in to continue.")

func (s *Service) account(ctx context.Context, userID string) (Account, error) {
	acc, err := s.accounts.Account(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return Account{}, errAccountGone
	}
	return acc, err
}

// Get returns the learner's profile. A learner without a profile row yet (fresh email sign-up)
// gets an empty, incomplete profile.
func (s *Service) Get(ctx context.Context, userID string) (View, error) {
	acc, err := s.account(ctx, userID)
	if err != nil {
		return View{}, err
	}
	p, err := s.store.Get(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		p, err = Profile{Timezone: "UTC"}, nil
	}
	if err != nil {
		return View{}, err
	}
	return view(acc, p), nil
}

func view(acc Account, p Profile) View {
	return View{
		Email: acc.Email, FirstName: p.FirstName, LastName: p.LastName, Phone: p.Phone,
		DateOfBirth: p.DateOfBirth, AvatarKey: p.AvatarKey, Timezone: p.Timezone,
		HasPassword: acc.HasPassword, PasswordRequired: !acc.HasPassword && !acc.HasGoogle,
		CompletedAt: p.CompletedAt,
	}
}

// CompleteSetup validates the Profile Setup form, sets the first password when one is given,
// and marks the profile complete. Validation errors come back all at once, per field.
func (s *Service) CompleteSetup(ctx context.Context, userID string, in SetupInput) (View, error) {
	acc, err := s.account(ctx, userID)
	if err != nil {
		return View{}, err
	}
	current, err := s.store.Get(ctx, userID)
	if err == nil && current.CompletedAt != nil {
		return View{}, errSetupDone
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return View{}, err
	}

	fields := map[string]string{}
	var setup Setup
	var msg string
	if setup.FirstName, msg = normalizeName(in.FirstName); msg != "" {
		fields["first_name"] = msg
	}
	if setup.LastName, msg = normalizeName(in.LastName); msg != "" {
		fields["last_name"] = msg
	}
	if setup.Phone, msg = normalizePhone(in.Phone); msg != "" {
		fields["phone"] = msg
	}
	if setup.DateOfBirth, msg = parseDateOfBirth(in.DateOfBirth, s.now().UTC()); msg != "" {
		fields["date_of_birth"] = msg
	}
	if setup.Timezone, msg = normalizeTimezone(in.Timezone); msg != "" {
		fields["timezone"] = msg
	}
	switch {
	case acc.HasPassword && in.Password != "":
		fields["password"] = "a password is already set for this account"
	case !acc.HasPassword && !acc.HasGoogle && in.Password == "":
		fields["password"] = "is required"
	}
	if len(fields) > 0 {
		return View{}, validation(fields)
	}

	var setPassword func(pgx.Tx) error
	if in.Password != "" {
		setPassword = func(tx pgx.Tx) error {
			return s.accounts.SetInitialPasswordTx(ctx, tx, userID, in.Password)
		}
	}

	p, err := s.store.CompleteSetup(ctx, userID, setup, s.now(), setPassword)
	switch {
	case errors.Is(err, ErrAlreadyCompleted):
		return View{}, errSetupDone
	case errors.Is(err, ErrNotFound):
		return View{}, errAccountGone
	case err != nil:
		return View{}, err
	}
	if setPassword != nil {
		acc.HasPassword = true
	}
	return view(acc, p), nil
}

var errSetupDone = apperr.Conflict("Your profile is already set up. You can change it in My Profile.").
	WithDetails(map[string]any{"reason": "profile_already_completed"})

func validation(fields map[string]string) error {
	return apperr.New(apperr.CodeValidation, "Some fields are invalid").
		WithDetails(map[string]any{"fields": fields})
}

// UploadAvatar validates and re-encodes the image, stores it under a fresh key and points the
// profile at it. The previous avatar file is deleted afterwards.
func (s *Service) UploadAvatar(ctx context.Context, userID string, data []byte) (View, error) {
	acc, err := s.account(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if len(data) > AvatarMaxBytes {
		return View{}, avatarError("The image is larger than 5 MB.")
	}
	img, err := processAvatar(data)
	switch {
	case errors.Is(err, errAvatarType):
		return View{}, avatarError("Upload a JPG, PNG or WebP image.")
	case errors.Is(err, errAvatarCorrupt):
		return View{}, avatarError("This image can't be read. Try another file.")
	case errors.Is(err, errAvatarTooBig):
		return View{}, avatarError("This image is too large. Use one under 40 megapixels.")
	case err != nil:
		return View{}, err
	}

	key, err := newAvatarKey(userID)
	if err != nil {
		return View{}, err
	}
	if _, err := s.storage.Put(ctx, key, bytes.NewReader(img), AvatarContentType); err != nil {
		return View{}, err
	}
	previous, err := s.store.SetAvatar(ctx, userID, key)
	if err != nil {
		s.deleteFile(ctx, key)
		if errors.Is(err, ErrNotFound) {
			return View{}, errAccountGone
		}
		return View{}, err
	}
	s.deleteFile(ctx, previous)
	return s.viewAfterAvatar(ctx, userID, acc)
}

// RemoveAvatar switches back to the default avatar.
func (s *Service) RemoveAvatar(ctx context.Context, userID string) (View, error) {
	acc, err := s.account(ctx, userID)
	if err != nil {
		return View{}, err
	}
	previous, err := s.store.ClearAvatar(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return View{}, errAccountGone
	}
	if err != nil {
		return View{}, err
	}
	s.deleteFile(ctx, previous)
	return s.viewAfterAvatar(ctx, userID, acc)
}

func (s *Service) viewAfterAvatar(ctx context.Context, userID string, acc Account) (View, error) {
	p, err := s.store.Get(ctx, userID)
	if err != nil {
		return View{}, err
	}
	return view(acc, p), nil
}

// deleteFile removes a replaced avatar. A failure leaves an orphan file, not a broken
// profile, so it is logged and not returned.
func (s *Service) deleteFile(ctx context.Context, key string) {
	if key == "" {
		return
	}
	if err := s.storage.Delete(ctx, key); err != nil {
		s.log.WarnContext(ctx, "avatar file not deleted", slog.String("key", key), slog.String("error", err.Error()))
	}
}

func avatarError(message string) error {
	return validation(map[string]string{"file": message})
}

// Avatar keys: avatars/<user-id>/<32 random hex>.jpg. The random part makes every upload a new
// URL (safe to cache forever) and the URL unguessable.
func newAvatarKey(userID string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	key := "avatars/" + userID + "/" + hex.EncodeToString(b) + ".jpg"
	if err := storage.ValidateKey(key); err != nil {
		return "", err
	}
	return key, nil
}

// AvatarFile is an open stored avatar; the caller closes Body.
type AvatarFile struct {
	Body io.ReadCloser
	storage.Object
}

// OpenAvatar returns a stored avatar for the public media route.
func (s *Service) OpenAvatar(ctx context.Context, key string) (AvatarFile, error) {
	rc, obj, err := s.storage.Open(ctx, key)
	if errors.Is(err, storage.ErrNotFound) || errors.Is(err, storage.ErrInvalidKey) {
		return AvatarFile{}, apperr.NotFound("Image not found.")
	}
	if err != nil {
		return AvatarFile{}, err
	}
	return AvatarFile{Body: rc, Object: obj}, nil
}
