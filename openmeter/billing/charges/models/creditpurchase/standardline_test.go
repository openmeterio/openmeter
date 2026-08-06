package creditpurchase

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/models/stddetailedline"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestWithDetailedLinesPreservesCreditValuation(t *testing.T) {
	creditCurrency, err := currencies.NewFiatCurrency("USD")
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	line := &billing.StandardLine{
		StandardLineBase: billing.StandardLineBase{
			ManagedResource: models.NewManagedResource(models.ManagedResourceInput{
				Namespace: "namespace",
				Name:      "100 USD credits",
			}),
			InvoiceID: "invoice-id",
			Currency:  "USD",
			Period:    servicePeriod,
		},
		UsageBased: &billing.UsageBasedLine{},
		DetailedLines: billing.DetailedLines{
			{
				DetailedLineBase: billing.DetailedLineBase{
					Base: stddetailedline.Base{
						ManagedResource:        models.NewManagedResource(models.ManagedResourceInput{Namespace: "namespace", Name: "100 USD credits"}),
						ChildUniqueReferenceID: CreditPurchaseChildUniqueReferenceID,
					},
				},
			},
		},
	}
	line.DetailedLines[0].ID = "existing-detailed-line-id"

	lineWithDetails, err := WithDetailedLines(WithDetailedLinesInput{
		Line:              line,
		Name:              "100 USD credits",
		CreditCurrency:    creditCurrency,
		CreditAmount:      alpacadecimal.NewFromInt(100),
		ResolvedCostBasis: alpacadecimal.NewFromFloat(0.5),
		FiatCurrency:      fiatCurrency,
		FiatAmount:        alpacadecimal.NewFromInt(50),
	})
	require.NoError(t, err)

	require.Equal(t, float64(0), line.DetailedLines[0].Quantity.InexactFloat64())
	require.Equal(t, float64(0), line.Totals.Total.InexactFloat64())
	require.Len(t, lineWithDetails.DetailedLines, 1)
	require.Equal(t, "existing-detailed-line-id", lineWithDetails.DetailedLines[0].ID)
	require.Equal(t, float64(100), lineWithDetails.DetailedLines[0].Quantity.InexactFloat64())
	require.Equal(t, float64(0.5), lineWithDetails.DetailedLines[0].PerUnitAmount.InexactFloat64())
	require.Equal(t, float64(50), lineWithDetails.DetailedLines[0].Totals.Total.InexactFloat64())
	require.Equal(t, float64(50), lineWithDetails.Totals.Total.InexactFloat64())
}

func TestWithDetailedLinesRejectsCurrencyMismatch(t *testing.T) {
	creditCurrency, err := currencies.NewFiatCurrency("USD")
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("EUR")
	require.NoError(t, err)

	_, err = WithDetailedLines(WithDetailedLinesInput{
		Line: &billing.StandardLine{
			StandardLineBase: billing.StandardLineBase{
				Currency: "USD",
			},
		},
		Name:              "100 USD credits",
		CreditCurrency:    creditCurrency,
		CreditAmount:      alpacadecimal.NewFromInt(100),
		ResolvedCostBasis: alpacadecimal.NewFromFloat(0.5),
		FiatCurrency:      fiatCurrency,
		FiatAmount:        alpacadecimal.NewFromInt(50),
	})
	require.ErrorContains(t, err, "line currency USD does not match fiat currency EUR")
}
