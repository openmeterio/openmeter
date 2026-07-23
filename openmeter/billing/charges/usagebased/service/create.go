package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
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

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]usagebased.ChargeWithGatheringLine, error) {
		now := clock.Now().UTC()
		createIntents, err := slicesx.MapWithErr(input.Intents, func(intent usagebased.Intent) (usagebased.CreateIntent, error) {
			if intent.Currency.IsCustom() &&
				intent.SettlementMode == productcatalog.CreditThenInvoiceSettlementMode &&
				!s.enableCustomCurrency.Load() {
				return usagebased.CreateIntent{}, fmt.Errorf("creating usage based charge with custom currency %q: %w", intent.Currency.GetCode(), meta.ErrCustomCurrencyNotSupported)
			}

			chargeIntent := intent.Normalized()

			var resolvedCostBasis *costbasis.State
			if chargeIntent.CostBasis != nil {
				var err error

				resolvedCostBasis, err = s.costbasisResolver.ResolveInitialState(ctx, costbasis.ResolveInitialStateInput{
					CurrencyID: chargeIntent.Currency.NamespacedID,
					Intent:     *chargeIntent.CostBasis,
					ResolvedAt: now,
				})
				if err != nil {
					return usagebased.CreateIntent{}, fmt.Errorf("resolving cost basis: %w", err)
				}
			}

			featureMeter, err := input.FeatureMeters.Get(chargeIntent.FeatureKey, false)
			if err != nil {
				return usagebased.CreateIntent{}, fmt.Errorf("resolve usage based feature for key %s: %w", chargeIntent.FeatureKey, err)
			}

			return usagebased.CreateIntent{
				Intent:            chargeIntent.AsOverridableIntent(),
				Annotations:       chargeIntent.Annotations,
				FeatureID:         featureMeter.Feature.ID,
				RatingEngine:      s.rater.GetPreferredRatingEngineFor(chargeIntent),
				ResolvedCostBasis: resolvedCostBasis,
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
