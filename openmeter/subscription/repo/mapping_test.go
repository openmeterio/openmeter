package repo_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	subscriptionrepo "github.com/openmeterio/openmeter/openmeter/subscription/repo"
)

func TestMapDBSubscriptionItemValidatesCustomCurrencyNamespace(t *testing.T) {
	const namespace = "subscription-item-namespace"

	currencyCode := "CREDITS"
	currencyID := "01K1CUSTOMCURRENCY00000000"
	item := &entdb.SubscriptionItem{
		ID:               "01K1SUBSCRIPTIONITEM000000",
		Namespace:        namespace,
		PhaseID:          "01K1SUBSCRIPTIONPHASE00000",
		Key:              "fee",
		Name:             "Fee",
		Currency:         &currencyCode,
		CustomCurrencyID: &currencyID,
		Edges: entdb.SubscriptionItemEdges{
			Phase: &entdb.SubscriptionPhase{
				ID:             "01K1SUBSCRIPTIONPHASE00000",
				SubscriptionID: "01K1SUBSCRIPTION0000000000",
			},
			CustomCurrency: &entdb.CustomCurrency{
				ID:                 currencyID,
				Namespace:          namespace,
				Code:               "CREDITS",
				Name:               "Credits",
				Symbol:             "CR",
				Precision:          2,
				DecimalMark:        ".",
				ThousandsSeparator: ",",
			},
		},
	}

	mapped, err := subscriptionrepo.MapDBSubscriptionItem(item)
	require.NoError(t, err)
	require.True(t, mapped.RateCard.AsMeta().Currency.IsResolved())

	item.Namespace = "other-namespace"

	_, err = subscriptionrepo.MapDBSubscriptionItem(item)
	require.ErrorContains(t, err, "invalid subscription item currency: namespace mismatch")
}
