package flatfee

import (
	"github.com/samber/lo"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	billingrating "github.com/openmeterio/openmeter/openmeter/billing/rating"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

type DetailedLine = stddetailedline.Base

type DetailedLines = stddetailedline.Bases

func NewDetailedLinesFromRating(defaultServicePeriod timeutil.ClosedPeriod, lines billingrating.DetailedLines) DetailedLines {
	return lo.Map(lines, func(line billingrating.DetailedLine, idx int) DetailedLine {
		return DetailedLine{
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
		}
	})
}
