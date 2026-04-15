package lineengine

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

var _ billing.LineEngine = (*Engine)(nil)

type Config struct {
	FlatFeeService flatfee.Service
	RatingService  rating.Service
}

func (c Config) Validate() error {
	if c.FlatFeeService == nil {
		return fmt.Errorf("flat fee service is required")
	}

	if c.RatingService == nil {
		return fmt.Errorf("rating service is required")
	}

	return nil
}

type Engine struct {
	flatFeeService flatfee.Service
	ratingService  rating.Service
}

// TODO[later]: Most probably we should just implement the LineEngine interface inside the service of flatfee, but let's see how the interface evolves.
func New(config Config) (*Engine, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &Engine{
		flatFeeService: config.FlatFeeService,
		ratingService:  config.RatingService,
	}, nil
}

func (e *Engine) GetLineEngineType() billing.LineEngineType {
	return billing.LineEngineTypeChargeFlatFee
}

func (e *Engine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	// Billing enforces that flat fees are never progressively billed, so there is no
	// engine-side partial-period filtering to do here.
	return true, nil
}

func (e *Engine) SplitGatheringLine(ctx context.Context, input billing.SplitGatheringLineInput) (billing.SplitGatheringLineResult, error) {
	return billing.SplitGatheringLineResult{}, fmt.Errorf("flat fee line is not progressively billed")
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

func (e *Engine) OnStandardInvoiceCreated(ctx context.Context, input billing.OnStandardInvoiceCreatedInput) (billing.StandardLines, error) {
	if err := input.Validate(); err != nil {
		return nil, fmt.Errorf("validating input: %w", err)
	}

	for _, stdLine := range input.Lines {
		chargeID := stdLine.ChargeID
		if chargeID == nil {
			return nil, fmt.Errorf("flat fee standard line[%s]: charge id is required", stdLine.ID)
		}

		charge, err := e.flatFeeService.GetByID(ctx, flatfee.GetByIDInput{
			ChargeID: meta.ChargeID{
				Namespace: stdLine.Namespace,
				ID:        *chargeID,
			},
			Expands: meta.Expands{
				meta.ExpandRealizations,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("getting flat fee charge for line[%s]: %w", stdLine.ID, err)
		}

		realizations, err := e.flatFeeService.PostLineAssignedToInvoice(ctx, charge, *stdLine)
		if err != nil {
			return nil, fmt.Errorf("allocating credits for line[%s]: %w", stdLine.ID, err)
		}

		if len(realizations) > 0 {
			stdLine.CreditsApplied = convertCreditRealizations(realizations)
		}
	}

	return e.CalculateLines(billing.CalculateLinesInput(input))
}

func (e *Engine) OnCollectionCompleted(_ context.Context, input billing.OnCollectionCompletedInput) (billing.StandardLines, error) {
	return input.Lines, nil
}

func (e *Engine) OnInvoiceIssued(_ context.Context, _ billing.OnInvoiceIssuedInput) error {
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
		generatedDetailedLines, err := e.ratingService.GenerateDetailedLines(stdLine)
		if err != nil {
			return nil, fmt.Errorf("generating detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(stdLine, generatedDetailedLines); err != nil {
			return nil, fmt.Errorf("merging generated detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
		}
	}

	return input.Lines, nil
}
