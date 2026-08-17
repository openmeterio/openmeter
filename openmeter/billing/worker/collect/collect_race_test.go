package billingworkercollect

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

// concurrentErrorBillingService releases all collection calls together. This
// makes the shared error in InvoiceCollector.All observable without relying
// exclusively on the race detector.
type concurrentErrorBillingService struct {
	billing.Service

	entered atomic.Int32
	ready   chan struct{}
	total   int32
}

func (s *concurrentErrorBillingService) InvoicePendingLines(
	_ context.Context,
	input billing.InvoicePendingLinesInput,
	_ ...billing.InvoicePendingLinesOption,
) ([]billing.StandardInvoice, error) {
	if s.entered.Add(1) == s.total {
		close(s.ready)
	}
	<-s.ready

	return nil, fmt.Errorf("synthetic invoice failure for %s", input.Customer.ID)
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

func TestInvoiceCollectorAllPreservesConcurrentCollectionErrors(t *testing.T) {
	const (
		customerCount = 128
		attempts      = 10
	)

	customers := make([]customer.CustomerID, customerCount)
	for index := range customers {
		customers[index] = customer.CustomerID{
			Namespace: "default",
			ID:        fmt.Sprintf("customer-%d", index),
		}
	}

	for attempt := range attempts {
		t.Run(fmt.Sprintf("attempt-%d", attempt), func(t *testing.T) {
			billingService := &concurrentErrorBillingService{
				ready: make(chan struct{}),
				total: customerCount,
			}

			collector, err := NewInvoiceCollector(Config{
				GatheringInvoiceService: &raceGatheringInvoiceService{
					customers: customers,
				},
				BillingService: billingService,
				Logger: slog.New(
					slog.NewTextHandler(io.Discard, nil),
				),
			})
			require.NoError(t, err)

			err = collector.All(
				t.Context(),
				[]string{"default"},
				nil,
				0,
			)
			require.Error(t, err)

			lines := strings.Split(err.Error(), "\n")
			require.Len(t, lines, customerCount)

			for _, customer := range customers {
				prefix := fmt.Sprintf(
					"failed to collect invoice for customer [namespace=%s customer=%s]:",
					customer.Namespace,
					customer.ID,
				)

				var customerLine string
				for _, line := range lines {
					if strings.HasPrefix(line, prefix) {
						customerLine = line
						break
					}
				}

				require.NotEmptyf(
					t,
					customerLine,
					"missing collection error for %s",
					customer.ID,
				)
				require.Contains(
					t,
					customerLine,
					"synthetic invoice failure for "+customer.ID,
				)
			}
		})
	}
}
