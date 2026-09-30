package service

import (
	"maps"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
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
	chargeType         chargesmeta.ChargeType
	settlementMode     productcatalog.SettlementMode
	flatFeePaymentTerm productcatalog.PaymentTermType
	disableProRating   bool
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
				chargeType:         chargesmeta.ChargeTypeFlatFee,
				settlementMode:     productcatalog.CreditOnlySettlementMode,
				flatFeePaymentTerm: productcatalog.InAdvancePaymentTerm,
			},
		},
		{
			name:    "flat fee credit then invoice",
			planKey: "period-direction-flat-fee-credit-then-invoice",
			cancellationRegressionTestCase: cancellationRegressionTestCase{
				chargeType:         chargesmeta.ChargeTypeFlatFee,
				settlementMode:     productcatalog.CreditThenInvoiceSettlementMode,
				flatFeePaymentTerm: productcatalog.InAdvancePaymentTerm,
			},
		},
		{
			name:    "flat fee credit then invoice with disabled proration in arrears",
			planKey: "period-direction-flat-fee-disabled-proration",
			cancellationRegressionTestCase: cancellationRegressionTestCase{
				chargeType:         chargesmeta.ChargeTypeFlatFee,
				settlementMode:     productcatalog.CreditThenInvoiceSettlementMode,
				flatFeePaymentTerm: productcatalog.InArrearsPaymentTerm,
				disableProRating:   true,
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
					tc.flatFeePaymentTerm,
				)
				proRatingConfig := productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				}
				if tc.disableProRating {
					proRatingConfig = productcatalog.ProRatingConfig{}
				}
				subsView = s.createSubscriptionFromPlan(plan.CreatePlanInput{
					NamespacedModel: models.NamespacedModel{
						Namespace: s.Namespace,
					},
					Plan: productcatalog.Plan{
						PlanMeta: productcatalog.PlanMeta{
							Name:            tc.name,
							Key:             tc.planKey,
							Version:         1,
							Currency:        currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
							SettlementMode:  tc.settlementMode,
							BillingCadence:  datetime.MustParseDuration(s.T(), "P1Y"),
							ProRatingConfig: proRatingConfig,
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
					if tc.flatFeePaymentTerm == productcatalog.InArrearsPaymentTerm {
						s.Equal(timeutil.ClosedPeriod{From: start, To: billingPeriodEnd}, initial.ServicePeriod)
						s.Equal(timeutil.ClosedPeriod{From: start, To: billingPeriodEnd}, initial.FullServicePeriod)
						s.Equal(billingPeriodEnd, initial.InvoiceAt)
					} else {
						// A one-time in-advance fee starts as an instant while retaining the
						// subscription-aligned annual billing period.
						s.Equal(timeutil.ClosedPeriod{From: start, To: start}, initial.ServicePeriod)
						s.Equal(timeutil.ClosedPeriod{From: start, To: start}, initial.FullServicePeriod)
						s.Equal(start, initial.InvoiceAt)
					}
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
				// Then the charge converges to the canceled subscription state without
				// replacing its identity, whether service extends or shrinks.
				s.Equal(initial.ID, updated.ID)
				s.Equal(timeutil.ClosedPeriod{From: start, To: cancelAt}, updated.ServicePeriod)
				s.Equal(timeutil.ClosedPeriod{From: start, To: cancelAt}, updated.BillingPeriod)
				if tc.chargeType == chargesmeta.ChargeTypeFlatFee {
					if tc.flatFeePaymentTerm == productcatalog.InArrearsPaymentTerm {
						s.Equal(timeutil.ClosedPeriod{From: start, To: billingPeriodEnd}, updated.FullServicePeriod)
						s.Equal(cancelAt, updated.InvoiceAt)
					} else {
						s.Equal(timeutil.ClosedPeriod{From: start, To: cancelAt}, updated.FullServicePeriod)
						s.Equal(start, updated.InvoiceAt)
					}
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

func cancellationRegressionRateCard(t *testing.T, chargeType chargesmeta.ChargeType, itemKey, featureKey, featureID string, flatFeePaymentTerm productcatalog.PaymentTermType) productcatalog.RateCard {
	t.Helper()

	switch chargeType {
	case chargesmeta.ChargeTypeFlatFee:
		rateCard := &productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:  itemKey,
				Name: itemKey,
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromFloat(5),
					PaymentTerm: flatFeePaymentTerm,
				}),
			},
		}
		if flatFeePaymentTerm == productcatalog.InArrearsPaymentTerm {
			rateCard.BillingCadence = lo.ToPtr(datetime.MustParseDuration(t, "P1Y"))
		}

		return rateCard
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

type paidCancellationRegressionInput struct {
	legacyAnnotationDrift bool
}

type paidCancellationPeriod struct {
	chargeID  string
	invoiceID billing.InvoiceID
	lineID    string
	period    timeutil.ClosedPeriod
	totals    totals.Totals
}

type paidCancellationHistorySnapshot struct {
	chargeIDs []string
	lineIDs   []string
}

func (s *CreditThenInvoiceTestSuite) TestCancellationRetainsPaidMonthlyFlatFeesWithoutAnnotationDrift() {
	s.testCancellationRetainsPaidMonthlyFlatFees(paidCancellationRegressionInput{})
}

func (s *CreditThenInvoiceTestSuite) TestCancellationRetainsPaidMonthlyFlatFeesWithLegacyAnnotationDrift() {
	s.testCancellationRetainsPaidMonthlyFlatFees(paidCancellationRegressionInput{legacyAnnotationDrift: true})
}

func (s *CreditThenInvoiceTestSuite) testCancellationRetainsPaidMonthlyFlatFees(input paidCancellationRegressionInput) {
	// given: a mid-month subscription has three paid monthly in-advance periods.
	// when: cancellation rematerializes its item, with or without historical annotation drift.
	// then: paid periods retain their charges and invoice lines without new collectible work.

	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-21T00:00:00Z")
	boundaries := []time.Time{
		start,
		s.mustParseTime("2024-02-01T00:00:00Z"),
		s.mustParseTime("2024-03-01T00:00:00Z"),
		s.mustParseTime("2024-04-01T00:00:00Z"),
	}
	const itemKey = "monthly-flat-fee"
	const migrationAnnotation = "dbmigration:backfill_subscription_item_currencies"
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	paid := make([]paidCancellationPeriod, 0, 3)
	var view subscription.SubscriptionView
	var canceledView subscription.SubscriptionView
	var oldItemID, newItemID string
	var itemAnnotations models.Annotations

	// given: a monthly in-advance subscription starts mid-month and three periods are invoiced and paid.
	s.RequireRun("setup plan and start subscription", func() {
		planEntity, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
			NamespacedModel: models.NamespacedModel{Namespace: s.Namespace},
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Name:           "Monthly flat fee",
					Key:            "monthly-flat-fee",
					Version:        1,
					Currency:       currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
					ProRatingConfig: productcatalog.ProRatingConfig{
						Enabled: true,
						Mode:    productcatalog.ProRatingModeProratePrices,
					},
				},
				Phases: []productcatalog.Phase{{
					PhaseMeta: s.phaseMeta("service", ""),
					RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:  itemKey,
							Name: "Monthly flat fee",
							Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
								Amount:      alpacadecimal.NewFromInt(30),
								PaymentTerm: productcatalog.InAdvancePaymentTerm,
							}),
						},
						BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
					}},
				}},
			},
		})
		s.Require().NoError(err)

		subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, s.Namespace, productcatalogsubscription.PlanRefInput{
			Key:     planEntity.Key,
			Version: lo.ToPtr(1),
		})
		s.Require().NoError(err)

		view, err = s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{Custom: lo.ToPtr(start)},
				Name:   "monthly-service",
			},
			Namespace:     s.Namespace,
			CustomerID:    s.Customer.ID,
			BillingAnchor: lo.ToPtr(s.mustParseTime("2024-01-01T00:00:00Z")),
		}, subscriptionPlan)
		s.Require().NoError(err)
		s.Require().Len(view.Phases, 1)
		s.Require().Len(view.Phases[0].ItemsByKey[itemKey], 1)
		oldItemID = view.Phases[0].ItemsByKey[itemKey][0].SubscriptionItem.ID
	})

	s.RequireRun("bill three monthly periods", func() {
		for i := 0; i < 3; i++ {
			clock.FreezeTime(boundaries[i].Add(time.Minute))
			s.Require().NoError(s.Service.SyncByView(ctx, view, boundaries[i+1]))

			invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
				Customer: s.Customer.GetID(),
				AsOf:     lo.ToPtr(clock.Now()),
			})
			s.Require().NoError(err)
			s.Require().Len(invoices, 1)
			s.Require().Len(invoices[0].Lines.OrEmpty(), 1)
			line := invoices[0].Lines.OrEmpty()[0]
			s.Require().Equal(timeutil.ClosedPeriod{From: boundaries[i], To: boundaries[i+1]}, line.Period)
			s.Require().NotNil(line.ChargeID)

			approved, err := s.BillingService.ApproveInvoice(ctx, invoices[0].GetInvoiceID())
			s.Require().NoError(err)
			s.Require().Equal(billing.StandardInvoiceStatusPaid, approved.Status)
			paidInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
				Invoice: approved.GetInvoiceID(),
				Expand:  billing.StandardInvoiceExpandAll,
			})
			s.Require().NoError(err)
			s.Require().Len(paidInvoice.Lines.OrEmpty(), 1)
			paidLine := paidInvoice.Lines.OrEmpty()[0]
			s.Require().Equal(line.ID, paidLine.ID)
			s.Require().Equal(timeutil.ClosedPeriod{From: boundaries[i], To: boundaries[i+1]}, paidLine.Period)
			s.Require().False(paidLine.Totals.IsZero())
			paid = append(paid, paidCancellationPeriod{
				chargeID:  *line.ChargeID,
				invoiceID: approved.GetInvoiceID(),
				lineID:    line.ID,
				period:    paidLine.Period,
				totals:    paidLine.Totals,
			})
		}
	})

	// when: cancellation follows either unchanged annotations or legacy annotation drift.
	if input.legacyAnnotationDrift {
		s.RequireRun("update annotations", func() {
			// A historical migration marked the item without changing its existing charges.
			itemAnnotations = maps.Clone(view.Phases[0].ItemsByKey[itemKey][0].SubscriptionItem.Annotations)
			s.Require().Contains(itemAnnotations, subscription.AnnotationOwnerSubSystem)
			itemAnnotations[migrationAnnotation] = "2024-03-19T00:00:00.000000Z"
			s.Require().NoError(s.DBClient.SubscriptionItem.UpdateOneID(oldItemID).SetAnnotations(itemAnnotations).Exec(ctx))
			for _, original := range paid {
				charge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
					ChargeID: chargesmeta.ChargeID{Namespace: s.Namespace, ID: original.chargeID},
				})
				s.Require().NoError(err)
				flatFee, err := charge.AsFlatFeeCharge()
				s.Require().NoError(err)
				chargeAnnotations := maps.Clone(flatFee.Intent.GetBaseIntent().Annotations)
				s.Require().NotContains(chargeAnnotations, migrationAnnotation)
				delete(chargeAnnotations, subscription.AnnotationOwnerSubSystem)
				s.Require().NoError(s.DBClient.ChargeFlatFee.UpdateOneID(original.chargeID).SetAnnotations(chargeAnnotations).Exec(ctx))
			}
		})
	}

	s.RequireRun("cancel at next boundary and sync", func() {
		clock.FreezeTime(s.mustParseTime("2024-03-20T00:00:00Z"))
		_, err := s.SubscriptionService.Cancel(ctx, view.Subscription.NamespacedID, subscription.Timing{
			Custom: lo.ToPtr(boundaries[3]),
		})
		s.Require().NoError(err)
		canceledView, err = s.SubscriptionService.GetView(ctx, view.Subscription.NamespacedID)
		s.Require().NoError(err)
		s.Require().Len(canceledView.Phases, 1)
		s.Require().Len(canceledView.Phases[0].ItemsByKey[itemKey], 1)
		newItemID = canceledView.Phases[0].ItemsByKey[itemKey][0].SubscriptionItem.ID
		s.Require().NotEqual(oldItemID, newItemID)
		if input.legacyAnnotationDrift {
			s.Require().Equal(itemAnnotations[migrationAnnotation], canceledView.Phases[0].ItemsByKey[itemKey][0].SubscriptionItem.Annotations[migrationAnnotation])
		}
		s.Require().NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, canceledView, boundaries[3]))
	})

	// then: paid history remains attached to the recreated item, with no duplicate collectible work.
	var first paidCancellationHistorySnapshot
	s.Run("preserve paid charge and invoice history", func() {
		first = s.assertPaidCancellationHistory(view.Subscription.ID, newItemID, paid)
	})
	s.Run("repeat reconciliation without new work", func() {
		s.Require().NoError(s.Service.SyncByViewAndInvoiceCustomer(ctx, canceledView, boundaries[3]))
		second := s.assertPaidCancellationHistory(view.Subscription.ID, newItemID, paid)
		s.ElementsMatch(first.chargeIDs, second.chargeIDs, "a repeated sync must not create more charges")
		s.ElementsMatch(first.lineIDs, second.lineIDs, "a repeated collection must not create more lines")
	})
}

func (s *CreditThenInvoiceTestSuite) assertPaidCancellationHistory(subscriptionID, itemID string, paid []paidCancellationPeriod) paidCancellationHistorySnapshot {
	s.T().Helper()
	ctx := s.T().Context()
	activeCharges, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Page:            pagination.Page{PageSize: 20, PageNumber: 1},
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{subscriptionID},
		ChargeTypes:     []chargesmeta.ChargeType{chargesmeta.ChargeTypeFlatFee},
	})
	s.Require().NoError(err)
	s.Require().Len(activeCharges.Items, len(paid), "no extra collectible historical or future charges")
	snapshot := paidCancellationHistorySnapshot{
		chargeIDs: make([]string, 0, len(activeCharges.Items)),
	}
	originalChargeIDs := make([]string, 0, len(paid))
	for _, charge := range activeCharges.Items {
		snapshot.chargeIDs = append(snapshot.chargeIDs, charge.GetID())
	}
	for _, original := range paid {
		originalChargeIDs = append(originalChargeIDs, original.chargeID)
	}
	s.ElementsMatch(originalChargeIDs, snapshot.chargeIDs, "paid charge identities must survive cancellation")

	for _, original := range paid {
		charge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: chargesmeta.ChargeID{Namespace: s.Namespace, ID: original.chargeID},
		})
		s.Require().NoError(err)
		flatFee, err := charge.AsFlatFeeCharge()
		s.Require().NoError(err)
		s.Require().NotNil(flatFee.Intent.GetSubscription())
		s.Equal(itemID, flatFee.Intent.GetSubscription().ItemID)

		invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: original.invoiceID,
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.Equal(billing.StandardInvoiceStatusPaid, invoice.Status)
		s.Require().Len(invoice.Lines.OrEmpty(), 1)
		line := invoice.Lines.OrEmpty()[0]
		s.Equal(original.lineID, line.ID)
		s.Equal(original.period, line.Period)
		s.True(original.totals.Equal(line.Totals), "paid line totals changed: before=%+v after=%+v", original.totals, line.Totals)
	}

	invoices, err := s.BillingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Page:         pagination.Page{PageSize: 20, PageNumber: 1},
		Namespace:    s.Namespace,
		CustomerID:   &filter.FilterULID{FilterString: filter.FilterString{Eq: &s.Customer.ID}},
		OnlyStandard: true,
		Expand:       billing.InvoiceExpandAll,
	})
	s.Require().NoError(err)
	snapshot.lineIDs = make([]string, 0, len(paid))
	for _, invoice := range invoices.Items {
		standard, err := invoice.AsStandardInvoice()
		s.Require().NoError(err)
		for _, line := range standard.Lines.OrEmpty() {
			snapshot.lineIDs = append(snapshot.lineIDs, line.ID)
		}
	}
	originalLineIDs := make([]string, 0, len(paid))
	for _, original := range paid {
		originalLineIDs = append(originalLineIDs, original.lineID)
	}
	s.ElementsMatch(originalLineIDs, snapshot.lineIDs, "no second collectible line for a paid period")
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	return snapshot
}

func (s *CreditThenInvoiceTestSuite) TestChargeReplacementRejectsPaidFlatFeeWithoutNewCollection() {
	ctx := s.T().Context()
	start := s.mustParseTime("2024-01-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	const itemKey = "replacement-guard-flat-fee"
	const referenceOnlyItemKey = "a-reference-only-flat-fee"
	var chargeID chargesmeta.ChargeID
	var referenceOnlyChargeID chargesmeta.ChargeID
	var invoiceID billing.InvoiceID
	var lineID string
	var originalAmount alpacadecimal.Decimal
	var oldItemID string
	var referenceOnlyItemID string
	var subscriptionID models.NamespacedID
	var originalChargeIDs []string

	// given: an in-advance flat fee has a paid invoice alongside a separate in-arrears charge.
	s.RequireRun("bill the original flat fee", func() {
		view := s.createSubscriptionFromPlan(plan.CreatePlanInput{
			NamespacedModel: models.NamespacedModel{Namespace: s.Namespace},
			Plan: productcatalog.Plan{
				PlanMeta: productcatalog.PlanMeta{
					Name:           "Guarded flat fee",
					Key:            "guarded-flat-fee",
					Version:        1,
					Currency:       currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
					SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
					BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				},
				Phases: []productcatalog.Phase{{
					PhaseMeta: s.phaseMeta("service", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  itemKey,
								Name: "Monthly flat fee",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(30),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  referenceOnlyItemKey,
								Name: "Reference-only flat fee",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromInt(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				}},
			},
		})
		oldItemID = view.Phases[0].ItemsByKey[itemKey][0].SubscriptionItem.ID
		referenceOnlyItemID = view.Phases[0].ItemsByKey[referenceOnlyItemKey][0].SubscriptionItem.ID
		subscriptionID = view.Subscription.NamespacedID
		s.Require().NoError(s.Service.SyncByView(ctx, view, start.AddDate(0, 1, 0)))

		invoices, err := s.BillingService.InvoicePendingLines(ctx, billing.InvoicePendingLinesInput{
			Customer: s.Customer.GetID(),
			AsOf:     lo.ToPtr(start),
		})
		s.Require().NoError(err)
		s.Require().Len(invoices, 1)
		s.Require().Len(invoices[0].Lines.OrEmpty(), 1)
		line := invoices[0].Lines.OrEmpty()[0]
		s.Require().NotNil(line.ChargeID)
		chargeID = chargesmeta.ChargeID{Namespace: s.Namespace, ID: *line.ChargeID}
		lineID = line.ID

		approved, err := s.BillingService.ApproveInvoice(ctx, invoices[0].GetInvoiceID())
		s.Require().NoError(err)
		invoiceID = approved.GetInvoiceID()
		paidInvoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoiceID,
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.Require().Equal(billing.StandardInvoiceStatusPaid, paidInvoice.Status)
		s.Require().True(paidInvoice.StatusDetails.Immutable)
		s.Require().Len(paidInvoice.Lines.OrEmpty(), 1)
		originalAmount = paidInvoice.Lines.OrEmpty()[0].Totals.Total

		allCharges, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
			Page:            pagination.Page{PageNumber: 1, PageSize: 20},
			Namespace:       s.Namespace,
			SubscriptionIDs: []string{subscriptionID.ID},
			IncludeDeleted:  true,
		})
		s.Require().NoError(err)
		for _, charge := range allCharges.Items {
			originalChargeIDs = append(originalChargeIDs, charge.GetID())
			flatFee, err := charge.AsFlatFeeCharge()
			s.Require().NoError(err)
			s.Require().NotNil(flatFee.Intent.GetSubscription())
			if flatFee.Intent.GetSubscription().ItemID == referenceOnlyItemID && flatFee.Intent.GetBaseIntent().ServicePeriod.From.Equal(start) {
				referenceOnlyChargeID = chargesmeta.ChargeID{Namespace: s.Namespace, ID: charge.GetID()}
			}
		}
		s.Require().Contains(originalChargeIDs, chargeID.ID)
		s.Require().NotEmpty(referenceOnlyChargeID.ID)
	})

	// when: a reference repair is planned before the paid item's new physical ID and price are checked.
	s.RequireRun("reject the changed item", func() {
		revised, err := s.SubscriptionService.GetView(ctx, subscriptionID)
		s.Require().NoError(err)
		item := &revised.Phases[0].ItemsByKey[itemKey][0]
		item.SubscriptionItem.ID = ulid.Make().String()
		s.Require().NotEqual(oldItemID, item.SubscriptionItem.ID)
		changedRateCard, ok := item.SubscriptionItem.RateCard.Clone().(*productcatalog.FlatFeeRateCard)
		s.Require().True(ok)
		changedRateCard.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
			Amount:      alpacadecimal.NewFromInt(40),
			PaymentTerm: productcatalog.InAdvancePaymentTerm,
		})
		item.SubscriptionItem.RateCard = changedRateCard
		item.Spec.RateCard = changedRateCard.Clone()
		revised.Phases[0].ItemsByKey[referenceOnlyItemKey][0].SubscriptionItem.ID = ulid.Make().String()
		s.Require().NoError(revised.Validate(true))

		err = s.Service.SyncByViewAndInvoiceCustomer(ctx, revised, start.AddDate(0, 1, 0))
		s.Require().ErrorContains(err, "immutable")
	})

	// then: neither the reference repair nor charge replacement is applied.
	s.Run("preserve the invoiced charge", func() {
		charge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: chargeID})
		s.Require().NoError(err)
		flatFee, err := charge.AsFlatFeeCharge()
		s.Require().NoError(err)
		s.Require().NotNil(flatFee.Intent.GetSubscription())
		s.Equal(oldItemID, flatFee.Intent.GetSubscription().ItemID)
		s.Equal(alpacadecimal.NewFromInt(30), flatFee.Intent.GetBaseIntent().AmountBeforeProration)

		allCharges, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
			Page:            pagination.Page{PageNumber: 1, PageSize: 20},
			Namespace:       s.Namespace,
			SubscriptionIDs: []string{subscriptionID.ID},
			IncludeDeleted:  true,
		})
		s.Require().NoError(err)
		chargeIDsAfter := make([]string, 0, len(allCharges.Items))
		for _, charge := range allCharges.Items {
			chargeIDsAfter = append(chargeIDsAfter, charge.GetID())
		}
		s.ElementsMatch(originalChargeIDs, chargeIDsAfter)
		referenceOnlyCharge, err := s.Charges.GetByID(ctx, charges.GetByIDInput{ChargeID: referenceOnlyChargeID})
		s.Require().NoError(err)
		referenceOnlyFlatFee, err := referenceOnlyCharge.AsFlatFeeCharge()
		s.Require().NoError(err)
		s.Equal(referenceOnlyItemID, referenceOnlyFlatFee.Intent.GetSubscription().ItemID)

		invoice, err := s.BillingService.GetStandardInvoiceById(ctx, billing.GetStandardInvoiceByIdInput{
			Invoice: invoiceID,
			Expand:  billing.StandardInvoiceExpandAll,
		})
		s.Require().NoError(err)
		s.Require().Len(invoice.Lines.OrEmpty(), 1)
		s.Equal(lineID, invoice.Lines.OrEmpty()[0].ID)
		s.Equal(originalAmount, invoice.Lines.OrEmpty()[0].Totals.Total)

		allInvoices, err := s.BillingService.ListStandardInvoices(ctx, billing.ListStandardInvoicesInput{
			Namespace: s.Namespace,
		})
		s.Require().NoError(err)
		s.Require().Len(allInvoices.Items, 1)
		s.Equal(invoiceID.ID, allInvoices.Items[0].ID)
	})
}
