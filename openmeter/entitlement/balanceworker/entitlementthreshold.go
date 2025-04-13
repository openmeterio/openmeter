package balanceworker

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpacadecimal"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/balanceworker/negcache"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
)

type ThresholdProvider interface {
	GetNextActiveThresholdsFor(ctx context.Context, entitlement entitlement.Entitlement, lastCalculatedValue snapshot.EntitlementValue) (*alpacadecimal.Decimal, error)
}

func (w *Worker) hitsWatchedThresholds(ctx context.Context, entitlementEnt *negcache.EntitlementCached) (bool, error) {
	for _, provider := range w.thresholdProviders {
		nextThreshold, err := provider.GetNextActiveThresholdsFor(ctx, entitlementEnt.Target.Entitlement, entitlementEnt.LastCalculation)
		if err != nil {
			return false, fmt.Errorf("failed to get next active thresholds: %w", err)
		}

		if nextThreshold != nil {
			if entitlementEnt.ApproxUsage.GreaterThanOrEqual(negcache.NewInfDecimalFromDecimal(*nextThreshold)) {
				return true, nil
			}
		}
	}

	return false, nil
}
