package billing

import (
	"context"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
)

type Service interface {
	ProfileService
	CustomerOverrideService
	InvoiceLineService
	InvoiceService
	SequenceService

	InvoiceAppService

	ConfigIntrospectionService
}

type ProfileService interface {
	CreateProfile(ctx context.Context, param CreateProfileInput) (*Profile, error)
	GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*Profile, error)
	GetProfile(ctx context.Context, input GetProfileInput) (*Profile, error)
	ListProfiles(ctx context.Context, input ListProfilesInput) (ListProfilesResult, error)
	DeleteProfile(ctx context.Context, input DeleteProfileInput) error
	UpdateProfile(ctx context.Context, input UpdateProfileInput) (*Profile, error)
	ProvisionDefaultBillingProfile(ctx context.Context, namespace string) error
	IsAppUsed(ctx context.Context, appID appentitybase.AppID) (bool, error)
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
	GetLinesForSubscription(ctx context.Context, input GetLinesForSubscriptionInput) ([]*Line, error)
	// SnapshotLineQuantity returns an updated line with the quantity snapshoted from meters
	// the invoice is used as contextual information to the call.
	SnapshotLineQuantity(ctx context.Context, input SnapshotLineQuantityInput) (*Line, error)
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

	// SimulateInvoice generates an invoice based on the provided input, but does not persist it
	// can be used to execute the invoice generation logic without actually creating an invoice in the database
	SimulateInvoice(ctx context.Context, input SimulateInvoiceInput) (Invoice, error)
	// UpsertValidationIssues upserts validation errors to the invoice bypassing the state machine, can only be
	// used on invoices in immutable state.
	UpsertValidationIssues(ctx context.Context, input UpsertValidationIssuesInput) error
}

type SequenceService interface {
	GenerateInvoiceSequenceNumber(ctx context.Context, in SequenceGenerationInput, def SequenceDefinition) (string, error)
}

type InvoiceAppService interface {
	// TriggerInvoice triggers the invoice state machine to start processing the invoice
	TriggerInvoice(ctx context.Context, input InvoiceTriggerServiceInput) error

	// UpdateInvoiceFields updates the fields of an invoice which are not managed by the state machine
	// These are usually metadata fields settable after the invoice has been finalized
	UpdateInvoiceFields(ctx context.Context, input UpdateInvoiceFieldsInput) error
}

type ConfigIntrospectionService interface {
	GetAdvancementStrategy() AdvancementStrategy
}
