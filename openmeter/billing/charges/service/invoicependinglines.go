package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) ([]billing.StandardInvoice, error) {
		createdInvoices, err := s.billingService.InvoicePendingLines(ctx, input)
		if err != nil {
			return nil, err
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
