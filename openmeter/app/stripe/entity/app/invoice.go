package appstripeentityapp

import (
	"context"
	"fmt"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/app/stripe/invoicesync"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
)

var (
	_ billing.InvoicingApp            = (*App)(nil)
	_ billing.InvoicingAppAsyncSyncer = (*App)(nil)
)

// ValidateStandardInvoice validates the invoice for the app
func (a App) ValidateStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
	customerID := customer.CustomerID{
		Namespace: invoice.Namespace,
		ID:        invoice.Customer.CustomerID,
	}

	// Check if the customer can be invoiced with Stripe.
	// We check this at app customer create but we need to ensure that OpenMeter is
	// still in sync with Stripe, for example that the customer wasn't deleted in Stripe.
	err := a.ValidateCustomerByID(ctx, customerID, []app.CapabilityType{
		// For now now we only support Stripe with automatic tax calculation and payment collection.
		app.CapabilityTypeCalculateTax,
		app.CapabilityTypeInvoiceCustomers,
		app.CapabilityTypeCollectPayments,
	})
	if err != nil {
		return fmt.Errorf("validate customer: %w", err)
	}

	// Check if the invoice has any capabilities that are not supported by Stripe.
	// Today all capabilities are supported.

	return nil
}

// UpsertStandardInvoice generates a persistent sync plan for the invoice and publishes
// an event for async execution. Results are written back via SyncDraftInvoice on completion.
func (a App) UpsertStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
	return a.generateDraftSyncPlan(ctx, invoice)
}

// buildSyncPlanInput builds the common input needed by all plan generators.
func (a App) buildSyncPlanInput(ctx context.Context, invoice billing.StandardInvoice) (invoicesync.CreateSyncPlanInput, error) {
	stripeCustomerData, err := a.StripeAppService.GetStripeCustomerData(ctx, appstripeentity.GetStripeCustomerDataInput{
		AppID: a.GetID(),
		CustomerID: customer.CustomerID{
			Namespace: invoice.Namespace,
			ID:        invoice.Customer.CustomerID,
		},
	})
	if err != nil {
		return invoicesync.CreateSyncPlanInput{}, fmt.Errorf("getting stripe customer data: %w", err)
	}

	var existingStripeLines []*stripe.InvoiceLineItem
	if invoice.ExternalIDs.Invoicing != "" {
		_, stripeClient, err := a.getStripeClient(ctx, "buildSyncPlanInput", "invoice_id", invoice.ID)
		if err != nil {
			return invoicesync.CreateSyncPlanInput{}, fmt.Errorf("getting stripe client: %w", err)
		}
		existingStripeLines, err = stripeClient.ListInvoiceLineItems(ctx, invoice.ExternalIDs.Invoicing)
		if err != nil {
			return invoicesync.CreateSyncPlanInput{}, fmt.Errorf("listing stripe line items: %w", err)
		}
	}

	return invoicesync.CreateSyncPlanInput{
		Invoice: invoice,
		GeneratorInput: invoicesync.PlanGeneratorInput{
			Invoice:              invoice,
			StripeCustomerID:     stripeCustomerData.StripeCustomerID,
			StripeDefaultPayment: lo.FromPtr(stripeCustomerData.StripeDefaultPaymentMethodID),
			AppID:                a.GetID().ID,
			Currency:             string(invoice.Currency),
			ExistingStripeLines:  existingStripeLines,
		},
	}, nil
}

func (a App) generateDraftSyncPlan(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
	input, err := a.buildSyncPlanInput(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("building plan input: %w", err)
	}

	if err := a.SyncPlanService.CreateDraftSyncPlan(ctx, input); err != nil {
		return nil, err
	}

	return nil, nil
}

// DeleteStandardInvoice generates a delete sync plan for async execution.
func (a App) DeleteStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
	return a.SyncPlanService.CreateDeleteSyncPlan(ctx, invoicesync.CreateSyncPlanInput{
		Invoice: invoice,
		GeneratorInput: invoicesync.PlanGeneratorInput{
			Invoice: invoice,
			AppID:   a.GetID().ID,
		},
	})
}

// FinalizeStandardInvoice generates an issuing sync plan for async execution.
func (a App) FinalizeStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.FinalizeStandardInvoiceResult, error) {
	input, err := a.buildSyncPlanInput(ctx, invoice)
	if err != nil {
		return nil, fmt.Errorf("building plan input: %w", err)
	}

	if err := a.SyncPlanService.CreateIssuingSyncPlan(ctx, input); err != nil {
		return nil, err
	}

	return nil, nil
}

// CanDraftSyncAdvance checks whether the draft sync has completed.
func (a App) CanDraftSyncAdvance(invoice billing.StandardInvoice) (bool, error) {
	return a.canSyncAdvance(invoice, invoicesync.MetadataKeyDraftSyncCompletedAt)
}

// CanIssuingSyncAdvance checks whether the issuing sync has completed.
func (a App) CanIssuingSyncAdvance(invoice billing.StandardInvoice) (bool, error) {
	return a.canSyncAdvance(invoice, invoicesync.MetadataKeyIssuingSyncCompletedAt)
}

func (a App) canSyncAdvance(invoice billing.StandardInvoice, metadataKey string) (bool, error) {
	if invoice.Metadata == nil {
		return false, nil
	}
	_, ok := invoice.Metadata[metadataKey]
	return ok, nil
}
