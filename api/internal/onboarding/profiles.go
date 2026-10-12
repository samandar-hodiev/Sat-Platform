package onboarding

import (
	"context"

	"github.com/samandar-hodiev/sat-platform/api/internal/profile"
)

// ProfileService adapts the profile service to Profiles. Its errors (the account is gone) are
// API errors and pass through unchanged.
type ProfileService struct{ Profile *profile.Service }

func (p ProfileService) Status(ctx context.Context, userID string) (ProfileStatus, error) {
	v, err := p.Profile.Get(ctx, userID)
	if err != nil {
		return ProfileStatus{}, err
	}
	return ProfileStatus{Completed: v.CompletedAt != nil, Timezone: v.Timezone}, nil
}
