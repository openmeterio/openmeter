package subscription_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

func TestMigrationStartingCurrentPhaseStillReplacesSubscription(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given identical phase timelines and an unchanged billing anchor
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, fixture.newPlanVersion())
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when the caller explicitly selects the phase that is already current
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		StartingPhase: lo.ToPtr("test_phase_1"),
		BillingAnchor: &start,
	})

	// then the explicit reset creates a replacement despite the matching timeline
	fixture.assertSubscriptionReplaced(after, target, clock.Now())
	require.Equal(t, start, after.Subscription.BillingAnchor)
	assertMigrationStartsInPhase(t, after, "test_phase_1")
}

func TestMigrationStartingPhaseReplacesIncompatibleTimeline(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a single-phase subscription and a target with two phases
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, migrationPlanWithTwoPhases(t))
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when the caller selects the target's second phase
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		StartingPhase: lo.ToPtr("test_phase_2"),
	})

	// then replacement starts directly in that phase and keeps the existing anchor
	fixture.assertSubscriptionReplaced(after, target, clock.Now())
	require.Equal(t, start, after.Subscription.BillingAnchor)
	assertMigrationStartsInPhase(t, after, "test_phase_2")
}

func TestMigrationStartingPhaseAtNextCycle(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given an incompatible target timeline
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, migrationPlanWithTwoPhases(t))
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when replacement is explicitly scheduled for the next cycle
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		StartingPhase: lo.ToPtr("test_phase_2"),
		Timing:        &subscription.Timing{Enum: lo.ToPtr(subscription.TimingNextBillingCycle)},
	})

	// then the old subscription ends exactly when the selected target phase starts
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	fixture.assertSubscriptionReplaced(after, target, at)
	assertMigrationStartsInPhase(t, after, "test_phase_2")
}

func TestMigrationStartingPhaseWithAnchorAndCustomTime(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given an incompatible target timeline and an explicit new anchor
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, migrationPlanWithTwoPhases(t))
	clock.FreezeTime(clock.Now().Add(time.Second))
	anchor := start.Add(3 * 24 * time.Hour)

	// when both overrides accompany a billing-aligned custom time
	at := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		StartingPhase: lo.ToPtr("test_phase_2"),
		BillingAnchor: &anchor,
		Timing:        &subscription.Timing{Custom: &at},
	})

	// then replacement honors the selected phase, time, and anchor together
	fixture.assertSubscriptionReplaced(after, target, at)
	require.Equal(t, anchor, after.Subscription.BillingAnchor)
	assertMigrationStartsInPhase(t, after, "test_phase_2")
}

func TestMigrationRejectsChangedPhaseCountWithoutReset(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a target with an additional phase
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	fixture.deps.PlanHelper.CreatePlan(t, migrationPlanWithTwoPhases(t))
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when no replacement is requested
	assertMigrationRequiresExplicitReset(t, fixture, "same phase timeline")
}

func TestMigrationRejectsChangedPhaseKeyWithoutReset(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a target with a renamed phase
	fixture := newMigrationFixture(t)
	next := fixture.newPlanVersion()
	next.Phases[0].Key = "renamed"
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	fixture.deps.PlanHelper.CreatePlan(t, next)
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when no replacement is requested
	assertMigrationRequiresExplicitReset(t, fixture, "in both specs")
}

func TestMigrationRejectsChangedPhaseStartWithoutReset(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a target whose second phase starts a month later
	fixture := newMigrationFixtureWithPlan(t, migrationPlanWithTwoPhases(t))
	next := migrationPlanWithTwoPhases(t)
	next.Phases[0].Duration = lo.ToPtr(datetime.MustParseDuration(t, "P2M"))
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	fixture.deps.PlanHelper.CreatePlan(t, next)
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when no replacement is requested
	assertMigrationRequiresExplicitReset(t, fixture, "same start")
}

func TestMigrationInvalidStartingPhaseLeavesSubscriptionIntact(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()

	// given a later plan version with no phase named missing
	fixture := newMigrationFixture(t)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	fixture.deps.PlanHelper.CreatePlan(t, fixture.newPlanVersion())
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when replacement requests the nonexistent phase
	_, err := fixture.deps.pcSubscriptionService.Migrate(t.Context(), plansubscription.MigrateSubscriptionRequest{
		ID:            fixture.before.Subscription.NamespacedID,
		StartingPhase: lo.ToPtr("missing"),
	})

	// then phase validation fails before cancellation
	require.ErrorContains(t, err, "starting phase missing not found")
	assertMigrationViewUnchanged(t, fixture)
}

func assertMigrationRequiresExplicitReset(t *testing.T, fixture migrationFixture, reason string) {
	t.Helper()

	_, err := fixture.deps.pcSubscriptionService.Migrate(t.Context(), plansubscription.MigrateSubscriptionRequest{
		ID: fixture.before.Subscription.NamespacedID,
	})

	// then the error describes the incompatibility and the explicit replacement option
	require.ErrorContains(t, err, reason)
	require.ErrorContains(t, err, "startingPhase")
	require.ErrorContains(t, err, "billing adjustments")
	require.ErrorContains(t, err, "does not transfer addons")
	assertMigrationViewUnchanged(t, fixture)
}

func assertMigrationStartsInPhase(t *testing.T, after subscription.SubscriptionView, key string) {
	t.Helper()

	phase, ok := after.Spec.GetCurrentPhaseAt(after.Subscription.ActiveFrom)
	require.True(t, ok)
	require.Equal(t, key, phase.PhaseKey)

	cadence, err := after.Spec.GetPhaseCadence(key)
	require.NoError(t, err)
	require.Equal(t, after.Subscription.ActiveFrom, cadence.ActiveFrom)
}

func TestMigrationStartingPhaseDoesNotTransferAddons(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start.Add(time.Millisecond))
	defer clock.UnFreeze()

	// given a running subscription with an addon compatible only with its old plan
	deps := setup(t, setupConfig{})
	defer deps.cleanup(t)

	input := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil,
		subscriptiontestutils.ExampleRateCard1.Clone(), subscriptiontestutils.ExampleRateCard2.Clone()).Build()
	original, addon := subscriptiontestutils.CreatePlanWithAddon(t, deps.SubscriptionDependencies, input,
		subscriptiontestutils.BuildAddonForTesting(t, productcatalog.EffectivePeriod{EffectiveFrom: &start},
			productcatalog.AddonInstanceTypeSingle, subscriptiontestutils.ExampleAddonRateCard1.Clone()))
	before := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.SubscriptionDependencies, original, start)
	fixture := migrationFixture{t: t, deps: deps, before: before}

	var err error

	clock.FreezeTime(start.Add(24 * time.Hour))
	fixture.before, _, err = fixture.deps.WorkflowService.AddAddon(t.Context(), fixture.before.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
		AddonID: addon.ID, InitialQuantity: 1,
		Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
	})
	require.NoError(t, err)

	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	target := fixture.deps.PlanHelper.CreatePlan(t, fixture.newPlanVersion())
	clock.FreezeTime(clock.Now().Add(time.Second))

	// when the caller requests replacement instead of an in-place addon migration
	after := fixture.migrate(plansubscription.MigrateSubscriptionRequest{
		StartingPhase: lo.ToPtr("test_phase_1"),
	})

	// then the old subscription ends and the replacement has no addon purchases
	fixture.assertSubscriptionReplaced(after, target, clock.Now())
	addons, err := fixture.deps.SubscriptionAddonService.List(t.Context(), after.Subscription.Namespace, subscriptionaddon.ListSubscriptionAddonsInput{
		SubscriptionID: after.Subscription.ID,
	})
	require.NoError(t, err)
	require.Empty(t, addons.Items)
}
