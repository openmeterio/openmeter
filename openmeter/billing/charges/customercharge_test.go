package charges

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCustomerChargeGetResolvedCostBasis(t *testing.T) {
	fiat, err := currencyx.NewFiatCurrency(currencyx.Code("USD"))
	require.NoError(t, err)

	periodFrom := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	period := timeutil.ClosedPeriod{From: periodFrom, To: periodFrom.Add(time.Hour)}
	state := &costbasis.State{CostBasis: alpacadecimal.NewFromFloat(1.5), ResolvedAt: periodFrom}
	dynamic := lo.ToPtr(costbasis.NewIntent(costbasis.DynamicIntent{FiatCurrency: fiat}))
	manual := lo.ToPtr(costbasis.NewIntent(costbasis.ManualIntent{FiatCurrency: fiat, Rate: alpacadecimal.NewFromFloat(1.5)}))

	newFlatFee := func(intent *costbasis.Intent, state *costbasis.State) CustomerCharge {
		return CustomerCharge{Charge: NewCharge(flatfee.Charge{
			ChargeBase: flatfee.ChargeBase{
				State: flatfee.State{ResolvedCostBasis: state},
				Intent: flatfee.NewOverridableIntent(flatfee.Intent{
					IntentMutableFields: flatfee.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{ServicePeriod: period, FullServicePeriod: period},
					},
					CostBasis: intent,
				}, nil),
			},
		})}
	}

	newUsageBased := func(intent *costbasis.Intent, state *costbasis.State) CustomerCharge {
		return CustomerCharge{Charge: NewCharge(usagebased.Charge{
			ChargeBase: usagebased.ChargeBase{
				State: usagebased.State{ResolvedCostBasis: state},
				Intent: usagebased.NewOverridableIntent(usagebased.Intent{
					IntentMutableFields: usagebased.IntentMutableFields{
						IntentMutableFields: meta.IntentMutableFields{ServicePeriod: period, FullServicePeriod: period},
					},
					CostBasis: intent,
				}, nil),
			},
		})}
	}

	tests := []struct {
		name    string
		intent  *costbasis.Intent
		state   *costbasis.State
		now     time.Time
		visible bool
	}{
		{name: "no intent", now: periodFrom},
		{name: "unresolved", intent: dynamic, now: periodFrom},
		{name: "manual is visible before the service period", intent: manual, state: state, now: periodFrom.Add(-time.Hour), visible: true},
		{name: "dynamic is hidden before the service period", intent: dynamic, state: nil, now: periodFrom.Add(-time.Hour)},
		{name: "dynamic is visible from the service period start", intent: dynamic, state: state, now: periodFrom, visible: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clock.FreezeTime(tt.now)
			defer clock.UnFreeze()

			for name, charge := range map[string]CustomerCharge{
				"flat fee":    newFlatFee(tt.intent, tt.state),
				"usage based": newUsageBased(tt.intent, tt.state),
			} {
				got, err := charge.GetResolvedCostBasis()
				require.NoError(t, err, name)

				if tt.visible {
					require.Equal(t, state, got, name)
				} else {
					require.Nil(t, got, name)
				}
			}
		})
	}

	t.Run("credit purchase is unsupported", func(t *testing.T) {
		_, err := CustomerCharge{Charge: NewCharge(creditpurchase.Charge{})}.GetResolvedCostBasis()
		require.Error(t, err)
	})
}
