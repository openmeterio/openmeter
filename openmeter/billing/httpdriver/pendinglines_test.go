package httpdriver

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
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
		handler     handler
		expected    bool
		expectedErr error
	}{
		{
			name: "no charge service uses billing",
			handler: handler{
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
			},
			expected: false,
		},
		{
			name: "credits disabled uses billing",
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
			name: "credit then invoice disabled uses billing",
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
			name: "enabled without feature flag uses charges",
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
			name: "feature flag disabled uses billing",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
					FeatureFlag:             "billing_credits",
				},
				featureGate: staticFeatureGate{enabled: false},
			},
			expected: false,
		},
		{
			name: "feature flag enabled uses charges",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
					FeatureFlag:             "billing_credits",
				},
				featureGate: staticFeatureGate{enabled: true},
			},
			expected: true,
		},
		{
			name: "feature gate error is returned",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
					FeatureFlag:             "billing_credits",
				},
				featureGate: staticFeatureGate{err: testErr},
			},
			expectedErr: testErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actual, err := tc.handler.shouldCreatePendingLinesWithCharges("ns")
			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.expected, actual)
		})
	}
}
