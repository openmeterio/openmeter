package appstripeentityapp

import (
	"context"
	"fmt"
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
// Upsert is idempotent and can be used to create or update an invoice.
// In case of failure the upsert should be retried.
//
// TODO: should we split invoice create and lines adds to make retries more robust?
// Currently if the create fails between the create and add lines we can end up with
// an invoice without lines.
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

	leafLines := invoice.GetLeafLines()

	// Check if we have any non integer amount or quantity
	// We use this to determinate if we add alreay calculated total or per unit amount and quantity to the Stripe line item
	// We decide this globally for all line items in the invoice for consistency of the invoice.
	isInteger := calculator.IsAllLinesInteger(leafLines)

	// Iterate over the leaf lines
	for _, line := range leafLines {
		// Add discounts for line if any
		line.Discounts.ForEach(func(discounts []billing.LineDiscount) {
			for _, discount := range discounts {
				stripeLineAdd = append(stripeLineAdd, getDiscountStripeAddLinesLineParams(calculator, line, discount))
			}
		})

		// Add line
		stripeLineAdd = append(stripeLineAdd, getStripeAddLinesLineParams(isInteger, line, calculator))
	}

	// Sort the Stripe line items for deterministic order
	// TODO: use invoice summaries to group lines when Stripe supports it
	sortInvoiceLines(stripeLineAdd)

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
		id, ok := stripeLine.Metadata[invoiceLineMetadataID]
		if !ok {
			return nil, fmt.Errorf("missing line ID in metadata")
		}

		// Add line discount external ID
		if stripeLine.Metadata[invoiceLineMetadataType] == invoiceLineMetadataTypeDiscount {
			result.AddLineDiscountExternalID(id, stripeLine.ID)
			continue
		}

		// Add line external ID
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

	// The result
	result := billing.NewUpsertInvoiceResult()
	result.SetExternalID(stripeInvoice.ID)
	result.SetInvoiceNumber(stripeInvoice.Number)

	// Collect the existing line items
	// We use this to determine which line items to remove.
	// Existing lines that were not updated are removed.
	stripeLinesToRemove := make(map[string]bool)

	for _, stripeLine := range stripeInvoice.Lines.Data {
		stripeLinesToRemove[stripeLine.ID] = true
	}

	var (
		stripeLineAdd     []*stripe.InvoiceAddLinesLineParams
		stripeLinesUpdate []*stripe.InvoiceUpdateLinesLineParams
		stripeLinesRemove []*stripe.InvoiceRemoveLinesLineParams
	)

	leafLines := invoice.GetLeafLines()

	// Check if we have any non integer amount or quantity
	// We use this to determinate if we add alreay calculated total or per unit amount and quantity to the Stripe line item
	// We decide this globally for all line items in the invoice for consistency of the invoice.
	isInteger := calculator.IsAllLinesInteger(leafLines)

	// Check if a line item already exists in the Stripe invoice
	// Used to determine if we should add or update the line item.
	isExisting := func(lineId string, lineType string) (*stripe.InvoiceLineItem, bool) {
		for _, stripeLine := range stripeInvoice.Lines.Data {
			if stripeLine.Metadata[invoiceLineMetadataID] == lineId && stripeLine.Metadata[invoiceLineMetadataType] == lineType {
				return stripeLine, true
			}
		}

		return nil, false
	}

	// Iterate over the leaf lines
	for _, line := range leafLines {
		// Add discounts
		line.Discounts.ForEach(func(discounts []billing.LineDiscount) {
			// Discounts
			for _, discount := range discounts {
				// Add line item if it doesn't exist
				// FIXME: discounts don't have an external ID
				// if line.ExternalIDs.Invoicing == "" {
				stripeLine, isUpdate := isExisting(discount.ID, invoiceLineMetadataTypeDiscount)

				if isUpdate {
					// Exclude line from the remove list as it is updated
					delete(stripeLinesToRemove, stripeLine.ID)

					result.AddLineDiscountExternalID(discount.ID, line.ExternalIDs.Invoicing)

					stripeLinesUpdate = append(stripeLinesUpdate, getDiscountStripeUpdateLinesLineParams(calculator, line, discount, stripeLine))
				} else {
					stripeLineAdd = append(stripeLineAdd, getDiscountStripeAddLinesLineParams(calculator, line, discount))
				}
			}
		})

		// Add line
		// FIXME: set external ID in the test invoice
		// if line.ExternalIDs.Invoicing == "" {
		stripeLine, isUpdate := isExisting(line.ID, invoiceLineMetadataTypeLine)

		if isUpdate {
			// Exclude line from the remove list as it is updated
			delete(stripeLinesToRemove, stripeLine.ID)

			// Add external line ID to the result
			result.AddLineExternalID(line.ID, stripeLine.ID)

			// Get stripe update line params
			stripeLinesUpdate = append(stripeLinesUpdate, getStripeUpdateLinesLineParams(isInteger, calculator, line, stripeLine))
		} else {
			// Get stripe add line params
			stripeLineAdd = append(stripeLineAdd, getStripeAddLinesLineParams(isInteger, line, calculator))
		}
	}

	// Add Stripe lines to the Stripe invoice
	if len(stripeLineAdd) > 0 {
		// Sort the line items by description
		sortInvoiceLines(stripeLineAdd)

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
			// Add line discount external ID
			if stripeLine.Metadata[invoiceLineMetadataType] == invoiceLineMetadataTypeDiscount {
				result.AddLineDiscountExternalID(lineIDs[idx], stripeLine.ID)
				continue
			}

			// Add line external ID
			result.AddLineExternalID(lineIDs[idx], stripeLine.ID)
		}
	}

	// Update Stripe lines on the Stripe invoice
	if len(stripeLinesUpdate) > 0 {
		// Sort the line items by description
		sortInvoiceLines(stripeLinesUpdate)

		stripeInvoice, err = stripeClient.UpdateInvoiceLines(ctx, stripeclient.UpdateInvoiceLinesInput{
			StripeInvoiceID: stripeInvoice.ID,
			Lines:           stripeLinesUpdate,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update line items in invoice in stripe: %w", err)
		}
	}

	// Remove Stripe lines from the Stripe invoice
	for stripeLineID := range stripeLinesToRemove {
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

type StripeInvoiceLineOperationParams interface {
	stripe.InvoiceAddLinesLineParams | stripe.InvoiceUpdateLinesLineParams
}

// sortInvoiceLines sorts the lines by description
func sortInvoiceLines[K StripeInvoiceLineOperationParams](stripeLineAdd []*K) {
	sort.Slice(stripeLineAdd, func(i, j int) bool {
		var (
			descA *string
			descB *string
		)

		// Go generics can't handle two structs with common fields
		// We need to switch on the type
		switch params := any(stripeLineAdd).(type) {
		case []*stripe.InvoiceAddLinesLineParams:
			descA = params[i].Description
			descB = params[j].Description

		case []*stripe.InvoiceUpdateLinesLineParams:
			descA = params[i].Description
			descB = params[j].Description
		}

		a := lo.FromPtrOr(descA, "")
		b := lo.FromPtrOr(descB, "")

		return a < b
	})
}

// getDiscountStripeUpdateLinesLineParams returns the Stripe line item for a discount
func getDiscountStripeUpdateLinesLineParams(
	calculator StripeCalculator,
	line *billing.Line,
	discount billing.LineDiscount,
	stripeLine *stripe.InvoiceLineItem,
) *stripe.InvoiceUpdateLinesLineParams {
	// Update is similar to add so we reuse the add method
	params := getDiscountStripeAddLinesLineParams(calculator, line, discount)

	return &stripe.InvoiceUpdateLinesLineParams{
		ID:          lo.ToPtr(stripeLine.ID),
		Description: params.Description,
		Amount:      params.Amount,
		Quantity:    params.Quantity,
		Metadata:    stripeLine.Metadata,
		Period: &stripe.InvoiceUpdateLinesLinePeriodParams{
			Start: params.Period.Start,
			End:   params.Period.End,
		},
	}
}

// getDiscountStripeAddLinesLineParams returns the Stripe line item for a discount
func getDiscountStripeAddLinesLineParams(calculator StripeCalculator, line *billing.Line, discount billing.LineDiscount) *stripe.InvoiceAddLinesLineParams {
	name := getDiscountLineName(line, discount)
	period := getPeriod(line)

	return &stripe.InvoiceAddLinesLineParams{
		Description: lo.ToPtr(name),
		Amount:      lo.ToPtr(-calculator.RoundToAmount(discount.Amount)),
		Quantity:    lo.ToPtr(int64(1)),
		Period:      period,
		Metadata: map[string]string{
			// TODO (OM-1062): should we use the discount ID as the external ID?
			invoiceLineMetadataID:   discount.ID,
			invoiceLineMetadataType: invoiceLineMetadataTypeDiscount,
		},
	}
}

// getStripeUpdateLinesLineParams returns the Stripe update line params
func getStripeUpdateLinesLineParams(
	isInteger bool,
	calculator StripeCalculator,
	line *billing.Line,
	stripeLine *stripe.InvoiceLineItem,
) *stripe.InvoiceUpdateLinesLineParams {
	// Update is similar to add so we reuse the add method
	params := getStripeAddLinesLineParams(isInteger, line, calculator)

	return &stripe.InvoiceUpdateLinesLineParams{
		ID:          lo.ToPtr(stripeLine.ID),
		Description: params.Description,
		Amount:      params.Amount,
		Quantity:    params.Quantity,
		Period: &stripe.InvoiceUpdateLinesLinePeriodParams{
			Start: params.Period.Start,
			End:   params.Period.End,
		},
		Metadata: stripeLine.Metadata,
	}
}

// getStripeAddLinesLineParams returns the Stripe line item
func getStripeAddLinesLineParams(isInteger bool, line *billing.Line, calculator StripeCalculator) *stripe.InvoiceAddLinesLineParams {
	name := getLineName(line)
	period := getPeriod(line)

	// If the per unit amount and quantity can be represented in stripe as integer we add the line item
	if isInteger {
		return &stripe.InvoiceAddLinesLineParams{
			Description: lo.ToPtr(name),
			Amount:      lo.ToPtr(calculator.RoundToAmount(line.FlatFee.PerUnitAmount)),
			Quantity:    lo.ToPtr(line.FlatFee.Quantity.IntPart()),
			Period:      period,
			Metadata: map[string]string{
				invoiceLineMetadataID:   line.ID,
				invoiceLineMetadataType: invoiceLineMetadataTypeLine,
			},
		}
	}

	amount := line.Totals.Amount

	// Handle usage based commitments like minimum spend
	if amount.IsZero() {
		// ChargesTotal is the amount of value of the line that are due to additional charges.
		// If the line is a commitment we use the total charges.
		amount = line.Totals.ChargesTotal
	}

	// Otherwise we add the calculated total with with quantity one
	return &stripe.InvoiceAddLinesLineParams{
		Description: lo.ToPtr(name),
		Amount:      lo.ToPtr(calculator.RoundToAmount(amount)),
		Quantity:    lo.ToPtr(int64(1)),
		Period:      period,
		Metadata: map[string]string{
			invoiceLineMetadataID:   line.ID,
			invoiceLineMetadataType: invoiceLineMetadataTypeLine,
		},
	}
}

// getPeriod returns the period
func getPeriod(line *billing.Line) *stripe.InvoiceAddLinesLinePeriodParams {
	return &stripe.InvoiceAddLinesLinePeriodParams{
		Start: lo.ToPtr(line.Period.Start.Unix()),
		End:   lo.ToPtr(line.Period.End.Unix()),
	}
}

// getDiscountLineName returns the line name
func getDiscountLineName(line *billing.Line, discount billing.LineDiscount) string {
	name := line.Name
	if discount.Description != nil {
		name = fmt.Sprintf("%s (%s)", name, *discount.Description)
	}

	return name
}

// getLineName returns the line name
func getLineName(line *billing.Line) string {
	name := line.Name
	if line.Description != nil {
		name = fmt.Sprintf("%s (%s)", name, *line.Description)
	}

	return name
}
