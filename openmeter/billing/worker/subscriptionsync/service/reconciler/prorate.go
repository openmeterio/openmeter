package reconciler

import (
	"errors"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
)

type ProrateDecision struct {
	ShouldProrate  bool
	OriginalAmount alpacadecimal.Decimal
	TargetAmount   alpacadecimal.Decimal
}

func semanticProrateDecision(existing persistedstate.Item, target targetstate.StateItem) (ProrateDecision, error) {
	expectedLine, err := target.GetExpectedLineOrErr()
	if err != nil {
		return ProrateDecision{}, fmt.Errorf("getting expected line for target[%s]: %w", target.UniqueID, err)
	}

	if !invoiceupdater.IsFlatFee(expectedLine) {
		return ProrateDecision{}, nil
	}

	// expectedLine is materialized through targetstate.LineFromSubscriptionRateCard, which
	// applies the existing subscription-sync proration rules when deriving the flat-fee amount.
	targetAmount, err := invoiceupdater.GetFlatFeePerUnitAmount(expectedLine)
	if err != nil {
		return ProrateDecision{}, fmt.Errorf("getting expected flat fee amount: %w", err)
	}

	switch existing.Type() {
	case billing.LineOrHierarchyTypeLine:
		existingLine, err := persistedstate.ItemAsLine(existing)
		if err != nil {
			return ProrateDecision{}, err
		}

		if !invoiceupdater.IsFlatFee(existingLine) {
			return ProrateDecision{}, nil
		}

		existingAmount, err := invoiceupdater.GetFlatFeePerUnitAmount(existingLine)
		if err != nil {
			return ProrateDecision{}, fmt.Errorf("getting existing flat fee amount: %w", err)
		}

		return ProrateDecision{
			ShouldProrate:  !existingAmount.Equal(targetAmount) || !existingLine.GetServicePeriod().Equal(expectedLine.ServicePeriod),
			OriginalAmount: existingAmount,
			TargetAmount:   targetAmount,
		}, nil
	case billing.LineOrHierarchyTypeHierarchy:
		return ProrateDecision{}, errors.New("flat fee lines cannot be reconciled against a split line hierarchy")
	default:
		return ProrateDecision{}, fmt.Errorf("unsupported line or hierarchy type: %s", existing.Type())
	}
}
