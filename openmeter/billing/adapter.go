package billing

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ProfileAdapter
	CustomerBillingProfileAdapter
	InvoiceLineAdapter
	InvoiceAdapter
	GatheringInvoiceAdapter
	StandardInvoiceAdapter
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

type CustomerBillingProfileAdapter interface {
	SetCustomerBillingProfile(ctx context.Context, input SetCustomerBillingProfileInput) (CustomerBillingProfileAdapterResult, error)
	GetCustomerBillingProfile(ctx context.Context, input GetCustomerBillingProfileInput) (CustomerBillingProfileAdapterResult, error)
	ClearCustomerBillingProfile(ctx context.Context, input ClearCustomerBillingProfileInput) error
	ListCustomerBillingProfiles(ctx context.Context, input ListCustomerBillingProfilesInput) (ListCustomerBillingProfilesAdapterResult, error)
	BulkAssignCustomersToProfile(ctx context.Context, input BulkAssignCustomersToProfileInput) error
	GetCustomersReferencingProfile(ctx context.Context, input ProfileID) ([]customer.CustomerID, error)
}

type CustomerSynchronizationAdapter interface {
	// UpsertCustomerLock creates the customer synchronization row outside the transactional context.
	UpsertCustomerLock(ctx context.Context, input UpsertCustomerLockAdapterInput) error
	LockCustomerForUpdate(ctx context.Context, input LockCustomerForUpdateAdapterInput) error
}

type InvoiceLineAdapter interface {
	UpsertInvoiceLines(ctx context.Context, input UpsertInvoiceLinesAdapterInput) ([]*StandardLine, error)
	ListInvoiceLines(ctx context.Context, input ListInvoiceLinesAdapterInput) ([]*StandardLine, error)
}

type InvoiceAdapter interface {
	CreateInvoice(ctx context.Context, input CreateInvoiceAdapterInput) (CreateInvoiceAdapterRespone, error)
	DeleteGatheringInvoices(ctx context.Context, input DeleteGatheringInvoicesInput) error
	ListInvoices(ctx context.Context, input ListInvoicesAdapterInput) (ListInvoicesResponse, error)
	AssociatedLineCounts(ctx context.Context, input AssociatedLineCountsAdapterInput) (AssociatedLineCountsAdapterResponse, error)

	GetInvoiceOwnership(ctx context.Context, input GetInvoiceOwnershipAdapterInput) (GetOwnershipAdapterResponse, error)

	GetInvoiceType(ctx context.Context, input GetInvoiceTypeAdapterInput) (InvoiceType, error)
}

type StandardInvoiceAdapter interface {
	GetStandardLinesForSubscription(ctx context.Context, input GetLinesForSubscriptionInput) (StandardLines, error)
	SetStandardLineSubscriptionReferenceByChargeID(ctx context.Context, input SetLineSubscriptionReferenceByChargeIDInput) error
	GetStandardInvoiceById(ctx context.Context, input GetStandardInvoiceByIdInput) (StandardInvoice, error)
	UpdateStandardInvoice(ctx context.Context, input UpdateStandardInvoiceAdapterInput) (StandardInvoice, error)
	ListStandardInvoicesPendingAdvancement(ctx context.Context, input ListStandardInvoicesPendingAdvancementInput) ([]InvoiceAdvancementCandidate, error)
	CountStandardInvoicesPendingAdvancement(ctx context.Context, input CountStandardInvoicesPendingAdvancementInput) (int64, error)
}

type GatheringInvoiceAdapter interface {
	GetGatheringLinesForSubscription(ctx context.Context, input GetLinesForSubscriptionInput) (GatheringLines, error)
	SetGatheringLineSubscriptionReferenceByChargeID(ctx context.Context, input SetLineSubscriptionReferenceByChargeIDInput) error
	CreateGatheringInvoice(ctx context.Context, input CreateGatheringInvoiceAdapterInput) (GatheringInvoice, error)
	UpdateGatheringInvoice(ctx context.Context, input UpdateGatheringInvoiceAdapterInput) error
	DeleteGatheringInvoice(ctx context.Context, input DeleteGatheringInvoiceAdapterInput) error
	GetGatheringInvoiceById(ctx context.Context, input GetGatheringInvoiceByIdInput) (GatheringInvoice, error)
	ListGatheringInvoices(ctx context.Context, input ListGatheringInvoicesInput) (pagination.Result[GatheringInvoice], error)
	ListCustomerIDsPendingCollection(ctx context.Context, input ListCustomerIDsPendingCollectionInput) ([]customer.CustomerID, error)

	HardDeleteGatheringInvoiceLines(ctx context.Context, invoiceID InvoiceID, lineIDs []string) error
}

type InvoiceAppAdapter interface {
	UpdateInvoiceFields(ctx context.Context, input UpdateInvoiceFieldsInput) error
}

// TODO: Remove the schema-level adapter once all data migrations are complete.
type SchemaLevelAdapter interface {
	// GetInvoiceDefaultSchemaLevel returns the current default schema level for invoices.
	GetInvoiceDefaultSchemaLevel(ctx context.Context) (int, error)
	// SetInvoiceDefaultSchemaLevel sets the current default schema level for invoices.
	SetInvoiceDefaultSchemaLevel(ctx context.Context, level int) error
}
