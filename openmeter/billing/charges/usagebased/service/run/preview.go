package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/samber/mo"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedrating "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

type BuildCreditThenInvoiceGatheringPreviewRunInput struct {
	Charge             usagebased.Charge
	CustomerOverride   billing.CustomerOverrideWithDetails
	FeatureMeter       feature.FeatureMeter
	Type               usagebased.RealizationRunType
	StoredAtLT         time.Time
	ServicePeriodTo    time.Time
	LineID             string
	InvoiceID          string
	CurrencyCalculator currencyx.Calculator
}

func (i BuildCreditThenInvoiceGatheringPreviewRunInput) Validate() error {
	var errs []error

	if err := i.Charge.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("charge: %w", err))
	}

	if i.Charge.Intent.SettlementMode != productcatalog.CreditThenInvoiceSettlementMode {
		errs = append(errs, fmt.Errorf("unsupported settlement mode for gathering preview: %s", i.Charge.Intent.SettlementMode))
	}

	if i.CustomerOverride.Customer == nil {
		errs = append(errs, errors.New("expanded customer is required"))
	}

	if i.FeatureMeter.Meter == nil {
		errs = append(errs, errors.New("feature meter is required"))
	}

	if i.FeatureMeter.Feature.ID == "" {
		errs = append(errs, errors.New("feature id is required"))
	}

	if err := i.Type.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("type: %w", err))
	}

	if i.StoredAtLT.IsZero() {
		errs = append(errs, errors.New("stored at lt is required"))
	}

	if i.ServicePeriodTo.IsZero() {
		errs = append(errs, errors.New("service period to is required"))
	}

	if i.LineID == "" {
		errs = append(errs, errors.New("line id is required"))
	}

	if i.InvoiceID == "" {
		errs = append(errs, errors.New("invoice id is required"))
	}

	if err := i.CurrencyCalculator.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("currency calculator: %w", err))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type BuildCreditThenInvoiceGatheringPreviewRunResult struct {
	Run  usagebased.RealizationRun
	Runs usagebased.RealizationRuns
}

// BuildCreditThenInvoiceGatheringPreviewRun creates the charge run shape needed
// to map a gathering invoice preview line without persisting realization state.
// Preview intentionally does not allocate credits: get/list expansion must stay
// side-effect-free, so returned standard lines show charge-rated totals before
// charge credit allocation.
func (s *Service) BuildCreditThenInvoiceGatheringPreviewRun(ctx context.Context, in BuildCreditThenInvoiceGatheringPreviewRunInput) (BuildCreditThenInvoiceGatheringPreviewRunResult, error) {
	if err := in.Validate(); err != nil {
		return BuildCreditThenInvoiceGatheringPreviewRunResult{}, err
	}

	ratingResult, err := s.rater.GetDetailedRatingForUsage(ctx, usagebasedrating.GetDetailedRatingForUsageInput{
		Charge:          in.Charge,
		StoredAtLT:      in.StoredAtLT,
		ServicePeriodTo: in.ServicePeriodTo,
		Customer:        in.CustomerOverride,
		FeatureMeter:    in.FeatureMeter,
	})
	if err != nil {
		return BuildCreditThenInvoiceGatheringPreviewRunResult{}, fmt.Errorf("get detailed rating for usage: %w", err)
	}

	runTotals := ratingResult.Totals.RoundToPrecision(in.CurrencyCalculator)
	if runTotals.Total.IsNegative() {
		return BuildCreditThenInvoiceGatheringPreviewRunResult{}, usagebased.ErrChargeTotalIsNegative.
			WithAttrs(models.Attributes{
				"total":     runTotals.Total.String(),
				"charge_id": in.Charge.ID,
			})
	}

	previewRun := usagebased.RealizationRun{
		RealizationRunBase: usagebased.RealizationRunBase{
			ID: usagebased.RealizationRunID{
				Namespace: in.Charge.Namespace,
				ID:        fmt.Sprintf("preview-%s", in.LineID),
			},
			FeatureID:                 in.FeatureMeter.Feature.ID,
			Type:                      in.Type,
			InitialType:               in.Type,
			StoredAtLT:                in.StoredAtLT,
			ServicePeriodTo:           in.ServicePeriodTo,
			LineID:                    lo.ToPtr(in.LineID),
			InvoiceID:                 lo.ToPtr(in.InvoiceID),
			MeteredQuantity:           ratingResult.Quantity,
			Totals:                    runTotals,
			NoFiatTransactionRequired: runTotals.Total.IsZero(),
		},
		DetailedLines: mo.Some(ratingResult.DetailedLines),
	}

	return BuildCreditThenInvoiceGatheringPreviewRunResult{
		Run:  previewRun,
		Runs: slices.Concat(in.Charge.Realizations, usagebased.RealizationRuns{previewRun}),
	}, nil
}
