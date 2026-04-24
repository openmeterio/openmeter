package usagebased

import (
	"cmp"
	"errors"
	"fmt"
	"slices"

	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type DetailedLine struct {
	stddetailedline.Base

	PricerReferenceID string  `json:"pricerReferenceID"`
	CorrectsRunID     *string `json:"correctsRunId,omitempty"`
}

func (l DetailedLine) Clone() DetailedLine {
	l.Base = l.Base.Clone()

	if l.CorrectsRunID != nil {
		l.CorrectsRunID = lo.ToPtr(*l.CorrectsRunID)
	}

	return l
}

func (l DetailedLine) Validate() error {
	var errs []error

	if err := l.Base.Validate(stddetailedline.IgnoreQuantityChecks()); err != nil {
		errs = append(errs, err)
	}

	if l.PricerReferenceID == "" {
		errs = append(errs, errors.New("pricer reference id must not be empty"))
	}

	if l.CorrectsRunID != nil && *l.CorrectsRunID == "" {
		errs = append(errs, errors.New("corrects run id must not be empty"))
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}

type DetailedLines []DetailedLine

func NewDetailedLinesFromBilling(
	intent Intent,
	defaultServicePeriod timeutil.ClosedPeriod,
	lines billingrating.DetailedLines,
) DetailedLines {
	return lo.Map(lines, func(line billingrating.DetailedLine, idx int) DetailedLine {
		period := defaultServicePeriod
		if line.Period != nil {
			period = *line.Period
		}

		category := line.Category
		if category == "" {
			category = stddetailedline.CategoryRegular
		}

		return DetailedLine{
			PricerReferenceID: line.ChildUniqueReferenceID,
			Base: stddetailedline.Base{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name: line.Name,
				}),
				ServicePeriod:          period,
				Index:                  lo.ToPtr(idx),
				Currency:               intent.Currency,
				ChildUniqueReferenceID: line.ChildUniqueReferenceID,
				PaymentTerm:            lo.CoalesceOrEmpty(line.PaymentTerm, productcatalog.InArrearsPaymentTerm),
				PerUnitAmount:          line.PerUnitAmount,
				Quantity:               line.Quantity,
				Category:               category,
				CreditsApplied:         line.CreditsApplied,
				Totals:                 line.Totals,
				TaxConfig:              cloneTaxConfig(intent.TaxConfig),
			},
		}
	})
}

func cloneTaxConfig(cfg *productcatalog.TaxConfig) *productcatalog.TaxConfig {
	if cfg == nil {
		return nil
	}

	cloned := cfg.Clone()
	return &cloned
}

func (l DetailedLines) Clone() DetailedLines {
	return lo.Map(l, func(dl DetailedLine, _ int) DetailedLine {
		return dl.Clone()
	})
}

func (l DetailedLines) Sort() {
	slices.SortStableFunc(l, compareDetailedLineForOutput)
}

func compareDetailedLineForOutput(a, b DetailedLine) int {
	if c := a.ServicePeriod.From.Compare(b.ServicePeriod.From); c != 0 {
		return c
	}

	if a.Index != nil && b.Index == nil {
		return -1
	}

	if a.Index == nil && b.Index != nil {
		return 1
	}

	if a.Index != nil && b.Index != nil {
		if c := cmp.Compare(*a.Index, *b.Index); c != 0 {
			return c
		}
	}

	if c := cmp.Compare(a.ChildUniqueReferenceID, b.ChildUniqueReferenceID); c != 0 {
		return c
	}

	return 0
}

func (l DetailedLines) SumTotals() totals.Totals {
	out := totals.Totals{}

	for _, line := range l {
		out = out.Add(line.Totals)
	}

	return out
}

func (l DetailedLines) Validate() error {
	var errs []error

	for idx, line := range l {
		if err := line.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("[%d]: %w", idx, err))
		}
	}

	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
