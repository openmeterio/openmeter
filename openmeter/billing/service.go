package billing

import "context"

type Service interface {
	ProfileService
	CustomerOverrideService
}

type ProfileService interface {
	CreateProfile(ctx context.Context, param CreateProfileInput) (*Profile, error)
	GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*Profile, error)
	GetProfile(ctx context.Context, input GetProfileInput) (*Profile, error)
	DeleteProfile(ctx context.Context, input DeleteProfileInput) error
	UpdateProfile(ctx context.Context, input UpdateProfileInput) (*Profile, error)
}

type CustomerOverrideService interface {
	CreateCustomerOverride(ctx context.Context, input CreateCustomerOverrideInput) (*CustomerOverride, error)
	UpdateCustomerOverride(ctx context.Context, input UpdateCustomerOverrideInput) (*CustomerOverride, error)
	GetCustomerOverride(ctx context.Context, input GetCustomerOverrideInput) (*CustomerOverride, error)
	DeleteCustomerOverride(ctx context.Context, input DeleteCustomerOverrideInput) error

	GetProfileWithCustomerOverride(ctx context.Context, input GetProfileWithCustomerOverrideInput) (*ProfileWithCustomerDetails, error)
}
