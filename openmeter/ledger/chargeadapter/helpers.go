package chargeadapter

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
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
// while custom-currency overage follows its credit-purchase-equivalent booking
// and attributes the fiat payment lifecycle to the synthetic credit source.
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

// resolvedCostBasisCharge is satisfied by usagebased.Charge and flatfee.Charge:
// both embed a ChargeBase exposing the charge's currency and persisted
// dynamic cost-basis snapshot.
type resolvedCostBasisCharge interface {
	GetCurrency() currencies.Currency
	GetResolvedCostBasis() *costbasis.State
}

// resolveInvoiceCostBasis returns the cost-basis route the invoice payment
// lifecycle must authorize and settle against. Fiat charges always settle at
// par (cost basis 1). Custom-currency charges settle at the same persisted
// rate the overage accrual already converted through: the fiat receivable
// created by OnCustomCurrencyOverageAccrued lives on that rate's route, not
// on the par route, so payment authorization/settlement must reference it
// rather than re-resolving or recomputing a rate.
func resolveInvoiceCostBasis(charge resolvedCostBasisCharge) (*alpacadecimal.Decimal, error) {
	if !charge.GetCurrency().IsCustom() {
		return invoiceCostBasis, nil
	}

	resolved := charge.GetResolvedCostBasis()
	if resolved == nil {
		return nil, fmt.Errorf("custom currency charge is missing a resolved cost basis")
	}

	return &resolved.CostBasis, nil
}
