package billing_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/event/metadata"
)

func TestCollectCustomerInvoicesEvent(t *testing.T) {
	asOf := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	event := billing.CollectCustomerInvoicesEvent{
		Namespace:  "ns",
		CustomerID: "customer-id",
		AsOf:       asOf,
	}

	require.NoError(t, event.Validate())
	require.Equal(t, metadata.GetEventName(metadata.EventType{
		Subsystem: billing.EventSubsystem,
		Name:      "invoice.collect",
		Version:   "v1",
	}), event.EventName())

	eventMetadata := event.EventMetadata()
	require.Equal(t, metadata.ComposeResourcePath("ns", metadata.EntityCustomer, "customer-id"), eventMetadata.Source)
	require.Equal(t, metadata.ComposeResourcePath("ns", metadata.EntityCustomer, "customer-id"), eventMetadata.Subject)
}

func TestCollectCustomerInvoicesEventValidate(t *testing.T) {
	asOf := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		event   billing.CollectCustomerInvoicesEvent
		wantErr string
	}{
		{
			name: "namespace is required",
			event: billing.CollectCustomerInvoicesEvent{
				CustomerID: "customer-id",
				AsOf:       asOf,
			},
			wantErr: "namespace cannot be empty",
		},
		{
			name: "customer id is required",
			event: billing.CollectCustomerInvoicesEvent{
				Namespace: "ns",
				AsOf:      asOf,
			},
			wantErr: "customer_id cannot be empty",
		},
		{
			name: "as of is required",
			event: billing.CollectCustomerInvoicesEvent{
				Namespace:  "ns",
				CustomerID: "customer-id",
			},
			wantErr: "as_of cannot be zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.ErrorContains(t, tt.event.Validate(), tt.wantErr)
		})
	}
}
