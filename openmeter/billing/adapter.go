package billing

import (
	"context"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ProfileAdapter
	CustomerOverrideAdapter
	InvoiceLineAdapter
	InvoiceSplitLineGroupAdapter
	InvoiceAdapter
	GatheringInvoiceAdapter
	SequenceAdapter
	InvoiceAppAdapter
	CustomerSynchronizationAdapter
	SchemaLevelAdapter

	entutils.TxCreator
}

type ProfileAdapter interface {
	CreateProfile(ctx context.Context, input CreateProfileInput) (*BaseProfile, error)
	ListProfiles(ctx context.Context, input ListProfilesInput) (pagination.Result[BaseProfile], error)
	GetProfile(ctx context.Context, input GetProfileInput) (*AdapterGetProfileResponse, error)
	GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*AdapterGetProfileResponse, error)
	DeleteProfile(ctx context.Context, input DeleteProfileInput) error
	UpdateProfile(ctx context.Context, input UpdateProfileAdapterInput) (*BaseProfile, error)

	IsAppUsed(ctx context.Context, appID app.AppID) error

	GetUnpinnedCustomerIDsWithPaidSubscription(ctx context.Context, input GetUnpinnedCustomerIDsWithPaidSubscriptionInput) ([]customer.CustomerID, error)
}

type CustomerOverrideAdapter interface {
	CreateCustomerOverride(ctx context.Context, input UpdateCustomerOverrideAdapterInput) (*CustomerOverride, error)
	GetCustomerOverride(ctx context.Context, input GetCustomerOverrideAdapterInput) (*CustomerOverride, error)
	UpdateCustomerOverride(ctx context.Context, input UpdateCustomerOverrideAdapterInput) (*CustomerOverride, error)
	DeleteCustomerOverride(ctx context.Context, input DeleteCustomerOverrideInput) error
	ListCustomerOverrides(ctx context.Context, input ListCustomerOverridesInput) (ListCustomerOverridesAdapterResult, error)

	BulkAssignCustomersToProfile(ctx context.Context, input BulkAssignCustomersToProfileInput) error

	GetCustomerOverrideReferencingProfile(ctx context.Context, input HasCustomerOverrideReferencingProfileAdapterInput) ([]customer.CustomerID, error)
}

type CustomerSynchronizationAdapter interface {
	// UpsertCustomerOverride upserts a customer override ignoring the transactional context, the override
	// will be empty.
	UpsertCustomerLock(ctx context.Context, input UpsertCustomerLockAdapterInput) error
	LockCustomerForUpdate(ctx context.Context, input LockCustomerForUpdateAdapterInput) error
}

type InvoiceLineAdapter interface {
	UpsertInvoiceLines(ctx context.Context, input UpsertInvoiceLinesAdapterInput) ([]*StandardLine, error)
	ListInvoiceLines(ctx context.Context, input ListInvoiceLinesAdapterInput) ([]*StandardLine, error)
	GetLinesForSubscription(ctx context.Context, input GetLinesForSubscriptionInput) ([]LineOrHierarchy, error)
}

type InvoiceAdapter interface {
	CreateInvoice(ctx context.Context, input CreateInvoiceAdapterInput) (CreateInvoiceAdapterRespone, error)
	GetInvoiceById(ctx context.Context, input GetInvoiceByIdInput) (StandardInvoice, error)
	DeleteGatheringInvoices(ctx context.Context, input DeleteGatheringInvoicesInput) error
	ListInvoices(ctx context.Context, input ListInvoicesInput) (ListInvoicesResponse, error)
	AssociatedLineCounts(ctx context.Context, input AssociatedLineCountsAdapterInput) (AssociatedLineCountsAdapterResponse, error)
	UpdateInvoice(ctx context.Context, input UpdateInvoiceAdapterInput) (StandardInvoice, error)

	GetInvoiceOwnership(ctx context.Context, input GetInvoiceOwnershipAdapterInput) (GetOwnershipAdapterResponse, error)
}

type GatheringInvoiceAdapter interface {
	CreateGatheringInvoice(ctx context.Context, input CreateGatheringInvoiceAdapterInput) (GatheringInvoice, error)
	UpdateGatheringInvoice(ctx context.Context, input UpdateGatheringInvoiceAdapterInput) (GatheringInvoice, error)
	DeleteGatheringInvoice(ctx context.Context, input DeleteGatheringInvoiceAdapterInput) error
	ListGatheringInvoices(ctx context.Context, input ListGatheringInvoicesInput) (pagination.Result[GatheringInvoice], error)
}

type InvoiceSplitLineGroupAdapter interface {
	CreateSplitLineGroup(ctx context.Context, input CreateSplitLineGroupAdapterInput) (SplitLineGroup, error)
	UpdateSplitLineGroup(ctx context.Context, input UpdateSplitLineGroupInput) (SplitLineGroup, error)
	DeleteSplitLineGroup(ctx context.Context, input DeleteSplitLineGroupInput) error
	GetSplitLineGroup(ctx context.Context, input GetSplitLineGroupInput) (SplitLineHierarchy, error)
}

type SequenceAdapter interface {
	NextSequenceNumber(ctx context.Context, input NextSequenceNumberInput) (alpacadecimal.Decimal, error)
}

type InvoiceAppAdapter interface {
	UpdateInvoiceFields(ctx context.Context, input UpdateInvoiceFieldsInput) error
}

type SchemaLevelAdapter interface {
	// GetInvoiceDefaultSchemaLevel returns the current default schema level for invoices.
	GetInvoiceDefaultSchemaLevel(ctx context.Context) (int, error)
	// SetInvoiceDefaultSchemaLevel sets the current default schema level for invoices.
	SetInvoiceDefaultSchemaLevel(ctx context.Context, level int) error
}
