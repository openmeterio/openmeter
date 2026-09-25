package customersentitlement

import (
	"testing"

	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	booleanentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/boolean"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	staticentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/static"
)

type unknownEntitlementValue struct{}

func (unknownEntitlementValue) HasAccess() bool { return true }

func TestMapEntitlementAccessToAPI(t *testing.T) {
	metered := entitlement.CustomerEntitlementAccess{
		FeatureKey: "tokens",
		Type:       entitlement.EntitlementTypeMetered,
		Value: &meteredentitlement.MeteredEntitlementValue{
			Balance:                   100.5,
			UsageInPeriod:             50,
			Overage:                   0,
			TotalAvailableGrantAmount: 150.5,
			GrantBalances:             map[string]float64{"grant-1": 100.5, "grant-2": 0},
		},
	}

	t.Run("metered without expand omits the value", func(t *testing.T) {
		result, err := mapEntitlementAccessToAPI(metered)
		require.NoError(t, err)
		require.Equal(t, api.BillingEntitlementValueResult{
			FeatureKey: "tokens",
			Type:       api.BillingEntitlementTypeMetered,
			HasAccess:  true,
		}, result)
	})

	t.Run("metered with the value expand maps balances as decimal strings", func(t *testing.T) {
		result, err := mapEntitlementAccessToAPI(metered, api.BillingEntitlementAccessExpandValue)
		require.NoError(t, err)
		require.Equal(t, api.BillingEntitlementTypeMetered, result.Type)
		require.True(t, result.HasAccess)
		require.Equal(t, &api.BillingEntitlementAccessValue{
			Balance:                   "100.5",
			Usage:                     "50",
			Overage:                   "0",
			TotalAvailableGrantAmount: "150.5",
			GrantBalances:             map[string]api.Numeric{"grant-1": "100.5", "grant-2": "0"},
		}, result.Value)
	})

	t.Run("metered with exhausted balance has no access", func(t *testing.T) {
		result, err := mapEntitlementAccessToAPI(entitlement.CustomerEntitlementAccess{
			FeatureKey: "tokens",
			Type:       entitlement.EntitlementTypeMetered,
			Value:      &meteredentitlement.MeteredEntitlementValue{Balance: 0, UsageInPeriod: 10, Overage: 2},
		}, api.BillingEntitlementAccessExpandValue)
		require.NoError(t, err)
		require.False(t, result.HasAccess)
		require.Equal(t, api.Numeric("2"), result.Value.Overage)
	})

	t.Run("static carries the config and ignores the value expand", func(t *testing.T) {
		config := `{"models":["gpt-5"]}`

		result, err := mapEntitlementAccessToAPI(entitlement.CustomerEntitlementAccess{
			FeatureKey: "models",
			Type:       entitlement.EntitlementTypeStatic,
			Value:      &staticentitlement.StaticEntitlementValue{Config: config},
		}, api.BillingEntitlementAccessExpandValue)
		require.NoError(t, err)
		require.Equal(t, api.BillingEntitlementValueResult{
			FeatureKey: "models",
			Type:       api.BillingEntitlementTypeStatic,
			HasAccess:  true,
			Config:     &config,
		}, result)
	})

	t.Run("boolean", func(t *testing.T) {
		result, err := mapEntitlementAccessToAPI(entitlement.CustomerEntitlementAccess{
			FeatureKey: "sso",
			Type:       entitlement.EntitlementTypeBoolean,
			Value:      &booleanentitlement.BooleanEntitlementValue{},
		})
		require.NoError(t, err)
		require.Equal(t, api.BillingEntitlementValueResult{
			FeatureKey: "sso",
			Type:       api.BillingEntitlementTypeBoolean,
			HasAccess:  true,
		}, result)
	})

	t.Run("inactive entitlement keeps its type", func(t *testing.T) {
		result, err := mapEntitlementAccessToAPI(entitlement.CustomerEntitlementAccess{
			FeatureKey: "tokens",
			Type:       entitlement.EntitlementTypeMetered,
			Value:      &entitlement.NoAccessValue{},
		}, api.BillingEntitlementAccessExpandValue)
		require.NoError(t, err)
		require.Equal(t, api.BillingEntitlementValueResult{
			FeatureKey: "tokens",
			Type:       api.BillingEntitlementTypeMetered,
			HasAccess:  false,
		}, result)
	})

	t.Run("unknown value type is rejected", func(t *testing.T) {
		_, err := mapEntitlementAccessToAPI(entitlement.CustomerEntitlementAccess{
			FeatureKey: "x",
			Value:      unknownEntitlementValue{},
		})
		require.Error(t, err)
	})
}

func TestMapEntitlementAccessCheckToAPI(t *testing.T) {
	t.Run("feature without an entitlement has no type", func(t *testing.T) {
		result, err := mapEntitlementAccessCheckToAPI(entitlement.CustomerEntitlementAccess{
			FeatureKey: "missing",
			Value:      &entitlement.NoAccessValue{},
		})
		require.NoError(t, err)
		require.Equal(t, api.BillingEntitlementAccessCheckResult{HasAccess: false}, result)
	})

	t.Run("inactive entitlement keeps its type", func(t *testing.T) {
		result, err := mapEntitlementAccessCheckToAPI(entitlement.CustomerEntitlementAccess{
			Type:  entitlement.EntitlementTypeMetered,
			Value: &entitlement.NoAccessValue{},
		})
		require.NoError(t, err)
		require.Equal(t, api.BillingEntitlementTypeMetered, *result.Type)
		require.False(t, result.HasAccess)
	})

	t.Run("static carries config", func(t *testing.T) {
		config := `{"models":["gpt-5"]}`
		result, err := mapEntitlementAccessCheckToAPI(entitlement.CustomerEntitlementAccess{
			Type:  entitlement.EntitlementTypeStatic,
			Value: &staticentitlement.StaticEntitlementValue{Config: config},
		})
		require.NoError(t, err)
		require.Equal(t, &config, result.Config)
		require.True(t, result.HasAccess)
	})

	t.Run("unknown value type is rejected", func(t *testing.T) {
		_, err := mapEntitlementAccessCheckToAPI(entitlement.CustomerEntitlementAccess{Value: unknownEntitlementValue{}})
		require.Error(t, err)
	})
}
