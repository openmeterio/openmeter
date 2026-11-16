package service_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
)

// The add-item patch of a running-subscription edit authors its Tax Code reference, unless the
// ID equals the one of the item version it replaces (the last one stored for the key): that is
// a price change echoing the rate card back, which inherits the reference. Deleted codes are
// rejected with rate_card_tax_code_deleted, missing ones with rate_card_tax_code_not_found.
func TestEditRunningAddItemTaxCodeReferences(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
	clock.SetTime(now)
	defer clock.ResetTime()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	// Deleting a Tax Code checks it isn't an organization default, so the defaults must exist.
	deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

	liveTaxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())

	// given: a running subscription whose items reference a Tax Code that gets deleted
	// afterwards, so the subscription starts out holding a deleted reference
	doomedPlanTaxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	planInput := subscriptiontestutils.GetExamplePlanInput(t, subscriptiontestutils.ExampleTaxConfig(doomedPlanTaxCode))

	plan := deps.PlanHelper.CreatePlan(t, planInput)

	require.NoError(t, deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: doomedPlanTaxCode.NamespacedID,
	}))

	cust := deps.CustomerAdapter.CreateExampleCustomer(t)

	subView, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: &now},
			Name:   "subscription for tax code edit tests",
		},
		CustomerID: cust.ID,
		Namespace:  subscriptiontestutils.ExampleNamespace,
	}, plan)
	require.NoError(t, err)

	// given: the item is switched to a live tax code, so the version active after this edit no
	// longer holds the deleted one
	subView, err = deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
		newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: liveTaxCode.ID}),
	}, immediate)
	require.NoError(t, err)

	t.Run("re-adding the deleted tax code a previous version held is rejected", func(t *testing.T) {
		// The exception matches only the version the patch replaces. The previous version's
		// deleted code was already swapped away, so this edit authors the deleted reference.
		// when: the item is re-added referencing the deleted tax code
		_, err := deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: doomedPlanTaxCode.ID}),
		}, immediate)

		// then: it is rejected with the deleted error
		pctestutils.RequireTaxCodeDeleted(t, err)
		require.ErrorContains(t, err, "patch at idx 0")
		require.ErrorContains(t, err, subscriptiontestutils.ExampleFeatureKey)
	})

	t.Run("add-item with a deleted tax code new to the item is rejected", func(t *testing.T) {
		// given: a phase-1 rate card key the plan doesn't hold
		itemKey := "new-item-deleted-tax-code"

		// when: the item is added referencing the deleted tax code
		_, err := deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			newTaxedAddItemPatch("test_phase_1", itemKey, &productcatalog.TaxCodeConfig{TaxCodeID: doomedPlanTaxCode.ID}),
		}, immediate)

		// then: the authored reference is rejected with the deleted error
		pctestutils.RequireTaxCodeDeleted(t, err)
		require.ErrorContains(t, err, "patch at idx 0")
		require.ErrorContains(t, err, itemKey)
	})

	t.Run("re-adding an item with the same tax code as the replaced version passes", func(t *testing.T) {
		// when: a price change echoes the rate card back with the same live tax code the item
		// version being replaced holds
		sub, err := deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: liveTaxCode.ID}),
		}, immediate)

		// then: the inherited reference passes and the new price is stored
		require.NoError(t, err)

		phase, ok := sub.GetPhaseByKey("test_phase_1")
		require.True(t, ok)
		items := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
		require.NotEmpty(t, items)

		lastItem := items[len(items)-1]
		cfg := lastItem.Spec.RateCard.AsMeta().TaxConfig
		require.NotNil(t, cfg)
		require.Equal(t, liveTaxCode.ID, cfg.TaxCodeID)
		require.NotNil(t, lastItem.Spec.RateCard.AsMeta().Price)
	})

	t.Run("switching an item to a live tax code passes", func(t *testing.T) {
		// when: the item is re-added referencing a second live tax code
		secondLive := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())

		sub, err := deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: secondLive.ID}),
		}, immediate)

		// then: the authored live reference passes
		require.NoError(t, err)

		phase, ok := sub.GetPhaseByKey("test_phase_1")
		require.True(t, ok)
		items := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
		require.NotEmpty(t, items)

		cfg := items[len(items)-1].Spec.RateCard.AsMeta().TaxConfig
		require.NotNil(t, cfg)
		require.Equal(t, secondLive.ID, cfg.TaxCodeID)
	})

	t.Run("add-item with a missing tax code is rejected", func(t *testing.T) {
		// given: a phase-1 rate card key the plan doesn't hold
		itemKey := "new-item-missing-tax-code"

		// when: the item is added referencing an unknown tax code
		_, err := deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			newTaxedAddItemPatch("test_phase_1", itemKey, &productcatalog.TaxCodeConfig{TaxCodeID: "01JABSENTTAXC0DE0000000000"}),
		}, immediate)

		// then: it is rejected with the not-found error
		pctestutils.RequireTaxCodeNotFound(t, err)
		require.ErrorContains(t, err, "patch at idx 0")
		require.ErrorContains(t, err, itemKey)
	})

	t.Run("deleted tax code inherited through an unrelated item is still rejected", func(t *testing.T) {
		// The exception matches the ID against the item being replaced at the same phase and
		// item key. A different item inheriting the same deleted code is a new authored
		// reference and must not slip through.
		otherTaxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
		require.NoError(t, deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: otherTaxCode.NamespacedID,
		}))

		// given: a phase-1 rate card key the plan doesn't hold
		itemKey := "item-inheriting-other-deleted-code"

		// when: the item is added referencing the deleted tax code another item inherited
		_, err := deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			newTaxedAddItemPatch("test_phase_1", itemKey, &productcatalog.TaxCodeConfig{TaxCodeID: otherTaxCode.ID}),
		}, immediate)

		// then: it is rejected with the deleted error
		pctestutils.RequireTaxCodeDeleted(t, err)
		require.ErrorContains(t, err, "patch at idx 0")
	})
}

// The EditRunning price-change exception also covers a deleted code: a price change that
// echoes the rate card of the version it replaces inherits the deleted reference instead of
// being rejected, so the stored one stays in place until the customer authorizes a new code.
func TestEditRunningPriceChangeWithDeletedTaxCode(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2021-01-01T00:00:00Z")
	clock.SetTime(now)
	defer clock.ResetTime()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	// Deleting a Tax Code checks it isn't an organization default, so the defaults must exist.
	deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

	// given: a published plan whose rate card references a Tax Code, a subscription created
	// from it, and the plan's Tax Code deleted afterwards
	planTaxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	planInput := subscriptiontestutils.GetExamplePlanInput(t, subscriptiontestutils.ExampleTaxConfig(planTaxCode))
	planInput.Plan.Key = "price-change-deleted-tax-code"

	plan := deps.PlanHelper.CreatePlan(t, planInput)

	cust := deps.CustomerAdapter.CreateExampleCustomer(t)

	subView, err := deps.WorkflowService.CreateFromPlan(t.Context(), subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: &now},
			Name:   "subscription for the deleted tax code price change",
		},
		CustomerID: cust.ID,
		Namespace:  subscriptiontestutils.ExampleNamespace,
	}, plan)
	require.NoError(t, err)

	require.NoError(t, deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: planTaxCode.NamespacedID,
	}))

	// when: the item is re-added for a price change carrying the same deleted tax code
	sub, err := deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
		newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: planTaxCode.ID}),
	}, immediate)

	// then: the inherited reference passes and the new price is stored
	require.NoError(t, err)

	phase, ok := sub.GetPhaseByKey("test_phase_1")
	require.True(t, ok)
	items := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
	require.NotEmpty(t, items)

	lastItem := items[len(items)-1]
	cfg := lastItem.Spec.RateCard.AsMeta().TaxConfig
	require.NotNil(t, cfg)
	require.Equal(t, planTaxCode.ID, cfg.TaxCodeID)

	price, err := lastItem.Spec.RateCard.AsMeta().Price.AsFlat()
	require.NoError(t, err)
	require.Equal(t, 42.0, price.Amount.InexactFloat64())
}

// The add-item exception inherits the Tax Code of exactly the version PatchAddItem.ApplyTo
// closes: the last one for the key as the request's earlier patches left it, even if that
// version already ended or hasn't started yet.
func TestEditRunningReplacedVersionWithDeletedTaxCode(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	t.Run("one request swapping to a live code and back to the deleted one is rejected", func(t *testing.T) {
		// given: a subscription whose item holds a deleted Tax Code, and a live one
		deps := newTaxCodeLifecycleDeps(t, now)
		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		subView, plan := deps.deletedSubscription(t, "swap-back-deleted-tax-code")
		deletedID := plan.TaxCodeIDs()[0]
		liveTaxCode := deps.Deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())

		clock.FreezeTime(now.Add(time.Hour))
		defer clock.UnFreeze()

		// when: one request re-adds the item with the live code, then again with the deleted one
		_, err := deps.Deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: liveTaxCode.ID}),
			newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: deletedID}),
		}, immediate)

		// then: the second patch replaces the live version, so it authors the deleted reference
		pctestutils.RequireTaxCodeDeleted(t, err)
		require.ErrorContains(t, err, "patch at idx 1")
	})

	t.Run("re-adding a removed item with its deleted tax code passes", func(t *testing.T) {
		// given: a subscription whose item holds a deleted Tax Code and was removed since
		deps := newTaxCodeLifecycleDeps(t, now)
		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		subView, plan := deps.deletedSubscription(t, "re-add-removed-deleted-tax-code")
		deletedID := plan.TaxCodeIDs()[0]

		clock.FreezeTime(now.Add(24 * time.Hour))
		defer clock.UnFreeze()

		_, err := deps.Deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			patch.PatchRemoveItem{PhaseKey: "test_phase_1", ItemKey: subscriptiontestutils.ExampleFeatureKey},
		}, immediate)
		require.NoError(t, err)

		clock.FreezeTime(now.Add(48 * time.Hour))
		defer clock.UnFreeze()

		// when: the item is re-added with the deleted code of its ended last version
		sub, err := deps.Deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: deletedID}),
		}, immediate)

		// then: ApplyTo replaces the ended version, so the reference is inherited
		require.NoError(t, err)

		phase, ok := sub.GetPhaseByKey("test_phase_1")
		require.True(t, ok)
		items := phase.ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
		require.NotEmpty(t, items)

		cfg := items[len(items)-1].Spec.RateCard.AsMeta().TaxConfig
		require.NotNil(t, cfg)
		require.Equal(t, deletedID, cfg.TaxCodeID)
	})

	t.Run("re-adding the deleted tax code over a scheduled live version is rejected", func(t *testing.T) {
		// given: a subscription whose active item holds a deleted Tax Code, with a version on a
		// live code scheduled later in the same phase
		deps := newTaxCodeLifecycleDeps(t, now)
		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		subView, plan := deps.deletedSubscription(t, "scheduled-live-version")
		deletedID := plan.TaxCodeIDs()[0]
		liveTaxCode := deps.Deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())

		clock.FreezeTime(now.Add(24 * time.Hour))
		defer clock.UnFreeze()

		scheduled := newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: liveTaxCode.ID})
		scheduled.CreateInput.ActiveFromOverrideRelativeToPhaseStart = lo.ToPtr(datetime.MustParseDuration(t, "P10D"))

		_, err := deps.Deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{scheduled}, immediate)
		require.NoError(t, err)

		// when: the item is re-added now with the deleted code of the active version
		_, err = deps.Deps.WorkflowService.EditRunning(t.Context(), subView.Subscription.NamespacedID, []subscription.Patch{
			newTaxedAddItemPatch("test_phase_1", subscriptiontestutils.ExampleFeatureKey, &productcatalog.TaxCodeConfig{TaxCodeID: deletedID}),
		}, immediate)

		// then: ApplyTo would replace the scheduled live version, so the deleted reference is authored
		pctestutils.RequireTaxCodeDeleted(t, err)
		require.ErrorContains(t, err, "patch at idx 0")
	})
}

// newTaxedAddItemPatch builds an add-item patch whose rate card carries the given tax config.
func newTaxedAddItemPatch(phaseKey, itemKey string, taxConfig *productcatalog.TaxCodeConfig) patch.PatchAddItem {
	rateCard := &productcatalog.FlatFeeRateCard{
		RateCardMeta: productcatalog.RateCardMeta{
			Key:       itemKey,
			Name:      itemKey,
			TaxConfig: taxConfig,
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount:      alpacadecimal.NewFromInt(42),
				PaymentTerm: productcatalog.InArrearsPaymentTerm,
			}),
		},
	}

	return patch.PatchAddItem{
		PhaseKey: phaseKey,
		ItemKey:  itemKey,
		CreateInput: subscription.SubscriptionItemSpec{
			CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
				CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
					PhaseKey: phaseKey,
					ItemKey:  itemKey,
					RateCard: rateCard,
				},
			},
		},
	}
}
