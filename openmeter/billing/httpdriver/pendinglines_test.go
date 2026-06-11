package httpdriver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/app/config"
	billingcharges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/pkg/featuregate"
)

type nonNilChargeService struct {
	billingcharges.ChargeService
}

func TestShouldCreatePendingLinesWithCharges(t *testing.T) {
	t.Parallel()

	ctxWithCreditsFlag := func(enabled bool) context.Context {
		return context.WithValue(t.Context(), featuregate.CtxKeyCredits, enabled)
	}

	testCases := []struct {
		name     string
		handler  handler
		ctx      context.Context
		expected bool
	}{
		{
			name: "no charge service uses billing",
			handler: handler{
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
			},
			ctx:      ctxWithCreditsFlag(true),
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
			ctx:      ctxWithCreditsFlag(true),
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
			ctx:      ctxWithCreditsFlag(true),
			expected: false,
		},
		{
			name: "feature flag enabled uses charges",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
			},
			ctx:      ctxWithCreditsFlag(true),
			expected: true,
		},
		{
			name: "feature flag disabled uses billing",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
			},
			ctx:      ctxWithCreditsFlag(false),
			expected: false,
		},
		{
			name: "missing feature flag resolution defaults to charges",
			handler: handler{
				chargeService: nonNilChargeService{},
				credits: config.CreditsConfiguration{
					Enabled:                 true,
					EnableCreditThenInvoice: true,
				},
			},
			ctx:      t.Context(),
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.expected, tc.handler.shouldCreatePendingLinesWithCharges(tc.ctx))
		})
	}
}
