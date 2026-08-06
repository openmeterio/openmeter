package creditpurchase

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestNewDetailedLine(t *testing.T) {
	customCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode("TOKENS").
		WithName("Tokens").
		WithPrecision(4).
		Build()
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}
	costBasis := alpacadecimal.NewFromInt(2123456).Shift(-6)
	customCurrencyAmount := alpacadecimal.NewFromInt(314159).Shift(-5)
	roundedCustomCurrencyAmount := customCurrency.RoundToPrecision(customCurrencyAmount)
	fiatOverage := fiatCurrency.RoundToPrecision(roundedCustomCurrencyAmount.Mul(costBasis))

	line, err := newDetailedLine(newDetailedLineInput{
		Namespace:     "namespace",
		InvoiceID:     "invoice-id",
		Name:          "usage (overage)",
		ServicePeriod: servicePeriod,
		CreditCurrency: currencies.Currency{
			NamespacedID: models.NamespacedID{
				Namespace: "namespace",
				ID:        "custom-currency-id",
			},
			Currency: customCurrency,
		},
		CreditAmount:      customCurrencyAmount,
		ResolvedCostBasis: costBasis,
		FiatCurrency:      fiatCurrency,
		FiatAmount:        fiatOverage,
	})
	require.NoError(t, err)

	require.Equal(t, "usage (overage)", line.Name)
	require.Equal(t, roundedCustomCurrencyAmount, line.Quantity)
	require.Equal(t, costBasis, line.PerUnitAmount)
	require.Equal(t, fiatOverage, line.Totals.Amount)
	require.Equal(t, fiatOverage, line.Totals.Total)
	require.NoError(t, line.Totals.Validate())
	require.NoError(t, line.Validate())
}

func TestNewDetailedLineRequiresName(t *testing.T) {
	customCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode("TOKENS").
		WithName("Tokens").
		WithPrecision(0).
		Build()
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	_, err = newDetailedLine(newDetailedLineInput{
		Namespace:     "namespace",
		InvoiceID:     "invoice-id",
		ServicePeriod: servicePeriod,
		CreditCurrency: currencies.Currency{
			NamespacedID: models.NamespacedID{
				Namespace: "namespace",
				ID:        "custom-currency-id",
			},
			Currency: customCurrency,
		},
		CreditAmount:      alpacadecimal.NewFromInt(3),
		ResolvedCostBasis: alpacadecimal.NewFromInt(2),
		FiatCurrency:      fiatCurrency,
		FiatAmount:        alpacadecimal.NewFromInt(6),
	})
	require.ErrorContains(t, err, "name is required")
}

func TestNewDetailedLineRejectsInconsistentFiatAmount(t *testing.T) {
	customCurrency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode("TOKENS").
		WithName("Tokens").
		WithPrecision(0).
		Build()
	require.NoError(t, err)

	fiatCurrency, err := currencyx.NewFiatCurrency("USD")
	require.NoError(t, err)

	servicePeriod := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	_, err = newDetailedLine(newDetailedLineInput{
		Namespace:     "namespace",
		InvoiceID:     "invoice-id",
		Name:          "usage (overage)",
		ServicePeriod: servicePeriod,
		CreditCurrency: currencies.Currency{
			NamespacedID: models.NamespacedID{
				Namespace: "namespace",
				ID:        "custom-currency-id",
			},
			Currency: customCurrency,
		},
		CreditAmount:      alpacadecimal.NewFromInt(3),
		ResolvedCostBasis: alpacadecimal.NewFromInt(2),
		FiatCurrency:      fiatCurrency,
		FiatAmount:        alpacadecimal.NewFromInt(5),
	})
	require.ErrorContains(t, err, "totals amount does not match")
}
