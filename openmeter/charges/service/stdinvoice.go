package service

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/charges"
)

func (s *service) handleStandardInvoiceUpdate(ctx context.Context, invoice billing.StandardInvoice) error {
	if invoice.Status == billing.StandardInvoiceStatusPaymentProcessingPending {
		return s.handlePaymentProcessingPending(ctx, invoice)
	}

	if invoice.Status == billing.StandardInvoiceStatusPaid {
		return s.handlePaymentProcessingSettled(ctx, invoice)
	}

	return nil
}

func (s *service) handlePaymentProcessingPending(ctx context.Context, invoice billing.StandardInvoice) error {
	return s.handleStandardInvoiceRealizations(ctx, invoice, func(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) error {
		if realization.Status == charges.StandardInvoiceRealizationStatusDraft {
			realization.Status = charges.StandardInvoiceRealizationStatusAuthorized

			charge, err := s.updateStandardInvoiceRealizationByID(ctx, charge, realization.StandardInvoiceRealization)
			if err != nil {
				return err
			}

			_, err = s.handler.OnStandardInvoiceRealizationAuthorized(ctx, charge, realization)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *service) handlePaymentProcessingSettled(ctx context.Context, invoice billing.StandardInvoice) error {
	return s.handleStandardInvoiceRealizations(ctx, invoice, func(ctx context.Context, charge charges.Charge, realization charges.StandardInvoiceRealizationWithLine) error {
		if realization.Status == charges.StandardInvoiceRealizationStatusAuthorized {
			realization.Status = charges.StandardInvoiceRealizationStatusSettled

			charge, err := s.updateStandardInvoiceRealizationByID(ctx, charge, realization.StandardInvoiceRealization)
			if err != nil {
				return err
			}

			_, err = s.handler.OnStandardInvoiceRealizationSettled(ctx, charge, realization)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
