package appstripeentityapp

import (
	"context"
	"fmt"
	"maps"
	"sort"

	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
)

const (
	invoiceLineMetadataID           = "om_line_id"
	invoiceLineMetadataType         = "om_line_type"
	invoiceLineMetadataTypeLine     = "line"
	invoiceLineMetadataTypeDiscount = "discount"
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
	// Get the Stripe client
	_, stripeClient, err := a.getStripeClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to get stripe client: %w", err)
	}

	// Delete the invoice in Stripe
	return stripeClient.DeleteInvoice(ctx, stripeclient.DeleteInvoiceInput{
		StripeInvoiceID: invoice.ExternalIDs.Invoicing,
	})
}

// FinalizeInvoice finalizes the invoice for the app
func (a App) FinalizeInvoice(ctx context.Context, invoice billing.Invoice) (*billing.FinalizeInvoiceResult, error) {
	// Get the Stripe client
	_, stripeClient, err := a.getStripeClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe client: %w", err)
	}

	// Finalize the invoice in Stripe
	stripeInvoice, err := stripeClient.FinalizeInvoice(ctx, stripeclient.FinalizeInvoiceInput{
		StripeInvoiceID: invoice.ExternalIDs.Invoicing,

		// Controls whether Stripe performs automatic collection of the invoice.
		// If false, the invoice’s state doesn’t automatically advance without an explicit action.
		// https://docs.stripe.com/api/invoices/finalize#finalize_invoice-auto_advance
		AutoAdvance: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to finalize invoice in stripe: %w", err)
	}

	// Result
	result := billing.NewFinalizeInvoiceResult()

	// The PaymentIntent is generated when the invoice is finalized,
	// and can then be used to pay the invoice.
	// https://docs.stripe.com/api/invoices/object#invoice_object-payment_intent
	if stripeInvoice.PaymentIntent != nil {
		result.SetPaymentExternalID(stripeInvoice.PaymentIntent.ID)
	}

	return result, nil
}

// createInvoice creates the invoice for the app
func (a App) createInvoice(ctx context.Context, invoice billing.Invoice) (*billing.UpsertInvoiceResult, error) {
	// Get the currency calculator
	calculator, err := NewStripeCalculator(invoice.Currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency calculator: %w", err)
	}

	customerID := customerentity.CustomerID{
		Namespace: invoice.Namespace,
		ID:        invoice.Customer.CustomerID,
	}

	// Get the Stripe client
	_, stripeClient, err := a.getStripeClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe client: %w", err)
	}

	// Get stripe customer data
	stripeCustomerData, err := a.StripeAppService.GetStripeCustomerData(ctx, appstripeentity.GetStripeCustomerDataInput{
		AppID:      a.GetID(),
		CustomerID: customerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe customer data: %w", err)
	}

	// Create the invoice in Stripe
	stripeInvoice, err := stripeClient.CreateInvoice(ctx, stripeclient.CreateInvoiceInput{
		Currency:                     invoice.Currency,
		StripeCustomerID:             stripeCustomerData.StripeCustomerID,
		StripeDefaultPaymentMethodID: stripeCustomerData.StripeDefaultPaymentMethodID,
		DueDate:                      invoice.DueAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice in stripe: %w", err)
	}

	// Return the result
	result := billing.NewUpsertInvoiceResult()
	result.SetExternalID(stripeInvoice.ID)
	result.SetInvoiceNumber(stripeInvoice.Number)

	// Add lines to the Stripe invoice
	var stripeLineAdd []*stripe.InvoiceAddLinesLineParams

	lines := invoice.FlattenLinesByID()

	// Check if we have any non integer amount or quantity
	// We use this to determinate if we add alreay calculated total or per unit amount and quantity to the Stripe line item
	isInteger := isAllInteger(calculator, lines)

	// Walk the tree
	for _, line := range lines {
		if line.Type != billing.InvoiceLineTypeFee {
			continue
		}

		period := &stripe.InvoiceAddLinesLinePeriodParams{
			Start: lo.ToPtr(line.Period.Start.Unix()),
			End:   lo.ToPtr(line.Period.End.Unix()),
		}

		// Add discounts
		line.Discounts.ForEach(func(discounts []billing.LineDiscount) {
			for _, discount := range discounts {
				name := line.Name
				if discount.Description != nil {
					name = fmt.Sprintf("%s (%s)", name, *discount.Description)
				}

				stripeLineAdd = append(stripeLineAdd, &stripe.InvoiceAddLinesLineParams{
					Description: lo.ToPtr(name),
					Amount:      lo.ToPtr(-calculator.RoundToAmount(discount.Amount)),
					Quantity:    lo.ToPtr(int64(1)),
					Period:      period,
					Metadata: map[string]string{
						// TODO: is a discount ID a line id?
						invoiceLineMetadataID:   discount.ID,
						invoiceLineMetadataType: invoiceLineMetadataTypeDiscount,
					},
				})
			}
		})

		// Add line
		name := line.Name
		if line.Description != nil {
			name = fmt.Sprintf("%s (%s)", name, *line.Description)
		}

		// If the per unit amount and quantity can be represented in stripe as integer we add the line item
		if isInteger {
			stripeLineAdd = append(stripeLineAdd, &stripe.InvoiceAddLinesLineParams{
				Description: lo.ToPtr(name),
				Amount:      lo.ToPtr(calculator.RoundToAmount(line.FlatFee.PerUnitAmount)),
				Quantity:    lo.ToPtr(line.FlatFee.Quantity.IntPart()),
				Period:      period,
				Metadata: map[string]string{
					invoiceLineMetadataID:   line.ID,
					invoiceLineMetadataType: invoiceLineMetadataTypeLine,
				},
			})
		} else {
			// Otherwise we add the calculated total with with quantity one
			stripeLineAdd = append(stripeLineAdd, &stripe.InvoiceAddLinesLineParams{
				Description: lo.ToPtr(name),
				Amount:      lo.ToPtr(calculator.RoundToAmount(line.Totals.Amount)),
				Quantity:    lo.ToPtr(int64(1)),
				Period:      period,
				Metadata: map[string]string{
					invoiceLineMetadataID:   line.ID,
					invoiceLineMetadataType: invoiceLineMetadataTypeLine,
				},
			})
		}
	}

	// Sort the line items by description
	sort.Slice(stripeLineAdd, func(i, j int) bool {
		descA := lo.FromPtrOr(stripeLineAdd[i].Description, "")
		descB := lo.FromPtrOr(stripeLineAdd[j].Description, "")

		return descA < descB
	})

	// Add Stripe line items to the Stripe invoice
	stripeInvoice, err = stripeClient.AddInvoiceLines(ctx, stripeclient.AddInvoiceLinesInput{
		StripeInvoiceID: stripeInvoice.ID,
		Lines:           stripeLineAdd,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add line items to invoice in stripe: %w", err)
	}

	// Add external line IDs
	for _, stripeLine := range stripeInvoice.Lines.Data {
		// TODO: currently we don't have a way to match Stripe discount line items
		if stripeLine.Metadata[invoiceLineMetadataType] == invoiceLineMetadataTypeDiscount {
			continue
		}

		id, ok := stripeLine.Metadata[invoiceLineMetadataID]
		if !ok {
			return nil, fmt.Errorf("missing line ID in metadata")
		}

		result.AddLineExternalID(id, stripeLine.ID)
	}

	return result, nil
}

// updateInvoice update the invoice for the app
func (a App) updateInvoice(ctx context.Context, invoice billing.Invoice) (*billing.UpsertInvoiceResult, error) {
	// Get the currency calculator
	calculator, err := NewStripeCalculator(invoice.Currency)
	if err != nil {
		return nil, fmt.Errorf("failed to get currency calculator: %w", err)
	}

	// Get the Stripe client
	_, stripeClient, err := a.getStripeClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe client: %w", err)
	}

	// Get the invoice from Stripe
	stripeInvoice, err := stripeClient.GetInvoice(ctx, stripeclient.GetInvoiceInput{
		StripeInvoiceID: invoice.ExternalIDs.Invoicing,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice in stripe: %w", err)
	}

	// Update the invoice in Stripe
	stripeInvoice, err = stripeClient.UpdateInvoice(ctx, stripeclient.UpdateInvoiceInput{
		StripeInvoiceID: invoice.ExternalIDs.Invoicing,
		DueDate:         invoice.DueAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update invoice in stripe: %w", err)
	}

	// Return the result
	result := billing.NewUpsertInvoiceResult()
	result.SetExternalID(stripeInvoice.ID)
	result.SetInvoiceNumber(stripeInvoice.Number)

	// Collect the existing line items
	existingStripeLines := make(map[string]bool)
	existingStripeDiscounts := make(map[string]bool)

	for _, stripeLine := range stripeInvoice.Lines.Data {
		if stripeLine.Metadata[invoiceLineMetadataType] == invoiceLineMetadataTypeDiscount {
			existingStripeDiscounts[stripeLine.ID] = true
			continue
		}

		existingStripeLines[stripeLine.ID] = true
	}

	var (
		stripeLineAdd     []*stripe.InvoiceAddLinesLineParams
		stripeLinesUpdate []*stripe.InvoiceUpdateLinesLineParams
		stripeLinesRemove []*stripe.InvoiceRemoveLinesLineParams
	)

	lines := invoice.FlattenLinesByID()

	// Check if we have any non integer amount or quantity
	// We use this to determinate if we add alreay calculated total or per unit amount and quantity to the Stripe line item
	isInteger := isAllInteger(calculator, lines)

	// Check if a line item already exists in the Stripe invoice
	isExisting := func(lineId string, lineType string) (*stripe.InvoiceLineItem, bool) {
		for _, stripeLine := range stripeInvoice.Lines.Data {
			if stripeLine.Metadata[invoiceLineMetadataID] == lineId && stripeLine.Metadata[invoiceLineMetadataType] == lineType {
				return stripeLine, true
			}
		}

		return nil, false
	}

	// Walk the tree
	for _, line := range lines {
		if line.Type != billing.InvoiceLineTypeFee {
			continue
		}

		// Add discounts
		line.Discounts.ForEach(func(discounts []billing.LineDiscount) {
			// Discounts
			for _, discount := range discounts {
				name := line.Name
				if discount.Description != nil {
					name = fmt.Sprintf("%s (%s)", name, *discount.Description)
				}

				// Add line item if it doesn't exist
				// FIXME: discounts don't have an external ID
				// if line.ExternalIDs.Invoicing == "" {
				stripeLine, isUpdate := isExisting(discount.ID, invoiceLineMetadataTypeDiscount)

				if isUpdate {
					// Update line item
					delete(existingStripeDiscounts, stripeLine.ID)

					// TODO: how do we add discount ID to result?
					// result.AddLineExternalID(discount.ID, line.ExternalIDs.Invoicing)

					stripeLinesUpdate = append(stripeLinesUpdate, &stripe.InvoiceUpdateLinesLineParams{
						ID:          lo.ToPtr(stripeLine.ID),
						Description: lo.ToPtr(name),
						Amount:      lo.ToPtr(-calculator.RoundToAmount(discount.Amount)),
						Quantity:    lo.ToPtr(int64(1)),
						Metadata:    stripeLine.Metadata,
						Period: &stripe.InvoiceUpdateLinesLinePeriodParams{
							Start: lo.ToPtr(line.Period.Start.Unix()),
							End:   lo.ToPtr(line.Period.End.Unix()),
						},
					})
				} else {
					stripeLineAdd = append(stripeLineAdd, &stripe.InvoiceAddLinesLineParams{
						Description: lo.ToPtr(name),
						Amount:      lo.ToPtr(-calculator.RoundToAmount(discount.Amount)),
						Quantity:    lo.ToPtr(int64(1)),
						Period: &stripe.InvoiceAddLinesLinePeriodParams{
							Start: lo.ToPtr(line.Period.Start.Unix()),
							End:   lo.ToPtr(line.Period.End.Unix()),
						},
						Metadata: map[string]string{
							invoiceLineMetadataID:   discount.ID,
							invoiceLineMetadataType: invoiceLineMetadataTypeDiscount,
						},
					})
				}
			}
		})

		// Add line
		name := line.Name
		if line.Description != nil {
			name = fmt.Sprintf("%s (%s)", name, *line.Description)
		}

		// FIXME: set external ID in the test invoice
		// if line.ExternalIDs.Invoicing == "" {
		stripeLine, isUpdate := isExisting(line.ID, invoiceLineMetadataTypeLine)

		if isUpdate {
			// Update line item
			delete(existingStripeLines, stripeLine.ID)

			result.AddLineExternalID(line.ID, stripeLine.ID)

			stripeLinesUpdate = append(stripeLinesUpdate, &stripe.InvoiceUpdateLinesLineParams{
				ID:          lo.ToPtr(stripeLine.ID),
				Description: lo.ToPtr(name),
				Amount:      lo.ToPtr(calculator.RoundToAmount(line.Totals.Amount)),
				Quantity:    lo.ToPtr(int64(1)),
				Metadata:    stripeLine.Metadata,
				Period: &stripe.InvoiceUpdateLinesLinePeriodParams{
					Start: lo.ToPtr(line.Period.Start.Unix()),
					End:   lo.ToPtr(line.Period.End.Unix()),
				},
			})
		} else {
			period := &stripe.InvoiceAddLinesLinePeriodParams{
				Start: lo.ToPtr(line.Period.Start.Unix()),
				End:   lo.ToPtr(line.Period.End.Unix()),
			}

			// If the per unit amount and quantity can be represented in stripe as integer we add the line item
			if isInteger {
				stripeLineAdd = append(stripeLineAdd, &stripe.InvoiceAddLinesLineParams{
					Description: lo.ToPtr(name),
					Amount:      lo.ToPtr(calculator.RoundToAmount(line.FlatFee.PerUnitAmount)),
					Quantity:    lo.ToPtr(line.FlatFee.Quantity.IntPart()),
					Period:      period,
					Metadata: map[string]string{
						invoiceLineMetadataID:   line.ID,
						invoiceLineMetadataType: invoiceLineMetadataTypeLine,
					},
				})
			} else {
				// Otherwise we add the calculated total with with quantity one
				stripeLineAdd = append(stripeLineAdd, &stripe.InvoiceAddLinesLineParams{
					Description: lo.ToPtr(name),
					Amount:      lo.ToPtr(calculator.RoundToAmount(line.Totals.Amount)),
					Quantity:    lo.ToPtr(int64(1)),
					Period:      period,
					Metadata: map[string]string{
						invoiceLineMetadataID:   line.ID,
						invoiceLineMetadataType: invoiceLineMetadataTypeLine,
					},
				})
			}
		}
	}

	// Add Stripe lines to the Stripe invoice
	if len(stripeLineAdd) > 0 {
		// Sort the line items by description
		sort.Slice(stripeLineAdd, func(i, j int) bool {
			descA := lo.FromPtrOr(stripeLineAdd[i].Description, "")
			descB := lo.FromPtrOr(stripeLineAdd[j].Description, "")

			return descA < descB
		})

		shift := len(stripeInvoice.Lines.Data) - 1

		// We collect the line IDs to match them with the Stripe line items from the response
		var lineIDs []string

		for _, stripeLine := range stripeLineAdd {
			lineIDs = append(lineIDs, stripeLine.Metadata[invoiceLineMetadataID])
			stripeLine.Metadata = nil
		}

		// Add Stripe line items to the Stripe invoice
		stripeInvoice, err = stripeClient.AddInvoiceLines(ctx, stripeclient.AddInvoiceLinesInput{
			StripeInvoiceID: stripeInvoice.ID,
			Lines:           stripeLineAdd,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add line items to invoice in stripe: %w", err)
		}

		// Add external line IDs
		newLines := stripeInvoice.Lines.Data[shift:]

		for idx, stripeLine := range newLines {
			result.AddLineExternalID(lineIDs[idx], stripeLine.ID)
		}
	}

	// Update Stripe lines on the Stripe invoice
	if len(stripeLinesUpdate) > 0 {
		// Sort the line items by description
		sort.Slice(stripeLinesUpdate, func(i, j int) bool {
			descA := lo.FromPtrOr(stripeLinesUpdate[i].Description, "")
			descB := lo.FromPtrOr(stripeLinesUpdate[j].Description, "")

			return descA < descB
		})

		stripeInvoice, err = stripeClient.UpdateInvoiceLines(ctx, stripeclient.UpdateInvoiceLinesInput{
			StripeInvoiceID: stripeInvoice.ID,
			Lines:           stripeLinesUpdate,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update line items in invoice in stripe: %w", err)
		}
	}

	// Remove Stripe lines from the Stripe invoice
	maps.Copy(existingStripeDiscounts, existingStripeLines)

	for stripeLineID := range existingStripeLines {
		stripeLinesRemove = append(stripeLinesRemove, &stripe.InvoiceRemoveLinesLineParams{
			ID:       lo.ToPtr(stripeLineID),
			Behavior: lo.ToPtr("delete"),
		})
	}

	if len(stripeLinesRemove) > 0 {
		stripeInvoice, err = stripeClient.RemoveInvoiceLines(ctx, stripeclient.RemoveInvoiceLinesInput{
			StripeInvoiceID: stripeInvoice.ID,
			Lines:           stripeLinesRemove,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to remove line items from invoice in stripe: %w", err)
		}
	}

	return result, nil
}

// isAllInteger checks if the invoice lines have only integer amount and quantity
func isAllInteger(calculator StripeCalculator, lines map[string]*billing.Line) bool {
	for _, line := range lines {
		if line.Type != billing.InvoiceLineTypeFee {
			continue
		}

		if !calculator.IsInteger(line.FlatFee.PerUnitAmount) || !line.FlatFee.Quantity.IsInteger() {
			return false
		}
	}

	return true
}
