package flatfee

import (
	"errors"
	"fmt"

	"github.com/samber/lo"

	chargedetailedline "github.com/openmeterio/openmeter/openmeter/billing/charges/models/detailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/creditsapplied"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type DetailedLine = chargedetailedline.Base

type DetailedLines []DetailedLine

func (l DetailedLines) Clone() DetailedLines {
	return lo.Map(l, func(line DetailedLine, _ int) DetailedLine {
		return line.Clone()
	})
}

func (l DetailedLines) mapBase(fn func(stddetailedline.Bases) (stddetailedline.Bases, error)) (DetailedLines, error) {
	out := l.Clone()
	bases, err := fn(lo.Map(out, func(line DetailedLine, _ int) stddetailedline.Base {
		return line.Base
	}))
	if err != nil {
		return nil, err
	}

	for idx := range out {
		out[idx].Base = bases[idx]
	}

	return out, nil
}

func (l DetailedLines) WithCreditsApplied(
	credits creditsapplied.CreditsApplied,
	currency currencyx.Currency,
) (DetailedLines, error) {
	return l.mapBase(func(bases stddetailedline.Bases) (stddetailedline.Bases, error) {
		return bases.WithCreditsApplied(credits, currency)
	})
}

func (l DetailedLines) SumTotals() totals.Totals {
	return stddetailedline.Bases(lo.Map(l, func(line DetailedLine, _ int) stddetailedline.Base {
		return line.Base
	})).SumTotals()
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

func NewDetailedLinesFromRating(defaultServicePeriod timeutil.ClosedPeriod, lines billingrating.DetailedLines) DetailedLines {
	return lo.Map(lines, func(line billingrating.DetailedLine, idx int) DetailedLine {
		return DetailedLine{
			AmountDiscounts: chargedetailedline.MapAmountDiscountsFromBilling(line.AmountDiscounts),
			Base: stddetailedline.Base{
				ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
					Name: line.Name,
				}),
				Category:               lo.CoalesceOrEmpty(line.Category, stddetailedline.CategoryRegular),
				ChildUniqueReferenceID: line.ChildUniqueReferenceID,
				Index:                  lo.ToPtr(idx),
				PaymentTerm:            lo.CoalesceOrEmpty(line.PaymentTerm, productcatalog.InArrearsPaymentTerm),
				ServicePeriod:          lo.FromPtrOr(line.Period, defaultServicePeriod),
				PerUnitAmount:          line.PerUnitAmount,
				Quantity:               line.Quantity,
				Totals:                 line.Totals,
			},
		}
	})
}
