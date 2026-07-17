package productcatalog

import (
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestValidateAddonRateCardCurrencies(t *testing.T) {
	customCurrency := currencyx.Code("CREDITS")
	usd := currencyx.Code(currency.USD)
	eur := currencyx.Code(currency.EUR)

	newRateCard := func(override currencyx.CurrencyIdentity) RateCard {
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
		defaultCurrency currencyx.CurrencyIdentity
		override        currencyx.CurrencyIdentity
		expectedError   error
	}{
		{
			name:          "missing add-on currency",
			expectedError: ErrCurrencyInvalid,
		},
		{
			name:            "custom default without override",
			defaultCurrency: customCurrency,
		},
		{
			name:            "fiat default with custom override",
			defaultCurrency: currencyx.Code(currency.USD),
			override:        customCurrency,
		},
		{
			name:            "custom default rejects override",
			defaultCurrency: customCurrency,
			override:        usd,
			expectedError:   ErrRateCardCurrencyOverrideNotAllowed,
		},
		{
			name:            "fiat default rejects redundant override",
			defaultCurrency: currencyx.Code(currency.USD),
			override:        usd,
			expectedError:   ErrRateCardCurrencyOverrideRedundant,
		},
		{
			name:            "fiat default rejects second fiat",
			defaultCurrency: currencyx.Code(currency.USD),
			override:        eur,
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
