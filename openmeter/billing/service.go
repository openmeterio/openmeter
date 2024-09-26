package billing

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/customer"
)

type Service interface {
	ProfileService
	CustomerOverrideService
	InvoiceItemService
	InvoiceService
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

type InvoiceItemService interface {
	CreateInvoiceItems(ctx context.Context, input CreateInvoiceItemsInput) ([]InvoiceItem, error)
}

type InvoiceService interface {
	// GetPendingInvoiceItems returns all pending invoice items for a customer
	// The call can return any number of invoices based on multiple factors:
	// - The customer has multiple currencies (e.g. USD and EUR)
	// - [later] The provider can also mandate separate invoices if needed
	GetPendingInvoiceItems(ctx context.Context, customerID customer.CustomerID) ([]InvoiceWithValidation, error)
}
