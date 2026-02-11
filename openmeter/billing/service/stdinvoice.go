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
