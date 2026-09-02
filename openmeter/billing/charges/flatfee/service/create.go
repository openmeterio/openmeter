package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
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
		now := clock.Now().UTC()
		// Let's create all the flat fee charges in bulk
		intentsWithStatus, err := slicesx.MapWithErr(input.Intents, func(intent flatfee.Intent) (flatfee.IntentWithInitialStatus, error) {
			chargeIntent := intent.Normalized()
			chargeIntent.PercentageDiscounts = chargeIntent.PercentageDiscounts.UpsertCorrelationID()

			var resolvedCostBasis *costbasis.State
			if chargeIntent.CostBasis != nil {
				var err error

				resolvedCostBasis, err = s.costbasisResolver.ResolveInitialState(ctx, costbasis.ResolveInitialStateInput{
					CurrencyID: chargeIntent.Currency.NamespacedID,
					Intent:     *chargeIntent.CostBasis,
					ResolvedAt: now,
				})
				if err != nil {
					return flatfee.IntentWithInitialStatus{}, fmt.Errorf("resolving cost basis: %w", err)
				}
			}

			amountAfterProration, err := chargeIntent.CalculateAmountAfterProration()
			if err != nil {
				return flatfee.IntentWithInitialStatus{}, fmt.Errorf("calculating amount after proration: %w", err)
			}

			featureRef, err := chargeIntent.GetFeatureRef()
			if err != nil {
				return flatfee.IntentWithInitialStatus{}, fmt.Errorf("getting feature ref: %w", err)
			}
			var featureID *string
			if featureRef != nil {
				featureMeter, err := input.FeatureMeters.Resolve(billingfeaturemeter.FeatureMeterRef{
					IDOrKey:      *featureRef,
					RequireMeter: false,
				})
				if err != nil {
					return flatfee.IntentWithInitialStatus{}, fmt.Errorf("resolve flat fee feature %+v: %w", *featureRef, err)
				}
				featureID = lo.ToPtr(featureMeter.Feature.ID)
				// note: we must set the feature key on the intent, because no other place is setting it
				// and we want to persist it
				chargeIntent.FeatureKey = lo.ToPtr(featureMeter.Feature.Key)
			}

			return flatfee.IntentWithInitialStatus{
				Intent:                    chargeIntent,
				FeatureID:                 featureID,
				InitialStatus:             flatfee.StatusCreated,
				InitialAdvanceAfter:       lo.ToPtr(meta.NormalizeTimestamp(chargeIntent.InvoiceAt)),
				AmountAfterProration:      amountAfterProration,
				NoFiatTransactionRequired: chargeIntent.SettlementMode == productcatalog.CreditOnlySettlementMode || amountAfterProration.IsZero(),
				ResolvedCostBasis:         resolvedCostBasis,
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

		// Preserve the input-intent order when returning charge results. Billing
		// API-created line handling pairs each returned charge target with the
		// preallocated source line at the same index.
		return slicesx.MapWithErr(charges, func(charge flatfee.Charge) (flatfee.ChargeWithGatheringLine, error) {
			// For credit only flat fees we are not relying on the invoicing stack at all, so we can return early.
			if charge.Intent.GetSettlementMode() == productcatalog.CreditOnlySettlementMode {
				return flatfee.ChargeWithGatheringLine{
					Charge: charge,
				}, nil
			}

			// Zero-amount flat-fee charges are tracked as charges, but they
			// must not materialize billable invoice lines.
			if charge.State.AmountAfterProration.IsZero() {
				return flatfee.ChargeWithGatheringLine{
					Charge: charge,
				}, nil
			}

			gatheringLine, err := buildFlatFeeGatheringLine(buildFlatFeeGatheringLineInput{
				Charge:    charge,
				InvoiceAt: charge.Intent.GetEffectiveInvoiceAt(),
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
	Charge    flatfee.Charge
	InvoiceAt time.Time
}

func (i buildFlatFeeGatheringLineInput) Validate() error {
	if err := i.Charge.Validate(); err != nil {
		return fmt.Errorf("charge: %w", err)
	}

	if i.InvoiceAt.IsZero() {
		return fmt.Errorf("invoice at is required")
	}

	if i.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		return fmt.Errorf("charge %s is not credit_then_invoice", i.Charge.ID)
	}

	return nil
}

func buildFlatFeeGatheringLine(input buildFlatFeeGatheringLineInput) (billing.GatheringLine, error) {
	if err := input.Validate(); err != nil {
		return billing.GatheringLine{}, err
	}

	flatFee := input.Charge
	rateableIntent, err := flatFee.GetRateableIntent()
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("getting rateable intent: %w", err)
	}

	lineIntent := rateableIntent.Intent
	lineIntent.InvoiceAt = meta.NormalizeTimestamp(input.InvoiceAt)

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

	lineName := lineIntent.Name
	linePrice := lo.FromPtr(rateableIntent.GetPrice())
	lineDiscounts := rateableIntent.GetRateCardDiscounts()
	if lineIntent.Currency.IsCustom() {
		if clonedAnnotations == nil {
			clonedAnnotations = models.Annotations{}
		}
		clonedAnnotations[billing.AnnotationKeyReason] = lo.ToPtr(billing.AnnotationValueReasonOveragePlaceholder)

		lineName = "overage"
		if lineIntent.Name != "" {
			lineName = fmt.Sprintf("%s (overage)", lineIntent.Name)
		}

		linePrice = lo.FromPtr(productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.Zero,
			PaymentTerm: productcatalog.InArrearsPaymentTerm,
		}))
		lineDiscounts = billing.Discounts{}
	}

	managedBy := lineIntent.ManagedBy
	if flatFee.Intent.HasOverrideLayer() {
		managedBy = billing.ManuallyManagedLine
	}

	invoiceCurrency, err := flatFee.GetInvoiceCurrency()
	if err != nil {
		return billing.GatheringLine{}, fmt.Errorf("getting invoice currency: %w", err)
	}

	gatheringLine := billing.GatheringLine{
		GatheringLineBase: billing.GatheringLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace:   flatFee.Namespace,
				Name:        lineName,
				Description: lineIntent.Description,
			}),

			Metadata:    lineIntent.Metadata.Clone(),
			Annotations: clonedAnnotations,
			ManagedBy:   managedBy,

			Price:      linePrice,
			FeatureKey: lo.FromPtr(lineIntent.FeatureKey),

			Currency:      invoiceCurrency,
			ServicePeriod: lineIntent.ServicePeriod,
			InvoiceAt:     lineIntent.InvoiceAt,

			TaxConfig: lo.ToPtr(lineIntent.TaxConfig.ToTaxConfig()),

			Engine:       billing.LineEngineTypeChargeFlatFee,
			ChargeID:     lo.ToPtr(flatFee.ID),
			Subscription: subscription,
		},
	}

	gatheringLine.RateCardDiscounts = lineDiscounts

	return gatheringLine, nil
}
