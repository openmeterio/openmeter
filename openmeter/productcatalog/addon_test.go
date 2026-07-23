package productcatalog

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestValidateAddonRateCardCurrencies(t *testing.T) {
	customCurrency := currencyx.Code("CREDITS")
	usd := currencyx.Code(currency.USD)
	eur := currencyx.Code(currency.EUR)

	newRateCard := func(override *currencies.CurrencyReference) RateCard {
		return &FlatFeeRateCard{
			RateCardMeta: RateCardMeta{
				Key:      "flat-fee",
				Name:     "Flat fee",
				Currency: override,
				Price: NewPriceFrom(FlatPrice{
					Amount: decimal.NewFromInt(10),
				}),
			},
		}
	}

	tests := []struct {
		name            string
		defaultCurrency currencies.CurrencyReference
		override        *currencies.CurrencyReference
		expectedError   error
	}{
		{
			name:          "missing add-on currency",
			expectedError: ErrCurrencyInvalid,
		},
		{
			name:            "custom default without override",
			defaultCurrency: currencies.NewCurrencyReference(customCurrency),
		},
		{
			name:            "fiat default with custom override",
			defaultCurrency: currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
			override:        currencyReferencePointer(customCurrency),
		},
		{
			name:            "custom default rejects override",
			defaultCurrency: currencies.NewCurrencyReference(customCurrency),
			override:        currencyReferencePointer(usd),
			expectedError:   ErrRateCardCurrencyOverrideNotAllowed,
		},
		{
			name:            "fiat default rejects redundant override",
			defaultCurrency: currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
			override:        currencyReferencePointer(usd),
			expectedError:   ErrRateCardCurrencyOverrideRedundant,
		},
		{
			name:            "fiat default rejects second fiat",
			defaultCurrency: currencies.NewCurrencyReference(currencyx.Code(currency.USD)),
			override:        currencyReferencePointer(eur),
			expectedError:   ErrPlanMultipleFiatCurrencies,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given:
			// - an add-on with one priced rate card
			// when:
			// - its default/override currency relationship is validated
			// then:
			// - it follows the same one-fiat and custom-default rules as a plan
			addon := Addon{
				AddonMeta: AddonMeta{Currency: tt.defaultCurrency},
				RateCards: RateCards{newRateCard(tt.override)},
			}

			err := ValidateAddonRateCardCurrencies()(addon)
			if tt.expectedError == nil {
				assert.NoError(t, err)
				return
			}

			assert.ErrorIs(t, err, tt.expectedError)
		})
	}
}

func TestValidateAddonWithCurrenciesRequiresResolvedReferences(t *testing.T) {
	usd := currencyx.Code(currency.USD)
	custom := currencyx.Code("CREDITS")

	tests := []struct {
		name  string
		addon Addon
	}{
		{
			name: "add-on currency",
			addon: Addon{
				AddonMeta: AddonMeta{Currency: currencies.NewCurrencyReference(custom)},
			},
		},
		{
			name: "rate card currency",
			addon: Addon{
				AddonMeta: AddonMeta{Currency: mustFiatCurrencyReference(t, usd)},
				RateCards: RateCards{
					newCurrencyTestRateCard("ratecard", currencies.NewCurrencyReference(custom)),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddonWithCurrencies()(tt.addon)
			require.ErrorContains(t, err, "is not resolved")
			require.False(t, models.IsGenericValidationError(err))
		})
	}
}

func TestValidateAddonWithCurrenciesUsesResolvedCurrency(t *testing.T) {
	// given:
	// - two managed custom currency resources reuse the same code
	// - only the older resource has a cost-basis pair with USD
	usd := currencyx.Code(currency.USD)
	oldCredits := mustManagedCustomCurrency(t, "old-credits-id", "CREDITS")
	oldCredits.CostBasis = &[]currencies.CostBasis{{
		CostBasis: currencyx.CostBasis{FiatCode: usd},
	}}
	newCredits := mustManagedCustomCurrency(t, "new-credits-id", "CREDITS")
	newCredits.CostBasis = &[]currencies.CostBasis{}

	addon := Addon{
		AddonMeta: AddonMeta{Currency: mustFiatCurrencyReference(t, usd)},
		RateCards: RateCards{
			newCurrencyTestRateCard("old", oldCredits.Reference()),
			newCurrencyTestRateCard("new", newCredits.Reference()),
		},
	}

	// when:
	// - add-on cost-basis validation checks both priced rate cards
	err := ValidateAddonWithCurrencies()(addon)

	// then:
	// - each managed identity is checked independently despite the shared code
	require.ErrorIs(t, err, ErrCurrencyCostBasisNotFound)
}
