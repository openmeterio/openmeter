package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *service) CreateCharges(ctx context.Context, input charges.CreateChargeInput) (charges.Charges, error) {
	input = input.WithCustomerAndCurrency()

	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (charges.Charges, error) {
		// TODO: validate feature references!
		// TODO: temporary forbid editing lines with charges or emulate single line charge edits.

		createdCharges, err := s.adapter.CreateCharges(ctx, input)
		if err != nil {
			return nil, err
		}

		gatheringLines, err := slicesx.MapWithErr(createdCharges, func(charge charges.Charge) (billing.GatheringLine, error) {
			return chargeIntentToGatheringLine(charge)
		})
		if err != nil {
			return nil, err
		}

		createLinesResult, err := s.billingService.CreatePendingInvoiceLines(ctx, billing.CreatePendingInvoiceLinesInput{
			Customer: input.Customer,
			Currency: input.Currency,
			Lines:    gatheringLines,
		})
		if err != nil {
			return nil, err
		}

		for idx := range createdCharges {
			createdCharges[idx].Expanded.GatheringLines = []billing.GatheringLineWithInvoiceHeader{
				{
					Line:    createLinesResult.Lines[idx],
					Invoice: createLinesResult.Invoice,
				},
			}
		}

		return createdCharges, nil
	})
}

func chargeIntentToGatheringLine(charge charges.Charge) (billing.GatheringLine, error) {
	intentMeta := charge.Intent.IntentMeta

	var subscription *billing.SubscriptionReference
	if charge.Intent.IntentMeta.Subscription != nil {
		subscription = &billing.SubscriptionReference{
			SubscriptionID: charge.Intent.IntentMeta.Subscription.SubscriptionID,
			PhaseID:        charge.Intent.IntentMeta.Subscription.PhaseID,
			ItemID:         charge.Intent.IntentMeta.Subscription.ItemID,
			BillingPeriod: timeutil.ClosedPeriod{
				From: charge.Intent.IntentMeta.BillingPeriod.From,
				To:   charge.Intent.IntentMeta.BillingPeriod.To,
			},
		}
	}

	gatheringLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   charge.Namespace,
				Name:        charge.Name,
				Description: charge.Description,
			}),

			Metadata:    intentMeta.Metadata,
			Annotations: intentMeta.Annotations,
			ManagedBy:   intentMeta.ManagedBy,

			Currency:      intentMeta.Currency,
			ServicePeriod: intentMeta.ServicePeriod,
			InvoiceAt:     intentMeta.InvoiceAt,

			TaxConfig: intentMeta.TaxConfig,

			ChargeID:               lo.ToPtr(charge.ID),
			ChildUniqueReferenceID: intentMeta.UniqueReferenceID,
			Subscription:           subscription,
		},
	}

	switch charge.Intent.IntentType {
	case charges.IntentTypeFlatFee:
		flatFeeIntent, err := charge.Intent.GetFlatFeeIntent()
		if err != nil {
			return billing.GatheringLine{}, err
		}

		gatheringLine.Price = lo.FromPtr(
			productcatalog.NewPriceFrom(
				productcatalog.FlatPrice{
					Amount:      flatFeeIntent.AmountAfterProration,
					PaymentTerm: flatFeeIntent.PaymentTerm,
				},
			),
		)
		gatheringLine.FeatureKey = flatFeeIntent.FeatureKey
		if flatFeeIntent.PercentageDiscounts != nil {
			gatheringLine.RateCardDiscounts = billing.Discounts{
				Percentage: &billing.PercentageDiscount{
					PercentageDiscount: *flatFeeIntent.PercentageDiscounts,
				},
			}
		}
	case charges.IntentTypeUsageBased:
		usageBasedIntent, err := charge.Intent.GetUsageBasedIntent()
		if err != nil {
			return billing.GatheringLine{}, err
		}

		gatheringLine.Price = usageBasedIntent.Price
		gatheringLine.FeatureKey = usageBasedIntent.FeatureKey
		gatheringLine.RateCardDiscounts = billing.NewDiscountsFromProductCatalogDiscounts(usageBasedIntent.Discounts)
	default:
		return billing.GatheringLine{}, fmt.Errorf("invalid intent type: %s", charge.Intent.IntentType)
	}
	return gatheringLine, nil
}
