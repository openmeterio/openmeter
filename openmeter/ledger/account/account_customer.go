package account

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

func (a *Account) AsCustomerAccount() (*CustomerAccount, error) {
	switch a.data.AccountType {
	case ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerReceivable:
	default:
		return nil, fmt.Errorf("account type %s is not a customer account", a.data.AccountType)
	}

	return &CustomerAccount{
		Account: a,
	}, nil
}

// AsCustomerFBOAccount wraps the Account as a CustomerFBOAccountImpl.
func (a *Account) AsCustomerFBOAccount() (*CustomerFBOAccountImpl, error) {
	if a.data.AccountType != ledger.AccountTypeCustomerFBO {
		return nil, fmt.Errorf("account type %s is not a customer FBO account", a.data.AccountType)
	}

	cAcc, err := a.AsCustomerAccount()
	if err != nil {
		return nil, err
	}

	return &CustomerFBOAccountImpl{CustomerAccount: cAcc}, nil
}

// AsCustomerReceivableAccount wraps the Account as a CustomerReceivableAccountImpl.
func (a *Account) AsCustomerReceivableAccount() (*CustomerReceivableAccountImpl, error) {
	if a.data.AccountType != ledger.AccountTypeCustomerReceivable {
		return nil, fmt.Errorf("account type %s is not a customer receivable account", a.data.AccountType)
	}

	cAcc, err := a.AsCustomerAccount()
	if err != nil {
		return nil, err
	}

	return &CustomerReceivableAccountImpl{CustomerAccount: cAcc}, nil
}

// AsBusinessAccount wraps the Account as a BusinessAccountImpl.
func (a *Account) AsBusinessAccount() (*BusinessAccountImpl, error) {
	switch a.data.AccountType {
	case ledger.AccountTypeWash, ledger.AccountTypeEarnings, ledger.AccountTypeBrokerage:
	default:
		return nil, fmt.Errorf("account type %s is not a business account", a.data.AccountType)
	}

	return &BusinessAccountImpl{Account: a}, nil
}

// ----------------------------------------------------------------------------
// CustomerAccount — base for FBO and Receivable
// ----------------------------------------------------------------------------

type CustomerAccount struct {
	*Account
}

var _ ledger.CustomerAccount = (*CustomerAccount)(nil)

// Lock locks the account for the duration of the transaction
func (c *CustomerAccount) Lock(ctx context.Context) error {
	key, err := lockr.NewKey("namespace", c.Account.data.ID.Namespace, "account", c.Account.data.ID.ID)
	if err != nil {
		return fmt.Errorf("failed to create lock key: %w", err)
	}

	return c.services.Locker.LockForTX(ctx, key)
}

// ----------------------------------------------------------------------------
// CustomerFBOAccountImpl
// ----------------------------------------------------------------------------

// CustomerFBOAccountImpl implements ledger.CustomerFBOAccount.
type CustomerFBOAccountImpl struct {
	*CustomerAccount
}

var _ ledger.CustomerFBOAccount = (*CustomerFBOAccountImpl)(nil)

// GetSubAccountForRoute finds or creates a sub-account for the given route.
func (a *CustomerFBOAccountImpl) GetSubAccountForRoute(ctx context.Context, params ledger.CustomerFBORouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return a.services.SubAccountService.EnsureSubAccount(ctx, CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}

// ----------------------------------------------------------------------------
// CustomerReceivableAccountImpl
// ----------------------------------------------------------------------------

// CustomerReceivableAccountImpl implements ledger.CustomerReceivableAccount.
type CustomerReceivableAccountImpl struct {
	*CustomerAccount
}

var _ ledger.CustomerReceivableAccount = (*CustomerReceivableAccountImpl)(nil)

// GetSubAccountForRoute finds or creates a sub-account for the given route.
func (a *CustomerReceivableAccountImpl) GetSubAccountForRoute(ctx context.Context, params ledger.CustomerReceivableRouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return a.services.SubAccountService.EnsureSubAccount(ctx, CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}

// ----------------------------------------------------------------------------
// BusinessAccountImpl
// ----------------------------------------------------------------------------

// BusinessAccountImpl implements ledger.BusinessAccount.
type BusinessAccountImpl struct {
	*Account
}

var _ ledger.BusinessAccount = (*BusinessAccountImpl)(nil)

// GetSubAccountForRoute finds or creates a sub-account for the given route.
func (a *BusinessAccountImpl) GetSubAccountForRoute(ctx context.Context, params ledger.BusinessRouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return a.services.SubAccountService.EnsureSubAccount(ctx, CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}
