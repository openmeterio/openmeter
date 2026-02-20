package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/slicesx"
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

	adapterInput := billing.ListInvoicesAdapterInput{
		Page:               input.Page,
		Namespaces:         input.Namespaces,
		IDs:                input.IDs,
		Statuses:           input.Statuses,
		ExtendedStatuses:   input.ExtendedStatuses,
		HasAvailableAction: input.HasAvailableAction,

		ExternalIDs:     input.ExternalIDs,
		DraftUntilLTE:   input.DraftUntilLTE,
		CollectionAtLTE: input.CollectionAtLTE,
		IncludeDeleted:  input.IncludeDeleted,

		Expand: billing.InvoiceExpands{}.
			SetOrUnsetIf(input.Expand.Has(billing.StandardInvoiceExpandLines), billing.InvoiceExpandLines).
			SetOrUnsetIf(input.Expand.Has(billing.StandardInvoiceExpandDeletedLines), billing.InvoiceExpandDeletedLines),
		OnlyStandard: true,
	}

	resp, err := s.adapter.ListInvoices(ctx, adapterInput)
	if err != nil {
		return billing.ListStandardInvoicesResponse{}, fmt.Errorf("listing invoices: %w", err)
	}

	stdInvoices, err := slicesx.MapWithErr(resp.Items, func(item billing.Invoice) (billing.StandardInvoice, error) {
		return item.AsStandardInvoice()
	})
	if err != nil {
		return billing.ListStandardInvoicesResponse{}, fmt.Errorf("mapping invoices to standard invoices: %w", err)
	}

	return billing.ListStandardInvoicesResponse{
		Items:      stdInvoices,
		Page:       resp.Page,
		TotalCount: resp.TotalCount,
	}, nil
}

func (s *Service) RegisterStandardInvoiceHooks(hooks ...billing.StandardInvoiceHook) {
	s.standardInvoiceHooks.RegisterHooks(hooks...)
}
