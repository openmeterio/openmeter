package appservice

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"
	"github.com/stripe/stripe-go/v80"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var _ appstripe.BillingService = (*Service)(nil)

func (s *Service) GetSupplierContact(ctx context.Context, input appstripeentity.GetSupplierContactInput) (billing.SupplierContact, error) {
	return s.adapter.GetSupplierContact(ctx, input)
}

// Invoice webhook handlers
func (s *Service) HandleInvoiceStateTransition(ctx context.Context, input appstripeentity.HandleInvoiceStateTransitionInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	ctx = context.WithValue(ctx, StripeInvoiceIDAttributeName, input.Invoice.ID)

	invoice, err := s.getInvoiceByStripeID(ctx, input.AppID, input.Invoice.ID)
	if err != nil {
		return err
	}

	if invoice == nil {
		return nil
	}

	ctx = context.WithValue(ctx, InvoiceIDAttributeName, invoice.ID)
	ctx = context.WithValue(ctx, InvoiceStatusAttributeName, invoice.Status)

	if slices.Contains(input.TargetStatuses, invoice.Status) {
		// No need to handle the event, the invoice is already in the target state
		s.logger.InfoContext(ctx, "invoice is already in the target state, ignoring state event")
		return nil
	}

	if invoice.Status.Matches(input.IgnoreInvoiceInStatus...) {
		// No need to handle the event, the invoice is in a state that should be ignored
		s.logger.InfoContext(ctx, "invoice is in a state that should be ignored, ignoring state event")
		return nil
	}

	var stripeInvoice *stripe.Invoice
	if input.ShouldTriggerOnEvent != nil || input.GetValidationErrors != nil {
		// Let's rule out any late events by validating the invoice status
		stripeInvoice, err = s.adapter.GetStripeInvoice(ctx, appstripeentity.GetStripeInvoiceInput{
			AppID:           input.AppID,
			StripeInvoiceID: input.Invoice.ID,
		})
		if err != nil {
			s.logger.Error("failed to get stripe invoice", "invoice_id", invoice.ID, "error", err)
			return err
		}
	}

	if input.ShouldTriggerOnEvent != nil {
		shouldTrigger, err := input.ShouldTriggerOnEvent(stripeInvoice)
		if err != nil {
			s.logger.Error("failed to determine if event should trigger", "error", err)
		}

		if !shouldTrigger {
			s.logger.InfoContext(ctx, "event should not trigger invoice state transition, ignoring state event")
			return nil
		}
	}

	var validationErrors *billing.InvoiceTriggerValidationInput
	if input.GetValidationErrors != nil {
		stripeValidationErrors, err := input.GetValidationErrors(stripeInvoice)
		if err != nil {
			s.logger.Error("failed to get validation errors", "error", err)
			return err
		}

		if stripeValidationErrors != nil {
			validationErrors = &billing.InvoiceTriggerValidationInput{
				Operation: billing.InvoiceOpInitiatePayment,
				Errors: lo.Map(stripeValidationErrors.Errors, func(stripeErr *stripe.Error, _ int) error {
					return stripeErrorToValidationError(stripeErr)
				}),
			}
		}
	}

	err = s.billingService.TriggerInvoice(ctx, billing.InvoiceTriggerServiceInput{
		InvoiceTriggerInput: billing.InvoiceTriggerInput{
			Invoice:          invoice.InvoiceID(),
			Trigger:          input.Trigger,
			ValidationErrors: validationErrors,
		},
		AppType:    appentitybase.AppTypeStripe,
		Capability: appentitybase.CapabilityTypeCollectPayments,
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to trigger invoice failed trigger")
		return err
	}

	s.logger.InfoContext(ctx, "invoice state transition handled successfully", "trigger", input.Trigger)

	return nil
}

func (s *Service) HandleInvoiceSentEvent(ctx context.Context, input appstripeentity.HandleInvoiceSentEventInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	ctx = context.WithValue(ctx, StripeInvoiceIDAttributeName, input.Invoice.ID)

	invoice, err := s.getInvoiceByStripeID(ctx, input.AppID, input.Invoice.ID)
	if err != nil {
		return err
	}

	if invoice == nil {
		return nil
	}

	ctx = context.WithValue(ctx, InvoiceIDAttributeName, invoice.ID)
	ctx = context.WithValue(ctx, InvoiceStatusAttributeName, invoice.Status)

	return s.billingService.UpdateInvoiceFields(ctx, billing.UpdateInvoiceFieldsInput{
		Invoice:          invoice.InvoiceID(),
		SentToCustomerAt: mo.Some(lo.ToPtr(time.Unix(input.SentAt, 0))),
	})
}

func stripeErrorToValidationError(stripeErr *stripe.Error) error {
	if stripeErr == nil {
		return nil
	}

	return billing.NewValidationError(string(stripeErr.Code), stripeErr.Msg)
}

// getInvoiceByStripeID retrieves an invoice by its stripe ID, it returns nil if the invoice is not found (thus not managed by the app)
func (s *Service) getInvoiceByStripeID(ctx context.Context, appID appentitybase.AppID, stripeInvoiceID string) (*billing.Invoice, error) {
	invoices, err := s.billingService.ListInvoices(ctx, billing.ListInvoicesInput{
		Namespaces: []string{appID.Namespace},
		ExternalIDs: &billing.ListInvoicesExternalIDFilter{
			Type: billing.InvoicingExternalIDType,
			IDs:  []string{stripeInvoiceID},
		},
		IncludeDeleted: true,
		Page: pagination.Page{
			PageNumber: 1,
			PageSize:   5,
		},
		Expand: billing.InvoiceExpand{},
	})
	if err != nil {
		return nil, err
	}

	if len(invoices.Items) == 0 {
		// Invoice is not found, log a warning
		s.logger.WarnContext(ctx, "stripe invoice not found in local database, assuming non-managed invoice")
		return nil, nil
	}

	if len(invoices.Items) > 1 {
		// This should never happen, log an error
		s.logger.ErrorContext(ctx, "multiple invoices found for the same external ID")
		return nil, fmt.Errorf("multiple invoices found for the same external ID: %s", stripeInvoiceID)
	}

	invoice := invoices.Items[0]
	if invoice.Workflow.AppReferences.Invoicing.ID != appID.ID {
		// Invoice is not managed by the app, log an error, should not happen, but if it happens we need to investigate
		s.logger.ErrorContext(ctx, "stripe invoice not managed by the app", "invoice_id", invoice.ID, "app_id", appID.ID)
		return nil, fmt.Errorf("stripe invoice not managed by the app: %s", invoice.ID)
	}

	return &invoice, nil
}
