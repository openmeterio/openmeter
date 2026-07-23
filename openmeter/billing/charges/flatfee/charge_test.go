package flatfee

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestIntentValidateCostBasis(t *testing.T) {
	customCurrency := newCustomCurrency(t)
	fiatCurrency := currenciestestutils.NewFiatCurrency(t, "USD")
	validCostBasis := newManualCostBasisIntent(t)
	invalidCostBasis := costbasis.Intent{}

	tests := []struct {
		name           string
		currency       currencies.Currency
		settlementMode productcatalog.SettlementMode
		costBasis      *costbasis.Intent
		wantErr        string
	}{
		{
			name:           "custom currency with credit then invoice requires cost basis",
			currency:       customCurrency,
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			wantErr:        "cost basis is required",
		},
		{
			name:           "custom currency with credit then invoice accepts valid cost basis",
			currency:       customCurrency,
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			costBasis:      &validCostBasis,
		},
		{
			name:           "custom currency with credit then invoice validates cost basis",
			currency:       customCurrency,
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			costBasis:      &invalidCostBasis,
			wantErr:        "cost basis",
		},
		{
			name:           "custom currency with credit only does not require cost basis",
			currency:       customCurrency,
			settlementMode: productcatalog.CreditOnlySettlementMode,
		},
		{
			name:           "custom currency with credit only rejects cost basis",
			currency:       customCurrency,
			settlementMode: productcatalog.CreditOnlySettlementMode,
			costBasis:      &validCostBasis,
			wantErr:        "cost basis must not be set",
		},
		{
			name:           "fiat currency with credit then invoice does not require cost basis",
			currency:       fiatCurrency,
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
		},
		{
			name:           "fiat currency with credit then invoice rejects cost basis",
			currency:       fiatCurrency,
			settlementMode: productcatalog.CreditThenInvoiceSettlementMode,
			costBasis:      &validCostBasis,
			wantErr:        "cost basis must not be set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent := newValidIntent(t, tt.currency, tt.settlementMode)
			intent.CostBasis = tt.costBasis

			validations := []struct {
				name string
				err  error
			}{
				{name: "intent", err: intent.Validate()},
				{name: "overridable intent", err: intent.AsOverridableIntent().Validate()},
			}

			for _, validation := range validations {
				t.Run(validation.name, func(t *testing.T) {
					if tt.wantErr == "" {
						require.NoError(t, validation.err)
						return
					}

					require.ErrorContains(t, validation.err, tt.wantErr)
				})
			}
		})
	}
}

func TestOverridableIntentPreservesCostBasis(t *testing.T) {
	intent := newValidIntent(t, newCustomCurrency(t), productcatalog.CreditThenInvoiceSettlementMode)
	costBasis := newManualCostBasisIntent(t)
	intent.CostBasis = &costBasis

	overridable := intent.AsOverridableIntent()
	requireManualCostBasisIntent(t, overridable.GetCostBasisIntent())
	requireManualCostBasisIntent(t, overridable.GetBaseIntent().CostBasis)
	requireManualCostBasisIntent(t, overridable.GetEffectiveIntent().CostBasis)

	baseIntent, err := overridable.GetIntentForTarget(meta.ChangeTargetBase)
	require.NoError(t, err)
	requireManualCostBasisIntent(t, baseIntent.CostBasis)

	overrideLayer := intent.IntentMutableFields.Clone()
	overrideLayer.Name = "override"
	overridable = NewOverridableIntent(intent, &overrideLayer)

	overrideIntent, err := overridable.GetIntentForTarget(meta.ChangeTargetOverride)
	require.NoError(t, err)
	requireManualCostBasisIntent(t, overrideIntent.CostBasis)

	returnedCostBasis := overridable.GetCostBasisIntent()
	*returnedCostBasis = costbasis.NewIntent(costbasis.DynamicIntent{
		FiatCurrency: newFiatCurrency(t, "EUR"),
	})
	requireManualCostBasisIntent(t, overridable.GetCostBasisIntent())
}

func newCustomCurrency(t testing.TB) currencies.Currency {
	t.Helper()

	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(currencyx.Code("TOKENS")).
		WithName("Tokens").
		Build()
	require.NoError(t, err)

	return currencies.Currency{Currency: currency}
}

func newValidIntent(t testing.TB, currency currencies.Currency, settlementMode productcatalog.SettlementMode) Intent {
	t.Helper()

	period := timeutil.ClosedPeriod{
		From: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	}

	return Intent{
		Intent: meta.Intent{
			ManagedBy:  "system",
			CustomerID: "customer-1",
			Currency:   currency,
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: "tax-code-1",
			},
		},
		IntentMutableFields: IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "flat fee",
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			InvoiceAt:             period.From,
			PaymentTerm:           productcatalog.InAdvancePaymentTerm,
			AmountBeforeProration: alpacadecimal.NewFromInt(100),
		},
		SettlementMode: settlementMode,
	}
}

func newManualCostBasisIntent(t testing.TB) costbasis.Intent {
	t.Helper()

	return costbasis.NewIntent(costbasis.ManualIntent{
		FiatCurrency: newFiatCurrency(t, "USD"),
		Rate:         alpacadecimal.NewFromInt(2),
	})
}

func newFiatCurrency(t testing.TB, code currencyx.Code) *currencyx.FiatCurrency {
	t.Helper()

	fiatCurrency, err := currencyx.NewFiatCurrency(code)
	require.NoError(t, err)

	return fiatCurrency
}

func requireManualCostBasisIntent(t testing.TB, intent *costbasis.Intent) {
	t.Helper()
	require.NotNil(t, intent)
	require.Equal(t, costbasis.ModeManual, intent.Kind())

	manualIntent, err := intent.AsManual()
	require.NoError(t, err)
	require.Equal(t, float64(2), manualIntent.Rate.InexactFloat64())
	require.Equal(t, currencyx.Code("USD"), manualIntent.FiatCurrency.Details().Code)
}

func TestCalculateAmountAfterProration(t *testing.T) {
	// 2026-01-01 to 2026-02-01 (full month)
	fullMonthStart := datetime.MustParseTimeInLocation(t, "2026-01-01T00:00:00Z", time.UTC).AsTime()
	fullMonthEnd := datetime.MustParseTimeInLocation(t, "2026-02-01T00:00:00Z", time.UTC).AsTime()
	// 2026-01-01 to 2026-01-16 (half month, 15 out of 31 days)
	halfMonthEnd := datetime.MustParseTimeInLocation(t, "2026-01-16T00:00:00Z", time.UTC).AsTime()

	fullMonth := timeutil.ClosedPeriod{
		From: fullMonthStart,
		To:   fullMonthEnd,
	}

	halfMonth := timeutil.ClosedPeriod{
		From: fullMonthStart,
		To:   halfMonthEnd,
	}

	amount100 := alpacadecimal.NewFromInt(100)

	baseIntent := func() Intent {
		return Intent{
			Intent: meta.Intent{
				CustomerID: "cust-1",
				Currency:   currenciestestutils.NewFiatCurrency(t, "USD"),
				ManagedBy:  "system",
			},
			IntentMutableFields: IntentMutableFields{
				IntentMutableFields: meta.IntentMutableFields{
					Name:              "test",
					ServicePeriod:     halfMonth,
					FullServicePeriod: fullMonth,
					BillingPeriod:     fullMonth,
				},
				InvoiceAt:             fullMonthStart,
				PaymentTerm:           productcatalog.InAdvancePaymentTerm,
				AmountBeforeProration: amount100,
				ProRating: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			SettlementMode: productcatalog.CreditThenInvoiceSettlementMode,
		}
	}

	t.Run("proration disabled returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.ProRating = productcatalog.ProRatingConfig{
			Enabled: false,
			Mode:    productcatalog.ProRatingModeProratePrices,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("equal periods returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.ServicePeriod = fullMonth
		intent.FullServicePeriod = fullMonth

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("half period returns prorated amount", func(t *testing.T) {
		intent := baseIntent()

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)

		// 15 days out of 31 days = 100 * 15/31 = 48.387... rounded to 48.39 for USD
		expected := alpacadecimal.NewFromFloat(48.39)
		assert.True(t, result.Equal(expected), "expected %s, got %s", expected, result)
	})

	t.Run("zero length service period returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: fullMonthStart,
			To:   fullMonthStart,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("zero length full service period returns full amount", func(t *testing.T) {
		intent := baseIntent()
		intent.FullServicePeriod = timeutil.ClosedPeriod{
			From: fullMonthStart,
			To:   fullMonthStart,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("rounds to currency precision", func(t *testing.T) {
		intent := baseIntent()
		// 10 days out of 31 = 100 * 10/31 = 32.258... rounded to 32.26 for USD
		tenDaysEnd := datetime.MustParseTimeInLocation(t, "2026-01-11T00:00:00Z", time.UTC).AsTime()
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: fullMonthStart,
			To:   tenDaysEnd,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)

		expected := alpacadecimal.NewFromFloat(32.26)
		assert.True(t, result.Equal(expected), "expected %s, got %s", expected, result)
	})

	t.Run("JPY rounds to zero decimal places", func(t *testing.T) {
		intent := baseIntent()
		intent.Currency = currenciestestutils.NewFiatCurrency(t, "JPY")
		intent.AmountBeforeProration = alpacadecimal.NewFromInt(1000)
		// 10 days out of 31 = 1000 * 10/31 = 322.580... rounded to 323 for JPY
		tenDaysEnd := datetime.MustParseTimeInLocation(t, "2026-01-11T00:00:00Z", time.UTC).AsTime()
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: fullMonthStart,
			To:   tenDaysEnd,
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)

		expected := alpacadecimal.NewFromInt(323)
		assert.True(t, result.Equal(expected), "expected %s, got %s", expected, result)
	})

	t.Run("service period exceeding full period returns full amount", func(t *testing.T) {
		intent := baseIntent()
		// ServicePeriod is longer than FullServicePeriod — proration must not increase the amount
		intent.ServicePeriod = timeutil.ClosedPeriod{
			From: fullMonthStart,
			To:   datetime.MustParseTimeInLocation(t, "2026-03-01T00:00:00Z", time.UTC).AsTime(),
		}

		result, err := intent.CalculateAmountAfterProration()
		require.NoError(t, err)
		assert.True(t, result.Equal(amount100), "expected %s, got %s", amount100, result)
	})

	t.Run("invalid currency returns error", func(t *testing.T) {
		intent := baseIntent()
		intent.Currency = currencies.Currency{}

		_, err := intent.CalculateAmountAfterProration()
		require.Error(t, err)
	})
}
