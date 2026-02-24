package ledger

import (
	"context"

	"github.com/samber/mo"
)

// ----------------------------------------------------------------------------
// Customer Accounts
// ----------------------------------------------------------------------------

// CustomerAccount is a Customer specific account
type CustomerAccount interface {
	Account

	// Lock locks the entire account for the duration of the transaction so balances are stable.
	Lock(ctx context.Context) error
}

// CustomerFBOAccount is a customer FBO account.
type CustomerFBOAccount interface {
	CustomerAccount

	GetSubAccountForDimensions(ctx context.Context, dimensions CustomerFBOSubAccountDimensions) (SubAccount, error)
}

// CustomerFBOSubAccountDimensions are dimensions specific to customer FBO sub-accounts.
type CustomerFBOSubAccountDimensions struct {
	Currency       DimensionCurrency
	TaxCode        DimensionTaxCode
	CreditPriority DimensionCreditPriority
	Features       mo.Option[DimensionFeature]
}

// CustomerReceivableAccount is a customer receivable account.
type CustomerReceivableAccount interface {
	CustomerAccount

	GetSubAccountForDimensions(ctx context.Context, dimensions CustomerReceivableSubAccountDimensions) (SubAccount, error)
}

// CustomerReceivableSubAccountDimensions are dimensions specific to customer receivable sub-accounts.
type CustomerReceivableSubAccountDimensions struct {
	Currency DimensionCurrency
}

// ----------------------------------------------------------------------------
// Business Accounts
// ----------------------------------------------------------------------------

// BusinessAccount is a business account.
type BusinessAccount interface {
	Account

	GetSubAccountForDimensions(ctx context.Context, dimensions BusinessSubAccountDimensions) (SubAccount, error)
}

type BusinessSubAccountDimensions struct {
	Currency DimensionCurrency
}
