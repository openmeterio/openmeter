package reconciler

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

func TestNewChargesSnapshotCurrentPlanVersion(t *testing.T) {
	for _, tc := range []struct {
		name  string
		card  productcatalog.RateCard
		build func(targetstate.StateItem) (charges.ChargeIntent, error)
		plan  *subscription.PlanRef
	}{
		{"flat fee", newChargePatchTestFlatRateCard(), newFlatFeeChargeIntent, &subscription.PlanRef{Id: "plan-v2", Key: "pro", Version: 2}},
		{"flat fee without plan", newChargePatchTestFlatRateCard(), newFlatFeeChargeIntent, nil},
		{"usage based", newChargePatchTestUsageRateCard(), newUsageBasedChargeIntent, &subscription.PlanRef{Id: "plan-v2", Key: "pro", Version: 2}},
		{"usage based without plan", newChargePatchTestUsageRateCard(), newUsageBasedChargeIntent, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// given: the subscription has its current catalog reference, or is custom.
			target := newChargePatchTestTarget(t, productcatalog.CreditOnlySettlementMode, tc.card)
			target.Subscription.PlanRef = tc.plan
			// when: sync creates a charge, without resolving a historical plan version.
			intent, err := tc.build(target)
			require.NoError(t, err)
			var plan *chargesmeta.SubscriptionPlan
			if tc.name == "flat fee" || tc.name == "flat fee without plan" {
				value, err := intent.AsFlatFeeIntent()
				require.NoError(t, err)
				plan = value.SubscriptionPlan
			} else {
				value, err := intent.AsUsageBasedIntent()
				require.NoError(t, err)
				plan = value.SubscriptionPlan
			}
			// then: the stored attribution is the current plan version and is detached from later subscription changes.
			if tc.plan == nil {
				require.Nil(t, plan)
				return
			}
			require.Equal(t, &chargesmeta.SubscriptionPlan{Key: "pro", Version: 2}, plan)
			target.Subscription.PlanRef.Version = 99
			require.Equal(t, 2, plan.Version)
		})
	}
}
