package account

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func newCustomerAccount(a *Account) *CustomerAccount {
	return &CustomerAccount{
		Account: a,
	}
}

type CustomerAccount struct {
	*Account
}

var _ ledger.CustomerAccount = (*CustomerAccount)(nil)

// ----------------------------------------------------------------------------
// CustomerFBOAccount
// ----------------------------------------------------------------------------

// CustomerFBOAccount implements ledger.CustomerFBOAccount.
type CustomerFBOAccount struct {
	*CustomerAccount
}

var _ ledger.CustomerFBOAccount = (*CustomerFBOAccount)(nil)

func newCustomerFBOAccount(a *Account) *CustomerFBOAccount {
	return &CustomerFBOAccount{CustomerAccount: newCustomerAccount(a)}
}

// GetSubAccountForRoute finds or creates a sub-account for the given route.
func (a *CustomerFBOAccount) GetSubAccountForRoute(ctx context.Context, params ledger.CustomerFBORouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return a.services.SubAccountService.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}

// ----------------------------------------------------------------------------
// CustomerReceivableAccount
// ----------------------------------------------------------------------------

// CustomerReceivableAccount implements ledger.CustomerReceivableAccount.
type CustomerReceivableAccount struct {
	*CustomerAccount
}

var _ ledger.CustomerReceivableAccount = (*CustomerReceivableAccount)(nil)

func newCustomerReceivableAccount(a *Account) *CustomerReceivableAccount {
	return &CustomerReceivableAccount{CustomerAccount: newCustomerAccount(a)}
}

// GetSubAccountForRoute finds or creates a sub-account for the given route.
func (a *CustomerReceivableAccount) GetSubAccountForRoute(ctx context.Context, params ledger.CustomerReceivableRouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return a.services.SubAccountService.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}

// ----------------------------------------------------------------------------
// CustomerAccruedAccount
// ----------------------------------------------------------------------------

// CustomerAccruedAccount implements ledger.CustomerAccruedAccount.
type CustomerAccruedAccount struct {
	*CustomerAccount
}

var _ ledger.CustomerAccruedAccount = (*CustomerAccruedAccount)(nil)

func newCustomerAccruedAccount(a *Account) *CustomerAccruedAccount {
	return &CustomerAccruedAccount{CustomerAccount: newCustomerAccount(a)}
}

// GetSubAccountForRoute finds or creates a sub-account for the given route.
// Accrued accounts are routed by currency only.
func (a *CustomerAccruedAccount) GetSubAccountForRoute(ctx context.Context, params ledger.CustomerAccruedRouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return a.services.SubAccountService.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}
