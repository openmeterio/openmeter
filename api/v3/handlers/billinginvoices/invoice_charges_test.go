package billinginvoices

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

const invoiceChargesCustomerID = "01K3Q8QG4B9YJYJVQ3MZ3V8KT6"

type invoiceChargesBillingService struct {
	billing.Service
	collect func(context.Context, billing.InvoicePendingLinesInput, ...billing.InvoicePendingLinesOption) ([]billing.StandardInvoice, error)
}

func (s invoiceChargesBillingService) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput, opts ...billing.InvoicePendingLinesOption) ([]billing.StandardInvoice, error) {
	return s.collect(ctx, input, opts...)
}

type invoiceChargesServer struct {
	api.Unimplemented
	handler Handler
}

func (s invoiceChargesServer) InvoiceCustomerCharges(w http.ResponseWriter, r *http.Request, customerID api.ULID) {
	s.handler.InvoiceCustomerCharges().With(customerID).ServeHTTP(w, r)
}

func TestInvoiceCustomerChargesManualCollection(t *testing.T) {
	asOf := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		body        string
		asOf        *time.Time
		progressive *bool
	}{
		{
			name: "empty body uses customer settings",
		},
		{
			name: "empty object uses customer settings",
			body: `{}`,
		},
		{
			name:        "cutoff and progressive billing enabled",
			body:        `{"as_of":"2026-01-01T12:00:00Z","progressive_billing_override":true}`,
			asOf:        &asOf,
			progressive: lo.ToPtr(true),
		},
		{
			name:        "progressive billing explicitly disabled",
			body:        `{"progressive_billing_override":false}`,
			progressive: lo.ToPtr(false),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// given: the customer has eligible lines in two currencies.
			called := false
			service := invoiceChargesBillingService{
				collect: func(_ context.Context, input billing.InvoicePendingLinesInput, opts ...billing.InvoicePendingLinesOption) ([]billing.StandardInvoice, error) {
					called = true
					require.Equal(t, customer.CustomerID{Namespace: "ns", ID: invoiceChargesCustomerID}, input.Customer)
					require.Equal(t, test.asOf, input.AsOf)
					require.False(t, input.IncludePendingLines.IsPresent())
					require.False(t, input.ForceAsyncAdvance)
					require.Equal(t, billing.InvoicePendingLinesOptions{
						BypassCollectionAlignment:  true,
						PartialInvoiceLinesEnabled: test.progressive,
					}, billing.NewInvoicePendingLinesOptions(opts...))

					return []billing.StandardInvoice{
						{StandardInvoiceBase: billing.StandardInvoiceBase{ID: "usd-invoice", Currency: "USD", Status: billing.StandardInvoiceStatusDraftCreated}},
						{StandardInvoiceBase: billing.StandardInvoiceBase{ID: "eur-invoice", Currency: "EUR", Status: billing.StandardInvoiceStatusDraftCreated}},
					}, nil
				},
			}
			handler := New(func(context.Context) (string, error) { return "ns", nil }, service)
			router := api.Handler(invoiceChargesServer{handler: handler})

			// when: manually collecting through the generated customer route.
			req := httptest.NewRequest(http.MethodPost, "/openmeter/customers/"+invoiceChargesCustomerID+"/charges/invoice", strings.NewReader(test.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)

			// then: both standard invoices are returned after forwarding the manual options.
			require.Equal(t, http.StatusCreated, res.Code, res.Body.String())
			require.True(t, called)
			var invoices []api.BillingInvoice
			require.NoError(t, json.Unmarshal(res.Body.Bytes(), &invoices))
			require.Len(t, invoices, 2)
			usdInvoice, err := invoices[0].AsBillingInvoiceStandard()
			require.NoError(t, err)
			require.Equal(t, "usd-invoice", usdInvoice.Id)
			require.Equal(t, api.CurrencyCode("USD"), usdInvoice.Currency)
			eurInvoice, err := invoices[1].AsBillingInvoiceStandard()
			require.NoError(t, err)
			require.Equal(t, api.CurrencyCode("EUR"), eurInvoice.Currency)
		})
	}
}

func TestInvoiceCustomerChargesAllBlockedReturnsEmptyArray(t *testing.T) {
	handler := New(func(context.Context) (string, error) { return "ns", nil }, invoiceChargesBillingService{
		collect: func(context.Context, billing.InvoicePendingLinesInput, ...billing.InvoicePendingLinesOption) ([]billing.StandardInvoice, error) {
			return nil, nil
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	res := httptest.NewRecorder()

	handler.InvoiceCustomerCharges().With(invoiceChargesCustomerID).ServeHTTP(res, req)

	require.Equal(t, http.StatusCreated, res.Code, res.Body.String())
	require.JSONEq(t, `[]`, res.Body.String())
}

func TestInvoiceCustomerChargesErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		serviceErr error
		status     int
		called     bool
	}{
		{
			name:   "malformed JSON",
			body:   `{`,
			status: http.StatusBadRequest,
		},
		{
			name:   "malformed cutoff",
			body:   `{"as_of":"yesterday"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "future cutoff",
			body:   `{"as_of":"9999-01-01T00:00:00Z"}`,
			status: http.StatusBadRequest,
			called: true,
		},
		{
			name:       "no eligible lines",
			serviceErr: billing.ValidationError{Err: billing.ErrInvoiceCreateNoLines},
			status:     http.StatusBadRequest,
			called:     true,
		},
		{
			name:       "line not found",
			serviceErr: billing.NotFoundError{Entity: billing.EntityInvoiceLine, ID: "missing-line"},
			status:     http.StatusNotFound,
			called:     true,
		},
		{
			name:       "service failure",
			serviceErr: errors.New("collection failed"),
			status:     http.StatusInternalServerError,
			called:     true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := New(func(context.Context) (string, error) { return "ns", nil }, invoiceChargesBillingService{
				collect: func(_ context.Context, input billing.InvoicePendingLinesInput, _ ...billing.InvoicePendingLinesOption) ([]billing.StandardInvoice, error) {
					called = true
					if err := input.Validate(); err != nil {
						return nil, billing.ValidationError{Err: err}
					}

					return nil, test.serviceErr
				},
			})
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			res := httptest.NewRecorder()

			handler.InvoiceCustomerCharges().With(invoiceChargesCustomerID).ServeHTTP(res, req)

			require.Equal(t, test.status, res.Code, res.Body.String())
			require.Equal(t, test.called, called)
		})
	}
}
