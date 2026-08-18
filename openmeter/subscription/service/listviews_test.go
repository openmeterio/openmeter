package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// TestListViewsMultiCustomer verifies that ListViews (and the multi-customer
// ExpandViews it relies on) attaches each subscription's own customer to its view,
// rather than stamping a single customer onto every view. PostgreSQL-backed; skips
// when Postgres is unavailable.
func TestListViewsMultiCustomer(t *testing.T) {
	ctx := t.Context()

	currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.FreezeTime(currentTime)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	service := deps.SubscriptionService

	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

	// given: two distinct customers, each with a subscription from the same plan
	cust1 := deps.CustomerAdapter.CreateExampleCustomer(t)
	cust2 := deps.CustomerAdapter.CreateExampleCustomerWithSubject(t, "Jane Doe", "jane-doe")

	createSubscriptionFromPlan(t, ctx, service, plan, cust1.ID, currentTime)
	createSubscriptionFromPlan(t, ctx, service, plan, cust2.ID, currentTime)

	// when: listing views across the namespace, which spans both customers
	res, err := service.ListViews(ctx, subscription.ListSubscriptionsInput{
		Namespaces: []string{subscriptiontestutils.ExampleNamespace},
		Page:       pagination.NewPage(1, 10),
	})
	require.NoError(t, err)

	// then: both subscriptions come back, each carrying its OWN customer and phases.
	// (With the previous single-customer expansion, both views would share one customer.)
	require.Equal(t, 2, res.TotalCount)
	require.Len(t, res.Items, 2)

	viewByCustomer := make(map[string]subscription.SubscriptionView, len(res.Items))
	for _, view := range res.Items {
		viewByCustomer[view.Subscription.CustomerId] = view
		require.NotEmpty(t, view.Phases, "each view should carry its phases")
	}

	require.Contains(t, viewByCustomer, cust1.ID)
	require.Contains(t, viewByCustomer, cust2.ID)
	require.Equal(t, cust1.ID, viewByCustomer[cust1.ID].Customer.ID)
	require.Equal(t, cust2.ID, viewByCustomer[cust2.ID].Customer.ID)
}

// TestGetViewResolvesSoftDeletedCustomer guards a regression: expanding a view
// resolves a subscription's customer even after that customer is soft-deleted. The
// prior GetCustomer-by-ID path returned soft-deleted customers, so the batched
// ListCustomers in ExpandViews must keep that behavior (IncludeDeleted); otherwise
// GetView — and everything built on it — would start failing for these subscriptions.
// PostgreSQL-backed; skips when Postgres is unavailable.
func TestGetViewResolvesSoftDeletedCustomer(t *testing.T) {
	ctx := t.Context()

	currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.FreezeTime(currentTime)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	service := deps.SubscriptionService

	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t))

	// given: a subscription whose customer is subsequently soft-deleted
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)

	sub := createSubscriptionFromPlan(t, ctx, service, plan, cust.ID, currentTime)

	require.NoError(t, deps.CustomerAdapter.DeleteCustomer(ctx, customer.CustomerID{
		Namespace: subscriptiontestutils.ExampleNamespace,
		ID:        cust.ID,
	}))

	// when: the subscription is expanded after its customer was soft-deleted
	view, err := service.GetView(ctx, models.NamespacedID{ID: sub.ID, Namespace: sub.Namespace})

	// then: it still resolves, with the (soft-deleted) customer attached
	require.NoError(t, err)
	require.Equal(t, cust.ID, view.Customer.ID)
	require.NotEmpty(t, view.Phases)
}

// createSubscriptionFromPlan creates a subscription for the given customer from the
// provided plan, failing the test on error.
func createSubscriptionFromPlan(t *testing.T, ctx context.Context, service subscription.Service, plan subscription.Plan, customerID string, activeFrom time.Time) subscription.Subscription {
	t.Helper()

	spec, err := subscription.NewSpecFromPlan(plan, subscription.CreateSubscriptionCustomerInput{
		CustomerId:      customerID,
		InvoiceCurrency: "USD",
		ActiveFrom:      activeFrom,
		BillingAnchor:   activeFrom,
		Name:            "Test Subscription",
		Annotations:     models.Annotations{},
	})
	require.NoError(t, err)

	sub, err := service.Create(ctx, subscriptiontestutils.ExampleNamespace, spec)
	require.NoError(t, err)

	return sub
}
