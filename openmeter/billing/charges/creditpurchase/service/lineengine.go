package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/creditpurchase"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	creditpurchasemodels "github.com/openmeterio/openmeter/openmeter/billing/charges/models/creditpurchase"
	"github.com/openmeterio/openmeter/pkg/models"
)

var _ billing.LineEngine = (*LineEngine)(nil)

type LineEngine struct {
	service *service
}

func (e *LineEngine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeChargeCreditPurchase
}

func (e *LineEngine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	// Billing enforces that credit purchases are never progressively billed, so there is no
	// engine-side partial-period filtering to do here.
	return true, nil
}

func (e *LineEngine) SplitGatheringLine(_ context.Context, _ billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	return billing.SplitGatheringLineResult{}, fmt.Errorf("credit purchase line is not progressively billed")
}

func (e *LineEngine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	return e.buildInvoiceCreditPurchaseStandardLines(ctx, input)
}

func (e *LineEngine) BuildStandardLinesForGatheringPreview(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	return e.buildInvoiceCreditPurchaseStandardLines(ctx, input)
}

// buildInvoiceCreditPurchaseStandardLines preserves the purchased credit
// quantity and its resolved cost basis instead of rerating the fiat line total.
func (e *LineEngine) buildInvoiceCreditPurchaseStandardLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	stdLines, err := input.GatheringLines.ToStandardLines(input.Invoice.ID)
	if err != nil {
		return nil, fmt.Errorf("converting gathering lines to standard lines: %w", err)
	}

	chargesByID, err := e.getChargesForStandardLines(ctx, getChargesForStandardLinesInput{
		Invoice: input.Invoice,
		Lines:   stdLines,
		Expands: meta.ExpandNone,
	})
	if err != nil {
		return nil, err
	}

	for idx, stdLine := range stdLines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return nil, fmt.Errorf("credit purchase charge[%s] not found for line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		invoiceSettlement, err := charge.Intent.Settlement.AsInvoiceSettlement()
		if err != nil {
			return nil, fmt.Errorf("getting invoice settlement for credit purchase charge[%s]: %w", charge.ID, err)
		}

		fiatCurrency, err := invoiceSettlement.Currency.AsFiatCurrency()
		if err != nil {
			return nil, fmt.Errorf("getting fiat currency for credit purchase charge[%s]: %w", charge.ID, err)
		}

		fiatAmount := fiatCurrency.RoundToPrecision(charge.Intent.CreditAmount.Mul(invoiceSettlement.CostBasis))
		stdLineWithDetails, err := creditpurchasemodels.WithDetailedLines(creditpurchasemodels.WithDetailedLinesInput{
			Line:              stdLine,
			Name:              stdLine.Name,
			CreditCurrency:    charge.Intent.Currency,
			CreditAmount:      charge.Intent.CreditAmount,
			ResolvedCostBasis: invoiceSettlement.CostBasis,
			FiatCurrency:      fiatCurrency,
			FiatAmount:        fiatAmount,
		})
		if err != nil {
			return nil, fmt.Errorf("populating credit purchase standard line[%s]: %w", stdLine.ID, err)
		}

		if err := stdLineWithDetails.Validate(); err != nil {
			return nil, fmt.Errorf("validating credit purchase standard line[%s]: %w", stdLine.ID, err)
		}

		stdLines[idx] = stdLineWithDetails
	}

	return stdLines, nil
}

type getChargesForStandardLinesInput struct {
	Invoice billing.StandardInvoice
	Lines   billing.StandardLines
	Expands meta.Expands
}

var _ models.Validator = getChargesForStandardLinesInput{}

func (i getChargesForStandardLinesInput) Validate() error {
	var errs []error

	if i.Invoice.ID == "" {
		errs = append(errs, errors.New("invoice ID is required"))
	}

	if i.Invoice.Namespace == "" {
		errs = append(errs, errors.New("invoice namespace is required"))
	}

	if len(i.Lines) == 0 {
		errs = append(errs, errors.New("standard lines are required"))
	}

	for idx, stdLine := range i.Lines {
		if stdLine == nil {
			errs = append(errs, fmt.Errorf("standard line[%d] is required", idx))
			continue
		}

		if stdLine.ChargeID == nil || *stdLine.ChargeID == "" {
			errs = append(errs, fmt.Errorf("credit purchase standard line[%s]: charge ID is required", stdLine.ID))
		}

		if stdLine.Namespace != i.Invoice.Namespace {
			errs = append(errs, fmt.Errorf(
				"credit purchase standard line[%s]: namespace %s does not match invoice namespace %s",
				stdLine.ID,
				stdLine.Namespace,
				i.Invoice.Namespace,
			))
		}

		if stdLine.InvoiceID != i.Invoice.ID {
			errs = append(errs, fmt.Errorf(
				"credit purchase standard line[%s]: invoice ID %s does not match invoice ID %s",
				stdLine.ID,
				stdLine.InvoiceID,
				i.Invoice.ID,
			))
		}
	}

	if err := i.Expands.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("expands: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

func (e *LineEngine) getChargesForStandardLines(ctx context.Context, input getChargesForStandardLinesInput) (map[string]creditpurchase.Charge, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	chargeIDs := make([]string, 0, len(input.Lines))
	seenChargeIDs := make(map[string]struct{}, len(input.Lines))

	for _, stdLine := range input.Lines {
		if _, ok := seenChargeIDs[*stdLine.ChargeID]; ok {
			continue
		}

		seenChargeIDs[*stdLine.ChargeID] = struct{}{}
		chargeIDs = append(chargeIDs, *stdLine.ChargeID)
	}

	charges, err := e.service.GetByIDs(ctx, creditpurchase.GetByIDsInput{
		Namespace: input.Invoice.Namespace,
		IDs:       chargeIDs,
		Expands:   input.Expands,
	})
	if err != nil {
		return nil, fmt.Errorf("getting credit purchase charges: %w", err)
	}

	return lo.KeyBy(charges, func(charge creditpurchase.Charge) string {
		return charge.ID
	}), nil
}

func (e *LineEngine) OnCollectionCompleted(_ context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
	return input.Lines, nil
}

func (e *LineEngine) OnStandardInvoiceCreated(ctx context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	if err := e.fireInvoiceLifecycleTriggerForLines(ctx, meta.TriggerInvoiceCreated, input); err != nil {
		return nil, err
	}

	return input.Lines, nil
}

func (e *LineEngine) ValidateMutableInvoiceLineEditViaAPI(_ context.Context, _ billing.OnMutableInvoiceUpdateInput) error {
	return billing.ErrCannotUpdateChargeManagedLine
}

func (e *LineEngine) OnMutableInvoiceLinesEditedViaAPI(_ context.Context, _ billing.OnMutableInvoiceUpdateInput) (billing.OnMutableInvoiceUpdateResult, error) {
	return billing.OnMutableInvoiceUpdateResult{}, billing.ErrCannotUpdateChargeManagedLine
}

func (e *LineEngine) OnMutableStandardLinesDeletedBySystem(_ context.Context, _ billing.OnMutableStandardLinesDeletedInput) error {
	return nil
}

func (e *LineEngine) OnUnsupportedCreditNote(_ context.Context, _ billing.OnUnsupportedCreditNoteInput) error {
	return nil
}

func (e *LineEngine) OnInvoiceIssued(_ context.Context, _ billing.OnInvoiceIssuedInput) error {
	return nil
}

func (e *LineEngine) OnPaymentAuthorized(ctx context.Context, input billing.OnPaymentAuthorizedInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	return e.fireInvoiceLifecycleTriggerForLines(ctx, billing.TriggerAuthorized, input)
}

func (e *LineEngine) OnPaymentSettled(ctx context.Context, input billing.OnPaymentSettledInput) error {
	if err := input.Validate(); err != nil {
		return fmt.Errorf("validating input: %w", err)
	}

	return e.fireInvoiceLifecycleTriggerForLines(ctx, billing.TriggerPaid, input)
}

func (e *LineEngine) fireInvoiceLifecycleTriggerForLines(ctx context.Context, trigger meta.Trigger, input billing.StandardLineEventInput) error {
	chargesByID, err := e.getChargesForStandardLines(ctx, getChargesForStandardLinesInput{
		Invoice: input.Invoice,
		Lines:   input.Lines,
		Expands: meta.Expands{meta.ExpandRealizations},
	})
	if err != nil {
		return err
	}

	for _, stdLine := range input.Lines {
		charge, ok := chargesByID[*stdLine.ChargeID]
		if !ok {
			return fmt.Errorf("credit purchase charge[%s] not found for line[%s]", *stdLine.ChargeID, stdLine.ID)
		}

		updatedCharge, err := e.service.handleInvoiceLifecycleTrigger(ctx, HandleInvoiceLifecycleTriggerInput{
			Charge:  charge,
			Trigger: trigger,
			LineWithHeader: billing.StandardLineWithInvoiceHeader{
				Line:    stdLine,
				Invoice: input.Invoice,
			},
		})
		if err != nil {
			return fmt.Errorf("triggering %s for credit purchase charge[%s]: %w", trigger, charge.ID, err)
		}

		// Multiple lines can reference the same charge. Keep its post-transition state
		// so subsequent triggers do not evaluate stale lifecycle data.
		chargesByID[updatedCharge.ID] = updatedCharge
	}

	return nil
}
