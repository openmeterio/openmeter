package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/ledger"
)

func settledBalanceForSubAccount(ctx context.Context, querier ledger.BalanceQuerier, subAccount ledger.SubAccount) (alpacadecimal.Decimal, error) {
	balance, err := querier.GetSubAccountBalance(ctx, subAccount, ledger.BalanceQuery{})
	if err != nil {
		return alpacadecimal.Decimal{}, fmt.Errorf("get balance for sub-account %s: %w", subAccount.Address().SubAccountID(), err)
	}

	return balance, nil
}

// customCurrencyIdentity extracts the managed custom currency identity from a
// resolved charge currency, for threading onto ledger routes. Returns nil for
// fiat currencies.
func customCurrencyIdentity(currency currencies.Currency) *ledger.CustomCurrencyIdentity {
	if !currency.IsCustom() {
		return nil
	}

	return &ledger.CustomCurrencyIdentity{
		ID:        currency.ID,
		Precision: int(currency.Details().Precision),
	}
}
