package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

func TestInvoicePendingLinesCollectsCappedBatchesUntilEmpty(t *testing.T) {
	billingService := &invoicePendingLinesBillingService{
		results: []invoicePendingLinesResult{
			{invoices: []billing.StandardInvoice{{}}},
			{invoices: []billing.StandardInvoice{{}}},
			{err: billing.ErrInvoiceCreateNoLines},
		},
	}
	service := &Service{
		billingService: billingService,
		featureFlags: FeatureFlags{
			MaxLinesPerCollectedInvoice: 1,
		},
		tracer: noop.NewTracerProvider().Tracer("test"),
	}

	err := service.invoicePendingLines(t.Context(), customer.CustomerID{
		Namespace: "ns",
		ID:        "customer-id",
	})

	require.NoError(t, err)
	require.Len(t, billingService.calls, 3)

	for _, call := range billingService.calls {
		require.Equal(t, customer.CustomerID{Namespace: "ns", ID: "customer-id"}, call.input.Customer)
		require.Equal(t, 1, call.options.MaxLinesPerInvoice)
		require.NotNil(t, call.options.PartialInvoiceLinesEnabled)
		require.False(t, *call.options.PartialInvoiceLinesEnabled)
	}
}

func TestInvoicePendingLinesUnlimitedCollectsOnce(t *testing.T) {
	billingService := &invoicePendingLinesBillingService{
		results: []invoicePendingLinesResult{
			{invoices: []billing.StandardInvoice{{}}},
			{err: billing.ErrInvoiceCreateNoLines},
		},
	}
	service := &Service{
		billingService: billingService,
		tracer:         noop.NewTracerProvider().Tracer("test"),
	}

	err := service.invoicePendingLines(t.Context(), customer.CustomerID{
		Namespace: "ns",
		ID:        "customer-id",
	})

	require.NoError(t, err)
	require.Len(t, billingService.calls, 1)
	require.Equal(t, 0, billingService.calls[0].options.MaxLinesPerInvoice)
}

func TestInvoicePendingLinesExitsOnEmptySuccess(t *testing.T) {
	billingService := &invoicePendingLinesBillingService{
		results: []invoicePendingLinesResult{
			{invoices: []billing.StandardInvoice{}},
			{err: billing.ErrInvoiceCreateNoLines},
		},
	}
	service := &Service{
		billingService: billingService,
		featureFlags: FeatureFlags{
			MaxLinesPerCollectedInvoice: 1,
		},
		tracer: noop.NewTracerProvider().Tracer("test"),
	}

	err := service.invoicePendingLines(t.Context(), customer.CustomerID{
		Namespace: "ns",
		ID:        "customer-id",
	})

	require.NoError(t, err)
	require.Len(t, billingService.calls, 1)
	require.Equal(t, 1, billingService.calls[0].options.MaxLinesPerInvoice)
}

type invoicePendingLinesBillingService struct {
	billing.Service

	results []invoicePendingLinesResult
	calls   []invoicePendingLinesCall
}

type invoicePendingLinesResult struct {
	invoices []billing.StandardInvoice
	err      error
}

type invoicePendingLinesCall struct {
	input   billing.InvoicePendingLinesInput
	options billing.InvoicePendingLinesOptions
}

func (s *invoicePendingLinesBillingService) InvoicePendingLines(
	_ context.Context,
	input billing.InvoicePendingLinesInput,
	opts ...billing.InvoicePendingLinesOption,
) ([]billing.StandardInvoice, error) {
	s.calls = append(s.calls, invoicePendingLinesCall{
		input:   input,
		options: billing.NewInvoicePendingLinesOptions(opts...),
	})

	if len(s.calls) > len(s.results) {
		return nil, billing.ErrInvoiceCreateNoLines
	}

	result := s.results[len(s.calls)-1]
	return result.invoices, result.err
}
