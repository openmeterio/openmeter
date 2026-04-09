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

	gatheringLinesByID := make(map[string]billing.GatheringLine, len(input.GatheringLines))
	for _, gatheringLine := range input.GatheringLines {
		gatheringLinesByID[gatheringLine.ID] = gatheringLine
	}

	for _, stdLine := range stdLines {
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

		sourceGatheringLine, ok := gatheringLinesByID[stdLine.ID]
		if !ok {
			return nil, fmt.Errorf("flat fee standard line[%s]: source gathering line not found", stdLine.ID)
		}

		realizations, err := e.flatFeeService.PostLineAssignedToInvoice(ctx, charge, sourceGatheringLine)
		if err != nil {
			return nil, fmt.Errorf("allocating credits for line[%s]: %w", stdLine.ID, err)
		}

		if len(realizations) > 0 {
			stdLine.CreditsApplied = convertCreditRealizations(realizations)
		}

		generatedDetailedLines, err := e.ratingService.GenerateDetailedLines(stdLine)
		if err != nil {
			return nil, fmt.Errorf("regenerating detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(stdLine, generatedDetailedLines); err != nil {
			return nil, fmt.Errorf("merging regenerated detailed lines for line[%s]: %w", stdLine.ID, err)
		}

		if err := stdLine.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", stdLine.ID, err)
		}
	}

	return stdLines, nil
}
