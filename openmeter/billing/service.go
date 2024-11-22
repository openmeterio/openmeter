package billing

import (
	"context"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
)

type Service interface {
	ProfileService
	CustomerOverrideService
	InvoiceLineService
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

type InvoiceLineService interface {
	CreateInvoiceLines(ctx context.Context, input CreateInvoiceLinesInput) ([]*billingentity.Line, error)
	GetInvoiceLine(ctx context.Context, input GetInvoiceLineInput) (*billingentity.Line, error)

	ValidateLineOwnership(ctx context.Context, input ValidateLineOwnershipInput) error
}

type InvoiceService interface {
	ListInvoices(ctx context.Context, input ListInvoicesInput) (ListInvoicesResponse, error)
	GetInvoiceByID(ctx context.Context, input GetInvoiceByIdInput) (billingentity.Invoice, error)
	CreateInvoice(ctx context.Context, input CreateInvoiceInput) ([]billingentity.Invoice, error)
	// AdvanceInvoice advances the invoice to the next stage, the advancement is stopped until:
	// - an error is occurred
	// - the invoice is in a state that cannot be advanced (e.g. waiting for draft period to expire)
	// - the invoice is advanced to the final state
	AdvanceInvoice(ctx context.Context, input AdvanceInvoiceInput) (billingentity.Invoice, error)
	ApproveInvoice(ctx context.Context, input ApproveInvoiceInput) (billingentity.Invoice, error)
	RetryInvoice(ctx context.Context, input RetryInvoiceInput) (billingentity.Invoice, error)

	ValidateInvoiceOwnership(ctx context.Context, input ValidateInvoiceOwnershipInput) error
}
