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

	return s.executeTriggerOnInvoice(
		ctx,
		input.Invoice,
		billing.TriggerUpdated,
		ExecuteTriggerWithIncludeDeletedLines(input.IncludeDeletedLines),
		ExecuteTriggerWithAllowInStates(billing.StandardInvoiceStatusDraftUpdating),
		ExecuteTriggerWithEditCallback(func(ctx context.Context, sm *InvoiceStateMachine) error {
			originalInvoice, err := sm.Invoice.Clone()
			if err != nil {
				return fmt.Errorf("cloning invoice before edit: %w", err)
			}

			if err := input.EditFn(&sm.Invoice); err != nil {
				return fmt.Errorf("editing invoice: %w", err)
			}

			lineDiff, err := s.diffMutableInvoiceLines(ctx, originalInvoice, sm.Invoice, input.ChangeSource)
			if err != nil {
				return billing.ValidationError{
					Err: fmt.Errorf("collecting mutable invoice line changes: %w", err),
				}
			}

			switch input.ChangeSource {
			case billing.ChangeSourceAPIRequest:
				invoiceWithLineEngineChanges, err := s.applyAPIInvoiceLineEdits(ctx, applyAPIInvoiceLineEditsInput{
					EditedInvoice: sm.Invoice,
					LineDiff:      lineDiff,
				})
				if err != nil {
					return fmt.Errorf("applying API standard invoice line edits: %w", err)
				}

				standardInvoice, err := invoiceWithLineEngineChanges.AsInvoice().AsStandardInvoice()
				if err != nil {
					return fmt.Errorf("converting edited invoice to standard invoice: %w", err)
				}
				sm.Invoice = standardInvoice

			case billing.ChangeSourceSystem:
				// System-originated create and update changes are initiated by billing or
				// charges, so there is no extra line-engine notification for them here.
				// Deletes still need the legacy deleted-by-system notification because
				// the charge line updater currently relies on it to clean up realizations.
				if err := s.dispatchSystemStandardLineDeletions(ctx, sm.Invoice, lineDiff.Deleted); err != nil {
					return fmt.Errorf("dispatching system standard line deletions: %w", err)
				}

			default:
				return fmt.Errorf("unsupported change source: %s", input.ChangeSource)
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
	if err := input.Validate(); err != nil {
		return billing.StandardInvoice{}, billing.ValidationError{
			Err: err,
		}
	}

	invoiceType, err := s.adapter.GetInvoiceType(ctx, input.Invoice)
	if err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("getting invoice type: %w", err)
	}

	if invoiceType != billing.InvoiceTypeStandard {
		return billing.StandardInvoice{}, billing.ValidationError{
			Err: fmt.Errorf("invoice[%s] is not a standard invoice, cannot be fetched via the standard invoice service", input.Invoice.ID),
		}
	}

	invoice, err := s.adapter.GetStandardInvoiceById(ctx, input)
	if err != nil {
		return billing.StandardInvoice{}, err
	}

	invoice, err = s.resolveWorkflowApps(ctx, invoice)
	if err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("error resolving workflow apps for invoice [%s]: %w", input.Invoice.ID, err)
	}

	invoice, err = s.resolveStatusDetails(ctx, invoice)
	if err != nil {
		return billing.StandardInvoice{}, fmt.Errorf("error resolving status details for invoice [%s]: %w", input.Invoice.ID, err)
	}

	return invoice, nil
}

func (s *Service) ListStandardInvoices(ctx context.Context, input billing.ListStandardInvoicesInput) (billing.ListStandardInvoicesResponse, error) {
	if err := input.Validate(); err != nil {
		return billing.ListStandardInvoicesResponse{}, billing.ValidationError{
			Err: err,
		}
	}

	resp, err := s.adapter.ListStandardInvoices(ctx, input)
	if err != nil {
		return billing.ListStandardInvoicesResponse{}, fmt.Errorf("listing standard invoices: %w", err)
	}

	return resp, nil
}

func (s *Service) ListStandardInvoicesPendingAdvancement(ctx context.Context, input billing.ListStandardInvoicesPendingAdvancementInput) ([]billing.StandardInvoice, error) {
	if err := input.Validate(); err != nil {
		return nil, billing.ValidationError{Err: err}
	}

	invoices, err := s.adapter.ListStandardInvoicesPendingAdvancement(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("listing standard invoices pending advancement: %w", err)
	}

	return invoices, nil
}
