package service

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestValidateSubscriptionUsesFiatOnly(t *testing.T) {
	customCurrency := currencyx.Code("CREDITS")

	newSpec := func(subscriptionCurrency currencyx.Code, rateCardCurrency currencyx.CurrencyIdentity) subscription.SubscriptionSpec {
		rateCard := &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:      "fee",
				Name:     "Fee",
				Currency: rateCardCurrency,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount: alpacadecimal.NewFromInt(1),
				}),
			},
		}
		item := &subscription.SubscriptionItemSpec{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: "default",
					ItemKey:  "fee",
					RateCard: rateCard,
				},
			},
		}

		return subscription.SubscriptionSpec{
			CreateSubscriptionCustomerInput: subscription.CreateSubscriptionCustomerInput{
				Currency: subscriptionCurrency,
			},
			Phases: map[string]*subscription.SubscriptionPhaseSpec{
				"default": {
					ItemsByKey: map[string][]*subscription.SubscriptionItemSpec{
						"fee": {item},
					},
				},
			},
		}
	}

	tests := []struct {
		name     string
		spec     subscription.SubscriptionSpec
		expected error
	}{
		{
			name: "fiat subscription with inherited currency",
			spec: newSpec(currencyx.Code("USD"), nil),
		},
		{
			name:     "custom subscription currency",
			spec:     newSpec(customCurrency, nil),
			expected: errCustomCurrencySubscriptionsNotSupported,
		},
		{
			name:     "fiat subscription with custom item currency",
			spec:     newSpec(currencyx.Code("USD"), customCurrency),
			expected: errCustomCurrencySubscriptionsNotSupported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - a subscription spec using only fiat or introducing a custom currency
			// when:
			// - the temporary subscription boundary is validated
			// then:
			// - custom currencies are rejected before persistence or billing sync
			err := validateSubscriptionUsesFiatOnly(tt.spec)
			if tt.expected == nil {
				require.NoError(t, err)
				return
			}

			require.ErrorIs(t, err, tt.expected)
		})
	}
}
