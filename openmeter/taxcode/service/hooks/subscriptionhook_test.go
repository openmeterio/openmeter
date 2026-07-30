package hooks_test

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/taxcode/service/hooks"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

// provisionSubscriptionOrgDefaults provisions the organization-default tax codes that
// DeleteTaxCode requires to exist before it calls the pre-delete hook. This is the
// subscription test environment variant, taking a taxcode.Service instead of pctestutils.TestEnv.
func provisionSubscriptionOrgDefaults(t *testing.T, svc taxcode.Service, ns string) {
	t.Helper()

	invoicing, err := svc.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "default-invoicing",
		Name:      "Provider Default",
	})
	require.NoError(t, err)

	creditGrant, err := svc.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "default-credit-grant",
		Name:      "Non-Taxable",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_00000000"},
		},
	})
	require.NoError(t, err)

	_, err = svc.UpsertOrganizationDefaultTaxCodes(t.Context(), taxcode.UpsertOrganizationDefaultTaxCodesInput{
		Namespace:            ns,
		InvoicingTaxCodeID:   invoicing.ID,
		CreditGrantTaxCodeID: creditGrant.ID,
	})
	require.NoError(t, err)
}

func TestSubscriptionHookPreDelete(t *testing.T) {
	// Setup real services backed by Postgres.
	currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.SetTime(currentTime)
	defer clock.ResetTime()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	// Register the subscription hook on the real tax code service.
	subHook, err := hooks.NewSubscriptionHook(hooks.SubscriptionHookConfig{SubscriptionService: deps.SubscriptionService})
	require.NoError(t, err)
	deps.TaxCodeService.RegisterHooks(subHook)

	ns := subscriptiontestutils.ExampleNamespace

	// Provision organization-default tax codes so DeleteTaxCode can proceed past
	// the org-defaults check and reach the pre-delete hook.
	provisionSubscriptionOrgDefaults(t, deps.TaxCodeService, ns)

	t.Run("blocks deletion when a subscription references the tax code", func(t *testing.T) {
		// given: a tax code that a subscription will reference
		referenced, err := deps.TaxCodeService.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       "sub-referenced",
			Name:      "Referenced Tax Code",
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_20000005"},
			},
		})
		require.NoError(t, err)

		// given: a plan whose rate card references the tax code
		rc := subscriptiontestutils.ExampleRateCard1.Clone().(*productcatalog.UsageBasedRateCard)
		rc.TaxConfig = &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(referenced.ID)}

		planInput := subscriptiontestutils.BuildTestPlanInput(t).
			AddPhase(nil, rc).
			Build()

		// Create example features before plan creation (plan publish validates feature references)
		_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

		// CreatePlan both creates and publishes the plan
		pl := deps.PlanHelper.CreatePlan(t, planInput)

		// given: a running subscription that references the tax code via the rate card
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, pl, clock.Now())
		_ = subView

		// when: attempting to delete the referenced tax code
		err = deps.TaxCodeService.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: referenced.ID},
		})

		// then: an error is returned and it is a TaxCodeReferencedByRateCard error
		require.Error(t, err)
		require.True(t, taxcode.IsTaxCodeReferencedByRateCardError(err),
			"expected TaxCodeReferencedByRateCard error, got: %v", err)
	})

	t.Run("allows deletion when no subscription references the tax code", func(t *testing.T) {
		// given: a tax code that no subscription references
		unreferenced, err := deps.TaxCodeService.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
			Namespace: ns,
			Key:       "sub-unreferenced",
			Name:      "Unreferenced Tax Code",
			AppMappings: taxcode.TaxCodeAppMappings{
				{AppType: app.AppTypeStripe, TaxCode: "txcd_20000006"},
			},
		})
		require.NoError(t, err)

		// when: deleting the unreferenced tax code
		err = deps.TaxCodeService.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: models.NamespacedID{Namespace: ns, ID: unreferenced.ID},
		})

		// then: no error is returned
		require.NoError(t, err)
	})
}

func TestSubscriptionHookPreDeleteScheduledSubscription(t *testing.T) {
	// A subscription scheduled to start in the future still references the tax code and
	// must block deletion with the typed error. This guards the "all statuses" requirement:
	// the hook lists every status, and the subscription view loads phases/items structurally
	// (only gated on soft-delete, not on being active as-of-now), so a not-yet-active
	// subscription still surfaces its rate cards and yields the typed referenced-by error.
	currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.SetTime(currentTime)
	defer clock.ResetTime()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	// Register the subscription hook on the real tax code service.
	subHook, err := hooks.NewSubscriptionHook(hooks.SubscriptionHookConfig{SubscriptionService: deps.SubscriptionService})
	require.NoError(t, err)
	deps.TaxCodeService.RegisterHooks(subHook)

	ns := subscriptiontestutils.ExampleNamespace

	// Provision organization-default tax codes so DeleteTaxCode reaches the pre-delete hook.
	provisionSubscriptionOrgDefaults(t, deps.TaxCodeService, ns)

	// given: a tax code referenced by a plan rate card
	referenced, err := deps.TaxCodeService.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "sub-scheduled-referenced",
		Name:      "Referenced Tax Code",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_20000007"},
		},
	})
	require.NoError(t, err)

	rc := subscriptiontestutils.ExampleRateCard1.Clone().(*productcatalog.UsageBasedRateCard)
	rc.TaxConfig = &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(referenced.ID)}

	planInput := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(nil, rc).
		Build()

	// Example features must exist before plan publish (plan publish validates feature references).
	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	// CreatePlan both creates and publishes the plan.
	pl := deps.PlanHelper.CreatePlan(t, planInput)

	// given: a subscription SCHEDULED to start one year in the future (not active as-of-now)
	futureStart := clock.Now().AddDate(1, 0, 0)
	_ = subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, pl, futureStart)

	// when: attempting to delete the referenced tax code
	err = deps.TaxCodeService.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: referenced.ID},
	})

	// then: deletion is blocked with the TYPED referenced-by-rate-card error (not a generic error)
	require.Error(t, err)
	require.True(t, taxcode.IsTaxCodeReferencedByRateCardError(err),
		"expected TaxCodeReferencedByRateCard error for a scheduled subscription, got: %v", err)
}

func TestSubscriptionHookPreDeleteSoftDeletedItem(t *testing.T) {
	// given:
	// - a running subscription with a FUTURE (not-yet-started) phase whose only referencing
	//   rate-card item is removed via EditRunning, then the clock advances well past the edit
	// - removing an item from a future phase (as opposed to the current phase, which can only
	//   ever be "closed" going forward, never truly deleted, since history cannot be falsified)
	//   genuinely soft-deletes its row with no replacement version, so its deleted_at timestamp
	//   ends up in the past once the clock advances - the exact shape the DB soft-delete guard
	//   in subscriptionrepo.go's List targets. The raw DB row still carries the tax code
	//   reference, so a filter without the "deleted_at IS NULL OR deleted_at > now" guard would
	//   still match this subscription even though the item is long gone.
	// when: the tax code is deleted
	// then: deletion must succeed, because the subscription view (which is what actually
	//   determines whether a rate card references the tax code) never surfaces an item that was
	//   soft-deleted in the past, so no view item references the tax code anymore. Without the
	//   guard in subscriptionrepo.go's List, the hook would still match this subscription via
	//   the DB filter, find no referencing view item, and return the generic (not the typed)
	//   "matched tax code filter but no rate card references" error instead of nil.
	currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.SetTime(currentTime)
	defer clock.ResetTime()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	// Register the subscription hook on the real tax code service.
	subHook, err := hooks.NewSubscriptionHook(hooks.SubscriptionHookConfig{SubscriptionService: deps.SubscriptionService})
	require.NoError(t, err)
	deps.TaxCodeService.RegisterHooks(subHook)

	ns := subscriptiontestutils.ExampleNamespace

	// Provision organization-default tax codes so DeleteTaxCode reaches the pre-delete hook.
	provisionSubscriptionOrgDefaults(t, deps.TaxCodeService, ns)

	// given: a tax code referenced by a plan rate card in the subscription's second (future) phase
	referenced, err := deps.TaxCodeService.CreateTaxCode(t.Context(), taxcode.CreateTaxCodeInput{
		Namespace: ns,
		Key:       "sub-soft-deleted-referenced",
		Name:      "Referenced Tax Code",
		AppMappings: taxcode.TaxCodeAppMappings{
			{AppType: app.AppTypeStripe, TaxCode: "txcd_20000008"},
		},
	})
	require.NoError(t, err)

	// given: a two-phase plan. The first phase (active immediately) has a single, unrelated
	// item so the subscription always has a current phase to compare against. The second
	// phase (starting in a month, i.e. still in the future at edit time) has two rate cards:
	// one referencing the tax code, and one that does not, so the phase keeps at least one
	// item once the referencing item is removed (a phase must keep at least one item).
	rc := subscriptiontestutils.ExampleRateCard1.Clone().(*productcatalog.UsageBasedRateCard)
	rc.TaxConfig = &productcatalog.TaxConfig{TaxCodeID: lo.ToPtr(referenced.ID)}
	rcPhase1 := subscriptiontestutils.ExampleRateCard2.Clone()
	rcPhase2Other := subscriptiontestutils.ExampleRateCard2.Clone()

	planInput := subscriptiontestutils.BuildTestPlanInput(t).
		AddPhase(lo.ToPtr(datetime.MustParseDuration(t, "P1M")), rcPhase1).
		AddPhase(nil, rc, rcPhase2Other).
		Build()

	// Example features must exist before plan publish (plan publish validates feature references).
	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	// CreatePlan both creates and publishes the plan.
	pl := deps.PlanHelper.CreatePlan(t, planInput)

	// given: a running subscription that references the tax code via the rc rate card in its
	// second, not-yet-started phase
	subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, pl, clock.Now())

	futurePhaseKey := subView.Phases[1].Spec.PhaseKey

	// given: the referencing item is removed from the future phase. Since it's not the current
	// phase, EditRunning genuinely deletes the item's history (no replacement version is kept,
	// see patch.PatchRemoveItem.ApplyTo's future-phase branch), leaving only a soft-deleted row.
	editTime := clock.Now()
	_, err = deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{PhaseKey: futurePhaseKey, ItemKey: rc.Key()},
	}, subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)})
	require.NoError(t, err)

	// given: time advances well past the edit, so the item's soft-delete is now in the past
	// and the subscription view will no longer surface it.
	clock.SetTime(editTime.Add(time.Hour))

	// when: attempting to delete the tax code whose only referencing item was soft-deleted in the past
	err = deps.TaxCodeService.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: models.NamespacedID{Namespace: ns, ID: referenced.ID},
	})

	// then: deletion is allowed, since no currently-surfaced rate card references the tax code
	require.NoError(t, err, "deleting a tax code whose only referencing subscription item was soft-deleted in the past must be allowed")
}
