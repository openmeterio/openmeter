package billing

import (
	"context"
)

type Service interface {
	ProfileService
	CustomerOverrideService
	InvoiceLineService
	InvoiceService
}

type ProfileService interface {
	CreateProfile(ctx context.Context, param CreateProfileInput) (*Profile, error)
	GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*Profile, error)
	GetProfile(ctx context.Context, input GetProfileInput) (*Profile, error)
	ListProfiles(ctx context.Context, input ListProfilesInput) (ListProfilesResult, error)
	DeleteProfile(ctx context.Context, input DeleteProfileInput) error
	UpdateProfile(ctx context.Context, input UpdateProfileInput) (*Profile, error)
	ProvisionDefaultBillingProfile(ctx context.Context, namespace string) error
}

type CustomerOverrideService interface {
	CreateCustomerOverride(ctx context.Context, input CreateCustomerOverrideInput) (*CustomerOverride, error)
	UpdateCustomerOverride(ctx context.Context, input UpdateCustomerOverrideInput) (*CustomerOverride, error)
	GetCustomerOverride(ctx context.Context, input GetCustomerOverrideInput) (*CustomerOverride, error)
	DeleteCustomerOverride(ctx context.Context, input DeleteCustomerOverrideInput) error

	GetProfileWithCustomerOverride(ctx context.Context, input GetProfileWithCustomerOverrideInput) (*ProfileWithCustomerDetails, error)
}

type InvoiceLineService interface {
	CreatePendingInvoiceLines(ctx context.Context, input CreateInvoiceLinesInput) ([]*Line, error)
	GetInvoiceLine(ctx context.Context, input GetInvoiceLineInput) (*Line, error)
	GetLinesForSubscription(ctx context.Context, input GetLinesForSubscriptionInput) ([]*Line, error)
	UpdateInvoiceLine(ctx context.Context, input UpdateInvoiceLineInput) (*Line, error)

	DeleteInvoiceLine(ctx context.Context, input DeleteInvoiceLineInput) error
}

type InvoiceService interface {
	ListInvoices(ctx context.Context, input ListInvoicesInput) (ListInvoicesResponse, error)
	GetInvoiceByID(ctx context.Context, input GetInvoiceByIdInput) (Invoice, error)
	InvoicePendingLines(ctx context.Context, input InvoicePendingLinesInput) ([]Invoice, error)
	// AdvanceInvoice advances the invoice to the next stage, the advancement is stopped until:
	// - an error is occurred
	// - the invoice is in a state that cannot be advanced (e.g. waiting for draft period to expire)
	// - the invoice is advanced to the final state
	AdvanceInvoice(ctx context.Context, input AdvanceInvoiceInput) (Invoice, error)
	ApproveInvoice(ctx context.Context, input ApproveInvoiceInput) (Invoice, error)
	RetryInvoice(ctx context.Context, input RetryInvoiceInput) (Invoice, error)
	DeleteInvoice(ctx context.Context, input DeleteInvoiceInput) error
	// UpdateInvoice updates an invoice as a whole
	UpdateInvoice(ctx context.Context, input UpdateInvoiceInput) (Invoice, error)

	// UpdateInvoiceLinesInternal updates the specified invoice lines and ensures that invoice states are properly syncronized
	// This method is intended to be used by OpenMeter internal services only, as it allows for updating invoice line values,
	// that are not allowed to be updated by external services.
	//
	// The call also ensures that the invoice's state is properly updated and invoice immutability is also considered.
	UpdateInvoiceLinesInternal(ctx context.Context, input UpdateInvoiceLinesInternalInput) error
}
