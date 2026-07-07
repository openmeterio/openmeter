package testutils

import (
	"context"

	"github.com/openmeterio/openmeter/openmeter/billing"
)

var _ billing.LineEngine = NoopLineEngine{}

// NoopLineEngine is a test helper intended for embedding in line-engine fakes.
// It intentionally does not implement billing.LineCalculator.
type NoopLineEngine struct {
	EngineType billing.LineEngineType
}

func (e NoopLineEngine) GetLineEngineType() billing.LineEngineType {
	if e.EngineType == "" {
		panic("engine type is required")
	}

	return e.EngineType
}

func (NoopLineEngine) IsLineBillableAsOf(context.Context, billing.IsLineBillableAsOfInput) (bool, error) {
	return true, nil
}

func (NoopLineEngine) SplitGatheringLine(_ context.Context, input billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	return billing.SplitGatheringLineResult{
		PreSplitAtLine: input.Line,
	}, nil
}

func (NoopLineEngine) BuildStandardInvoiceLines(_ context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	return input.GatheringLines.ToStandardLines(input.Invoice.ID)
}

func (NoopLineEngine) BuildStandardLinesForGatheringPreview(_ context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	return input.GatheringLines.ToStandardLines(input.Invoice.ID)
}

func (NoopLineEngine) OnStandardInvoiceCreated(_ context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	return input.Lines, nil
}

func (NoopLineEngine) OnCollectionCompleted(_ context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
	return input.Lines, nil
}

func (NoopLineEngine) ValidateMutableInvoiceLineEditViaAPI(_ context.Context, input billing.OnMutableInvoiceUpdateInput) error {
	return input.Validate()
}

func (NoopLineEngine) OnMutableInvoiceLinesEditedViaAPI(_ context.Context, input billing.OnMutableInvoiceUpdateInput) (billing.OnMutableInvoiceUpdateResult, error) {
	updatedLines := make([]billing.GenericInvoiceLine, 0, len(input.Updated))
	for _, override := range input.Updated {
		line, err := override.ChangesToApply.Apply(override.ExistingLine)
		if err != nil {
			return billing.OnMutableInvoiceUpdateResult{}, err
		}

		updatedLines = append(updatedLines, line)
	}

	return billing.OnMutableInvoiceUpdateResult{
		CreatedLines: input.Created,
		UpdatedLines: updatedLines,
	}, nil
}

func (NoopLineEngine) OnMutableStandardLinesDeletedBySystem(context.Context, billing.OnMutableStandardLinesDeletedInput) error {
	return nil
}

func (NoopLineEngine) OnUnsupportedCreditNote(context.Context, billing.OnUnsupportedCreditNoteInput) error {
	return nil
}

func (NoopLineEngine) OnInvoiceIssued(context.Context, billing.OnInvoiceIssuedInput) error {
	return nil
}

func (NoopLineEngine) OnPaymentAuthorized(context.Context, billing.OnPaymentAuthorizedInput) error {
	return nil
}

func (NoopLineEngine) OnPaymentSettled(context.Context, billing.OnPaymentSettledInput) error {
	return nil
}
