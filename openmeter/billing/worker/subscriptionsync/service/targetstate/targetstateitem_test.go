package targetstate

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestStateItemGetExpectedLineValidatesFeatureReference(t *testing.T) {
	featureID := "feature-id"
	featureKey := "feature-key"
	emptyFeatureKey := ""

	tests := []struct {
		name            string
		priceType       productcatalog.PriceType
		feature         *productcatalog.FeatureReference
		wantError       string
		wantErrorDetail string
		wantFeatureKey  string
	}{
		{name: "usage based without feature", priceType: productcatalog.UnitPriceType, wantError: "feature is required for usage-based price"},
		{
			name:            "usage based with empty reference",
			priceType:       productcatalog.UnitPriceType,
			feature:         &productcatalog.FeatureReference{},
			wantError:       "invalid feature reference for billing line",
			wantErrorDetail: "id or key is required",
		},
		{
			name:      "usage based with id only",
			priceType: productcatalog.UnitPriceType,
			feature:   productcatalog.NewFeatureReference(&featureID, nil),
			wantError: "feature key is required for billing line",
		},
		{
			name:            "usage based with empty key",
			priceType:       productcatalog.UnitPriceType,
			feature:         productcatalog.NewFeatureReference(nil, &emptyFeatureKey),
			wantError:       "invalid feature reference for billing line",
			wantErrorDetail: "key cannot be empty",
		},
		{
			name:           "usage based with key",
			priceType:      productcatalog.UnitPriceType,
			feature:        productcatalog.NewFeatureReference(nil, &featureKey),
			wantFeatureKey: featureKey,
		},
		{name: "flat fee without feature", priceType: productcatalog.FlatPriceType},
		{
			name:            "flat fee with empty reference",
			priceType:       productcatalog.FlatPriceType,
			feature:         &productcatalog.FeatureReference{},
			wantError:       "invalid feature reference for billing line",
			wantErrorDetail: "id or key is required",
		},
		{
			name:      "flat fee with id only",
			priceType: productcatalog.FlatPriceType,
			feature:   productcatalog.NewFeatureReference(&featureID, nil),
			wantError: "feature key is required for billing line",
		},
		{
			name:            "flat fee with empty key",
			priceType:       productcatalog.FlatPriceType,
			feature:         productcatalog.NewFeatureReference(nil, &emptyFeatureKey),
			wantError:       "invalid feature reference for billing line",
			wantErrorDetail: "key cannot be empty",
		},
		{
			name:           "flat fee with key",
			priceType:      productcatalog.FlatPriceType,
			feature:        productcatalog.NewFeatureReference(nil, &featureKey),
			wantFeatureKey: featureKey,
		},
	}

	fiatCurrency, err := currencies.NewFiatCurrency(currencyx.Code(currency.USD))
	require.NoError(t, err)

	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
	}
	billingCadence := datetime.MustParseDuration(t, "P1M")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a subscription item with the tested price and feature identity.
			rateCard := newExpectedLineTestRateCard(t, tt.priceType, tt.feature, billingCadence)
			item := newExpectedLineTestStateItem(t, fiatCurrency, period, rateCard)

			// When subscription sync builds its expected gathering line.
			line, err := item.GetExpectedLine()

			// Then supplied references are valid and representable by a feature key.
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				if tt.wantErrorDetail != "" {
					require.ErrorContains(t, err, tt.wantErrorDetail)
				}
				require.Nil(t, line)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, line)
			require.Equal(t, tt.wantFeatureKey, line.FeatureKey)
		})
	}
}

func newExpectedLineTestRateCard(t *testing.T, priceType productcatalog.PriceType, feature *productcatalog.FeatureReference, billingCadence datetime.ISODuration) productcatalog.RateCard {
	t.Helper()

	meta := productcatalog.RateCardMeta{Feature: feature}

	switch priceType {
	case productcatalog.FlatPriceType:
		meta.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{Amount: alpacadecimal.NewFromInt(1)})
		return &productcatalog.FlatFeeRateCard{RateCardMeta: meta, BillingCadence: &billingCadence}
	case productcatalog.UnitPriceType:
		meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)})
		return &productcatalog.UsageBasedRateCard{RateCardMeta: meta, BillingCadence: billingCadence}
	default:
		require.FailNow(t, "unsupported test price type", priceType)
		return nil
	}
}

func newExpectedLineTestStateItem(t *testing.T, fiatCurrency currencies.Currency, period timeutil.ClosedPeriod, rateCard productcatalog.RateCard) StateItem {
	t.Helper()

	return StateItem{
		SubscriptionItemWithPeriods: SubscriptionItemWithPeriods{
			SubscriptionItemView: subscription.SubscriptionItemView{
				SubscriptionItem: subscription.SubscriptionItem{RateCard: rateCard},
				Spec: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							RateCard: rateCard,
						},
					},
				},
			},
			ServicePeriod:     period,
			FullServicePeriod: period,
			BillingPeriod:     period,
		},
		Currency: fiatCurrency,
	}
}

func TestStateItemShouldProrateSubscriptionEndMode(t *testing.T) {
	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	activeTo := servicePeriod.To

	item := StateItem{
		SubscriptionItemWithPeriods: SubscriptionItemWithPeriods{
			SubscriptionItemView: subscription.SubscriptionItemView{
				SubscriptionItem: subscription.SubscriptionItem{
					RateCard: flatProratedRateCard(),
				},
				Spec: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							RateCard: flatProratedRateCard(),
						},
					},
				},
			},
			ServicePeriod: servicePeriod,
		},
		Subscription: subscription.Subscription{
			CadencedModel: models.CadencedModel{
				ActiveTo: &activeTo,
			},
			ProRatingConfig: productcatalog.ProRatingConfig{
				Enabled: true,
				Mode:    productcatalog.ProRatingModeProratePrices,
			},
		},
	}

	item.SubscriptionEndProrationMode = billing.SubscriptionEndProrationModeBillFullPeriod
	require.False(t, item.shouldProrate())

	item.SubscriptionEndProrationMode = billing.SubscriptionEndProrationModeBillActualPeriod
	require.True(t, item.shouldProrate())

	item.SubscriptionEndProrationMode = ""
	require.True(t, item.shouldProrate())

	item.SubscriptionEndProrationMode = billing.SubscriptionEndProrationMode("unknown")
	require.True(t, item.shouldProrate())
}

func flatProratedRateCard() productcatalog.RateCard {
	return &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(10),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
	}
}
