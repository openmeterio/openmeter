package account

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

// ----------------------------------------------------------------------------
// BusinessAccount
// ----------------------------------------------------------------------------

// BusinessAccount implements ledger.BusinessAccount.
type BusinessAccount struct {
	*Account
}

var _ ledger.BusinessAccount = (*BusinessAccount)(nil)

// GetSubAccountForRoute finds or creates a sub-account for the given route.
func (a *BusinessAccount) GetSubAccountForRoute(ctx context.Context, params ledger.BusinessRouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return a.services.SubAccountService.EnsureSubAccount(ctx, CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}

// AsBusinessAccount wraps the Account as a BusinessAccount.
func (a *Account) AsBusinessAccount() (*BusinessAccount, error) {
	switch a.data.AccountType {
	case ledger.AccountTypeWash, ledger.AccountTypeEarnings, ledger.AccountTypeBrokerage:
	default:
		return nil, fmt.Errorf("account type %s is not a business account", a.data.AccountType)
	}

	return &BusinessAccount{Account: a}, nil
}
