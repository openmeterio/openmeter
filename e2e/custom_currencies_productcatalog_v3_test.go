package e2e

import (
	"net/http"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

type customCurrencyCatalogLifecycleInput struct {
	Plan  v3sdk.CreatePlanRequest
	Addon v3sdk.CreateAddonRequest
}

type customCurrencyCatalogLifecycleResult struct {
	Plan  *v3sdk.Plan
	Addon *v3sdk.Addon
}

// TestV3CustomCurrencyProductCatalogLifecycle proves custom currencies survive
// the full product-catalog authoring path: plan/add-on creation and publication,
// followed by assignment.
func TestV3CustomCurrencyProductCatalogLifecycle(t *testing.T) {
	t.Run("fiat plan and addon with custom-currency rate cards", func(t *testing.T) {
		// given:
		// - a managed custom currency with an active USD cost basis
		// - a USD plan and add-on containing both inherited USD and custom-priced rate cards
		c := newV3Client(t)
		custom := createCustomCurrency(t, c, uniqueCustomCurrencyCode("mix"), "USD")
		customCode := v3sdk.BillingCurrencyCode(custom.Code)

		planBody := validPlanRequest("custom_currency_mixed_plan")
		planBaseRateCard := validFlatRateCard("plan_usd")
		planCustomRateCard := validFlatRateCard("plan_custom")
		planCustomRateCard.Currency = &customCode
		planBody.Phases[0].RateCards = []v3sdk.RateCardInput{planBaseRateCard, planCustomRateCard}

		addonBody := validAddonRequest("custom_currency_mixed_addon")
		addonBaseRateCard := validFlatRateCard("addon_usd")
		addonCustomRateCard := validFlatRateCard("addon_custom")
		addonCustomRateCard.Currency = &customCode
		addonBody.RateCards = []v3sdk.RateCardInput{addonBaseRateCard, addonCustomRateCard}

		// when:
		// - both resources are published and assigned
		result := createCustomCurrencyCatalogResources(t, c, customCurrencyCatalogLifecycleInput{
			Plan:  planBody,
			Addon: addonBody,
		})

		// then:
		// - the default fiat and explicit custom overrides are retained
		assert.Equal(t, v3sdk.BillingCurrencyCode("USD"), result.Plan.Currency)
		assert.Equal(t, &customCode, findRateCardByKey(t, result.Plan, planCustomRateCard.Key).Currency)
		assert.Equal(t, v3sdk.BillingCurrencyCode("USD"), result.Addon.Currency)
		require.Len(t, result.Addon.RateCards, 2)
		var returnedAddonCustomRateCard *v3sdk.RateCard
		for i := range result.Addon.RateCards {
			if result.Addon.RateCards[i].Key == addonCustomRateCard.Key {
				returnedAddonCustomRateCard = &result.Addon.RateCards[i]
				break
			}
		}
		require.NotNil(t, returnedAddonCustomRateCard)
		assert.Equal(t, &customCode, returnedAddonCustomRateCard.Currency)
	})

	t.Run("plan and addon use the same custom currency", func(t *testing.T) {
		// given:
		// - a plan and add-on whose default currency is the same managed custom currency
		c := newV3Client(t)
		custom := createCustomCurrency(t, c, uniqueCustomCurrencyCode("all"), "USD")
		customCode := v3sdk.BillingCurrencyCode(custom.Code)

		planBody := validPlanRequest("custom_currency_plan")
		planBody.Currency = customCode
		planBody.Phases[0].RateCards = []v3sdk.RateCardInput{validFlatRateCard("plan_custom")}

		addonBody := validAddonRequest("custom_currency_addon")
		addonBody.Currency = customCode
		addonBody.RateCards = []v3sdk.RateCardInput{validFlatRateCard("addon_custom")}

		// when:
		// - both resources are published and assigned
		result := createCustomCurrencyCatalogResources(t, c, customCurrencyCatalogLifecycleInput{
			Plan:  planBody,
			Addon: addonBody,
		})

		// then:
		// - both resources inherit the custom default without redundant overrides
		assert.Equal(t, customCode, result.Plan.Currency)
		require.Len(t, result.Plan.Phases, 1)
		require.Len(t, result.Plan.Phases[0].RateCards, 1)
		assert.Nil(t, result.Plan.Phases[0].RateCards[0].Currency)
		assert.Equal(t, customCode, result.Addon.Currency)
		require.Len(t, result.Addon.RateCards, 1)
		assert.Nil(t, result.Addon.RateCards[0].Currency)
	})
}

func TestV3CustomCurrencyProductCatalogValidation(t *testing.T) {
	t.Run("fiat plan rejects custom override without matching cost basis", func(t *testing.T) {
		// given:
		// - a USD plan with a custom-priced rate card whose currency has no USD cost basis
		c := newV3Client(t)
		custom := createCustomCurrency(t, c, uniqueCustomCurrencyCode("pcb"))
		customCode := v3sdk.BillingCurrencyCode(custom.Code)

		body := validPlanRequest("custom_currency_missing_plan_cost_basis")
		rateCard := validFlatRateCard("custom_without_cost_basis")
		rateCard.Currency = &customCode
		body.Phases[0].RateCards = []v3sdk.RateCardInput{rateCard}

		plan, err := c.Plans.Create(t.Context(), body)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		assert.Equal(t, v3sdk.PlanStatusDraft, plan.Status)

		// when:
		// - the invalid draft is published
		_, err = c.Plans.Publish(t.Context(), plan.ID)

		// then:
		// - publication is blocked by the missing active cost basis
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "currency_cost_basis_not_found")
	})

	t.Run("addon publishes without cost basis but cannot be assigned to fiat plan", func(t *testing.T) {
		// given:
		// - a published USD add-on with a custom-priced rate card that has no USD cost basis
		// - a compatible draft USD plan
		c := newV3Client(t)
		custom := createCustomCurrency(t, c, uniqueCustomCurrencyCode("acb"))
		customCode := v3sdk.BillingCurrencyCode(custom.Code)

		addonBody := validAddonRequest("custom_currency_missing_addon_cost_basis")
		rateCard := validFlatRateCard("custom_without_cost_basis")
		rateCard.Currency = &customCode
		addonBody.RateCards = []v3sdk.RateCardInput{rateCard}

		addon, err := c.Addons.Create(t.Context(), addonBody)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, addon)

		addon, err = c.Addons.Publish(t.Context(), addon.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, addon)
		assert.Equal(t, v3sdk.AddonStatusActive, addon.Status)

		plan, err := c.Plans.Create(t.Context(), validPlanRequest("custom_currency_addon_cost_basis_plan"))
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		require.NotEmpty(t, plan.Phases)

		// when:
		// - the standalone add-on is assigned to the plan
		_, err = c.PlanAddons.Create(t.Context(), plan.ID, validPlanAddonRequest(plan.Phases[0].Key, addon.ID))

		// then:
		// - assignment enforces the plan-specific USD cost-basis requirement
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "currency_cost_basis_not_found")
	})

	t.Run("custom addon rejects a different custom-currency override", func(t *testing.T) {
		// given:
		// - an add-on with one custom default and a rate card overriding it with another
		c := newV3Client(t)
		defaultCurrency := createCustomCurrency(t, c, uniqueCustomCurrencyCode("adb"), "USD")
		overrideCurrency := createCustomCurrency(t, c, uniqueCustomCurrencyCode("ado"), "USD")
		overrideCode := v3sdk.BillingCurrencyCode(overrideCurrency.Code)

		body := validAddonRequest("custom_currency_invalid_addon_override")
		body.Currency = v3sdk.BillingCurrencyCode(defaultCurrency.Code)
		rateCard := validFlatRateCard("invalid_override")
		rateCard.Currency = &overrideCode
		body.RateCards = []v3sdk.RateCardInput{rateCard}

		// when:
		// - the invalid add-on draft is created
		addon, err := c.Addons.Create(t.Context(), body)

		// then:
		// - the draft surfaces the specific override issue
		// - publishing is blocked by the same issue
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, addon)

		var codes []string
		for _, validationError := range addon.ValidationErrors {
			codes = append(codes, validationError.Code)
		}
		assert.Contains(t, codes, "rate_card_currency_override_not_allowed")

		_, err = c.Addons.Publish(t.Context(), addon.ID)
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "rate_card_currency_override_not_allowed")
	})

	t.Run("custom addon default can be attached to fiat subscription", func(t *testing.T) {
		// given:
		// - a valid USD plan and an assigned add-on whose priced rate card inherits
		//   a managed custom currency with an active USD cost basis
		c := newV3Client(t)
		custom := createCustomCurrency(t, c, uniqueCustomCurrencyCode("sai"), "USD")
		customCode := v3sdk.BillingCurrencyCode(custom.Code)

		addonBody := validAddonRequest("custom_currency_subscription_addon")
		addonBody.Currency = customCode
		addonBody.RateCards = []v3sdk.RateCardInput{validFlatRateCard("inherited_custom_currency")}

		resources := createCustomCurrencyCatalogResources(t, c, customCurrencyCatalogLifecycleInput{
			Plan:  validPlanRequest("custom_currency_subscription_addon_plan"),
			Addon: addonBody,
		})

		customerKey := uniqueKey("custom_currency_subscription_addon_customer")
		customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
			Key:      customerKey,
			Name:     "Custom Currency Subscription Add-on Customer",
			Currency: lo.ToPtr("USD"),
			UsageAttribution: &v3sdk.CustomerUsageAttribution{
				SubjectKeys: []string{customerKey},
			},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, customer)

		subscription, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
			Customer: v3sdk.SubscriptionChangeCustomer{ID: &customer.ID},
			Plan:     v3sdk.SubscriptionChangePlan{ID: &resources.Plan.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, subscription)

		// when:
		// - the custom-priced add-on is attached to the fiat subscription
		timing := lo.Must(v3sdk.SubscriptionEditTimingFromEnum(v3sdk.SubscriptionEditTimingEnumImmediate))
		subscriptionAddon, err := c.Subscriptions.CreateAddon(t.Context(), subscription.ID, v3sdk.CreateSubscriptionAddonRequest{
			Addon:    v3sdk.AddonReference{ID: resources.Addon.ID},
			Quantity: 1,
			Timing:   timing,
		})

		// then:
		// - the compatible custom currency is accepted and the attachment is persisted
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, subscriptionAddon)
		assert.Equal(t, resources.Addon.ID, subscriptionAddon.Addon.ID)
		assert.EqualValues(t, 1, subscriptionAddon.Quantity)

		page, err := c.Subscriptions.ListAddons(t.Context(), subscription.ID, v3sdk.SubscriptionAddonListParams{
			Page: &v3sdk.PageParams{Size: lo.ToPtr(100)},
		})
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, page)
		require.Len(t, page.Data, 1)
		assert.Equal(t, subscriptionAddon.ID, page.Data[0].ID)
	})

	t.Run("plan and addon reject different custom default currencies", func(t *testing.T) {
		// given:
		// - a custom-currency plan and a published add-on using a different custom default
		c := newV3Client(t)
		planCurrency := createCustomCurrency(t, c, uniqueCustomCurrencyCode("pcm"), "USD")
		addonCurrency := createCustomCurrency(t, c, uniqueCustomCurrencyCode("acm"), "USD")

		planBody := validPlanRequest("custom_currency_mismatched_plan")
		planBody.Currency = v3sdk.BillingCurrencyCode(planCurrency.Code)
		plan, err := c.Plans.Create(t.Context(), planBody)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		require.NotEmpty(t, plan.Phases)

		addonBody := validAddonRequest("custom_currency_mismatched_addon")
		addonBody.Currency = v3sdk.BillingCurrencyCode(addonCurrency.Code)
		addon, err := c.Addons.Create(t.Context(), addonBody)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, addon)

		addon, err = c.Addons.Publish(t.Context(), addon.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, addon)

		// when:
		// - the add-on is assigned to the plan
		_, err = c.PlanAddons.Create(t.Context(), plan.ID, validPlanAddonRequest(plan.Phases[0].Key, addon.ID))

		// then:
		// - an add-on cannot introduce a second custom currency into a custom plan
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "rate_card_currency_override_not_allowed")
	})

	t.Run("fiat plan and addon reject different fiat currencies", func(t *testing.T) {
		// given:
		// - a draft USD plan and a published EUR add-on
		c := newV3Client(t)
		plan, err := c.Plans.Create(t.Context(), validPlanRequest("fiat_currency_mismatched_plan"))
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, plan)
		require.NotEmpty(t, plan.Phases)

		addonBody := validAddonRequest("fiat_currency_mismatched_addon")
		addonBody.Currency = "EUR"
		addon, err := c.Addons.Create(t.Context(), addonBody)
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, addon)

		addon, err = c.Addons.Publish(t.Context(), addon.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, addon)

		// when:
		// - the add-on is assigned to the plan
		_, err = c.PlanAddons.Create(t.Context(), plan.ID, validPlanAddonRequest(plan.Phases[0].Key, addon.ID))

		// then:
		// - the existing single-fiat rule remains enforced
		problem := requireProblem(t, err, http.StatusBadRequest)
		assertValidationCode(t, problem, "plan_multiple_fiat_currencies")
	})
}

func createCustomCurrency(t *testing.T, c *v3Client, code string, fiatCodes ...string) *v3sdk.CurrencyCustom {
	t.Helper()

	currency, err := c.Currencies.CreateCustomCurrency(t.Context(), v3sdk.CreateCurrencyCustomRequest{
		Name:              "Test Custom Currency " + code,
		Symbol:            lo.ToPtr("¤"),
		Precision:         2,
		DecimalMark:       ".",
		ThousandSeparator: ",",
		Code:              code,
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, currency)
	require.NotEmpty(t, currency.ID)
	assert.Equal(t, code, currency.Code)

	for _, fiatCode := range fiatCodes {
		costBasis, err := c.Currencies.CreateCostBasis(t.Context(), currency.ID, v3sdk.CreateCostBasisRequest{
			FiatCode: fiatCode,
			Rate:     "0.01",
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, costBasis)
		assert.Equal(t, fiatCode, costBasis.FiatCode)
	}

	return currency
}

func uniqueCustomCurrencyCode(prefix string) string {
	return strings.ReplaceAll(uniqueKey(prefix), "_", "")
}

func createCustomCurrencyCatalogResources(
	t *testing.T,
	c *v3Client,
	input customCurrencyCatalogLifecycleInput,
) customCurrencyCatalogLifecycleResult {
	t.Helper()

	plan, err := c.Plans.Create(t.Context(), input.Plan)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)
	require.Equal(t, v3sdk.PlanStatusDraft, plan.Status)
	require.NotEmpty(t, plan.Phases)

	addon, err := c.Addons.Create(t.Context(), input.Addon)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, addon)
	require.Equal(t, v3sdk.AddonStatusDraft, addon.Status)

	addon, err = c.Addons.Publish(t.Context(), addon.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, addon)
	require.Equal(t, v3sdk.AddonStatusActive, addon.Status)

	assignment, err := c.PlanAddons.Create(t.Context(), plan.ID, validPlanAddonRequest(plan.Phases[0].Key, addon.ID))
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, assignment)
	assert.Equal(t, addon.ID, assignment.Addon.ID)
	assert.Equal(t, plan.Phases[0].Key, assignment.FromPlanPhase)

	plan, err = c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, plan)
	require.Equal(t, v3sdk.PlanStatusActive, plan.Status)

	return customCurrencyCatalogLifecycleResult{
		Plan:  plan,
		Addon: addon,
	}
}
