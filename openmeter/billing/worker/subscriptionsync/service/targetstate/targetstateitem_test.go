package targetstate

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

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
