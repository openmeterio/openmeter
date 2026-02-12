package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ billing.StandardInvoiceService = (*Service)(nil)

func (s *Service) UpdateStandardInvoice(ctx context.Context, input billing.UpdateStandardInvoiceInput) (billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil {
		return billing.StandardInvoice{}, billing.ValidationError{
			Err: err,
		}
	}

	invoice, err := s.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
		Invoice: input.Invoice,
		Expand:  billing.InvoiceExpand{}, // We don't want to expand anything as we will have to refetch the invoice anyway
	})
	if err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("fetching invoice: %w", err)
	}

	if invoice.Status == billing.StandardInvoiceStatusGathering {
		return billing.StandardInvoice{}, billing.ValidationError{
			Err: fmt.Errorf("invoice[%s] is a gathering invoice, cannot be updated via the standard invoice service", invoice.ID),
		}
	}

	return s.executeTriggerOnInvoice(
		ctx,
		input.Invoice,
		billing.TriggerUpdated,
		ExecuteTriggerWithIncludeDeletedLines(input.IncludeDeletedLines),
		ExecuteTriggerWithAllowInStates(billing.StandardInvoiceStatusDraftUpdating),
		ExecuteTriggerWithEditCallback(func(sm *InvoiceStateMachine) error {
			if err := input.EditFn(&sm.Invoice); err != nil {
				return fmt.Errorf("editing invoice: %w", err)
			}

			if err := sm.Invoice.Validate(); err != nil {
				return billing.ValidationError{
					Err: err,
				}
			}

			return nil
		}),
	)
}

func (s *Service) GetStandardInvoiceById(ctx context.Context, input billing.GetStandardInvoiceByIdInput) (billing.StandardInvoice, error) {
	invoice, err := s.adapter.GetInvoiceById(ctx, input)
	if err != nil {
		return billing.StandardInvoice{}, err
	}

	return invoice, nil
}

func (s *Service) ListStandardInvoices(ctx context.Context, input billing.ListStandardInvoicesInput) (billing.ListStandardInvoicesResponse, error) {
	invoices, err := s.adapter.ListInvoices(ctx, input)
	if err != nil {
		return billing.ListStandardInvoicesResponse{}, err
	}

	updatedInvoices, err := s.emulateStandardInvoicesGatheringInvoiceFields(ctx, invoices.Items)
	if err != nil {
		return billing.ListInvoicesResponse{}, fmt.Errorf("error emulating standard invoices gathering invoice fields: %w", err)
	}

	invoices.Items = updatedInvoices

	for i := range invoices.Items {
		invoiceID := invoices.Items[i].ID

		invoices.Items[i], err = s.resolveWorkflowApps(ctx, invoices.Items[i])
		if err != nil {
			return billing.ListInvoicesResponse{}, fmt.Errorf("error resolving workflow apps [%s]: %w", invoiceID, err)
		}

		invoices.Items[i], err = s.resolveStatusDetails(ctx, invoices.Items[i])
		if err != nil {
			return billing.ListInvoicesResponse{}, fmt.Errorf("error resolving status details for invoice [%s]: %w", invoiceID, err)
		}

		if input.Expand.RecalculateGatheringInvoice {
			invoices.Items[i], err = s.recalculateGatheringInvoice(ctx, recalculateGatheringInvoiceInput{
				Invoice: invoices.Items[i],
				Expand:  input.Expand,
			})
			if err != nil {
				return billing.ListInvoicesResponse{}, fmt.Errorf("error recalculating gathering invoice [%s]: %w", invoiceID, err)
			}
		}
	}

	return invoices, nil
}
