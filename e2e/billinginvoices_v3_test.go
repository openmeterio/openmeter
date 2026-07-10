package e2e

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
)

// TestV3GetBillingInvoice exercises GET /api/v3/openmeter/billing/invoices/{invoiceId}.
//
// Flow:
//   - Create a customer (v3)
//   - Create a meter and feature for usage-based billing (v3)
//   - Create and publish a plan with flat and usage-based rate cards (v3)
//   - Create a subscription for the customer (v3)
//   - Create a future-dated pending line and wait for the subscription charge to produce a standard invoice
//   - List the customer's invoices via the v1 SDK; separate standard from gathering
//   - GET the standard invoice via the v3 endpoint and assert the response shape
//   - GET the gathering invoice via the v3 endpoint → 404 (gathering invoices are not exposed)
//   - GET with an unknown ID → 404
func TestV3GetBillingInvoice(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	var (
		customerID         string
		customerKey        string
		planID             string
		feature            *v3sdk.Feature
		invoiceID          string // standard invoice ID
		gatheringInvoiceID string // gathering invoice ID
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("inv_customer")

		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      key,
			Name:     "Invoice Test Customer",
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)
		require.NotEmpty(t, customer.ID)

		customerID = customer.ID
		customerKey = key
	})

	t.Run("Should create a meter and a feature for usage-based billing", func(t *testing.T) {
		meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:         uniqueKey("inv_meter"),
			Name:        "Invoice Test Meter",
			Aggregation: v3sdk.MeterAggregationCount,
			EventType:   uniqueKey("inv_event"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, meter)

		f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:   uniqueKey("inv_feature"),
			Name:  "Invoice Test Feature",
			Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, f)

		feature = f
	})

	t.Run("Should create and publish a plan with flat and unit rate cards", func(t *testing.T) {
		require.NotNil(t, feature, "depends on feature creation")

		plan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            uniqueKey("inv_plan"),
			Name:           "Invoice Test Plan",
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:  "phase_1",
				Name: "Test Phase",
				RateCards: []v3sdk.RateCardInput{
					validFlatRateCard("inv_fee"),
					validUnitRateCard(*feature),
				},
			}},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)

		planID = plan.ID

		plan, err = c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plan)
		assert.Equal(t, v3sdk.PlanStatusActive, plan.Status)
	})

	t.Run("Should create a subscription for the customer", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")
		require.NotEmpty(t, planID, "depends on plan creation")

		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer:       v3sdk.SubscriptionChangeCustomer{ID: lo.ToPtr(customerID)},
			Plan:           v3sdk.SubscriptionChangePlan{ID: lo.ToPtr(planID)},
			SettlementMode: lo.ToPtr(v3sdk.SettlementModeCreditThenInvoice),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, sub)
	})

	t.Run("Should create a single gathering invoice", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      api.Numeric("10.00"),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		invoiceAt := now.Add(time.Hour)
		lineResp, err := v1.CreatePendingInvoiceLineWithResponse(t.Context(), customerID, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{
				{
					Name:      uniqueKey("inv_gathering_line_name"),
					InvoiceAt: invoiceAt,
					Period: api.Period{
						From: now.Add(-24 * time.Hour),
						To:   invoiceAt,
					},
					Price: &price,
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, lineResp.StatusCode(), "line: %s", string(lineResp.Body))
		require.NotNil(t, lineResp.JSON201)

		gatheringInvoiceID = (*lineResp.JSON201).Invoice.Id
		require.NotEmpty(t, gatheringInvoiceID)
	})

	t.Run("Should create a single standard invoice", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		customers := api.InvoiceListParamsCustomers{customerID}
		expand := api.InvoiceListParamsExpand{api.InvoiceExpandLines}
		pollAttempt := 0
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			pollAttempt++

			listResp, err := v1.ListInvoicesWithResponse(t.Context(), &api.ListInvoicesParams{
				Customers: &customers,
				Expand:    &expand,
				PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
			})
			require.NoError(collect, err)
			require.Equal(collect, http.StatusOK, listResp.StatusCode(), "list invoices: %s", string(listResp.Body))
			require.NotNil(collect, listResp.JSON200)

			t.Logf("standard invoice poll %02d: invoices=%s", pollAttempt, formatInvoicesForLog(listResp.JSON200.Items))

			standardInvoiceIdx := slices.IndexFunc(listResp.JSON200.Items, func(inv api.Invoice) bool {
				return inv.Status != api.InvoiceStatusGathering
			})
			require.NotEqual(collect, -1, standardInvoiceIdx, "expected subscription charge to produce a standard invoice")

			invoiceID = listResp.JSON200.Items[standardInvoiceIdx].Id
		}, 2*time.Minute, time.Second)
		t.Logf("created standard invoice: customer_id=%s invoice_id=%s", customerID, invoiceID)
		require.NotEmpty(t, invoiceID)
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

		standardInvoiceIdx := slices.IndexFunc(listResp.JSON200.Items, func(inv api.Invoice) bool {
			return inv.Status != api.InvoiceStatusGathering
		})
		require.NotEqual(t, -1, standardInvoiceIdx, "expected at least one non-gathering invoice in the list")
	})

	t.Run("Should return the invoice via v3 GET", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice advance step")

		inv, err := c.Invoices.Get(t.Context(), invoiceID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, inv)

		// The union must decode as a standard invoice.
		stdInv, err := inv.AsInvoiceStandard()
		require.NoError(t, err, "expected invoice to decode as BillingInvoiceStandard")

		assert.Equal(t, invoiceID, stdInv.ID)
		assert.Equal(t, v3sdk.InvoiceTypeStandard, stdInv.Type)
		assert.Equal(t, "USD", stdInv.Currency)
		assert.NotEmpty(t, stdInv.Status)
		assert.NotEmpty(t, stdInv.CreatedAt)
	})

	t.Run("Should return 404 for the gathering invoice", func(t *testing.T) {
		if gatheringInvoiceID == "" {
			t.Skip("no gathering invoice found in the list; skipping")
		}

		inv, err := c.Invoices.Get(t.Context(), gatheringInvoiceID)
		problem := requireProblem(t, err, http.StatusNotFound)
		assert.Nil(t, inv)
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 for an unknown invoice ID", func(t *testing.T) {
		inv, err := c.Invoices.Get(t.Context(), "01JAAAAAAAAAAAAAAAAAAAAAAA")
		problem := requireProblem(t, err, http.StatusNotFound)
		assert.Nil(t, inv)
		assert.NotNil(t, problem)
	})
}

func formatInvoicesForLog(invoices []api.Invoice) string {
	return formatLogJSON(invoices)
}

func formatLogJSON(v any) string {
	out, err := json.Marshal(v)
	if err != nil {
		return `{"marshal_error":` + strconv.Quote(err.Error()) + `}`
	}

	return string(out)
}

// TestV3ListBillingInvoices exercises GET /api/v3/openmeter/billing/invoices.
//
// Flow:
//   - Create a customer (v3)
//   - Create a meter, feature, plan, and subscription (v3)
//   - Create a charge-backed pending line and wait for it to advance into a standard invoice
//   - List invoices: assert the standard invoice appears and gathering is excluded
//   - List invoices filtered by customer_id: assert only the customer's invoices are returned
//   - List invoices filtered by status: assert only matching statuses are returned
//   - List invoices with sort=issued_at: assert 200 response
//   - List invoices for an unknown customer: assert empty result
//   - List invoices with an unsupported filter operator: assert 400
func TestV3ListBillingInvoices(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	var (
		customerID string
		planID     string
		feature    *v3sdk.Feature
		invoiceID  string
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("listinv_customer")

		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)
		customerID = customer.ID
	})

	t.Run("Should create meter, feature, plan, and subscription", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:         uniqueKey("listinv_meter"),
			Name:        gofakeit.ProductName(),
			Aggregation: v3sdk.MeterAggregationCount,
			EventType:   uniqueKey("listinv_event"),
		})
		c.requireStatus(http.StatusCreated, err)

		f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:   uniqueKey("listinv_feature"),
			Name:  gofakeit.ProductName(),
			Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		feature = f

		plan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            uniqueKey("listinv_plan"),
			Name:           gofakeit.ProductName(),
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       uniqueKey("inv_phase_1"),
				Name:      uniqueKey("Test Phase"),
				RateCards: []v3sdk.RateCardInput{validUnitRateCard(*feature)},
			}},
		})
		c.requireStatus(http.StatusCreated, err)
		planID = plan.ID

		plan, err = c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, v3sdk.PlanStatusActive, plan.Status)

		_, err = c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: lo.ToPtr(customerID)},
			Plan:     v3sdk.SubscriptionChangePlan{ID: lo.ToPtr(planID)},
		})
		c.requireStatus(http.StatusCreated, err)
	})

	t.Run("Should create a standard invoice and wait for it to advance", func(t *testing.T) {
		require.NotEmpty(t, customerID)
		require.NotNil(t, feature)

		now := time.Now().UTC()
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      api.Numeric("10.00"),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		lineResp, err := v1.CreatePendingInvoiceLineWithResponse(t.Context(), customerID, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{{
				Name:      uniqueKey("listinv_line"),
				InvoiceAt: now.Add(-10 * time.Hour),
				Period: api.Period{
					From: now.Add(-24 * time.Hour),
					To:   now.Add(-2 * time.Hour),
				},
				Price: &price,
			}},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, lineResp.StatusCode())

		ctx := t.Context()
		customers := api.InvoiceListParamsCustomers{customerID}
		assert.EventuallyWithT(t, func(co *assert.CollectT) {
			listResp, err := v1.ListInvoicesWithResponse(ctx, &api.ListInvoicesParams{
				Customers: &customers,
				PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
			})
			require.NoError(co, err)
			require.Equal(co, http.StatusOK, listResp.StatusCode())

			idx := slices.IndexFunc(listResp.JSON200.Items, func(inv api.Invoice) bool {
				return inv.Status != api.InvoiceStatusGathering
			})
			require.NotEqual(co, -1, idx, "charges have not advanced a pending line into a standard invoice yet")
			invoiceID = listResp.JSON200.Items[idx].Id
		}, time.Minute, time.Second)
		require.NotEmpty(t, invoiceID)
	})

	t.Run("Should list invoices and exclude gathering", func(t *testing.T) {
		require.NotEmpty(t, customerID)

		resp, err := c.Invoices.List(t.Context(), v3sdk.InvoiceListParams{
			Filter: &v3sdk.InvoiceFilter{
				CustomerID: &v3sdk.StringExactFilter{Eq: &customerID},
			},
			Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)
		require.NotEmpty(t, resp.Data, "expected at least one invoice for customer %s", customerID)

		for _, inv := range resp.Data {
			stdInv, err := inv.AsInvoiceStandard()
			require.NoError(t, err, "all returned invoices must decode as standard (no gathering)")
			assert.NotEqual(t, "gathering", string(stdInv.Status), "gathering invoice leaked into list response")
		}
	})

	t.Run("Should filter by customer_id and return only that customer's invoices", func(t *testing.T) {
		require.NotEmpty(t, customerID)
		require.NotEmpty(t, invoiceID)

		resp, err := c.Invoices.List(t.Context(), v3sdk.InvoiceListParams{
			Filter: &v3sdk.InvoiceFilter{
				CustomerID: &v3sdk.StringExactFilter{Eq: &customerID},
			},
			Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)

		found := slices.ContainsFunc(resp.Data, func(inv v3sdk.Invoice) bool {
			stdInv, err := inv.AsInvoiceStandard()
			if err != nil {
				return false
			}
			return stdInv.ID == invoiceID
		})
		assert.True(t, found, "invoice %s must appear when filtered by customer %s", invoiceID, customerID)
	})

	t.Run("Should filter by status oeq and return only matching invoices", func(t *testing.T) {
		require.NotEmpty(t, customerID)

		resp, err := c.Invoices.List(t.Context(), v3sdk.InvoiceListParams{
			Filter: &v3sdk.InvoiceFilter{
				CustomerID: &v3sdk.StringExactFilter{Eq: &customerID},
				Status:     &v3sdk.StringExactFilter{Oeq: []string{"draft", "issued"}},
			},
			Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)

		for _, inv := range resp.Data {
			stdInv, err := inv.AsInvoiceStandard()
			require.NoError(t, err)
			s := string(stdInv.Status)
			assert.True(t, s == "draft" || s == "issued",
				"expected status draft or issued, got %s", s)
		}
	})

	t.Run("Should accept sort=issued_at and return 200", func(t *testing.T) {
		require.NotEmpty(t, customerID)

		resp, err := c.Invoices.List(t.Context(), v3sdk.InvoiceListParams{
			Filter: &v3sdk.InvoiceFilter{
				CustomerID: &v3sdk.StringExactFilter{Eq: &customerID},
			},
			Sort: &v3sdk.Sort{By: "issued_at"},
			Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)
	})

	t.Run("Should accept sort=service_period_start desc and return 200", func(t *testing.T) {
		require.NotEmpty(t, customerID)

		resp, err := c.Invoices.List(t.Context(), v3sdk.InvoiceListParams{
			Filter: &v3sdk.InvoiceFilter{
				CustomerID: &v3sdk.StringExactFilter{Eq: &customerID},
			},
			Sort: &v3sdk.Sort{By: "service_period_start", Order: v3sdk.SortOrderDesc},
			Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)
	})

	t.Run("Should return empty result for an unknown customer ID", func(t *testing.T) {
		customerID := "01JAAAAAAAAAAAAAAAAAAAAAAA"
		resp, err := c.Invoices.List(t.Context(), v3sdk.InvoiceListParams{
			Filter: &v3sdk.InvoiceFilter{
				CustomerID: &v3sdk.StringExactFilter{Eq: &customerID},
			},
			Page: &v3sdk.PageParams{Size: lo.ToPtr(20)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Data, "unknown customer should yield no invoices")
	})

	t.Run("Should return 400 for an unsupported filter operator (neq on status)", func(t *testing.T) {
		_, err := c.Invoices.List(t.Context(), v3sdk.InvoiceListParams{
			Filter: &v3sdk.InvoiceFilter{
				Status: &v3sdk.StringExactFilter{Neq: lo.ToPtr("draft")},
			},
		})
		problem := requireProblem(t, err, http.StatusBadRequest)
		assert.NotNil(t, problem)
	})

	t.Run("Should return page metadata", func(t *testing.T) {
		require.NotEmpty(t, customerID)

		resp, err := c.Invoices.List(t.Context(), v3sdk.InvoiceListParams{
			Filter: &v3sdk.InvoiceFilter{
				CustomerID: &v3sdk.StringExactFilter{Eq: &customerID},
			},
			Page: &v3sdk.PageParams{
				Size:   lo.ToPtr(1),
				Number: lo.ToPtr(1),
			},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, resp)
		assert.Equal(t, 1, resp.Meta.Page.Size)
		assert.Equal(t, 1, resp.Meta.Page.Number)
		assert.GreaterOrEqual(t, resp.Meta.Page.Total, 1)
	})
}

// TestV3UpdateBillingInvoice exercises PUT /api/v3/openmeter/billing/invoices/{invoiceId}.
//
// Flow:
//   - Create a customer (v3)
//   - Pin the customer to a manual-approval billing profile (auto_advance=false,
//     send_invoice) so the standard invoice it produces stays in draft status;
//     only draft invoices can be updated via v3, and the default profile's
//     auto-collect settings otherwise advance the invoice to paid too quickly
//     for the update test to observe a draft invoice.
//   - Create a meter, feature, plan, and subscription (v3)
//   - Create a standard invoice (v1)
//   - Create a gathering invoice (v1) for the gathering-invoice-update case
//   - Update the standard invoice via v3 PUT and assert the response shape
//   - Update the standard invoice with an invalid type and assert 400
//   - Update a gathering invoice via v3 PUT and assert 404
//   - Update an unknown invoice ID via v3 PUT and assert 404
//   - Update a standard invoice with invalid data (e.g., negative amount) and assert 400
//   - Update a standard invoice with a valid change and assert the change is reflected in the response
func TestV3UpdateBillingInvoice(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	var (
		customerID         string
		planID             string
		feature            *v3sdk.Feature
		invoice            api.Invoice
		gatheringInvoiceID string
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("listinv_customer")

		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)
		customerID = customer.ID
	})

	t.Run("Should pin the customer to a manual-approval billing profile", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		profile := createNewBillingProfileFromDefault(t, c, uniqueKey("update_invoice"), func(profile *v3sdk.CreateBillingProfileRequest) {
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &v3sdk.WorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)

			sendInvoice := lo.Must(v3sdk.WorkflowPaymentSettingsFromWorkflowPaymentSendInvoiceSettings(v3sdk.WorkflowPaymentSendInvoiceSettings{
				CollectionMethod: v3sdk.CollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})

		_, err := c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: &v3sdk.ProfileReference{ID: profile.ID},
		})
		c.requireStatus(http.StatusOK, err)
	})

	t.Run("Should create a single gathering invoice", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      api.Numeric("10.00"),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		invoiceAt := now.Add(time.Hour)
		lineResp, err := v1.CreatePendingInvoiceLineWithResponse(t.Context(), customerID, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{
				{
					Name:      uniqueKey("update_inv_gathering_line"),
					InvoiceAt: invoiceAt,
					Period: api.Period{
						From: now.Add(-24 * time.Hour),
						To:   invoiceAt,
					},
					Price: &price,
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, lineResp.StatusCode(), "line: %s", string(lineResp.Body))
		require.NotNil(t, lineResp.JSON201)

		gatheringInvoiceID = (*lineResp.JSON201).Invoice.Id
		require.NotEmpty(t, gatheringInvoiceID)
	})

	t.Run("Should create meter, feature, plan, and subscription", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:         uniqueKey("listinv_meter"),
			Name:        gofakeit.ProductName(),
			Aggregation: v3sdk.MeterAggregationCount,
			EventType:   uniqueKey("listinv_event"),
		})
		c.requireStatus(http.StatusCreated, err)

		feature, err = c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:   uniqueKey("listinv_feature"),
			Name:  gofakeit.ProductName(),
			Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		})
		c.requireStatus(http.StatusCreated, err)

		plan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            uniqueKey("listinv_plan"),
			Name:           gofakeit.ProductName(),
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       uniqueKey("inv_phase_1"),
				Name:      uniqueKey("Test Phase"),
				RateCards: []v3sdk.RateCardInput{validUnitRateCard(*feature)},
			}},
		})
		c.requireStatus(http.StatusCreated, err)
		planID = plan.ID

		plan, err = c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, v3sdk.PlanStatusActive, plan.Status)

		_, err = c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: lo.ToPtr(customerID)},
			Plan:     v3sdk.SubscriptionChangePlan{ID: lo.ToPtr(planID)},
		})
		c.requireStatus(http.StatusCreated, err)
	})

	t.Run("Should create a standard invoice and wait for it to advance", func(t *testing.T) {
		require.NotEmpty(t, customerID)
		require.NotNil(t, feature)

		now := time.Now().UTC()
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      api.Numeric("10.00"),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		lineResp, err := v1.CreatePendingInvoiceLineWithResponse(t.Context(), customerID, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{{
				Name:      uniqueKey("listinv_line"),
				InvoiceAt: now.Add(-10 * time.Hour),
				Period: api.Period{
					From: now.Add(-24 * time.Hour),
					To:   now.Add(-2 * time.Hour),
				},
				Price: &price,
			}},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, lineResp.StatusCode())

		ctx := t.Context()
		customers := api.InvoiceListParamsCustomers{customerID}
		assert.EventuallyWithT(t, func(co *assert.CollectT) {
			listResp, err := v1.ListInvoicesWithResponse(ctx, &api.ListInvoicesParams{
				Customers: &customers,
				PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
			})
			require.NoError(co, err)
			require.Equal(co, http.StatusOK, listResp.StatusCode())

			idx := slices.IndexFunc(listResp.JSON200.Items, func(inv api.Invoice) bool {
				return inv.Status != api.InvoiceStatusGathering
			})
			require.NotEqual(co, -1, idx, "charges have not advanced a pending line into a standard invoice yet")
			invoice = listResp.JSON200.Items[idx]
		}, time.Minute, time.Second)
		require.NotEmpty(t, invoice)
	})

	t.Run("Should update the standard invoice via v3 PUT", func(t *testing.T) {
		require.NotEmpty(t, invoice)

		newDescription := gofakeit.LoremIpsumSentence(5)

		updateReq := lo.Must(v3sdk.UpdateInvoiceRequestFromUpdateInvoiceStandardRequest(v3sdk.UpdateInvoiceStandardRequest{
			Type:        v3sdk.InvoiceTypeStandard,
			Description: lo.ToPtr(newDescription),
			Customer: v3sdk.InvoiceCustomer{
				ID:   *invoice.Customer.Id,
				Name: *invoice.Customer.Name,
			},
			Supplier: updateSupplierFromInvoiceParty(invoice.Supplier),
			Workflow: v3sdk.UpdateInvoiceWorkflowSettings{},
			Lines:    nil,
		}))

		updatedInv, err := c.Invoices.Update(t.Context(), invoice.Id, updateReq)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updatedInv)

		stdInv, err := updatedInv.AsInvoiceStandard()
		require.NoError(t, err)
		assert.Equal(t, invoice.Id, stdInv.ID)
		assert.Equal(t, newDescription, lo.FromPtr(stdInv.Description))
	})

	t.Run("Should reject an update with an unsupported invoice type", func(t *testing.T) {
		require.NotEmpty(t, invoice)

		body := map[string]any{
			"type": "unsupported_invoice_type",
			"customer": map[string]any{
				"id":   *invoice.Customer.Id,
				"name": *invoice.Customer.Name,
			},
			"supplier": map[string]any{},
			"workflow": map[string]any{"workflow": map[string]any{}},
		}

		raw, err := json.Marshal(body)
		require.NoError(t, err)

		var updateReq v3sdk.UpdateInvoiceRequest
		require.NoError(t, json.Unmarshal(raw, &updateReq))

		_, err = c.Invoices.Update(t.Context(), invoice.Id, updateReq)
		problem := requireProblem(t, err, http.StatusBadRequest)
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 when updating a gathering invoice via v3 PUT", func(t *testing.T) {
		require.NotEmpty(t, gatheringInvoiceID, "depends on gathering invoice creation")

		updateReq := lo.Must(v3sdk.UpdateInvoiceRequestFromUpdateInvoiceStandardRequest(v3sdk.UpdateInvoiceStandardRequest{
			Type: v3sdk.InvoiceTypeStandard,
			Customer: v3sdk.InvoiceCustomer{
				ID:   customerID,
				Name: "placeholder",
			},
			Supplier: v3sdk.Supplier{},
			Workflow: v3sdk.UpdateInvoiceWorkflowSettings{},
		}))

		updatedInv, err := c.Invoices.Update(t.Context(), gatheringInvoiceID, updateReq)
		problem := requireProblem(t, err, http.StatusNotFound)
		assert.Nil(t, updatedInv)
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 for an unknown invoice ID via v3 PUT", func(t *testing.T) {
		updateReq := lo.Must(v3sdk.UpdateInvoiceRequestFromUpdateInvoiceStandardRequest(v3sdk.UpdateInvoiceStandardRequest{
			Type: v3sdk.InvoiceTypeStandard,
			Customer: v3sdk.InvoiceCustomer{
				ID:   customerID,
				Name: "placeholder",
			},
			Supplier: v3sdk.Supplier{},
			Workflow: v3sdk.UpdateInvoiceWorkflowSettings{},
		}))

		updatedInv, err := c.Invoices.Update(t.Context(), "01JAAAAAAAAAAAAAAAAAAAAAAA", updateReq)
		problem := requireProblem(t, err, http.StatusNotFound)
		assert.Nil(t, updatedInv)
		assert.NotNil(t, problem)
	})

	t.Run("Should reject a line update with a negative amount", func(t *testing.T) {
		require.NotEmpty(t, invoice)

		currentInv, err := c.Invoices.Get(t.Context(), invoice.Id)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, currentInv)
		stdInv, err := currentInv.AsInvoiceStandard()
		require.NoError(t, err)

		lines := existingUpdateLines(t, *stdInv)

		negativeLine := lo.Must(v3sdk.UpdateInvoiceLineFromUpdateInvoiceStandardLine(newFlatUpdateLine(uniqueKey("negative_amount_line"), stdInv.ServicePeriod, "-5")))
		lines = append(lines, negativeLine)

		updateReq := lo.Must(v3sdk.UpdateInvoiceRequestFromUpdateInvoiceStandardRequest(v3sdk.UpdateInvoiceStandardRequest{
			Type: v3sdk.InvoiceTypeStandard,
			Customer: v3sdk.InvoiceCustomer{
				ID:   *invoice.Customer.Id,
				Name: *invoice.Customer.Name,
			},
			Supplier: updateSupplierFromInvoiceParty(invoice.Supplier),
			Workflow: v3sdk.UpdateInvoiceWorkflowSettings{},
			Lines:    &lines,
		}))

		updatedInv, err := c.Invoices.Update(t.Context(), invoice.Id, updateReq)
		problem := requireProblem(t, err, http.StatusBadRequest)
		assert.Nil(t, updatedInv)
		assertProblemDetail(t, problem, "must not be negative")
	})

	t.Run("Should apply a valid line addition and reflect it in the response", func(t *testing.T) {
		require.NotEmpty(t, invoice)

		currentInv, err := c.Invoices.Get(t.Context(), invoice.Id)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, currentInv)
		stdInv, err := currentInv.AsInvoiceStandard()
		require.NoError(t, err)

		lines := existingUpdateLines(t, *stdInv)

		newLineName := uniqueKey("valid_new_line")
		newLine := lo.Must(v3sdk.UpdateInvoiceLineFromUpdateInvoiceStandardLine(newFlatUpdateLine(newLineName, stdInv.ServicePeriod, "15")))
		lines = append(lines, newLine)

		updateReq := lo.Must(v3sdk.UpdateInvoiceRequestFromUpdateInvoiceStandardRequest(v3sdk.UpdateInvoiceStandardRequest{
			Type: v3sdk.InvoiceTypeStandard,
			Customer: v3sdk.InvoiceCustomer{
				ID:   *invoice.Customer.Id,
				Name: *invoice.Customer.Name,
			},
			Supplier: updateSupplierFromInvoiceParty(invoice.Supplier),
			Workflow: v3sdk.UpdateInvoiceWorkflowSettings{},
			Lines:    &lines,
		}))

		updatedInv, err := c.Invoices.Update(t.Context(), invoice.Id, updateReq)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, updatedInv)

		updatedStdInv, err := updatedInv.AsInvoiceStandard()
		require.NoError(t, err)
		require.NotNil(t, updatedStdInv.Lines)

		found := false
		for _, l := range updatedStdInv.Lines {
			stdLine, err := l.AsInvoiceStandardLine()
			require.NoError(t, err)
			if stdLine.Name == newLineName {
				found = true
				p, err := stdLine.RateCard.Price.AsPriceFlat()
				require.NoError(t, err)
				assert.Equal(t, "15", p.Amount)
			}
		}
		assert.True(t, found, "expected the newly added line %q to appear in the updated invoice", newLineName)
	})
}

// updateSupplierFromInvoiceParty maps a v1 BillingParty (the invoice's supplier
// snapshot) into a Supplier that round-trips its current name and
// address. Update requests replace the full supplier object: an omitted name
// or address is not "leave unchanged", it's reset to empty, which then fails
// SupplierContact.Validate() (name and country are required for invoicing).
func updateSupplierFromInvoiceParty(party api.BillingParty) v3sdk.Supplier {
	supplier := v3sdk.Supplier{
		ID:   party.Id,
		Name: party.Name,
	}

	if party.Addresses != nil && len(*party.Addresses) > 0 {
		addr := (*party.Addresses)[0]
		supplier.Addresses = &v3sdk.PartyAddresses{
			BillingAddress: v3sdk.Address{
				City:        addr.City,
				Country:     addr.Country,
				Line1:       addr.Line1,
				Line2:       addr.Line2,
				PhoneNumber: addr.PhoneNumber,
				PostalCode:  addr.PostalCode,
				State:       addr.State,
			},
		}
	}

	return supplier
}

// existingUpdateLines maps a standard invoice's current lines back into
// UpdateInvoiceLine values (preserving IDs and content), so a follow-up PUT
// can append or modify lines without deleting the ones it doesn't mention.
// Update requests replace the full `lines` array: any existing line whose ID
// is omitted from the request is deleted.
func existingUpdateLines(t *testing.T, stdInv v3sdk.InvoiceStandard) []v3sdk.UpdateInvoiceLine {
	t.Helper()

	if stdInv.Lines == nil {
		return nil
	}

	lines := make([]v3sdk.UpdateInvoiceLine, 0, len(stdInv.Lines))
	for _, l := range stdInv.Lines {
		stdLine, err := l.AsInvoiceStandardLine()
		require.NoError(t, err)
		lines = append(lines, updateLineFromStandardLine(t, *stdLine))
	}

	return lines
}

// updateLineFromStandardLine maps a read-side standard line back into an
// UpdateInvoiceLine, preserving its ID and rate card so that resubmitting it
// unchanged is a no-op rather than an unintended edit or deletion.
func updateLineFromStandardLine(t *testing.T, line v3sdk.InvoiceStandardLine) v3sdk.UpdateInvoiceLine {
	t.Helper()

	// Rebuild the price explicitly: the server's update schema accepts only the
	// flat/unit subset, and the rebuild keeps the unsupported-price-type guard.
	var price v3sdk.Price
	switch line.RateCard.Price.Type {
	case string(v3sdk.PriceTypeFlat):
		p, err := line.RateCard.Price.AsPriceFlat()
		require.NoError(t, err)
		price = lo.Must(v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{
			Amount: p.Amount,
		}))
	case string(v3sdk.PriceTypeUnit):
		p, err := line.RateCard.Price.AsPriceUnit()
		require.NoError(t, err)
		price = lo.Must(v3sdk.PriceFromPriceUnit(v3sdk.PriceUnit{
			Amount: p.Amount,
		}))
	default:
		t.Fatalf("unsupported rate card price type in test fixture: %s", line.RateCard.Price.Type)
	}

	return lo.Must(v3sdk.UpdateInvoiceLineFromUpdateInvoiceStandardLine(v3sdk.UpdateInvoiceStandardLine{
		ID:            line.ID,
		Name:          line.Name,
		Description:   line.Description,
		Labels:        lo.EmptyableToPtr(line.Labels),
		Type:          v3sdk.InvoiceLineTypeStandardLine,
		ServicePeriod: line.ServicePeriod,
		RateCard: v3sdk.InvoiceLineRateCard{
			FeatureKey: line.RateCard.FeatureKey,
			Discounts:  line.RateCard.Discounts,
			TaxConfig:  line.RateCard.TaxConfig,
			Price:      price,
		},
	}))
}

// newFlatUpdateLine builds a brand-new (no ID) flat-priced UpdateInvoiceStandardLine
// for tests that append a manual line to an existing invoice.
func newFlatUpdateLine(name string, period v3sdk.ClosedPeriod, amount string) v3sdk.UpdateInvoiceStandardLine {
	return v3sdk.UpdateInvoiceStandardLine{
		Name:          name,
		Type:          v3sdk.InvoiceLineTypeStandardLine,
		ServicePeriod: period,
		RateCard: v3sdk.InvoiceLineRateCard{
			Price: lo.Must(v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{Amount: amount})),
		},
	}
}

// TestV3DeleteBillingInvoice exercises DELETE /api/v3/openmeter/billing/invoices/{invoiceId}.
// Flow:
//   - Create a customer (v3)
//   - Create a meter, feature, plan, and subscription (v3)
//   - Create a standard invoice (v1)
//   - Create a gathering invoice (v1)
//   - Delete the standard invoice via v3 DELETE and assert 204
//   - Attempt to delete the same invoice again and assert 400
//   - Attempt to delete a gathering invoice via v3 DELETE and assert 404
//   - Attempt to delete an unknown invoice ID via v3 DELETE and assert 404
func TestV3DeleteBillingInvoice(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	var (
		customerID         string
		planID             string
		feature            *v3sdk.Feature
		invoiceID          string // standard invoice ID
		gatheringInvoiceID string // gathering invoice ID
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("deleteinv_customer")

		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)
		customerID = customer.ID
	})

	t.Run("Should pin the customer to a manual-approval billing profile", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		// Only draft invoices can be deleted; the default profile's auto-collect
		// settings otherwise advance the invoice past draft too quickly for this
		// test to observe (and delete) a draft invoice.
		profile := createNewBillingProfileFromDefault(t, c, uniqueKey("delete_invoice"), func(profile *v3sdk.CreateBillingProfileRequest) {
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &v3sdk.WorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)

			sendInvoice := lo.Must(v3sdk.WorkflowPaymentSettingsFromWorkflowPaymentSendInvoiceSettings(v3sdk.WorkflowPaymentSendInvoiceSettings{
				CollectionMethod: v3sdk.CollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})

		_, err := c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: &v3sdk.ProfileReference{ID: profile.ID},
		})
		c.requireStatus(http.StatusOK, err)
	})

	t.Run("Should create a single gathering invoice", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      api.Numeric("10.00"),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		invoiceAt := now.Add(time.Hour)
		lineResp, err := v1.CreatePendingInvoiceLineWithResponse(t.Context(), customerID, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{
				{
					Name:      uniqueKey("delete_inv_gathering_line"),
					InvoiceAt: invoiceAt,
					Period: api.Period{
						From: now.Add(-24 * time.Hour),
						To:   invoiceAt,
					},
					Price: &price,
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, lineResp.StatusCode(), "line: %s", string(lineResp.Body))
		require.NotNil(t, lineResp.JSON201)

		gatheringInvoiceID = (*lineResp.JSON201).Invoice.Id
		require.NotEmpty(t, gatheringInvoiceID)
	})

	t.Run("Should create meter, feature, plan, and subscription", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:         uniqueKey("deleteinv_meter"),
			Name:        gofakeit.ProductName(),
			Aggregation: v3sdk.MeterAggregationCount,
			EventType:   uniqueKey("deleteinv_event"),
		})
		c.requireStatus(http.StatusCreated, err)

		f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:   uniqueKey("deleteinv_feature"),
			Name:  gofakeit.ProductName(),
			Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		feature = f

		plan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            uniqueKey("deleteinv_plan"),
			Name:           gofakeit.ProductName(),
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       uniqueKey("inv_phase_1"),
				Name:      uniqueKey("Test Phase"),
				RateCards: []v3sdk.RateCardInput{validUnitRateCard(*feature)},
			}},
		})
		c.requireStatus(http.StatusCreated, err)
		planID = plan.ID

		plan, err = c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, v3sdk.PlanStatusActive, plan.Status)

		_, err = c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: lo.ToPtr(customerID)},
			Plan:     v3sdk.SubscriptionChangePlan{ID: lo.ToPtr(planID)},
		})
		c.requireStatus(http.StatusCreated, err)
	})

	t.Run("Should create a standard invoice and wait for it to advance", func(t *testing.T) {
		require.NotEmpty(t, customerID)
		require.NotNil(t, feature)

		now := time.Now().UTC()
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      api.Numeric("10.00"),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		lineResp, err := v1.CreatePendingInvoiceLineWithResponse(t.Context(), customerID, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{{
				Name:      uniqueKey("deleteinv_line"),
				InvoiceAt: now.Add(-10 * time.Hour),
				Period: api.Period{
					From: now.Add(-24 * time.Hour),
					To:   now.Add(-2 * time.Hour),
				},
				Price: &price,
			}},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, lineResp.StatusCode())

		ctx := t.Context()
		customers := api.InvoiceListParamsCustomers{customerID}
		assert.EventuallyWithT(t, func(co *assert.CollectT) {
			listResp, err := v1.ListInvoicesWithResponse(ctx, &api.ListInvoicesParams{
				Customers: &customers,
				PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
			})
			require.NoError(co, err)
			require.Equal(co, http.StatusOK, listResp.StatusCode())

			idx := slices.IndexFunc(listResp.JSON200.Items, func(inv api.Invoice) bool {
				return inv.Status != api.InvoiceStatusGathering
			})
			require.NotEqual(co, -1, idx, "charges have not advanced a pending line into a standard invoice yet")
			invoiceID = listResp.JSON200.Items[idx].Id
		}, time.Minute, time.Second)
		require.NotEmpty(t, invoiceID)
	})

	t.Run("Should delete the standard invoice via v3 DELETE", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice creation")

		err := c.Invoices.Delete(t.Context(), invoiceID)
		c.requireStatus(http.StatusNoContent, err)
	})

	t.Run("Should return 400 when deleting the same invoice again", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice deletion")

		err := c.Invoices.Delete(t.Context(), invoiceID)
		problem := requireProblem(t, err, http.StatusBadRequest)
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 when deleting a gathering invoice via v3 DELETE", func(t *testing.T) {
		if gatheringInvoiceID == "" {
			t.Skip("no gathering invoice found in the list; skipping")
		}

		err := c.Invoices.Delete(t.Context(), gatheringInvoiceID)
		problem := requireProblem(t, err, http.StatusNotFound)
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 for an unknown invoice ID", func(t *testing.T) {
		err := c.Invoices.Delete(t.Context(), "01JAAAAAAAAAAAAAAAAAAAAAAA")
		problem := requireProblem(t, err, http.StatusNotFound)
		assert.NotNil(t, problem)
	})
}

// TestV3AdvanceBillingInvoice exercises POST /api/v3/openmeter/billing/invoices/{invoiceId}/advance
// Flow:
//   - Create a customer (v3)
//   - Create a standard invoice with draft status (v1)
//   - Apply the advance operation via v3 POST and assert 200
//   - Attempt to advance the same invoice again and assert 400
//   - Attempt to advance a gathering invoice via v3 POST and assert 400
func TestV3AdvanceBillingInvoice(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	var (
		customerID         string
		planID             string
		invoiceID          string // standard invoice ID
		gatheringInvoiceID string // gathering invoice ID
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("advanceinv_customer")

		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)
		customerID = customer.ID
	})

	t.Run("Should pin the customer to a short-draft-period billing profile", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		// A short, nonzero draft period parks the invoice in draft.waiting_auto_approval
		// right after creation (auto-advance is otherwise instantaneous with the default
		// P0D draft period, which would leave nothing in draft to advance). Once the
		// period elapses, the first advance call unblocks trigger_next and drives the
		// invoice forward; the second call then finds no further automatic transition.
		//
		// The default profile's payment settings auto-charge via CollectionMethodChargeAutomatically,
		// which (with credits enabled in e2e) settles the invoice to "paid" synchronously
		// the moment it's advanced past issued — leaving nothing observable to assert on.
		// Overriding payment to send_invoice (same as the manual-approval profiles used
		// elsewhere in this file) stops that auto-settlement, so advancing lands on a real,
		// inspectable non-final status instead.
		profile := createNewBillingProfileFromDefault(t, c, uniqueKey("advance_invoice"), func(profile *v3sdk.CreateBillingProfileRequest) {
			profile.Name = uniqueKey("advance_invoice_profile")
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &v3sdk.WorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(true)
			profile.Workflow.Invoicing.DraftPeriod = lo.ToPtr("PT5S")

			sendInvoice := lo.Must(v3sdk.WorkflowPaymentSettingsFromWorkflowPaymentSendInvoiceSettings(v3sdk.WorkflowPaymentSendInvoiceSettings{
				CollectionMethod: v3sdk.CollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})

		_, err := c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{
			BillingProfile: &v3sdk.ProfileReference{ID: profile.ID},
		})
		c.requireStatus(http.StatusOK, err)
	})

	t.Run("Should create a single gathering invoice", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      api.Numeric("10.00"),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		invoiceAt := now.Add(time.Hour)
		lineResp, err := v1.CreatePendingInvoiceLineWithResponse(t.Context(), customerID, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{
				{
					Name:      uniqueKey("advance_inv_gathering_line"),
					InvoiceAt: invoiceAt,
					Period: api.Period{
						From: now.Add(-24 * time.Hour),
						To:   invoiceAt,
					},
					Price: &price,
				},
			},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, lineResp.StatusCode(), "line: %s", string(lineResp.Body))
		require.NotNil(t, lineResp.JSON201)

		gatheringInvoiceID = (*lineResp.JSON201).Invoice.Id
		require.NotEmpty(t, gatheringInvoiceID)
	})

	t.Run("Should create meter, feature, plan, and subscription", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		// An active subscription is what actually drives the billing-worker's sync
		// loop for this customer; without one, a raw past-due pending line just sits
		// in the gathering invoice forever (nothing else periodically collects it).
		meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:         uniqueKey("advanceinv_meter"),
			Name:        gofakeit.ProductName(),
			Aggregation: v3sdk.MeterAggregationCount,
			EventType:   uniqueKey("advanceinv_event"),
		})
		c.requireStatus(http.StatusCreated, err)

		feature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:   uniqueKey("advanceinv_feature"),
			Name:  gofakeit.ProductName(),
			Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		})
		c.requireStatus(http.StatusCreated, err)

		plan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            uniqueKey("advanceinv_plan"),
			Name:           gofakeit.ProductName(),
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       uniqueKey("inv_phase_1"),
				Name:      uniqueKey("Test Phase"),
				RateCards: []v3sdk.RateCardInput{validUnitRateCard(*feature)},
			}},
		})
		c.requireStatus(http.StatusCreated, err)
		planID = plan.ID

		plan, err = c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, v3sdk.PlanStatusActive, plan.Status)

		_, err = c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: lo.ToPtr(customerID)},
			Plan:     v3sdk.SubscriptionChangePlan{ID: lo.ToPtr(planID)},
		})
		c.requireStatus(http.StatusCreated, err)
	})

	t.Run("Should create a standard invoice in draft status", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      api.Numeric("10.00"),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		lineResp, err := v1.CreatePendingInvoiceLineWithResponse(t.Context(), customerID, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{{
				Name:      uniqueKey("advanceinv_line"),
				InvoiceAt: now.Add(-10 * time.Hour),
				Period: api.Period{
					From: now.Add(-24 * time.Hour),
					To:   now.Add(-2 * time.Hour),
				},
				Price: &price,
			}},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, lineResp.StatusCode())

		ctx := t.Context()
		customers := api.InvoiceListParamsCustomers{customerID}
		assert.EventuallyWithT(t, func(co *assert.CollectT) {
			listResp, err := v1.ListInvoicesWithResponse(ctx, &api.ListInvoicesParams{
				Customers: &customers,
				PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
			})
			require.NoError(co, err)
			require.Equal(co, http.StatusOK, listResp.StatusCode())

			invoice, found := lo.Find(listResp.JSON200.Items, func(inv api.Invoice) bool {
				return inv.Status == api.InvoiceStatusDraft
			})
			assert.True(co, found, "charges have not advanced a pending line into a standard invoice yet")
			if found {
				invoiceID = invoice.Id
			}
		}, time.Minute, time.Second)
		require.NotEmpty(t, invoiceID)
	})

	t.Run("Should advance the standard invoice via v3 POST", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice creation")

		// The invoice rests in draft.waiting_auto_approval until the draft period
		// (PT5S) elapses, so retry until the state machine's guard clears.
		assert.EventuallyWithT(t, func(co *assert.CollectT) {
			inv, err := c.Invoices.Advance(t.Context(), invoiceID)
			require.Equal(co, http.StatusOK, c.statuses.last())
			require.NoError(co, err)
			require.NotNil(co, inv)

			stdInv, err := inv.AsInvoiceStandard()
			require.NoError(co, err)
			assert.Equal(co, invoiceID, stdInv.ID)
			// send_invoice payment settings prevent auto-settlement, so advancing
			// should land on a real non-final status (e.g. issued/payment_processing),
			// not race straight through to paid.
			assert.NotEqual(co, v3sdk.InvoiceStandardStatusPaid, stdInv.Status)
		}, 20*time.Second, time.Second)
	})

	t.Run("Should return 400 when advancing the same invoice again", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice advance")

		inv, err := c.Invoices.Advance(t.Context(), invoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})

	t.Run("Should return 400 when advancing a gathering invoice via v3 POST", func(t *testing.T) {
		require.NotEmpty(t, gatheringInvoiceID, "depends on gathering invoice creation")

		inv, err := c.Invoices.Advance(t.Context(), gatheringInvoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})
}

// pinBillingProfileFromDefault clones the default billing profile with the given edits (see
// createNewBillingProfileFromDefault) and pins the customer to it. Returns the created profile.
func pinBillingProfileFromDefault(t *testing.T, c *v3Client, customerID, keyPrefix string, edit func(*v3sdk.CreateBillingProfileRequest)) v3sdk.Profile {
	t.Helper()

	profile := createNewBillingProfileFromDefault(t, c, keyPrefix, edit)

	_, err := c.Customers.Billing.Update(t.Context(), customerID, v3sdk.UpsertCustomerBillingDataRequest{
		BillingProfile: &v3sdk.ProfileReference{ID: profile.ID},
	})
	c.requireStatus(http.StatusOK, err)

	return profile
}

// createFlatPendingLine posts a single flat, in-advance $10 pending line for the customer and
// returns the (gathering) invoice ID it landed on.
func createFlatPendingLine(t *testing.T, v1 *api.ClientWithResponses, customerID, name string, invoiceAt time.Time, period api.Period) string {
	t.Helper()

	price := api.RateCardUsageBasedPrice{}
	require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
		Amount:      api.Numeric("10.00"),
		Type:        api.FlatPriceWithPaymentTermTypeFlat,
		PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
	}))

	lineResp, err := v1.CreatePendingInvoiceLineWithResponse(t.Context(), customerID, api.InvoicePendingLineCreateInput{
		Currency: "USD",
		Lines: []api.InvoicePendingLineCreate{{
			Name:      name,
			InvoiceAt: invoiceAt,
			Period:    period,
			Price:     &price,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, lineResp.StatusCode(), "line: %s", string(lineResp.Body))
	require.NotNil(t, lineResp.JSON201)

	invoiceID := (*lineResp.JSON201).Invoice.Id
	require.NotEmpty(t, invoiceID)

	return invoiceID
}

// invoicePendingLinesNow explicitly triggers billing.Service.InvoicePendingLines for the customer
// (POST /billing/invoices/invoice), converting any currently-due pending lines into a standard
// invoice, and returns that invoice. A raw pending line just sits in the gathering invoice until
// something explicitly asks for it to be invoiced — nothing in this stack does that on a schedule
// (no cron, no ticker), so tests must trigger it themselves rather than poll and hope. asOf is
// optional; pass nil to invoice everything already due as of now, or a future time to force lines
// that aren't due yet (e.g. a subscription's next-period charge) to be included.
func invoicePendingLinesNow(t *testing.T, v1 *api.ClientWithResponses, customerID string, asOf *time.Time) api.Invoice {
	t.Helper()

	invoiceResp, err := v1.InvoicePendingLinesActionWithResponse(t.Context(), api.InvoicePendingLinesActionInput{
		CustomerId: customerID,
		AsOf:       asOf,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, invoiceResp.StatusCode(), "invoice pending lines: %s", string(invoiceResp.Body))
	require.NotNil(t, invoiceResp.JSON201)
	require.Len(t, *invoiceResp.JSON201, 1)

	return (*invoiceResp.JSON201)[0]
}

// TestV3ApproveBillingInvoice exercises POST /api/v3/openmeter/billing/invoices/{invoiceId}/approve
// Flow:
//   - Create a customer (v3)
//   - Create a standard invoice with draft status (v1)
//   - Apply the approve operation via v3 POST and assert 200
//   - Attempt to approve the same invoice again and assert 400
//   - Attempt to approve a gathering invoice via v3 POST and assert 400
//   - Create a standard invoice with manual_approval_needed status (v1)
//   - Apply the approve operation via v3 POST and assert 200
//   - Attempt to approve the same invoice again and assert 400
//   - Attempt to approve a gathering invoice via v3 POST and assert 400
func TestV3ApproveBillingInvoice(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	var (
		customerID              string
		gatheringInvoiceID      string
		invoiceID               string
		manualApprovalInvoiceID string
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("approveinv_customer")

		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)
		customerID = customer.ID
	})

	t.Run("Should pin the customer to an auto-advance billing profile", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		// Approve bypasses the draft-period wall-clock gate entirely (it's the manual
		// override for early issuance), so the draft period just needs to not have
		// elapsed by the time we call approve — it doesn't need to be short like in the
		// advance test. send_invoice keeps the post-approve status inspectable/non-final
		// instead of racing straight through to paid via credit auto-settlement.
		pinBillingProfileFromDefault(t, c, customerID, uniqueKey("approve_invoice"), func(profile *v3sdk.CreateBillingProfileRequest) {
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &v3sdk.WorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(true)
			profile.Workflow.Invoicing.DraftPeriod = lo.ToPtr("PT5M")

			sendInvoice := lo.Must(v3sdk.WorkflowPaymentSettingsFromWorkflowPaymentSendInvoiceSettings(v3sdk.WorkflowPaymentSendInvoiceSettings{
				CollectionMethod: v3sdk.CollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})
	})

	t.Run("Should create a single gathering invoice", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		invoiceAt := now.Add(time.Hour)
		gatheringInvoiceID = createFlatPendingLine(t, v1, customerID, uniqueKey("approve_inv_gathering_line"), invoiceAt, api.Period{
			From: now.Add(-24 * time.Hour),
			To:   invoiceAt,
		})
	})

	t.Run("Should create a standard invoice in draft status", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		createFlatPendingLine(t, v1, customerID, uniqueKey("approveinv_line"), now.Add(-10*time.Hour), api.Period{
			From: now.Add(-24 * time.Hour),
			To:   now.Add(-2 * time.Hour),
		})

		// Flat lines have a nil CollectionAt, so invoicing them cascades straight past
		// draft.waiting_for_collection to draft.waiting_auto_approval (AutoAdvance).
		invoice := invoicePendingLinesNow(t, v1, customerID, nil)
		require.Equal(t, api.InvoiceStatusDraft, invoice.Status)
		require.Equal(t, string(billing.StandardInvoiceStatusDraftWaitingAutoApproval), invoice.StatusDetails.ExtendedStatus)
		invoiceID = invoice.Id
	})

	t.Run("Should approve the standard invoice via v3 POST", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice creation")

		inv, err := c.Invoices.Approve(t.Context(), invoiceID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, inv)

		stdInv, err := inv.AsInvoiceStandard()
		require.NoError(t, err)
		assert.Equal(t, invoiceID, stdInv.ID)
	})

	t.Run("Should return 400 when approving the same invoice again", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice approval")

		inv, err := c.Invoices.Approve(t.Context(), invoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})

	t.Run("Should return 400 when approving a gathering invoice via v3 POST", func(t *testing.T) {
		require.NotEmpty(t, gatheringInvoiceID, "depends on gathering invoice creation")

		inv, err := c.Invoices.Approve(t.Context(), gatheringInvoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})

	t.Run("Should pin the customer to a manual-approval billing profile", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		pinBillingProfileFromDefault(t, c, customerID, uniqueKey("approve_invoice_manual"), func(profile *v3sdk.CreateBillingProfileRequest) {
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &v3sdk.WorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)

			sendInvoice := lo.Must(v3sdk.WorkflowPaymentSettingsFromWorkflowPaymentSendInvoiceSettings(v3sdk.WorkflowPaymentSendInvoiceSettings{
				CollectionMethod: v3sdk.CollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})
	})

	t.Run("Should create a standard invoice with manual_approval_needed status", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		createFlatPendingLine(t, v1, customerID, uniqueKey("approveinv_manual_line"), now.Add(-10*time.Hour), api.Period{
			From: now.Add(-24 * time.Hour),
			To:   now.Add(-2 * time.Hour),
		})

		invoice := invoicePendingLinesNow(t, v1, customerID, nil)
		require.Equal(t, api.InvoiceStatusDraft, invoice.Status)
		require.Equal(t, "draft.manual_approval_needed", invoice.StatusDetails.ExtendedStatus)
		manualApprovalInvoiceID = invoice.Id
	})

	t.Run("Should approve the manual-approval-needed invoice via v3 POST", func(t *testing.T) {
		require.NotEmpty(t, manualApprovalInvoiceID, "depends on invoice creation")

		inv, err := c.Invoices.Approve(t.Context(), manualApprovalInvoiceID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, inv)

		stdInv, err := inv.AsInvoiceStandard()
		require.NoError(t, err)
		assert.Equal(t, manualApprovalInvoiceID, stdInv.ID)
	})

	t.Run("Should return 400 when approving the manual-approval-needed invoice again", func(t *testing.T) {
		require.NotEmpty(t, manualApprovalInvoiceID, "depends on invoice approval")

		inv, err := c.Invoices.Approve(t.Context(), manualApprovalInvoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})

	t.Run("Should return 400 when approving a gathering invoice via v3 POST again", func(t *testing.T) {
		require.NotEmpty(t, gatheringInvoiceID, "depends on gathering invoice creation")

		inv, err := c.Invoices.Approve(t.Context(), gatheringInvoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})
}

// TestV3RetryBillingInvoice exercises POST /api/v3/openmeter/billing/invoices/{invoiceId}/retry
// Flow:
//   - Create a customer (v3)
//   - Create a standard invoice with draft status (v1)
//   - Attempt to retry it and assert 400 (retry is only valid from a failed/invalid state;
//     that state can't be reliably reproduced through the public API without a mocked
//     invoicing app or line engine, so this test covers retry's rejection behavior)
//   - Attempt to retry a gathering invoice via v3 POST and assert 400
func TestV3RetryBillingInvoice(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	var (
		customerID         string
		gatheringInvoiceID string
		invoiceID          string
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("retryinv_customer")

		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)
		customerID = customer.ID
	})

	t.Run("Should pin the customer to an auto-advance billing profile", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		pinBillingProfileFromDefault(t, c, customerID, uniqueKey("retry_invoice"), func(profile *v3sdk.CreateBillingProfileRequest) {
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &v3sdk.WorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(true)
			profile.Workflow.Invoicing.DraftPeriod = lo.ToPtr("PT5M")

			sendInvoice := lo.Must(v3sdk.WorkflowPaymentSettingsFromWorkflowPaymentSendInvoiceSettings(v3sdk.WorkflowPaymentSendInvoiceSettings{
				CollectionMethod: v3sdk.CollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})
	})

	t.Run("Should create a single gathering invoice", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		invoiceAt := now.Add(time.Hour)
		gatheringInvoiceID = createFlatPendingLine(t, v1, customerID, uniqueKey("retry_inv_gathering_line"), invoiceAt, api.Period{
			From: now.Add(-24 * time.Hour),
			To:   invoiceAt,
		})
	})

	t.Run("Should create a standard invoice in draft status", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		createFlatPendingLine(t, v1, customerID, uniqueKey("retryinv_line"), now.Add(-10*time.Hour), api.Period{
			From: now.Add(-24 * time.Hour),
			To:   now.Add(-2 * time.Hour),
		})

		// Flat lines have a nil CollectionAt, so invoicing them cascades straight past
		// draft.waiting_for_collection to draft.waiting_auto_approval (AutoAdvance).
		invoice := invoicePendingLinesNow(t, v1, customerID, nil)
		require.Equal(t, api.InvoiceStatusDraft, invoice.Status)
		require.Equal(t, string(billing.StandardInvoiceStatusDraftWaitingAutoApproval), invoice.StatusDetails.ExtendedStatus)
		invoiceID = invoice.Id
	})

	t.Run("Should return 400 when retrying a non-retryable draft invoice", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice creation")

		inv, err := c.Invoices.Retry(t.Context(), invoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})

	t.Run("Should return 400 when retrying a gathering invoice via v3 POST", func(t *testing.T) {
		require.NotEmpty(t, gatheringInvoiceID, "depends on gathering invoice creation")

		inv, err := c.Invoices.Retry(t.Context(), invoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})
}

// TestV3SnapshotQuantitiesBillingInvoice exercises POST /api/v3/openmeter/billing/invoices/{invoiceId}/snapshot-quantities
// Flow:
//   - Create a customer (v3)
//   - Create a standard invoice with a usage-based line resting in draft.waiting_for_collection (v1)
//   - Apply the snapshot-quantities (force-collect) operation via v3 POST and assert 200
//   - Attempt to snapshot-quantities the same invoice again and assert 400
//   - Attempt to snapshot-quantities a gathering invoice via v3 POST and assert 400
func TestV3SnapshotQuantitiesBillingInvoice(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	var (
		customerID         string
		gatheringInvoiceID string
		invoiceID          string
		subscriptionID     string
		feature            *v3sdk.Feature
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("snapshotinv_customer")

		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: lo.ToPtr("USD"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)
		customerID = customer.ID
	})

	t.Run("Should pin the customer to a billing profile with a 1-day collection interval", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		// Preserve the cloned Collection.Alignment (subscription-aligned by default) and
		// only override Interval — replacing the whole Collection struct drops Alignment,
		// which then fails collection-config validation with a 500 (an anchored
		// alignment requires AnchoredAlignmentDetail). send_invoice avoids racing to paid
		// once force-collected.
		pinBillingProfileFromDefault(t, c, customerID, uniqueKey("snapshot_invoice"), func(profile *v3sdk.CreateBillingProfileRequest) {
			if profile.Workflow.Collection == nil {
				profile.Workflow.Invoicing = &v3sdk.WorkflowInvoicingSettings{}
			}
			profile.Workflow.Collection.Interval = lo.ToPtr("P1D")

			sendInvoice := lo.Must(v3sdk.WorkflowPaymentSettingsFromWorkflowPaymentSendInvoiceSettings(v3sdk.WorkflowPaymentSendInvoiceSettings{
				CollectionMethod: v3sdk.CollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})
	})

	t.Run("Should create meter, feature, plan, and subscription", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		// A raw manually-created usage-based pending line (no subscription behind it)
		// isn't ratable — the customer has no usage attribution / subject keys to query
		// usage against, and the server 500s trying to calculate it. A subscription's own
		// usage-based rate card line doesn't have this problem, so drive the test invoice
		// through a subscription instead of a manual pending line.
		meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:         uniqueKey("snapshotinv_meter"),
			Name:        gofakeit.ProductName(),
			Aggregation: v3sdk.MeterAggregationCount,
			EventType:   uniqueKey("snapshotinv_event"),
		})
		c.requireStatus(http.StatusCreated, err)

		feature, err = c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:   uniqueKey("snapshotinv_feature"),
			Name:  gofakeit.ProductName(),
			Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		})
		c.requireStatus(http.StatusCreated, err)

		plan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            uniqueKey("snapshotinv_plan"),
			Name:           gofakeit.ProductName(),
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       uniqueKey("snapshotinv_phase_1"),
				Name:      uniqueKey("Test Phase"),
				RateCards: []v3sdk.RateCardInput{validUnitRateCard(*feature)},
			}},
		})
		c.requireStatus(http.StatusCreated, err)
		planID := plan.ID

		plan, err = c.Plans.Publish(t.Context(), planID)
		c.requireStatus(http.StatusOK, err)
		assert.Equal(t, v3sdk.PlanStatusActive, plan.Status)

		sub, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: lo.ToPtr(customerID)},
			Plan:     v3sdk.SubscriptionChangePlan{ID: lo.ToPtr(planID)},
		})
		c.requireStatus(http.StatusCreated, err)
		subscriptionID = sub.ID
	})

	t.Run("Should create a standard invoice from the subscription's usage-based line", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")
		require.NotEmpty(t, subscriptionID, "depends on subscription creation")

		// Billing cadences are restricted to weekly-or-longer presets, so the
		// subscription's first-period line naturally isn't due for a week+. Canceling
		// the subscription immediately settles its current (short, just-started) period,
		// making the line due right away — the same mechanism the credit-then-invoice
		// cancellation flow in TestInvoiceEditFlatFeeManualOverrides relies on (that test
		// sleeps briefly before canceling to let the subscription's charge materialize first).
		time.Sleep(2 * time.Second)

		_, err := c.Subscriptions.Cancel(t.Context(), subscriptionID, v3sdk.SubscriptionCancel{})
		c.requireStatus(http.StatusOK, err)

		invoice := invoicePendingLinesNow(t, v1, customerID, nil)
		require.Equal(t, api.InvoiceStatusDraft, invoice.Status)
		require.Equal(t, "draft.waiting_for_collection", invoice.StatusDetails.ExtendedStatus)
		invoiceID = invoice.Id
	})

	t.Run("Should create a single gathering invoice", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		now := time.Now().UTC()
		invoiceAt := now.Add(time.Hour)
		gatheringInvoiceID = createFlatPendingLine(t, v1, customerID, uniqueKey("snapshot_inv_gathering_line"), invoiceAt, api.Period{
			From: now.Add(-24 * time.Hour),
			To:   invoiceAt,
		})
	})

	t.Run("Should snapshot-quantities the standard invoice via v3 POST", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice creation")

		inv, err := c.Invoices.SnapshotQuantities(t.Context(), invoiceID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, inv)

		stdInv, err := inv.AsInvoiceStandard()
		require.NoError(t, err)
		assert.Equal(t, invoiceID, stdInv.ID)
	})

	t.Run("Should return 400 when snapshot-quantities-ing the same invoice again", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice snapshot-quantities")

		inv, err := c.Invoices.SnapshotQuantities(t.Context(), invoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})

	t.Run("Should return 400 when snapshot-quantities-ing a gathering invoice via v3 POST", func(t *testing.T) {
		require.NotEmpty(t, gatheringInvoiceID, "depends on gathering invoice creation")

		inv, err := c.Invoices.SnapshotQuantities(t.Context(), gatheringInvoiceID)
		require.Equal(t, http.StatusBadRequest, c.statuses.last())
		require.Error(t, err)
		assert.Nil(t, inv)
	})
}
