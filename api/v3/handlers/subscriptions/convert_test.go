package subscriptions

import (
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestToAPIBillingSubscriptionCurrencySemantics(t *testing.T) {
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
	// - the subscription is mapped to the v3 API
	result := ToAPIBillingSubscription(sub)

	// then:
	// - the invoice currency, mode, and pin resource identities are preserved
	require.Equal(t, "USD", result.InvoiceCurrency)
	require.Equal(t, "pinned", string(result.CostBasisMode))
	require.Equal(t, "currency_credits", result.CostBasisPins[0].CustomCurrencyId)
	require.Equal(t, "USD", result.CostBasisPins[0].InvoiceCurrency)
	require.Equal(t, "cost_basis_credits_usd", result.CostBasisPins[0].CostBasisId)
}

func TestFromAPIBillingSubscriptionCreateCostBasisMode(t *testing.T) {
	// given:
	// - a v3 create request explicitly selecting pinned cost bases
	mode := api.BillingSubscriptionCostBasisMode("pinned")
	request := api.BillingSubscriptionCreate{CostBasisMode: &mode}

	// when:
	// - the request is mapped to the subscription workflow input
	result, err := FromAPIBillingSubscriptionCreate(
		"default",
		customer.CustomerID{ID: "customer_1", Namespace: "default"},
		"Subscription",
		request,
	)

	// then:
	// - the mode survives the API boundary
	require.NoError(t, err)
	require.Equal(t, subscription.CostBasisModePinned, result.CostBasisMode)
}
