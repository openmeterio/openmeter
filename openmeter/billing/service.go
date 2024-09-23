package billing

import "context"

type Service interface {
	ProfileService
}

type ProfileService interface {
	CreateProfile(ctx context.Context, param CreateProfileInput) (*Profile, error)
}
