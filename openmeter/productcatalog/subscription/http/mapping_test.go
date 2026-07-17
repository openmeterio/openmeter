package httpdriver

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestMapSubscriptionToAPICurrencySemantics(t *testing.T) {
	// given:
	// - a pinned subscription with a materialized invoice fiat and cost-basis pin
	sub := subscription.Subscription{
		InvoiceCurrency: currencyx.Code("USD"),
		CostBasisMode:   subscription.CostBasisModePinned,
		CostBasisPins: []subscription.CostBasisPin{
			{
				CustomCurrencyID: "currency_credits",
				InvoiceCurrency:  currencyx.Code("USD"),
				CostBasis: currencies.CostBasis{
					NamespacedID: models.NamespacedID{ID: "cost_basis_credits_usd"},
				},
			},
		},
	}

	// when:
	// - the subscription is mapped to the v1 API
	result := MapSubscriptionToAPI(sub)

	// then:
	// - the existing currency field carries invoice fiat and the new pin fields are exposed
	require.Equal(t, "USD", result.Currency)
	require.Equal(t, "pinned", string(result.CostBasisMode))
	require.Equal(t, "currency_credits", result.CostBasisPins[0].CustomCurrencyId)
	require.Equal(t, "USD", result.CostBasisPins[0].InvoiceCurrency)
	require.Equal(t, "cost_basis_credits_usd", result.CostBasisPins[0].CostBasisId)
}

func TestMapSubscriptionItemToAPIMaterializedCurrency(t *testing.T) {
	// given:
	// - a subscription item carrying a managed custom currency on its rate card
	customCurrency := currencies.Currency{
		NamespacedID: models.NamespacedID{ID: "currency_credits"},
		Code:         "CREDITS",
		Name:         "Credits",
	}
	item := subscription.SubscriptionItemView{
		SubscriptionItem: subscription.SubscriptionItem{
			RateCard: &productcatalog.FlatFeeRateCard{
				RateCardMeta: productcatalog.RateCardMeta{
					Currency: customCurrency,
				},
			},
		},
	}

	// when:
	// - the item is mapped to the expanded v1 subscription response
	result, err := MapSubscriptionItemToAPI(item)

	// then:
	// - the materialized currency code is exposed
	require.NoError(t, err)
	require.Equal(t, "CREDITS", *result.Currency)
}
