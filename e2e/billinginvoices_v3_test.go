package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
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
		feature            *apiv3.Feature
		invoiceID          string // standard invoice ID
		gatheringInvoiceID string // gathering invoice ID
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

	t.Run("Should create a meter and a feature for usage-based billing", func(t *testing.T) {
		status, meter, problem := c.CreateMeter(apiv3.CreateMeterRequest{
			Key:         uniqueKey("inv_meter"),
			Name:        "Invoice Test Meter",
			Aggregation: apiv3.MeterAggregationCount,
			EventType:   uniqueKey("inv_event"),
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, meter)

		status, f, problem := c.CreateFeature(apiv3.CreateFeatureRequest{
			Key:   uniqueKey("inv_feature"),
			Name:  "Invoice Test Feature",
			Meter: &apiv3.FeatureMeterReference{Id: meter.Id},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, f)

		feature = f
	})

	t.Run("Should create and publish a plan with flat and unit rate cards", func(t *testing.T) {
		require.NotNil(t, feature, "depends on feature creation")

		status, plan, problem := c.CreatePlan(apiv3.CreatePlanRequest{
			Key:            uniqueKey("inv_plan"),
			Name:           "Invoice Test Plan",
			Currency:       "USD",
			BillingCadence: apiv3.ISO8601Duration("P1M"),
			Phases: []apiv3.BillingPlanPhase{{
				Key:  "phase_1",
				Name: "Test Phase",
				RateCards: []apiv3.BillingRateCard{
					validFlatRateCard("inv_fee"),
					validUnitRateCard(*feature),
				},
			}},
		})
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
			SettlementMode: lo.ToPtr(apiv3.BillingSettlementModeCreditThenInvoice),
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
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

		status, inv, problem := c.GetBillingInvoice(invoiceID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, inv)

		// The union must decode as a standard invoice.
		stdInv, err := inv.AsBillingInvoiceStandard()
		require.NoError(t, err, "expected invoice to decode as BillingInvoiceStandard")

		assert.Equal(t, invoiceID, stdInv.Id)
		assert.Equal(t, apiv3.BillingInvoiceStandardTypeStandard, stdInv.Type)
		assert.Equal(t, apiv3.CurrencyCode("USD"), stdInv.Currency)
		assert.NotEmpty(t, stdInv.Status)
		assert.NotEmpty(t, stdInv.CreatedAt)
	})

	t.Run("Should return 404 for the gathering invoice", func(t *testing.T) {
		if gatheringInvoiceID == "" {
			t.Skip("no gathering invoice found in the list; skipping")
		}

		status, inv, problem := c.GetBillingInvoice(gatheringInvoiceID)
		assert.Equal(t, http.StatusNotFound, status, "body: %+v", inv)
		assert.Nil(t, inv)
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 for an unknown invoice ID", func(t *testing.T) {
		status, inv, problem := c.GetBillingInvoice("01JAAAAAAAAAAAAAAAAAAAAAAA")
		assert.Equal(t, http.StatusNotFound, status)
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
		feature    *apiv3.Feature
		invoiceID  string
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("listinv_customer")
		currency := apiv3.CurrencyCode("USD")

		status, customer, problem := c.CreateCustomer(apiv3.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: &currency,
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, customer)
		customerID = customer.Id
	})

	t.Run("Should create meter, feature, plan, and subscription", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		status, meter, problem := c.CreateMeter(apiv3.CreateMeterRequest{
			Key:         uniqueKey("listinv_meter"),
			Name:        gofakeit.ProductName(),
			Aggregation: apiv3.MeterAggregationCount,
			EventType:   uniqueKey("listinv_event"),
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)

		status, f, problem := c.CreateFeature(apiv3.CreateFeatureRequest{
			Key:   uniqueKey("listinv_feature"),
			Name:  gofakeit.ProductName(),
			Meter: &apiv3.FeatureMeterReference{Id: meter.Id},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		feature = f

		status, plan, problem := c.CreatePlan(apiv3.CreatePlanRequest{
			Key:            uniqueKey("listinv_plan"),
			Name:           gofakeit.ProductName(),
			Currency:       "USD",
			BillingCadence: apiv3.ISO8601Duration("P1M"),
			Phases: []apiv3.BillingPlanPhase{{
				Key:       uniqueKey("inv_phase_1"),
				Name:      uniqueKey("Test Phase"),
				RateCards: []apiv3.BillingRateCard{validUnitRateCard(*feature)},
			}},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		planID = plan.Id

		status, plan, problem = c.PublishPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		assert.Equal(t, apiv3.BillingPlanStatusActive, plan.Status)

		status, _, problem = c.CreateSubscription(apiv3.BillingSubscriptionCreate{
			Customer: struct {
				Id  *apiv3.ULID                `json:"id,omitempty"`
				Key *apiv3.ExternalResourceKey `json:"key,omitempty"`
			}{Id: lo.ToPtr(customerID)},
			Plan: struct {
				Id      *apiv3.ULID        `json:"id,omitempty"`
				Key     *apiv3.ResourceKey `json:"key,omitempty"`
				Version *int               `json:"version,omitempty"`
			}{Id: lo.ToPtr(planID)},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
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

		status, resp, problem := c.ListBillingInvoices(ListBillingInvoicesOptions{
			CustomerID: customerID,
			PageSize:   100,
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, resp)
		require.NotEmpty(t, resp.Data, "expected at least one invoice for customer %s", customerID)

		for _, inv := range resp.Data {
			stdInv, err := inv.AsBillingInvoiceStandard()
			require.NoError(t, err, "all returned invoices must decode as standard (no gathering)")
			assert.NotEqual(t, "gathering", string(stdInv.Status), "gathering invoice leaked into list response")
		}
	})

	t.Run("Should filter by customer_id and return only that customer's invoices", func(t *testing.T) {
		require.NotEmpty(t, customerID)
		require.NotEmpty(t, invoiceID)

		status, resp, problem := c.ListBillingInvoices(ListBillingInvoicesOptions{
			CustomerID: customerID,
			PageSize:   100,
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, resp)

		found := slices.ContainsFunc(resp.Data, func(inv apiv3.BillingInvoice) bool {
			stdInv, err := inv.AsBillingInvoiceStandard()
			if err != nil {
				return false
			}
			return stdInv.Id == invoiceID
		})
		assert.True(t, found, "invoice %s must appear when filtered by customer %s", invoiceID, customerID)
	})

	t.Run("Should filter by status oeq and return only matching invoices", func(t *testing.T) {
		require.NotEmpty(t, customerID)

		status, resp, problem := c.ListBillingInvoices(ListBillingInvoicesOptions{
			CustomerID: customerID,
			StatusOeq:  []string{"draft", "issued"},
			PageSize:   100,
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, resp)

		for _, inv := range resp.Data {
			stdInv, err := inv.AsBillingInvoiceStandard()
			require.NoError(t, err)
			s := string(stdInv.Status)
			assert.True(t, s == "draft" || s == "issued",
				"expected status draft or issued, got %s", s)
		}
	})

	t.Run("Should accept sort=issued_at and return 200", func(t *testing.T) {
		require.NotEmpty(t, customerID)

		status, resp, problem := c.ListBillingInvoices(ListBillingInvoicesOptions{
			CustomerID: customerID,
			Sort:       "issued_at",
			PageSize:   100,
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, resp)
	})

	t.Run("Should accept sort=service_period_start desc and return 200", func(t *testing.T) {
		require.NotEmpty(t, customerID)

		status, resp, problem := c.ListBillingInvoices(ListBillingInvoicesOptions{
			CustomerID: customerID,
			Sort:       "service_period_start desc",
			PageSize:   100,
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, resp)
	})

	t.Run("Should return empty result for an unknown customer ID", func(t *testing.T) {
		status, resp, problem := c.ListBillingInvoices(ListBillingInvoicesOptions{
			CustomerID: "01JAAAAAAAAAAAAAAAAAAAAAAA",
			PageSize:   20,
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, resp)
		assert.Empty(t, resp.Data, "unknown customer should yield no invoices")
	})

	t.Run("Should return 400 for an unsupported filter operator (neq on status)", func(t *testing.T) {
		v := url.Values{}
		v.Set("filter[status][neq]", "draft")
		pathWithQuery := fmt.Sprintf("/billing/invoices?%s", v.Encode())
		rawStatus, rawBody, problem := c.do(http.MethodGet, pathWithQuery, nil)
		assert.Equal(t, http.StatusBadRequest, rawStatus, "body: %s", string(rawBody))
		assert.NotNil(t, problem)
	})

	t.Run("Should return page metadata", func(t *testing.T) {
		require.NotEmpty(t, customerID)

		status, resp, problem := c.ListBillingInvoices(ListBillingInvoicesOptions{
			CustomerID: customerID,
			PageSize:   1,
			PageNumber: 1,
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, resp)
		assert.Equal(t, float32(1), resp.Meta.Page.Size)
		assert.Equal(t, float32(1), resp.Meta.Page.Number)
		assert.GreaterOrEqual(t, resp.Meta.Page.Total, float32(1))
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
		feature            *apiv3.Feature
		invoice            api.Invoice
		gatheringInvoiceID string
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("listinv_customer")
		currency := apiv3.CurrencyCode("USD")

		status, customer, problem := c.CreateCustomer(apiv3.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: &currency,
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, customer)
		customerID = customer.Id
	})

	t.Run("Should pin the customer to a manual-approval billing profile", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		profile := createNewBillingProfileFromDefault(t, c, uniqueKey("update_invoice"), func(profile *apiv3.CreateBillingProfileRequest) {
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &apiv3.BillingWorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)

			sendInvoice := apiv3.BillingWorkflowPaymentSettings{}
			require.NoError(t, sendInvoice.FromBillingWorkflowPaymentSendInvoiceSettings(apiv3.BillingWorkflowPaymentSendInvoiceSettings{
				CollectionMethod: apiv3.BillingWorkflowPaymentSendInvoiceSettingsCollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})

		status, _, problem := c.UpdateCustomerBilling(customerID, apiv3.UpsertCustomerBillingDataRequest{
			BillingProfile: &apiv3.BillingProfileReference{Id: profile.Id},
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
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

		status, meter, problem := c.CreateMeter(apiv3.CreateMeterRequest{
			Key:         uniqueKey("listinv_meter"),
			Name:        gofakeit.ProductName(),
			Aggregation: apiv3.MeterAggregationCount,
			EventType:   uniqueKey("listinv_event"),
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)

		status, f, problem := c.CreateFeature(apiv3.CreateFeatureRequest{
			Key:   uniqueKey("listinv_feature"),
			Name:  gofakeit.ProductName(),
			Meter: &apiv3.FeatureMeterReference{Id: meter.Id},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		feature = f

		status, plan, problem := c.CreatePlan(apiv3.CreatePlanRequest{
			Key:            uniqueKey("listinv_plan"),
			Name:           gofakeit.ProductName(),
			Currency:       "USD",
			BillingCadence: apiv3.ISO8601Duration("P1M"),
			Phases: []apiv3.BillingPlanPhase{{
				Key:       uniqueKey("inv_phase_1"),
				Name:      uniqueKey("Test Phase"),
				RateCards: []apiv3.BillingRateCard{validUnitRateCard(*feature)},
			}},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		planID = plan.Id

		status, plan, problem = c.PublishPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		assert.Equal(t, apiv3.BillingPlanStatusActive, plan.Status)

		status, _, problem = c.CreateSubscription(apiv3.BillingSubscriptionCreate{
			Customer: struct {
				Id  *apiv3.ULID                `json:"id,omitempty"`
				Key *apiv3.ExternalResourceKey `json:"key,omitempty"`
			}{Id: lo.ToPtr(customerID)},
			Plan: struct {
				Id      *apiv3.ULID        `json:"id,omitempty"`
				Key     *apiv3.ResourceKey `json:"key,omitempty"`
				Version *int               `json:"version,omitempty"`
			}{Id: lo.ToPtr(planID)},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
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

		var updateReq apiv3.UpdateInvoiceRequest
		require.NoError(t, updateReq.FromUpdateInvoiceStandardRequest(apiv3.UpdateInvoiceStandardRequest{
			Type:        apiv3.UpdateInvoiceStandardRequestTypeStandard,
			Description: lo.ToPtr(newDescription),
			Customer: apiv3.UpdateInvoiceCustomer{
				Id:   *invoice.Customer.Id,
				Name: *invoice.Customer.Name,
			},
			Supplier: updateSupplierFromInvoiceParty(invoice.Supplier),
			Workflow: apiv3.UpdateInvoiceWorkflowSettings{},
			Lines:    nil,
		}))

		status, updatedInv, problem := c.UpdateBillingInvoice(invoice.Id, updateReq)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, updatedInv)

		stdInv, err := updatedInv.AsBillingInvoiceStandard()
		require.NoError(t, err)
		assert.Equal(t, invoice.Id, stdInv.Id)
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

		status, raw, problem := c.do(http.MethodPut, "/billing/invoices/"+invoice.Id, body)
		assert.Equal(t, http.StatusBadRequest, status, "body: %s", string(raw))
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 when updating a gathering invoice via v3 PUT", func(t *testing.T) {
		require.NotEmpty(t, gatheringInvoiceID, "depends on gathering invoice creation")

		var updateReq apiv3.UpdateInvoiceRequest
		require.NoError(t, updateReq.FromUpdateInvoiceStandardRequest(apiv3.UpdateInvoiceStandardRequest{
			Type: apiv3.UpdateInvoiceStandardRequestTypeStandard,
			Customer: apiv3.UpdateInvoiceCustomer{
				Id:   customerID,
				Name: "placeholder",
			},
			Supplier: apiv3.UpdateSupplier{},
			Workflow: apiv3.UpdateInvoiceWorkflowSettings{},
		}))

		status, updatedInv, problem := c.UpdateBillingInvoice(gatheringInvoiceID, updateReq)
		assert.Equal(t, http.StatusNotFound, status, "problem: %+v", problem)
		assert.Nil(t, updatedInv)
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 for an unknown invoice ID via v3 PUT", func(t *testing.T) {
		var updateReq apiv3.UpdateInvoiceRequest
		require.NoError(t, updateReq.FromUpdateInvoiceStandardRequest(apiv3.UpdateInvoiceStandardRequest{
			Type: apiv3.UpdateInvoiceStandardRequestTypeStandard,
			Customer: apiv3.UpdateInvoiceCustomer{
				Id:   customerID,
				Name: "placeholder",
			},
			Supplier: apiv3.UpdateSupplier{},
			Workflow: apiv3.UpdateInvoiceWorkflowSettings{},
		}))

		status, updatedInv, problem := c.UpdateBillingInvoice("01JAAAAAAAAAAAAAAAAAAAAAAA", updateReq)
		assert.Equal(t, http.StatusNotFound, status, "problem: %+v", problem)
		assert.Nil(t, updatedInv)
		assert.NotNil(t, problem)
	})

	t.Run("Should reject a line update with a negative amount", func(t *testing.T) {
		require.NotEmpty(t, invoice)

		status, currentInv, problem := c.GetBillingInvoice(invoice.Id)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, currentInv)
		stdInv, err := currentInv.AsBillingInvoiceStandard()
		require.NoError(t, err)

		lines := existingUpdateLines(t, stdInv)

		var negativeLine apiv3.UpdateInvoiceLine
		require.NoError(t, negativeLine.FromUpdateInvoiceStandardLine(newFlatUpdateLine(t, uniqueKey("negative_amount_line"), stdInv.ServicePeriod, "-5")))
		lines = append(lines, negativeLine)

		var updateReq apiv3.UpdateInvoiceRequest
		require.NoError(t, updateReq.FromUpdateInvoiceStandardRequest(apiv3.UpdateInvoiceStandardRequest{
			Type: apiv3.UpdateInvoiceStandardRequestTypeStandard,
			Customer: apiv3.UpdateInvoiceCustomer{
				Id:   *invoice.Customer.Id,
				Name: *invoice.Customer.Name,
			},
			Supplier: updateSupplierFromInvoiceParty(invoice.Supplier),
			Workflow: apiv3.UpdateInvoiceWorkflowSettings{},
			Lines:    &lines,
		}))

		status, updatedInv, problem := c.UpdateBillingInvoice(invoice.Id, updateReq)
		require.Equal(t, http.StatusBadRequest, status, "problem: %+v", problem)
		assert.Nil(t, updatedInv)
		assertProblemDetail(t, problem, "must not be negative")
	})

	t.Run("Should apply a valid line addition and reflect it in the response", func(t *testing.T) {
		require.NotEmpty(t, invoice)

		status, currentInv, problem := c.GetBillingInvoice(invoice.Id)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, currentInv)
		stdInv, err := currentInv.AsBillingInvoiceStandard()
		require.NoError(t, err)

		lines := existingUpdateLines(t, stdInv)

		newLineName := uniqueKey("valid_new_line")
		var newLine apiv3.UpdateInvoiceLine
		require.NoError(t, newLine.FromUpdateInvoiceStandardLine(newFlatUpdateLine(t, newLineName, stdInv.ServicePeriod, "15")))
		lines = append(lines, newLine)

		var updateReq apiv3.UpdateInvoiceRequest
		require.NoError(t, updateReq.FromUpdateInvoiceStandardRequest(apiv3.UpdateInvoiceStandardRequest{
			Type: apiv3.UpdateInvoiceStandardRequestTypeStandard,
			Customer: apiv3.UpdateInvoiceCustomer{
				Id:   *invoice.Customer.Id,
				Name: *invoice.Customer.Name,
			},
			Supplier: updateSupplierFromInvoiceParty(invoice.Supplier),
			Workflow: apiv3.UpdateInvoiceWorkflowSettings{},
			Lines:    &lines,
		}))

		status, updatedInv, problem := c.UpdateBillingInvoice(invoice.Id, updateReq)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, updatedInv)

		updatedStdInv, err := updatedInv.AsBillingInvoiceStandard()
		require.NoError(t, err)
		require.NotNil(t, updatedStdInv.Lines)

		found := false
		for _, l := range *updatedStdInv.Lines {
			stdLine, err := l.AsBillingInvoiceStandardLine()
			require.NoError(t, err)
			if stdLine.Name == newLineName {
				found = true
				p, err := stdLine.RateCard.Price.AsBillingPriceFlat()
				require.NoError(t, err)
				assert.Equal(t, "15", p.Amount)
			}
		}
		assert.True(t, found, "expected the newly added line %q to appear in the updated invoice", newLineName)
	})
}

// updateSupplierFromInvoiceParty maps a v1 BillingParty (the invoice's supplier
// snapshot) into an UpdateSupplier that round-trips its current name and
// address. Update requests replace the full supplier object: an omitted name
// or address is not "leave unchanged", it's reset to empty, which then fails
// SupplierContact.Validate() (name and country are required for invoicing).
func updateSupplierFromInvoiceParty(party api.BillingParty) apiv3.UpdateSupplier {
	supplier := apiv3.UpdateSupplier{
		Id:   party.Id,
		Name: party.Name,
	}

	if party.Addresses != nil && len(*party.Addresses) > 0 {
		addr := (*party.Addresses)[0]
		supplier.Addresses = &apiv3.UpdateBillingPartyAddresses{
			BillingAddress: apiv3.UpdateAddress{
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
func existingUpdateLines(t *testing.T, stdInv apiv3.BillingInvoiceStandard) []apiv3.UpdateInvoiceLine {
	t.Helper()

	if stdInv.Lines == nil {
		return nil
	}

	lines := make([]apiv3.UpdateInvoiceLine, 0, len(*stdInv.Lines))
	for _, l := range *stdInv.Lines {
		stdLine, err := l.AsBillingInvoiceStandardLine()
		require.NoError(t, err)
		lines = append(lines, updateLineFromStandardLine(t, stdLine))
	}

	return lines
}

// updateLineFromStandardLine maps a read-side standard line back into an
// UpdateInvoiceLine, preserving its ID and rate card so that resubmitting it
// unchanged is a no-op rather than an unintended edit or deletion.
func updateLineFromStandardLine(t *testing.T, line apiv3.BillingInvoiceStandardLine) apiv3.UpdateInvoiceLine {
	t.Helper()

	rateCard := apiv3.UpdateInvoiceLineRateCard{
		FeatureKey: line.RateCard.FeatureKey,
	}

	if line.RateCard.Discounts != nil {
		rateCard.Discounts = &apiv3.UpdateDiscounts{
			Percentage: line.RateCard.Discounts.Percentage,
			Usage:      line.RateCard.Discounts.Usage,
		}
	}

	if line.RateCard.TaxConfig != nil {
		rateCard.TaxConfig = &apiv3.UpdateRateCardTaxConfig{
			Behavior: line.RateCard.TaxConfig.Behavior,
			Code:     apiv3.UpdateResourceReference{Id: line.RateCard.TaxConfig.Code.Id},
		}
	}

	disc, err := line.RateCard.Price.Discriminator()
	require.NoError(t, err)

	switch disc {
	case string(apiv3.BillingPriceFlatTypeFlat):
		p, err := line.RateCard.Price.AsBillingPriceFlat()
		require.NoError(t, err)
		require.NoError(t, rateCard.Price.FromUpdatePriceFlat(apiv3.UpdatePriceFlat{
			Type:   apiv3.UpdatePriceFlatTypeFlat,
			Amount: p.Amount,
		}))
	case string(apiv3.BillingPriceUnitTypeUnit):
		p, err := line.RateCard.Price.AsBillingPriceUnit()
		require.NoError(t, err)
		require.NoError(t, rateCard.Price.FromUpdatePriceUnit(apiv3.UpdatePriceUnit{
			Type:   apiv3.UpdatePriceUnitTypeUnit,
			Amount: p.Amount,
		}))
	default:
		t.Fatalf("unsupported rate card price type in test fixture: %s", disc)
	}

	stdLine := apiv3.UpdateInvoiceStandardLine{
		Id:          line.Id,
		Name:        line.Name,
		Description: line.Description,
		Labels:      line.Labels,
		Type:        apiv3.UpdateInvoiceStandardLineTypeStandardLine,
		ServicePeriod: apiv3.UpdateClosedPeriod{
			From: line.ServicePeriod.From,
			To:   line.ServicePeriod.To,
		},
		RateCard: rateCard,
	}

	var out apiv3.UpdateInvoiceLine
	require.NoError(t, out.FromUpdateInvoiceStandardLine(stdLine))
	return out
}

// newFlatUpdateLine builds a brand-new (no ID) flat-priced UpdateInvoiceStandardLine
// for tests that append a manual line to an existing invoice.
func newFlatUpdateLine(t *testing.T, name string, period apiv3.ClosedPeriod, amount string) apiv3.UpdateInvoiceStandardLine {
	t.Helper()

	rateCard := apiv3.UpdateInvoiceLineRateCard{}
	require.NoError(t, rateCard.Price.FromUpdatePriceFlat(apiv3.UpdatePriceFlat{
		Type:   apiv3.UpdatePriceFlatTypeFlat,
		Amount: amount,
	}))

	return apiv3.UpdateInvoiceStandardLine{
		Name:          name,
		Type:          apiv3.UpdateInvoiceStandardLineTypeStandardLine,
		ServicePeriod: apiv3.UpdateClosedPeriod(period),
		RateCard:      rateCard,
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
		feature            *apiv3.Feature
		invoiceID          string // standard invoice ID
		gatheringInvoiceID string // gathering invoice ID
	)

	t.Run("Should create a customer", func(t *testing.T) {
		key := uniqueKey("deleteinv_customer")
		currency := apiv3.CurrencyCode("USD")

		status, customer, problem := c.CreateCustomer(apiv3.CreateCustomerRequest{
			Key:      key,
			Name:     gofakeit.ProductName(),
			Currency: &currency,
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, customer)
		customerID = customer.Id
	})

	t.Run("Should pin the customer to a manual-approval billing profile", func(t *testing.T) {
		require.NotEmpty(t, customerID, "depends on customer creation")

		// Only draft invoices can be deleted; the default profile's auto-collect
		// settings otherwise advance the invoice past draft too quickly for this
		// test to observe (and delete) a draft invoice.
		profile := createNewBillingProfileFromDefault(t, c, uniqueKey("delete_invoice"), func(profile *apiv3.CreateBillingProfileRequest) {
			if profile.Workflow.Invoicing == nil {
				profile.Workflow.Invoicing = &apiv3.BillingWorkflowInvoicingSettings{}
			}
			profile.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)

			sendInvoice := apiv3.BillingWorkflowPaymentSettings{}
			require.NoError(t, sendInvoice.FromBillingWorkflowPaymentSendInvoiceSettings(apiv3.BillingWorkflowPaymentSendInvoiceSettings{
				CollectionMethod: apiv3.BillingWorkflowPaymentSendInvoiceSettingsCollectionMethodSendInvoice,
			}))
			profile.Workflow.Payment = &sendInvoice
		})

		status, _, problem := c.UpdateCustomerBilling(customerID, apiv3.UpsertCustomerBillingDataRequest{
			BillingProfile: &apiv3.BillingProfileReference{Id: profile.Id},
		})
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
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

		status, meter, problem := c.CreateMeter(apiv3.CreateMeterRequest{
			Key:         uniqueKey("deleteinv_meter"),
			Name:        gofakeit.ProductName(),
			Aggregation: apiv3.MeterAggregationCount,
			EventType:   uniqueKey("deleteinv_event"),
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)

		status, f, problem := c.CreateFeature(apiv3.CreateFeatureRequest{
			Key:   uniqueKey("deleteinv_feature"),
			Name:  gofakeit.ProductName(),
			Meter: &apiv3.FeatureMeterReference{Id: meter.Id},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		feature = f

		status, plan, problem := c.CreatePlan(apiv3.CreatePlanRequest{
			Key:            uniqueKey("deleteinv_plan"),
			Name:           gofakeit.ProductName(),
			Currency:       "USD",
			BillingCadence: apiv3.ISO8601Duration("P1M"),
			Phases: []apiv3.BillingPlanPhase{{
				Key:       uniqueKey("inv_phase_1"),
				Name:      uniqueKey("Test Phase"),
				RateCards: []apiv3.BillingRateCard{validUnitRateCard(*feature)},
			}},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		planID = plan.Id

		status, plan, problem = c.PublishPlan(planID)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		assert.Equal(t, apiv3.BillingPlanStatusActive, plan.Status)

		status, _, problem = c.CreateSubscription(apiv3.BillingSubscriptionCreate{
			Customer: struct {
				Id  *apiv3.ULID                `json:"id,omitempty"`
				Key *apiv3.ExternalResourceKey `json:"key,omitempty"`
			}{Id: lo.ToPtr(customerID)},
			Plan: struct {
				Id      *apiv3.ULID        `json:"id,omitempty"`
				Key     *apiv3.ResourceKey `json:"key,omitempty"`
				Version *int               `json:"version,omitempty"`
			}{Id: lo.ToPtr(planID)},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
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

		status, problem := c.DeleteBillingInvoice(invoiceID)
		require.Equal(t, http.StatusNoContent, status, "problem: %+v", problem)
	})

	t.Run("Should return 400 when deleting the same invoice again", func(t *testing.T) {
		require.NotEmpty(t, invoiceID, "depends on invoice deletion")

		status, problem := c.DeleteBillingInvoice(invoiceID)
		assert.Equal(t, http.StatusBadRequest, status, "problem: %+v", problem)
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 when deleting a gathering invoice via v3 DELETE", func(t *testing.T) {
		if gatheringInvoiceID == "" {
			t.Skip("no gathering invoice found in the list; skipping")
		}

		status, problem := c.DeleteBillingInvoice(gatheringInvoiceID)
		assert.Equal(t, http.StatusNotFound, status, "problem: %+v", problem)
		assert.NotNil(t, problem)
	})

	t.Run("Should return 404 for an unknown invoice ID", func(t *testing.T) {
		status, problem := c.DeleteBillingInvoice("01JAAAAAAAAAAAAAAAAAAAAAAA")
		assert.Equal(t, http.StatusNotFound, status)
		assert.NotNil(t, problem)
	})
}
