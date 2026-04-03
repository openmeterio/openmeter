package service

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/slicesx"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *service) Create(ctx context.Context, input flatfee.CreateInput) ([]flatfee.ChargeWithGatheringLine, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	if len(input.Intents) == 0 {
		return nil, nil
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]flatfee.ChargeWithGatheringLine, error) {
		// Let's create all the flat fee charges in bulk
		intentsWithStatus, err := slicesx.MapWithErr(input.Intents, func(intent flatfee.Intent) (flatfee.IntentWithInitialStatus, error) {
			intent = intent.Normalized()

			initialStatus := meta.ChargeStatusActive
			if intent.SettlementMode == productcatalog.CreditOnlySettlementMode {
				initialStatus = meta.ChargeStatusCreated
			}

			amountAfterProration, err := intent.CalculateAmountAfterProration()
			if err != nil {
				return flatfee.IntentWithInitialStatus{}, fmt.Errorf("calculating amount after proration: %w", err)
			}

			var featureID *string
			if intent.FeatureKey != "" {
				featureMeter, err := input.FeatureMeters.Get(intent.FeatureKey, false)
				if err != nil {
					return flatfee.IntentWithInitialStatus{}, fmt.Errorf("resolve flat fee feature for key %s: %w", intent.FeatureKey, err)
				}
				featureID = lo.ToPtr(featureMeter.Feature.ID)
			}

			return flatfee.IntentWithInitialStatus{
				Intent:               intent,
				FeatureID:            featureID,
				InitialStatus:        initialStatus,
				AmountAfterProration: amountAfterProration,
			}, nil
		})
		if err != nil {
			return nil, err
		}

		charges, err := s.adapter.CreateCharges(ctx, flatfee.CreateChargesInput{
			Namespace: input.Namespace,
			Intents:   intentsWithStatus,
		})
		if err != nil {
			return nil, err
		}

		return slicesx.MapWithErr(charges, func(charge flatfee.Charge) (flatfee.ChargeWithGatheringLine, error) {
			// For credit only flat fees we are not relying on the invoicing stack at all, so we can return early.
			if charge.Intent.SettlementMode == productcatalog.CreditOnlySettlementMode {
				return flatfee.ChargeWithGatheringLine{
					Charge: charge,
				}, nil
			}

			return gatheringLineFromFlatFeeCharge(charge)
		})
	})
}

func gatheringLineFromFlatFeeCharge(flatFee flatfee.Charge) (flatfee.ChargeWithGatheringLine, error) {
	intent := flatFee.Intent

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
		return flatfee.ChargeWithGatheringLine{}, fmt.Errorf("cloning annotations: %w", err)
	}

	gatheringLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   flatFee.Namespace,
				Name:        intent.Name,
				Description: intent.Description,
			}),

			Metadata:    intent.Metadata.Clone(),
			Annotations: clonedAnnotations,
			ManagedBy:   intent.ManagedBy,

			Price: lo.FromPtr(
				productcatalog.NewPriceFrom(
					productcatalog.FlatPrice{
						Amount:      flatFee.State.AmountAfterProration,
						PaymentTerm: intent.PaymentTerm,
					},
				),
			),
			FeatureKey: intent.FeatureKey,

			Currency:      intent.Currency,
			ServicePeriod: intent.ServicePeriod,
			InvoiceAt:     intent.InvoiceAt,

			TaxConfig: intent.TaxConfig,

			ChargeID:               lo.ToPtr(flatFee.ID),
			ChildUniqueReferenceID: intent.UniqueReferenceID,
			Subscription:           subscription,
		},
	}

	if intent.PercentageDiscounts != nil {
		gatheringLine.RateCardDiscounts = billing.Discounts{
			Percentage: &billing.PercentageDiscount{
				PercentageDiscount: *intent.PercentageDiscounts,
			},
		}
	}

	return flatfee.ChargeWithGatheringLine{
		Charge:                flatFee,
		GatheringLineToCreate: &gatheringLine,
	}, nil
}
