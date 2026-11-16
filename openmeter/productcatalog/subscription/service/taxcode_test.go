package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	pctestutils "github.com/openmeterio/openmeter/openmeter/productcatalog/testutils"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// newTaxedCustomPlanInput builds an inline plan whose taxed rate cards carry taxConfig, suitable
// for the custom-plan branch of subscription create and change.
func newTaxedCustomPlanInput(t *testing.T, taxConfig productcatalog.TaxCodeConfig) plan.CreatePlanInput {
	t.Helper()

	planInput := subscriptiontestutils.GetExamplePlanInput(t, taxConfig)
	planInput.Plan.Key = ""
	planInput.Plan.Version = 0

	return planInput
}

func provisionDeletedTaxCode(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) productcatalog.TaxCodeConfig {
	t.Helper()

	// Deleting a Tax Code checks it isn't an organization default, so the defaults must exist.
	deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

	deleted := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	require.NoError(t, deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: deleted.NamespacedID,
	}))

	return subscriptiontestutils.ExampleTaxConfig(deleted)
}

// The inline plan of a custom subscription create or change is authored by the request, so every
// Tax Code its rate cards reference must be live: deleted codes are rejected with
// rate_card_tax_code_deleted, missing ones with rate_card_tax_code_not_found.
func TestCreateInlineCustomPlanTaxCodeReferences(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	svc := newPlanSubscriptionService(t, deps, testutils.NewLogger(t))
	customer := deps.CustomerAdapter.CreateExampleCustomer(t)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	liveTaxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	deletedTaxConfig := provisionDeletedTaxCode(t, deps)

	t.Run("live tax code is accepted", func(t *testing.T) {
		// given: an inline plan referencing a live Tax Code
		planInput := newTaxedCustomPlanInput(t, subscriptiontestutils.ExampleTaxConfig(liveTaxCode))
		requestPlan := plansubscription.PlanInput{}
		requestPlan.FromInput(&planInput)

		// when: the subscription is created from it
		sub, err := svc.Create(t.Context(), plansubscription.CreateSubscriptionRequest{
			PlanInput: requestPlan,
			WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Name:   "inline plan with live tax code",
					Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
				},
				Namespace:  customer.Namespace,
				CustomerID: customer.ID,
			},
		})

		// then: the subscription is created
		require.NoError(t, err)
		require.NotEmpty(t, sub.ID)
	})

	t.Run("deleted tax code is rejected", func(t *testing.T) {
		// given: an inline plan referencing a deleted Tax Code
		planInput := newTaxedCustomPlanInput(t, deletedTaxConfig)
		requestPlan := plansubscription.PlanInput{}
		requestPlan.FromInput(&planInput)

		// when: the subscription is created from it
		_, err := svc.Create(t.Context(), plansubscription.CreateSubscriptionRequest{
			PlanInput: requestPlan,
			WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Name:   "inline plan with deleted tax code",
					Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
				},
				Namespace:  customer.Namespace,
				CustomerID: customer.ID,
			},
		})

		// then: it is rejected with the deleted error on the plan's rate card field
		pctestutils.RequireTaxCodeDeleted(t, err)
		assertIssueOnField(t, err, "plan.phases")
		assertIssueOnField(t, err, "ratecards")
	})

	t.Run("missing tax code is rejected", func(t *testing.T) {
		// given: an inline plan referencing an unknown Tax Code
		planInput := newTaxedCustomPlanInput(t, productcatalog.TaxCodeConfig{TaxCodeID: "01JABSENTTAXC0DE0000000000"})
		requestPlan := plansubscription.PlanInput{}
		requestPlan.FromInput(&planInput)

		// when: the subscription is created from it
		_, err := svc.Create(t.Context(), plansubscription.CreateSubscriptionRequest{
			PlanInput: requestPlan,
			WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Name:   "inline plan with missing tax code",
					Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
				},
				Namespace:  customer.Namespace,
				CustomerID: customer.ID,
			},
		})

		// then: it is rejected with the not-found error on the plan field
		pctestutils.RequireTaxCodeNotFound(t, err)
		assertIssueOnField(t, err, "plan.phases")
	})
}

// Changing to an inline custom plan is authored by the request, so the same live-only rule
// applies as for create.
func TestChangeInlineCustomPlanTaxCodeReferences(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	svc := newPlanSubscriptionService(t, deps, testutils.NewLogger(t))
	customer := deps.CustomerAdapter.CreateExampleCustomer(t)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	liveTaxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	deletedTaxConfig := provisionDeletedTaxCode(t, deps)

	// given: a running subscription created from an inline plan with a live tax code
	startPlanInput := newTaxedCustomPlanInput(t, subscriptiontestutils.ExampleTaxConfig(liveTaxCode))
	startPlan := plansubscription.PlanInput{}
	startPlan.FromInput(&startPlanInput)

	sub, err := svc.Create(t.Context(), plansubscription.CreateSubscriptionRequest{
		PlanInput: startPlan,
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Name:   "subscription before tax code change",
				Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
			},
			Namespace:  customer.Namespace,
			CustomerID: customer.ID,
		},
	})
	require.NoError(t, err)

	t.Run("deleted tax code is rejected", func(t *testing.T) {
		// given: an inline replacement plan referencing a deleted Tax Code
		planInput := newTaxedCustomPlanInput(t, deletedTaxConfig)
		replacementPlan := plansubscription.PlanInput{}
		replacementPlan.FromInput(&planInput)

		// when: the subscription is changed to it
		_, err := svc.Change(t.Context(), plansubscription.ChangeSubscriptionRequest{
			ID:        sub.NamespacedID,
			PlanInput: replacementPlan,
			WorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Name:   sub.Name,
				Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
			},
		})

		// then: it is rejected with the deleted error on the plan's rate card field
		pctestutils.RequireTaxCodeDeleted(t, err)
		assertIssueOnField(t, err, "plan.phases")
		assertIssueOnField(t, err, "ratecards")
	})

	t.Run("missing tax code is rejected", func(t *testing.T) {
		// given: an inline replacement plan referencing an unknown Tax Code
		planInput := newTaxedCustomPlanInput(t, productcatalog.TaxCodeConfig{TaxCodeID: "01JABSENTTAXC0DE0000000000"})
		replacementPlan := plansubscription.PlanInput{}
		replacementPlan.FromInput(&planInput)

		// when: the subscription is changed to it
		_, err := svc.Change(t.Context(), plansubscription.ChangeSubscriptionRequest{
			ID:        sub.NamespacedID,
			PlanInput: replacementPlan,
			WorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Name:   sub.Name,
				Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
			},
		})

		// then: it is rejected with the not-found error on the plan's rate card field
		pctestutils.RequireTaxCodeNotFound(t, err)
		assertIssueOnField(t, err, "plan.phases")
		assertIssueOnField(t, err, "ratecards")
	})
}

// A plan reference is inherited: the published plan's deleted Tax Code comes from something
// already approved, so subscribing to it succeeds even though the same reference is rejected
// when authored inline.
func TestCreateFromPlanRefTaxCodeDeleted(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	svc := newPlanSubscriptionService(t, deps, testutils.NewLogger(t))
	customer := deps.CustomerAdapter.CreateExampleCustomer(t)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	// given: a published plan whose rate card references a Tax Code that gets deleted afterwards
	doomedTaxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	taxConfig := subscriptiontestutils.ExampleTaxConfig(doomedTaxCode)

	// Deleting a Tax Code checks it isn't an organization default, so the defaults must exist.
	deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

	planInput := subscriptiontestutils.GetExamplePlanInput(t, taxConfig)
	published := deps.PlanHelper.CreatePlan(t, planInput)

	require.NoError(t, deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: doomedTaxCode.NamespacedID,
	}))

	// The plan is published with EffectiveFrom at its creation time, which is only active
	// strictly after that instant.
	clock.SetTime(now.Add(time.Second))

	// when: the subscription is created from the plan reference
	refPlan := plansubscription.PlanInput{}
	refPlan.FromRef(&plansubscription.PlanRefInput{
		Key:     published.ToCreateSubscriptionPlanInput().Plan.Key,
		Version: &published.ToCreateSubscriptionPlanInput().Plan.Version,
	})

	sub, err := svc.Create(t.Context(), plansubscription.CreateSubscriptionRequest{
		PlanInput: refPlan,
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Name:   "subscription from plan with deleted tax code",
				Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
			},
			Namespace:  customer.Namespace,
			CustomerID: customer.ID,
		},
	})

	// then: creation succeeds and the item carries the inherited deleted reference
	require.NoError(t, err)

	view, err := deps.SubscriptionService.GetView(t.Context(), sub.NamespacedID)
	require.NoError(t, err)

	requireItemWithTaxCode(t, view, doomedTaxCode.ID)
}

// Changing to a plan reference inherits the published plan's references: the deleted Tax Code
// passes, unlike an inline plan carrying the same reference.
func TestChangeToPlanRefTaxCodeDeleted(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	svc := newPlanSubscriptionService(t, deps, testutils.NewLogger(t))
	customer := deps.CustomerAdapter.CreateExampleCustomer(t)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	liveTaxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	startPlan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t, subscriptiontestutils.ExampleTaxConfig(liveTaxCode)))

	// The plan is published with EffectiveFrom at its creation time, which is only active
	// strictly after that instant.
	clock.SetTime(now.Add(time.Second))

	startRef := plansubscription.PlanInput{}
	startRef.FromRef(&plansubscription.PlanRefInput{
		Key:     startPlan.ToCreateSubscriptionPlanInput().Plan.Key,
		Version: &startPlan.ToCreateSubscriptionPlanInput().Plan.Version,
	})

	sub, err := svc.Create(t.Context(), plansubscription.CreateSubscriptionRequest{
		PlanInput: startRef,
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Name:   "subscription before deleted tax code change",
				Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
			},
			Namespace:  customer.Namespace,
			CustomerID: customer.ID,
		},
	})
	require.NoError(t, err)

	// given: a second published plan whose rate card references a Tax Code deleted after publishing
	doomedTaxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	targetPlanInput := subscriptiontestutils.GetExamplePlanInput(t, subscriptiontestutils.ExampleTaxConfig(doomedTaxCode))
	targetPlanInput.Plan.Key = "target-plan-deleted-tax-code"

	// Deleting a Tax Code checks it isn't an organization default, so the defaults must exist.
	deps.TaxCodeEnv.ProvisionDefaultTaxCodes(t, subscriptiontestutils.ExampleNamespace)

	targetPlan, err := deps.PlanService.CreatePlan(t.Context(), targetPlanInput)
	require.NoError(t, err)

	publishAt := clock.Now().Add(time.Minute)
	_, err = deps.PlanService.PublishPlan(t.Context(), plan.PublishPlanInput{
		NamespacedID: targetPlan.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: &publishAt,
		},
	})
	require.NoError(t, err)

	require.NoError(t, deps.TaxCodeEnv.Service.DeleteTaxCode(t.Context(), taxcode.DeleteTaxCodeInput{
		NamespacedID: doomedTaxCode.NamespacedID,
	}))

	clock.SetTime(publishAt.Add(time.Second))

	// when: the subscription is changed to the plan reference
	targetRef := plansubscription.PlanInput{}
	targetRef.FromRef(&plansubscription.PlanRefInput{
		Key:     targetPlan.Key,
		Version: &targetPlan.Version,
	})

	resp, err := svc.Change(t.Context(), plansubscription.ChangeSubscriptionRequest{
		ID:        sub.NamespacedID,
		PlanInput: targetRef,
		WorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Name:   sub.Name,
			Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
		},
	})

	// then: the change succeeds and the new subscription keeps the deleted reference
	require.NoError(t, err)
	require.Equal(t, targetPlan.Key, resp.Next.Subscription.PlanRef.Key)
	requireItemWithTaxCode(t, resp.Next, doomedTaxCode.ID)
}

// requireItemWithTaxCode fails unless an item of the subscription references the Tax Code,
// so a sync that silently drops an inherited deleted reference is caught.
func requireItemWithTaxCode(t *testing.T, view subscription.SubscriptionView, taxCodeID string) {
	t.Helper()

	for _, phase := range view.Phases {
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				cfg := item.SubscriptionItem.RateCard.AsMeta().TaxConfig
				if cfg != nil && cfg.TaxCodeID == taxCodeID {
					return
				}
			}
		}
	}

	require.Fail(t, "expected an item to reference the tax code", taxCodeID)
}

// assertIssueOnField requires the error to carry a validation issue whose field path
// contains the given segment, so the offending rate card stays identifiable.
func assertIssueOnField(t *testing.T, err error, fieldPart string) {
	t.Helper()

	issues, systemErr := models.AsValidationIssues(err)
	require.NoError(t, systemErr)
	require.NotEmpty(t, issues)

	for _, issue := range issues {
		if issue.Field() != nil && strings.Contains(issue.Field().String(), fieldPart) {
			return
		}
	}

	require.Failf(t, "expected an issue on field", "no issue with field %s, got %v", fieldPart, issues)
}
