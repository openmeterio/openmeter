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
