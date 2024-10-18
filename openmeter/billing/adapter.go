package billing

import (
	"context"
	"fmt"

	billingentity "github.com/openmeterio/openmeter/openmeter/billing/entity"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

type Adapter interface {
	ProfileAdapter
	CustomerOverrideAdapter
	InvoiceItemAdapter

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

type InvoiceItemAdapter interface {
	CreateInvoiceItems(ctx context.Context, input CreateInvoiceItemsInput) ([]billingentity.InvoiceItem, error)
	GetPendingInvoiceItems(ctx context.Context, customerID customerentity.CustomerID) ([]billingentity.InvoiceItem, error)
}

type GetCustomerOverrideAdapterInput struct {
	Namespace  string
	CustomerID string

	IncludeDeleted bool
}

func (i GetCustomerOverrideAdapterInput) Validate() error {
	if i.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}

	if i.CustomerID == "" {
		return fmt.Errorf("customer id is required")
	}

	return nil
}

type UpdateCustomerOverrideAdapterInput struct {
	UpdateCustomerOverrideInput

	ResetDeletedAt bool
}

func (i UpdateCustomerOverrideAdapterInput) Validate() error {
	if err := i.UpdateCustomerOverrideInput.Validate(); err != nil {
		return fmt.Errorf("error validating update customer override input: %w", err)
	}

	return nil
}

type HasCustomerOverrideReferencingProfileAdapterInput genericNamespaceID

func (i HasCustomerOverrideReferencingProfileAdapterInput) Validate() error {
	return genericNamespaceID(i).Validate()
}
