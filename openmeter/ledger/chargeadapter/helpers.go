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

// invoicePaymentIdentity keeps ordinary invoiced charges attributed as spend,
// while custom-currency overage follows its credit-purchase invoice line and
// attributes the fiat payment lifecycle to the synthetic credit source.
func invoicePaymentIdentity(chargeID string, chargeCurrency currencies.Currency) ledger.EntryIdentityParts {
	if chargeCurrency.IsCustom() {
		return ledger.EntryIdentityParts{
			SourceChargeID: &chargeID,
		}
	}

	return ledger.EntryIdentityParts{
		SpendChargeID: &chargeID,
	}
}
