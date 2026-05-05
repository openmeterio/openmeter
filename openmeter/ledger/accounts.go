package ledger

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// ----------------------------------------------------------------------------
// Customer Accounts
// ----------------------------------------------------------------------------

// CustomerAccount is a Customer specific account
type CustomerAccount interface {
	Account
}

// CustomerFBOAccount is a customer FBO account.
type CustomerFBOAccount interface {
	CustomerAccount

	GetSubAccountForRoute(ctx context.Context, route CustomerFBORouteParams) (SubAccount, error)
}

// CustomerFBORouteParams are routing parameters specific to customer FBO sub-accounts.
// CreditPriority is required (non-pointer) — the type system enforces its presence.
type CustomerFBORouteParams struct {
	Currency       currencyx.Code
	CreditPriority int
	TaxCode        *string
	TaxBehavior    *TaxBehavior
	Features       []string
	CostBasis      *alpacadecimal.Decimal
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
		TaxBehavior:    p.TaxBehavior,
		Features:       p.Features,
		CostBasis:      p.CostBasis,
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
// TransactionAuthorizationStatus is required; callers must explicitly select the open or authorized route.
type CustomerReceivableRouteParams struct {
	Currency                       currencyx.Code
	CostBasis                      *alpacadecimal.Decimal
	TransactionAuthorizationStatus TransactionAuthorizationStatus
}

func (p CustomerReceivableRouteParams) Validate() error {
	if err := p.TransactionAuthorizationStatus.Validate(); err != nil {
		return err
	}

	return p.Route().Validate()
}

func (p CustomerReceivableRouteParams) Route() Route {
	return Route{
		Currency:                       p.Currency,
		CostBasis:                      p.CostBasis,
		TransactionAuthorizationStatus: &p.TransactionAuthorizationStatus,
	}
}

// CustomerAccruedAccount is a customer accrued account used as a staging area for
// usage that has been acknowledged but not yet recognized as earnings.
type CustomerAccruedAccount interface {
	CustomerAccount

	GetSubAccountForRoute(ctx context.Context, route CustomerAccruedRouteParams) (SubAccount, error)
}

// CustomerAccruedRouteParams are routing parameters specific to customer accrued sub-accounts.
// Routed by currency only for now.
type CustomerAccruedRouteParams struct {
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (p CustomerAccruedRouteParams) Validate() error {
	return p.Route().Validate()
}

func (p CustomerAccruedRouteParams) Route() Route {
	return Route{
		Currency:  p.Currency,
		CostBasis: p.CostBasis,
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
	Currency  currencyx.Code
	CostBasis *alpacadecimal.Decimal
}

func (p BusinessRouteParams) Validate() error {
	return p.Route().Validate()
}

func (p BusinessRouteParams) Route() Route {
	return Route{
		Currency:  p.Currency,
		CostBasis: p.CostBasis,
	}
}
