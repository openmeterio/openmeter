package reconciler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription/planhistory"
)

func TestNewChargesSnapshotEffectivePlanVersion(t *testing.T) {
	for _, tc := range []struct {
		name  string
		card  productcatalog.RateCard
		build func(targetstate.StateItem) (charges.ChargeIntent, error)
	}{
		{"flat fee", newChargePatchTestFlatRateCard(), newFlatFeeChargeIntent},
		{"usage based", newChargePatchTestUsageRateCard(), newUsageBasedChargeIntent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: a migration takes effect after this charge's service period starts.
			target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, tc.card)
			start := target.GetServicePeriod().From
			target.Subscription.PlanHistory = planhistory.History{
				{EffectiveAt: start.Add(-time.Hour), Plan: planhistory.PlanVersion{Key: "pro", Version: 1}},
				{EffectiveAt: start.Add(time.Hour), Plan: planhistory.PlanVersion{Key: "pro", Version: 2}},
			}
			// when: sync creates the charge from the subscription with a scheduled migration.
			intent, err := tc.build(target)
			require.NoError(t, err)
			var plan *planhistory.PlanVersion
			if tc.name == "flat fee" {
				value, err := intent.AsFlatFeeIntent()
				require.NoError(t, err)
				plan = value.SubscriptionPlan
			} else {
				value, err := intent.AsUsageBasedIntent()
				require.NoError(t, err)
				plan = value.SubscriptionPlan
			}
			// then: the stored attribution is plan version 1 and is detached from later subscription changes.
			require.NotNil(t, plan)
			require.Equal(t, 1, plan.Version)
			target.Subscription.PlanHistory[0].Plan.Version = 99
			require.Equal(t, 1, plan.Version)
		})
	}
}
