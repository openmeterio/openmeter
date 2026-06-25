package realizations

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/invoiceupdater"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
)

type BuildCreditThenInvoiceGatheringPreviewRunInput struct {
	Charge flatfee.Charge
	Line   billing.StandardLine
}

func (i BuildCreditThenInvoiceGatheringPreviewRunInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if err := i.Line.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("line: %w", err))
	}

	lineChargeID := lo.FromPtrOr(i.Line.ChargeID, "<nil>")
	if lineChargeID != i.Charge.ID {
		errs = append(errs, fmt.Errorf("line charge id mismatch: got %s, want %s", lineChargeID, i.Charge.ID))
	}

	if i.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		errs = append(errs, fmt.Errorf("unsupported settlement mode for gathering preview: %s", i.Charge.Intent.GetSettlementMode()))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type BuildCreditThenInvoiceGatheringPreviewRunResult struct {
	Run flatfee.RealizationRun
}

// BuildCreditThenInvoiceGatheringPreviewRun creates the charge run shape needed
// to map a gathering invoice preview line without persisting realization state.
// Preview intentionally does not allocate credits: get/list expansion must stay
// side-effect-free, so returned standard lines show charge-rated totals before
// charge credit allocation.
func (s *Service) BuildCreditThenInvoiceGatheringPreviewRun(in BuildCreditThenInvoiceGatheringPreviewRunInput) (BuildCreditThenInvoiceGatheringPreviewRunResult, error) {
	if err := in.Validate(); err != nil {
		return BuildCreditThenInvoiceGatheringPreviewRunResult{}, err
	}

	currencyCalculator, err := in.Charge.Intent.GetCurrency().Calculator()
	if err != nil {
		return BuildCreditThenInvoiceGatheringPreviewRunResult{}, fmt.Errorf("get currency calculator: %w", err)
	}

	amountAfterProration, err := invoiceupdater.GetFlatFeePerUnitAmount(&in.Line)
	if err != nil {
		return BuildCreditThenInvoiceGatheringPreviewRunResult{}, fmt.Errorf("get flat fee line amount: %w", err)
	}

	amountAfterProration = currencyCalculator.RoundToPrecision(amountAfterProration)

	line, err := rateFlatFeeLine(in.Line, s.ratingService)
	if err != nil {
		return BuildCreditThenInvoiceGatheringPreviewRunResult{}, err
	}

	detailedLines := flatfee.DetailedLines(lo.Map(line.DetailedLines, func(detailedLine billing.DetailedLine, _ int) flatfee.DetailedLine {
		return detailedLine.Base.Clone()
	}))

	runTotals := line.Totals.RoundToPrecision(currencyCalculator)
	runType := flatfee.RealizationRunTypeFinalRealization
	run := flatfee.RealizationRun{
		RealizationRunBase: flatfee.RealizationRunBase{
			ID: flatfee.RealizationRunID{
				Namespace: in.Line.Namespace,
				ID:        fmt.Sprintf("preview-%s", in.Line.ID),
			},
			LineID:                    lo.ToPtr(in.Line.ID),
			InvoiceID:                 lo.ToPtr(in.Line.InvoiceID),
			Type:                      runType,
			InitialType:               runType,
			ServicePeriod:             in.Line.Period,
			AmountAfterProration:      amountAfterProration,
			Totals:                    runTotals,
			NoFiatTransactionRequired: runTotals.Total.IsZero(),
		},
		DetailedLines: mo.Some(detailedLines),
	}

	return BuildCreditThenInvoiceGatheringPreviewRunResult{Run: run}, nil
}
