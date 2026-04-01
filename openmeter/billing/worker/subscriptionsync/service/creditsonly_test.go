package service

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	chargesmeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type CreditsOnlySubscriptionHandlerTestSuite struct {
	SuiteBase
}

type expectedFlatFeeCharge struct {
	ChildUniqueReferenceIDs []string
	ServicePeriods          []timeutil.ClosedPeriod
	FullServicePeriods      []timeutil.ClosedPeriod
	BillingPeriods          []timeutil.ClosedPeriod
	InvoiceAt               []time.Time
	AmountBeforeProration   []alpacadecimal.Decimal
	AmountAfterProration    []alpacadecimal.Decimal
}

func (e expectedFlatFeeCharge) Indexes(indexes ...int) expectedFlatFeeCharge {
	return expectedFlatFeeCharge{
		ChildUniqueReferenceIDs: lo.Map(indexes, func(index int, _ int) string { return e.ChildUniqueReferenceIDs[index] }),
		ServicePeriods:          lo.Map(indexes, func(index int, _ int) timeutil.ClosedPeriod { return e.ServicePeriods[index] }),
		FullServicePeriods:      lo.Map(indexes, func(index int, _ int) timeutil.ClosedPeriod { return e.FullServicePeriods[index] }),
		BillingPeriods:          lo.Map(indexes, func(index int, _ int) timeutil.ClosedPeriod { return e.BillingPeriods[index] }),
		InvoiceAt:               lo.Map(indexes, func(index int, _ int) time.Time { return e.InvoiceAt[index] }),
		AmountBeforeProration:   lo.Map(indexes, func(index int, _ int) alpacadecimal.Decimal { return e.amountBeforeProration(index) }),
		AmountAfterProration:    lo.Map(indexes, func(index int, _ int) alpacadecimal.Decimal { return e.amountAfterProration(index) }),
	}
}

func (e expectedFlatFeeCharge) amountBeforeProration(index int) alpacadecimal.Decimal {
	if len(e.AmountBeforeProration) == 0 {
		return alpacadecimal.NewFromFloat(100)
	}

	return e.AmountBeforeProration[index]
}

func (e expectedFlatFeeCharge) amountAfterProration(index int) alpacadecimal.Decimal {
	if len(e.AmountAfterProration) == 0 {
		return alpacadecimal.NewFromFloat(100)
	}

	return e.AmountAfterProration[index]
}

type expectedUsageBasedCharge struct {
	ChildUniqueReferenceIDs []string
	ServicePeriods          []timeutil.ClosedPeriod
	FullServicePeriods      []timeutil.ClosedPeriod
	BillingPeriods          []timeutil.ClosedPeriod
	InvoiceAt               []time.Time
	FeatureKey              string
	Price                   productcatalog.Price
}

func (e expectedUsageBasedCharge) Indexes(indexes ...int) expectedUsageBasedCharge {
	return expectedUsageBasedCharge{
		ChildUniqueReferenceIDs: lo.Map(indexes, func(index int, _ int) string { return e.ChildUniqueReferenceIDs[index] }),
		ServicePeriods:          lo.Map(indexes, func(index int, _ int) timeutil.ClosedPeriod { return e.ServicePeriods[index] }),
		FullServicePeriods:      lo.Map(indexes, func(index int, _ int) timeutil.ClosedPeriod { return e.FullServicePeriods[index] }),
		BillingPeriods:          lo.Map(indexes, func(index int, _ int) timeutil.ClosedPeriod { return e.BillingPeriods[index] }),
		InvoiceAt:               lo.Map(indexes, func(index int, _ int) time.Time { return e.InvoiceAt[index] }),
		FeatureKey:              e.FeatureKey,
		Price:                   e.Price,
	}
}

func TestCreditsOnlySubscriptionHandlerScenarios(t *testing.T) {
	suite.Run(t, new(CreditsOnlySubscriptionHandlerTestSuite))
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) SetupSuite() {
	s.SuiteBase.SetupSuite()
	handlers := chargestestutils.NewMockHandlers()

	s.setupChargesService(chargestestutils.Config{
		Client:                s.DBClient,
		Logger:                slog.Default(),
		BillingService:        s.BillingService,
		FeatureService:        s.FeatureService,
		StreamingConnector:    s.MockStreamingConnector,
		FlatFeeHandler:        handlers.FlatFee,
		CreditPurchaseHandler: handlers.CreditPurchase,
		UsageBasedHandler:     handlers.UsageBased,
	})
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) TestCreditsOnlyFlatFeeProvisioningAndReconciliation() {
	// Given:
	// - a subscription is created with credits_only settlement
	// - the subscription is single phase with a flat fee charge of $100
	//
	// When:
	// - the charge is provisioned for the next two billing cycles
	//
	// Then:
	// - two charges are created with matching properties and child unique reference IDs
	//
	// Given:
	// - the two expected charges already exist
	//
	// When:
	// - clock advances
	// - we reprovision the flat fees for the next two billing cycles
	//
	// Then:
	// - the existing charges remain unchanged
	ctx := s.testContext()
	setupAt := s.mustParseTime("2024-01-01T00:00:00Z")
	startAt := s.mustParseTime("2024-02-01T00:00:00Z")
	syncUntil := s.mustParseTime("2024-02-15T00:00:00Z")

	clock.SetTime(setupAt)
	defer clock.ResetTime()

	subscriptionView := s.createSubscriptionFromPlanAt(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Credits Only Flat Fee",
				Key:            "credits-only-flat-fee",
				Version:        1,
				Currency:       currency.USD,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name: "flat-fee",
								Key:  "flat-fee",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	}, startAt)
	timeline := timeutil.NewSimpleTimeline([]time.Time{
		s.mustParseTime("2024-02-01T00:00:00Z"),
		s.mustParseTime("2024-03-01T00:00:00Z"),
		s.mustParseTime("2024-04-01T00:00:00Z"),
	})
	periods := timeline.GetClosedPeriods()
	invoiceAt := timeline.GetTimes()[:len(timeline.GetTimes())-1]

	expectedCharges := []expectedFlatFeeCharge{
		{
			ChildUniqueReferenceIDs: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "flat-fee",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			}.ChildIDs(subscriptionView.Subscription.ID),
			ServicePeriods:     periods,
			FullServicePeriods: periods,
			BillingPeriods:     periods,
			InvoiceAt:          invoiceAt,
		},
	}

	var initialCharges []flatfee.Charge

	s.Run("provisions the next two billing cycles", func() {
		// When we provision the next two billing cycles.
		s.NoError(s.Service.SynchronizeSubscription(ctx, subscriptionView, syncUntil))

		// Then two matching flat fee charges are created.
		initialCharges = s.expectCreditsOnlyFlatFeeCharges(ctx, subscriptionView.Subscription.ID, expectedCharges)
	})

	s.Run("reconciliation leaves charges unchanged", func() {
		// Given the two charges already exist.
		initialUpdatedAtByID := lo.SliceToMap(initialCharges, func(charge flatfee.Charge) (string, time.Time) {
			return charge.ID, charge.UpdatedAt
		})

		// When the clock advances and we re-provision the same next two billing cycles.
		clock.SetTime(s.mustParseTime("2024-01-15T00:00:00Z"))
		s.NoError(s.Service.SynchronizeSubscription(ctx, subscriptionView, syncUntil))

		// Then the existing charges are unchanged.
		reconciledCharges := s.expectCreditsOnlyFlatFeeCharges(ctx, subscriptionView.Subscription.ID, expectedCharges)
		s.Len(reconciledCharges, len(initialCharges))

		for _, charge := range reconciledCharges {
			updatedAt, ok := initialUpdatedAtByID[charge.ID]
			s.Truef(ok, "unexpected charge %s after reconciliation", charge.ID)
			s.Equal(updatedAt, charge.UpdatedAt, "charge %s should not have been updated", charge.ID)
		}
	})
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) TestCreditsOnlyFlatFeeCancellationAtPeriodBoundary() {
	// Given:
	// - a subscription is created with credits_only settlement
	// - the subscription is single phase with a flat fee charge of $100
	//
	// When:
	// - the charge is provisioned for the next two billing cycles
	//
	// Then:
	// - two charges are created with matching properties and child unique reference IDs
	//
	// Given:
	// - the previous two flat fees exist
	//
	// When:
	// - the subscription is canceled at the end of the first period exactly
	//
	// Then:
	// - the second flat fee charge is deleted
	ctx := s.testContext()
	setupAt := s.mustParseTime("2024-01-01T00:00:00Z")
	startAt := s.mustParseTime("2024-02-01T00:00:00Z")
	syncUntil := s.mustParseTime("2024-02-15T00:00:00Z")

	clock.SetTime(setupAt)
	defer clock.ResetTime()

	subscriptionView := s.createSubscriptionFromPlanAt(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Credits Only Flat Fee",
				Key:            "credits-only-flat-fee",
				Version:        1,
				Currency:       currency.USD,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name: "flat-fee",
								Key:  "flat-fee",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	}, startAt)
	timeline := timeutil.NewSimpleTimeline([]time.Time{
		s.mustParseTime("2024-02-01T00:00:00Z"),
		s.mustParseTime("2024-03-01T00:00:00Z"),
		s.mustParseTime("2024-04-01T00:00:00Z"),
	})
	periods := timeline.GetClosedPeriods()
	invoiceAt := timeline.GetTimes()[:len(timeline.GetTimes())-1]

	expectedCharges := []expectedFlatFeeCharge{
		{
			ChildUniqueReferenceIDs: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "flat-fee",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			}.ChildIDs(subscriptionView.Subscription.ID),
			ServicePeriods:     periods,
			FullServicePeriods: periods,
			BillingPeriods:     periods,
			InvoiceAt:          invoiceAt,
		},
	}

	var originalSecondPeriodCharge flatfee.Charge

	s.Run("provisions the next two billing cycles", func() {
		s.NoError(s.Service.SynchronizeSubscription(ctx, subscriptionView, syncUntil))
		provisionedCharges := s.expectCreditsOnlyFlatFeeCharges(ctx, subscriptionView.Subscription.ID, expectedCharges)
		s.Len(provisionedCharges, 2)

		originalSecondPeriodCharge = provisionedCharges[1]
		s.Equal(expectedCharges[0].ChildUniqueReferenceIDs[1], lo.FromPtr(originalSecondPeriodCharge.Intent.UniqueReferenceID))
	})

	s.Run("canceling at the first period boundary deletes the second flat fee", func() {
		cancelAt := s.mustParseTime("2024-03-01T00:00:00Z")
		clock.SetTime(cancelAt)

		subscriptionModel, err := s.SubscriptionService.Cancel(ctx, subscriptionView.Subscription.NamespacedID, subscription.Timing{
			Custom: lo.ToPtr(cancelAt),
		})
		s.NoError(err)

		subscriptionView, err = s.SubscriptionService.GetView(ctx, subscriptionModel.NamespacedID)
		s.NoError(err)

		s.NoError(s.Service.SynchronizeSubscription(ctx, subscriptionView, syncUntil))

		remainingCharges := s.expectCreditsOnlyFlatFeeCharges(ctx, subscriptionView.Subscription.ID, []expectedFlatFeeCharge{
			expectedCharges[0].Indexes(0),
		})
		s.Len(remainingCharges, 1)

		deletedChargeRes, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: chargesmeta.ChargeID{
				Namespace: s.Namespace,
				ID:        originalSecondPeriodCharge.ID,
			},
		})
		s.NoError(err)

		deletedCharge, err := deletedChargeRes.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(chargesmeta.ChargeStatusDeleted, deletedCharge.Status)
		s.NotNil(deletedCharge.DeletedAt)
		s.Equal(expectedCharges[0].ChildUniqueReferenceIDs[1], lo.FromPtr(deletedCharge.Intent.UniqueReferenceID))
	})
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) TestCreditsOnlyFlatFeeMidPeriodCancellation() {
	// Given:
	// - a subscription is created with credits_only settlement
	// - the subscription is single phase with a flat fee charge of $100
	//
	// When:
	// - the charge is provisioned for the current and next billing cycle
	//
	// Then:
	// - two charges are created with matching properties and child unique reference IDs
	//
	// Given:
	// - the subscription is canceled mid-period
	//
	// When:
	// - the subscription is synchronized again past the cancellation timestamp
	//
	// Then:
	// - the current period charge is shrunk and prorated
	// - the future period charge is deleted
	ctx := s.testContext()
	setupAt := s.mustParseTime("2024-01-01T00:00:00Z")
	startAt := s.mustParseTime("2024-02-01T00:00:00Z")
	initialSyncUntil := s.mustParseTime("2024-02-15T00:00:00Z")
	cancelAt := s.mustParseTime("2024-02-16T00:00:00Z")
	resyncUntil := s.mustParseTime("2024-03-01T00:00:00Z")

	clock.SetTime(setupAt)
	defer clock.ResetTime()

	subscriptionView := s.createSubscriptionFromPlanAt(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Credits Only Flat Fee Mid Period Cancellation",
				Key:            "credits-only-flat-fee-mid-period-cancellation",
				Version:        1,
				Currency:       currency.USD,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name: "flat-fee",
								Key:  "flat-fee",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
					},
				},
			},
		},
	}, startAt)
	timeline := timeutil.NewSimpleTimeline([]time.Time{
		s.mustParseTime("2024-02-01T00:00:00Z"),
		s.mustParseTime("2024-03-01T00:00:00Z"),
		s.mustParseTime("2024-04-01T00:00:00Z"),
	})
	periods := timeline.GetClosedPeriods()
	invoiceAt := timeline.GetTimes()[:len(timeline.GetTimes())-1]

	expectedCharges := []expectedFlatFeeCharge{
		{
			ChildUniqueReferenceIDs: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "flat-fee",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			}.ChildIDs(subscriptionView.Subscription.ID),
			ServicePeriods:     periods,
			FullServicePeriods: periods,
			BillingPeriods:     periods,
			InvoiceAt:          invoiceAt,
		},
	}

	var originalSecondPeriodCharge flatfee.Charge

	s.Run("provisions the current and next billing cycle", func() {
		s.NoError(s.Service.SynchronizeSubscription(ctx, subscriptionView, initialSyncUntil))

		provisionedCharges := s.expectCreditsOnlyFlatFeeCharges(ctx, subscriptionView.Subscription.ID, expectedCharges)
		s.Len(provisionedCharges, 2)

		originalSecondPeriodCharge = provisionedCharges[1]
	})

	s.Run("canceling mid-period shrinks the current charge and deletes the future one", func() {
		clock.FreezeTime(cancelAt)
		defer clock.UnFreeze()

		subscriptionModel, err := s.SubscriptionService.Cancel(ctx, subscriptionView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingImmediate),
		})
		s.NoError(err)

		subscriptionView, err = s.SubscriptionService.GetView(ctx, subscriptionModel.NamespacedID)
		s.NoError(err)

		s.NoError(s.Service.SynchronizeSubscription(ctx, subscriptionView, resyncUntil))

		remainingCharges := s.expectCreditsOnlyFlatFeeCharges(ctx, subscriptionView.Subscription.ID, []expectedFlatFeeCharge{
			{
				ChildUniqueReferenceIDs: []string{expectedCharges[0].ChildUniqueReferenceIDs[0]},
				ServicePeriods: []timeutil.ClosedPeriod{
					{
						From: periods[0].From,
						To:   cancelAt,
					},
				},
				FullServicePeriods: []timeutil.ClosedPeriod{periods[0]},
				BillingPeriods: []timeutil.ClosedPeriod{
					{
						From: periods[0].From,
						To:   cancelAt,
					},
				},
				InvoiceAt:             []time.Time{invoiceAt[0]},
				AmountBeforeProration: []alpacadecimal.Decimal{alpacadecimal.NewFromFloat(100)},
				AmountAfterProration:  []alpacadecimal.Decimal{alpacadecimal.NewFromFloat(51.72)},
			},
		})
		s.Len(remainingCharges, 1)

		deletedChargeRes, err := s.Charges.GetByID(ctx, charges.GetByIDInput{
			ChargeID: chargesmeta.ChargeID{
				Namespace: s.Namespace,
				ID:        originalSecondPeriodCharge.ID,
			},
		})
		s.NoError(err)

		deletedCharge, err := deletedChargeRes.AsFlatFeeCharge()
		s.NoError(err)
		s.Equal(chargesmeta.ChargeStatusDeleted, deletedCharge.Status)
		s.NotNil(deletedCharge.DeletedAt)
		s.Equal(expectedCharges[0].ChildUniqueReferenceIDs[1], lo.FromPtr(deletedCharge.Intent.UniqueReferenceID))
	})
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) TestCreditsOnlyUsageBasedProvisioningAndReconciliation() {
	// Given:
	// - a subscription is created with credits_only settlement
	// - the subscription is single phase with a usage based charge priced at $1 per usage
	//
	// When:
	// - the charge is provisioned for the next two billing cycles
	//
	// Then:
	// - two charges are created with matching properties and child unique reference IDs
	//
	// Given:
	// - the two expected charges already exist
	//
	// When:
	// - clock advances
	// - we reprovision the usage based charges for the next two billing cycles
	//
	// Then:
	// - the existing charges remain unchanged
	ctx := s.testContext()
	setupAt := s.mustParseTime("2024-01-01T00:00:00Z")
	startAt := s.mustParseTime("2024-02-01T00:00:00Z")
	syncUntil := s.mustParseTime("2024-04-01T00:00:00Z")

	clock.SetTime(setupAt)
	defer clock.ResetTime()

	unitPrice := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromFloat(1),
	})

	subscriptionView := s.createSubscriptionFromPlanAt(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Credits Only Usage Based",
				Key:            "credits-only-usage-based",
				Version:        1,
				Currency:       currency.USD,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name:       s.APIRequestsTotalFeature.Key,
								Key:        s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
								Price:      unitPrice,
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	}, startAt)
	timeline := timeutil.NewSimpleTimeline([]time.Time{
		s.mustParseTime("2024-02-01T00:00:00Z"),
		s.mustParseTime("2024-03-01T00:00:00Z"),
		s.mustParseTime("2024-04-01T00:00:00Z"),
	})
	periods := timeline.GetClosedPeriods()
	invoiceAt := timeline.GetTimes()[1:]

	expectedCharges := []expectedUsageBasedCharge{
		{
			ChildUniqueReferenceIDs: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			}.ChildIDs(subscriptionView.Subscription.ID),
			ServicePeriods:     periods,
			FullServicePeriods: periods,
			BillingPeriods:     periods,
			InvoiceAt:          invoiceAt,
			FeatureKey:         s.APIRequestsTotalFeature.Key,
			Price:              *unitPrice,
		},
	}

	var initialCharges []usagebased.Charge

	s.Run("provisions the next two billing cycles", func() {
		s.NoError(s.Service.SynchronizeSubscription(ctx, subscriptionView, syncUntil))
		initialCharges = s.expectCreditsOnlyUsageBasedCharges(ctx, subscriptionView.Subscription.ID, expectedCharges)
	})

	s.Run("reconciliation leaves charges unchanged", func() {
		initialUpdatedAtByID := lo.SliceToMap(initialCharges, func(charge usagebased.Charge) (string, time.Time) {
			return charge.ID, charge.UpdatedAt
		})

		clock.SetTime(s.mustParseTime("2024-01-15T00:00:00Z"))
		s.NoError(s.Service.SynchronizeSubscription(ctx, subscriptionView, syncUntil))

		reconciledCharges := s.expectCreditsOnlyUsageBasedCharges(ctx, subscriptionView.Subscription.ID, expectedCharges)
		s.Len(reconciledCharges, len(initialCharges))

		for _, charge := range reconciledCharges {
			updatedAt, ok := initialUpdatedAtByID[charge.ID]
			s.Truef(ok, "unexpected charge %s after reconciliation", charge.ID)
			s.Equal(updatedAt, charge.UpdatedAt, "charge %s should not have been updated", charge.ID)
		}
	})
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) TestCreditsOnlyMixedProvisioning() {
	// Given:
	// - a subscription is created with credits_only settlement
	// - the subscription is single phase with a usage based charge priced at $1 per usage
	// - the subscription is single phase with a flat fee charge of $100
	//
	// When:
	// - the charge is provisioned for the next two billing cycles
	//
	// Then:
	// - three charges are created with matching properties and child unique reference IDs
	// - the flat fee produces two in-advance charges, while the usage-based item produces one
	//   in-arrears charge at this sync horizon
	ctx := s.testContext()
	setupAt := s.mustParseTime("2024-01-01T00:00:00Z")
	startAt := s.mustParseTime("2024-02-01T00:00:00Z")
	syncUntil := s.mustParseTime("2024-02-15T00:00:00Z")

	clock.SetTime(setupAt)
	defer clock.ResetTime()

	unitPrice := productcatalog.NewPriceFrom(productcatalog.UnitPrice{
		Amount: alpacadecimal.NewFromFloat(1),
	})

	subscriptionView := s.createSubscriptionFromPlanAt(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: s.Namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Credits Only Mixed",
				Key:            "credits-only-mixed",
				Version:        1,
				Currency:       currency.USD,
				SettlementMode: productcatalog.CreditOnlySettlementMode,
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: s.phaseMeta("first-phase", ""),
					RateCards: productcatalog.RateCards{
						&productcatalog.FlatFeeRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name: "flat-fee",
								Key:  "flat-fee",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(100),
									PaymentTerm: productcatalog.InAdvancePaymentTerm,
								}),
							},
							BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
						},
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name:       s.APIRequestsTotalFeature.Key,
								Key:        s.APIRequestsTotalFeature.Key,
								FeatureKey: lo.ToPtr(s.APIRequestsTotalFeature.Key),
								FeatureID:  lo.ToPtr(s.APIRequestsTotalFeature.ID),
								Price:      unitPrice,
							},
							BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
						},
					},
				},
			},
		},
	}, startAt)
	timeline := timeutil.NewSimpleTimeline([]time.Time{
		s.mustParseTime("2024-02-01T00:00:00Z"),
		s.mustParseTime("2024-03-01T00:00:00Z"),
		s.mustParseTime("2024-04-01T00:00:00Z"),
	})
	periods := timeline.GetClosedPeriods()
	flatFeeInvoiceAt := timeline.GetTimes()[:len(timeline.GetTimes())-1]
	usageBasedInvoiceAt := timeline.GetTimes()[1:]

	expectedFlatFeeCharges := []expectedFlatFeeCharge{
		{
			ChildUniqueReferenceIDs: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   "flat-fee",
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			}.ChildIDs(subscriptionView.Subscription.ID),
			ServicePeriods:     periods,
			FullServicePeriods: periods,
			BillingPeriods:     periods,
			InvoiceAt:          flatFeeInvoiceAt,
		},
	}

	expectedUsageBasedCharges := []expectedUsageBasedCharge{
		(expectedUsageBasedCharge{
			ChildUniqueReferenceIDs: recurringLineMatcher{
				PhaseKey:  "first-phase",
				ItemKey:   s.APIRequestsTotalFeature.Key,
				Version:   0,
				PeriodMin: 0,
				PeriodMax: 1,
			}.ChildIDs(subscriptionView.Subscription.ID),
			ServicePeriods:     periods,
			FullServicePeriods: periods,
			BillingPeriods:     periods,
			InvoiceAt:          usageBasedInvoiceAt,
			FeatureKey:         s.APIRequestsTotalFeature.Key,
			Price:              *unitPrice,
		}).Indexes(0),
	}

	s.NoError(s.Service.SynchronizeSubscription(ctx, subscriptionView, syncUntil))
	s.expectCreditsOnlyMixedCharges(ctx, subscriptionView.Subscription.ID, expectedFlatFeeCharges, expectedUsageBasedCharges)
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) expectCreditsOnlyMixedCharges(ctx context.Context, subscriptionID string, expectedFlatFee []expectedFlatFeeCharge, expectedUsageBased []expectedUsageBasedCharge) {
	s.T().Helper()

	flatFeeResult, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{subscriptionID},
		ChargeTypes:     []chargesmeta.ChargeType{chargesmeta.ChargeTypeFlatFee},
	})
	s.NoError(err)

	usageBasedResult, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{subscriptionID},
		ChargeTypes:     []chargesmeta.ChargeType{chargesmeta.ChargeTypeUsageBased},
	})
	s.NoError(err)

	flatFeeCharges := make([]flatfee.Charge, 0, len(flatFeeResult.Items))
	for _, charge := range flatFeeResult.Items {
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		flatFeeCharges = append(flatFeeCharges, flatFeeCharge)
	}

	usageBasedCharges := make([]usagebased.Charge, 0, len(usageBasedResult.Items))
	for _, charge := range usageBasedResult.Items {
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		s.NoError(err)
		usageBasedCharges = append(usageBasedCharges, usageBasedCharge)
	}

	expectedChargeCount := lo.SumBy(expectedFlatFee, func(charge expectedFlatFeeCharge) int {
		return len(charge.ChildUniqueReferenceIDs)
	}) + lo.SumBy(expectedUsageBased, func(charge expectedUsageBasedCharge) int {
		return len(charge.ChildUniqueReferenceIDs)
	})

	slices.SortFunc(flatFeeCharges, func(left, right flatfee.Charge) int {
		return left.Intent.ServicePeriod.From.Compare(right.Intent.ServicePeriod.From)
	})
	slices.SortFunc(usageBasedCharges, func(left, right usagebased.Charge) int {
		return left.Intent.ServicePeriod.From.Compare(right.Intent.ServicePeriod.From)
	})

	s.Equal(expectedChargeCount, len(flatFeeCharges)+len(usageBasedCharges))

	s.assertExpectedFlatFeeCharges(ctx, subscriptionID, flatFeeCharges, expectedFlatFee)
	s.assertExpectedUsageBasedCharges(ctx, subscriptionID, usageBasedCharges, expectedUsageBased)
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) expectCreditsOnlyFlatFeeCharges(ctx context.Context, subscriptionID string, expected []expectedFlatFeeCharge) []flatfee.Charge {
	s.T().Helper()

	res, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{subscriptionID},
		ChargeTypes:     []chargesmeta.ChargeType{chargesmeta.ChargeTypeFlatFee},
	})
	s.NoError(err)

	out := make([]flatfee.Charge, 0, len(res.Items))
	for _, charge := range res.Items {
		flatFeeCharge, err := charge.AsFlatFeeCharge()
		s.NoError(err)
		out = append(out, flatFeeCharge)
	}

	slices.SortFunc(out, func(left, right flatfee.Charge) int {
		return left.Intent.ServicePeriod.From.Compare(right.Intent.ServicePeriod.From)
	})

	s.assertExpectedFlatFeeCharges(ctx, subscriptionID, out, expected)

	return out
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) assertExpectedFlatFeeCharges(ctx context.Context, subscriptionID string, out []flatfee.Charge, expected []expectedFlatFeeCharge) {
	s.T().Helper()

	expectedChargeCount := lo.SumBy(expected, func(charge expectedFlatFeeCharge) int {
		return len(charge.ChildUniqueReferenceIDs)
	})
	s.Len(out, expectedChargeCount)

	for expectedIdx, expectedCharge := range expected {
		for periodIdx, childID := range expectedCharge.ChildUniqueReferenceIDs {
			charge, found := lo.Find(out, func(charge flatfee.Charge) bool {
				return charge.Intent.UniqueReferenceID != nil && *charge.Intent.UniqueReferenceID == childID
			})
			if !found {
				s.T().Fatalf("expected[%d] charge[%d] not found with child unique reference id %s", expectedIdx, periodIdx, childID)
			}
			expectedPhaseID := s.getExpectedPhaseIDForChildReference(ctx, subscriptionID, childID)

			s.NotNilf(charge.Intent.UniqueReferenceID, "expected[%d] charge[%d] should have child unique reference id", expectedIdx, periodIdx)
			s.Equalf(childID, lo.FromPtr(charge.Intent.UniqueReferenceID), "expected[%d] charge[%d] child unique reference id", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.ServicePeriods[periodIdx], charge.Intent.ServicePeriod, "expected[%d] charge[%d] service period", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.FullServicePeriods[periodIdx], charge.Intent.FullServicePeriod, "expected[%d] charge[%d] full service period", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.BillingPeriods[periodIdx], charge.Intent.BillingPeriod, "expected[%d] charge[%d] billing period", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.InvoiceAt[periodIdx], charge.Intent.InvoiceAt, "expected[%d] charge[%d] invoice at", expectedIdx, periodIdx)
			s.Equalf(productcatalog.CreditOnlySettlementMode, charge.Intent.SettlementMode, "expected[%d] charge[%d] settlement mode", expectedIdx, periodIdx)
			s.Equalf(productcatalog.InAdvancePaymentTerm, charge.Intent.PaymentTerm, "expected[%d] charge[%d] payment term", expectedIdx, periodIdx)
			s.Equalf(string(currency.USD), string(charge.Intent.Currency), "expected[%d] charge[%d] currency", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.amountBeforeProration(periodIdx), charge.Intent.AmountBeforeProration, "expected[%d] charge[%d] amount before proration", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.amountAfterProration(periodIdx), charge.State.AmountAfterProration, "expected[%d] charge[%d] amount after proration", expectedIdx, periodIdx)
			s.Equalf(subscriptionID, charge.Intent.Subscription.SubscriptionID, "expected[%d] charge[%d] subscription id", expectedIdx, periodIdx)
			s.Equalf(expectedPhaseID, charge.Intent.Subscription.PhaseID, "expected[%d] charge[%d] subscription phase id", expectedIdx, periodIdx)
			s.Equalf("flat-fee", charge.Intent.Name, "expected[%d] charge[%d] charge name", expectedIdx, periodIdx)
		}
	}
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) expectCreditsOnlyUsageBasedCharges(ctx context.Context, subscriptionID string, expected []expectedUsageBasedCharge) []usagebased.Charge {
	s.T().Helper()

	res, err := s.Charges.ListCharges(ctx, charges.ListChargesInput{
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{subscriptionID},
		ChargeTypes:     []chargesmeta.ChargeType{chargesmeta.ChargeTypeUsageBased},
	})
	s.NoError(err)

	out := make([]usagebased.Charge, 0, len(res.Items))
	for _, charge := range res.Items {
		usageBasedCharge, err := charge.AsUsageBasedCharge()
		s.NoError(err)
		out = append(out, usageBasedCharge)
	}

	slices.SortFunc(out, func(left, right usagebased.Charge) int {
		return left.Intent.ServicePeriod.From.Compare(right.Intent.ServicePeriod.From)
	})

	s.assertExpectedUsageBasedCharges(ctx, subscriptionID, out, expected)

	return out
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) assertExpectedUsageBasedCharges(ctx context.Context, subscriptionID string, out []usagebased.Charge, expected []expectedUsageBasedCharge) {
	s.T().Helper()

	expectedChargeCount := lo.SumBy(expected, func(charge expectedUsageBasedCharge) int {
		return len(charge.ChildUniqueReferenceIDs)
	})
	s.Len(out, expectedChargeCount)

	for expectedIdx, expectedCharge := range expected {
		for periodIdx, childID := range expectedCharge.ChildUniqueReferenceIDs {
			charge, found := lo.Find(out, func(charge usagebased.Charge) bool {
				return charge.Intent.UniqueReferenceID != nil && *charge.Intent.UniqueReferenceID == childID
			})
			if !found {
				s.T().Fatalf("expected[%d] charge[%d] not found with child unique reference id %s", expectedIdx, periodIdx, childID)
			}
			expectedPhaseID := s.getExpectedPhaseIDForChildReference(ctx, subscriptionID, childID)

			s.NotNilf(charge.Intent.UniqueReferenceID, "expected[%d] charge[%d] should have child unique reference id", expectedIdx, periodIdx)
			s.Equalf(childID, lo.FromPtr(charge.Intent.UniqueReferenceID), "expected[%d] charge[%d] child unique reference id", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.ServicePeriods[periodIdx], charge.Intent.ServicePeriod, "expected[%d] charge[%d] service period", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.FullServicePeriods[periodIdx], charge.Intent.FullServicePeriod, "expected[%d] charge[%d] full service period", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.BillingPeriods[periodIdx], charge.Intent.BillingPeriod, "expected[%d] charge[%d] billing period", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.InvoiceAt[periodIdx], charge.Intent.InvoiceAt, "expected[%d] charge[%d] invoice at", expectedIdx, periodIdx)
			s.Equalf(productcatalog.CreditOnlySettlementMode, charge.Intent.SettlementMode, "expected[%d] charge[%d] settlement mode", expectedIdx, periodIdx)
			s.Equalf(string(currency.USD), string(charge.Intent.Currency), "expected[%d] charge[%d] currency", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.FeatureKey, charge.Intent.FeatureKey, "expected[%d] charge[%d] feature key", expectedIdx, periodIdx)
			s.Equalf(expectedCharge.Price, charge.Intent.Price, "expected[%d] charge[%d] price", expectedIdx, periodIdx)
			s.Equalf(subscriptionID, charge.Intent.Subscription.SubscriptionID, "expected[%d] charge[%d] subscription id", expectedIdx, periodIdx)
			s.Equalf(expectedPhaseID, charge.Intent.Subscription.PhaseID, "expected[%d] charge[%d] subscription phase id", expectedIdx, periodIdx)
			s.Equalf(s.APIRequestsTotalFeature.Key, charge.Intent.Name, "expected[%d] charge[%d] charge name", expectedIdx, periodIdx)
		}
	}
}

func (s *CreditsOnlySubscriptionHandlerTestSuite) getExpectedPhaseIDForChildReference(ctx context.Context, subscriptionID string, childID string) string {
	s.T().Helper()

	parts := strings.Split(childID, "/")
	s.Len(parts, 5, "invalid child unique reference id format")
	s.Equal(subscriptionID, parts[0], "child unique reference id subscription id")

	subscriptionView, err := s.SubscriptionService.GetView(ctx, models.NamespacedID{
		Namespace: s.Namespace,
		ID:        subscriptionID,
	})
	s.NoError(err)

	return s.getPhaseByKey(s.T(), subscriptionView, parts[1]).SubscriptionPhase.ID
}
