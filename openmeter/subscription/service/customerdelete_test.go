package service_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

// This harness registers no customer request validators, so the rejection comes
// from the check inside the customer delete transaction.
func TestCustomerDeleteRejectedForScheduledSubscription(t *testing.T) {
	ctx := t.Context()

	currentTime := testutils.GetRFC3339Time(t, "2021-01-01T00:00:11Z")
	clock.FreezeTime(currentTime)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)

	_ = deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	taxCode := deps.TaxCodeEnv.CreateTaxCode(t, subscriptiontestutils.ExampleNamespace, subscriptiontestutils.GetExampleTaxCodeInput())
	plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.GetExamplePlanInput(t, subscriptiontestutils.ExampleTaxConfig(taxCode)))

	// given: a customer whose only subscription starts tomorrow
	cust := deps.CustomerAdapter.CreateExampleCustomer(t)
	customerID := customer.CustomerID{Namespace: subscriptiontestutils.ExampleNamespace, ID: cust.ID}

	sub := createSubscriptionFromPlan(t, ctx, deps.SubscriptionService, plan, cust.ID, currentTime.Add(24*time.Hour))

	// when: the customer is deleted before the subscription starts
	err := deps.CustomerService.DeleteCustomer(ctx, customerID)

	// then: the delete is rejected and names the subscription
	require.True(t, models.IsGenericPreConditionFailedError(err), "expected precondition failed, got %v", err)

	issues, err := models.AsValidationIssues(err)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	require.Equal(t, customer.ErrCodeDeletingCustomerWithActiveSubscriptions, issues[0].Code())
	require.Equal(t, []string{sub.ID}, issues[0].Attributes()["subscriptions"])

	// when: the scheduled subscription is deleted first
	require.NoError(t, deps.SubscriptionService.Delete(ctx, sub.NamespacedID))

	// then: the customer can be deleted
	require.NoError(t, deps.CustomerService.DeleteCustomer(ctx, customerID))
}
