package consumer

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/notification"
)

func newNumberThreshold(v float64) notification.BalanceThreshold {
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

func newUsageValueThreshold(v float64) notification.BalanceThreshold {
	return notification.BalanceThreshold{
		Value: v,
		Type:  api.NotificationRuleBalanceThresholdValueTypeUsageValue,
	}
}

func newUsagePercentageThreshold(v float64) notification.BalanceThreshold {
	return notification.BalanceThreshold{
		Value: v,
		Type:  api.NotificationRuleBalanceThresholdValueTypeUsagePercentage,
	}
}

func newBalanceValueThreshold(v float64) notification.BalanceThreshold {
	return notification.BalanceThreshold{
		Value: v,
		Type:  api.NotificationRuleBalanceThresholdValueTypeBalanceValue,
	}
}

func Test_GetActiveThresholdsWithHighestPriority(t *testing.T) {
	tests := []struct {
		Name              string
		BalanceThresholds []notification.BalanceThreshold
		EntitlementValue  snapshot.EntitlementValue
		Expected          *activeThresholds
	}{
		// Usage value and percentage thresholds
		{
			Name: "Number values only (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(20),
				newNumberThreshold(10),
				newNumberThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(10.0),
				Usage:   lo.ToPtr(20.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newNumberThreshold(20)),
				Balance: nil,
			},
		},
		{
			Name: "Usage values only",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(20),
				newUsageValueThreshold(10),
				newUsageValueThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(10.0),
				Usage:   lo.ToPtr(20.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(20)),
				Balance: nil,
			},
		},
		{
			Name: "Number values only - 100% (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(20),
				newNumberThreshold(10),
				newNumberThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newNumberThreshold(30)),
				Balance: nil,
			},
		},
		{
			Name: "Usage values only - 100%",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(20),
				newUsageValueThreshold(10),
				newUsageValueThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(30)),
				Balance: nil,
			},
		},
		{
			Name: "Number values only - 100% with overage (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(20),
				newNumberThreshold(10),
				newNumberThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(30.0),
				Overage: lo.ToPtr(10.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newNumberThreshold(30)),
				Balance: nil,
			},
		},
		{
			Name: "Usage values only - 100% with overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(20),
				newUsageValueThreshold(10),
				newUsageValueThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(30.0),
				Overage: lo.ToPtr(10.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(30)),
				Balance: nil,
			},
		},
		{
			Name: "Percentage values with overage (deprecated)",
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
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newPercentThreshold(110)),
				Balance: nil,
			},
		},
		{
			Name: "Usage percentage with overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsagePercentageThreshold(50),
				newUsagePercentageThreshold(100),
				newUsagePercentageThreshold(110),
				newUsagePercentageThreshold(120),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(110.0),
				Overage: lo.ToPtr(10.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsagePercentageThreshold(110)),
				Balance: nil,
			},
		},
		{
			Name: "Mixed number and percentage thresholds (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(20),
				newNumberThreshold(10),
				newNumberThreshold(30),
				newPercentThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(14.0),
				Usage:   lo.ToPtr(16.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newPercentThreshold(50)),
				Balance: nil,
			},
		},
		{
			Name: "Mixed number and percentage thresholds",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(20),
				newUsageValueThreshold(10),
				newUsageValueThreshold(30),
				newUsagePercentageThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(14.0),
				Usage:   lo.ToPtr(16.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsagePercentageThreshold(50)),
				Balance: nil,
			},
		},
		// Usage value and percentage thresholds: edge cases
		{
			Name: "No grants (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(20),
				newPercentThreshold(100),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(0.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: nil,
			},
		},
		{
			Name: "No grants",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(20),
				newUsagePercentageThreshold(100),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(0.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: nil,
			},
		},
		{
			Name: "Last threshold",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(20),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newNumberThreshold(20)),
				Balance: nil,
			},
		},
		{
			Name: "Last threshold (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(20),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(20)),
				Balance: nil,
			},
		},
		{
			Name: "Same threshold in percentage and number (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(15),
				newPercentThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(14.0),
				Usage:   lo.ToPtr(16.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newPercentThreshold(50)),
				Balance: nil,
			},
		},
		{
			Name: "Same threshold in usage percentage and value",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(15),
				newUsagePercentageThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(14.0),
				Usage:   lo.ToPtr(16.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsagePercentageThreshold(50)),
				Balance: nil,
			},
		},
		{
			Name: "Exact threshold match (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(15),
				newPercentThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(15.0),
				Usage:   lo.ToPtr(15.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newPercentThreshold(50)),
				Balance: nil,
			},
		},
		{
			Name: "Exact threshold match",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(15),
				newPercentThreshold(50),
				newUsagePercentageThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(15.0),
				Usage:   lo.ToPtr(15.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsagePercentageThreshold(50)),
				Balance: nil,
			},
		},
		// Balance value
		{
			Name: "Balance values only",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(20),
				newBalanceValueThreshold(10),
				newBalanceValueThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(20.0),
				Usage:   lo.ToPtr(10.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: lo.ToPtr(newBalanceValueThreshold(20)),
			},
		},
		{
			Name: "Balance values only - 100%",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(20),
				newBalanceValueThreshold(10),
				newBalanceValueThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(30.0),
				Usage:   lo.ToPtr(0.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: lo.ToPtr(newBalanceValueThreshold(30)),
			},
		},
		{
			Name: "Mixed balance and usage value thresholds",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(10),
				newUsageValueThreshold(20),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(10.0),
				Usage:   lo.ToPtr(20.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(20)),
				Balance: lo.ToPtr(newBalanceValueThreshold(10)),
			},
		},
		{
			Name: "Mixed balance and usage percentage thresholds",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(10),
				newUsagePercentageThreshold(90),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(10.0),
				Usage:   lo.ToPtr(90.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsagePercentageThreshold(90)),
				Balance: lo.ToPtr(newBalanceValueThreshold(10)),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			actual, err := getActiveThresholdsWithHighestPriority(test.BalanceThresholds, test.EntitlementValue)
			assert.NoErrorf(t, err, "must not return an error: %s", err)
			assert.Equalf(t, test.Expected, actual, "must be equal")
		})
	}
}
