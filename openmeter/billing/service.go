package billing

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

type Service interface {
	ProfileService
	CustomerOverrideService
	InvoiceLineService
	SplitLineGroupService
	InvoiceService
	SequenceService
	LockableService

	InvoiceAppService

	ConfigService
}

type ProfileService interface {
	CreateProfile(ctx context.Context, param CreateProfileInput) (*Profile, error)
	GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*Profile, error)
	GetProfile(ctx context.Context, input GetProfileInput) (*Profile, error)
	ListProfiles(ctx context.Context, input ListProfilesInput) (ListProfilesResult, error)
	DeleteProfile(ctx context.Context, input DeleteProfileInput) error
	UpdateProfile(ctx context.Context, input UpdateProfileInput) (*Profile, error)
	ProvisionDefaultBillingProfile(ctx context.Context, namespace string) error
	IsAppUsed(ctx context.Context, appID app.AppID) error
}

type CustomerOverrideService interface {
	UpsertCustomerOverride(ctx context.Context, input UpsertCustomerOverrideInput) (CustomerOverrideWithDetails, error)
	DeleteCustomerOverride(ctx context.Context, input DeleteCustomerOverrideInput) error

	GetCustomerOverride(ctx context.Context, input GetCustomerOverrideInput) (CustomerOverrideWithDetails, error)
	GetCustomerApp(ctx context.Context, input GetCustomerAppInput) (app.App, error)
	ListCustomerOverrides(ctx context.Context, input ListCustomerOverridesInput) (ListCustomerOverridesResult, error)
}

type InvoiceLineService interface {
	// CreatePendingInvoiceLines creates pending invoice lines for a customer, if the lines are zero valued, the response is nil
	CreatePendingInvoiceLines(ctx context.Context, input CreatePendingInvoiceLinesInput) (*CreatePendingInvoiceLinesResult, error)
	GetLinesForSubscription(ctx context.Context, input GetLinesForSubscriptionInput) ([]LineOrHierarchy, error)
	// SnapshotLineQuantity returns an updated line with the quantity snapshoted from meters
	// the invoice is used as contextual information to the call.
	SnapshotLineQuantity(ctx context.Context, input SnapshotLineQuantityInput) (*Line, error)
}

type SplitLineGroupService interface {
	DeleteSplitLineGroup(ctx context.Context, input DeleteSplitLineGroupInput) error
	UpdateSplitLineGroup(ctx context.Context, input UpdateSplitLineGroupInput) (SplitLineGroup, error)
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
	SnapshotQuantities(ctx context.Context, input SnapshotQuantitiesInput) (Invoice, error)
	ApproveInvoice(ctx context.Context, input ApproveInvoiceInput) (Invoice, error)
	RetryInvoice(ctx context.Context, input RetryInvoiceInput) (Invoice, error)
	DeleteInvoice(ctx context.Context, input DeleteInvoiceInput) (Invoice, error)
	// UpdateInvoice updates an invoice as a whole
	UpdateInvoice(ctx context.Context, input UpdateInvoiceInput) (Invoice, error)

	// SimulateInvoice generates an invoice based on the provided input, but does not persist it
	// can be used to execute the invoice generation logic without actually creating an invoice in the database
	SimulateInvoice(ctx context.Context, input SimulateInvoiceInput) (Invoice, error)
	// UpsertValidationIssues upserts validation errors to the invoice bypassing the state machine, can only be
	// used on invoices in immutable state.
	UpsertValidationIssues(ctx context.Context, input UpsertValidationIssuesInput) error

	// RecalculateGatheringInvoices recalculates the gathering invoices for a given customer, updating the
	// collection_at attribute and deleting the gathering invoice if it has no lines.
	RecalculateGatheringInvoices(ctx context.Context, input RecalculateGatheringInvoicesInput) error
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

	// Async sync support
	SyncDraftInvoice(ctx context.Context, input SyncDraftInvoiceInput) (Invoice, error)
	SyncIssuingInvoice(ctx context.Context, input SyncIssuingInvoiceInput) (Invoice, error)
}

type LockableService interface {
	WithLock(ctx context.Context, customerID customer.CustomerID, fn func(ctx context.Context) error) error
}

type ConfigService interface {
	GetAdvancementStrategy() AdvancementStrategy
	WithAdvancementStrategy(strategy AdvancementStrategy) Service
	WithLockedNamespaces(namespaces []string) Service
}
