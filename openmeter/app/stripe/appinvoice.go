package appstripe

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"sort"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/app"
	stripeclient "github.com/openmeterio/openmeter/openmeter/app/stripe/client"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/taxcode"
)

const (
	invoiceLineMetadataID           = "om_line_id"
	invoiceLineMetadataType         = "om_line_type"
	invoiceLineMetadataTypeLine     = "line"
	invoiceLineMetadataTypeDiscount = "discount"
	invoiceLineMetadataTypeCredit   = "credit"
)

var _ billing.InvoicingApp = (*App)(nil)

// ValidateStandardInvoice validates the invoice for the app
func (a appOperations) ValidateStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
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

// UpsertStandardInvoice upserts the invoice for the app
// Upsert is idempotent and can be used to create or update an invoice.
// In case of failure the upsert should be retried.
//
// TODO: should we split invoice create and lines adds to make retries more robust?
// Currently if the create fails between the create and add lines we can end up with
// an invoice without lines.
func (a appOperations) UpsertStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
	// Create the invoice in Stripe.
	if invoice.ExternalIDs.Invoicing == "" {
		return a.createInvoice(ctx, invoice)
	}

	// Update the invoice in Stripe.
	return a.updateInvoice(ctx, invoice)
}

// WouldStandardInvoiceBeEmpty reports invoices that would reach Stripe without line items or with
// nothing to pay.
func (a appOperations) WouldStandardInvoiceBeEmpty(invoice billing.StandardInvoice) (bool, error) {
	return len(invoice.GetLeafLinesWithResolvedTaxConfig()) == 0 || invoice.Totals.Total.IsZero(), nil
}

// DeleteStandardInvoice deletes the invoice for the app
func (a appOperations) DeleteStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) error {
	// If the invoice has not been synced to Stripe yet, there is no Stripe invoice to delete.
	if invoice.ExternalIDs.Invoicing == "" {
		return nil
	}

	// Get the Stripe client
	_, stripeClient, err := a.getStripeClient(ctx, "deleteInvoice", "invoice_id", invoice.ID, "stripe_invoice_id", invoice.ExternalIDs.GetInvoicingOrEmpty())
	if err != nil {
		return fmt.Errorf("failed to get stripe client: %w", err)
	}

	// Delete the invoice in Stripe
	err = stripeClient.DeleteInvoice(ctx, stripeclient.DeleteInvoiceInput{
		StripeInvoiceID: invoice.ExternalIDs.Invoicing,
	})
	if err != nil {
		// The Stripe invoice is in an unknown state, so a later create must not get it back.
		return errors.Join(err, a.StripeAppService.IncrementInvoiceGeneration(ctx, invoice.GetInvoiceID()))
	}

	return nil
}

// FinalizeStandardInvoice finalizes the invoice for the app
func (a appOperations) FinalizeStandardInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.FinalizeStandardInvoiceResult, error) {
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
		// If customer tax location is invalid but tax is not enforced,
		// we can finalize the invoice without tax calculation.
		if stripeclient.IsStripeInvoiceCustomerTaxLocationInvalidError(err) {
			if invoice.Workflow.Config.Tax.Enforced {
				return nil, fmt.Errorf("tax enforced but stripe tax returns error: %w", err)
			}

			// We can finalize the invoice without tax calculation.
			_, err = stripeClient.UpdateInvoice(ctx, stripeclient.UpdateInvoiceInput{
				// Disable tax calculation
				AutomaticTaxEnabled: false,
				StripeInvoiceID:     invoice.ExternalIDs.Invoicing,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update invoice in stripe to disable tax calculation: %w", err)
			}

			// Finalize the invoice again
			return a.FinalizeStandardInvoice(ctx, invoice)
		}

		return nil, fmt.Errorf("failed to finalize invoice in stripe: %w", err)
	}

	// Result
	result := billing.NewFinalizeStandardInvoiceResult()

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

// resolveLinesTaxCodes looks up the Tax Codes the leaf lines reference by ID, soft-deleted
// ones included. Every stored ID must resolve: lines with an ID got it through validated
// writes and FKs keep stored references pointing at existing rows, so a miss means corrupt
// data, not something invoicing can skip.
func (a appOperations) resolveLinesTaxCodes(ctx context.Context, namespace string, lines []billing.DetailedLineWithResolvedTaxConfig) (productcatalog.TaxCodesByID, error) {
	ids := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if line.TaxConfig == nil || line.TaxConfig.TaxCodeID == nil || *line.TaxConfig.TaxCodeID == "" {
			continue
		}

		ids[*line.TaxConfig.TaxCodeID] = struct{}{}
	}

	if len(ids) == 0 {
		return nil, nil
	}

	taxCodes, err := a.TaxCodeResolver.BatchResolve(ctx, namespace, slices.Collect(maps.Keys(ids))...)
	if err != nil {
		return nil, err
	}

	for id := range ids {
		if _, ok := taxCodes.Get(id); !ok {
			return nil, fmt.Errorf("tax code %s: %w", id, taxcode.NewTaxCodeNotFoundError(id))
		}
	}

	return taxCodes, nil
}

// createInvoice creates the invoice for the app
func (a appOperations) createInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
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
	stripeCustomerData, err := a.StripeAppService.GetStripeCustomerData(ctx, GetStripeCustomerDataInput{
		AppID:      a.GetID(),
		CustomerID: customerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe customer data: %w", err)
	}

	// Create the invoice in Stripe
	createInvoiceParams := stripeclient.CreateInvoiceInput{
		AppID:                        a.GetID(),
		CustomerID:                   customerID,
		InvoiceID:                    invoice.ID,
		AutomaticTaxEnabled:          invoice.Workflow.Config.Tax.Enabled,
		CollectionMethod:             invoice.Workflow.Config.Payment.CollectionMethod,
		Currency:                     invoice.Currency,
		StripeCustomerID:             stripeCustomerData.StripeCustomerID,
		StripeDefaultPaymentMethodID: stripeCustomerData.StripeDefaultPaymentMethodID,
	}

	// Set the days until due if the invoice is sent
	if invoice.Workflow.Config.Payment.CollectionMethod == billing.CollectionMethodSendInvoice {
		daysUntilDue, _, ok := invoice.Workflow.Config.Invoicing.DueAfter.DaysDecimal().Int64(0)
		if !ok {
			return nil, fmt.Errorf("failed to get days until due")
		}

		// This is a workaround to handle the case when someone defines the due in months like P1M instead of days like P30D.
		// With P1M the library will truncate the period to 0 days, which is not what we want.
		// Defining due in days like P30D is preferred over P1M.
		if daysUntilDue == 0 {
			futureDueAt, _ := invoice.Workflow.Config.Invoicing.DueAfter.AddTo(time.Now())
			daysUntilDue = int64(math.Round(time.Until(futureDueAt).Hours() / 24))
		}

		createInvoiceParams.DaysUntilDue = lo.ToPtr(daysUntilDue)
	}

	stripeInvoice, err := a.createStripeInvoice(ctx, stripeClient, invoice.GetInvoiceID(), createInvoiceParams)
	if err != nil {
		return nil, err
	}

	// Return the result
	result := billing.NewUpsertStandardInvoiceResult()
	result.SetExternalID(stripeInvoice.ID)

	// Stripe is the source of truth for invoice number
	// We set it on result to save it
	result.SetInvoiceNumber(stripeInvoice.Number)

	// Add lines to the Stripe invoice
	var stripeLineAdd []*stripe.InvoiceItemParams

	leafLines := invoice.GetLeafLinesWithResolvedTaxConfig()

	taxCodes, err := a.resolveLinesTaxCodes(ctx, invoice.Namespace, leafLines)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	// Iterate over the leaf lines
	for _, line := range leafLines {
		// Add discounts for line if any
		for _, discount := range line.AmountDiscounts {
			stripeLineAdd = append(stripeLineAdd, getDiscountStripeAddInvoiceItemParams(calculator, line, discount, stripeCustomerData.StripeCustomerID, taxCodes))
		}

		// Add applied credits for line if any
		for _, credit := range line.CreditsApplied {
			if credit.CreditRealizationID == "" {
				return nil, fmt.Errorf("credit realization ID is required")
			}

			stripeLineAdd = append(stripeLineAdd, getCreditStripeAddInvoiceItemParams(calculator, line, credit, stripeCustomerData.StripeCustomerID, taxCodes))
		}

		// Add line
		stripeLineAdd = append(stripeLineAdd, getStripeAddInvoiceItemParams(line, calculator, stripeCustomerData.StripeCustomerID, taxCodes))
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

// createStripeInvoice creates the invoice in Stripe and reads it back. Stripe answers a create request
// with the response it stored for the idempotency key, which can be an earlier failure or an invoice
// that was deleted since. When that happens the generation is incremented, so the next create
// request uses a new key.
func (a appOperations) createStripeInvoice(
	ctx context.Context,
	stripeClient stripeclient.StripeAppClient,
	invoiceID billing.InvoiceID,
	input stripeclient.CreateInvoiceInput,
) (*stripe.Invoice, error) {
	// A stored response that points at a missing invoice is replaced by one create with the new key.
	const maxAttempts = 2

	var getErr error

	for range maxAttempts {
		generation, err := a.StripeAppService.GetLatestInvoiceGeneration(ctx, invoiceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get stripe invoice generation: %w", err)
		}

		input.Generation = generation

		stripeInvoice, err := stripeClient.CreateInvoice(ctx, input)
		if err != nil {
			err = fmt.Errorf("failed to create invoice in stripe: %w", err)

			// An error Stripe answered with is replayed for the key, so the key is given up. Without
			// an answer the invoice may exist in Stripe, and only the same key gives it back.
			if stripeclient.IsStripeInvoiceCreateRejectedError(err) {
				err = errors.Join(err, a.StripeAppService.IncrementInvoiceGeneration(ctx, invoiceID))
			}

			return nil, err
		}

		_, getErr = stripeClient.GetInvoice(ctx, stripeclient.GetInvoiceInput{
			StripeInvoiceID: stripeInvoice.ID,
		})
		if getErr == nil {
			return stripeInvoice, nil
		}

		if !givesUpCreatedStripeInvoice(getErr) {
			break
		}

		if err := a.StripeAppService.IncrementInvoiceGeneration(ctx, invoiceID); err != nil {
			return nil, errors.Join(getErr, err)
		}
	}

	return nil, fmt.Errorf("failed to get created invoice from stripe: %w", getErr)
}

// givesUpCreatedStripeInvoice decides whether a failed read of a just created Stripe invoice means
// the invoice is gone, so the create has to be repeated with a new idempotency key.
// Any other failure says nothing about the invoice: the key is kept, and the next sync gets the
// same Stripe invoice back instead of leaving it behind for a second one.
func givesUpCreatedStripeInvoice(err error) bool {
	return stripeclient.IsStripeInvoiceNotFoundError(err)
}

// updateInvoice update the invoice for the app
func (a appOperations) updateInvoice(ctx context.Context, invoice billing.StandardInvoice) (*billing.UpsertStandardInvoiceResult, error) {
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
	stripeCustomerData, err := a.StripeAppService.GetStripeCustomerData(ctx, GetStripeCustomerDataInput{
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
		AutomaticTaxEnabled: invoice.Workflow.Config.Tax.Enabled,
		StripeInvoiceID:     invoice.ExternalIDs.Invoicing,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update invoice in stripe: %w", err)
	}

	// The result
	result := billing.NewUpsertStandardInvoiceResult()
	result.SetExternalID(stripeInvoice.ID)

	// Stripe is the source of truth for invoice number
	// We set it on result to save it
	result.SetInvoiceNumber(stripeInvoice.Number)

	// Collect the existing line items
	// We use this to determine which line items to remove.
	// Existing lines that were not updated are removed.
	stripeLinesToRemove := make(map[string]bool)

	var (
		stripeLineAdd     []*stripe.InvoiceItemParams
		stripeLinesUpdate []*stripeclient.StripeInvoiceItemWithID
		stripeLinesRemove []string
	)

	// Get the existing line items from Stripe to build the maps
	stripeInvoiceLineItems, err := stripeClient.ListInvoiceLineItems(ctx, stripeInvoice.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing line items from stripe: %w", err)
	}

	stripeLinesByID := make(map[string]*stripe.InvoiceLineItem)
	stripeCreditLinesByCreditRealizationID := make(map[string]*stripe.InvoiceLineItem)

	for _, stripeLine := range stripeInvoiceLineItems {
		// We set all to true and the code later clears the ones that we keep
		stripeLinesToRemove[stripeLine.ID] = true

		stripeLinesByID[stripeLine.ID] = stripeLine
		// This allows looking up by stripe invoice item ID too (in case we ran into any inconsistencies going forward)
		if stripeLine.InvoiceItem != nil {
			stripeLinesByID[stripeLine.InvoiceItem.ID] = stripeLine
		}

		if stripeLine.Metadata[invoiceLineMetadataType] == invoiceLineMetadataTypeCredit {
			if creditRealizationID := stripeLine.Metadata[invoiceLineMetadataID]; creditRealizationID != "" {
				stripeCreditLinesByCreditRealizationID[creditRealizationID] = stripeLine
			}
		}
	}

	// Iterate over the leaf lines
	leafLines := invoice.GetLeafLinesWithResolvedTaxConfig()

	taxCodes, err := a.resolveLinesTaxCodes(ctx, invoice.Namespace, leafLines)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tax codes: %w", err)
	}

	for _, line := range leafLines {
		amountDiscountsById, err := line.AmountDiscounts.GetByID()
		if err != nil {
			return nil, fmt.Errorf("failed to get amount discounts by ID: %w", err)
		}

		// Add discounts for line if any
		for _, discount := range amountDiscountsById {
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

				stripeLinesUpdate = append(stripeLinesUpdate, getDiscountStripeUpdateInvoiceItemParams(calculator, line, discount, stripeLine, taxCodes))
			} else {
				// Add the discount line item if it doesn't have an external ID yet
				stripeLineAdd = append(stripeLineAdd, getDiscountStripeAddInvoiceItemParams(calculator, line, discount, stripeCustomerData.StripeCustomerID, taxCodes))
			}
		}

		// Add or update applied credit line items.
		for _, credit := range line.CreditsApplied {
			if credit.CreditRealizationID == "" {
				return nil, fmt.Errorf("credit realization ID is required")
			}

			if stripeLine, ok := stripeCreditLinesByCreditRealizationID[credit.CreditRealizationID]; ok {
				delete(stripeLinesToRemove, stripeLine.ID)

				stripeLinesUpdate = append(stripeLinesUpdate, getCreditStripeUpdateInvoiceItemParams(calculator, line, credit, stripeLine, taxCodes))
			} else {
				stripeLineAdd = append(stripeLineAdd, getCreditStripeAddInvoiceItemParams(calculator, line, credit, stripeCustomerData.StripeCustomerID, taxCodes))
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
			stripeLinesUpdate = append(stripeLinesUpdate, getStripeUpdateInvoiceItemParams(calculator, line, stripeLine, taxCodes))
		} else {
			// Add the line item if it doesn't have an external ID yet
			stripeLineAdd = append(stripeLineAdd, getStripeAddInvoiceItemParams(line, calculator, stripeCustomerData.StripeCustomerID, taxCodes))
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

		a := lo.FromPtr(descA)
		b := lo.FromPtr(descB)

		return a < b
	})
}

// getDiscountStripeUpdateInvoiceItemParams returns the Stripe line item for a discount
func getDiscountStripeUpdateInvoiceItemParams(
	calculator StripeCalculator,
	line billing.DetailedLineWithResolvedTaxConfig,
	discount billing.AmountLineDiscountManaged,
	stripeLine *stripe.InvoiceLineItem,
	taxCodes productcatalog.TaxCodesByID,
) *stripeclient.StripeInvoiceItemWithID {
	return &stripeclient.StripeInvoiceItemWithID{
		ID:                stripeLine.ID,
		InvoiceItemParams: getDiscountStripeInvoiceItemParams(calculator, line, discount, taxCodes),
	}
}

// getDiscountStripeInvoiceItemParams returns the Stripe line item for a discount
func getDiscountStripeInvoiceItemParams(calculator StripeCalculator, line billing.DetailedLineWithResolvedTaxConfig, discount billing.AmountLineDiscountManaged, taxCodes productcatalog.TaxCodesByID) *stripe.InvoiceItemParams {
	name := getDiscountLineName(line.DetailedLine, discount)
	period := getPeriod(line.DetailedLine)

	addParams := &stripe.InvoiceItemParams{
		Description: lo.ToPtr(name),
		Amount:      lo.ToPtr(-calculator.RoundToAmount(discount.Amount.Add(discount.RoundingAmount))),
		Period:      period,
		Metadata: map[string]string{
			invoiceLineMetadataID:   discount.ID,
			invoiceLineMetadataType: invoiceLineMetadataTypeDiscount,
		},
	}

	return applyTaxSettingsToInvoiceItem(addParams, line.TaxConfig, taxCodes)
}

func getDiscountStripeAddInvoiceItemParams(calculator StripeCalculator, line billing.DetailedLineWithResolvedTaxConfig, discount billing.AmountLineDiscountManaged, stripeCustomerID string, taxCodes productcatalog.TaxCodesByID) *stripe.InvoiceItemParams {
	params := getDiscountStripeInvoiceItemParams(calculator, line, discount, taxCodes)
	// Customer is required for adds
	params.Customer = stripe.String(stripeCustomerID)

	return params
}

// getCreditStripeUpdateInvoiceItemParams returns the Stripe line item for an applied credit.
func getCreditStripeUpdateInvoiceItemParams(
	calculator StripeCalculator,
	line billing.DetailedLineWithResolvedTaxConfig,
	credit billing.CreditApplied,
	stripeLine *stripe.InvoiceLineItem,
	taxCodes productcatalog.TaxCodesByID,
) *stripeclient.StripeInvoiceItemWithID {
	return &stripeclient.StripeInvoiceItemWithID{
		ID:                stripeLine.ID,
		InvoiceItemParams: getCreditStripeInvoiceItemParams(calculator, line, credit, taxCodes),
	}
}

// getCreditStripeInvoiceItemParams returns the Stripe line item for an applied credit.
func getCreditStripeInvoiceItemParams(calculator StripeCalculator, line billing.DetailedLineWithResolvedTaxConfig, credit billing.CreditApplied, taxCodes productcatalog.TaxCodesByID) *stripe.InvoiceItemParams {
	name := getCreditLineName(line.DetailedLine, credit)
	period := getPeriod(line.DetailedLine)

	addParams := &stripe.InvoiceItemParams{
		Description: lo.ToPtr(name),
		Amount:      lo.ToPtr(-calculator.RoundToAmount(credit.Amount)),
		Period:      period,
		Metadata: map[string]string{
			invoiceLineMetadataID:   credit.CreditRealizationID,
			invoiceLineMetadataType: invoiceLineMetadataTypeCredit,
		},
	}

	return applyTaxSettingsToInvoiceItem(addParams, line.TaxConfig, taxCodes)
}

func getCreditStripeAddInvoiceItemParams(calculator StripeCalculator, line billing.DetailedLineWithResolvedTaxConfig, credit billing.CreditApplied, stripeCustomerID string, taxCodes productcatalog.TaxCodesByID) *stripe.InvoiceItemParams {
	params := getCreditStripeInvoiceItemParams(calculator, line, credit, taxCodes)
	// Customer is required for adds
	params.Customer = stripe.String(stripeCustomerID)

	return params
}

// applyTaxSettingsToInvoiceItem sets the tax settings on the Stripe line item from the line's
// resolved tax config. The Stripe tax code comes from the Tax Code the line references by ID,
// not from the stored Stripe config: the mapping lives on the Tax Code and a re-sync must send
// its current value. Tax Codes without a Stripe mapping (e.g. the provider default) leave the
// code unset, so Stripe applies its own default.
func applyTaxSettingsToInvoiceItem(add *stripe.InvoiceItemParams, taxConfig *billing.TaxConfig, taxCodes productcatalog.TaxCodesByID) *stripe.InvoiceItemParams {
	if taxConfig != nil && !lo.IsEmpty(taxConfig) {
		if taxConfig.Behavior != nil {
			add.TaxBehavior = getStripeTaxBehavior(taxConfig.Behavior)
		}

		if tc, ok := taxCodes.Get(lo.FromPtr(taxConfig.TaxCodeID)); ok {
			if mapping, ok := tc.GetAppMapping(app.AppTypeStripe); ok {
				add.TaxCode = stripe.String(mapping.TaxCode)
			}
		}
	}

	return add
}

// getStripeUpdateInvoiceItemParams returns the Stripe update line params
func getStripeUpdateInvoiceItemParams(
	calculator StripeCalculator,
	line billing.DetailedLineWithResolvedTaxConfig,
	stripeLine *stripe.InvoiceLineItem,
	taxCodes productcatalog.TaxCodesByID,
) *stripeclient.StripeInvoiceItemWithID {
	return &stripeclient.StripeInvoiceItemWithID{
		ID:                stripeLine.ID,
		InvoiceItemParams: getStripeInvoiceItemParams(line, calculator, taxCodes),
	}
}

// getStripeAddLinesLineParams returns the Stripe line item
func getStripeInvoiceItemParams(line billing.DetailedLineWithResolvedTaxConfig, calculator StripeCalculator, taxCodes productcatalog.TaxCodesByID) *stripe.InvoiceItemParams {
	description := getLineName(line.DetailedLine)
	period := getPeriod(line.DetailedLine)
	amount := line.Totals.Amount

	// Handle usage based commitments like minimum spend
	if amount.IsZero() {
		// ChargesTotal is the amount of value of the line that are due to additional charges.
		// If the line is a commitment we use the total charges.
		amount = line.Totals.ChargesTotal
	}

	// If the line has a quantity we add the quantity and per unit amount to the description
	if line.Quantity.GreaterThan(alpacadecimal.NewFromInt(1)) || line.Quantity.IsNegative() {
		description = fmt.Sprintf(
			"%s (%s x %s)",
			description,
			calculator.FormatQuantity(line.Quantity),
			calculator.FormatAmount(line.PerUnitAmount),
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

	return applyTaxSettingsToInvoiceItem(addParams, line.TaxConfig, taxCodes)
}

// getStripeAddInvoiceItemParams returns the Stripe line item
func getStripeAddInvoiceItemParams(line billing.DetailedLineWithResolvedTaxConfig, calculator StripeCalculator, stripeCustomerID string, taxCodes productcatalog.TaxCodesByID) *stripe.InvoiceItemParams {
	params := getStripeInvoiceItemParams(line, calculator, taxCodes)
	params.Customer = stripe.String(stripeCustomerID)

	return params
}

// getPeriod returns the period
func getPeriod(line billing.DetailedLine) *stripe.InvoiceItemPeriodParams {
	return &stripe.InvoiceItemPeriodParams{
		Start: lo.ToPtr(line.ServicePeriod.From.Unix()),
		End:   lo.ToPtr(line.ServicePeriod.To.Unix()),
	}
}

// getDiscountLineName returns the line name
func getDiscountLineName(line billing.DetailedLine, discount billing.AmountLineDiscountManaged) string {
	name := line.Name
	if discount.Description != nil {
		name = fmt.Sprintf("%s (%s)", name, *discount.Description)
	}

	return name
}

// getCreditLineName returns the applied-credit line name.
func getCreditLineName(line billing.DetailedLine, credit billing.CreditApplied) string {
	name := fmt.Sprintf("credits applied for %s", getLineName(line))
	if credit.Description != "" {
		name = fmt.Sprintf("%s (%s)", name, credit.Description)
	}

	return name
}

// getLineName returns the line name
func getLineName(line billing.DetailedLine) string {
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
	result *billing.UpsertStandardInvoiceResult,
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

		if lineType == invoiceLineMetadataTypeCredit {
			continue
		}

		// Add line external ID
		result.AddLineExternalID(id, stripeLine.LineID)
	}

	return nil
}
