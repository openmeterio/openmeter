package ledger

import (
	"context"
	"fmt"
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

	GetSubAccountForRoute(ctx context.Context, route CustomerFBORouteParams) (SubAccount, error)
}

// CustomerFBORouteParams are routing parameters specific to customer FBO sub-accounts.
// CreditPriority is required (non-pointer) — the type system enforces its presence.
type CustomerFBORouteParams struct {
	Currency       string
	CreditPriority int
	TaxCode        *string
	Features       []string
}

func (p CustomerFBORouteParams) Validate() error {
	if err := p.Route().Validate(); err != nil {
		return err
	}

	// FBO-specific: credit priority is always present (enforced by type) but must be valid
	if err := ValidateCreditPriority(p.CreditPriority); err != nil {
		return fmt.Errorf("credit priority: %w", err)
	}
	return nil
}

func (p CustomerFBORouteParams) Route() Route {
	return Route{
		Currency:       p.Currency,
		TaxCode:        p.TaxCode,
		Features:       p.Features,
		CreditPriority: &p.CreditPriority,
	}
}

const DefaultCustomerFBOPriority = 100

// CustomerReceivableAccount is a customer receivable account.
type CustomerReceivableAccount interface {
	CustomerAccount

	GetSubAccountForRoute(ctx context.Context, route CustomerReceivableRouteParams) (SubAccount, error)
}

// CustomerReceivableRouteParams are routing parameters specific to customer receivable sub-accounts.
type CustomerReceivableRouteParams struct {
	Currency string
}

func (p CustomerReceivableRouteParams) Validate() error {
	return p.Route().Validate()
}

func (p CustomerReceivableRouteParams) Route() Route {
	return Route{
		Currency: p.Currency,
	}
}

// ----------------------------------------------------------------------------
// Business Accounts
// ----------------------------------------------------------------------------

// BusinessAccount is a business account.
type BusinessAccount interface {
	Account

	GetSubAccountForRoute(ctx context.Context, route BusinessRouteParams) (SubAccount, error)
}

type BusinessRouteParams struct {
	Currency string
}

func (p BusinessRouteParams) Validate() error {
	return p.Route().Validate()
}

func (p BusinessRouteParams) Route() Route {
	return Route{
		Currency: p.Currency,
	}
}
