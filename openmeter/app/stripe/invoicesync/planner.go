package invoicesync

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stripe/stripe-go/v80"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
)

// Line metadata keys used by both the sync plan path and the legacy inline sync path.
// These must stay in sync with the constants in entity/app/invoice.go.
const (
	LineMetadataID           = "om_line_id"
	LineMetadataType         = "om_line_type"
	LineMetadataTypeLine     = "line"
	LineMetadataTypeDiscount = "discount"

	// defaultDaysUntilDue is the fallback when DueAfter cannot be converted to days.
	defaultDaysUntilDue = 30
)

// PlanGeneratorInput contains the data needed to generate a sync plan.
type PlanGeneratorInput struct {
	Invoice              billing.StandardInvoice
	StripeCustomerID     string
	StripeDefaultPayment string
	AppID                string
	Currency             string
	// ExistingStripeLines are the current line items on the Stripe invoice (only for update plans).
	ExistingStripeLines []*stripe.InvoiceLineItem
}

// GenerateDraftSyncPlan generates an ordered list of operations for syncing a draft invoice to Stripe.
func GenerateDraftSyncPlan(input PlanGeneratorInput) (sessionID string, ops []SyncOperation, err error) {
	sessionID = ulid.Make().String()

	if input.Invoice.ExternalIDs.Invoicing == "" {
		ops, err = generateCreatePlan(input, sessionID)
	} else {
		ops, err = generateUpdatePlan(input, sessionID)
	}

	return sessionID, ops, err
}

// GenerateIssuingSyncPlan generates operations for finalizing an invoice in Stripe.
func GenerateIssuingSyncPlan(input PlanGeneratorInput) (sessionID string, ops []SyncOperation, err error) {
	sessionID = ulid.Make().String()
	seq := 0

	// First, do a final upsert to make sure Stripe is up to date
	if input.Invoice.ExternalIDs.Invoicing == "" {
		// Should not happen in issuing — invoice must have been created during draft sync
		return "", nil, fmt.Errorf("invoice has no Stripe external ID for issuing sync")
	}

	updateOps, err := generateUpdatePlan(input, sessionID)
	if err != nil {
		return "", nil, fmt.Errorf("generating update plan for issuing: %w", err)
	}
	ops = append(ops, updateOps...)
	seq = len(ops)

	// Then finalize
	finalizePayload := InvoiceFinalizePayload{
		StripeInvoiceID: input.Invoice.ExternalIDs.Invoicing,
		AutoAdvance:     true,
		TaxEnforced:     input.Invoice.Workflow.Config.Tax.Enforced,
	}
	payloadBytes, err := json.Marshal(finalizePayload)
	if err != nil {
		return "", nil, fmt.Errorf("marshaling finalize payload: %w", err)
	}

	ops = append(ops, SyncOperation{
		Sequence:       seq,
		Type:           OpTypeInvoiceFinalize,
		Payload:        payloadBytes,
		IdempotencyKey: GenerateIdempotencyKey(input.Invoice.ID, sessionID, seq, OpTypeInvoiceFinalize),
		Status:         OpStatusPending,
	})

	return sessionID, ops, nil
}

// GenerateDeleteSyncPlan generates operations for deleting an invoice from Stripe.
func GenerateDeleteSyncPlan(input PlanGeneratorInput) (sessionID string, ops []SyncOperation, err error) {
	sessionID = ulid.Make().String()

	if input.Invoice.ExternalIDs.Invoicing == "" {
		// Nothing to delete on Stripe side
		return sessionID, nil, nil
	}

	payload := InvoiceDeletePayload{
		StripeInvoiceID: input.Invoice.ExternalIDs.Invoicing,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", nil, fmt.Errorf("marshaling delete payload: %w", err)
	}

	ops = append(ops, SyncOperation{
		Sequence:       0,
		Type:           OpTypeInvoiceDelete,
		Payload:        payloadBytes,
		IdempotencyKey: GenerateIdempotencyKey(input.Invoice.ID, sessionID, 0, OpTypeInvoiceDelete),
		Status:         OpStatusPending,
	})

	return sessionID, ops, nil
}

func generateCreatePlan(input PlanGeneratorInput, sessionID string) ([]SyncOperation, error) {
	invoice := input.Invoice
	seq := 0

	var ops []SyncOperation

	// Op 1: Create invoice
	createPayload := InvoiceCreatePayload{
		AppID:                        input.AppID,
		Namespace:                    invoice.Namespace,
		CustomerID:                   invoice.Customer.CustomerID,
		InvoiceID:                    invoice.ID,
		AutomaticTaxEnabled:          invoice.Workflow.Config.Tax.Enabled,
		CollectionMethod:             invoice.Workflow.Config.Payment.CollectionMethod,
		Currency:                     input.Currency,
		StripeCustomerID:             input.StripeCustomerID,
		StripeDefaultPaymentMethodID: input.StripeDefaultPayment,
	}

	// Set days until due for send_invoice collection method
	if invoice.Workflow.Config.Payment.CollectionMethod == billing.CollectionMethodSendInvoice {
		daysUntilDue, _, ok := invoice.Workflow.Config.Invoicing.DueAfter.DaysDecimal().Int64(0)
		if !ok {
			return nil, fmt.Errorf("failed to get days until due")
		}

		if daysUntilDue == 0 {
			futureDueAt, addOK := invoice.Workflow.Config.Invoicing.DueAfter.AddTo(time.Now())
			if !addOK {
				futureDueAt = time.Now().Add(defaultDaysUntilDue * 24 * time.Hour)
			}
			duration := time.Until(futureDueAt)
			daysUntilDue = int64(math.Round(duration.Hours() / 24))
			if daysUntilDue < 0 {
				daysUntilDue = 0
			}
		}

		createPayload.DaysUntilDue = lo.ToPtr(daysUntilDue)
	}

	payloadBytes, err := json.Marshal(createPayload)
	if err != nil {
		return nil, fmt.Errorf("marshaling create payload: %w", err)
	}

	ops = append(ops, SyncOperation{
		Sequence:       seq,
		Type:           OpTypeInvoiceCreate,
		Payload:        payloadBytes,
		IdempotencyKey: GenerateIdempotencyKey(invoice.ID, sessionID, seq, OpTypeInvoiceCreate),
		Status:         OpStatusPending,
	})
	seq++

	// Op 2: Add line items
	lineParams, err := buildAddLineParams(invoice, input.StripeCustomerID, input.Currency)
	if err != nil {
		return nil, fmt.Errorf("building add line params: %w", err)
	}
	if len(lineParams) > 0 {
		linePayload := LineItemAddPayload{
			// StripeInvoiceID will be resolved from the InvoiceCreate response at execution time
			StripeInvoiceID: "", // resolved at execution time
			Lines:           lineParams,
		}
		payloadBytes, err := json.Marshal(linePayload)
		if err != nil {
			return nil, fmt.Errorf("marshaling line add payload: %w", err)
		}

		ops = append(ops, SyncOperation{
			Sequence:       seq,
			Type:           OpTypeLineItemAdd,
			Payload:        payloadBytes,
			IdempotencyKey: GenerateIdempotencyKey(invoice.ID, sessionID, seq, OpTypeLineItemAdd),
			Status:         OpStatusPending,
		})
	}

	return ops, nil
}

func generateUpdatePlan(input PlanGeneratorInput, sessionID string) ([]SyncOperation, error) {
	invoice := input.Invoice
	seq := 0
	var ops []SyncOperation

	stripeInvoiceID := invoice.ExternalIDs.Invoicing

	// Op 1: Update invoice metadata/settings
	updatePayload := InvoiceUpdatePayload{
		StripeInvoiceID:     stripeInvoiceID,
		AutomaticTaxEnabled: invoice.Workflow.Config.Tax.Enabled,
	}
	payloadBytes, err := json.Marshal(updatePayload)
	if err != nil {
		return nil, fmt.Errorf("marshaling update payload: %w", err)
	}
	ops = append(ops, SyncOperation{
		Sequence:       seq,
		Type:           OpTypeInvoiceUpdate,
		Payload:        payloadBytes,
		IdempotencyKey: GenerateIdempotencyKey(invoice.ID, sessionID, seq, OpTypeInvoiceUpdate),
		Status:         OpStatusPending,
	})
	seq++

	// Build existing Stripe line index for diffing
	stripeLinesByID := make(map[string]*stripe.InvoiceLineItem)
	stripeLinesToRemove := make(map[string]bool)
	for _, sl := range input.ExistingStripeLines {
		stripeLinesToRemove[sl.ID] = true
		stripeLinesByID[sl.ID] = sl
		if sl.InvoiceItem != nil {
			stripeLinesByID[sl.InvoiceItem.ID] = sl
		}
	}

	var (
		addLines    []LineItemParams
		updateLines []LineItemUpdateParams
		removeIDs   []string
	)

	// Diff lines
	for _, line := range invoice.GetLeafLinesWithConsolidatedTaxBehavior() {
		amountDiscountsById, err := line.AmountDiscounts.GetByID()
		if err != nil {
			return nil, fmt.Errorf("getting amount discounts by ID: %w", err)
		}

		// Process discount lines
		for _, discount := range amountDiscountsById {
			if discount.ExternalIDs.Invoicing != "" {
				stripeLine, ok := stripeLinesByID[discount.ExternalIDs.Invoicing]
				if ok {
					delete(stripeLinesToRemove, stripeLine.ID)
					p, err := toDiscountUpdateParams(line, discount, stripeLine, input.Currency)
					if err != nil {
						return nil, fmt.Errorf("building discount update params: %w", err)
					}
					updateLines = append(updateLines, p)
				} else {
					// Stale or unknown Stripe ID: re-add so the invoice stays consistent (do not touch stripeLinesToRemove).
					p, err := toDiscountAddParams(line, discount, input.StripeCustomerID, input.Currency)
					if err != nil {
						return nil, fmt.Errorf("building discount add params: %w", err)
					}
					addLines = append(addLines, p)
				}
			} else {
				p, err := toDiscountAddParams(line, discount, input.StripeCustomerID, input.Currency)
				if err != nil {
					return nil, fmt.Errorf("building discount add params: %w", err)
				}
				addLines = append(addLines, p)
			}
		}

		// Process regular lines
		if line.ExternalIDs.Invoicing != "" {
			stripeLine, ok := stripeLinesByID[line.ExternalIDs.Invoicing]
			if ok {
				delete(stripeLinesToRemove, stripeLine.ID)
				p, err := toLineUpdateParams(line, stripeLine, input.Currency)
				if err != nil {
					return nil, fmt.Errorf("building line update params: %w", err)
				}
				updateLines = append(updateLines, p)
			} else {
				// Stale or unknown Stripe ID: re-add so the invoice stays consistent (do not touch stripeLinesToRemove).
				p, err := toLineAddParams(line, input.StripeCustomerID, input.Currency)
				if err != nil {
					return nil, fmt.Errorf("building line add params: %w", err)
				}
				addLines = append(addLines, p)
			}
		} else {
			p, err := toLineAddParams(line, input.StripeCustomerID, input.Currency)
			if err != nil {
				return nil, fmt.Errorf("building line add params: %w", err)
			}
			addLines = append(addLines, p)
		}
	}

	// Collect lines to remove; sort for deterministic JSON payloads across runs.
	for id := range stripeLinesToRemove {
		removeIDs = append(removeIDs, id)
	}
	sort.Strings(removeIDs)

	// Op 2: Remove old line items (before add to avoid hitting limits)
	if len(removeIDs) > 0 {
		payload := LineItemRemovePayload{
			StripeInvoiceID: stripeInvoiceID,
			LineIDs:         removeIDs,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling line remove payload: %w", err)
		}
		ops = append(ops, SyncOperation{
			Sequence:       seq,
			Type:           OpTypeLineItemRemove,
			Payload:        payloadBytes,
			IdempotencyKey: GenerateIdempotencyKey(invoice.ID, sessionID, seq, OpTypeLineItemRemove),
			Status:         OpStatusPending,
		})
		seq++
	}

	// Op 3: Update existing line items
	if len(updateLines) > 0 {
		payload := LineItemUpdatePayload{
			StripeInvoiceID: stripeInvoiceID,
			Lines:           updateLines,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling line update payload: %w", err)
		}
		ops = append(ops, SyncOperation{
			Sequence:       seq,
			Type:           OpTypeLineItemUpdate,
			Payload:        payloadBytes,
			IdempotencyKey: GenerateIdempotencyKey(invoice.ID, sessionID, seq, OpTypeLineItemUpdate),
			Status:         OpStatusPending,
		})
		seq++
	}

	// Op 4: Add new line items
	if len(addLines) > 0 {
		// Sort for deterministic order
		sort.Slice(addLines, func(i, j int) bool {
			return addLines[i].Description < addLines[j].Description
		})

		payload := LineItemAddPayload{
			StripeInvoiceID: stripeInvoiceID,
			Lines:           addLines,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshaling line add payload: %w", err)
		}
		ops = append(ops, SyncOperation{
			Sequence:       seq,
			Type:           OpTypeLineItemAdd,
			Payload:        payloadBytes,
			IdempotencyKey: GenerateIdempotencyKey(invoice.ID, sessionID, seq, OpTypeLineItemAdd),
			Status:         OpStatusPending,
		})
	}

	return ops, nil
}

// buildAddLineParams builds LineItemParams for all leaf lines (used for create flow).
func buildAddLineParams(invoice billing.StandardInvoice, stripeCustomerID, currency string) ([]LineItemParams, error) {
	var params []LineItemParams

	leafLines := invoice.GetLeafLinesWithConsolidatedTaxBehavior()

	for _, line := range leafLines {
		for _, discount := range line.AmountDiscounts {
			p, err := toDiscountAddParams(line, discount, stripeCustomerID, currency)
			if err != nil {
				return nil, fmt.Errorf("building discount add params: %w", err)
			}
			params = append(params, p)
		}
		p, err := toLineAddParams(line, stripeCustomerID, currency)
		if err != nil {
			return nil, fmt.Errorf("building line add params: %w", err)
		}
		params = append(params, p)
	}

	// Sort for deterministic order
	sort.Slice(params, func(i, j int) bool {
		return params[i].Description < params[j].Description
	})

	return params, nil
}

func toLineAddParams(line billing.DetailedLine, stripeCustomerID, currency string) (LineItemParams, error) {
	description := getLineName(line)
	amount := getLineAmount(line)

	if line.Quantity.GreaterThan(alpacadecimal.NewFromInt(1)) {
		formattedAmount, err := FormatAmount(line.PerUnitAmount, currency)
		if err != nil {
			return LineItemParams{}, err
		}
		description = fmt.Sprintf(
			"%s (%s x %s)",
			description,
			FormatQuantity(line.Quantity, currency),
			formattedAmount,
		)
	}

	roundedAmount, err := RoundToAmount(amount, currency)
	if err != nil {
		return LineItemParams{}, err
	}

	p := LineItemParams{
		Description: description,
		Amount:      roundedAmount,
		Currency:    currency,
		CustomerID:  stripeCustomerID,
		PeriodStart: line.ServicePeriod.Start.Unix(),
		PeriodEnd:   line.ServicePeriod.End.Unix(),
		Metadata: map[string]string{
			LineMetadataID:   line.ID,
			LineMetadataType: LineMetadataTypeLine,
		},
	}

	applyTax(&p, line)
	return p, nil
}

func toDiscountAddParams(line billing.DetailedLine, discount billing.AmountLineDiscountManaged, stripeCustomerID, currency string) (LineItemParams, error) {
	name := getDiscountLineName(line, discount)

	roundedAmount, err := RoundToAmount(discount.Amount.Add(discount.RoundingAmount), currency)
	if err != nil {
		return LineItemParams{}, err
	}

	p := LineItemParams{
		Description: name,
		Amount:      -roundedAmount,
		Currency:    currency,
		CustomerID:  stripeCustomerID,
		PeriodStart: line.ServicePeriod.Start.Unix(),
		PeriodEnd:   line.ServicePeriod.End.Unix(),
		Metadata: map[string]string{
			LineMetadataID:   discount.ID,
			LineMetadataType: LineMetadataTypeDiscount,
		},
	}

	applyTax(&p, line)
	return p, nil
}

func toLineUpdateParams(line billing.DetailedLine, stripeLine *stripe.InvoiceLineItem, currency string) (LineItemUpdateParams, error) {
	description := getLineName(line)
	amount := getLineAmount(line)

	if line.Quantity.GreaterThan(alpacadecimal.NewFromInt(1)) {
		formattedAmount, err := FormatAmount(line.PerUnitAmount, currency)
		if err != nil {
			return LineItemUpdateParams{}, err
		}
		description = fmt.Sprintf(
			"%s (%s x %s)",
			description,
			FormatQuantity(line.Quantity, currency),
			formattedAmount,
		)
	}

	roundedAmount, err := RoundToAmount(amount, currency)
	if err != nil {
		return LineItemUpdateParams{}, err
	}

	p := LineItemUpdateParams{
		ID:          stripeLine.ID,
		Description: description,
		Amount:      roundedAmount,
		Currency:    currency,
		PeriodStart: line.ServicePeriod.Start.Unix(),
		PeriodEnd:   line.ServicePeriod.End.Unix(),
		Metadata: map[string]string{
			LineMetadataID:   line.ID,
			LineMetadataType: LineMetadataTypeLine,
		},
	}

	applyTax(&p, line)
	return p, nil
}

func toDiscountUpdateParams(line billing.DetailedLine, discount billing.AmountLineDiscountManaged, stripeLine *stripe.InvoiceLineItem, currency string) (LineItemUpdateParams, error) {
	name := getDiscountLineName(line, discount)

	roundedAmount, err := RoundToAmount(discount.Amount.Add(discount.RoundingAmount), currency)
	if err != nil {
		return LineItemUpdateParams{}, err
	}

	p := LineItemUpdateParams{
		ID:          stripeLine.ID,
		Description: name,
		Amount:      -roundedAmount,
		Currency:    currency,
		PeriodStart: line.ServicePeriod.Start.Unix(),
		PeriodEnd:   line.ServicePeriod.End.Unix(),
		Metadata: map[string]string{
			LineMetadataID:   discount.ID,
			LineMetadataType: LineMetadataTypeDiscount,
		},
	}

	applyTax(&p, line)
	return p, nil
}

type taxSettable interface {
	setTax(behavior *string, code *string)
}

func (p *LineItemParams) setTax(behavior *string, code *string) {
	p.TaxBehavior = behavior
	p.TaxCode = code
}

func (p *LineItemUpdateParams) setTax(behavior *string, code *string) {
	p.TaxBehavior = behavior
	p.TaxCode = code
}

func applyTax(p taxSettable, line billing.DetailedLine) {
	if line.TaxConfig != nil && !lo.IsEmpty(line.TaxConfig) {
		var behavior *string
		var code *string
		if line.TaxConfig.Behavior != nil {
			behavior = getStripeTaxBehavior(line.TaxConfig.Behavior)
		}
		if line.TaxConfig.Stripe != nil {
			code = lo.ToPtr(line.TaxConfig.Stripe.Code)
		}
		p.setTax(behavior, code)
	}
}

func getLineName(line billing.DetailedLine) string {
	name := line.Name
	if line.Description != nil {
		name = fmt.Sprintf("%s (%s)", name, *line.Description)
	}
	return name
}

func getDiscountLineName(line billing.DetailedLine, discount billing.AmountLineDiscountManaged) string {
	name := line.Name
	if discount.Description != nil {
		name = fmt.Sprintf("%s (%s)", name, *discount.Description)
	}
	return name
}

// getLineAmount returns the line's total amount for Stripe.
// Totals.Amount is zero for lines that have only charges (no base price),
// in which case we fall back to ChargesTotal.
func getLineAmount(line billing.DetailedLine) alpacadecimal.Decimal {
	amount := line.Totals.Amount
	if amount.IsZero() {
		amount = line.Totals.ChargesTotal
	}
	return amount
}

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
