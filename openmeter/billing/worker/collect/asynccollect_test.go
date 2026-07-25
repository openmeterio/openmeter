package billingworkercollect

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

func TestInvoiceCollectorHandleCollectCustomerInvoicesEvent(t *testing.T) {
	collector := &InvoiceCollector{}

	t.Run("nil event", func(t *testing.T) {
		require.NoError(t, collector.HandleCollectCustomerInvoicesEvent(t.Context(), nil))
	})

	t.Run("invalid event", func(t *testing.T) {
		err := collector.HandleCollectCustomerInvoicesEvent(t.Context(), &billing.CollectCustomerInvoicesEvent{
			Namespace:  "ns",
			CustomerID: "customer-id",
		})

		require.ErrorContains(t, err, "invalid collect customer invoices event")
		require.ErrorContains(t, err, "as_of cannot be zero")
	})
}
