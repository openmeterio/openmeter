package billing

import (
	"context"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ProfileAdapter
	CustomerOverrideAdapter
	InvoiceLineAdapter
	InvoiceAdapter

	Tx(ctx context.Context) (context.Context, transaction.Driver, error)
	WithTx(ctx context.Context, tx *entutils.TxDriver) Adapter
}

type ProfileAdapter interface {
	CreateProfile(ctx context.Context, input CreateProfileInput) (*billingentity.BaseProfile, error)
	ListProfiles(ctx context.Context, input ListProfilesInput) (pagination.PagedResponse[billingentity.BaseProfile], error)
	GetProfile(ctx context.Context, input GetProfileInput) (*billingentity.BaseProfile, error)
	GetDefaultProfile(ctx context.Context, input GetDefaultProfileInput) (*billingentity.BaseProfile, error)
	DeleteProfile(ctx context.Context, input DeleteProfileInput) error
	UpdateProfile(ctx context.Context, input UpdateProfileAdapterInput) (*billingentity.BaseProfile, error)
}

type CustomerOverrideAdapter interface {
	CreateCustomerOverride(ctx context.Context, input CreateCustomerOverrideInput) (*billingentity.CustomerOverride, error)
	GetCustomerOverride(ctx context.Context, input GetCustomerOverrideAdapterInput) (*billingentity.CustomerOverride, error)
	UpdateCustomerOverride(ctx context.Context, input UpdateCustomerOverrideAdapterInput) (*billingentity.CustomerOverride, error)
	DeleteCustomerOverride(ctx context.Context, input DeleteCustomerOverrideInput) error

	GetCustomerOverrideReferencingProfile(ctx context.Context, input HasCustomerOverrideReferencingProfileAdapterInput) ([]customerentity.CustomerID, error)
}

type InvoiceLineAdapter interface {
	CreateInvoiceLines(ctx context.Context, input CreateInvoiceLinesAdapterInput) (*CreateInvoiceLinesResponse, error)
}

type InvoiceAdapter interface {
	CreateInvoice(ctx context.Context, input CreateInvoiceAdapterInput) (CreateInvoiceAdapterRespone, error)
	GetInvoiceById(ctx context.Context, input GetInvoiceByIdInput) (billingentity.Invoice, error)
	ListInvoices(ctx context.Context, input ListInvoicesInput) (ListInvoicesResponse, error)
}
