package consumer

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

func newNumericThreshold(v float64) notification.BalanceThreshold {
	return notification.BalanceThreshold{
		Value: v,
		Type:  api.NotificationRuleBalanceThresholdValueTypeNumber,
	}
}

func newPercentThreshold(v float64) notification.BalanceThreshold {
	return notification.BalanceThreshold{
		Value: v,
		Type:  api.NotificationRuleBalanceThresholdValueTypePercent,
	}
}

func TestGetHighestMatchingBalanceThreshold(t *testing.T) {
	tcs := []struct {
		Name              string
		BalanceThresholds []notification.BalanceThreshold
		EntitlementValue  snapshot.EntitlementValue
		Expect            *notification.BalanceThreshold
	}{
		{
			Name: "Numerical values only",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newNumericThreshold(10),
				newNumericThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(10.0),
				Usage:   lo.ToPtr(20.0),
			},
			// Already used 20, so the matching threshold is the 20
			Expect: lo.ToPtr(newNumericThreshold(20)),
		},
		{
			Name: "Numerical values only - 100%",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newNumericThreshold(10),
				newNumericThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(30.0),
			},
			Expect: lo.ToPtr(newNumericThreshold(30)),
		},
		{
			Name: "Numerical values only - 100%+ with overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newNumericThreshold(10),
				newNumericThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(30.0),
				Overage: lo.ToPtr(10.0),
			},
			Expect: lo.ToPtr(newNumericThreshold(30)),
		},
		{
			Name: "Percentages with overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newPercentThreshold(50),
				newPercentThreshold(100),
				newPercentThreshold(110),
				newPercentThreshold(120),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(110.0),
				Overage: lo.ToPtr(10.0),
			},
			Expect: lo.ToPtr(newPercentThreshold(110)),
		},
		{
			Name: "Mixed values",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newNumericThreshold(10),
				newNumericThreshold(30),
				newPercentThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(14.0),
				Usage:   lo.ToPtr(16.0),
			},
			Expect: lo.ToPtr(newPercentThreshold(50)),
		},
		// Corner cases
		{
			Name: "No grants",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
				newPercentThreshold(100),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(0.0),
			},
			Expect: nil,
		},
		{
			Name: "Last threshold is ",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(20),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(30.0),
			},
			Expect: lo.ToPtr(newNumericThreshold(20)),
		},
		{
			Name: "Same threshold in percentage and number",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(15),
				newPercentThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(14.0),
				Usage:   lo.ToPtr(16.0),
			},
			Expect: lo.ToPtr(newPercentThreshold(50)),
		},
		{
			Name: "Exact threshold match",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumericThreshold(15),
				newPercentThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(15.0),
				Usage:   lo.ToPtr(15.0),
			},
			Expect: lo.ToPtr(newPercentThreshold(50)),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			got, err := getHighestMatchingThreshold(tc.BalanceThresholds, tc.EntitlementValue)
			assert.NoError(t, err)
			assert.Equal(t, tc.Expect, got)
		})
	}
}
