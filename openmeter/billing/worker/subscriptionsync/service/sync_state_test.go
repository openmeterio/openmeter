package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/lib/pq"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/persistedstate"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/reconciler"
	"github.com/openmeterio/openmeter/openmeter/billing/worker/subscriptionsync/service/targetstate"
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

func TestNextReconciliationAt(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(10 * 24 * time.Hour)
	period := timeutil.ClosedPeriod{From: start, To: end}
	for _, tc := range []struct {
		name              string
		nextBillingPeriod *timeutil.ClosedPeriod
		reconciledAt      time.Time
		activeTo          *time.Time
		fraction          float64
		want              time.Time
	}{
		{
			name:              "ongoing subscription uses eighty percent of the whole period",
			nextBillingPeriod: &period,
			reconciledAt:      start.Add(5 * 24 * time.Hour),
			want:              start.Add(8 * 24 * time.Hour),
		},
		{
			name:              "ongoing subscription uses ninety-five percent of the whole period",
			nextBillingPeriod: &period,
			reconciledAt:      start,
			fraction:          1,
			want:              start.Add(228 * time.Hour),
		},
		{
			name:              "cancellation after the next period still uses jitter",
			nextBillingPeriod: &period,
			reconciledAt:      start,
			activeTo:          lo.ToPtr(end.Add(24 * time.Hour)),
			want:              start.Add(8 * 24 * time.Hour),
		},
		{
			name:         "no next period schedules the subscription end exactly",
			reconciledAt: start,
			activeTo:     &end,
			want:         end,
		},
		{
			name:         "no next period preserves an end before the billing boundary",
			reconciledAt: start,
			activeTo:     lo.ToPtr(end.Add(-time.Hour)),
			want:         end.Add(-time.Hour),
		},
		{
			name:              "selected deadline equal to completion moves to the period end",
			nextBillingPeriod: &period,
			reconciledAt:      start.Add(8 * 24 * time.Hour),
			want:              end,
		},
		{
			name:              "late sync does not reschedule a deadline in the past",
			nextBillingPeriod: &period,
			reconciledAt:      start.Add(9 * 24 * time.Hour),
			want:              end,
		},
		{
			name:              "historical period remains due",
			nextBillingPeriod: &period,
			reconciledAt:      end.Add(time.Hour),
			want:              end,
		},
		{
			name:         "historical subscription end remains due",
			reconciledAt: end.Add(time.Hour),
			activeTo:     &end,
			want:         end,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &Service{}
			got, err := service.getNextReconciliationAt(nextReconciliationAtInput{
				NextBillingPeriod:    tc.nextBillingPeriod,
				SubscriptionActiveTo: tc.activeTo,
				ReconciledAt:         tc.reconciledAt,
				RandomFraction:       tc.fraction,
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestNextReconciliationAtRequiresPeriodOrEnd(t *testing.T) {
	for _, tc := range []struct {
		name     string
		activeTo *time.Time
	}{
		{name: "missing end"},
		{name: "zero end", activeTo: lo.ToPtr(time.Time{})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// An active result cannot lose its deadline when the plan has no next period.
			service := &Service{}
			_, err := service.getNextReconciliationAt(nextReconciliationAtInput{
				SubscriptionActiveTo: tc.activeTo,
				ReconciledAt:         time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
			})
			require.ErrorContains(t, err, "active to is required when there is no next billing period")
		})
	}
}

// SyncStateTestSuite follows the deadlines recorded by actual subscription syncs.
// The shared fixture provides both legacy billing and the real charge/ledger services.
type SyncStateTestSuite struct {
	CreditThenInvoiceSuiteBase
}

func TestSyncState(t *testing.T) {
	suite.Run(t, new(SyncStateTestSuite))
}

func (s *SyncStateTestSuite) BeforeTest(suiteName, testName string) {
	s.CreditThenInvoiceSuiteBase.BeforeTest(suiteName, testName)
	s.Service = s.LegacySyncService
}

type syncStateSubscriptionInput struct {
	Start  time.Time
	Phases []productcatalog.Phase
}

func (s *SyncStateTestSuite) createSyncStateSubscription(input syncStateSubscriptionInput) subscription.SubscriptionView {
	s.T().Helper()

	return s.createSubscriptionFromPlanAt(plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: s.Namespace},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Monthly reconciliation",
				Key:            ulid.Make().String(),
				Version:        1,
				Currency:       currencies.NewCurrencyReference(currencyx.Code("USD")),
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			},
			Phases: input.Phases,
		},
	}, input.Start)
}

func (s *SyncStateTestSuite) paidMonthlyPhase() productcatalog.Phase {
	s.T().Helper()

	return productcatalog.Phase{
		PhaseMeta: s.phaseMeta("paid", ""),
		RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:  "monthly-fee",
				Name: "Monthly fee",
				Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
					Amount:      alpacadecimal.NewFromInt(10),
					PaymentTerm: productcatalog.InAdvancePaymentTerm,
				}),
			},
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}},
	}
}

func (s *SyncStateTestSuite) freeMonthlyPhase(duration string) productcatalog.Phase {
	s.T().Helper()

	return productcatalog.Phase{
		PhaseMeta: s.phaseMeta("free", duration),
		RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
			RateCardMeta:   productcatalog.RateCardMeta{Key: "free-item", Name: "Free item"},
			BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
		}},
	}
}

// requireActiveDeadline checks the persisted result against the expected calendar
// period. Subsequent syncs use the returned deadline, including its random offset.
func (s *SyncStateTestSuite) requireActiveDeadline(subscriptionID models.NamespacedID, period timeutil.ClosedPeriod) time.Time {
	s.T().Helper()
	state := s.getSyncStateV2(subscriptionID)
	s.Equal(subscriptionsync.SyncStateV2StateActive, state.LastSuccessState)
	s.Equal(lo.ToPtr(clock.Now().UTC()), state.LastSuccessAt)
	s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
	s.Require().NotNil(state.NextReconciliationAt)
	duration := period.To.Sub(period.From)
	s.Require().False(state.NextReconciliationAt.Before(period.From.Add(duration*80/100)), "deadline %s is before the 80%% point of %v", state.NextReconciliationAt, period)
	s.Require().False(state.NextReconciliationAt.After(period.From.Add(duration*95/100)), "deadline %s is after the 95%% point of %v", state.NextReconciliationAt, period)
	s.Require().True(state.NextReconciliationAt.After(clock.Now()), "next sync must advance the clock")

	return *state.NextReconciliationAt
}

// requireMonthlyItems checks all provisioned periods, so duplicate or missing
// upcoming items fail even if the reconciliation deadline itself is correct.
func (s *SyncStateTestSuite) requireMonthlyItems(view subscription.SubscriptionView, periods []timeutil.ClosedPeriod) {
	s.T().Helper()
	if s.Service == s.ChargeSyncService {
		result, err := s.Charges.ListCharges(s.T().Context(), charges.ListChargesInput{
			Namespace:       s.Namespace,
			SubscriptionIDs: []string{view.Subscription.ID},
		})
		s.Require().NoError(err)
		s.Require().Len(result.Items, len(periods))
		expectedIDs := (recurringLineMatcher{PhaseKey: "paid", ItemKey: "monthly-fee", PeriodMax: len(periods) - 1}).ChildIDs(view.Subscription.ID)
		actualIDs := make([]string, 0, len(result.Items))
		actualPeriods := make([]timeutil.ClosedPeriod, 0, len(result.Items))
		for _, item := range result.Items {
			charge, err := item.AsFlatFeeCharge()
			s.Require().NoError(err)
			intent := charge.Intent.GetBaseIntent()
			s.Equal(intent.ServicePeriod.From, intent.InvoiceAt, "in-advance fee is invoiceable at its service period start")
			actualPeriods = append(actualPeriods, intent.ServicePeriod)
			reference, err := item.GetUniqueReferenceID()
			s.Require().NoError(err)
			s.Require().NotNil(reference)
			actualIDs = append(actualIDs, *reference)
		}

		s.ElementsMatch(expectedIDs, actualIDs)
		s.ElementsMatch(periods, actualPeriods)

		return
	}

	invoice := s.gatheringInvoice(s.T().Context(), s.Namespace, s.Customer.ID)
	s.expectLines(invoice, view.Subscription.ID, []expectedLine{{
		Matcher:   recurringLineMatcher{PhaseKey: "paid", ItemKey: "monthly-fee", PeriodMax: len(periods) - 1},
		Periods:   periods,
		InvoiceAt: mo.Some(lo.Map(periods, func(period timeutil.ClosedPeriod, _ int) time.Time { return period.From })),
	}})
}

func (s *SyncStateTestSuite) TestMonthlyLegacyProgression() {
	s.requireMonthlyProgression()
}

func (s *SyncStateTestSuite) TestMonthlyChargeProgression() {
	s.Service = s.ChargeSyncService
	s.requireMonthlyProgression()
}

func (s *SyncStateTestSuite) requireMonthlyProgression() {
	s.T().Helper()
	// Given a monthly in-advance fee, beginning November 1.
	start := s.mustParseTime("2026-11-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{Start: start, Phases: []productcatalog.Phase{s.paidMonthlyPhase()}})

	// When each persisted deadline arrives, sync provisions the next month's fee.
	// February's shorter period also verifies that scheduling follows the calendar.
	periods := []timeutil.ClosedPeriod{
		{From: start, To: s.mustParseTime("2026-12-01T00:00:00Z")},
		{From: s.mustParseTime("2026-12-01T00:00:00Z"), To: s.mustParseTime("2027-01-01T00:00:00Z")},
		{From: s.mustParseTime("2027-01-01T00:00:00Z"), To: s.mustParseTime("2027-02-01T00:00:00Z")},
		{From: s.mustParseTime("2027-02-01T00:00:00Z"), To: s.mustParseTime("2027-03-01T00:00:00Z")},
		{From: s.mustParseTime("2027-03-01T00:00:00Z"), To: s.mustParseTime("2027-04-01T00:00:00Z")},
	}
	syncAt := start
	for index, period := range periods {
		s.Run(period.From.Format("January"), func() {
			clock.FreezeTime(syncAt)
			defer clock.UnFreeze()
			s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))

			// Then all prior fees remain, the upcoming fee is provisioned exactly once,
			// and the next invocation belongs to that upcoming billing period.
			s.requireMonthlyItems(view, periods[:index+1])
			syncAt = s.requireActiveDeadline(view.Subscription.NamespacedID, period)
		})
	}
}

func (s *SyncStateTestSuite) TestFutureStartProgression() {
	// Given a subscription starting December 1, created in November.
	now := s.mustParseTime("2026-11-15T00:00:00Z")
	start := s.mustParseTime("2026-12-01T00:00:00Z")
	january := s.mustParseTime("2027-01-01T00:00:00Z")
	february := s.mustParseTime("2027-02-01T00:00:00Z")
	clock.FreezeTime(now)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{Start: start, Phases: []productcatalog.Phase{s.paidMonthlyPhase()}})

	// When synced before its start, the first fee and first-period deadline are provisioned.
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	firstPeriod := timeutil.ClosedPeriod{From: start, To: january}
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{firstPeriod})
	next := s.requireActiveDeadline(view.Subscription.NamespacedID, firstPeriod)

	// Then following that deadline provisions January and schedules within January.
	clock.FreezeTime(next)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	secondPeriod := timeutil.ClosedPeriod{From: january, To: february}
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{firstPeriod, secondPeriod})
	s.requireActiveDeadline(view.Subscription.NamespacedID, secondPeriod)
}

func (s *SyncStateTestSuite) TestFreeToPaidProgression() {
	// Given a free November followed by a monthly in-advance fee from December.
	start := s.mustParseTime("2026-11-01T00:00:00Z")
	december := s.mustParseTime("2026-12-01T00:00:00Z")
	january := s.mustParseTime("2027-01-01T00:00:00Z")
	february := s.mustParseTime("2027-02-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{
		Start: start,
		Phases: []productcatalog.Phase{
			s.freeMonthlyPhase("P1M"),
			s.paidMonthlyPhase(),
		},
	})

	// When syncing during the free phase, the first paid period is already provisioned.
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	firstPaidPeriod := timeutil.ClosedPeriod{From: december, To: january}
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{firstPaidPeriod})
	next := s.requireActiveDeadline(view.Subscription.NamespacedID, firstPaidPeriod)

	// Then its persisted deadline provisions the second paid period and advances scheduling.
	clock.FreezeTime(next)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	secondPaidPeriod := timeutil.ClosedPeriod{From: january, To: february}
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{firstPaidPeriod, secondPaidPeriod})
	s.requireActiveDeadline(view.Subscription.NamespacedID, secondPaidPeriod)
}

func (s *SyncStateTestSuite) TestPaidToPaidPhaseTransitionProvisionsAdvanceFeeOnTime() {
	for _, tc := range []struct {
		name    string
		service *Service
		asOf    time.Time
	}{
		{
			name:    "legacy billing before the phase boundary",
			service: s.LegacySyncService,
			asOf:    s.mustParseTime("2026-11-25T00:00:00Z"),
		},
		{
			name:    "charges before the phase boundary",
			service: s.ChargeSyncService,
			asOf:    s.mustParseTime("2026-11-25T00:00:00Z"),
		},
		{
			name:    "legacy billing at the phase boundary",
			service: s.LegacySyncService,
			asOf:    s.mustParseTime("2026-12-01T00:00:00Z"),
		},
		{
			name:    "charges at the phase boundary",
			service: s.ChargeSyncService,
			asOf:    s.mustParseTime("2026-12-01T00:00:00Z"),
		},
	} {
		s.Run(tc.name, func() {
			// Given two paid phases, the second phase's advance fee is due on December 1.
			ctx := s.T().Context()
			start := s.mustParseTime("2026-11-01T00:00:00Z")
			december := s.mustParseTime("2026-12-01T00:00:00Z")
			january := s.mustParseTime("2027-01-01T00:00:00Z")
			february := s.mustParseTime("2027-02-01T00:00:00Z")
			clock.FreezeTime(start)
			defer clock.UnFreeze()
			s.Service = tc.service
			s.Customer = s.CreateTestCustomer(s.Namespace, ulid.Make().String())
			_, err := s.LedgerResolver.CreateCustomerAccounts(ctx, s.Customer.GetID())
			s.Require().NoError(err)

			firstPhase := s.paidMonthlyPhase()
			firstPhase.PhaseMeta = s.phaseMeta("first-paid", "P1M")
			view := s.createSyncStateSubscription(syncStateSubscriptionInput{
				Start: start,
				Phases: []productcatalog.Phase{
					firstPhase,
					s.paidMonthlyPhase(),
				},
			})
			firstFeeID := (recurringLineMatcher{PhaseKey: "first-paid", ItemKey: "monthly-fee"}).ChildIDs(view.Subscription.ID)[0]
			secondPhaseFeeIDs := (recurringLineMatcher{PhaseKey: "paid", ItemKey: "monthly-fee", PeriodMax: 1}).ChildIDs(view.Subscription.ID)
			expectedFees := map[string]timeutil.ClosedPeriod{
				firstFeeID:           {From: start, To: december},
				secondPhaseFeeIDs[0]: {From: december, To: january},
			}

			// When sync generates through December 1, it must include the second phase's fee.
			clock.FreezeTime(tc.asOf)
			defer clock.UnFreeze()
			s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now()))
			s.requirePhaseTransitionFees(view, expectedFees)
			next := s.requireActiveDeadline(view.Subscription.NamespacedID, timeutil.ClosedPeriod{From: december, To: january})

			// Then following the December deadline adds January's fee without duplicating earlier fees.
			clock.FreezeTime(next)
			defer clock.UnFreeze()
			s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now()))
			expectedFees[secondPhaseFeeIDs[1]] = timeutil.ClosedPeriod{From: january, To: february}
			s.requirePhaseTransitionFees(view, expectedFees)
			s.requireActiveDeadline(view.Subscription.NamespacedID, timeutil.ClosedPeriod{From: january, To: february})
		})
	}
}

func (s *SyncStateTestSuite) requirePhaseTransitionFees(view subscription.SubscriptionView, expected map[string]timeutil.ClosedPeriod) {
	s.T().Helper()
	actual := make(map[string]timeutil.ClosedPeriod)
	if s.Service == s.ChargeSyncService {
		result, err := s.Charges.ListCharges(s.T().Context(), charges.ListChargesInput{
			Namespace:       s.Namespace,
			SubscriptionIDs: []string{view.Subscription.ID},
		})
		s.Require().NoError(err)
		s.Require().Len(result.Items, len(expected))
		for _, item := range result.Items {
			charge, err := item.AsFlatFeeCharge()
			s.Require().NoError(err)
			intent := charge.Intent.GetBaseIntent()
			reference, err := item.GetUniqueReferenceID()
			s.Require().NoError(err)
			s.Require().NotNil(reference)
			s.Equal(intent.ServicePeriod.From, intent.InvoiceAt)
			actual[*reference] = intent.ServicePeriod
		}
	} else {
		invoice := s.gatheringInvoice(s.T().Context(), s.Namespace, s.Customer.ID)
		s.Require().Len(invoice.Lines.OrEmpty(), len(expected))
		for _, line := range invoice.Lines.OrEmpty() {
			s.Require().NotNil(line.ChildUniqueReferenceID)
			s.Equal(line.GetServicePeriod().From, line.InvoiceAt)
			actual[*line.ChildUniqueReferenceID] = line.GetServicePeriod()
		}
	}

	s.Equal(expected, actual)
}

func (s *SyncStateTestSuite) TestNoBillablesHasNoReconciliationDeadline() {
	// Given a subscription with a monthly billing calendar but no priced items.
	start := s.mustParseTime("2026-11-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{
		Start:  start,
		Phases: []productcatalog.Phase{s.freeMonthlyPhase("")},
	})

	// When sync completes, it acknowledges the subscription without scheduling more work.
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	state := s.getSyncStateV2(view.Subscription.NamespacedID)
	s.Equal(subscriptionsync.SyncStateV2StateNoBillables, state.LastSuccessState)
	s.Equal(&start, state.LastSuccessAt)
	s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
	s.Nil(state.NextReconciliationAt)

	// Then neither legacy billing nor charge-based billing has artifacts to collect.
	s.expectNoGatheringInvoice(s.T().Context(), s.Namespace, s.Customer.ID)
	result, err := s.Charges.ListCharges(s.T().Context(), charges.ListChargesInput{
		Namespace:       s.Namespace,
		SubscriptionIDs: []string{view.Subscription.ID},
	})
	s.Require().NoError(err)
	s.Empty(result.Items)
}

func (s *SyncStateTestSuite) TestRemovingAndRestoringBillablesUpdatesReconciliation() {
	ctx := s.T().Context()
	now := s.mustParseTime("2026-11-01T00:00:00Z")
	start := now
	end := s.mustParseTime("2026-12-01T00:00:00Z")
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given a paid subscription at its start, with its first fee and active deadline.
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{
		Start:  start,
		Phases: []productcatalog.Phase{s.paidMonthlyPhase()},
	})
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, now))
	period := timeutil.ClosedPeriod{From: start, To: end}
	s.requireActiveDeadline(view.Subscription.NamespacedID, period)
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{period})

	// When the last price is removed, sync must still delete the provisioned fee
	// before recording no_billables and clearing the previously saved deadline.
	currentView, err := s.SubscriptionService.GetView(ctx, view.Subscription.NamespacedID)
	s.Require().NoError(err)
	freeSpec := currentView.AsSpec()
	freeItem := freeSpec.Phases["paid"].ItemsByKey["monthly-fee"][0]
	freeRateCard := freeItem.RateCard.Clone().(*productcatalog.FlatFeeRateCard)
	freeRateCard.Price = nil
	freeRateCard.Currency = nil
	freeItem.RateCard = freeRateCard
	_, err = s.SubscriptionService.Update(ctx, view.Subscription.NamespacedID, freeSpec)
	s.Require().NoError(err)
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, now))
	state := s.getSyncStateV2(view.Subscription.NamespacedID)
	s.Equal(subscriptionsync.SyncStateV2StateNoBillables, state.LastSuccessState)
	s.Equal(&now, state.LastSuccessAt)
	s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
	s.Nil(state.NextReconciliationAt)
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)

	// Then a later edit restoring the price can provision the fee and resume scheduling.
	updatedView, err := s.SubscriptionService.GetView(ctx, view.Subscription.NamespacedID)
	s.Require().NoError(err)
	paidSpec := updatedView.AsSpec()
	paidItem := paidSpec.Phases["paid"].ItemsByKey["monthly-fee"][0]
	paidRateCard := paidItem.RateCard.Clone().(*productcatalog.FlatFeeRateCard)
	paidRateCard.Price = productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(10),
		PaymentTerm: productcatalog.InAdvancePaymentTerm,
	})
	paidItem.RateCard = paidRateCard
	_, err = s.SubscriptionService.Update(ctx, view.Subscription.NamespacedID, paidSpec)
	s.Require().NoError(err)
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, now))
	s.requireActiveDeadline(view.Subscription.NamespacedID, period)
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{period})
}

func (s *SyncStateTestSuite) TestOneTimeFeeWaitsForCancellation() {
	ctx := s.T().Context()
	start := s.mustParseTime("2027-01-01T00:00:00Z")
	february := s.mustParseTime("2027-02-01T00:00:00Z")
	march := s.mustParseTime("2027-03-01T00:00:00Z")
	end := s.mustParseTime("2027-02-15T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// Given an open-ended phase with only a non-recurring fee billed in arrears.
	// Its service end is unknown, so no billing line can be provisioned yet.
	price := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount:      alpacadecimal.NewFromInt(10),
		PaymentTerm: productcatalog.InArrearsPaymentTerm,
	})
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{
		Start: start,
		Phases: []productcatalog.Phase{{
			PhaseMeta: s.phaseMeta("one-time", ""),
			RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Key:   "completion-fee",
					Name:  "Completion fee",
					Price: price,
				},
			}},
		}},
	})

	// When the subscription starts, sync remains active with a January deadline.
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now()))
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	next := s.requireActiveDeadline(view.Subscription.NamespacedID, timeutil.ClosedPeriod{From: start, To: february})

	// The saved January deadline still produces no line, but schedules February's sync.
	clock.FreezeTime(next)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now()))
	s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	next = s.requireActiveDeadline(view.Subscription.NamespacedID, timeutil.ClosedPeriod{From: february, To: march})
	s.True(next.After(end), "cancellation happens before the next periodic sync")

	// When canceled immediately on February 15, the cancellation event triggers sync
	// before the next periodic deadline. The fee now has a known service end.
	clock.FreezeTime(end)
	defer clock.UnFreeze()
	_, err := s.SubscriptionService.Cancel(ctx, view.Subscription.NamespacedID, subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)})
	s.Require().NoError(err)
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now()))
	expectedLines := []expectedLine{{
		Matcher:   oneTimeLineMatcher{PhaseKey: "one-time", ItemKey: "completion-fee"},
		Price:     mo.Some(price),
		Periods:   []timeutil.ClosedPeriod{{From: start, To: end}},
		InvoiceAt: mo.Some([]time.Time{end}),
	}}
	invoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.expectLines(invoice, view.Subscription.ID, expectedLines)
	state := s.getSyncStateV2(view.Subscription.NamespacedID)
	s.Equal(subscriptionsync.SyncStateV2StateEnded, state.LastSuccessState)
	s.Equal(&end, state.LastSuccessAt)
	s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
	s.Nil(state.NextReconciliationAt)

	// Then a retried cancellation sync keeps the single fee line and stays ended.
	clock.FreezeTime(end)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now()))
	finalInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.expectLines(finalInvoice, view.Subscription.ID, expectedLines)
	s.Equal(invoice.Lines.OrEmpty()[0].ID, finalInvoice.Lines.OrEmpty()[0].ID)
	state = s.getSyncStateV2(view.Subscription.NamespacedID)
	s.Equal(subscriptionsync.SyncStateV2StateEnded, state.LastSuccessState)
	s.Equal(&end, state.LastSuccessAt)
	s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
	s.Nil(state.NextReconciliationAt)
}

func (s *SyncStateTestSuite) TestCancellationProgression() {
	// Given a subscription ending February 1, with one final reconciliation required.
	start := s.mustParseTime("2026-11-01T00:00:00Z")
	december := s.mustParseTime("2026-12-01T00:00:00Z")
	january := s.mustParseTime("2027-01-01T00:00:00Z")
	end := s.mustParseTime("2027-02-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{Start: start, Phases: []productcatalog.Phase{s.paidMonthlyPhase()}})
	_, err := s.SubscriptionService.Cancel(s.T().Context(), view.Subscription.NamespacedID, subscription.Timing{Custom: &end})
	s.Require().NoError(err)

	// When successive saved deadlines arrive, November and December use periodic scheduling.
	periods := []timeutil.ClosedPeriod{{From: start, To: december}, {From: december, To: january}, {From: january, To: end}}
	syncAt := start
	for index, period := range periods[:2] {
		s.Run(period.From.Format("January"), func() {
			clock.FreezeTime(syncAt)
			defer clock.UnFreeze()
			s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
			s.requireMonthlyItems(view, periods[:index+1])
			syncAt = s.requireActiveDeadline(view.Subscription.NamespacedID, period)
		})
	}

	clock.FreezeTime(syncAt)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	s.requireMonthlyItems(view, periods)
	state := s.getSyncStateV2(view.Subscription.NamespacedID)
	s.Equal(subscriptionsync.SyncStateV2StateActive, state.LastSuccessState)
	s.Require().Equal(&end, state.NextReconciliationAt)

	// Then the final saved deadline records completion without provisioning February.
	clock.FreezeTime(*state.NextReconciliationAt)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	s.requireMonthlyItems(view, periods)
	state = s.getSyncStateV2(view.Subscription.NamespacedID)
	s.Equal(subscriptionsync.SyncStateV2StateEnded, state.LastSuccessState)
	s.Equal(&end, state.LastSuccessAt)
	s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
	s.Nil(state.NextReconciliationAt)
}

func (s *SyncStateTestSuite) TestRepeatedSyncAndDryRun() {
	// Given a successfully synced subscription with its first fee provisioned.
	start := s.mustParseTime("2026-11-01T00:00:00Z")
	end := s.mustParseTime("2026-12-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{Start: start, Phases: []productcatalog.Phase{s.paidMonthlyPhase()}})
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	initial := s.getSyncStateV2(view.Subscription.NamespacedID)

	// When the identical horizon is synced later, the empty billing diff still records success.
	clock.FreezeTime(start.Add(time.Minute))
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, start))
	next := s.requireActiveDeadline(view.Subscription.NamespacedID, timeutil.ClosedPeriod{From: start, To: end})
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{{From: start, To: end}})
	before := s.getSyncStateV2(view.Subscription.NamespacedID)
	s.NotEqual(initial.LastSuccessAt, before.LastSuccessAt)

	// Then a dry run at that deadline cannot advance success or provision the upcoming fee.
	clock.FreezeTime(next)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now(), subscriptionsync.EnableDryRun()))
	s.Equal(before, s.getSyncStateV2(view.Subscription.NamespacedID))
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{{From: start, To: end}})
}
func (s *SyncStateTestSuite) TestCustomerDeletionSetsExactReconciliationDeadline() {
	createdAt := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		name      string
		deletedAt time.Time
		asOf      time.Time
	}{
		{
			name:      "future deletion",
			deletedAt: createdAt.AddDate(0, 1, 0),
			asOf:      createdAt.Add(14 * 24 * time.Hour),
		},
		{
			name:      "past deletion",
			deletedAt: createdAt.AddDate(0, 1, 0),
			asOf:      createdAt.AddDate(0, 1, 0).Add(14 * 24 * time.Hour),
		},
	} {
		s.Run(tc.name, func() {
			// Given an open-ended subscription whose customer was deleted at a billing boundary.
			ctx := s.T().Context()
			clock.FreezeTime(createdAt)
			defer clock.UnFreeze()
			s.Customer = s.CreateTestCustomer(s.Namespace, ulid.Make().String())
			view := s.createSyncStateSubscription(syncStateSubscriptionInput{Start: createdAt, Phases: []productcatalog.Phase{s.paidMonthlyPhase()}})
			s.Require().Nil(view.Spec.ActiveTo)
			// Deleted customers no longer resolve subject keys, so keep their invoice snapshot identifiable.
			s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).
				SetKey(s.Customer.ID).
				SetDeletedAt(tc.deletedAt).Exec(ctx))
			clock.FreezeTime(tc.asOf)
			defer clock.UnFreeze()

			// When the next billing period is nil, use the target's customer-deletion cutoff.
			s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, tc.asOf))
			state := s.getSyncStateV2(view.Subscription.NamespacedID)
			s.Equal(subscriptionsync.SyncStateV2StateActive, state.LastSuccessState)
			s.Equal(&tc.deletedAt, state.NextReconciliationAt)
		})
	}
}

func (s *SyncStateTestSuite) TestActiveSyncResultRollback() {
	ctx := s.T().Context()
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given a successfully synced subscription whose next success write will fail in PostgreSQL.
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{Start: now, Phases: []productcatalog.Phase{s.paidMonthlyPhase()}})
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, now))
	before := s.getSyncStateV2(view.Subscription.NamespacedID)
	legacyBefore, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{view.Subscription.NamespacedID})
	s.Require().NoError(err)
	s.Require().NotNil(before.NextReconciliationAt)
	const constraintName = "test_active_sync_result_failure"
	_, err = s.TestDB.PGDriver.DB().ExecContext(ctx, fmt.Sprintf(
		"ALTER TABLE subscription_sync_states_v2 ADD CONSTRAINT %s CHECK (subscription_id <> %s OR last_success_at <= %s::timestamptz)",
		pq.QuoteIdentifier(constraintName), pq.QuoteLiteral(view.Subscription.ID), pq.QuoteLiteral(now.Format(time.RFC3339))))
	s.Require().NoError(err)
	defer func() {
		_, cleanupErr := s.TestDB.PGDriver.DB().ExecContext(ctx, fmt.Sprintf(
			"ALTER TABLE subscription_sync_states_v2 DROP CONSTRAINT IF EXISTS %s", pq.QuoteIdentifier(constraintName)))
		s.NoError(cleanupErr)
	}()

	// When its scheduled sync cannot record success, the upcoming fee also rolls back.
	clock.FreezeTime(*before.NextReconciliationAt)
	defer clock.UnFreeze()
	s.Require().ErrorContains(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now()), constraintName)

	// Then the first fee and both previously successful sync states are preserved.
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{{From: now, To: now.AddDate(0, 1, 0)}})
	states, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{view.Subscription.NamespacedID})
	s.Require().NoError(err)
	s.Equal(legacyBefore, states)
	s.Equal(before, s.getSyncStateV2(view.Subscription.NamespacedID))
}

type nilPlanReconciler struct {
	reconciler.Reconciler
}

func (nilPlanReconciler) Plan(context.Context, reconciler.PlanInput) (*reconciler.Plan, error) {
	return nil, nil
}

func (s *SyncStateTestSuite) TestNilPlanPreservesPreviousSyncResult() {
	ctx := s.T().Context()
	start := s.mustParseTime("2026-11-01T00:00:00Z")
	end := s.mustParseTime("2026-12-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// Given a successful sync with an existing fee and reconciliation deadline.
	view := s.createSyncStateSubscription(syncStateSubscriptionInput{
		Start:  start,
		Phases: []productcatalog.Phase{s.paidMonthlyPhase()},
	})
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, start))
	before := s.getSyncStateV2(view.Subscription.NamespacedID)
	legacyBefore, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{view.Subscription.NamespacedID})
	s.Require().NoError(err)
	s.Require().NotNil(before.NextReconciliationAt)

	// When the next sync unexpectedly receives no plan and no error from the reconciler.
	clock.FreezeTime(*before.NextReconciliationAt)
	defer clock.UnFreeze()
	brokenService := *s.Service
	brokenService.reconciler = nilPlanReconciler{Reconciler: s.Service.reconciler}
	err = brokenService.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now())
	s.Require().ErrorContains(err, "building subscription sync plan returned nil without an error")

	// Then it must not record success or replace either saved schedule with fallback values.
	s.Equal(before, s.getSyncStateV2(view.Subscription.NamespacedID))
	legacyAfter, err := s.Service.GetSyncStates(ctx, subscriptionsync.GetSyncStatesInput{view.Subscription.NamespacedID})
	s.Require().NoError(err)
	s.Equal(legacyBefore, legacyAfter)
	s.requireMonthlyItems(view, []timeutil.ClosedPeriod{{From: start, To: end}})
}

func (s *SyncStateTestSuite) TestNextBillingPeriodForMidPeriodCustomerDeletion() {
	// These stay at the target-builder boundary: legacy billing currently rejects
	// lines whose original period extends past a mid-period customer deletion.
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	end := now.AddDate(0, 1, 0)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	for _, tc := range []struct {
		name              string
		asOf              time.Time
		customerDeletedAt *time.Time
		generationLimit   time.Time
	}{
		{
			name:              "customer deletion ends the current period early",
			asOf:              now.Add(14 * 24 * time.Hour),
			customerDeletedAt: lo.ToPtr(now.Add(20 * 24 * time.Hour)),
			generationLimit:   now.Add(20 * 24 * time.Hour),
		},
		{
			name:              "customer deletion ends the following period early",
			asOf:              now.Add(14 * 24 * time.Hour),
			customerDeletedAt: lo.ToPtr(end.Add(14 * 24 * time.Hour)),
			generationLimit:   end,
		},
		{
			name:              "past customer deletion leaves no next period",
			asOf:              now.Add(14 * 24 * time.Hour),
			customerDeletedAt: lo.ToPtr(now.Add(10 * 24 * time.Hour)),
			generationLimit:   now.Add(10 * 24 * time.Hour),
		},
	} {
		s.Run(tc.name, func() {
			// Given a real subscription and its effective service end.
			ctx := s.T().Context()
			view := s.createSubscriptionForRevisionChange(now)

			currentView, err := s.SubscriptionService.GetView(ctx, view.Subscription.NamespacedID)
			s.Require().NoError(err)

			// When generation completes, scheduling follows that coverage rather than AsOf.
			builder := targetstate.NewBuilder(s.Service.logger, s.Service.tracer)
			target, err := builder.Build(ctx, targetstate.BuildInput{
				AsOf:              tc.asOf,
				SubscriptionView:  &currentView,
				CustomerDeletedAt: tc.customerDeletedAt,
				Persisted: persistedstate.State{
					ByUniqueID: map[string]persistedstate.Item{},
					Invoices:   persistedstate.Invoices{},
				},
			})
			s.Require().NoError(err)
			s.Equal(tc.generationLimit, target.MaxGenerationTimeLimit)
			s.Nil(target.NextBillingPeriod)
		})
	}
}
