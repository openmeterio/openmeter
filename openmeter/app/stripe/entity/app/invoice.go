package appstripeentityapp

import (
	"context"
	"fmt"
	"sort"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
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
	_, stripeClient, err := a.getStripeClient(ctx, "deleteInvoice", "invoice_id", invoice.ID, "stripe_invoice_id", invoice.ExternalIDs.GetInvoicingOrEmpty())
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
	_, stripeClient, err := a.getStripeClient(ctx, "finalizeInvoice", "invoice_id", invoice.ID, "stripe_invoice_id", invoice.ExternalIDs.GetInvoicingOrEmpty())
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

	// Stripe is the source of truth for invoice number
	// We set it on result to save it
	result.SetInvoiceNumber(stripeInvoice.Number)

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

	customerID := customer.CustomerID{
		Namespace: invoice.Namespace,
		ID:        invoice.Customer.CustomerID,
	}

	// Get the Stripe client
	_, stripeClient, err := a.getStripeClient(ctx, "createInvoice", "customer_id", customerID.ID)
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

	// Get customer billing profile
	customerOverrideWithDetails, err := a.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: customerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get customer override: %w", err)
	}

	// Create the invoice in Stripe
	createInvoiceParams := stripeclient.CreateInvoiceInput{
		// TODO: Automatic tax is currently always set to true because we onl support automated tax via Stripe.
		// In the future set it to false when:
		// 1. OpenMeter orchestrates tax calculation via Stripe API
		// 2. Tax collection is done by a separate third party app
		AutomaticTaxEnabled:          true,
		CollectionMethod:             customerOverrideWithDetails.MergedProfile.WorkflowConfig.Payment.CollectionMethod,
		Currency:                     invoice.Currency,
		DueDate:                      invoice.DueAt,
		StripeCustomerID:             stripeCustomerData.StripeCustomerID,
		StripeDefaultPaymentMethodID: stripeCustomerData.StripeDefaultPaymentMethodID,
	}

	stripeInvoice, err := stripeClient.CreateInvoice(ctx, createInvoiceParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice in stripe: %w", err)
	}

	// Return the result
	result := billing.NewUpsertInvoiceResult()
	result.SetExternalID(stripeInvoice.ID)

	// Stripe is the source of truth for invoice number
	// We set it on result to save it
	result.SetInvoiceNumber(stripeInvoice.Number)

	// Add lines to the Stripe invoice
	var stripeLineAdd []*stripe.InvoiceItemParams

	leafLines := invoice.GetLeafLinesWithConsolidatedTaxBehavior()

	// Iterate over the leaf lines
	for _, line := range leafLines {
		// Add discounts for line if any
		for _, discount := range line.FlattenDiscountsByID() {
			stripeLineAdd = append(stripeLineAdd, getDiscountStripeAddInvoiceItemParams(calculator, line, discount, stripeCustomerData.StripeCustomerID))
		}

		// Add line
		stripeLineAdd = append(stripeLineAdd, getStripeAddInvoiceItemParams(line, calculator, stripeCustomerData.StripeCustomerID))
	}

	// Sort the Stripe line items for deterministic order
	// TODO: use invoice summaries to group lines when Stripe supports it
	sortInvoiceLines(stripeLineAdd)

	newLines := []stripeclient.StripeInvoiceItemWithLineID{}

	// It is valid to have an invoice with no lines: this signifies that the customer has no outstanding
	// charges.
	if len(stripeLineAdd) > 0 {
		// Add Stripe line items to the Stripe invoice
		newLines, err = stripeClient.AddInvoiceLines(ctx, stripeclient.AddInvoiceLinesInput{
			StripeInvoiceID: stripeInvoice.ID,
			Lines:           stripeLineAdd,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add line items to invoice in stripe: %w", err)
		}
	}

	// Add external line IDs
	err = addResultExternalIDs(newLines, result)
	if err != nil {
		return nil, fmt.Errorf("failed to add external line IDs to result: %w", err)
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
	_, stripeClient, err := a.getStripeClient(ctx, "updateInvoice", "invoice_id", invoice.ID, "stripe_invoice_id", invoice.ExternalIDs.GetInvoicingOrEmpty())
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe client: %w", err)
	}

	// Get stripe customer data
	stripeCustomerData, err := a.StripeAppService.GetStripeCustomerData(ctx, appstripeentity.GetStripeCustomerDataInput{
		AppID: a.GetID(),
		CustomerID: customer.CustomerID{
			Namespace: invoice.Namespace,
			ID:        invoice.Customer.CustomerID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe customer data: %w", err)
	}

	// Update the invoice in Stripe
	stripeInvoice, err := stripeClient.UpdateInvoice(ctx, stripeclient.UpdateInvoiceInput{
		StripeInvoiceID: invoice.ExternalIDs.Invoicing,
		DueDate:         invoice.DueAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update invoice in stripe: %w", err)
	}

	// The result
	result := billing.NewUpsertInvoiceResult()
	result.SetExternalID(stripeInvoice.ID)

	// Stripe is the source of truth for invoice number
	// We set it on result to save it
	result.SetInvoiceNumber(stripeInvoice.Number)

	// Collect the existing line items
	// We use this to determine which line items to remove.
	// Existing lines that were not updated are removed.
	stripeLinesToRemove := make(map[string]bool)

	for _, stripeLine := range stripeInvoice.Lines.Data {
		stripeLinesToRemove[stripeLine.ID] = true
	}

	var (
		stripeLineAdd     []*stripe.InvoiceItemParams
		stripeLinesUpdate []*stripeclient.StripeInvoiceItemWithID
		stripeLinesRemove []string
	)

	leafLines := invoice.GetLeafLinesWithConsolidatedTaxBehavior()

	// Helper to get a Stripe line item by ID
	stripeLinesByID := make(map[string]*stripe.InvoiceLineItem)

	for _, stripeLine := range stripeInvoice.Lines.Data {
		stripeLinesByID[stripeLine.ID] = stripeLine
		// This allows looking up by stripe invoice item ID too (in case we ran into any inconsistencies going forward)
		if stripeLine.InvoiceItem != nil {
			stripeLinesByID[stripeLine.InvoiceItem.ID] = stripeLine
		}
	}

	// Iterate over the leaf lines
	for _, line := range leafLines {
		// Add discounts for line if any
		for _, discount := range line.FlattenDiscountsByID() {
			// Update discount line item if it already has an external ID
			if discount.ExternalIDs.Invoicing != "" {
				// Get the Stripe line item for the discount
				stripeLine, ok := stripeLinesByID[discount.ExternalIDs.Invoicing]
				if !ok {
					return nil, fmt.Errorf("discount not found in stripe lines: %s", discount.ExternalIDs.Invoicing)
				}

				// Exclude line from the remove list as it is updated
				delete(stripeLinesToRemove, stripeLine.ID)

				result.AddLineDiscountExternalID(discount.ID, line.ExternalIDs.Invoicing)

				stripeLinesUpdate = append(stripeLinesUpdate, getDiscountStripeUpdateInvoiceItemParams(calculator, line, discount, stripeLine))
			} else {
				// Add the discount line item if it doesn't have an external ID yet
				stripeLineAdd = append(stripeLineAdd, getDiscountStripeAddInvoiceItemParams(calculator, line, discount, stripeCustomerData.StripeCustomerID))
			}
		}

		// Update line item if it already has an external ID
		if line.ExternalIDs.Invoicing != "" {
			// Get the Stripe line item for the line
			stripeLine, ok := stripeLinesByID[line.ExternalIDs.Invoicing]
			if !ok {
				return nil, fmt.Errorf("line not found in stripe lines: %s", line.ExternalIDs.Invoicing)
			}

			// Exclude line from the remove list as it is updated
			delete(stripeLinesToRemove, stripeLine.ID)

			// Add external line ID to the result
			result.AddLineExternalID(line.ID, stripeLine.ID)

			// Get stripe update line params
			stripeLinesUpdate = append(stripeLinesUpdate, getStripeUpdateInvoiceItemParams(calculator, line, stripeLine))
		} else {
			// Add the line item if it doesn't have an external ID yet
			stripeLineAdd = append(stripeLineAdd, getStripeAddInvoiceItemParams(line, calculator, stripeCustomerData.StripeCustomerID))
		}
	}

	// Add Stripe lines to the Stripe invoice
	if len(stripeLineAdd) > 0 {
		// Sort the line items by description
		sortInvoiceLines(stripeLineAdd)

		// Add Stripe line items to the Stripe invoice
		newInvoiceItems, err := stripeClient.AddInvoiceLines(ctx, stripeclient.AddInvoiceLinesInput{
			StripeInvoiceID: stripeInvoice.ID,
			Lines:           stripeLineAdd,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to add line items to invoice in stripe: %w", err)
		}

		err = addResultExternalIDs(newInvoiceItems, result)
		if err != nil {
			return nil, fmt.Errorf("failed to add external line IDs to result: %w", err)
		}
	}

	// Update Stripe lines on the Stripe invoice
	if len(stripeLinesUpdate) > 0 {
		// Sort the line items by description
		sortInvoiceLines(stripeLinesUpdate)

		_, err = stripeClient.UpdateInvoiceLines(ctx, stripeclient.UpdateInvoiceLinesInput{
			StripeInvoiceID: stripeInvoice.ID,
			Lines:           stripeLinesUpdate,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update line items in invoice in stripe: %w", err)
		}
	}

	// Remove Stripe lines from the Stripe invoice
	stripeLinesRemove = append(stripeLinesRemove, lo.Keys(stripeLinesToRemove)...)

	if len(stripeLinesRemove) > 0 {
		err = stripeClient.RemoveInvoiceLines(ctx, stripeclient.RemoveInvoiceLinesInput{
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
	stripe.InvoiceItemParams | stripeclient.StripeInvoiceItemWithID
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

// getDiscountStripeUpdateInvoiceItemParams returns the Stripe line item for a discount
func getDiscountStripeUpdateInvoiceItemParams(
	calculator StripeCalculator,
	line *billing.Line,
	discount billing.LineDiscount,
	stripeLine *stripe.InvoiceLineItem,
) *stripeclient.StripeInvoiceItemWithID {
	return &stripeclient.StripeInvoiceItemWithID{
		ID:                stripeLine.ID,
		InvoiceItemParams: getDiscountStripeInvoiceItemParams(calculator, line, discount),
	}
}

// getDiscountStripeInvoiceItemParams returns the Stripe line item for a discount
func getDiscountStripeInvoiceItemParams(calculator StripeCalculator, line *billing.Line, discount billing.LineDiscount) *stripe.InvoiceItemParams {
	name := getDiscountLineName(line, discount)
	period := getPeriod(line)

	addParams := &stripe.InvoiceItemParams{
		Description: lo.ToPtr(name),
		Amount:      lo.ToPtr(-calculator.RoundToAmount(discount.Amount)),
		Period:      period,
		Metadata: map[string]string{
			invoiceLineMetadataID:   discount.ID,
			invoiceLineMetadataType: invoiceLineMetadataTypeDiscount,
		},
	}

	return applyTaxSettingsToInvoiceItem(addParams, line)
}

func getDiscountStripeAddInvoiceItemParams(calculator StripeCalculator, line *billing.Line, discount billing.LineDiscount, stripeCustomerID string) *stripe.InvoiceItemParams {
	params := getDiscountStripeInvoiceItemParams(calculator, line, discount)
	// Customer is required for adds
	params.Customer = stripe.String(stripeCustomerID)
	return params
}

func applyTaxSettingsToInvoiceItem(add *stripe.InvoiceItemParams, line *billing.Line) *stripe.InvoiceItemParams {
	if line.TaxConfig != nil && !lo.IsEmpty(line.TaxConfig) {
		if line.TaxConfig.Behavior != nil {
			add.TaxBehavior = getStripeTaxBehavior(line.TaxConfig.Behavior)
		}

		if line.TaxConfig.Stripe != nil {
			add.TaxCode = stripe.String(line.TaxConfig.Stripe.Code)
		}
	}

	return add
}

// getStripeUpdateInvoiceItemParams returns the Stripe update line params
func getStripeUpdateInvoiceItemParams(
	calculator StripeCalculator,
	line *billing.Line,
	stripeLine *stripe.InvoiceLineItem,
) *stripeclient.StripeInvoiceItemWithID {
	return &stripeclient.StripeInvoiceItemWithID{
		ID:                stripeLine.ID,
		InvoiceItemParams: getStripeInvoiceItemParams(line, calculator),
	}
}

// getStripeAddLinesLineParams returns the Stripe line item
func getStripeInvoiceItemParams(line *billing.Line, calculator StripeCalculator) *stripe.InvoiceItemParams {
	description := getLineName(line)
	period := getPeriod(line)
	amount := line.Totals.Amount

	// Handle usage based commitments like minimum spend
	if amount.IsZero() {
		// ChargesTotal is the amount of value of the line that are due to additional charges.
		// If the line is a commitment we use the total charges.
		amount = line.Totals.ChargesTotal
	}

	// If the line has a quantity we add the quantity and per unit amount to the description
	if line.FlatFee.Quantity.GreaterThan(alpacadecimal.NewFromInt(1)) {
		description = fmt.Sprintf(
			"%s (%s x %s)",
			description,
			calculator.FormatQuantity(line.FlatFee.Quantity),
			calculator.FormatAmount(line.FlatFee.PerUnitAmount),
		)
	}

	// Otherwise we add the calculated total with with quantity one
	addParams := &stripe.InvoiceItemParams{
		Description: lo.ToPtr(description),
		Amount:      lo.ToPtr(calculator.RoundToAmount(amount)),
		Period:      period,
		Metadata: map[string]string{
			invoiceLineMetadataID:   line.ID,
			invoiceLineMetadataType: invoiceLineMetadataTypeLine,
		},
	}

	return applyTaxSettingsToInvoiceItem(addParams, line)
}

// getStripeAddInvoiceItemParams returns the Stripe line item
func getStripeAddInvoiceItemParams(line *billing.Line, calculator StripeCalculator, stripeCustomerID string) *stripe.InvoiceItemParams {
	params := getStripeInvoiceItemParams(line, calculator)
	params.Customer = stripe.String(stripeCustomerID)
	return params
}

// getPeriod returns the period
func getPeriod(line *billing.Line) *stripe.InvoiceItemPeriodParams {
	return &stripe.InvoiceItemPeriodParams{
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

// getStripeTaxBehavior returns the Stripe tax behavior from a TaxBehavior
func getStripeTaxBehavior(tb *productcatalog.TaxBehavior) *string {
	if tb == nil {
		return nil
	}

	switch *tb {
	case productcatalog.InclusiveTaxBehavior:
		return lo.ToPtr(string(stripe.PriceCurrencyOptionsTaxBehaviorInclusive))
	case productcatalog.ExclusiveTaxBehavior:
		return lo.ToPtr(string(stripe.PriceCurrencyOptionsTaxBehaviorExclusive))
	default:
		return nil
	}
}

// addResultExternalIDs adds the Stripe line item IDs to the result external IDs
func addResultExternalIDs(
	newLines []stripeclient.StripeInvoiceItemWithLineID,
	result *billing.UpsertInvoiceResult,
) error {
	// Check if we have the same number of params and new lines

	for idx, stripeLine := range newLines {
		// Get the line ID from the param metadata
		// We always read it from params as it's our source of truth
		id, ok := newLines[idx].Metadata[invoiceLineMetadataID]
		if !ok {
			return fmt.Errorf("line ID not found in stripe line metadata")
		}

		// Get the line type from the param metadata
		// We always read it from params as it's our source of truth
		lineType, ok := newLines[idx].Metadata[invoiceLineMetadataType]
		if !ok {
			return fmt.Errorf("line type not found in stripe line metadata")
		}

		// Add line discount external ID
		if lineType == invoiceLineMetadataTypeDiscount {
			result.AddLineDiscountExternalID(id, stripeLine.LineID)
			continue
		}

		// Add line external ID
		result.AddLineExternalID(id, stripeLine.LineID)
	}

	return nil
}
