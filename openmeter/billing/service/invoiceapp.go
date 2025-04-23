package billingservice

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

var _ billing.InvoiceAppService = (*Service)(nil)

func (s *Service) TriggerInvoice(ctx context.Context, input billing.InvoiceTriggerServiceInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		_, err := s.withLockedInvoiceStateMachine(ctx, withLockedStateMachineInput{
			InvoiceID: input.Invoice,
			Callback: func(ctx context.Context, sm *InvoiceStateMachine) error {
				errOrValidationErrors := sm.HandleInvoiceTrigger(ctx, input.InvoiceTriggerInput)

				op := billing.InvoiceOpTriggerInvoice
				if input.ValidationErrors != nil {
					op = input.ValidationErrors.Operation
				}

				component := billing.AppTypeCapabilityToComponent(
					input.AppType,
					input.Capability,
					op,
				)

				remainingErrors := sm.Invoice.MergeValidationIssues(
					billing.ValidationWithComponent(component, errOrValidationErrors),
					component,
				)

				if remainingErrors != nil {
					return remainingErrors
				}

				_, err := s.adapter.UpdateInvoice(ctx, sm.Invoice)
				return err
			},
		})

		return err
	})
}

func (s *Service) UpdateInvoiceFields(ctx context.Context, input billing.UpdateInvoiceFieldsInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
		return s.adapter.UpdateInvoiceFields(ctx, input)
	})
}

type syncEditInvoiceInput struct {
	SyncInput             billing.SyncInput
	ExpectedStartingState billing.InvoiceStatus
}

func (s *Service) syncEditInvoice(ctx context.Context, input syncEditInvoiceInput) (billing.Invoice, error) {
	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.Invoice, error) {
		if err := input.SyncInput.Validate(); err != nil {
			return billing.Invoice{}, billing.ValidationError{
				Err: err,
			}
		}

		invoice, err := s.withLockedInvoiceStateMachine(ctx, withLockedStateMachineInput{
			InvoiceID: input.SyncInput.GetInvoiceID(),
			Callback: func(ctx context.Context, sm *InvoiceStateMachine) error {
				if sm.Invoice.Status != input.ExpectedStartingState {
					return billing.ValidationError{
						Err: fmt.Errorf("invoice is not in %s state", input.ExpectedStartingState),
					}
				}

				if err := input.SyncInput.MergeIntoInvoice(&sm.Invoice); err != nil {
					return billing.ValidationError{
						Err: err,
					}
				}

				if sm.Invoice.Metadata == nil {
					sm.Invoice.Metadata = make(map[string]string)
				}

				for k, v := range input.SyncInput.GetAdditionalMetadata() {
					sm.Invoice.Metadata[k] = v
				}

				err := sm.AdvanceUntilStateStable(ctx)
				if err != nil {
					return billing.ValidationError{
						Err: err,
					}
				}

				return nil
			},
		})
		if err != nil {
			return billing.Invoice{}, err
		}

		return s.updateInvoice(ctx, invoice)
	})
}

func (s *Service) SyncDraftInvoice(ctx context.Context, input billing.SyncDraftInvoiceInput) (billing.Invoice, error) {
	return s.syncEditInvoice(ctx, syncEditInvoiceInput{
		SyncInput:             input,
		ExpectedStartingState: billing.InvoiceStatusDraftSyncing,
	})
}

func (s *Service) SyncIssuingInvoice(ctx context.Context, input billing.SyncIssuingInvoiceInput) (billing.Invoice, error) {
	return s.syncEditInvoice(ctx, syncEditInvoiceInput{
		SyncInput:             input,
		ExpectedStartingState: billing.InvoiceStatusIssuingSyncing,
	})
}
