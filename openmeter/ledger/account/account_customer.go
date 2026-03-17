package account

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

func (a *Account) AsCustomerAccount() (*CustomerAccount, error) {
	switch a.data.AccountType {
	case ledger.AccountTypeCustomerFBO, ledger.AccountTypeCustomerReceivable, ledger.AccountTypeCustomerAccrued:
	default:
		return nil, fmt.Errorf("account type %s is not a customer account", a.data.AccountType)
	}

	return &CustomerAccount{
		Account: a,
	}, nil
}

// AsCustomerFBOAccount wraps the Account as a CustomerFBOAccount.
func (a *Account) AsCustomerFBOAccount() (*CustomerFBOAccount, error) {
	if a.data.AccountType != ledger.AccountTypeCustomerFBO {
		return nil, fmt.Errorf("account type %s is not a customer FBO account", a.data.AccountType)
	}

	cAcc, err := a.AsCustomerAccount()
	if err != nil {
		return nil, err
	}

	return &CustomerFBOAccount{CustomerAccount: cAcc}, nil
}

// AsCustomerReceivableAccount wraps the Account as a CustomerReceivableAccount.
func (a *Account) AsCustomerReceivableAccount() (*CustomerReceivableAccount, error) {
	if a.data.AccountType != ledger.AccountTypeCustomerReceivable {
		return nil, fmt.Errorf("account type %s is not a customer receivable account", a.data.AccountType)
	}

	cAcc, err := a.AsCustomerAccount()
	if err != nil {
		return nil, err
	}

	return &CustomerReceivableAccount{CustomerAccount: cAcc}, nil
}

// AsCustomerAccruedAccount wraps the Account as a CustomerAccruedAccount.
func (a *Account) AsCustomerAccruedAccount() (*CustomerAccruedAccount, error) {
	if a.data.AccountType != ledger.AccountTypeCustomerAccrued {
		return nil, fmt.Errorf("account type %s is not a customer accrued account", a.data.AccountType)
	}

	cAcc, err := a.AsCustomerAccount()
	if err != nil {
		return nil, err
	}

	return &CustomerAccruedAccount{CustomerAccount: cAcc}, nil
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
// CustomerFBOAccount
// ----------------------------------------------------------------------------

// CustomerFBOAccount implements ledger.CustomerFBOAccount.
type CustomerFBOAccount struct {
	*CustomerAccount
}

var _ ledger.CustomerFBOAccount = (*CustomerFBOAccount)(nil)

// GetSubAccountForRoute finds or creates a sub-account for the given route.
func (a *CustomerFBOAccount) GetSubAccountForRoute(ctx context.Context, params ledger.CustomerFBORouteParams) (ledger.SubAccount, error) {
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
// CustomerReceivableAccount
// ----------------------------------------------------------------------------

// CustomerReceivableAccount implements ledger.CustomerReceivableAccount.
type CustomerReceivableAccount struct {
	*CustomerAccount
}

var _ ledger.CustomerReceivableAccount = (*CustomerReceivableAccount)(nil)

// GetSubAccountForRoute finds or creates a sub-account for the given route.
func (a *CustomerReceivableAccount) GetSubAccountForRoute(ctx context.Context, params ledger.CustomerReceivableRouteParams) (ledger.SubAccount, error) {
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
// CustomerAccruedAccount
// ----------------------------------------------------------------------------

// CustomerAccruedAccount implements ledger.CustomerAccruedAccount.
type CustomerAccruedAccount struct {
	*CustomerAccount
}

var _ ledger.CustomerAccruedAccount = (*CustomerAccruedAccount)(nil)

// GetSubAccountForRoute finds or creates a sub-account for the given route.
// Accrued accounts are routed by currency only.
func (a *CustomerAccruedAccount) GetSubAccountForRoute(ctx context.Context, params ledger.CustomerAccruedRouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return a.services.SubAccountService.EnsureSubAccount(ctx, CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}
