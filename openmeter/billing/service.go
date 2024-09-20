package billing

import "context"

type Service interface {
	ProfileService
}

type ProfileService interface {
	CreateProfile(ctx context.Context, param CreateProfileInput) (*Profile, error)
	GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*Profile, error)
	GetProfileByKeyOrID(ctx context.Context, input GetProfileByKeyOrIDInput) (*Profile, error)
	DeleteProfileByKeyOrID(ctx context.Context, input DeleteProfileByKeyOrIDInput) error
	UpdateProfile(ctx context.Context, input UpdateProfileInput) (*Profile, error)
}
