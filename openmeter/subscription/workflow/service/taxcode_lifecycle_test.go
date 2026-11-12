package service_test

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// A subscription whose items reference a deleted Tax Code keeps operating: sync doesn't check
// Tax Codes and these operations inherit the stored references, so cancel, continue, migration
// and add-on changes all succeed.
func TestSubscriptionLifecycleWithDeletedTaxCode(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	t.Run("cancel immediately succeeds", func(t *testing.T) {
		// given: a subscription holding a deleted Tax Code reference
		deps := newTaxCodeLifecycleDeps(t, now)

		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		subView, _ := deps.deletedSubscription(t, "cancel-immediate-deleted-tax-code")

		// when: the subscription is canceled immediately
		_, err := deps.Deps.SubscriptionService.Cancel(t.Context(), subView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingImmediate),
		})

		// then: the inherited reference passes
		require.NoError(t, err)
	})

	t.Run("cancel at the next billing cycle succeeds", func(t *testing.T) {
		// given: a subscription holding a deleted Tax Code reference
		deps := newTaxCodeLifecycleDeps(t, now)

		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		subView, _ := deps.deletedSubscription(t, "cancel-next-cycle-deleted-tax-code")

		// when: the subscription is canceled at the next billing cycle
		sub, err := deps.Deps.SubscriptionService.Cancel(t.Context(), subView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		})

		// then: the inherited reference passes
		require.NoError(t, err)
		require.NotNil(t, sub.ActiveTo)
	})

	t.Run("continue after cancellation succeeds", func(t *testing.T) {
		// given: a subscription holding a deleted Tax Code reference, canceled at the next
		// billing cycle
		deps := newTaxCodeLifecycleDeps(t, now)

		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		subView, _ := deps.deletedSubscription(t, "continue-deleted-tax-code")

		_, err := deps.Deps.SubscriptionService.Cancel(t.Context(), subView.Subscription.NamespacedID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		})
		require.NoError(t, err)

		// when: the canceled subscription is continued
		_, err = deps.Deps.SubscriptionService.Continue(t.Context(), subView.Subscription.NamespacedID)

		// then: the inherited reference passes
		require.NoError(t, err)
	})

	t.Run("migrating to a later published version holding the deleted tax code succeeds", func(t *testing.T) {
		// The subscription's items are replaced by the target version's items through sync, and
		// the target version inherits the deleted reference the previous version already holds.
		// The version bump is published while the code is still live, like an in-flight version
		// change, so the deletion only affects references authored after it.
		// given: a subscription running on version 1 of a plan whose Tax Code is deleted after
		// version 2 of the same plan is published with a changed price
		deps := newTaxCodeLifecycleDeps(t, now)

		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		// Deleting a Tax Code checks it isn't an organization default, so the defaults must exist.
		deps.Deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

		liveTaxCode := deps.Deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
		taxConfig := subscriptiontestutils.ExampleTaxConfig(liveTaxCode)
		planKey := "migrate-published-deleted-tax-code"

		startInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, subscriptiontestutils.ExampleRateCard1(taxConfig)).Build()
		startInput.Plan.Key = planKey

		startDraft, err := deps.Deps.PlanService.CreatePlan(t.Context(), startInput)
		require.NoError(t, err)

		// The clock is frozen a millisecond past now, so version 1 is active for everything
		// that follows, including the publish of version 2 archiving it.
		startPublished, err := deps.Deps.PlanService.PublishPlan(t.Context(), plan.PublishPlanInput{
			NamespacedID: startDraft.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			},
		})
		require.NoError(t, err)

		startPlan := &plansubscription.Plan{
			Plan: startPublished.AsProductCatalogPlan(),
			Ref:  &startPublished.NamespacedID,
		}

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.Deps, startPlan, now)

		// The version bump must re-create the item, so it differs in price while holding the
		// same Tax Code reference, and it goes effective the next day like a scheduled change.
		nextInput := subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, subscriptiontestutils.ExampleRateCard1(taxConfig)).Build()
		nextInput.Plan.Key = planKey

		require.NoError(t, nextInput.Phases[0].RateCards[0].ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
			meta.Price = productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(200)})

			return meta, nil
		}))

		nextDay := now.AddDate(0, 0, 1)

		nextDraft, err := deps.Deps.PlanService.CreatePlan(t.Context(), nextInput)
		require.NoError(t, err)

		nextPublished, err := deps.Deps.PlanService.PublishPlan(t.Context(), plan.PublishPlanInput{
			NamespacedID: nextDraft.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(nextDay),
			},
		})
		require.NoError(t, err)

		nextPlan := &plansubscription.Plan{
			Plan: nextPublished.AsProductCatalogPlan(),
			Ref:  &nextPublished.NamespacedID,
		}

		require.NoError(t, deps.Deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: liveTaxCode.NamespacedID,
		}))

		// The version bump must be active at migration time, like a scheduled change that
		// has already kicked in.
		clock.FreezeTime(nextDay.Add(time.Hour))
		defer clock.UnFreeze()

		require.Equal(t, productcatalog.PlanStatusActive, nextPublished.Status())

		// when: the subscription migrates to version 2
		_, after, err := deps.Deps.WorkflowService.MigrateToPlan(t.Context(), subscriptionworkflow.MigrateSubscriptionWorkflowInput{
			SubscriptionID: subView.Subscription.NamespacedID,
			Plan:           nextPlan,
			Timing:         subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
		})

		// then: the migrated subscription's items inherit the deleted reference and the new price
		require.NoError(t, err)
		require.Equal(t, 2, after.Subscription.PlanRef.Version)

		items := after.Phases[0].ItemsByKey[subscriptiontestutils.ExampleFeatureKey]
		require.Len(t, items, 2)

		cfg := items[len(items)-1].Spec.RateCard.AsMeta().TaxConfig
		require.NotNil(t, cfg)
		require.Equal(t, liveTaxCode.ID, cfg.TaxCodeID)

		price, err := items[len(items)-1].Spec.RateCard.AsMeta().Price.AsUnit()
		require.NoError(t, err)
		require.Equal(t, 200.0, price.Amount.InexactFloat64())
	})

	t.Run("adding an addon succeeds", func(t *testing.T) {
		// given: a subscription with a deleted tax code and an add-on linked to its plan.
		// The add-on must be linked while the plan is still a draft, so the whole setup runs
		// before the plan is published.
		deps := newTaxCodeLifecycleDeps(t, now)

		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		deps.Deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

		liveTaxCode := deps.Deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
		planInput := subscriptiontestutils.GetExamplePlanInput(t, subscriptiontestutils.ExampleTaxConfig(liveTaxCode))
		planInput.Plan.Key = "add-addon-deleted-tax-code"

		draftPlan, err := deps.Deps.PlanService.CreatePlan(t.Context(), planInput)
		require.NoError(t, err)

		addInp := subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(clock.Now()),
			},
			productcatalog.AddonInstanceTypeSingle,
			subscriptiontestutils.ExampleAddonRateCard2.Clone(),
		)
		add, err := deps.Deps.AddonService.CreateAddon(t.Context(), addInp)
		require.NoError(t, err)

		// The add-on must be active before it can be attached, and the plan must still be a
		// draft while the add-on is attached to it.
		// The frozen clock is a millisecond past now, so an EffectiveFrom of now makes the
		// add-on and the plan active for everything that follows.
		add, err = deps.Deps.AddonService.PublishAddon(t.Context(), addon.PublishAddonInput{
			NamespacedID: add.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			},
		})
		require.NoError(t, err)

		_, err = deps.Deps.PlanAddonService.CreatePlanAddon(t.Context(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: draftPlan.Namespace,
			},
			PlanID:        draftPlan.ID,
			AddonID:       add.ID,
			FromPlanPhase: draftPlan.Phases[0].Key,
		})
		require.NoError(t, err)

		published, err := deps.Deps.PlanService.PublishPlan(t.Context(), plan.PublishPlanInput{
			NamespacedID: draftPlan.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			},
		})
		require.NoError(t, err)

		subPlan := &plansubscription.Plan{
			Plan: published.AsProductCatalogPlan(),
			Ref:  &published.NamespacedID,
		}

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.Deps, subPlan, now)

		require.NoError(t, deps.Deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: liveTaxCode.NamespacedID,
		}))

		// when: the add-on is purchased
		_, _, err = deps.Deps.WorkflowService.AddAddon(t.Context(), subView.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		})

		// then: the subscription's inherited deleted reference passes
		require.NoError(t, err)
	})

	t.Run("changing an addon quantity succeeds", func(t *testing.T) {
		// given: a subscription with a deleted tax code holding a purchased add-on
		deps := newTaxCodeLifecycleDeps(t, now)

		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		deps.Deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

		liveTaxCode := deps.Deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
		planInput := subscriptiontestutils.GetExamplePlanInput(t, subscriptiontestutils.ExampleTaxConfig(liveTaxCode))
		planInput.Plan.Key = "change-addon-qty-deleted-tax-code"

		draftPlan, err := deps.Deps.PlanService.CreatePlan(t.Context(), planInput)
		require.NoError(t, err)

		addInp := subscriptiontestutils.BuildAddonForTesting(t,
			productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(clock.Now()),
			},
			productcatalog.AddonInstanceTypeSingle,
			subscriptiontestutils.ExampleAddonRateCard2.Clone(),
		)
		add, err := deps.Deps.AddonService.CreateAddon(t.Context(), addInp)
		require.NoError(t, err)

		// The add-on must be active before it can be attached, and the plan must still be a
		// draft while the add-on is attached to it.
		// The frozen clock is a millisecond past now, so an EffectiveFrom of now makes the
		// add-on and the plan active for everything that follows.
		add, err = deps.Deps.AddonService.PublishAddon(t.Context(), addon.PublishAddonInput{
			NamespacedID: add.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			},
		})
		require.NoError(t, err)

		_, err = deps.Deps.PlanAddonService.CreatePlanAddon(t.Context(), planaddon.CreatePlanAddonInput{
			NamespacedModel: models.NamespacedModel{
				Namespace: draftPlan.Namespace,
			},
			PlanID:        draftPlan.ID,
			AddonID:       add.ID,
			FromPlanPhase: draftPlan.Phases[0].Key,
		})
		require.NoError(t, err)

		published, err := deps.Deps.PlanService.PublishPlan(t.Context(), plan.PublishPlanInput{
			NamespacedID: draftPlan.NamespacedID,
			EffectivePeriod: productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			},
		})
		require.NoError(t, err)

		subPlan := &plansubscription.Plan{
			Plan: published.AsProductCatalogPlan(),
			Ref:  &published.NamespacedID,
		}

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.Deps, subPlan, now)

		require.NoError(t, deps.Deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
			NamespacedID: liveTaxCode.NamespacedID,
		}))

		// when: the add-on is purchased
		_, subAdd, err := deps.Deps.WorkflowService.AddAddon(t.Context(), subView.Subscription.NamespacedID, subscriptionworkflow.AddAddonWorkflowInput{
			AddonID:         add.ID,
			InitialQuantity: 1,
			Timing: subscription.Timing{
				Custom: &now,
			},
		})
		require.NoError(t, err)

		changeTime := now.AddDate(0, 0, 1)

		// when: the add-on's quantity is changed to 0
		_, _, err = deps.Deps.WorkflowService.ChangeAddonQuantity(t.Context(), subView.Subscription.NamespacedID, subscriptionworkflow.ChangeAddonQuantityWorkflowInput{
			SubscriptionAddonID: models.NamespacedID{
				ID:        subAdd.ID,
				Namespace: subView.Subscription.Namespace,
			},
			Quantity: 0,
			Timing: subscription.Timing{
				Custom: &changeTime,
			},
		})

		// then: the subscription's inherited deleted reference passes
		require.NoError(t, err)
	})
}

// A plan reference is inherited by both create and change: the published plan's deleted Tax Code
// comes from something already approved, so both succeed.
func TestWorkflowPlanReferenceWithDeletedTaxCode(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	t.Run("create from a plan reference holding a deleted tax code succeeds", func(t *testing.T) {
		// given: a published plan holding a deleted Tax Code reference
		deps := newTaxCodeLifecycleDeps(t, now)

		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		plan := deps.deletedPlan(t, "create-deleted-tax-code")

		// when: the subscription is created from the plan reference
		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.Deps, plan, now)

		// then: the inherited reference passes
		require.NotEmpty(t, subView.Subscription.ID)
	})

	t.Run("change to a plan reference holding a deleted tax code succeeds", func(t *testing.T) {
		// given: a running subscription and a published plan, both holding deleted Tax Code
		// references
		deps := newTaxCodeLifecycleDeps(t, now)

		deps.Deps.FeatureConnector.CreateExampleFeatures(t, deps.Deps.ExampleMeterID)

		startPlan := deps.deletedPlan(t, "change-start-deleted-tax-code")

		subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps.Deps, startPlan, now)

		targetPlan := deps.deletedPlan(t, "change-target-deleted-tax-code")

		// when: the subscription is changed to the target plan reference
		_, new, err := deps.Deps.WorkflowService.ChangeToPlan(t.Context(), subView.Subscription.NamespacedID, subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
			Name:   subView.Subscription.Name,
		}, targetPlan)

		// then: the inherited reference passes
		require.NoError(t, err)
		require.Equal(t, "change-target-deleted-tax-code", new.Subscription.PlanRef.Key)
	})
}

// taxCodeLifecycleSuite holds the frozen-clock environment shared by the deleted-tax-code
// lifecycle tests: the clock is frozen a millisecond past the suite's base time, so plans and
// add-ons published with EffectiveFrom at the base time are active for everything that follows.
type taxCodeLifecycleSuite struct {
	Deps subscriptiontestutils.SubscriptionDependencies

	now time.Time
}

// newTaxCodeLifecycleDeps freezes the clock and builds the test dependencies. A t.Cleanup
// registered here unfreezes the clock and tears the database down when the subtest ends.
func newTaxCodeLifecycleDeps(t *testing.T, now time.Time) *taxCodeLifecycleSuite {
	t.Helper()

	clock.FreezeTime(now.Add(time.Millisecond))

	dbDeps := subscriptiontestutils.SetupDBDeps(t)

	t.Cleanup(func() {
		clock.UnFreeze()
		dbDeps.Cleanup(t)
	})

	return &taxCodeLifecycleSuite{
		Deps: subscriptiontestutils.NewService(t, dbDeps),
		now:  now,
	}
}

// deletedSubscription creates a subscription from a published plan whose Tax Code is deleted
// right after, so every item of the subscription references a soft-deleted code. planKey keeps
// the plans of different subtests apart.
func (s *taxCodeLifecycleSuite) deletedSubscription(t *testing.T, planKey string) (subscription.SubscriptionView, *plansubscription.Plan) {
	t.Helper()

	// Deleting a Tax Code checks it isn't an organization default, so the defaults must exist.
	s.Deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

	liveTaxCode := s.Deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	planInput := subscriptiontestutils.GetExamplePlanInput(t, subscriptiontestutils.ExampleTaxConfig(liveTaxCode))
	planInput.Plan.Key = planKey

	plan := s.Deps.PlanHelper.CreatePlan(t, planInput)
	concretePlan := plan.(*plansubscription.Plan)

	subView := subscriptiontestutils.CreateSubscriptionFromPlan(t, &s.Deps, plan, s.now)

	require.NoError(t, s.Deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: liveTaxCode.NamespacedID,
	}))

	return subView, concretePlan
}

// deletedPlan returns the plan of a Tax Code deleted right after the plan holding it is
// published. The plan's rate cards keep referencing the deleted code.
func (s *taxCodeLifecycleSuite) deletedPlan(t *testing.T, key string) *plansubscription.Plan {
	t.Helper()

	// Deleting a Tax Code checks it isn't an organization default, so the defaults must exist.
	s.Deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

	liveTaxCode := s.Deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	planInput := subscriptiontestutils.GetExamplePlanInput(t, subscriptiontestutils.ExampleTaxConfig(liveTaxCode))
	planInput.Plan.Key = key

	plan := s.Deps.PlanHelper.CreatePlan(t, planInput)
	concretePlan := plan.(*plansubscription.Plan)

	require.NoError(t, s.Deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: liveTaxCode.NamespacedID,
	}))

	return concretePlan
}
