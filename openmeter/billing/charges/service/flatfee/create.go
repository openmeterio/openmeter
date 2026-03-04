package flatfee

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func (s *service) PostCreate(ctx context.Context, charge charges.FlatFeeCharge) (charges.PostCreateFlatFeeResult, error) {
	charge.Status = charges.ChargeStatusActive

	_, err := s.adapter.UpdateFlatFeeCharge(ctx, charge)
	if err != nil {
		return charges.PostCreateFlatFeeResult{}, err
	}

	// For credit only flat fees we are not relying on the invoicing stack at all, so we can return early.
	if charge.Intent.SettlementMode == productcatalog.CreditOnlySettlementMode {
		return charges.PostCreateFlatFeeResult{
			Charge: charge,
		}, nil
	}

	gatheringLine, err := gatheringLineFromFlatFeeCharge(charge)
	if err != nil {
		return charges.PostCreateFlatFeeResult{}, err
	}

	return charges.PostCreateFlatFeeResult{
		Charge:                charge,
		GatheringLineToCreate: &gatheringLine,
	}, nil
}

func gatheringLineFromFlatFeeCharge(flatFee charges.FlatFeeCharge) (billing.GatheringLine, error) {
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
		return billing.GatheringLine{}, fmt.Errorf("cloning annotations: %w", err)
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
						Amount:      intent.AmountAfterProration,
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

	return gatheringLine, nil
}
