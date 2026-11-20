package subscription_test

import (
	"errors"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/featuregate"
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
		{
			Name: "Should be inactive after deletion",
			Sub: subscription.Subscription{
				ManagedModel: models.ManagedModel{
					DeletedAt: lo.ToPtr(testutils.GetRFC3339Time(t, "2020-01-01T00:00:02Z")),
				},
			},
			QueryTime: testutils.GetRFC3339Time(t, "2020-01-01T00:00:06Z"),
			Expected:  subscription.SubscriptionStatusInactive,
		},
		{
			Name: "Should be inactive after deletion after being scheduled",
			Sub: subscription.Subscription{
				CadencedModel: models.CadencedModel{
					ActiveFrom: testutils.GetRFC3339Time(t, "2020-01-01T00:00:00Z"),
				},
				ManagedModel: models.ManagedModel{
					DeletedAt: lo.ToPtr(testutils.GetRFC3339Time(t, "2020-01-01T00:00:02Z")),
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

func TestSubscriptionSpecBillingStartsAt(t *testing.T) {
	activeFrom := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	// A subscription billed on billingCadence from 2026-01-01.
	makeSpec := func(billingCadence string, billingStartsAt *time.Time) subscription.SubscriptionSpec {
		return subscription.SubscriptionSpec{
			CreateSubscriptionPlanInput: subscription.CreateSubscriptionPlanInput{
				BillingCadence: datetime.MustParseDuration(t, billingCadence),
			},
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
				InvoiceCurrency: "USD",
				ActiveFrom:      activeFrom,
				BillingAnchor:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				BillingStartsAt: billingStartsAt,
			},
		}
	}

	t.Run("billing cannot start before the subscription", func(t *testing.T) {
		spec := makeSpec("P1M", lo.ToPtr(activeFrom.Add(-time.Second)))

		require.ErrorIs(t, spec.Validate(), subscription.ErrSubscriptionBillingStartsAtBeforeSubscriptionStart)
	})

	t.Run("billing can start together with the subscription", func(t *testing.T) {
		spec := makeSpec("P1M", &activeFrom)

		require.NotErrorIs(t, spec.Validate(), subscription.ErrSubscriptionBillingStartsAtBeforeSubscriptionStart)
	})

	for _, tc := range []struct {
		name            string
		billingCadence  string
		billingStartsAt *time.Time
		aligned         bool
	}{
		{
			name:           "unset",
			billingCadence: "P1M",
			aligned:        true,
		},
		{
			name:            "start of a monthly billing period",
			billingCadence:  "P1M",
			billingStartsAt: lo.ToPtr(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)),
			aligned:         true,
		},
		{
			name:            "inside a monthly billing period",
			billingCadence:  "P1M",
			billingStartsAt: lo.ToPtr(time.Date(2026, 12, 17, 0, 0, 0, 0, time.UTC)),
			aligned:         false,
		},
		{
			name:            "start of a yearly billing period",
			billingCadence:  "P1Y",
			billingStartsAt: lo.ToPtr(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)),
			aligned:         true,
		},
		{
			// 2026-12-01 is the first of a month, but the yearly period runs 2026-01-01..2027-01-01.
			name:            "inside a yearly billing period",
			billingCadence:  "P1Y",
			billingStartsAt: lo.ToPtr(time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)),
			aligned:         false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := makeSpec(tc.billingCadence, tc.billingStartsAt)

			aligned, err := spec.IsBillingStartsAtAligned()
			require.NoError(t, err)
			require.Equal(t, tc.aligned, aligned)
		})
	}
}

type namespaceGate struct {
	enabledNamespace string
	err              error
}

func (g namespaceGate) EvaluateBool(namespace string, _ string, _ bool) (bool, error) {
	return namespace == g.enabledNamespace, g.err
}

func TestChargesConfigIsChargeBacked(t *testing.T) {
	gateFor := func(gate featuregate.Gate) *featuregate.FeatureGateChecker {
		return featuregate.NewFeatureGateChecker(
			gate,
			featuregate.Flags{featuregate.CtxKeyCredits: "credits"},
			map[featuregate.FeatureFlag]bool{featuregate.CtxKeyCredits: true},
		)
	}

	for _, tc := range []struct {
		name           string
		config         subscription.ChargesConfig
		namespace      string
		settlementMode productcatalog.SettlementMode
		expected       bool
	}{
		{
			name:           "credits disabled on the deployment",
			config:         subscription.ChargesConfig{},
			namespace:      "with-credits",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			expected:       false,
		},
		{
			name: "credits disabled for the namespace",
			config: subscription.ChargesConfig{
				Enabled:                 true,
				EnableCreditThenInvoice: true,
				FeatureGate:             gateFor(namespaceGate{enabledNamespace: "with-credits"}),
			},
			namespace:      "without-credits",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			expected:       false,
		},
		{
			name: "credit only",
			config: subscription.ChargesConfig{
				Enabled:     true,
				FeatureGate: gateFor(namespaceGate{enabledNamespace: "with-credits"}),
			},
			namespace:      "with-credits",
			settlementMode: productcatalog.CreditOnlySettlementMode,
			expected:       true,
		},
		{
			name: "credit then invoice stays on invoice lines by default",
			config: subscription.ChargesConfig{
				Enabled:     true,
				FeatureGate: gateFor(namespaceGate{enabledNamespace: "with-credits"}),
			},
			namespace:      "with-credits",
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			expected:       false,
		},
		{
			name: "credit then invoice moved to charges",
			config: subscription.ChargesConfig{
				Enabled:                 true,
				EnableCreditThenInvoice: true,
				FeatureGate:             gateFor(namespaceGate{enabledNamespace: "with-credits"}),
			},
			namespace:      "with-credits",
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			expected:       true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.config.Validate())

			chargeBacked, err := tc.config.IsChargeBacked(tc.namespace, tc.settlementMode)
			require.NoError(t, err)
			require.Equal(t, tc.expected, chargeBacked)
		})
	}

	t.Run("feature gate failure is reported", func(t *testing.T) {
		gateErr := errors.New("gate is down")
		config := subscription.ChargesConfig{
			Enabled:     true,
			FeatureGate: gateFor(namespaceGate{err: gateErr}),
		}

		_, err := config.IsChargeBacked("with-credits", productcatalog.CreditOnlySettlementMode)
		require.ErrorIs(t, err, gateErr)
	})

	t.Run("feature gate is required when credits are enabled", func(t *testing.T) {
		require.Error(t, subscription.ChargesConfig{Enabled: true}.Validate())
		require.Error(t, subscription.ChargesConfig{Enabled: true, FeatureGate: gateFor(nil)}.Validate())
		require.NoError(t, subscription.ChargesConfig{FeatureGate: gateFor(nil)}.Validate())
	})
}
