package billingservice

import (
	"context"

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
