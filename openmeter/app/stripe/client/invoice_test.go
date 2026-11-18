package client

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v80"
	stripeapi "github.com/stripe/stripe-go/v80/client"

	app "github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

func TestDeleteInvoice(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		responseBody string
		wantErr      bool
	}{
		{
			name:         "deleted",
			status:       http.StatusOK,
			responseBody: `{"id":"in_123","object":"invoice","deleted":true}`,
		},
		{
			name:         "already deleted",
			status:       http.StatusNotFound,
			responseBody: `{"error":{"type":"invalid_request_error","code":"resource_missing","message":"No such invoice: 'in_123'"}}`,
		},
		{
			name:         "not a draft",
			status:       http.StatusBadRequest,
			responseBody: `{"error":{"type":"invalid_request_error","message":"You can only delete draft invoices."}}`,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.responseBody))
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

			err := stripeClient.DeleteInvoice(t.Context(), DeleteInvoiceInput{StripeInvoiceID: "in_123"})

			require.Equal(t, []string{"DELETE /v1/invoices/in_123"}, requests)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCreateInvoiceIdempotencyKey(t *testing.T) {
	tests := []struct {
		name       string
		generation int
		wantKey    string
	}{
		{
			// Has to stay the key used before generations existed.
			name:       "first generation",
			generation: 0,
			wantKey:    "invoice-create-inv_123",
		},
		{
			name:       "later generation",
			generation: 2,
			wantKey:    "invoice-create-inv_123-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var idempotencyKeys []string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				idempotencyKeys = append(idempotencyKeys, r.Header.Get("Idempotency-Key"))

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"in_123","object":"invoice","status":"draft"}`))
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

			_, err := stripeClient.CreateInvoice(t.Context(), CreateInvoiceInput{
				AppID:            app.AppID{Namespace: "ns", ID: "app"},
				CustomerID:       customer.CustomerID{Namespace: "ns", ID: "cus"},
				InvoiceID:        "inv_123",
				Generation:       tt.generation,
				CollectionMethod: billing.CollectionMethodChargeAutomatically,
				Currency:         "USD",
				StripeCustomerID: "cus_123",
			})
			require.NoError(t, err)
			require.Equal(t, []string{tt.wantKey}, idempotencyKeys)
		})
	}
}

func TestGetInvoiceNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","code":"resource_missing","message":"No such invoice: 'in_123'"}}`))
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

	_, err := stripeClient.GetInvoice(t.Context(), GetInvoiceInput{StripeInvoiceID: "in_123"})

	require.True(t, IsStripeInvoiceNotFoundError(err))

	var providerErr *app.AppProviderError
	require.ErrorAs(t, err, &providerErr)
}

func TestCreateInvoiceRejected(t *testing.T) {
	tests := []struct {
		name         string
		answer       bool
		wantRejected bool
	}{
		{
			name:         "stripe answers with an error",
			answer:       true,
			wantRejected: true,
		},
		{
			name: "no answer arrives",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":{"type":"invalid_request_error","code":"rate_limit","message":"Too many requests"}}`))
			}))
			defer server.Close()

			if !tt.answer {
				server.Close()
			}

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

			_, err := stripeClient.CreateInvoice(t.Context(), CreateInvoiceInput{
				AppID:            app.AppID{Namespace: "ns", ID: "app"},
				CustomerID:       customer.CustomerID{Namespace: "ns", ID: "cus"},
				InvoiceID:        "inv_123",
				CollectionMethod: billing.CollectionMethodChargeAutomatically,
				Currency:         "USD",
				StripeCustomerID: "cus_123",
			})

			require.Error(t, err)
			require.Equal(t, tt.wantRejected, IsStripeInvoiceCreateRejectedError(err))
		})
	}
}
