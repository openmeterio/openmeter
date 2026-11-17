package service

import (
	"fmt"
	"math"
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
		name          string
		billingEndsAt *time.Time
		fraction      float64
		want          time.Time
	}{
		{
			name: "ongoing subscription uses eighty percent of the whole period",
			want: start.Add(8 * 24 * time.Hour),
		},
		{
			name:     "ongoing subscription uses ninety-five percent of the whole period",
			fraction: 1,
			want:     start.Add(228 * time.Hour),
		},
		{
			name:          "billing end after the period still uses jitter",
			billingEndsAt: lo.ToPtr(end.Add(24 * time.Hour)),
			want:          start.Add(8 * 24 * time.Hour),
		},
		{
			name:          "period ending with billing schedules the end exactly",
			billingEndsAt: &end,
			want:          end,
		},
		{
			name:          "customer deletion inside the period schedules the deletion exactly",
			billingEndsAt: lo.ToPtr(start.Add(5 * 24 * time.Hour)),
			want:          start.Add(5 * 24 * time.Hour),
		},
		{
			name:          "billing end before the period stays due",
			billingEndsAt: lo.ToPtr(start.Add(-time.Hour)),
			want:          start.Add(-time.Hour),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := nextReconciliationAt(nextReconciliationAtInput{
				BillingPeriod:  period,
				BillingEndsAt:  tc.billingEndsAt,
				RandomFraction: tc.fraction,
			})
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}

	for _, tc := range []struct {
		name  string
		input nextReconciliationAtInput
	}{
		{
			name:  "missing billing period",
			input: nextReconciliationAtInput{},
		},
		{
			name:  "inverted billing period",
			input: nextReconciliationAtInput{BillingPeriod: timeutil.ClosedPeriod{From: end, To: start}},
		},
		{
			name:  "zero billing end",
			input: nextReconciliationAtInput{BillingPeriod: period, BillingEndsAt: &time.Time{}},
		},
		{
			name:  "random fraction above one",
			input: nextReconciliationAtInput{BillingPeriod: period, RandomFraction: 1.5},
		},
		{
			name:  "random fraction not a number",
			input: nextReconciliationAtInput{BillingPeriod: period, RandomFraction: math.NaN()},
		},
	} {
		t.Run("rejects "+tc.name, func(t *testing.T) {
			_, err := nextReconciliationAt(tc.input)
			require.True(t, models.IsGenericValidationError(err), "unexpected error: %v", err)
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

// forEachBackend runs the scenario against legacy billing and charges. Scenarios
// call useNewCustomer so the two runs do not share a customer.
func (s *SyncStateTestSuite) forEachBackend(scenario func()) {
	for _, backend := range []struct {
		name    string
		service *Service
	}{
		{name: "legacy billing", service: s.LegacySyncService},
		{name: "charges", service: s.ChargeSyncService},
	} {
		s.Run(backend.name, func() {
			s.Service = backend.service
			scenario()
		})
	}
}

// useNewCustomer gives a scenario its own customer, so its subscription cannot
// overlap one created by another case.
func (s *SyncStateTestSuite) useNewCustomer() {
	s.T().Helper()
	s.Customer = s.CreateTestCustomer(s.Namespace, ulid.Make().String())
	_, err := s.LedgerResolver.CreateCustomerAccounts(s.T().Context(), s.Customer.GetID())
	s.Require().NoError(err)
}

func (s *SyncStateTestSuite) createSyncStateSubscription(start time.Time, phases ...productcatalog.Phase) subscription.SubscriptionView {
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
			Phases: phases,
		},
	}, start)
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

// requireUnscheduled checks that the sync that just ran recorded the given state
// without a next reconciliation deadline. Only active results are scheduled.
func (s *SyncStateTestSuite) requireUnscheduled(subscriptionID models.NamespacedID, want subscriptionsync.SyncStateV2State) {
	s.T().Helper()
	state := s.getSyncStateV2(subscriptionID)
	s.Equal(want, state.LastSuccessState)
	s.Equal(lo.ToPtr(clock.Now().UTC()), state.LastSuccessAt)
	s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
	s.Nil(state.NextReconciliationAt)
}

// requireFees checks every provisioned in-advance fee by its sync identity, so
// duplicate or missing items fail even if the reconciliation deadline is correct.
func (s *SyncStateTestSuite) requireFees(view subscription.SubscriptionView, expected map[string]timeutil.ClosedPeriod) {
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
			s.Equal(intent.ServicePeriod.From, intent.InvoiceAt, "in-advance fee is invoiceable at its service period start")
			actual[*reference] = intent.ServicePeriod
		}
	} else {
		invoice := s.gatheringInvoice(s.T().Context(), s.Namespace, s.Customer.ID)
		s.Require().Len(invoice.Lines.OrEmpty(), len(expected))
		for _, line := range invoice.Lines.OrEmpty() {
			s.Require().NotNil(line.ChildUniqueReferenceID)
			s.Equal(line.GetServicePeriod().From, line.InvoiceAt, "in-advance fee is invoiceable at its service period start")
			actual[*line.ChildUniqueReferenceID] = line.GetServicePeriod()
		}
	}

	s.Equal(expected, actual)
}

// requireMonthlyItems expects exactly one monthly fee of the paid phase per period.
func (s *SyncStateTestSuite) requireMonthlyItems(view subscription.SubscriptionView, periods []timeutil.ClosedPeriod) {
	s.T().Helper()
	ids := (recurringLineMatcher{PhaseKey: "paid", ItemKey: "monthly-fee", PeriodMax: len(periods) - 1}).ChildIDs(view.Subscription.ID)
	expected := make(map[string]timeutil.ClosedPeriod, len(periods))
	for index, period := range periods {
		expected[ids[index]] = period
	}

	s.requireFees(view, expected)
}

func (s *SyncStateTestSuite) TestMonthlyProgression() {
	s.forEachBackend(func() {
		// Given a monthly in-advance fee, beginning November 1.
		start := s.mustParseTime("2026-11-01T00:00:00Z")
		clock.FreezeTime(start)
		defer clock.UnFreeze()
		s.useNewCustomer()
		view := s.createSyncStateSubscription(start, s.paidMonthlyPhase())

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
	})
}

func (s *SyncStateTestSuite) TestFutureStartProgression() {
	// Given a subscription starting December 1, created in November.
	now := s.mustParseTime("2026-11-15T00:00:00Z")
	start := s.mustParseTime("2026-12-01T00:00:00Z")
	january := s.mustParseTime("2027-01-01T00:00:00Z")
	february := s.mustParseTime("2027-02-01T00:00:00Z")
	clock.FreezeTime(now)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(start, s.paidMonthlyPhase())

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
	view := s.createSyncStateSubscription(start, s.freeMonthlyPhase("P1M"), s.paidMonthlyPhase())

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
	s.forEachBackend(func() {
		for _, tc := range []struct {
			name string
			asOf time.Time
		}{
			{name: "before the phase boundary", asOf: s.mustParseTime("2026-11-25T00:00:00Z")},
			{name: "at the phase boundary", asOf: s.mustParseTime("2026-12-01T00:00:00Z")},
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
				s.useNewCustomer()

				firstPhase := s.paidMonthlyPhase()
				firstPhase.PhaseMeta = s.phaseMeta("first-paid", "P1M")
				view := s.createSyncStateSubscription(start, firstPhase, s.paidMonthlyPhase())
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
				s.requireFees(view, expectedFees)
				next := s.requireActiveDeadline(view.Subscription.NamespacedID, timeutil.ClosedPeriod{From: december, To: january})

				// Then following the December deadline adds January's fee without duplicating earlier fees.
				clock.FreezeTime(next)
				defer clock.UnFreeze()
				s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now()))
				expectedFees[secondPhaseFeeIDs[1]] = timeutil.ClosedPeriod{From: january, To: february}
				s.requireFees(view, expectedFees)
				s.requireActiveDeadline(view.Subscription.NamespacedID, timeutil.ClosedPeriod{From: january, To: february})
			})
		}
	})
}

func (s *SyncStateTestSuite) TestNoBillablesHasNoReconciliationDeadline() {
	// Given a subscription with a monthly billing calendar but no priced items.
	start := s.mustParseTime("2026-11-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(start, s.freeMonthlyPhase(""))

	// When sync completes, it acknowledges the subscription without scheduling more work.
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	s.requireUnscheduled(view.Subscription.NamespacedID, subscriptionsync.SyncStateV2StateNoBillables)

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
	view := s.createSyncStateSubscription(start, s.paidMonthlyPhase())
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
	s.requireUnscheduled(view.Subscription.NamespacedID, subscriptionsync.SyncStateV2StateNoBillables)
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
	view := s.createSyncStateSubscription(start, productcatalog.Phase{
		PhaseMeta: s.phaseMeta("one-time", ""),
		RateCards: productcatalog.RateCards{&productcatalog.FlatFeeRateCard{
			RateCardMeta: productcatalog.RateCardMeta{
				Key:   "completion-fee",
				Name:  "Completion fee",
				Price: price,
			},
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
	s.requireUnscheduled(view.Subscription.NamespacedID, subscriptionsync.SyncStateV2StateEnded)

	// Then a retried cancellation sync keeps the single fee line and stays ended.
	s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, clock.Now()))
	finalInvoice := s.gatheringInvoice(ctx, s.Namespace, s.Customer.ID)
	s.expectLines(finalInvoice, view.Subscription.ID, expectedLines)
	s.Equal(invoice.Lines.OrEmpty()[0].ID, finalInvoice.Lines.OrEmpty()[0].ID)
	s.requireUnscheduled(view.Subscription.NamespacedID, subscriptionsync.SyncStateV2StateEnded)
}

func (s *SyncStateTestSuite) TestCancellationProgression() {
	// Given a subscription ending February 1, with one final reconciliation required.
	start := s.mustParseTime("2026-11-01T00:00:00Z")
	december := s.mustParseTime("2026-12-01T00:00:00Z")
	january := s.mustParseTime("2027-01-01T00:00:00Z")
	end := s.mustParseTime("2027-02-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(start, s.paidMonthlyPhase())
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
	clock.FreezeTime(end)
	defer clock.UnFreeze()
	s.Require().NoError(s.Service.SyncByID(s.T().Context(), view.Subscription.NamespacedID, clock.Now()))
	s.requireMonthlyItems(view, periods)
	s.requireUnscheduled(view.Subscription.NamespacedID, subscriptionsync.SyncStateV2StateEnded)
}

func (s *SyncStateTestSuite) TestRepeatedSyncAndDryRun() {
	// Given a successfully synced subscription with its first fee provisioned.
	start := s.mustParseTime("2026-11-01T00:00:00Z")
	end := s.mustParseTime("2026-12-01T00:00:00Z")
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	view := s.createSyncStateSubscription(start, s.paidMonthlyPhase())
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

// TestCustomerDeletionReconciliation covers what is specific to customer deletion:
// it can end billing before ActiveTo. How the worker clock and AsOf decide between
// active and ended is covered for ActiveTo by TestEndedSubscriptionSyncResult.
func (s *SyncStateTestSuite) TestCustomerDeletionReconciliation() {
	createdAt := s.mustParseTime("2026-11-01T00:00:00Z")
	november15 := s.mustParseTime("2026-11-15T00:00:00Z")
	december := s.mustParseTime("2026-12-01T00:00:00Z")
	december15 := s.mustParseTime("2026-12-15T00:00:00Z")
	january := s.mustParseTime("2027-01-01T00:00:00Z")

	s.forEachBackend(func() {
		for _, tc := range []struct {
			name                 string
			deletedAt            time.Time
			subscriptionActiveTo *time.Time
			now                  time.Time
			withoutBillables     bool
			wantState            subscriptionsync.SyncStateV2State
			wantDeadline         *time.Time
		}{
			{
				name:         "future deletion schedules the final sync",
				deletedAt:    december,
				now:          november15,
				wantState:    subscriptionsync.SyncStateV2StateActive,
				wantDeadline: &december,
			},
			{
				name:      "sync after deletion clears the final deadline",
				deletedAt: december,
				now:       december15,
				wantState: subscriptionsync.SyncStateV2StateEnded,
			},
			{
				name:                 "subscription end before deletion wins",
				deletedAt:            january,
				subscriptionActiveTo: &december,
				now:                  december15,
				wantState:            subscriptionsync.SyncStateV2StateEnded,
			},
			{
				name:                 "deletion before subscription end wins",
				deletedAt:            december,
				subscriptionActiveTo: &january,
				now:                  december15,
				wantState:            subscriptionsync.SyncStateV2StateEnded,
			},
			{
				name:             "empty plan still records completion",
				deletedAt:        december,
				now:              december15,
				withoutBillables: true,
				wantState:        subscriptionsync.SyncStateV2StateEnded,
			},
		} {
			s.Run(tc.name, func() {
				// Given a subscription whose customer deletion may end billing before ActiveTo.
				ctx := s.T().Context()
				clock.FreezeTime(createdAt)
				defer clock.UnFreeze()
				s.useNewCustomer()

				phase := s.paidMonthlyPhase()
				if tc.withoutBillables {
					phase = s.freeMonthlyPhase("")
				}

				view := s.createSyncStateSubscription(createdAt, phase)
				if tc.subscriptionActiveTo != nil {
					_, err := s.SubscriptionService.Cancel(ctx, view.Subscription.NamespacedID, subscription.Timing{Custom: tc.subscriptionActiveTo})
					s.Require().NoError(err)
				}

				// Deleted customers no longer resolve subject keys; retain an identifiable invoice snapshot.
				s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).
					SetKey(s.Customer.ID).
					SetDeletedAt(tc.deletedAt).Exec(ctx))

				// First record the future deadline, so completion must clear an existing active result.
				s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, createdAt))
				if !tc.withoutBillables {
					initial := s.getSyncStateV2(view.Subscription.NamespacedID)
					s.Equal(subscriptionsync.SyncStateV2StateActive, initial.LastSuccessState)
					s.Equal(&december, initial.NextReconciliationAt)
				}

				// When sync completes, its result and deadline must use the same end as billing generation.
				clock.FreezeTime(tc.now)
				defer clock.UnFreeze()
				s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, tc.now))
				state := s.getSyncStateV2(view.Subscription.NamespacedID)
				s.Equal(tc.wantState, state.LastSuccessState)
				s.Equal(tc.wantDeadline, state.NextReconciliationAt)
				s.Equal(&tc.now, state.LastSuccessAt)
				s.Equal(state.CurrentSubsRevision, state.LastSyncedSubsRevision)
				if tc.withoutBillables {
					s.expectNoGatheringInvoice(ctx, s.Namespace, s.Customer.ID)
				} else {
					s.requireMonthlyItems(view, []timeutil.ClosedPeriod{{From: createdAt, To: december}})
				}

				// Then retrying a completed reconciliation stays ended and does not add billing artifacts.
				if tc.wantState == subscriptionsync.SyncStateV2StateEnded {
					clock.FreezeTime(tc.now.Add(time.Minute))
					defer clock.UnFreeze()
					s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, tc.now))
					s.requireUnscheduled(view.Subscription.NamespacedID, subscriptionsync.SyncStateV2StateEnded)
					if !tc.withoutBillables {
						s.requireMonthlyItems(view, []timeutil.ClosedPeriod{{From: createdAt, To: december}})
					}
				}
			})
		}
	})
}

func (s *SyncStateTestSuite) TestMidPeriodCustomerDeletionTruncatesServicePeriod() {
	s.forEachBackend(func() {
		// Given an open-ended subscription with its November monthly fee already provisioned.
		ctx := s.T().Context()
		start := s.mustParseTime("2026-11-01T00:00:00Z")
		deletedAt := s.mustParseTime("2026-11-15T00:00:00Z")
		syncAt := s.mustParseTime("2026-11-20T00:00:00Z")
		periodEnd := s.mustParseTime("2026-12-01T00:00:00Z")
		clock.FreezeTime(start)
		defer clock.UnFreeze()
		s.useNewCustomer()

		view := s.createSyncStateSubscription(start, s.paidMonthlyPhase())
		s.Require().Nil(view.Spec.ActiveTo)
		s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, start))
		s.requireMonthlyItems(view, []timeutil.ClosedPeriod{{From: start, To: periodEnd}})

		// Simulate an existing customer deletion that did not cancel the subscription.
		// Retain the customer key because deleted customers no longer resolve subject keys.
		s.Require().NoError(s.DBClient.Customer.UpdateOneID(s.Customer.ID).
			SetKey(s.Customer.ID).
			SetDeletedAt(deletedAt).Exec(ctx))

		// When sync reconciles after the mid-period deletion, it must shorten the existing fee.
		clock.FreezeTime(syncAt)
		defer clock.UnFreeze()
		s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, syncAt))

		// Then no service after November 15 remains billable, and final reconciliation is complete.
		s.requireMonthlyItems(view, []timeutil.ClosedPeriod{{From: start, To: deletedAt}})
		s.requireUnscheduled(view.Subscription.NamespacedID, subscriptionsync.SyncStateV2StateEnded)

		// Repeating cleanup must preserve the same subscription-sync identities and periods.
		s.Require().NoError(s.Service.SyncByID(ctx, view.Subscription.NamespacedID, syncAt))
		s.requireMonthlyItems(view, []timeutil.ClosedPeriod{{From: start, To: deletedAt}})
		s.requireUnscheduled(view.Subscription.NamespacedID, subscriptionsync.SyncStateV2StateEnded)
	})
}

func (s *SyncStateTestSuite) TestActiveSyncResultRollback() {
	ctx := s.T().Context()
	now := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	// Given a successfully synced subscription whose next success write will fail in PostgreSQL.
	view := s.createSyncStateSubscription(now, s.paidMonthlyPhase())
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
