package httpdriver

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/featuregate"
)

type nonNilChargeService struct {
	billingcharges.ChargeService
}

type staticFeatureGate struct {
	enabled bool
	err     error
}

func (g staticFeatureGate) EvaluateBool(string, string, bool) (bool, error) {
	return g.enabled, g.err
}

func TestShouldCreatePendingLinesWithCharges(t *testing.T) {
	t.Parallel()

	testErr := errors.New("feature gate failed")

	testCases := []struct {
		name        string
		namespace   string
		handler     handler
		expected    bool
		expectedErr error
	}{
		{
			name:      "no charge service uses billing",
			namespace: "ns-no-charge-service",
			handler: handler{
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
			},
			expected: false,
		},
		{
			name:      "credits disabled uses billing",
			namespace: "ns-credits-disabled",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 false,
					EnableCreditThenInvoice: true,
				},
			},
			expected: false,
		},
		{
			name:      "credit then invoice disabled uses billing",
			namespace: "ns-credit-then-invoice-disabled",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: false,
				},
			},
			expected: false,
		},
		{
			name:      "enabled without feature flag uses charges",
			namespace: "ns-empty-feature-flag",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
				featureGate: featuregate.NewFeatureGateChecker(staticFeatureGate{enabled: false}, featuregate.Flags{}, nil),
			},
			expected: true,
		},
		{
			name:      "nil feature gate uses charges",
			namespace: "ns-nil-feature-gate",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
			},
			expected: true,
		},
		{
			name:      "feature flag disabled uses billing",
			namespace: "ns-feature-flag-disabled",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
				featureGate: featuregate.NewFeatureGateChecker(staticFeatureGate{enabled: false}, featuregate.Flags{
					featuregate.CtxKeyCredits: "billing_credits_disabled",
				}, nil),
			},
			expected: false,
		},
		{
			name:      "feature flag enabled uses charges",
			namespace: "ns-feature-flag-enabled",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
				featureGate: featuregate.NewFeatureGateChecker(staticFeatureGate{enabled: true}, featuregate.Flags{
					featuregate.CtxKeyCredits: "billing_credits_enabled",
				}, nil),
			},
			expected: true,
		},
		{
			name:      "feature gate error is returned",
			namespace: "ns-feature-gate-error",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
				featureGate: featuregate.NewFeatureGateChecker(staticFeatureGate{err: testErr}, featuregate.Flags{
					featuregate.CtxKeyCredits: "billing_credits_error",
				}, nil),
			},
			expectedErr: testErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := tc.handler.shouldCreatePendingLinesWithCharges(tc.namespace)
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
