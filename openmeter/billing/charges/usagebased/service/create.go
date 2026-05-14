package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *service) Create(ctx context.Context, input usagebased.CreateInput) ([]usagebased.ChargeWithGatheringLine, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if len(input.Intents) == 0 {
		return nil, nil
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]usagebased.ChargeWithGatheringLine, error) {
		createIntents, err := slicesx.MapWithErr(input.Intents, func(intent usagebased.Intent) (usagebased.CreateIntent, error) {
			intent = intent.Normalized()

			featureMeter, err := input.FeatureMeters.Get(intent.FeatureKey, false)
			if err != nil {
				return usagebased.CreateIntent{}, fmt.Errorf("resolve usage based feature for key %s: %w", intent.FeatureKey, err)
			}

			return usagebased.CreateIntent{
				Intent:       intent,
				FeatureID:    featureMeter.Feature.ID,
				RatingEngine: s.rater.GetPreferredRatingEngineFor(intent),
			}, nil
		})
		if err != nil {
			return nil, err
		}

		// Let's create all the flat fee charges in bulk
		charges, err := s.adapter.CreateCharges(ctx, usagebased.CreateChargesInput{
			Namespace: input.Namespace,
			Intents:   createIntents,
		})
		if err != nil {
			return nil, err
		}

		err = s.metaAdapter.RegisterCharges(ctx, meta.RegisterChargesInput{
			Namespace: input.Namespace,
			Type:      meta.ChargeTypeUsageBased,
			Charges: lo.Map(charges, func(charge usagebased.Charge, idx int) meta.IDWithUniqueReferenceID {
				return meta.IDWithUniqueReferenceID{
					ID:                charge.ID,
					UniqueReferenceID: charge.Intent.UniqueReferenceID,
				}
			}),
		})
		if err != nil {
			return nil, err
		}

		return slicesx.MapWithErr(charges, func(charge usagebased.Charge) (usagebased.ChargeWithGatheringLine, error) {
			// For credit only flat fees we are not relying on the invoicing stack at all, so we can return early.
			if charge.Intent.SettlementMode == productcatalog.CreditOnlySettlementMode {
				return usagebased.ChargeWithGatheringLine{
					Charge: charge,
				}, nil
			}

			return gatheringLineFromUsageBasedCharge(charge)
		})
	})
}

func gatheringLineFromUsageBasedCharge(charge usagebased.Charge) (usagebased.ChargeWithGatheringLine, error) {
	return gatheringLineFromUsageBasedChargeForPeriod(charge, charge.Intent.ServicePeriod, charge.Intent.InvoiceAt)
}

func gatheringLineFromUsageBasedChargeForPeriod(charge usagebased.Charge, servicePeriod timeutil.ClosedPeriod, invoiceAt time.Time) (usagebased.ChargeWithGatheringLine, error) {
	intent := charge.Intent

	var subscription *billing.SubscriptionReference
	if intent.Subscription != nil {
		subscription = &billing.SubscriptionReference{
			SubscriptionID: intent.Subscription.SubscriptionID,
			PhaseID:        intent.Subscription.PhaseID,
			ItemID:         intent.Subscription.ItemID,
			BillingPeriod: timeutil.ClosedPeriod{
				From: intent.BillingPeriod.From,
				To:   intent.BillingPeriod.To,
			},
		}
	}

	clonedAnnotations, err := intent.Annotations.Clone()
	if err != nil {
		return usagebased.ChargeWithGatheringLine{}, fmt.Errorf("cloning annotations: %w", err)
	}

	gatheringLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   charge.Namespace,
				Name:        intent.Name,
				Description: intent.Description,
			}),

			Metadata:    intent.Metadata.Clone(),
			Annotations: clonedAnnotations,
			ManagedBy:   intent.ManagedBy,

			Price:      intent.Price,
			FeatureKey: intent.FeatureKey,

			Currency:      intent.Currency,
			ServicePeriod: servicePeriod,
			InvoiceAt:     invoiceAt,

			TaxConfig: intent.TaxConfig.ToTaxConfig(),

			ChargeID:               lo.ToPtr(charge.ID),
			Engine:                 billing.LineEngineTypeChargeUsageBased,
			ChildUniqueReferenceID: intent.UniqueReferenceID,
			Subscription:           subscription,
		},
	}

	if intent.Discounts.Usage != nil {
		gatheringLine.RateCardDiscounts.Usage = &billing.UsageDiscount{
			UsageDiscount: *intent.Discounts.Usage,
		}
	}

	if intent.Discounts.Percentage != nil {
		gatheringLine.RateCardDiscounts.Percentage = &billing.PercentageDiscount{
			PercentageDiscount: *intent.Discounts.Percentage,
		}
	}

	return usagebased.ChargeWithGatheringLine{
		Charge:                charge,
		GatheringLineToCreate: &gatheringLine,
	}, nil
}
