package client

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"
	stripeapi "github.com/stripe/stripe-go/v80/client"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

// TestRecreateOmittedDraftInvoice isolates the client calls made when a synced draft
// becomes empty and then billable again. Billing's existing lifecycle tests mock
// CreateInvoice above the HTTP boundary and therefore cannot catch a cached response.
func TestRecreateOmittedDraftInvoice(t *testing.T) {
	var mu sync.Mutex
	creationResponses := map[string]string{}
	liveInvoices := map[string]bool{}
	invoiceItems := map[string][]string{}
	invoiceCount := 0
	itemCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/invoices":
			key := r.Header.Get("Idempotency-Key")
			response, cached := creationResponses[key]
			if !cached || key == "" {
				invoiceCount++
				invoiceID := fmt.Sprintf("in_%d", invoiceCount)
				liveInvoices[invoiceID] = true
				response = fmt.Sprintf(`{"id":%q,"object":"invoice","status":"draft"}`, invoiceID)
				creationResponses[key] = response
			}

			t.Logf("create: key=%q cached=%t response=%s", key, cached, response)
			_, _ = w.Write([]byte(response))

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v1/invoices/"):
			invoiceID := strings.TrimPrefix(r.URL.Path, "/v1/invoices/")
			delete(liveInvoices, invoiceID)
			delete(invoiceItems, invoiceID)
			// Stripe keeps API v1 idempotency responses after resource deletion.
			// https://docs.stripe.com/api/idempotent_requests
			_, _ = fmt.Fprintf(w, `{"id":%q,"object":"invoice","deleted":true}`, invoiceID)

		case r.Method == http.MethodPost && r.URL.Path == "/v1/invoiceitems":
			invoiceID := r.Form.Get("invoice")
			if !liveInvoices[invoiceID] {
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprintf(w, `{"error":{"type":"invalid_request_error","code":"resource_missing","message":"No such invoice: '%s'"}}`, invoiceID)

				return
			}

			itemCount++
			itemID := fmt.Sprintf("ii_%d", itemCount)
			invoiceItems[invoiceID] = append(invoiceItems[invoiceID], itemID)
			_, _ = fmt.Fprintf(w, `{"id":%q,"object":"invoiceitem"}`, itemID)

		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/lines"):
			invoiceID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/invoices/"), "/lines")
			lines := lo.Map(invoiceItems[invoiceID], func(itemID string, _ int) string {
				return fmt.Sprintf(`{"id":%q,"object":"line_item","invoice_item":%q}`, "il_"+itemID, itemID)
			})
			_, _ = fmt.Fprintf(w, `{"object":"list","has_more":false,"data":[%s]}`, strings.Join(lines, ","))

		default:
			http.Error(w, "unexpected request: "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	defer server.Close()

	backend := stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
		URL:               lo.ToPtr(server.URL),
		MaxNetworkRetries: lo.ToPtr(int64(0)),
	})
	api := &stripeapi.API{}
	api.Init("sk_test_123", &stripe.Backends{API: backend, Connect: backend, Uploads: backend})

	stripeClient := &stripeAppClient{
		appID:  app.AppID{Namespace: "ns", ID: "app"},
		client: api,
		logger: slog.New(slog.DiscardHandler),
	}
	input := CreateInvoiceInput{
		AppID:            stripeClient.appID,
		CustomerID:       customer.CustomerID{Namespace: "ns", ID: "customer"},
		InvoiceID:        "openmeter_invoice",
		CollectionMethod: billing.CollectionMethodChargeAutomatically,
		Currency:         currencyx.FiatCode("USD"),
		StripeCustomerID: "cus_123",
	}

	// given: a billable draft has synced to Stripe; retrying creation must reuse it.
	originalInvoice, err := stripeClient.CreateInvoice(t.Context(), input)
	require.NoError(t, err)

	originalLines, err := stripeClient.AddInvoiceLines(t.Context(), AddInvoiceLinesInput{
		StripeInvoiceID: originalInvoice.ID,
		Lines: []*stripe.InvoiceItemParams{
			{
				Customer: lo.ToPtr(input.StripeCustomerID),
				Currency: lo.ToPtr("usd"),
				Amount:   lo.ToPtr(int64(1000)),
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, originalLines, 1)

	retriedInvoice, err := stripeClient.CreateInvoice(t.Context(), input)
	require.NoError(t, err)
	require.Equal(t, originalInvoice.ID, retriedInvoice.ID, "a creation retry must not duplicate the draft")

	// when: omit deletes the empty draft, then an edit makes the same OpenMeter invoice billable again.
	err = stripeClient.DeleteInvoice(t.Context(), DeleteInvoiceInput{StripeInvoiceID: originalInvoice.ID})
	require.NoError(t, err)

	recreatedInvoice, err := stripeClient.CreateInvoice(t.Context(), input)
	require.NoError(t, err)

	// then: the new billable line must be accepted on a fresh Stripe invoice.
	recreatedLines, err := stripeClient.AddInvoiceLines(t.Context(), AddInvoiceLinesInput{
		StripeInvoiceID: recreatedInvoice.ID,
		Lines: []*stripe.InvoiceItemParams{
			{
				Customer: lo.ToPtr(input.StripeCustomerID),
				Currency: lo.ToPtr("usd"),
				Amount:   lo.ToPtr(int64(2000)),
			},
		},
	})
	require.NoError(t, err, "a billable edit after omission must not target deleted Stripe invoice %s (recreated ID: %s)", originalInvoice.ID, recreatedInvoice.ID)
	require.Len(t, recreatedLines, 1)
	require.NotEqual(t, originalInvoice.ID, recreatedInvoice.ID)
}
