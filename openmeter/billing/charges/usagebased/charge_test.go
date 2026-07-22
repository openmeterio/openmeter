package usagebased

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/models/costbasis"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	currenciestestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils/currency"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
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
			ManagedBy:  billing.ManuallyManagedLine,
			CustomerID: "customer-1",
			Currency:   currency,
			TaxConfig: productcatalog.TaxCodeConfig{
				TaxCodeID: "tax-code-1",
			},
		},
		IntentMutableFields: IntentMutableFields{
			IntentMutableFields: meta.IntentMutableFields{
				Name:              "usage based",
				ServicePeriod:     period,
				FullServicePeriod: period,
				BillingPeriod:     period,
			},
			InvoiceAt: period.To,
			Price: *productcatalog.NewPriceFrom(productcatalog.UnitPrice{
				Amount: alpacadecimal.NewFromInt(1),
			}),
		},
		SettlementMode: settlementMode,
		FeatureKey:     "feature-key",
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

// TestOverridableIntentGetEffectiveUnitConfig locks in that the override layer is a
// full snapshot of the effective mutable fields: an override created from the effective
// intent inherits the base unit_config, so an unrelated edit does not drop the
// conversion, while an explicit nil is a genuine cleared state.
func TestOverridableIntentGetEffectiveUnitConfig(t *testing.T) {
	unitConfig := &productcatalog.UnitConfig{
		Operation:        productcatalog.UnitConfigOperationDivide,
		ConversionFactor: alpacadecimal.NewFromInt(1000),
		Rounding:         productcatalog.UnitConfigRoundingModeCeiling,
	}

	base := Intent{
		IntentMutableFields: IntentMutableFields{
			Price:      *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
			UnitConfig: unitConfig,
		},
		FeatureKey: "f",
	}

	oi := base.AsOverridableIntent()

	t.Run("override with an unrelated edit inherits the base unit_config", func(t *testing.T) {
		// Mirror how the state machine creates the first override: snapshot the full
		// effective mutable fields, then change one unrelated field only.
		overrideFields := oi.GetEffectiveIntent().IntentMutableFields
		overrideFields.Name = "unrelated override edit"

		withOverride := NewOverridableIntent(base, &overrideFields)

		got := withOverride.GetEffectiveUnitConfig()
		require.NotNil(t, got, "unrelated override must not drop the base unit_config")
		require.True(t, unitConfig.Equal(got))
	})

	t.Run("explicit nil on the override is a cleared state, not inherited", func(t *testing.T) {
		clearedFields := oi.GetEffectiveIntent().IntentMutableFields
		clearedFields.UnitConfig = nil

		cleared := NewOverridableIntent(base, &clearedFields)
		require.Nil(t, cleared.GetEffectiveUnitConfig())
	})
}
