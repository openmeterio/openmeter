package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
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

	featureMeters, err := s.featureMeterResolver.Resolve(ctx, input.Namespace, input.Intents.AsIntents()...)
	if err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]usagebased.ChargeWithGatheringLine, error) {
		now := clock.Now().UTC()
		createIntents, err := slicesx.MapWithErr(input.Intents, func(createIntent usagebased.CreateIntent) (usagebased.CreateIntentAdapterInput, error) {
			chargeIntent := createIntent.Intent.Normalized()
			chargeIntent.Discounts = chargeIntent.Discounts.UpsertCorrelationIDs()

			var resolvedCostBasis *costbasis.State
			if chargeIntent.CostBasis != nil {
				var err error

				resolvedCostBasis, err = s.costbasisResolver.ResolveInitialState(ctx, costbasis.ResolveInitialStateInput{
					CurrencyID: chargeIntent.Currency.NamespacedID,
					Intent:     *chargeIntent.CostBasis,
					ResolvedAt: now,
				})
				if err != nil {
					return usagebased.CreateIntentAdapterInput{}, fmt.Errorf("resolving cost basis: %w", err)
				}
			}

			featureMeter, err := featureMeters.Get(chargeIntent)
			var validationIssues billing.ValidationIssues
			if err != nil {
				if !billing.IsValidationIssueOnly(err) {
					return usagebased.CreateIntentAdapterInput{}, fmt.Errorf("resolve usage based feature %+v: %w", chargeIntent.GetFeatureRef(), err)
				}

				if !createIntent.Options.BypassFeatureMeterValidation {
					return usagebased.CreateIntentAdapterInput{}, fmt.Errorf("resolve usage based feature %+v: %w", chargeIntent.GetFeatureRef(), err)
				}

				validationIssues, _ = billing.ToValidationIssues(
					billing.ValidationWithComponent(billing.ValidationComponentProductCatalog, err),
				)
			}

			if featureMeter.Feature.ID != "" && chargeIntent.FeatureID != "" && chargeIntent.FeatureKey != "" && chargeIntent.FeatureKey != featureMeter.Feature.Key {
				return usagebased.CreateIntentAdapterInput{}, models.NewGenericValidationError(fmt.Errorf(
					"feature key %q does not match key %q resolved from feature id %q",
					chargeIntent.FeatureKey,
					featureMeter.Feature.Key,
					chargeIntent.FeatureID,
				))
			}

			featureID := chargeIntent.FeatureID
			if featureMeter.Feature.ID != "" {
				chargeIntent.FeatureKey = featureMeter.Feature.Key
			}
			if chargeIntent.FeatureID != "" && featureMeter.Feature.ID != "" {
				featureID = featureMeter.Feature.ID
			}

			return usagebased.CreateIntentAdapterInput{
				Intent:            chargeIntent.AsOverridableIntent(),
				Annotations:       chargeIntent.Annotations,
				FeatureID:         featureID,
				RatingEngine:      s.rater.GetPreferredRatingEngineFor(chargeIntent),
				ResolvedCostBasis: resolvedCostBasis,
				ValidationIssues:  validationIssues,
			}, nil
		})
		if err != nil {
			return nil, err
		}

		charges, err := s.adapter.CreateCharges(ctx, usagebased.CreateChargesAdapterInput{
			Namespace: input.Namespace,
			Intents:   createIntents,
		})
		if err != nil {
			return nil, err
		}

		return slicesx.MapWithErr(charges, func(charge usagebased.Charge) (usagebased.ChargeWithGatheringLine, error) {
			// For credit only flat fees we are not relying on the invoicing stack at all, so we can return early.
			if charge.Intent.GetSettlementMode() == productcatalog.CreditOnlySettlementMode {
				return usagebased.ChargeWithGatheringLine{
					Charge: charge,
				}, nil
			}

			return gatheringLineFromUsageBasedChargeForPeriod(charge, charge.Intent.GetEffectiveServicePeriod(), charge.Intent.GetEffectiveInvoiceAt())
		})
	})
}

func gatheringLineFromUsageBasedChargeForPeriod(charge usagebased.Charge, servicePeriod timeutil.ClosedPeriod, invoiceAt time.Time) (usagebased.ChargeWithGatheringLine, error) {
	intent := charge.Intent.GetEffectiveIntent()

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

	if intent.Currency.IsCustom() {
		// TODO: This should be a different typed gathering line, but for now we don't have that.
		if clonedAnnotations == nil {
			clonedAnnotations = models.Annotations{}
		}
		clonedAnnotations[billing.AnnotationKeyReason] = lo.ToPtr(billing.AnnotationValueReasonOveragePlaceholder)
	}

	var unitConfig *productcatalog.UnitConfig
	if intent.UnitConfig != nil {
		unitConfig = lo.ToPtr(intent.UnitConfig.Clone())
	}

	invoiceCurrency, err := charge.GetInvoiceCurrency()
	if err != nil {
		return usagebased.ChargeWithGatheringLine{}, fmt.Errorf("getting invoice currency: %w", err)
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
			UnitConfig: unitConfig,

			Currency:      invoiceCurrency,
			ServicePeriod: servicePeriod,
			InvoiceAt:     invoiceAt,

			TaxConfig: lo.ToPtr(intent.TaxConfig.ToTaxConfig()),

			ChargeID:     lo.ToPtr(charge.ID),
			Engine:       billing.LineEngineTypeChargeUsageBased,
			Subscription: subscription,

			RateCardDiscounts: intent.Discounts.Clone(),
		},
	}

	return usagebased.ChargeWithGatheringLine{
		Charge:                charge,
		GatheringLineToCreate: &gatheringLine,
	}, nil
}
