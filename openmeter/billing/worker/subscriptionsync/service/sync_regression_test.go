package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type cancellationRegressionChargeSnapshot struct {
	ID                chargesmeta.ChargeID
	SettlementMode    productcatalog.SettlementMode
	ServicePeriod     timeutil.ClosedPeriod
	FullServicePeriod timeutil.ClosedPeriod
	BillingPeriod     timeutil.ClosedPeriod
	InvoiceAt         time.Time
	UpdatedAt         time.Time
}

type cancellationRegressionTestCase struct {
	chargeType     chargesmeta.ChargeType
	settlementMode productcatalog.SettlementMode
}

func (s *CreditThenInvoiceTestSuite) TestCancellationReconcilesPeriodsByServiceDirection() {
	tcs := []struct {
		name    string
		planKey string
		cancellationRegressionTestCase
	}{
		{
			name:    "flat fee credits only",
			planKey: "period-direction-flat-fee-credits-only",
			cancellationRegressionTestCase: cancellationRegressionTestCase{
				chargeType:     chargesmeta.ChargeTypeFlatFee,
				settlementMode: productcatalog.CreditOnlySettlementMode,
			},
		},
		{
			name:    "flat fee credit then invoice",
			planKey: "period-direction-flat-fee-credit-then-invoice",
			cancellationRegressionTestCase: cancellationRegressionTestCase{
				chargeType:     chargesmeta.ChargeTypeFlatFee,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			},
		},
		{
			name:    "usage based credits only",
			planKey: "period-direction-usage-based-credits-only",
			cancellationRegressionTestCase: cancellationRegressionTestCase{
				chargeType:     chargesmeta.ChargeTypeUsageBased,
				settlementMode: productcatalog.CreditOnlySettlementMode,
			},
		},
		{
			name:    "usage based credit then invoice",
			planKey: "period-direction-usage-based-credit-then-invoice",
			cancellationRegressionTestCase: cancellationRegressionTestCase{
				chargeType:     chargesmeta.ChargeTypeUsageBased,
				settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			},
		},
	}

	baseStart := s.mustParseTime("2024-01-01T00:00:00Z")
	for i, tc := range tcs {
		s.Run(tc.name, func() {
			// Keep subscriptions in the shared suite namespace from overlapping.
			start := baseStart.AddDate(0, 0, i)
			ctx := s.T().Context()
			billingPeriodEnd := start.AddDate(1, 0, 0)
			cancelAt := start.Add(5 * time.Hour)
			clock.SetTime(start)
			defer clock.ResetTime()

			var subsView subscription.SubscriptionView
			var canceledView subscription.SubscriptionView
			var initial cancellationRegressionChargeSnapshot
			var updated cancellationRegressionChargeSnapshot

			s.Run("given", func() {
				// Given an annual subscription whose single charge has been synchronized.
				rateCard := cancellationRegressionRateCard(
					s.T(),
					tc.chargeType,
					"charge-item",
					s.APIRequestsTotalFeature.Key,
					s.APIRequestsTotalFeature.ID,
				)
				subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
					NamespacedModel: models.NamespacedModel{
						Namespace: s.Namespace,
					},
					Plan: productcatalog.Plan{
						PlanMeta: productcatalog.PlanMeta{
							Name:           tc.name,
							Key:            tc.planKey,
							Version:        1,
							Currency:       currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
							SettlementMode: tc.settlementMode,
							BillingCadence: datetime.MustParseDuration(s.T(), "P1Y"),
							ProRatingConfig: productcatalog.ProRatingConfig{
								Enabled: true,
								Mode:    productcatalog.ProRatingModeProratePrices,
							},
						},
						Phases: []productcatalog.Phase{
							{
								PhaseMeta: s.phaseMeta("default", ""),
								RateCards: productcatalog.RateCards{rateCard},
							},
						},
					},
				})

				if tc.chargeType == chargesmeta.ChargeTypeUsageBased {
					s.Require().NotNil(s.APIRequestsTotalFeature.MeterSlug)
					s.MockStreamingConnector.AddSimpleEvent(*s.APIRequestsTotalFeature.MeterSlug, 0, start)
				}

				s.Require().NoError(s.Service.SyncByView(ctx, subsView, start.Add(time.Minute)))
				chargePage, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
					Namespace:       subsView.Subscription.Namespace,
					SubscriptionIDs: []string{subsView.Subscription.ID},
				})
				s.Require().NoError(err)
				s.Require().Len(chargePage.Items, 1)

				initial = cancellationRegressionSnapshot(s.T(), chargePage.Items[0], tc.chargeType)
				s.Equal(tc.settlementMode, initial.SettlementMode)
				s.Equal(timeutil.ClosedPeriod{From: start, To: billingPeriodEnd}, initial.BillingPeriod)

				switch tc.chargeType {
				case chargesmeta.ChargeTypeFlatFee:
					// A one-time in-advance fee starts as an instant while retaining the
					// subscription-aligned annual billing period.
					s.Equal(timeutil.ClosedPeriod{From: start, To: start}, initial.ServicePeriod)
					s.Equal(timeutil.ClosedPeriod{From: start, To: start}, initial.FullServicePeriod)
					s.Equal(start, initial.InvoiceAt)
				case chargesmeta.ChargeTypeUsageBased:
					s.Equal(timeutil.ClosedPeriod{From: start, To: billingPeriodEnd}, initial.ServicePeriod)
					s.Equal(timeutil.ClosedPeriod{From: start, To: billingPeriodEnd}, initial.FullServicePeriod)
					s.Equal(billingPeriodEnd, initial.InvoiceAt)
				}
			})

			s.Run("when", func() {
				// When the subscription is canceled inside its first billing period and synchronized again.
				clock.SetTime(cancelAt)
				subscriptionModel, err := s.SubscriptionService.Cancel(ctx, subsView.Subscription.NamespacedID, subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingImmediate),
				})
				s.Require().NoError(err)

				canceledView, err = s.SubscriptionService.GetView(ctx, subscriptionModel.NamespacedID)
				s.Require().NoError(err)
				s.Require().NoError(s.Service.SyncByView(ctx, canceledView, cancelAt.Add(time.Minute)))

				updatedGeneric, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: initial.ID})
				s.Require().NoError(err)
				updated = cancellationRegressionSnapshot(s.T(), updatedGeneric, tc.chargeType)
			})

			s.Run("then", func() {
				// Then flat fees extend service while shrinking billing, whereas usage
				// charges follow their normal service shrink path. Both converge to the
				// canceled subscription state without replacing the charge.
				s.Equal(initial.ID, updated.ID)
				s.Equal(timeutil.ClosedPeriod{From: start, To: cancelAt}, updated.ServicePeriod)
				s.Equal(timeutil.ClosedPeriod{From: start, To: cancelAt}, updated.BillingPeriod)
				if tc.chargeType == chargesmeta.ChargeTypeFlatFee {
					s.Equal(timeutil.ClosedPeriod{From: start, To: cancelAt}, updated.FullServicePeriod)
					s.Equal(start, updated.InvoiceAt)
				} else {
					s.Equal(timeutil.ClosedPeriod{From: start, To: billingPeriodEnd}, updated.FullServicePeriod)
					s.Equal(cancelAt, updated.InvoiceAt)
				}

				// And retrying the same synchronization is an idempotent no-op.
				s.Require().NoError(s.Service.SyncByView(ctx, canceledView, cancelAt.Add(time.Minute)))
				retriedGeneric, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: initial.ID})
				s.Require().NoError(err)
				retried := cancellationRegressionSnapshot(s.T(), retriedGeneric, tc.chargeType)
				s.Equal(updated.UpdatedAt, retried.UpdatedAt)
			})
		})
	}
}

func cancellationRegressionRateCard(t *testing.T, chargeType chargesmeta.ChargeType, itemKey, featureKey, featureID string) productcatalog.RateCard {
	t.Helper()

	switch chargeType {
	case chargesmeta.ChargeTypeFlatFee:
		return &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:  itemKey,
				Name: itemKey,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(5),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
			},
		}
	case chargesmeta.ChargeTypeUsageBased:
		return &productcatalog.UsageBasedRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:     featureKey,
				Name:    featureKey,
				Feature: productcatalog.NewFeatureReference(lo.ToPtr(featureID), lo.ToPtr(featureKey)),
				Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
					Amount: alpacadecimal.NewFromFloat(1),
				}),
			},
			BillingCadence: datetime.MustParseDuration(t, "P1Y"),
		}
	default:
		require.FailNow(t, "unsupported cancellation regression charge type", "charge type: %s", chargeType)
		return nil
	}
}

func cancellationRegressionSnapshot(t *testing.T, charge charges.Charge, wantType chargesmeta.ChargeType) cancellationRegressionChargeSnapshot {
	t.Helper()
	require.Equal(t, wantType, charge.Type())

	switch wantType {
	case chargesmeta.ChargeTypeFlatFee:
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		require.NoError(t, err)
		intent := flatFeeCharge.Intent.GetBaseIntent()

		return cancellationRegressionChargeSnapshot{
			ID:                flatFeeCharge.GetChargeID(),
			SettlementMode:    intent.SettlementMode,
			ServicePeriod:     intent.ServicePeriod,
			FullServicePeriod: intent.FullServicePeriod,
			BillingPeriod:     intent.BillingPeriod,
			InvoiceAt:         intent.InvoiceAt,
			UpdatedAt:         flatFeeCharge.UpdatedAt,
		}
	case chargesmeta.ChargeTypeUsageBased:
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		require.NoError(t, err)
		intent := usageBasedCharge.Intent.GetBaseIntent()

		return cancellationRegressionChargeSnapshot{
			ID:                usageBasedCharge.GetChargeID(),
			SettlementMode:    intent.SettlementMode,
			ServicePeriod:     intent.ServicePeriod,
			FullServicePeriod: intent.FullServicePeriod,
			BillingPeriod:     intent.BillingPeriod,
			InvoiceAt:         intent.InvoiceAt,
			UpdatedAt:         usageBasedCharge.UpdatedAt,
		}
	default:
		require.FailNow(t, "unsupported cancellation regression charge type", "charge type: %s", wantType)
		return cancellationRegressionChargeSnapshot{}
	}
}
