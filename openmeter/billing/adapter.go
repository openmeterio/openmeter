package billing

import (
	"context"

	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ProfileAdapter
	CustomerOverrideAdapter
	InvoiceLineAdapter
	InvoiceAdapter

	entutils.TxCreator
}

type ProfileAdapter interface {
	CreateProfile(ctx context.Context, input CreateProfileInput) (*BaseProfile, error)
	ListProfiles(ctx context.Context, input ListProfilesInput) (pagination.PagedResponse[BaseProfile], error)
	GetProfile(ctx context.Context, input GetProfileInput) (*BaseProfile, error)
	GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*BaseProfile, error)
	DeleteProfile(ctx context.Context, input DeleteProfileInput) error
	UpdateProfile(ctx context.Context, input UpdateProfileAdapterInput) (*BaseProfile, error)
}

type CustomerOverrideAdapter interface {
	CreateCustomerOverride(ctx context.Context, input CreateCustomerOverrideInput) (*CustomerOverride, error)
	GetCustomerOverride(ctx context.Context, input GetCustomerOverrideAdapterInput) (*CustomerOverride, error)
	UpdateCustomerOverride(ctx context.Context, input UpdateCustomerOverrideAdapterInput) (*CustomerOverride, error)
	DeleteCustomerOverride(ctx context.Context, input DeleteCustomerOverrideInput) error

	// UpsertCustomerOverride upserts a customer override ignoring the transactional context, the override
	// will be empty.
	UpsertCustomerOverride(ctx context.Context, input UpsertCustomerOverrideAdapterInput) error
	LockCustomerForUpdate(ctx context.Context, input LockCustomerForUpdateAdapterInput) error

	GetCustomerOverrideReferencingProfile(ctx context.Context, input HasCustomerOverrideReferencingProfileAdapterInput) ([]customerentity.CustomerID, error)
}

type InvoiceLineAdapter interface {
	UpsertInvoiceLines(ctx context.Context, input UpsertInvoiceLinesAdapterInput) ([]*Line, error)
	ListInvoiceLines(ctx context.Context, input ListInvoiceLinesAdapterInput) ([]*Line, error)
	AssociateLinesToInvoice(ctx context.Context, input AssociateLinesToInvoiceAdapterInput) ([]*Line, error)
	GetInvoiceLine(ctx context.Context, input GetInvoiceLineAdapterInput) (*Line, error)
	GetLinesForSubscription(ctx context.Context, input GetLinesForSubscriptionInput) ([]*Line, error)

	GetInvoiceLineOwnership(ctx context.Context, input GetInvoiceLineOwnershipAdapterInput) (GetOwnershipAdapterResponse, error)
}

type InvoiceAdapter interface {
	CreateInvoice(ctx context.Context, input CreateInvoiceAdapterInput) (CreateInvoiceAdapterRespone, error)
	GetInvoiceById(ctx context.Context, input GetInvoiceByIdInput) (Invoice, error)
	LockInvoicesForUpdate(ctx context.Context, input LockInvoicesForUpdateInput) error
	DeleteInvoices(ctx context.Context, input DeleteInvoicesAdapterInput) error
	ListInvoices(ctx context.Context, input ListInvoicesInput) (ListInvoicesResponse, error)
	AssociatedLineCounts(ctx context.Context, input AssociatedLineCountsAdapterInput) (AssociatedLineCountsAdapterResponse, error)
	UpdateInvoice(ctx context.Context, input UpdateInvoiceAdapterInput) (Invoice, error)

	GetInvoiceOwnership(ctx context.Context, input GetInvoiceOwnershipAdapterInput) (GetOwnershipAdapterResponse, error)
}
