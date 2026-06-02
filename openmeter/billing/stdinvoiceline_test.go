package billing

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/billing/models/totals"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestStandardLineValidateAllowsNonNegativeTotals(t *testing.T) {
	line := validStandardLineForValidation()
	line.Totals.Total = alpacadecimal.Zero

	require.NoError(t, line.Validate())

	line.Totals.Total = alpacadecimal.NewFromInt(1)

	require.NoError(t, line.Validate())
}

func TestStandardLineValidateRejectsNegativeTotals(t *testing.T) {
	line := validStandardLineForValidation()
	line.Totals.Total = alpacadecimal.NewFromInt(-1)

	require.ErrorContains(t, line.Validate(), "totals: total is negative")
}

func TestStandardLineValidateAllowsNegativeDetailedLineQuantityWithPositiveTotal(t *testing.T) {
	line := validStandardLineForValidation()
	line.Totals.Total = alpacadecimal.NewFromInt(1)
	line.DetailedLines = DetailedLines{
		{
			DetailedLineBase: DetailedLineBase{
				InvoiceID: line.InvoiceID,
				Base: stddetailedline.Base{
					ManagedResource: models.ManagedResource{
						NamespacedModel: models.NamespacedModel{
							Namespace: line.Namespace,
						},
						ID:   "detail_123",
						Name: "usage correction",
					},
					Category:               stddetailedline.CategoryRegular,
					ChildUniqueReferenceID: "detail_123",
					PaymentTerm:            productcatalog.InArrearsPaymentTerm,
					ServicePeriod:          line.Period,
					Currency:               line.Currency,
					PerUnitAmount:          alpacadecimal.NewFromInt(10),
					Quantity:               alpacadecimal.NewFromInt(-1),
				},
			},
		},
	}

	require.NoError(t, line.Validate())
}

func validStandardLineForValidation() StandardLine {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	return StandardLine{
		StandardLineBase: StandardLineBase{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: "test-namespace",
				},
				ID:   "line_123",
				Name: "usage",
			},
			ManagedBy: SystemManagedLine,
			InvoiceID: "inv_123",
			Currency:  currencyx.Code("USD"),
			Period: timeutil.ClosedPeriod{
				From: start,
				To:   start.Add(time.Hour),
			},
			InvoiceAt: start.Add(time.Hour),
			Totals: totals.Totals{
				Total: alpacadecimal.NewFromInt(1),
			},
		},
		UsageBased: &UsageBasedLine{
			Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
	}
}
