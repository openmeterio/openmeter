package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

// TestV3GetBillingInvoice exercises GET /api/v3/openmeter/billing/invoices/{invoiceId}.
//
// Flow:
//   - Create a customer (v3)
//   - Create and publish a plan with a flat rate card (v3)
//   - Create a subscription for the customer (v3)
//   - List the customer's invoices via the v1 SDK to obtain an invoice ID
//   - GET the invoice via the v3 endpoint and assert the response shape
//   - GET with an unknown ID → 404
func TestV3GetBillingInvoice(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	var (
		customerID  string
		customerKey string
		planID      string
		invoiceID   string
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("inv_customer")
		currency := apiv3.CurrencyCode("USD")

		status, customer, problem := c.CreateCustomer(apiv3.CreateCustomerRequest{
			Key:      key,
			Name:     "Invoice Test Customer",
			Currency: &currency,
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, customer)
		require.NotEmpty(t, customer.Id)

		customerID = customer.Id
		customerKey = key
	})

	t.Run("Should create and publish a plan with a flat rate card", func(t *testing.T) {
		planBody := validPlanRequest("inv_plan")

		status, plan, problem := c.CreatePlan(planBody)
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, plan)

		planID = plan.Id

		status, plan, problem = c.PublishPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plan)
		assert.Equal(t, apiv3.BillingPlanStatusActive, plan.Status)
	})

	t.Run("Should create a subscription for the customer", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")
		require.NotEmpty(t, planID, "depends on plan creation")

		status, sub, problem := c.CreateSubscription(apiv3.BillingSubscriptionCreate{
			Customer: struct {
				Id  *apiv3.ULID                `json:"id,omitempty"`
				Key *apiv3.ExternalResourceKey `json:"key,omitempty"`
			}{
				Id: lo.ToPtr(customerID),
			},
			Plan: struct {
				Id      *apiv3.ULID        `json:"id,omitempty"`
				Key     *apiv3.ResourceKey `json:"key,omitempty"`
				Version *int               `json:"version,omitempty"`
			}{
				Id: lo.ToPtr(planID),
			},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, sub)
	})

	t.Run("Should list invoices and find one for the customer via v1 SDK", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		customers := api.InvoiceListParamsCustomers{customerID}
		listResp, err := v1.ListInvoicesWithResponse(t.Context(), &api.ListInvoicesParams{
			Customers: &customers,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, listResp.StatusCode(), "list invoices: %s", string(listResp.Body))
		require.NotNil(t, listResp.JSON200)
		require.NotEmpty(t, listResp.JSON200.Items, "expected at least one invoice for customer %s (key: %s)", customerID, customerKey)

		invoiceID = listResp.JSON200.Items[0].Id
		require.NotEmpty(t, invoiceID)
	})

	t.Run("Should return the invoice via v3 GET", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice listing")

		status, inv, problem := c.GetBillingInvoice(invoiceID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, inv)

		// The union must decode as a standard invoice.
		stdInv, err := inv.AsBillingStandardInvoice()
		require.NoError(t, err, "expected invoice to decode as BillingStandardInvoice")

		assert.Equal(t, invoiceID, stdInv.Id)
		assert.Equal(t, apiv3.BillingStandardInvoiceTypeStandard, stdInv.Type)
		assert.Equal(t, apiv3.CurrencyCode("USD"), stdInv.Currency)
		assert.NotEmpty(t, stdInv.Status)
		assert.NotEmpty(t, stdInv.CreatedAt)
	})

	t.Run("Should return 404 for an unknown invoice ID", func(t *testing.T) {
		status, inv, problem := c.GetBillingInvoice("01JAAAAAAAAAAAAAAAAAAAAAAA")
		assert.Equal(t, http.StatusNotFound, status)
		assert.Nil(t, inv)
		assert.NotNil(t, problem)
	})
}
