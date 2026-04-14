package reconciler

import (
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
)

func TestPatchCollectionRouterResolveDefaultCollection(t *testing.T) {
	t.Parallel()

	flatRateCard := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key: "flat",
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(100),
				PaymentTerm: productcatalog.InAdvancePaymentTerm,
			}),
		},
	}
	usageRateCard := &productcatalog.UsageBasedRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:   "usage",
			Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
		},
	}

	testCases := []struct {
		name                    string
		settlementMode          productcatalog.SettlementMode
		enableCreditThenInvoice bool
		enableCredits           bool
		rateCard                productcatalog.RateCard
		expectedCollection      any
	}{
		{
			name:               "invoice only stays on invoice lines",
			settlementMode:     productcatalog.InvoiceOnlySettlementMode,
			rateCard:           flatRateCard,
			expectedCollection: &lineInvoicePatchCollection{},
		},
		{
			name:               "credit only flat fee uses flat fee charges",
			settlementMode:     productcatalog.CreditOnlySettlementMode,
			rateCard:           flatRateCard,
			expectedCollection: &flatFeeChargeCollection{},
			enableCredits:      true,
		},
		{
			name:               "credit only usage uses usage based charges",
			settlementMode:     productcatalog.CreditOnlySettlementMode,
			rateCard:           usageRateCard,
			expectedCollection: &usageBasedChargeCollection{},
			enableCredits:      true,
		},
		{
			name:                    "credit then invoice disabled stays on invoice lines",
			settlementMode:          productcatalog.CreditThenInvoiceSettlementMode,
			enableCredits:           true,
			enableCreditThenInvoice: false,
			rateCard:                flatRateCard,
			expectedCollection:      &lineInvoicePatchCollection{},
		},
		{
			name:                    "credit then invoice enabled flat fee uses flat fee charges",
			settlementMode:          productcatalog.CreditThenInvoiceSettlementMode,
			enableCredits:           true,
			enableCreditThenInvoice: true,
			rateCard:                flatRateCard,
			expectedCollection:      &flatFeeChargeCollection{},
		},
		{
			name:                    "credit then invoice enabled usage uses usage based charges",
			settlementMode:          productcatalog.CreditThenInvoiceSettlementMode,
			enableCredits:           true,
			enableCreditThenInvoice: true,
			rateCard:                usageRateCard,
			expectedCollection:      &usageBasedChargeCollection{},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router, err := newPatchCollectionRouter(patchCollectionRouterConfig{
				capacity:                 1,
				invoices:                 persistedstate.Invoices{},
				creditThenInvoiceEnabled: tt.enableCreditThenInvoice,
				creditsEnabled:           tt.enableCredits,
			})
			require.NoError(t, err)

			collection, err := router.ResolveDefaultCollection(testTargetStateItem(tt.settlementMode, tt.rateCard))
			require.NoError(t, err)
			require.IsType(t, tt.expectedCollection, collection)
		})
	}
}

func testTargetStateItem(settlementMode productcatalog.SettlementMode, rateCard productcatalog.RateCard) targetstate.StateItem {
	return targetstate.StateItem{
		SubscriptionItemWithPeriods: targetstate.SubscriptionItemWithPeriods{
			UniqueID: "item-1",
			SubscriptionItemView: subscription.SubscriptionItemView{
				Spec: subscription.SubscriptionItemSpec{
					CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
						CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
							PhaseKey: "phase-1",
							ItemKey:  "item-1",
							RateCard: rateCard,
						},
					},
				},
			},
		},
		Subscription: subscription.Subscription{
			SettlementMode: settlementMode,
		},
	}
}
