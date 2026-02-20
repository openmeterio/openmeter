package service

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/samber/lo"
)

func (s *service) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	// TODO: handle lockdown namespaces!
	// if slices.Contains(s.fsNamespaceLockdown, input.Customer.Namespace) {
	//	return nil, billing.ValidationError{
	//		Err: fmt.Errorf("%w: %s", billing.ErrNamespaceLocked, input.Customer.Namespace),
	//	}
	// }

	return withBillingServiceLock(ctx, s, input.Customer, func(ctx context.Context) ([]billing.StandardInvoice, error) {
		billableLines, err := s.billingService.PrepareBillableLines(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("preparing billable lines: %w", err)
		}

		if billableLines == nil {
			// Should not happen, but we want to be defensive, but we are not surfacing this error to the caller.
			return nil, fmt.Errorf("billable lines are nil")
		}

		createdInvoices := make([]billing.StandardInvoice, 0, len(billableLines.LinesByCurrency))

		for currency, inScopeLines := range billableLines.LinesByCurrency {
			createdInvoice, err := s.billingService.CreateStandardInvoiceFromGatheringLines(ctx, billing.CreateStandardInvoiceFromGatheringLinesInput{
				Customer: input.Customer,
				Currency: currency,
				Lines:    inScopeLines,
			})
			if err != nil {
				return nil, fmt.Errorf("creating standard invoice from gathering lines: %w", err)
			}

			createdInvoices = append(createdInvoices, *createdInvoice)
		}

		// TODO: here we need to be able to update the lines before they are associated with the standard invoice
		// to add any credit lines

		linesWithCharges, err := s.getLinesWithChargesForStandardInvoice(ctx, input.Customer.Namespace, createdInvoices...)
		if err != nil {
			return nil, err
		}

		for _, line := range linesWithCharges {
			_, err := s.handleNewStandardLineCreation(ctx, line)
			if err != nil {
				return nil, err
			}
		}

		return createdInvoices, nil
	})
}

func (s *service) handleNewStandardLineCreation(ctx context.Context, in standardLineWithCharge) (charges.Charge, error) {
	charge, realization, err := s.addStandardInvoiceRealization(ctx, in.Charge, in.StandardLineWithInvoiceHeader)
	if err != nil {
		return charge, err
	}

	charge, err = s.handler.OnStandardInvoiceRealizationCreated(ctx, charge, charges.StandardInvoiceRealizationWithLine{
		StandardInvoiceRealization:    realization,
		StandardLineWithInvoiceHeader: in.StandardLineWithInvoiceHeader,
	})
	if err != nil {
		return charge, err
	}

	return charge, nil
}

func withBillingServiceLock[T any](ctx context.Context, s *service, customerID billing.CustomerID, fn func(ctx context.Context) (T, error)) (T, error) {
	var out T

	err := s.billingService.WithLock(ctx, customerID, func(ctx context.Context) error {
		var err error
		out, err = fn(ctx)
		if err != nil {
			return err
		}

		return err
	})
	if err != nil {
		return lo.Empty[T](), err
	}

	return out, nil
}
