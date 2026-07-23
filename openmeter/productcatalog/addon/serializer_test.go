package addon

import (
	"encoding/json"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	currencytestutils "github.com/openmeterio/openmeter/openmeter/currencies/testutils"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestAddonSerializationUsesCurrencyCodes(t *testing.T) {
	// given:
	// - an add-on and rate card backed by a managed custom currency
	managedCurrencyValue := currencytestutils.NewManagedCurrency(t, "test", "currency-resource-id", "CREDITS")
	managedCurrency := &managedCurrencyValue
	addon := Addon{
		AddonMeta: productcatalog.AddonMeta{
			Currency: managedCurrency.Reference(),
		},
		RateCards: RateCards{
			{
				RateCard: &productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Currency: lo.ToPtr(managedCurrency.Reference()),
						Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
							Amount: decimal.NewFromInt(1),
						}),
					},
				},
			},
		},
	}

	// when:
	// - the add-on crosses the JSON event boundary
	data, err := json.Marshal(addon)
	require.NoError(t, err)

	// then:
	// - only stable currency codes are serialized, and decoding restores code-only references
	var serialized struct {
		Currency  currencyx.Code `json:"currency"`
		RateCards []struct {
			RateCard struct {
				Currency currencyx.Code `json:"currency"`
			} `json:"RateCard"`
		} `json:"rateCards"`
	}
	require.NoError(t, json.Unmarshal(data, &serialized))
	assert.Equal(t, currencyx.Code("CREDITS"), serialized.Currency)
	require.Len(t, serialized.RateCards, 1)
	assert.Equal(t, currencyx.Code("CREDITS"), serialized.RateCards[0].RateCard.Currency)
	assert.NotContains(t, string(data), managedCurrency.ID)

	var decoded Addon
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Equal(t, currencyx.Code("CREDITS"), decoded.Currency.Code)
	assert.False(t, decoded.Currency.IsResolved())
	require.Len(t, decoded.RateCards, 1)
	decodedRateCardCurrency := decoded.RateCards[0].AsMeta().Currency
	require.NotNil(t, decodedRateCardCurrency)
	assert.Equal(t, currencyx.Code("CREDITS"), decodedRateCardCurrency.Code)
	assert.False(t, decodedRateCardCurrency.IsResolved())
}
