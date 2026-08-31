package subscriptions

import (
	"testing"

	"github.com/oapi-codegen/nullable"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// TestEditRunningAddItemRoundtrip exercises the full edit path end-to-end minus
// HTTP: an API add_item operation is mapped to a domain patch via
// FromAPIBillingSubscriptionEditOperation, applied through the real EditRunning
// workflow, and the resulting view converted back with ToAPIBillingSubscription.
// It is the counterpart to the create-side TestToAPIBillingSubscriptionViewRoundtrip
// and self-skips without a Postgres backend.
func TestEditRunningAddItemRoundtrip(t *testing.T) {
	ctx := t.Context()

	// given: time is frozen at the subscription start so phase 1 is still a future phase
	currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.FreezeTime(currentTime)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	// given: a running subscription created from the example plan (phases
	// test_phase_0/1/2), so there is a future phase to edit
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)
	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

	spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:      cust.ID,
		InvoiceCurrency: "USD",
		ActiveFrom:      currentTime,
		BillingAnchor:   currentTime,
		Name:            "Test Subscription",
		Annotations:     models.Annotations{},
	})
	require.NoError(t, err)

	sub, err := deps.SubscriptionService.Create(ctx, subscriptiontestutils.ExampleNamespace, spec)
	require.NoError(t, err)

	// when: an add_item edit (a free flat-fee rate card on the future phase) is
	// mapped through the API converter and applied via EditRunning
	var price api.BillingPrice
	require.NoError(t, price.FromBillingPriceFree(api.BillingPriceFree{Type: "free"}))

	var op api.BillingSubscriptionEditOperation
	require.NoError(t, op.FromBillingSubscriptionEditAddItem(api.BillingSubscriptionEditAddItem{
		Type:     api.BillingSubscriptionEditAddItemTypeAddItem,
		PhaseKey: "test_phase_1",
		RateCard: api.BillingRateCard{Key: "integration_extra", Name: "Integration Extra", Price: price},
	}))

	p, err := FromAPIBillingSubscriptionEditOperation(op)
	require.NoError(t, err)

	view, err := deps.WorkflowService.EditRunning(
		ctx,
		models.NamespacedID{Namespace: sub.Namespace, ID: sub.ID},
		[]subscription.Patch{p},
		subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
	)
	require.NoError(t, err)

	// then: the converted subscription exposes the new item on the target phase
	result, err := ToAPIBillingSubscription(view)
	require.NoError(t, err)

	var found bool
	for _, phase := range result.Phases {
		if phase.Key != "test_phase_1" {
			continue
		}
		for _, item := range phase.Items {
			if item.RateCard.Key == "integration_extra" {
				found = true
			}
		}
	}
	require.True(t, found, "expected the added item to appear on test_phase_1 after edit")
}

// TestEditRunningAddPhaseRoundtrip exercises the add_phase edit end-to-end minus
// HTTP. A lone add_phase would produce an empty phase (invalid — a phase must have
// at least one item), so it is batched with an add_item that populates the new
// phase. The phase is inserted at P2M, which lands between the two future phases
// (starting at P1M and P3M) so the current phase is untouched and the new P1M
// phase tiles the gap exactly, requiring no shift of the trailing phase.
func TestEditRunningAddPhaseRoundtrip(t *testing.T) {
	ctx := t.Context()

	// given: time frozen at the subscription start so phases after the first are
	// still in the future
	currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.FreezeTime(currentTime)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	// given: a running subscription created from the example plan (phase durations
	// P1M, P2M, open) — so future phases start at P1M and P3M
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)
	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

	spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:      cust.ID,
		InvoiceCurrency: "USD",
		ActiveFrom:      currentTime,
		BillingAnchor:   currentTime,
		Name:            "Test Subscription",
		Annotations:     models.Annotations{},
	})
	require.NoError(t, err)

	sub, err := deps.SubscriptionService.Create(ctx, subscriptiontestutils.ExampleNamespace, spec)
	require.NoError(t, err)

	// when: an add_phase (inserting a bounded phase between the two future phases)
	// batched with an add_item that populates it, mapped through the API converter
	// and applied via EditRunning
	var addPhaseOp api.BillingSubscriptionEditOperation
	require.NoError(t, addPhaseOp.FromBillingSubscriptionEditAddPhase(api.BillingSubscriptionEditAddPhase{
		Type: api.BillingSubscriptionEditAddPhaseTypeAddPhase,
		Phase: api.BillingSubscriptionPhaseCreate{
			Key:        "integration_phase",
			Name:       "Integration Phase",
			StartAfter: nullable.NewNullableWithValue(api.ISO8601Duration("P2M")),
			Duration:   lo.ToPtr("P1M"),
		},
	}))

	var price api.BillingPrice
	require.NoError(t, price.FromBillingPriceFree(api.BillingPriceFree{Type: "free"}))

	var addItemOp api.BillingSubscriptionEditOperation
	require.NoError(t, addItemOp.FromBillingSubscriptionEditAddItem(api.BillingSubscriptionEditAddItem{
		Type:     api.BillingSubscriptionEditAddItemTypeAddItem,
		PhaseKey: "integration_phase",
		RateCard: api.BillingRateCard{Key: "integration_phase_item", Name: "Integration Phase Item", Price: price},
	}))

	patches := make([]subscription.Patch, 0, 2)
	for _, op := range []api.BillingSubscriptionEditOperation{addPhaseOp, addItemOp} {
		p, err := FromAPIBillingSubscriptionEditOperation(op)
		require.NoError(t, err)
		patches = append(patches, p)
	}

	view, err := deps.WorkflowService.EditRunning(
		ctx,
		models.NamespacedID{Namespace: sub.Namespace, ID: sub.ID},
		patches,
		subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
	)
	require.NoError(t, err)

	// then: the converted subscription exposes the new phase with its populating item
	result, err := ToAPIBillingSubscription(view)
	require.NoError(t, err)

	var found bool
	for _, phase := range result.Phases {
		if phase.Key != "integration_phase" {
			continue
		}
		for _, item := range phase.Items {
			if item.RateCard.Key == "integration_phase_item" {
				found = true
			}
		}
	}
	require.True(t, found, "expected the added phase with its item to appear after edit")
}
