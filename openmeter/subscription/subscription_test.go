package subscription_test

import (
	"testing"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSubscriptionStatus(t *testing.T) {
	tt := []struct {
		Name      string
		Sub       subscription.Subscription
		QueryTime time.Time
		Expected  subscription.SubscriptionStatus
	}{
		{
			Name:      "Should be inactive for zero sub",
			Sub:       subscription.Subscription{},
			QueryTime: testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z"),
			Expected:  subscription.SubscriptionStatusInactive,
		},
		{
			Name: "Should be active at start time",
			Sub: subscription.Subscription{
				CadencedModel: models.CadencedModel{
					ActiveFrom: testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z"),
				},
			},
			QueryTime: testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z"),
			Expected:  subscription.SubscriptionStatusActive,
		},
		{
			Name: "Should be active after start time",
			Sub: subscription.Subscription{
				CadencedModel: models.CadencedModel{
					ActiveFrom: testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z"),
				},
			},
			QueryTime: testutils.GetRFC3339Time(t, "2020-01-01T00:00:01Z"),
			Expected:  subscription.SubscriptionStatusActive,
		},
		{
			Name: "Should be canceled between start and end times",
			Sub: subscription.Subscription{
				CadencedModel: models.CadencedModel{
					ActiveFrom: testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z"),
					ActiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2020-01-01T00:00:05Z")),
				},
			},
			QueryTime: testutils.GetRFC3339Time(t, "2020-01-01T00:00:01Z"),
			Expected:  subscription.SubscriptionStatusCanceled,
		},
		{
			Name: "Should be inactive at end time",
			Sub: subscription.Subscription{
				CadencedModel: models.CadencedModel{
					ActiveFrom: testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z"),
					ActiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2020-01-01T00:00:05Z")),
				},
			},
			QueryTime: testutils.GetRFC3339Time(t, "2020-01-01T00:00:05Z"),
			Expected:  subscription.SubscriptionStatusInactive,
		},
		{
			Name: "Should be inactive after end time",
			Sub: subscription.Subscription{
				CadencedModel: models.CadencedModel{
					ActiveFrom: testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z"),
					ActiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2020-01-01T00:00:05Z")),
				},
			},
			QueryTime: testutils.GetRFC3339Time(t, "2020-01-01T00:00:06Z"),
			Expected:  subscription.SubscriptionStatusInactive,
		},
	}

	for _, tc := range tt {
		t.Run(tc.Name, func(t *testing.T) {
			actual := tc.Sub.GetStatusAt(tc.QueryTime)
			if actual != tc.Expected {
				t.Errorf("expected %v, got %v", tc.Expected, actual)
			}
		})
	}
}
