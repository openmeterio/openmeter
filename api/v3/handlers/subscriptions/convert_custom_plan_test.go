package subscriptions

import (
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func TestFromAPIBillingSubscriptionCustomPlan(t *testing.T) {
	// given:
	// - an inline custom plan priced in a CUSTOM currency
	body := api.BillingSubscriptionCustomPlan{
		Name:             "Inline plan",
		Description:      lo.ToPtr("inline description"),
		Currency:         "CREDITS",
		BillingCadence:   "P1M",
		ProRatingEnabled: lo.ToPtr(true),
		Phases: []api.BillingPlanPhase{
			{
				Key:       "first",
				Name:      "First",
				Duration:  lo.ToPtr(api.ISO8601Duration("P1M")),
				RateCards: []api.BillingRateCard{},
			},
		},
	}

	// when:
	// - the inline plan is mapped to the domain plan create input
	got, err := FromAPIBillingSubscriptionCustomPlan("ns", body)

	// then:
	// - scalar fields map through and the custom currency is preserved verbatim,
	//   proving the converter uses currencyx.Code (fiat OR custom) rather than
	//   FiatCode, which would reject a custom code.
	require.NoError(t, err)
	assert.Equal(t, "ns", got.Namespace)
	assert.Equal(t, "Inline plan", got.Name)
	assert.Equal(t, "inline description", lo.FromPtr(got.Description))
	assert.Equal(t, currencyx.Code("CREDITS"), got.Currency.GetCode())
	assert.True(t, got.ProRatingConfig.Enabled)
	require.Len(t, got.Phases, 1)
	assert.Equal(t, "first", got.Phases[0].Key)
}
