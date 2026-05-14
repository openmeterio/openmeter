package service

import (
	"context"
	"fmt"
	"time"

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
				Intent:                    intent,
				FeatureID:                 featureID,
				InitialStatus:             flatfee.StatusCreated,
				InitialAdvanceAfter:       lo.ToPtr(meta.NormalizeTimestamp(intent.ServicePeriod.From)),
				AmountAfterProration:      amountAfterProration,
				NoFiatTransactionRequired: intent.SettlementMode == productcatalog.CreditOnlySettlementMode || amountAfterProration.IsZero(),
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

			gatheringLine, err := buildFlatFeeGatheringLine(buildFlatFeeGatheringLineInput{
				Charge:        charge,
				ServicePeriod: charge.Intent.ServicePeriod,
				InvoiceAt:     charge.Intent.InvoiceAt,
			})
			if err != nil {
				return flatfee.ChargeWithGatheringLine{}, err
			}

			return flatfee.ChargeWithGatheringLine{
				Charge:                charge,
				GatheringLineToCreate: &gatheringLine,
			}, nil
		})
	})
}

type buildFlatFeeGatheringLineInput struct {
	Charge        flatfee.Charge
	ServicePeriod timeutil.ClosedPeriod
	InvoiceAt     time.Time
}

func (i buildFlatFeeGatheringLineInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if err := i.ServicePeriod.Validate(); err != nil {
		return fmt.Errorf("service period: %w", err)
	}

	if i.InvoiceAt.IsZero() {
		return fmt.Errorf("invoice at is required")
	}

	if i.Charge.Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		return fmt.Errorf("charge %s is not credit_then_invoice", i.Charge.ID)
	}

	return nil
}

func buildFlatFeeGatheringLine(input buildFlatFeeGatheringLineInput) (billing.GatheringLine, error) {
	if err := input.Validate(); err != nil {
		return billing.GatheringLine{}, err
	}

	flatFee := input.Charge
	lineIntent := flatFee.Intent
	lineIntent.ServicePeriod = input.ServicePeriod
	lineIntent.InvoiceAt = input.InvoiceAt
	lineIntent = lineIntent.Normalized()

	if err := lineIntent.Validate(); err != nil {
		return billing.GatheringLine{}, fmt.Errorf("validating line intent: %w", err)
	}

	amountAfterProration, err := lineIntent.CalculateAmountAfterProration()
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("calculating amount after proration: %w", err)
	}

	var subscription *billing.SubscriptionReference
	if lineIntent.Subscription != nil {
		subscription = &billing.SubscriptionReference{
			SubscriptionID: lineIntent.Subscription.SubscriptionID,
			PhaseID:        lineIntent.Subscription.PhaseID,
			ItemID:         lineIntent.Subscription.ItemID,
			BillingPeriod: timeutil.ClosedPeriod{
				From: lineIntent.BillingPeriod.From,
				To:   lineIntent.BillingPeriod.To,
			},
		}
	}

	clonedAnnotations, err := lineIntent.Annotations.Clone()
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("cloning annotations: %w", err)
	}

	gatheringLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   flatFee.Namespace,
				Name:        lineIntent.Name,
				Description: lineIntent.Description,
			}),

			Metadata:    lineIntent.Metadata.Clone(),
			Annotations: clonedAnnotations,
			ManagedBy:   lineIntent.ManagedBy,

			Price: lo.FromPtr(
				productcatalog.NewPriceFrom(
					productcatalog.FlatPrice{
						Amount:      amountAfterProration,
						PaymentTerm: lineIntent.PaymentTerm,
					},
				),
			),
			FeatureKey: lineIntent.FeatureKey,

			Currency:      lineIntent.Currency,
			ServicePeriod: lineIntent.ServicePeriod,
			InvoiceAt:     lineIntent.InvoiceAt,

			TaxConfig: lineIntent.TaxConfig.ToTaxConfig(),

			Engine:                 billing.LineEngineTypeChargeFlatFee,
			ChargeID:               lo.ToPtr(flatFee.ID),
			ChildUniqueReferenceID: lineIntent.UniqueReferenceID,
			Subscription:           subscription,
		},
	}

	if lineIntent.PercentageDiscounts != nil {
		gatheringLine.RateCardDiscounts = billing.Discounts{
			Percentage: &billing.PercentageDiscount{
				PercentageDiscount: *lineIntent.PercentageDiscounts,
			},
		}
	}

	return gatheringLine, nil
}
