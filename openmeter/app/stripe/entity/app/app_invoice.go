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
	return a.updateInvoice(ctx, invoice)
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
	stripeInvoiceLineParams := &stripe.InvoiceAddLinesParams{}

	// Walk the tree
	var queue []*billing.Line

	// Feed the queue with the root lines
	invoice.Lines.ForEach(func(lines []*billing.Line) {
		queue = append(queue, lines...)
	})

	// We collect the line IDs to match them with the Stripe line items from the response
	var lineIDs []string

	for len(queue) > 0 {
		line := queue[0]
		queue = queue[1:]

		// Add children to the queue
		childrens := line.Children.OrEmpty()
		for _, l := range childrens {
			queue = append(queue, l)
		}

		// Only add line items for leaf nodes
		if len(childrens) > 0 {
			continue
		}

		period := &stripe.InvoiceAddLinesLinePeriodParams{
			Start: lo.ToPtr(line.Period.Start.Unix()),
			End:   lo.ToPtr(line.Period.End.Unix()),
		}

		// Add discounts
		line.Discounts.ForEach(func(discounts []billing.LineDiscount) {
			for _, discount := range discounts {
				lineIDs = append(lineIDs, line.ID)

				stripeInvoiceLineParams.Lines = append(stripeInvoiceLineParams.Lines, &stripe.InvoiceAddLinesLineParams{
					Description: discount.Description,
					Amount:      lo.ToPtr(-discount.Amount.GetFixed()),
					Quantity:    lo.ToPtr(int64(1)),
					Period:      period,
				})
			}
		})

		// Add line item
		switch line.Type {
		case billing.InvoiceLineTypeFee:
			lineIDs = append(lineIDs, line.ID)

			stripeInvoiceLineParams.Lines = append(stripeInvoiceLineParams.Lines, &stripe.InvoiceAddLinesLineParams{
				Description: line.Description,
				Amount:      lo.ToPtr(line.Totals.Amount.GetFixed()),
				Quantity:    lo.ToPtr(int64(1)),
				Period:      period,
			})
		case billing.InvoiceLineTypeUsageBased:
			lineIDs = append(lineIDs, line.ID)

			stripeInvoiceLineParams.Lines = append(stripeInvoiceLineParams.Lines, &stripe.InvoiceAddLinesLineParams{
				Description: line.Description,
				Amount:      lo.ToPtr(line.Totals.Amount.GetFixed()),
				Quantity:    lo.ToPtr(line.UsageBased.Quantity.GetFixed()),
				Period:      period,
			})
		default:
			return result, fmt.Errorf("unsupported line type: %s", line.Type)
		}
	}

	// Add Stripe line items to the Stripe invoice
	stripeInvoice, err = client.Invoices.AddLines(stripeInvoice.ID, stripeInvoiceLineParams)
	if err != nil {
		return nil, fmt.Errorf("failed to add line items to invoice in stripe: %w", err)
	}

	// Add external line IDs
	for idx, stripeLine := range stripeInvoice.Lines.Data {
		result.AddLineExternalID(lineIDs[idx], stripeLine.ID)
	}

	return result, nil
}

// updateInvoice update the invoice for the app
func (a App) updateInvoice(ctx context.Context, invoice billing.Invoice) (*billing.UpsertInvoiceResult, error) {
	// Get the Stripe client
	_, stripeClient, err := a.getStripeClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe client: %w", err)
	}

	client := stripeClient.GetClient()

	// Get the invoice from Stripe
	stripeInvoice, err := client.Invoices.Get(invoice.ExternalIDs.Invoicing, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice in stripe: %w", err)
	}

	// TODO: do we need to update Stripe invoice? Like due date?

	existingStripeLines := make(map[string]bool)

	// Return the result
	result := &billing.UpsertInvoiceResult{}
	result.SetExternalID(stripeInvoice.ID)
	result.SetInvoiceNumber(stripeInvoice.Number)

	for _, stripeLine := range stripeInvoice.Lines.Data {
		existingStripeLines[stripeLine.ID] = true
	}

	addStripeLines := &stripe.InvoiceAddLinesParams{}
	updateStripeLines := &stripe.InvoiceUpdateLinesParams{}
	removeStripeLines := &stripe.InvoiceRemoveLinesParams{}

	// Walk the tree
	var queue []*billing.Line

	// Feed the queue with the root lines
	invoice.Lines.ForEach(func(lines []*billing.Line) {
		queue = append(queue, lines...)
	})

	// We collect the line IDs to match them with the Stripe line items from the response
	var lineIDs []string

	for len(queue) > 0 {
		line := queue[0]
		queue = queue[1:]

		// Add children to the queue
		childrens := line.Children.OrEmpty()
		for _, l := range childrens {
			queue = append(queue, l)
		}

		// Only add line items for leaf nodes
		if len(childrens) > 0 {
			continue
		}

		period := &stripe.InvoiceAddLinesLinePeriodParams{
			Start: lo.ToPtr(line.Period.Start.Unix()),
			End:   lo.ToPtr(line.Period.End.Unix()),
		}

		// Add discounts
		line.Discounts.ForEach(func(discounts []billing.LineDiscount) {
			for _, discount := range discounts {
				// Add line item if it doesn't exist
				if line.ExternalIDs.Invoicing == "" {
					lineIDs = append(lineIDs, line.ID)

					addStripeLines.Lines = append(addStripeLines.Lines, &stripe.InvoiceAddLinesLineParams{
						Description: discount.Description,
						Amount:      lo.ToPtr(-discount.Amount.GetFixed()),
						Quantity:    lo.ToPtr(int64(1)),
						Period:      period,
					})
				} else {
					// Update line item
					delete(existingStripeLines, line.ExternalIDs.Invoicing)

					result.AddLineExternalID(line.ID, line.ExternalIDs.Invoicing)

					updateStripeLines.Lines = append(updateStripeLines.Lines, &stripe.InvoiceUpdateLinesLineParams{
						ID:          lo.ToPtr(line.ExternalIDs.Invoicing),
						Description: discount.Description,
						Amount:      lo.ToPtr(-discount.Amount.GetFixed()),
						Quantity:    lo.ToPtr(int64(1)),
						// period is not updatable
					})
				}
			}
		})

		// Add line item
		switch line.Type {
		case billing.InvoiceLineTypeFee:
			// Add line item if it doesn't exist
			if line.ExternalIDs.Invoicing == "" {
				lineIDs = append(lineIDs, line.ID)

				addStripeLines.Lines = append(addStripeLines.Lines, &stripe.InvoiceAddLinesLineParams{
					Description: line.Description,
					Amount:      lo.ToPtr(line.Totals.Amount.GetFixed()),
					Quantity:    lo.ToPtr(int64(1)),
					Period:      period,
				})
			} else {
				// Update line item
				delete(existingStripeLines, line.ExternalIDs.Invoicing)

				result.AddLineExternalID(line.ID, line.ExternalIDs.Invoicing)

				updateStripeLines.Lines = append(updateStripeLines.Lines, &stripe.InvoiceUpdateLinesLineParams{
					ID:          lo.ToPtr(line.ExternalIDs.Invoicing),
					Description: line.Description,
					Amount:      lo.ToPtr(line.Totals.Amount.GetFixed()),
					Quantity:    lo.ToPtr(int64(1)),
					// period is not updatable
				})
			}

		case billing.InvoiceLineTypeUsageBased:
			// Add line item if it doesn't exist
			if line.ExternalIDs.Invoicing == "" {
				lineIDs = append(lineIDs, line.ID)

				addStripeLines.Lines = append(addStripeLines.Lines, &stripe.InvoiceAddLinesLineParams{
					Description: line.Description,
					Amount:      lo.ToPtr(line.Totals.Amount.GetFixed()),
					Quantity:    lo.ToPtr(line.UsageBased.Quantity.GetFixed()),
					Period:      period,
				})
			} else {
				// Update line item
				delete(existingStripeLines, line.ExternalIDs.Invoicing)

				result.AddLineExternalID(line.ID, line.ExternalIDs.Invoicing)

				updateStripeLines.Lines = append(updateStripeLines.Lines, &stripe.InvoiceUpdateLinesLineParams{
					ID:          lo.ToPtr(line.ExternalIDs.Invoicing),
					Description: line.Description,
					Amount:      lo.ToPtr(line.Totals.Amount.GetFixed()),
					Quantity:    lo.ToPtr(line.UsageBased.Quantity.GetFixed()),
					// period is not updatable
				})
			}
		default:
			return result, fmt.Errorf("unsupported line type: %s", line.Type)
		}
	}

	// Delete line items that are not in the invoice
	for stripeLineID := range existingStripeLines {
		removeStripeLines.Lines = append(removeStripeLines.Lines, &stripe.InvoiceRemoveLinesLineParams{
			ID:       lo.ToPtr(stripeLineID),
			Behavior: lo.ToPtr("delete"),
		})
	}

	// Add Stripe line items to the Stripe invoice
	if len(addStripeLines.Lines) > 0 {
		shift := len(stripeInvoice.Lines.Data) - 1

		stripeInvoice, err = client.Invoices.AddLines(stripeInvoice.ID, addStripeLines)
		if err != nil {
			return nil, fmt.Errorf("failed to add line items to invoice in stripe: %w", err)
		}

		// Add new line IDs
		for idx, stripeLine := range stripeInvoice.Lines.Data {
			result.AddLineExternalID(lineIDs[idx+shift], stripeLine.ID)
		}
	}

	// Update Stripe line items in the Stripe invoice
	if len(updateStripeLines.Lines) > 0 {
		stripeInvoice, err = client.Invoices.UpdateLines(stripeInvoice.ID, updateStripeLines)
		if err != nil {
			return nil, fmt.Errorf("failed to update line items in invoice in stripe: %w", err)
		}
	}

	// Remove Stripe line items from the Stripe invoice
	if len(removeStripeLines.Lines) > 0 {
		stripeInvoice, err = client.Invoices.RemoveLines(stripeInvoice.ID, removeStripeLines)
		if err != nil {
			return nil, fmt.Errorf("failed to remove line items from invoice in stripe: %w", err)
		}
	}

	return result, nil
}

// getStripeClient gets the Stripe client for the app
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
