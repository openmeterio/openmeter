package realizations

import (
	"errors"
	"fmt"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
)

type BuildCreditThenInvoiceGatheringPreviewRunInput struct {
	Charge    flatfee.Charge
	LineID    string
	InvoiceID string
}

func (i BuildCreditThenInvoiceGatheringPreviewRunInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.LineID == "" {
		errs = append(errs, errors.New("line ID is required"))
	}

	if i.InvoiceID == "" {
		errs = append(errs, errors.New("invoice ID is required"))
	}

	if i.Charge.Intent.GetSettlementMode() != productcatalog.CreditThenInvoiceSettlementMode {
		errs = append(errs, fmt.Errorf("unsupported settlement mode for gathering preview: %s", i.Charge.Intent.GetSettlementMode()))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

// BuildCreditThenInvoiceGatheringPreviewRun creates the charge run shape needed
// to map a gathering invoice preview line without persisting realization state.
// Preview intentionally does not allocate credits: get/list expansion must stay
// side-effect-free, so returned standard lines show charge-rated totals before
// charge credit allocation.
func (s *Service) BuildCreditThenInvoiceGatheringPreviewRun(in BuildCreditThenInvoiceGatheringPreviewRunInput) (flatfee.RealizationRun, error) {
	if err := in.Validate(); err != nil {
		return flatfee.RealizationRun{}, err
	}

	rateableIntent, err := in.Charge.GetRateableIntent()
	if err != nil {
		return flatfee.RealizationRun{}, fmt.Errorf("getting rateable intent: %w", err)
	}

	ratingResult, err := s.Rate(rateableIntent)
	if err != nil {
		return flatfee.RealizationRun{}, fmt.Errorf("rating flat fee: %w", err)
	}

	runType := flatfee.RealizationRunTypeFinalRealization
	previewAt := clock.Now()
	run := flatfee.RealizationRun{
		RealizationRunBase: flatfee.RealizationRunBase{
			ID: flatfee.RealizationRunID{
				Namespace: in.Charge.Namespace,
				ID:        fmt.Sprintf("preview-%s", in.LineID),
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: previewAt,
				UpdatedAt: previewAt,
			},
			LineID:                    lo.ToPtr(in.LineID),
			InvoiceID:                 lo.ToPtr(in.InvoiceID),
			Type:                      runType,
			InitialType:               runType,
			ServicePeriod:             rateableIntent.ServicePeriod,
			AmountAfterProration:      rateableIntent.AmountAfterProration,
			Totals:                    ratingResult.Totals,
			NoFiatTransactionRequired: ratingResult.Totals.Total.IsZero(),
		},
		DetailedLines: mo.Some(ratingResult.DetailedLines),
	}

	return run, nil
}
