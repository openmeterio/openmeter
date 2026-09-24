package e2e

import (
	"net/http"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestV3SystemEventOutbox exercises HTTP -> transactional outbox -> Kafka ->
// billing worker -> HTTP. The test never calls a reconciliation endpoint.
func TestV3SystemEventOutbox(t *testing.T) {
	c := newV3Client(t)
	var customer *v3sdk.Customer
	var plan *v3sdk.Plan
	var subscription *v3sdk.BillingSubscription
	chargeName := uniqueKey("outbox_fee")

	runRequired(t, "create customer and published plan", func(t *testing.T) {
		// Given a unique customer and one monthly fee billed in arrears.
		var err error
		customer, err = c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      uniqueKey("outbox_customer"),
			Name:     "Outbox Customer",
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)

		request := validPlanRequest("outbox_plan")
		request.ProRatingEnabled = lo.ToPtr(false)
		request.Phases[0].RateCards[0].Name = chargeName
		request.Phases[0].RateCards[0].PaymentTerm = lo.ToPtr(v3sdk.PricePaymentTermInArrears)
		plan, err = c.Plans.Create(t.Context(), request)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		plan, err = c.Plans.Publish(t.Context(), plan.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)
	})

	runRequired(t, "create subscription over HTTP", func(t *testing.T) {
		// When the API commits the subscription and its system event.
		var err error
		subscription, err = c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer:       v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:           &v3sdk.SubscriptionChangePlan{ID: &plan.ID},
			SettlementMode: lo.ToPtr(v3sdk.SettlementModeCreditThenInvoice),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, subscription)
		require.Equal(t, v3sdk.SubscriptionStatusActive, subscription.Status)
		t.Logf("customer_id=%s plan_id=%s subscription_id=%s", customer.ID, plan.ID, subscription.ID)
	})

	runRequired(t, "billing worker reconciles the committed subscription", func(t *testing.T) {
		// Then the worker consumes the event and produces a charge visible through the API.
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			charges, err := c.Customers.Charges.List(t.Context(), customer.ID, v3sdk.ListCustomerChargesParams{
				Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
			})
			require.NoError(collect, err)
			require.NotNil(collect, charges)
			t.Logf("subscription_id=%s charges=%s", subscription.ID, formatLogJSON(charges.Data))
			require.Len(collect, charges.Data, 1)
			charge, err := charges.Data[0].AsChargeFlatFee()
			require.NoError(collect, err)
			require.Equal(collect, chargeName, charge.Name)
			require.Equal(collect, v3sdk.LifecycleControllerSystem, charge.LifecycleController)
			// An in-arrears fee waits in created until its future invoice date.
			require.Equal(collect, v3sdk.ChargeStatusCreated, charge.Status)
			require.Equal(collect, "10", charge.Price.Amount)
			require.Empty(collect, charge.ValidationIssues)
			require.NotNil(collect, charge.Subscription)
			ref, err := charge.Subscription.AsSubscriptionReference()
			require.NoError(collect, err)
			require.Equal(collect, subscription.ID, ref.ID)
		}, time.Minute, time.Second)
	})
}
