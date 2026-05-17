package consumer

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/entitlement/snapshot"
	"github.com/openmeterio/openmeter/openmeter/event/models"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/subject"
	pkgmodels "github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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

var validateEntitlementValue = func(t *testing.T, value snapshot.EntitlementValue) {
	t.Helper()

	var (
		balance              = lo.FromPtr(value.Balance)
		usage                = lo.FromPtr(value.Usage)
		overage              = lo.FromPtr(value.Overage)
		totalAvailableGrants = lo.FromPtr(value.TotalAvailableGrantAmount)
	)

	var errs []error

	if balance+usage != totalAvailableGrants {
		errs = append(errs, fmt.Errorf("balance + usage != totalAvailableGrants: %v != %v", balance+usage, totalAvailableGrants))
	}

	if overage > 0 && usage < totalAvailableGrants {
		errs = append(errs, fmt.Errorf("overage > 0 && usage < totalAvailableGrants: %v > 0 && %v < %v", overage, usage, totalAvailableGrants))
	}

	if overage > 0 && balance > 0 {
		errs = append(errs, fmt.Errorf("overage > 0 && balance > 0: %v > 0 && %v > 0", overage, balance))
	}

	if err := errors.Join(errs...); err != nil {
		require.NoErrorf(t, err, "invalid entitlement value: %v", errs)
	}
}

func Test_GetActiveThresholdsWithHighestPriority(t *testing.T) {
	tests := []struct {
		Name              string
		BalanceThresholds []notification.BalanceThreshold
		EntitlementValue  snapshot.EntitlementValue
		Expected          *activeThresholds
		ExpectedErr       error
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
				Balance:                   lo.ToPtr(5.0),
				Usage:                     lo.ToPtr(25.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
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
				Balance:                   lo.ToPtr(5.0),
				Usage:                     lo.ToPtr(25.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(20)),
				Balance: nil,
			},
		},
		{
			Name: "Usage values only - total usage over the total available grant amount",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(50),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(50.0),
				Overage:                   lo.ToPtr(50.0),
				TotalAvailableGrantAmount: lo.ToPtr(50.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(50)),
				Balance: nil,
			},
		},
		{
			Name: "Usage values only - with overage over the total available grant amount",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(100),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(50.0),
				Overage:                   lo.ToPtr(100.0),
				TotalAvailableGrantAmount: lo.ToPtr(50.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(100)),
				Balance: nil,
			},
		},
		{
			Name: "Number values only - 100% (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(35),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(35.0),
				TotalAvailableGrantAmount: lo.ToPtr(35.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: nil,
			},
		},
		{
			Name: "Usage values only - 100%",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(35),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(35.0),
				TotalAvailableGrantAmount: lo.ToPtr(35.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: nil,
			},
		},
		{
			Name: "Number values only - 100% with overage (deprecated)",
			BalanceThresholds: []notification.BalanceThreshold{
				newNumberThreshold(20),
				newNumberThreshold(10),
				newNumberThreshold(40),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(35.0),
				Overage:                   lo.ToPtr(10.0),
				TotalAvailableGrantAmount: lo.ToPtr(35.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newNumberThreshold(40)),
				Balance: nil,
			},
		},
		{
			Name: "Usage values only - 100% with overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(20),
				newUsageValueThreshold(10),
				newUsageValueThreshold(40),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(35.0),
				Overage:                   lo.ToPtr(10.0),
				TotalAvailableGrantAmount: lo.ToPtr(35.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(40)),
				Balance: nil,
			},
		},
		{
			Name: "Usage values only - no grant with overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(50),
				newUsageValueThreshold(100),
				newUsageValueThreshold(120),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(0.0),
				Overage:                   lo.ToPtr(110.0),
				TotalAvailableGrantAmount: lo.ToPtr(0.0),
			},
			Expected: &activeThresholds{
				Usage:   lo.ToPtr(newUsageValueThreshold(100)),
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
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(95.0),
				Overage:                   lo.ToPtr(15.0),
				TotalAvailableGrantAmount: lo.ToPtr(95.0),
			},
			Expected: &activeThresholds{
				// 50% of 95 = 47.5 < 110 usage, active
				// 100% of 95 = 95 < 110 usage, active
				// 110% of 95 = 104.5 < 110 usage, active (highest)
				// 120% of 95 = 114, NOT < 110 usage, not active
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
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(95.0),
				Overage:                   lo.ToPtr(15.0),
				TotalAvailableGrantAmount: lo.ToPtr(95.0),
			},
			Expected: &activeThresholds{
				// 50% of 95 = 47.5 < 110 usage, active
				// 100% of 95 = 95 < 110 usage, active
				// 110% of 95 = 104.5 < 110 usage, active (highest)
				// 120% of 95 = 114, NOT < 110 usage, not active
				Usage:   lo.ToPtr(newUsagePercentageThreshold(110)),
				Balance: nil,
			},
		},
		{
			Name: "Usage percentage with no grants",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsagePercentageThreshold(50),
				newUsagePercentageThreshold(100),
				newUsagePercentageThreshold(110),
				newUsagePercentageThreshold(120),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(0.0),
				Overage:                   lo.ToPtr(100.0),
				TotalAvailableGrantAmount: lo.ToPtr(0.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
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
				Balance:                   lo.ToPtr(14.0),
				Usage:                     lo.ToPtr(16.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				// number 20 < 16 usage = not active
				// number 10 < 16 usage = active (thresholdValue=10)
				// number 30 < 16 usage = not active
				// 50% of 30 = 15 < 16 usage = active (thresholdValue=15, highest)
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
				Balance:                   lo.ToPtr(14.0),
				Usage:                     lo.ToPtr(16.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				// usageValue 20 < 16 usage = not active
				// usageValue 10 < 16 usage = active (thresholdValue=10)
				// usageValue 30 < 16 usage = not active
				// 50% of 30 = 15 < 16 usage = active (thresholdValue=15, highest)
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
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(30.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
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
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(30.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
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
				Balance:                   lo.ToPtr(14.0),
				Usage:                     lo.ToPtr(16.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				// number 15 < 16 usage = active (thresholdValue=15)
				// 50% of 30 = 15 < 16 usage = active (thresholdValue=15, replaces via <=)
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
				Balance:                   lo.ToPtr(14.0),
				Usage:                     lo.ToPtr(16.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				// usageValue 15 < 16 usage = active (thresholdValue=15)
				// 50% of 30 = 15 < 16 usage = active (thresholdValue=15, replaces via <=)
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
				Balance:                   lo.ToPtr(15.0),
				Usage:                     lo.ToPtr(15.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				// number 15 < 15 usage = not active (strict <)
				// 50% of 30 = 15 < 15 usage = not active (strict <)
				Usage:   nil,
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
				Balance:                   lo.ToPtr(14.0),
				Usage:                     lo.ToPtr(16.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				// usageValue 15 < 16 usage = active (thresholdValue=15)
				// 50% (deprecated) of 30 = 15 < 16 usage = active (thresholdValue=15, replaces via <=)
				// 50% (usagePercentage) of 30 = 15 < 16 usage = active (thresholdValue=15, replaces via <=)
				Usage:   lo.ToPtr(newUsagePercentageThreshold(50)),
				Balance: nil,
			},
		},
		// Balance value
		{
			Name: "Exact balance threshold",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(20),
				newBalanceValueThreshold(10),
				newBalanceValueThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(15.0),
				Usage:                     lo.ToPtr(15.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: lo.ToPtr(newBalanceValueThreshold(20)),
			},
		},
		{
			Name: "Non-zero balance thresholds value",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(20),
				newBalanceValueThreshold(10),
				newBalanceValueThreshold(30),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(25.0),
				Usage:                     lo.ToPtr(0.0),
				TotalAvailableGrantAmount: lo.ToPtr(25.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: lo.ToPtr(newBalanceValueThreshold(30)),
			},
		},
		{
			Name: "Zero balance threshold value",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(20),
				newBalanceValueThreshold(10),
				newBalanceValueThreshold(0),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(30.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: lo.ToPtr(newBalanceValueThreshold(0)),
			},
		},
		{
			Name: "Mixed balance and usage value thresholds",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(10),
				newUsageValueThreshold(20),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(5.0),
				Usage:                     lo.ToPtr(25.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
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
				Balance:                   lo.ToPtr(5.0),
				Usage:                     lo.ToPtr(95.0),
				TotalAvailableGrantAmount: lo.ToPtr(100.0),
			},
			Expected: &activeThresholds{
				// 90% of 100 = 90 < 95 usage = active
				Usage:   lo.ToPtr(newUsagePercentageThreshold(90)),
				Balance: lo.ToPtr(newBalanceValueThreshold(10)),
			},
		},
		{
			Name: "No balance available",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(10),
				newUsagePercentageThreshold(90),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance: lo.ToPtr(0.0),
				Usage:   lo.ToPtr(0.0),
				Overage: lo.ToPtr(0.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: nil,
			},
		},
		// Edge cases
		{
			Name:              "Empty thresholds list",
			BalanceThresholds: []notification.BalanceThreshold{},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(5.0),
				Usage:                     lo.ToPtr(25.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: nil,
			},
		},
		{
			Name: "Exact non-zero balance threshold match is not active",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(15),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(15.0),
				Usage:                     lo.ToPtr(15.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: nil,
			},
		},
		{
			Name: "All three threshold kinds mixed",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(10),
				newUsageValueThreshold(20),
				newUsagePercentageThreshold(80),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(5.0),
				Usage:                     lo.ToPtr(25.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				// usageValue 20 < 25 usage = active (thresholdValue=20)
				// 80% of 30 = 24 < 25 usage = active (thresholdValue=24, highest)
				Usage:   lo.ToPtr(newUsagePercentageThreshold(80)),
				Balance: lo.ToPtr(newBalanceValueThreshold(10)),
			},
		},
		{
			Name: "No active thresholds when usage is below all thresholds",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsageValueThreshold(50),
				newUsageValueThreshold(100),
				newUsagePercentageThreshold(90),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(90.0),
				Usage:                     lo.ToPtr(10.0),
				TotalAvailableGrantAmount: lo.ToPtr(100.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: nil,
			},
		},
		{
			Name: "Usage percentage over 100% not active without overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newUsagePercentageThreshold(50),
				newUsagePercentageThreshold(100),
				newUsagePercentageThreshold(150),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(10.0),
				Usage:                     lo.ToPtr(90.0),
				TotalAvailableGrantAmount: lo.ToPtr(100.0),
			},
			Expected: &activeThresholds{
				// 50% of 100 = 50, which is < 90 usage, so active
				// 100% of 100 = 100, which is NOT < 90, so not active
				// 150% of 100 = 150, which is NOT < 90, so not active
				Usage:   lo.ToPtr(newUsagePercentageThreshold(50)),
				Balance: nil,
			},
		},
		{
			Name: "Balance threshold with overage",
			BalanceThresholds: []notification.BalanceThreshold{
				newBalanceValueThreshold(20),
				newBalanceValueThreshold(10),
				newBalanceValueThreshold(5),
			},
			EntitlementValue: snapshot.EntitlementValue{
				Balance:                   lo.ToPtr(0.0),
				Usage:                     lo.ToPtr(30.0),
				Overage:                   lo.ToPtr(10.0),
				TotalAvailableGrantAmount: lo.ToPtr(30.0),
			},
			Expected: &activeThresholds{
				Usage:   nil,
				Balance: lo.ToPtr(newBalanceValueThreshold(5)),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			// Make sure we test valid entitlement values
			validateEntitlementValue(t, test.EntitlementValue)

			actual, err := getActiveThresholdsWithHighestPriority(test.BalanceThresholds, test.EntitlementValue)
			if test.ExpectedErr == nil {
				assert.NoErrorf(t, err, "must not return an error: %s", err)
				assert.Equalf(t, test.Expected, actual, "must be equal")
			} else {
				assert.ErrorIsf(t, err, test.ExpectedErr, "must return the expected error: %s", err)
			}
		})
	}
}

func MustParseISOTime(t *testing.T, str string) time.Time {
	t.Helper()

	dt, err := time.Parse(time.RFC3339, str)
	if err != nil {
		t.Fatalf("failed to parse time: %s", err)
	}

	return dt
}

func Test_BalanceEventDedupHash(t *testing.T) {
	tests := []struct {
		Name      string
		Snapshot  snapshot.SnapshotEvent
		RuleID    string
		Threshold notification.BalanceThreshold

		ExpectedV1 string
		ExpectedV2 string
	}{
		{
			Name: "Legacy balance threshold",
			Snapshot: snapshot.SnapshotEvent{
				Entitlement: entitlement.Entitlement{
					GenericProperties: entitlement.GenericProperties{
						NamespacedModel: pkgmodels.NamespacedModel{
							Namespace: "",
						},
						ManagedModel: pkgmodels.ManagedModel{
							CreatedAt: MustParseISOTime(t, "2025-08-25T15:32:54.417461Z"),
							UpdatedAt: MustParseISOTime(t, "2025-08-25T15:32:54.417461Z"),
							DeletedAt: nil,
						},
						MetadataModel: pkgmodels.MetadataModel{
							Metadata: nil,
						},
						Annotations:     nil,
						ActiveFrom:      lo.ToPtr(MustParseISOTime(t, "2025-08-25T15:32:54.417461Z")),
						ActiveTo:        lo.ToPtr(MustParseISOTime(t, "2025-08-25T15:32:54.417461Z")),
						ID:              "01K3GXRSGHFABJ10SDK83PMSBS",
						FeatureID:       "01K3GXRC9AEDQP27NYXTB42FFJ",
						FeatureKey:      "api_requests_total",
						EntitlementType: entitlement.EntitlementTypeMetered,
						UsagePeriod:     &entitlement.UsagePeriod{},
						CurrentUsagePeriod: &timeutil.ClosedPeriod{
							From: MustParseISOTime(t, "2025-08-25T15:32:00Z"),
							To:   MustParseISOTime(t, "2025-09-25T15:32:00Z"),
						},
						OriginalUsagePeriodAnchor: nil,
					},
					MeasureUsageFrom:        lo.ToPtr(MustParseISOTime(t, "2025-08-25T17:32:00+02:00")),
					IssueAfterReset:         lo.ToPtr(0.0),
					IssueAfterResetPriority: nil,
					IsSoftLimit:             nil,
					LastReset:               lo.ToPtr(MustParseISOTime(t, "2025-08-25T15:32:00Z")),
					PreserveOverageAtReset:  nil,
					Config:                  nil,
				},
				Namespace: models.NamespaceID{
					ID: "test-namespace",
				},
				Subject: subject.Subject{
					Namespace:        "test-namespace",
					Id:               "01K3GXSQMXW2YXNFBV1783K53Q",
					Key:              "test",
					DisplayName:      nil,
					Metadata:         nil,
					StripeCustomerId: nil,
				},
				Feature: feature.Feature{
					Namespace:           "",
					ID:                  "01K3GXRC9AEDQP27NYXTB42FFJ",
					Name:                "API requests total",
					Key:                 "api_requests_total",
					MeterID:             lo.ToPtr("01K3GXRC9AEDQP27NYXTB42FFJ"),
					MeterGroupByFilters: nil,
					Metadata:            nil,
					ArchivedAt:          nil,
					CreatedAt:           MustParseISOTime(t, "2025-08-25T15:32:40.874442Z"),
					UpdatedAt:           MustParseISOTime(t, "2025-08-25T15:32:40.874443Z"),
				},
				Operation:    snapshot.ValueOperationUpdate,
				CalculatedAt: nil,
				Value: lo.ToPtr(snapshot.EntitlementValue{
					Balance:   lo.ToPtr(10.0),
					Config:    nil,
					HasAccess: true,
					Overage:   lo.ToPtr(10.0),
					Usage:     lo.ToPtr(90.0),
				}),
				CurrentUsagePeriod: &timeutil.ClosedPeriod{
					From: MustParseISOTime(t, "2025-08-25T15:32:00Z"),
					To:   MustParseISOTime(t, "2021-09-29T12:00:00Z"),
				},
			},
			RuleID:     "01K3K3JFZ9M36YD287YGAMZX5X",
			Threshold:  newNumberThreshold(10),
			ExpectedV1: "bsnap_v1_b8a36575f7f70393b3b23a032b7acf17202f767f11f061bb17815fd04e7ca24a",
			ExpectedV2: "bsnap_v2_521777eff8de3eb39301c492ff75620e",
		},
		{
			Name: "Balance threshold",
			Snapshot: snapshot.SnapshotEvent{
				Entitlement: entitlement.Entitlement{
					GenericProperties: entitlement.GenericProperties{
						NamespacedModel: pkgmodels.NamespacedModel{
							Namespace: "",
						},
						ManagedModel: pkgmodels.ManagedModel{
							CreatedAt: MustParseISOTime(t, "2025-08-25T15:32:54.417461Z"),
							UpdatedAt: MustParseISOTime(t, "2025-08-25T15:32:54.417461Z"),
							DeletedAt: nil,
						},
						MetadataModel: pkgmodels.MetadataModel{
							Metadata: nil,
						},
						Annotations:     nil,
						ActiveFrom:      lo.ToPtr(MustParseISOTime(t, "2025-08-25T15:32:54.417461Z")),
						ActiveTo:        lo.ToPtr(MustParseISOTime(t, "2025-08-25T15:32:54.417461Z")),
						ID:              "01K3GXRSGHFABJ10SDK83PMSBS",
						FeatureID:       "01K3GXRC9AEDQP27NYXTB42FFJ",
						FeatureKey:      "api_requests_total",
						EntitlementType: entitlement.EntitlementTypeMetered,
						UsagePeriod:     &entitlement.UsagePeriod{},
						CurrentUsagePeriod: &timeutil.ClosedPeriod{
							From: MustParseISOTime(t, "2025-08-25T15:32:00Z"),
							To:   MustParseISOTime(t, "2025-09-25T15:32:00Z"),
						},
						OriginalUsagePeriodAnchor: nil,
					},
					MeasureUsageFrom:        lo.ToPtr(MustParseISOTime(t, "2025-08-25T17:32:00+02:00")),
					IssueAfterReset:         lo.ToPtr(0.0),
					IssueAfterResetPriority: nil,
					IsSoftLimit:             nil,
					LastReset:               lo.ToPtr(MustParseISOTime(t, "2025-08-25T15:32:00Z")),
					PreserveOverageAtReset:  nil,
					Config:                  nil,
				},
				Namespace: models.NamespaceID{
					ID: "test-namespace",
				},
				Subject: subject.Subject{
					Namespace:        "test-namespace",
					Id:               "01K3GXSQMXW2YXNFBV1783K53Q",
					Key:              "test",
					DisplayName:      nil,
					Metadata:         nil,
					StripeCustomerId: nil,
				},
				Feature: feature.Feature{
					Namespace:           "",
					ID:                  "01K3GXRC9AEDQP27NYXTB42FFJ",
					Name:                "API requests total",
					Key:                 "api_requests_total",
					MeterID:             lo.ToPtr("01K3GXRC9AEDQP27NYXTB42FFJ"),
					MeterGroupByFilters: nil,
					Metadata:            nil,
					ArchivedAt:          nil,
					CreatedAt:           MustParseISOTime(t, "2025-08-25T15:32:40.874442Z"),
					UpdatedAt:           MustParseISOTime(t, "2025-08-25T15:32:40.874443Z"),
				},
				Operation:    snapshot.ValueOperationUpdate,
				CalculatedAt: nil,
				Value: lo.ToPtr(snapshot.EntitlementValue{
					Balance:   lo.ToPtr(10.0),
					Config:    nil,
					HasAccess: true,
					Overage:   lo.ToPtr(10.0),
					Usage:     lo.ToPtr(90.0),
				}),
				CurrentUsagePeriod: &timeutil.ClosedPeriod{
					From: MustParseISOTime(t, "2025-08-25T15:32:00Z"),
					To:   MustParseISOTime(t, "2021-09-29T12:00:00Z"),
				},
			},
			RuleID:     "01K3K3JFZ9M36YD287YGAMZX5X",
			Threshold:  newUsageValueThreshold(100.0),
			ExpectedV1: "bsnap_v1_b8a36575f7f70393b3b23a032b7acf17202f767f11f061bb17815fd04e7ca24a",
			ExpectedV2: "bsnap_v2_521777eff8de3eb39301c492ff75620e",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			h, err := NewBalanceEventDedupHash(test.Snapshot, test.RuleID, test.Threshold)
			assert.NoErrorf(t, err, "must not return an error: %s", err)

			assert.Equalf(t, test.ExpectedV1, h.V1(), "generated v1 version must be equal")
			assert.Equalf(t, test.ExpectedV2, h.V2(), "generated v2 version must be equal")
		})
	}
}
