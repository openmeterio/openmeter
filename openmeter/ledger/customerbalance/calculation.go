package customerbalance

import (
	"github.com/alpacahq/alpacadecimal"
	"slices"

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
		if charge.Realizations.CurrentRun == nil {
			return alpacadecimal.Zero
		}
		if charge.Realizations.CurrentRun.IsVoidedBillingHistory() {
			return alpacadecimal.Zero
		}

		return charge.Realizations.CurrentRun.CreditRealizations.Sum()
	case meta.ChargeTypeUsageBased:
		charge, _ := i.AsUsageBasedCharge()
		total := alpacadecimal.Zero

		for _, run := range charge.Realizations {
			// Voided billing history either has already been reversed through billing,
			// or should have been removed by prorating/credit-note support. In both
			// cases it must not reduce the customer's outstanding balance.
			if run.IsVoidedBillingHistory() {
				continue
			}

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

func (i Impact) FeatureKey() string {
	switch i.Type() {
	case meta.ChargeTypeFlatFee:
		charge, _ := i.AsFlatFeeCharge()
		return charge.Intent.GetEffectiveFeatureKey()
	case meta.ChargeTypeUsageBased:
		charge, _ := i.AsUsageBasedCharge()
		return charge.Intent.GetEffectiveFeatureKey()
	default:
		return ""
	}
}

type chargeLiveBalanceCalculator struct{}

func (chargeLiveBalanceCalculator) CalculateLiveBalance(bookedBalance alpacadecimal.Decimal, impacts []Impact) alpacadecimal.Decimal {
	boundedAmount, unboundedAmount := sumImpactAmounts(impacts)

	// credit_then_invoice can only consume positive balance, while credit_only can drive it negative.
	liveBalance := applyBoundedAmount(bookedBalance, boundedAmount)

	return liveBalance.Sub(unboundedAmount)
}

func (chargeLiveBalanceCalculator) CalculateLiveBalanceFromSources(settledBalance alpacadecimal.Decimal, sources []liveBalanceSource, impacts []Impact) alpacadecimal.Decimal {
	liveBalance := settledBalance

	for _, impact := range impacts {
		if boundedAmount := impact.BoundedAmount(); boundedAmount.IsPositive() {
			liveBalance = liveBalance.Sub(consumeLiveBalanceSources(sources, impact.FeatureKey(), boundedAmount))
		}

		// credit_only can create feature-attributed advance/negative balance, so
		// it still changes live balance even when no positive eligible source
		// exists for the impact.
		liveBalance = liveBalance.Sub(impact.UnboundedAmount())
	}

	return liveBalance
}

func consumeLiveBalanceSources(sources []liveBalanceSource, featureKey string, target alpacadecimal.Decimal) alpacadecimal.Decimal {
	remaining := target
	consumed := alpacadecimal.Zero

	for idx := range sources {
		if !liveBalanceSourceMatchesFeature(sources[idx], featureKey) {
			continue
		}

		amount := sources[idx].amount
		if amount.GreaterThan(remaining) {
			amount = remaining
		}

		sources[idx].amount = sources[idx].amount.Sub(amount)
		remaining = remaining.Sub(amount)
		consumed = consumed.Add(amount)
		if remaining.IsZero() {
			break
		}
	}

	return consumed
}

// liveBalanceSourceMatchesFeature is allocability matching, not public balance
// filter matching. Unrestricted credit sources can cover any charge, but
// feature-restricted sources can only cover charges for that feature.
func liveBalanceSourceMatchesFeature(source liveBalanceSource, featureKey string) bool {
	if len(source.route.Features) == 0 {
		return true
	}

	return featureKey != "" && slices.Contains(source.route.Features, featureKey)
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
