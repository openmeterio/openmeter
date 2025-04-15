package balanceworker

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker/estimator"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
)

type ThresholdProvider interface {
	GetNextActiveThresholdsFor(ctx context.Context, entitlement entitlement.Entitlement, lastCalculatedValue snapshot.EntitlementValue) (*alpacadecimal.Decimal, error)
}

func (w *Worker) hitsWatchedThresholds(ctx context.Context, ent entitlement.Entitlement, entitlementEnt estimator.EntitlementCached) (bool, error) {
	for _, provider := range w.estimator.thresholdProviders {
		nextThreshold, err := provider.GetNextActiveThresholdsFor(ctx, ent, entitlementEnt.LastCalculation)
		if err != nil {
			return false, fmt.Errorf("failed to get next active thresholds: %w", err)
		}

		// TODO: Let's check if overage / usage is correctly handled in the code (e.g. > 100% notifications etc)

		if nextThreshold != nil {
			if entitlementEnt.ApproxUsage.GreaterThanOrEqual(estimator.NewInfDecimalFromDecimal(*nextThreshold)) {
				return true, nil
			}
		}
	}

	return false, nil
}
