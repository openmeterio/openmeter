package account

import (
	"context"

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

func newBusinessAccount(a *Account) *BusinessAccount {
	return &BusinessAccount{Account: a}
}

// GetSubAccountForRoute finds or creates a sub-account for the given route.
func (a *BusinessAccount) GetSubAccountForRoute(ctx context.Context, params ledger.BusinessRouteParams) (ledger.SubAccount, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}

	return a.services.SubAccountService.EnsureSubAccount(ctx, ledger.CreateSubAccountInput{
		Namespace: a.data.ID.Namespace,
		AccountID: a.data.ID.ID,
		Route:     params.Route(),
	})
}
