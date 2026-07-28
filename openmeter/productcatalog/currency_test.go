package productcatalog

import (
	"testing"
	"time"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidateCurrencyLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	currency := mustManagedCustomCurrency(t, "custom-currency-id", currencyx.Code("CREDITS"))
	currency.CostBasis = &[]currencies.CostBasis{}

	t.Run("archived custom currency is rejected", func(t *testing.T) {
		archived := currency.Clone()
		archived.DeletedAt = lo.ToPtr(now)

		require.ErrorIs(t, ValidateCurrency()(archived.Reference()), ErrCurrencyNotFound)
	})

	t.Run("scheduled custom currency deletion remains valid", func(t *testing.T) {
		scheduled := currency.Clone()
		scheduled.DeletedAt = lo.ToPtr(now.Add(time.Hour))

		require.NoError(t, ValidateCurrency()(scheduled.Reference()))
	})
}

func TestValidateCurrencyWithOverrideCostBasisPolicy(t *testing.T) {
	usd := mustFiatCurrencyReference(t, "USD")
	eur := mustFiatCurrencyReference(t, "EUR")

	credits := mustManagedCustomCurrency(t, "credits-id", "CREDITS")
	credits.CostBasis = &[]currencies.CostBasis{}
	creditsReference := credits.Reference()

	tokens := mustManagedCustomCurrency(t, "tokens-id", "TOKENS")
	tokens.CostBasis = &[]currencies.CostBasis{}
	tokensReference := tokens.Reference()

	unresolved := currencies.NewCurrencyReference("UNRESOLVED")

	tests := []struct {
		name             string
		reference        currencies.CurrencyReference
		override         currencies.CurrencyReference
		validationOption ValidationOption
		expected         error
		errorContains    string
	}{
		{
			name:             "credit only accepts custom override without cost basis",
			reference:        usd,
			override:         creditsReference,
			validationOption: ValidationOptionCostBasisRequiredFalse,
		},
		{
			name:             "credit then invoice rejects custom override without cost basis",
			reference:        usd,
			override:         creditsReference,
			validationOption: ValidationOptionCostBasisRequiredTrue,
			expected:         ErrCurrencyCostBasisNotFound,
		},
		{
			name:             "unknown option remains strict",
			reference:        usd,
			override:         creditsReference,
			validationOption: ValidationOption("unknown"),
			expected:         ErrCurrencyCostBasisNotFound,
		},
		{
			name:             "credit only still rejects fiat override",
			reference:        usd,
			override:         eur,
			validationOption: ValidationOptionCostBasisRequiredFalse,
			errorContains:    "fiat currency cannot be overridden",
		},
		{
			name:             "credit only still rejects custom default override",
			reference:        creditsReference,
			override:         tokensReference,
			validationOption: ValidationOptionCostBasisRequiredFalse,
			expected:         ErrCurrencyInvalid,
			errorContains:    "custom currency cannot be overridden",
		},
		{
			name:             "credit only still rejects unresolved override",
			reference:        usd,
			override:         unresolved,
			validationOption: ValidationOptionCostBasisRequiredFalse,
			errorContains:    "is not resolved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - resolved containing and override currency references unless the scenario tests resolution
			// when:
			// - override validation runs with or without cost-basis enforcement
			err := ValidateCurrencyWithOverride(tt.reference, tt.validationOption)(&tt.override)

			// then:
			// - only the active cost-basis rule depends on the policy
			if tt.expected == nil && tt.errorContains == "" {
				require.NoError(t, err)
				return
			}

			if tt.expected != nil {
				require.ErrorIs(t, err, tt.expected)
			}
			if tt.errorContains != "" {
				require.ErrorContains(t, err, tt.errorContains)
			}
		})
	}
}

func TestRateCardCurrencyRequiresPrice(t *testing.T) {
	custom := currencyx.Code("CREDITS")

	t.Run("currency without price is invalid", func(t *testing.T) {
		err := (RateCardMeta{Currency: currencyReferencePointer(custom)}).Validate()
		require.ErrorIs(t, err, ErrRateCardCurrencyRequiresPrice)
	})

	t.Run("currency with price is valid", func(t *testing.T) {
		err := (RateCardMeta{
			Currency: currencyReferencePointer(custom),
			Price: NewPriceFrom(FlatPrice{
				Amount:      decimal.NewFromInt(1),
				PaymentTerm: InAdvancePaymentTerm,
			}),
		}).Validate()
		require.NoError(t, err)
	})
}

func newCurrencyTestRateCard(key string, reference currencies.CurrencyReference) RateCard {
	return &FlatFeeRateCard{RateCardMeta: RateCardMeta{
		Key:      key,
		Name:     key,
		Currency: lo.ToPtr(reference),
		Price: NewPriceFrom(FlatPrice{
			Amount:      decimal.NewFromInt(1),
			PaymentTerm: InAdvancePaymentTerm,
		}),
	}}
}

func mustManagedCustomCurrency(t *testing.T, id string, code currencyx.Code) currencies.Currency {
	t.Helper()

	currency, err := currencyx.NewCurrencyBuilder(currencyx.CurrencyTypeCustom).
		WithCode(code).
		WithName(code.String()).
		Build()
	require.NoError(t, err)

	return currencies.Currency{
		NamespacedID: models.NamespacedID{ID: id},
		Currency:     currency,
	}
}

func mustFiatCurrencyReference(t *testing.T, code currencyx.Code) currencies.CurrencyReference {
	t.Helper()

	currency, err := currencies.NewFiatCurrency(code)
	require.NoError(t, err)

	return currency.Reference()
}

func currencyReferencePointer(code currencyx.Code) *currencies.CurrencyReference {
	reference := currencies.NewCurrencyReference(code)
	return &reference
}
