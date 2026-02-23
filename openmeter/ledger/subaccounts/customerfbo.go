package subaccounts

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
)

// CustomerFBOSubAccount is a sub-account that is a customer FBO sub-account.
type CustomerFBOSubAccount struct {
	locker *lockr.Locker
	ledger.SubAccount
}

func AsCustomerFBOSubAccount(l *lockr.Locker, s ledger.SubAccount) (*CustomerFBOSubAccount, error) {
	if l == nil {
		return nil, fmt.Errorf("locker is required")
	}

	if s.Address().AccountType() != ledger.AccountTypeCustomerFBO {
		return nil, fmt.Errorf("sub-account is not a customer FBO sub-account")
	}

	return &CustomerFBOSubAccount{
		locker:     l,
		SubAccount: s,
	}, nil
}

func (c *CustomerFBOSubAccount) CustomerDimensions() (ledger.CustomerSubAccountDimensions, error) {
	dim := c.SubAccount.Dimensions()

	res := ledger.CustomerSubAccountDimensions{
		Currency: dim.Currency,
		Features: dim.Feature,
	}

	var ok bool

	res.TaxCode, ok = dim.TaxCode.Get()
	if !ok {
		return ledger.CustomerSubAccountDimensions{}, fmt.Errorf("tax code is required")
	}

	res.CreditPriority, ok = dim.CreditPriority.Get()
	if !ok {
		return ledger.CustomerSubAccountDimensions{}, fmt.Errorf("credit priority is required")
	}

	return res, nil
}

// Lock locks the sub-account for the duration of the transaction
// FIXME: this should be a more explicit locking not just being for transaction scope...
func (c *CustomerFBOSubAccount) Lock(ctx context.Context, namespace string) error {
	key, err := lockr.NewKey("namespace", namespace, "subaccount", c.Address().SubAccountID())
	if err != nil {
		return fmt.Errorf("failed to create lock key: %w", err)
	}

	return c.locker.LockForTX(ctx, key)
}
