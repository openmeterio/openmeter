package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func settledBalanceForSubAccount(ctx context.Context, querier ledger.BalanceQuerier, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := querier.GetSubAccountBalance(ctx, subAccount, ledger.BalanceQuery{})
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}
