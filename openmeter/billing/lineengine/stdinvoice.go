package lineengine

import (
	"context"
	"fmt"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/service/invoicecalc"
	"github.com/openmeterio/openmeter/pkg/slicesx"
)

type QuantitySnapshotter interface {
	SnapshotLineQuantities(ctx context.Context, invoice billing.StandardInvoice, lines billing.StandardLines) error
}

func (e *Engine) BuildStandardInvoiceLines(ctx context.Context, input billing.BuildStandardInvoiceLinesInput) (billing.StandardLines, error) {
	if input.Invoice.ID == "" {
		return nil, fmt.Errorf("invoice id is required")
	}

	if len(input.GatheringLines) == 0 {
		return nil, fmt.Errorf("gathering lines are required")
	}

	stdLines, err := slicesx.MapWithErr(input.GatheringLines, func(gatheringLine billing.GatheringLine) (*billing.StandardLine, error) {
		newStandardLine, err := gatheringLine.AsNewStandardLine(input.Invoice.ID)
		if err != nil {
			return nil, fmt.Errorf("converting gathering line to standard line: %w", err)
		}

		return newStandardLine, nil
	})
	if err != nil {
		return nil, err
	}

	if err := e.ResolveSplitLineGroupHeaders(ctx, input.Invoice.Namespace, stdLines); err != nil {
		return nil, fmt.Errorf("resolving split line group headers: %w", err)
	}

	if err := e.quantitySnapshotter.SnapshotLineQuantities(ctx, input.Invoice, stdLines); err != nil {
		return nil, fmt.Errorf("snapshotting line quantities: %w", err)
	}

	for _, line := range stdLines {
		generatedDetailedLines, err := e.ratingService.GenerateDetailedLines(line)
		if err != nil {
			return nil, fmt.Errorf("generating detailed lines for line[%s]: %w", line.ID, err)
		}

		if err := invoicecalc.MergeGeneratedDetailedLines(line, generatedDetailedLines); err != nil {
			return nil, fmt.Errorf("merging generated detailed lines for line[%s]: %w", line.ID, err)
		}

		if err := line.Validate(); err != nil {
			return nil, fmt.Errorf("validating standard line[%s]: %w", line.ID, err)
		}
	}

	return stdLines, nil
}

func (e *Engine) IsLineBillableAsOf(_ context.Context, input billing.IsLineBillableAsOfInput) (bool, error) {
	if err := input.Validate(); err != nil {
		return false, fmt.Errorf("validating input: %w", err)
	}

	return !lo.IsEmpty(input.ResolvedBillablePeriod), nil
}
