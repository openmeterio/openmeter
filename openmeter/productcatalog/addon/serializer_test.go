package addon

import (
	"encoding/json"
	"testing"

	decimal "github.com/alpacahq/alpacadecimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestAddonSerializationUsesCurrencyCodes(t *testing.T) {
	// given:
	// - an add-on and rate card backed by a managed custom currency
	managedCurrency := &currencies.Currency{
		NamespacedID: models.NamespacedID{
			Namespace: "test",
			ID:        "currency-resource-id",
		},
		Code: "CREDITS",
		Name: "Credits",
	}
	addon := Addon{
		AddonMeta: productcatalog.AddonMeta{
			Currency: managedCurrency,
		},
		RateCards: RateCards{
			{
				RateCard: &productcatalog.FlatFeeRateCard{
					RateCardMeta: productcatalog.RateCardMeta{
						Currency: managedCurrency,
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
	// - only stable currency codes are serialized, and decoding restores code identities
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
	assert.IsType(t, currencyx.Code(""), decoded.Currency)
	assert.Equal(t, currencyx.Code("CREDITS"), decoded.Currency.GetCode())
	require.Len(t, decoded.RateCards, 1)
	assert.IsType(t, currencyx.Code(""), decoded.RateCards[0].AsMeta().Currency)
	assert.Equal(t, currencyx.Code("CREDITS"), decoded.RateCards[0].AsMeta().Currency.GetCode())
}
