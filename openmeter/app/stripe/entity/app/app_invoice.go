package appstripeentityapp

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/app"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"
)

var _ billing.InvoicingApp = (*App)(nil)

// ValidateInvoice validates the invoice for the app
func (a App) ValidateInvoice(ctx context.Context, invoice billing.Invoice) error {
	customerID := customerentity.CustomerID{
		Namespace: invoice.Namespace,
		ID:        invoice.Customer.CustomerID,
	}

	// Check if the customer can be invoiced with Stripe.
	// We check this at app customer create but we need to ensure that OpenMeter is
	// still in sync with Stripe, for example that the customer wasn't deleted in Stripe.
	err := a.ValidateCustomerByID(ctx, customerID, []appentitybase.CapabilityType{
		// For now now we only support Stripe with automatic tax calculation and payment collection.
		appentitybase.CapabilityTypeCalculateTax,
		appentitybase.CapabilityTypeInvoiceCustomers,
		appentitybase.CapabilityTypeCollectPayments,
	})
	if err != nil {
		return fmt.Errorf("validate customer: %w", err)
	}

	// Check if the invoice has any capabilities that are not supported by Stripe.
	// Today all capabilities are supported.

	return nil
}

// UpsertInvoice upserts the invoice for the app
func (a App) UpsertInvoice(ctx context.Context, invoice billing.Invoice) (*billing.UpsertInvoiceResult, error) {
	// Create the invoice in Stripe.
	if invoice.ExternalIDs.Invoicing == "" {
		return a.createInvoice(ctx, invoice)
	}

	// Update the invoice in Stripe.

	// TODO: Implement
	return nil, fmt.Errorf("upsert invoice operation not implemented")
}

// DeleteInvoice deletes the invoice for the app
func (a App) DeleteInvoice(ctx context.Context, invoice billing.Invoice) error {
	return fmt.Errorf("delete invoice operation not implemented")
}

// FinalizeInvoice finalizes the invoice for the app
func (a App) FinalizeInvoice(ctx context.Context, invoice billing.Invoice) (*billing.FinalizeInvoiceResult, error) {
	return nil, fmt.Errorf("finalize invoice operation not implemented")
}

// createInvoice creates the invoice for the app
func (a App) createInvoice(ctx context.Context, invoice billing.Invoice) (*billing.UpsertInvoiceResult, error) {
	customerID := customerentity.CustomerID{
		Namespace: invoice.Namespace,
		ID:        invoice.Customer.CustomerID,
	}

	// Get the Stripe client
	_, stripeClient, err := a.getStripeClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe client: %w", err)
	}

	client := stripeClient.GetClient()

	// Get stripe customer data
	stripeCustomerData, err := a.StripeAppService.GetStripeCustomerData(ctx, appstripeentity.GetStripeCustomerDataInput{
		AppID:      a.GetID(),
		CustomerID: customerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe customer data: %w", err)
	}

	// This should never happen as invoices call ValidateInvoice before calling this method.
	if stripeCustomerData.StripeDefaultPaymentMethodID == nil {
		return nil, app.CustomerPreConditionError{
			AppID:      a.GetID(),
			AppType:    a.GetType(),
			CustomerID: customerID,
			Condition:  "default payment method cannot be null",
		}
	}

	// Create the invoice in Stripe
	stripeInvoice, err := client.Invoices.New(&stripe.InvoiceParams{
		// FinalizeInvoice will advance the invoice
		AutoAdvance:          stripe.Bool(false),
		Currency:             stripe.String(string(invoice.Currency)),
		Customer:             stripe.String(stripeCustomerData.StripeCustomerID),
		DueDate:              lo.ToPtr(invoice.DueAt.Unix()),
		DefaultPaymentMethod: stripe.String(*stripeCustomerData.StripeDefaultPaymentMethodID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice in stripe: %w", err)
	}

	// Return the result
	result := &billing.UpsertInvoiceResult{}
	result.SetExternalID(stripeInvoice.ID)
	result.SetInvoiceNumber(stripeInvoice.Number)

	// Add line items
	lineParams := &stripe.InvoiceAddLinesParams{}

	invoice.Lines.ForEach(func(lines []*billing.Line) {
		for _, line := range lines {
			switch line.Type {
			case billing.InvoiceLineTypeFee:
				lineParams.Lines = append(lineParams.Lines, &stripe.InvoiceAddLinesLineParams{
					InvoiceItem: lo.ToPtr(line.Name),
					Description: line.Description,
					Amount:      lo.ToPtr(line.FlatFee.PerUnitAmount.GetFixed()),
					Quantity:    lo.ToPtr(line.FlatFee.Quantity.GetFixed()),
				})
			case billing.InvoiceLineTypeUsageBased:
				lineParams.Lines = append(lineParams.Lines, &stripe.InvoiceAddLinesLineParams{
					// TODO
				})
			default:
				err = fmt.Errorf("unsupported line type: %s", line.Type)
				return
			}
		}
	})

	if err != nil {
		return nil, fmt.Errorf("failed to map line items to stripe: %w", err)
	}

	stripeInvoice, err = client.Invoices.AddLines(stripeInvoice.ID, lineParams)
	if err != nil {
		return nil, fmt.Errorf("failed to add line items to invoice in stripe: %w", err)
	}

	return result, nil
}

func (a App) getStripeClient(ctx context.Context) (appstripeentity.AppData, stripeclient.StripeClient, error) {
	// Get Stripe App
	stripeAppData, err := a.StripeAppService.GetStripeAppData(ctx, appstripeentity.GetStripeAppDataInput{
		AppID: a.GetID(),
	})
	if err != nil {
		return appstripeentity.AppData{}, nil, fmt.Errorf("failed to get stripe app data: %w", err)
	}

	// Get Stripe API Key
	apiKeySecret, err := a.SecretService.GetAppSecret(ctx, secretentity.NewSecretID(a.GetID(), stripeAppData.APIKey.ID, appstripeentity.APIKeySecretKey))
	if err != nil {
		return appstripeentity.AppData{}, nil, fmt.Errorf("failed to get stripe api key secret: %w", err)
	}

	// Stripe Client
	stripeClient, err := a.StripeClientFactory(stripeclient.StripeClientConfig{
		Namespace: apiKeySecret.SecretID.Namespace,
		APIKey:    apiKeySecret.Value,
	})
	if err != nil {
		return appstripeentity.AppData{}, nil, fmt.Errorf("failed to create stripe client: %w", err)
	}

	return appstripeentity.AppData{}, stripeClient, nil
}
