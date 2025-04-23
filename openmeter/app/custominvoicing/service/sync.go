package service

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ appcustominvoicing.SyncService = (*Service)(nil)

func (s *Service) SyncDraftInvoice(ctx context.Context, input appcustominvoicing.SyncDraftInvoiceInput) (billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billing.Invoice{}, err
	}

	return s.billingService.SyncDraftInvoice(ctx, billing.SyncDraftInvoiceInput{
		InvoiceID:            input.InvoiceID,
		UpsertInvoiceResults: input.UpsertInvoiceResults,
		AdditionalMetadata: map[string]string{
			appcustominvoicing.MetadataKeyDraftSyncedAt: clock.Now().Format(time.RFC3339),
		},
		InvoiceValidator: s.ValidateInvoiceApp,
	})
}

func (s *Service) SyncIssuingInvoice(ctx context.Context, input appcustominvoicing.SyncIssuingInvoiceInput) (billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billing.Invoice{}, err
	}

	return s.billingService.SyncIssuingInvoice(ctx, billing.SyncIssuingInvoiceInput{
		InvoiceID:             input.InvoiceID,
		FinalizeInvoiceResult: input.FinalizeInvoiceResult,
		AdditionalMetadata: map[string]string{
			appcustominvoicing.MetadataKeyFinalizedAt: clock.Now().Format(time.RFC3339),
		},
		InvoiceValidator: s.ValidateInvoiceApp,
	})
}

func (s *Service) ValidateInvoiceApp(invoice billing.Invoice) error {
	if invoice.Workflow.Apps == nil {
		return models.NewGenericValidationError(fmt.Errorf("invoice %s has no apps", invoice.ID))
	}

	if invoice.Workflow.Apps.Invoicing == nil {
		return models.NewGenericValidationError(fmt.Errorf("invoice %s has no invoicing app", invoice.ID))
	}

	if invoice.Workflow.Apps.Invoicing.GetType() != app.AppTypeCustomInvoicing {
		return models.NewGenericValidationError(fmt.Errorf("invoice %s is not managed by the custom invoicing app", invoice.ID))
	}

	return nil
}

func (s *Service) HandlePaymentTrigger(ctx context.Context, input appcustominvoicing.HandlePaymentTriggerInput) (billing.Invoice, error) {
	if err := input.Validate(); err != nil {
		return billing.Invoice{}, err
	}

	return transaction.Run(ctx, s.adapter, func(ctx context.Context) (billing.Invoice, error) {
		invoice, err := s.billingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: input.InvoiceID,
		})
		if err != nil {
			return billing.Invoice{}, err
		}

		if err := s.ValidateInvoiceApp(invoice); err != nil {
			return billing.Invoice{}, err
		}

		err = s.billingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
			InvoiceTriggerInput: billing.InvoiceTriggerInput{
				Invoice: input.InvoiceID,
				Trigger: input.Trigger,
			},
			AppType:    app.AppTypeCustomInvoicing,
			Capability: app.CapabilityTypeCollectPayments,
		})
		if err != nil {
			return billing.Invoice{}, err
		}

		invoice, err = s.billingService.GetInvoiceByID(ctx, billing.GetInvoiceByIdInput{
			Invoice: input.InvoiceID,
		})
		if err != nil {
			return billing.Invoice{}, err
		}

		if len(invoice.ValidationIssues) > 0 {
			criticalIssues := lo.Filter(invoice.ValidationIssues, func(issue billing.ValidationIssue, _ int) bool {
				return issue.Severity == billing.ValidationIssueSeverityCritical
			})

			if len(criticalIssues) > 0 {
				// Warning: This causes a rollback of the transaction
				return billing.Invoice{}, billing.ValidationError{
					Err: criticalIssues.AsError(),
				}
			}
		}

		return invoice, nil
	})
}
