package billing

import (
	"context"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

type Service interface {
	ProfileService
	CustomerOverrideService
	InvoiceItemService
	InvoiceService
}

type ProfileService interface {
	CreateProfile(ctx context.Context, param CreateProfileInput) (*billingentity.Profile, error)
	GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*billingentity.Profile, error)
	GetProfile(ctx context.Context, input GetProfileInput) (*billingentity.Profile, error)
	ListProfiles(ctx context.Context, input ListProfilesInput) (ListProfilesResult, error)
	DeleteProfile(ctx context.Context, input DeleteProfileInput) error
	UpdateProfile(ctx context.Context, input UpdateProfileInput) (*billingentity.Profile, error)
}

type CustomerOverrideService interface {
	CreateCustomerOverride(ctx context.Context, input CreateCustomerOverrideInput) (*billingentity.CustomerOverride, error)
	UpdateCustomerOverride(ctx context.Context, input UpdateCustomerOverrideInput) (*billingentity.CustomerOverride, error)
	GetCustomerOverride(ctx context.Context, input GetCustomerOverrideInput) (*billingentity.CustomerOverride, error)
	DeleteCustomerOverride(ctx context.Context, input DeleteCustomerOverrideInput) error

	GetProfileWithCustomerOverride(ctx context.Context, input GetProfileWithCustomerOverrideInput) (*billingentity.ProfileWithCustomerDetails, error)
}

type InvoiceItemService interface {
	CreateInvoiceItems(ctx context.Context, input CreateInvoiceItemsInput) ([]billingentity.InvoiceItem, error)
}

type InvoiceService interface {
	// GetPendingInvoiceItems returns all pending invoice items for a customer
	// The call can return any number of invoices based on multiple factors:
	// - The customer has multiple currencies (e.g. USD and EUR)
	// - [later] The provider can also mandate separate invoices if needed
	GetPendingInvoiceItems(ctx context.Context, customerID customerentity.CustomerID) ([]billingentity.InvoiceWithValidation, error)
}
