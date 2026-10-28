package billingworkercollect

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

type gatedCollectionBillingService struct {
	billing.Service
}

func (gatedCollectionBillingService) InvoicePendingLines(context.Context, billing.InvoicePendingLinesInput, ...billing.InvoicePendingLinesOption) ([]billing.StandardInvoice, error) {
	return nil, fmt.Errorf("collection deferred: %w", billing.ValidationError{Err: billing.ErrInvoiceCreateAllLinesGated})
}

func TestInvoiceCollectorDefersFullyGatedCollection(t *testing.T) {
	// given: a fully gated collection has already recorded its diagnostics.
	collector := &InvoiceCollector{
		billingService: gatedCollectionBillingService{},
		logger:         slog.Default(),
	}

	// when: the automatic collector encounters that deferred outcome.
	invoices, err := collector.CollectCustomerInvoice(t.Context(), CollectCustomerInvoiceInput{
		CustomerID: customer.CustomerID{Namespace: "ns", ID: "customer-id"},
		AsOf:       time.Now(),
	})
	require.NoError(t, err)

	// then: it neither fails nor calls gathering-invoice recalculation.
	require.Empty(t, invoices)
}

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
