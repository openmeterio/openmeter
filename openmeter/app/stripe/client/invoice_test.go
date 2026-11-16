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
