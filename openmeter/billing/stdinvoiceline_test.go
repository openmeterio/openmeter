package billing

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/mo"
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
					PerUnitAmount:          alpacadecimal.NewFromInt(10),
					Quantity:               alpacadecimal.NewFromInt(-1),
				},
			},
		},
	}

	require.NoError(t, line.Validate())
}

func TestExistingLineOverrideApplyStandardLineDoesNotMutateOriginalUsageBasedPrice(t *testing.T) {
	line := validStandardLineForValidation()
	originalPrice := line.UsageBased.Price.Clone()
	overridePrice := productcatalog.NewPriceFrom(productcatalog.FlatPrice{
		Amount: alpacadecimal.NewFromInt(2),
	})

	updatedLine, err := ExistingLineOverride{
		Price: mo.Some(overridePrice),
	}.Apply(line.AsGenericLine())

	require.NoError(t, err)
	require.True(t, originalPrice.Equal(line.UsageBased.Price))

	updatedStandardLine, err := updatedLine.AsInvoiceLine().AsStandardLine()
	require.NoError(t, err)
	require.True(t, overridePrice.Equal(updatedStandardLine.UsageBased.Price))
	require.NotSame(t, overridePrice, updatedStandardLine.UsageBased.Price)
}

func TestStandardLineDoesNotExposeInvoiceAtAccessor(t *testing.T) {
	type invoiceAtReader interface {
		GetInvoiceAt() time.Time
	}

	line := validStandardLineForValidation()

	// StandardLine.InvoiceAt is retained only to display the original invoice-at
	// timestamp when a gathering line is rendered into a standard invoice line.
	// Standard-line business logic must not discover it through an accessor and
	// treat it as scheduling state.
	_, valueImplements := any(line).(invoiceAtReader)
	_, pointerImplements := any(&line).(invoiceAtReader)
	require.False(t, valueImplements, "StandardLine must not expose InvoiceAt through an accessor")
	require.False(t, pointerImplements, "*StandardLine must not expose InvoiceAt through an accessor")
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
