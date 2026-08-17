package billingworkercollect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

// raceBillingService releases all collection calls together. This makes the
// unsynchronized writes to the collector's shared err variable overlap under
// the race detector instead of relying on scheduler timing.
type raceBillingService struct {
	billing.Service

	entered atomic.Int32
	ready   chan struct{}
	total   int32
}

func (s *raceBillingService) InvoicePendingLines(
	context.Context,
	billing.InvoicePendingLinesInput,
	...billing.InvoicePendingLinesOption,
) ([]billing.StandardInvoice, error) {
	if s.entered.Add(1) == s.total {
		close(s.ready)
	}
	<-s.ready

	return nil, errors.New("synthetic invoice failure")
}

type raceGatheringInvoiceService struct {
	billing.GatheringInvoiceService

	customers []customer.CustomerID
}

func (s *raceGatheringInvoiceService) ListCustomerIDsPendingCollection(
	context.Context,
	billing.ListCustomerIDsPendingCollectionInput,
) ([]customer.CustomerID, error) {
	return s.customers, nil
}

func TestInvoiceCollectorAllReportsConcurrentCollectionErrors(t *testing.T) {
	const customerCount = 32

	customers := make([]customer.CustomerID, customerCount)
	for index := range customers {
		customers[index] = customer.CustomerID{
			Namespace: "default",
			ID:        fmt.Sprintf("customer-%d", index),
		}
	}

	billingService := &raceBillingService{
		ready: make(chan struct{}),
		total: customerCount,
	}
	collector, err := NewInvoiceCollector(Config{
		GatheringInvoiceService: &raceGatheringInvoiceService{customers: customers},
		BillingService:          billingService,
		Logger:                  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	require.NoError(t, err)

	err = collector.All(t.Context(), []string{"default"}, nil, 0)
	require.ErrorContains(t, err, "synthetic invoice failure")
	require.Equal(t, int32(customerCount), billingService.entered.Load())
}
