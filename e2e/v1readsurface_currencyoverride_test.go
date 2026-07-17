package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// TestV1ProductCatalogListsExcludeCurrencyOverrides proves that v3-authored
// rate-card currency overrides do not break the v1 product-catalog list APIs.
// The v1 query layer must omit the unrepresentable resources before pagination
// and counting while leaving otherwise equivalent resources visible.
func TestV1ProductCatalogListsExcludeCurrencyOverrides(t *testing.T) {
	v3 := newV3Client(t)
	v1 := initClient(t)

	customCurrencyCode := uniqueKey("cc")
	customCurrency, err := v3.Currencies.CreateCustomCurrency(t.Context(), v3sdk.CreateCurrencyCustomRequest{
		Code:   customCurrencyCode,
		Name:   "E2E custom currency",
		Symbol: lo.ToPtr("CC"),
	})
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customCurrency)

	costBasis, err := v3.Currencies.CreateCostBasis(t.Context(), customCurrency.ID, v3sdk.CreateCostBasisRequest{
		FiatCode: "USD",
		Rate:     v3sdk.Numeric("1"),
	})
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, costBasis)

	currencyOverride := v3sdk.BillingCurrencyCode(customCurrencyCode)

	customPlanRequest := validPlanRequest("v1_hidden_currency_plan")
	customPlanRequest.Phases[0].RateCards[0].Currency = &currencyOverride
	customPlan, err := v3.Plans.Create(t.Context(), customPlanRequest)
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customPlan)
	customPlan, err = v3.Plans.Publish(t.Context(), customPlan.ID)
	v3.requireStatus(http.StatusOK, err)
	require.NotNil(t, customPlan)

	plainPlanRequest := validPlanRequest("v1_visible_currency_plan")
	plainPlan, err := v3.Plans.Create(t.Context(), plainPlanRequest)
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plainPlan)
	plainPlan, err = v3.Plans.Publish(t.Context(), plainPlan.ID)
	v3.requireStatus(http.StatusOK, err)
	require.NotNil(t, plainPlan)

	customAddonRequest := validAddonRequest("v1_hidden_currency_addon")
	customAddonRequest.RateCards[0].Currency = &currencyOverride
	customAddon, err := v3.Addons.Create(t.Context(), customAddonRequest)
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customAddon)
	customAddon, err = v3.Addons.Publish(t.Context(), customAddon.ID)
	v3.requireStatus(http.StatusOK, err)
	require.NotNil(t, customAddon)

	plainAddonRequest := validAddonRequest("v1_visible_currency_addon")
	plainAddon, err := v3.Addons.Create(t.Context(), plainAddonRequest)
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plainAddon)
	plainAddon, err = v3.Addons.Publish(t.Context(), plainAddon.ID)
	v3.requireStatus(http.StatusOK, err)
	require.NotNil(t, plainAddon)

	// then:
	// - v1 plan LIST succeeds, omits the currency-override plan, keeps the plain
	//   plan, and computes TotalCount after applying the compatibility filter.
	planResponse, err := v1.ListPlansWithResponse(t.Context(), &api.ListPlansParams{
		Key:      &[]string{customPlan.Key, plainPlan.Key},
		PageSize: lo.ToPtr(api.PaginationPageSize(1000)),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, planResponse.StatusCode(), "body: %s", string(planResponse.Body))
	require.NotNil(t, planResponse.JSON200)

	planKeys := lo.Map(planResponse.JSON200.Items, func(item api.Plan, _ int) string { return item.Key })
	assert.NotContains(t, planKeys, customPlan.Key)
	assert.Contains(t, planKeys, plainPlan.Key)
	assert.Equal(t, 1, planResponse.JSON200.TotalCount)

	// and:
	// - the equivalent v1 add-on LIST behavior uses the same compatibility
	//   boundary and reports an exact filtered count.
	addonResponse, err := v1.ListAddonsWithResponse(t.Context(), &api.ListAddonsParams{
		Key:      &[]string{customAddon.Key, plainAddon.Key},
		PageSize: lo.ToPtr(api.PaginationPageSize(1000)),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, addonResponse.StatusCode(), "body: %s", string(addonResponse.Body))
	require.NotNil(t, addonResponse.JSON200)

	addonKeys := lo.Map(addonResponse.JSON200.Items, func(item api.Addon, _ int) string { return item.Key })
	assert.NotContains(t, addonKeys, customAddon.Key)
	assert.Contains(t, addonKeys, plainAddon.Key)
	assert.Equal(t, 1, addonResponse.JSON200.TotalCount)
}
