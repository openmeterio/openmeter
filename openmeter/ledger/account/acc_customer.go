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

// GetSubAccountForDimensions finds or creates a sub-account for the given dimensions.
// For now only Currency is used for filtering; TaxCode/CreditPriority/Features are TBD.
func (a *CustomerFBOAccountImpl) GetSubAccountForDimensions(ctx context.Context, dimensions ledger.CustomerFBOSubAccountDimensions) (ledger.SubAccount, error) {
	currDimID, err := extractDimensionID(dimensions.Currency)
	if err != nil {
		return nil, fmt.Errorf("currency dimension: %w", err)
	}

	return findOrCreateSubAccount(ctx, a.services.SubAccountService, a.data.ID.Namespace, a.data.ID.ID, currDimID)
}

// ----------------------------------------------------------------------------
// CustomerReceivableAccountImpl
// ----------------------------------------------------------------------------

// CustomerReceivableAccountImpl implements ledger.CustomerReceivableAccount.
type CustomerReceivableAccountImpl struct {
	*CustomerAccount
}

var _ ledger.CustomerReceivableAccount = (*CustomerReceivableAccountImpl)(nil)

// GetSubAccountForDimensions finds or creates a sub-account for the given dimensions.
func (a *CustomerReceivableAccountImpl) GetSubAccountForDimensions(ctx context.Context, dimensions ledger.CustomerReceivableSubAccountDimensions) (ledger.SubAccount, error) {
	currDimID, err := extractDimensionID(dimensions.Currency)
	if err != nil {
		return nil, fmt.Errorf("currency dimension: %w", err)
	}

	return findOrCreateSubAccount(ctx, a.services.SubAccountService, a.data.ID.Namespace, a.data.ID.ID, currDimID)
}

// ----------------------------------------------------------------------------
// BusinessAccountImpl
// ----------------------------------------------------------------------------

// BusinessAccountImpl implements ledger.BusinessAccount.
type BusinessAccountImpl struct {
	*Account
}

var _ ledger.BusinessAccount = (*BusinessAccountImpl)(nil)

// GetSubAccountForDimensions finds or creates a sub-account for the given dimensions.
func (a *BusinessAccountImpl) GetSubAccountForDimensions(ctx context.Context, dimensions ledger.BusinessSubAccountDimensions) (ledger.SubAccount, error) {
	currDimID, err := extractDimensionID(dimensions.Currency)
	if err != nil {
		return nil, fmt.Errorf("currency dimension: %w", err)
	}

	return findOrCreateSubAccount(ctx, a.services.SubAccountService, a.data.ID.Namespace, a.data.ID.ID, currDimID)
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func extractDimensionID(dim ledger.DimensionCurrency) (string, error) {
	ider, ok := dim.(dimensionIDer)
	if !ok {
		return "", fmt.Errorf("dimension does not expose its DB id (type %T)", dim)
	}

	return ider.dimensionID(), nil
}

func findOrCreateSubAccount(
	ctx context.Context,
	svc SubAccountCreatorLister,
	namespace, accountID, currencyDimID string,
) (ledger.SubAccount, error) {
	subs, err := svc.ListSubAccounts(ctx, ListSubAccountsInput{
		Namespace: namespace,
		AccountID: accountID,
		Dimensions: ledger.QueryDimensions{
			CurrencyID: currencyDimID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list sub-accounts: %w", err)
	}

	if len(subs) > 0 {
		return subs[0], nil
	}

	sub, err := svc.CreateSubAccount(ctx, CreateSubAccountInput{
		Namespace: namespace,
		AccountID: accountID,
		Dimensions: SubAccountDimensionInput{
			CurrencyDimensionID: currencyDimID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sub-account: %w", err)
	}

	return sub, nil
}
