package customerbalance

import (
	"github.com/alpacahq/alpacadecimal"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

type Impact struct {
	charges.Charge

	amount alpacadecimal.Decimal
}

func NewImpact(charge charges.Charge, amount alpacadecimal.Decimal) (Impact, error) {
	if _, err := charge.SettlementMode(); err != nil {
		return Impact{}, err
	}

	return Impact{
		Charge: charge,
		amount: amount,
	}, nil
}

func (i Impact) OutstandingAmount() alpacadecimal.Decimal {
	amount := i.amount.Sub(i.RealizedCredits())
	if amount.IsNegative() {
		return alpacadecimal.Zero
	}

	return amount
}

func (i Impact) RealizedCredits() alpacadecimal.Decimal {
	switch i.Type() {
	case meta.ChargeTypeFlatFee:
		charge, _ := i.AsFlatFeeCharge()
		return charge.Realizations.CreditRealizations.Sum()
	case meta.ChargeTypeUsageBased:
		charge, _ := i.AsUsageBasedCharge()
		total := alpacadecimal.Zero

		for _, run := range charge.Realizations {
			total = total.Add(run.CreditsAllocated.Sum())
		}

		return total
	default:
		return alpacadecimal.Zero
	}
}

func (i Impact) BoundedAmount() alpacadecimal.Decimal {
	settlementMode, err := i.SettlementMode()
	if err != nil || settlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		return alpacadecimal.Zero
	}

	return i.OutstandingAmount()
}

func (i Impact) UnboundedAmount() alpacadecimal.Decimal {
	settlementMode, err := i.SettlementMode()
	if err != nil || settlementMode != productcatalog.CreditOnlySettlementMode {
		return alpacadecimal.Zero
	}

	return i.OutstandingAmount()
}

type chargePendingBalanceCalculator struct{}

func (chargePendingBalanceCalculator) CalculatePendingBalance(bookedBalance alpacadecimal.Decimal, impacts []Impact) alpacadecimal.Decimal {
	boundedAmount, unboundedAmount := sumImpactAmounts(impacts)

	// credit_then_invoice can only consume positive balance, while credit_only can drive it negative.
	pendingBalance := applyBoundedAmount(bookedBalance, boundedAmount)

	return pendingBalance.Sub(unboundedAmount)
}

func sumImpactAmounts(impacts []Impact) (bounded alpacadecimal.Decimal, unbounded alpacadecimal.Decimal) {
	bounded = alpacadecimal.Zero
	unbounded = alpacadecimal.Zero

	for _, impact := range impacts {
		bounded = bounded.Add(impact.BoundedAmount())
		unbounded = unbounded.Add(impact.UnboundedAmount())
	}

	return bounded, unbounded
}

func applyBoundedAmount(balance alpacadecimal.Decimal, amount alpacadecimal.Decimal) alpacadecimal.Decimal {
	if !balance.GreaterThan(alpacadecimal.Zero) {
		return balance
	}

	if amount.GreaterThan(balance) {
		return alpacadecimal.Zero
	}

	return balance.Sub(amount)
}
