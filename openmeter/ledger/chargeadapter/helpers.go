package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/ledger"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

func settledBalanceForSubAccount(ctx context.Context, querier ledger.BalanceQuerier, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := querier.GetSubAccountBalance(ctx, subAccount, ledger.BalanceQuery{})
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance.Settled(), nil
}

func taxCodeIDFromIntent(taxConfig *productcatalog.TaxCodeConfig) *string {
	if taxConfig == nil {
		return nil
	}
	return taxConfig.TaxCodeID
}

func taxBehaviorFromIntent(taxConfig *productcatalog.TaxCodeConfig) *ledger.TaxBehavior {
	if taxConfig == nil || taxConfig.TaxCodeID == nil || taxConfig.Behavior == nil {
		return nil
	}

	return lo.ToPtr(ledger.TaxBehavior(*taxConfig.Behavior))
}
