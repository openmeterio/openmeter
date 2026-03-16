package account

import (
	"context"
	"errors"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/ledgerv2"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/samber/mo"
)

type CreateAccountInput struct {
	Namespace   string
	Type        ledger.AccountType
	Annotations models.Annotations
}

type CreateCustomerAccountInput struct {
	Namespace   string
	CustomerID  string
	Type        ledger.AccountType
	Annotations models.Annotations
}

type DimensionGetter interface {
	Validate() error
	Dimensions() ledgerv2.SubAccountDimensions
}

var _ DimensionGetter = (*CreateBusinessSubAccountInput)(nil)

type CreateBusinessSubAccountInput struct {
	Currency currencyx.Code
}

func (c *CreateBusinessSubAccountInput) Validate() error {
	if c.Currency == "" {
		return errors.New("currency is required")
	}
	return nil
}

func (c *CreateBusinessSubAccountInput) Dimensions() ledgerv2.SubAccountDimensions {
	return ledgerv2.SubAccountDimensions{
		Currency: mo.Some(c.Currency),
	}
}

type Service interface {
	CreateAccount(ctx context.Context, input CreateAccountInput) (ledgerv2.OrganizationalAccount, error)
	CreateCustomerAccount(ctx context.Context, input CreateCustomerAccountInput) (ledgerv2.CustomerAccount, error)
	CreateSubAccount(ctx context.Context, account ledgerv2.Account, input DimensionGetter) (ledgerv2.SubAccount, error)

	GetAccountByID(ctx context.Context, id models.NamespacedID) (ledgerv2.Account, error)
	GetSubAccountForDimensions(ctx context.Context, account ledgerv2.Account, dimensions DimensionGetter) (ledgerv2.SubAccount, error)
}
