package lineengine

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var (
	_ billing.LineEngine     = (*Engine)(nil)
	_ billing.LineCalculator = (*Engine)(nil)
)

type Config struct {
	RatingService rating.Service
}

func (c Config) Validate() error {
	if c.RatingService == nil {
		return fmt.Errorf("rating service is required")
	}

	return nil
}

type Engine struct {
	ratingService rating.Service
}

func New(config Config) (*Engine, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Engine{
		ratingService: config.RatingService,
	}, nil
}

func (e *Engine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeChargeCreditPurchase
}

func (e *Engine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	// Billing enforces that credit purchases are never progressively billed, so there is no
	// engine-side partial-period filtering to do here.
	return true, nil
}

func (e *Engine) SplitGatheringLine(_ context.Context, _ billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	return billing.SplitGatheringLineResult{}, fmt.Errorf("credit purchase line is not progressively billed")
}

func (e *Engine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	stdLines, err := slicesx.MapWithErr(input.GatheringLines, func(gatheringLine billing.GatheringLine) (*billing.StandardLine, error) {
		stdLine, err := gatheringLine.AsNewStandardLine(input.Invoice.ID)
		if err != nil {
			return nil, fmt.Errorf("converting gathering line to standard line: %w", err)
		}

		return stdLine, nil
	})
	if err != nil {
		return nil, err
	}

	return e.CalculateLines(billing.CalculateLinesInput{
		Invoice: input.Invoice,
		Lines:   stdLines,
	})
}

func (e *Engine) OnCollectionCompleted(_ context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
	return input.Lines, nil
}

func (e *Engine) OnStandardInvoiceCreated(_ context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	return input.Lines, nil
}

func (e *Engine) OnMutableStandardLinesDeleted(_ context.Context, _ billing.OnMutableStandardLinesDeletedInput) error {
	return nil
}

func (e *Engine) OnUnsupportedCreditNote(_ context.Context, _ billing.OnUnsupportedCreditNoteInput) error {
	return nil
}

func (e *Engine) OnInvoiceIssued(_ context.Context, _ billing.OnInvoiceIssuedInput) error {
	return nil
}

func (e *Engine) OnPaymentAuthorized(_ context.Context, _ billing.OnPaymentAuthorizedInput) error {
	return nil
}

func (e *Engine) OnPaymentSettled(_ context.Context, _ billing.OnPaymentSettledInput) error {
	return nil
}

func (e *Engine) CalculateLines(input billing.CalculateLinesInput) (billing.StandardLines, error) {
	if input.Invoice.ID == "" {
		return nil, fmt.Errorf("invoice id is required")
	}

	if len(input.Lines) == 0 {
		return nil, fmt.Errorf("lines are required")
	}

	for _, stdLine := range input.Lines {
		if stdLine.ChargeID == nil {
			return nil, fmt.Errorf("credit purchase standard line[%s]: charge id is required", stdLine.ID)
		}

		generatedDetailedLines, err := e.ratingService.GenerateDetailedLines(stdLine)
		if err != nil {
			return nil, fmt.Errorf("generating detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(stdLine, generatedDetailedLines); err != nil {
			return nil, fmt.Errorf("merging detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
		}
	}

	return input.Lines, nil
}
