package reconciler

import (
	"errors"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

func semanticProrateDecision(existing persistedstate.Entity, targetState targetstate.SubscriptionItemWithPeriods) (bool, error) {
	if !existing.IsFlatFee() {
		return false, nil
	}

	// For the prorate decision we need to understand the underlying entity type, as the proration logic is different:
	// - for a line phaseiterator yields the prorated amount based on the service period and the per-unit amount
	// - for a charge the charge service will handle the proration logic
	switch existing.GetType() {
	case persistedstate.EntityTypeLineOrHierarchy:
		lineOrHierarchy, err := existing.AsLineOrHierarchy()
		if err != nil {
			return false, fmt.Errorf("getting line or hierarchy: %w", err)
		}
		return semanticLineProrateDecision(lineOrHierarchy, targetState)
	case persistedstate.EntityTypeCharge:
		charge, err := existing.AsCharge()
		if err != nil {
			return false, fmt.Errorf("getting charge: %w", err)
		}

		return semanticChargeProrateDecision(charge, targetState)
	default:
		return false, fmt.Errorf("unsupported entity type: %s", existing.GetType())
	}
}

func semanticLineProrateDecision(existing billing.LineOrHierarchy, targetState targetstate.SubscriptionItemWithPeriods) (bool, error) {
	gatheringLine, err := targetState.GetExpectedLineOrErr(input.Subscription, input.Currency)
	if err != nil {
		return false, fmt.Errorf("getting expected line: %w", err)
	}

	// expectedLine is materialized through targetstate.LineFromSubscriptionRateCard, which
	// applies the existing subscription-sync proration rules when deriving the flat-fee amount.
	targetAmount, err := invoiceupdater.GetFlatFeePerUnitAmount(expectedLine)
	if err != nil {
		return false, fmt.Errorf("getting expected flat fee amount: %w", err)
	}

	switch existing.Type() {
	case billing.LineOrHierarchyTypeLine:
		existingLine, err := existing.AsGenericLine()
		if err != nil {
			return false, fmt.Errorf("getting existing line: %w", err)
		}

		if !invoiceupdater.IsFlatFee(existingLine) {
			return false, nil
		}

		existingAmount, err := invoiceupdater.GetFlatFeePerUnitAmount(existingLine)
		if err != nil {
			return false, fmt.Errorf("getting existing flat fee amount: %w", err)
		}

		return !existingAmount.Equal(targetAmount) || !existingLine.GetServicePeriod().Equal(expectedLine.ServicePeriod), nil
	case billing.LineOrHierarchyTypeHierarchy:
		return false, errors.New("flat fee lines cannot be reconciled against a split line hierarchy")
	default:
		return false, fmt.Errorf("unsupported line or hierarchy type: %s", existing.Type())
	}
}

func semanticChargeProrateDecision(existing charges.Charge, targetState targetstate.SubscriptionItemWithPeriods) (bool, error) {
	if existing.Type() != meta.ChargeTypeFlatFee {
		return false, nil
	}

	existingFlatFee, err := existing.AsFlatFeeCharge()
	if err != nil {
		return false, fmt.Errorf("getting existing flat fee charge: %w", err)
	}

	// Do not prorate if pro-rating is not enabled.
	if !existingFlatFee.Intent.ProRating.Enabled {
		return false, nil
	}

	price := targetState.SubscriptionItem.RateCard.AsMeta().Price
	if price == nil {
		return false, fmt.Errorf("price is nil")
	}

	priceFlat, err := price.AsFlat()
	if err != nil {
		return false, fmt.Errorf("getting price flat: %w", err)
	}

	// Proration is required if:
	// - the service period or full service period has changed
	// - the amount before proration has changed
	//
	// As proration is calculated as lengthOfServicePeriod / lengthOfFullServicePeriod * amountBeforeProration,

	if !existingFlatFee.Intent.ServicePeriod.Equal(targetState.ServicePeriod.ToClosedPeriod()) {
		return true, nil
	}

	if !existingFlatFee.Intent.FullServicePeriod.Equal(targetState.FullServicePeriod.ToClosedPeriod()) {
		return true, nil
	}

	if !existingFlatFee.Intent.AmountBeforeProration.Equal(priceFlat.Amount) {
		return true, nil
	}

	return false, nil
}
