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

func formatInvoiceLinesForLog(lines []api.InvoiceLine) string {
	return formatLogJSON(lines)
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
