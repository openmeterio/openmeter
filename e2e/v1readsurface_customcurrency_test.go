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

const (
	currencyNotRepresentableCode         = "currency_not_representable"
	rateCardCurrencyNotRepresentableCode = "rate_card_currency_not_representable"
)

func TestV1AuthoringRejectsCustomDefaultCurrencies(t *testing.T) {
	v3 := newV3Client(t)
	v1 := initClient(t)

	custom := createCustomCurrency(t, v3, uniqueCustomCurrencyCode("v1"), "USD")
	customCode := api.CurrencyCode(custom.Code)

	t.Run("plan", func(t *testing.T) {
		// given:
		// - otherwise identical v1 plan inputs using a custom currency and USD
		customInput := validV1PlanCreate(t, "v1_custom_currency_plan", customCode)
		fiatInput := validV1PlanCreate(t, "v1_fiat_currency_plan", "USD")

		// when:
		// - the plans are authored through v1
		customResponse, err := v1.CreatePlanWithResponse(t.Context(), customInput)
		require.NoError(t, err)
		fiatResponse, err := v1.CreatePlanWithResponse(t.Context(), fiatInput)
		require.NoError(t, err)

		// then:
		// - v1 rejects the out-of-contract custom code while retaining fiat authoring
		require.Equal(t, http.StatusBadRequest, customResponse.StatusCode(), "body: %s", string(customResponse.Body))
		require.Equal(t, http.StatusCreated, fiatResponse.StatusCode(), "body: %s", string(fiatResponse.Body))
		require.NotNil(t, fiatResponse.JSON201)
		assert.Equal(t, fiatInput.Key, fiatResponse.JSON201.Key)
	})

	t.Run("addon", func(t *testing.T) {
		// given:
		// - otherwise identical v1 add-on inputs using a custom currency and USD
		customInput := validV1AddonCreate(t, "v1_custom_currency_addon", customCode)
		fiatInput := validV1AddonCreate(t, "v1_fiat_currency_addon", "USD")

		// when:
		// - the add-ons are authored through v1
		customResponse, err := v1.CreateAddonWithResponse(t.Context(), customInput)
		require.NoError(t, err)
		fiatResponse, err := v1.CreateAddonWithResponse(t.Context(), fiatInput)
		require.NoError(t, err)

		// then:
		// - v1 rejects the out-of-contract custom code while retaining fiat authoring
		require.Equal(t, http.StatusBadRequest, customResponse.StatusCode(), "body: %s", string(customResponse.Body))
		require.Equal(t, http.StatusCreated, fiatResponse.StatusCode(), "body: %s", string(fiatResponse.Body))
		require.NotNil(t, fiatResponse.JSON201)
		assert.Equal(t, fiatInput.Key, fiatResponse.JSON201.Key)
	})

	t.Run("custom subscription", func(t *testing.T) {
		// given:
		// - a v1 customer and otherwise identical custom-plan inputs using a custom currency and USD
		customer := CreateCustomerWithSubject(
			t,
			v1,
			uniqueKey("v1_custom_currency_customer"),
			uniqueKey("v1_custom_currency_subject"),
		)
		require.NotNil(t, customer)

		timing := &api.SubscriptionTiming{}
		require.NoError(t, timing.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

		customInput := api.SubscriptionCreate{}
		require.NoError(t, customInput.FromCustomSubscriptionCreate(api.CustomSubscriptionCreate{
			Timing:     timing,
			CustomerId: lo.ToPtr(customer.Id),
			CustomPlan: validV1CustomPlanInput(t, "v1_custom_currency_subscription", customCode),
		}))
		fiatInput := api.SubscriptionCreate{}
		require.NoError(t, fiatInput.FromCustomSubscriptionCreate(api.CustomSubscriptionCreate{
			Timing:     timing,
			CustomerId: lo.ToPtr(customer.Id),
			CustomPlan: validV1CustomPlanInput(t, "v1_fiat_currency_subscription", "USD"),
		}))

		// when:
		// - the custom subscriptions are authored through v1
		customResponse, err := v1.CreateSubscriptionWithResponse(t.Context(), customInput)
		require.NoError(t, err)
		fiatResponse, err := v1.CreateSubscriptionWithResponse(t.Context(), fiatInput)
		require.NoError(t, err)

		// then:
		// - v1 rejects the custom currency before persistence while retaining fiat subscriptions
		require.Equal(t, http.StatusBadRequest, customResponse.StatusCode(), "body: %s", string(customResponse.Body))
		require.Equal(t, http.StatusCreated, fiatResponse.StatusCode(), "body: %s", string(fiatResponse.Body))
		require.NotNil(t, fiatResponse.JSON201)
	})
}

func TestV1ReadAndMutationBoundaryRejectsUnrepresentableCurrencies(t *testing.T) {
	tests := []struct {
		name         string
		keyPrefix    string
		expectedCode string
		configure    func(*v3sdk.CreatePlanRequest, *v3sdk.CreateAddonRequest, v3sdk.BillingCurrencyCode)
	}{
		{
			name:         "custom default currency",
			keyPrefix:    "default",
			expectedCode: currencyNotRepresentableCode,
			configure: func(plan *v3sdk.CreatePlanRequest, addon *v3sdk.CreateAddonRequest, customCode v3sdk.BillingCurrencyCode) {
				plan.Currency = customCode
				addon.Currency = customCode
			},
		},
		{
			name:         "custom rate-card override",
			keyPrefix:    "override",
			expectedCode: rateCardCurrencyNotRepresentableCode,
			configure: func(plan *v3sdk.CreatePlanRequest, addon *v3sdk.CreateAddonRequest, customCode v3sdk.BillingCurrencyCode) {
				plan.Phases[0].RateCards[0].Currency = &customCode
				addon.RateCards[0].Currency = &customCode
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testV1ReadAndMutationCurrencyBoundary(t, tt.keyPrefix, tt.expectedCode, tt.configure)
		})
	}
}

func testV1ReadAndMutationCurrencyBoundary(
	t *testing.T,
	keyPrefix string,
	expectedCode string,
	configure func(*v3sdk.CreatePlanRequest, *v3sdk.CreateAddonRequest, v3sdk.BillingCurrencyCode),
) {
	t.Helper()

	v3 := newV3Client(t)
	v1 := initClient(t)

	custom := createCustomCurrency(t, v3, uniqueCustomCurrencyCode("v1read"), "USD")
	customCode := v3sdk.BillingCurrencyCode(custom.Code)

	customPlanInput := validPlanRequest("v1_hidden_" + keyPrefix + "_currency_plan")
	customAddonInput := validAddonRequest("v1_hidden_" + keyPrefix + "_currency_addon")
	configure(&customPlanInput, &customAddonInput, customCode)

	customPlan, err := v3.Plans.Create(t.Context(), customPlanInput)
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customPlan)

	customAddon, err := v3.Addons.Create(t.Context(), customAddonInput)
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customAddon)
	customAddon, err = v3.Addons.Publish(t.Context(), customAddon.ID)
	v3.requireStatus(http.StatusOK, err)
	require.NotNil(t, customAddon)

	assignment, err := v3.PlanAddons.Create(
		t.Context(),
		customPlan.ID,
		validPlanAddonRequest(customPlan.Phases[0].Key, customAddon.ID),
	)
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, assignment)

	plainPlanInput := validPlanRequest("v1_visible_fiat_plan")
	plainPlan, err := v3.Plans.Create(t.Context(), plainPlanInput)
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plainPlan)
	plainPlan, err = v3.Plans.Publish(t.Context(), plainPlan.ID)
	v3.requireStatus(http.StatusOK, err)
	require.NotNil(t, plainPlan)

	plainAddonInput := validAddonRequest("v1_visible_fiat_addon")
	plainAddon, err := v3.Addons.Create(t.Context(), plainAddonInput)
	v3.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plainAddon)
	plainAddon, err = v3.Addons.Publish(t.Context(), plainAddon.ID)
	v3.requireStatus(http.StatusOK, err)
	require.NotNil(t, plainAddon)

	runRequired(t, "v1 plan-addon endpoints reject unrepresentable currencies", func(t *testing.T) {
		getResponse, err := v1.GetPlanAddonWithResponse(t.Context(), customPlan.ID, customAddon.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, getResponse.StatusCode(), "body: %s", string(getResponse.Body))
		assert.Contains(t, string(getResponse.Body), expectedCode)

		listResponse, err := v1.ListPlanAddonsWithResponse(t.Context(), customPlan.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, listResponse.StatusCode(), "body: %s", string(listResponse.Body))
		assert.Contains(t, string(listResponse.Body), expectedCode)

		updateResponse, err := v1.UpdatePlanAddonWithResponse(
			t.Context(),
			customPlan.ID,
			customAddon.ID,
			api.PlanAddonReplaceUpdate{FromPlanPhase: customPlan.Phases[0].Key},
		)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, updateResponse.StatusCode(), "body: %s", string(updateResponse.Body))
		assert.Contains(t, string(updateResponse.Body), expectedCode)

		createResponse, err := v1.CreatePlanAddonWithResponse(t.Context(), customPlan.ID, api.PlanAddonCreate{
			AddonId:       plainAddon.ID,
			FromPlanPhase: customPlan.Phases[0].Key,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, createResponse.StatusCode(), "body: %s", string(createResponse.Body))
		assert.Contains(t, string(createResponse.Body), expectedCode)

		page, err := v3.PlanAddons.List(t.Context(), customPlan.ID, v3sdk.PlanAddonListParams{})
		v3.requireStatus(http.StatusOK, err)
		require.NotNil(t, page)
		assert.Equal(t, 1, page.Meta.Page.Total, "rejected v1 create must not persist an assignment")
	})

	customPlan, err = v3.Plans.Publish(t.Context(), customPlan.ID)
	v3.requireStatus(http.StatusOK, err)
	require.NotNil(t, customPlan)

	runRequired(t, "v1 GET rejects unrepresentable currencies but retains fiat resources", func(t *testing.T) {
		customPlanResponse, err := v1.GetPlanWithResponse(t.Context(), customPlan.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, customPlanResponse.StatusCode(), "body: %s", string(customPlanResponse.Body))
		assert.Contains(t, string(customPlanResponse.Body), expectedCode)

		plainPlanResponse, err := v1.GetPlanWithResponse(t.Context(), plainPlan.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, plainPlanResponse.StatusCode(), "body: %s", string(plainPlanResponse.Body))
		require.NotNil(t, plainPlanResponse.JSON200)

		customAddonResponse, err := v1.GetAddonWithResponse(t.Context(), customAddon.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, customAddonResponse.StatusCode(), "body: %s", string(customAddonResponse.Body))
		assert.Contains(t, string(customAddonResponse.Body), expectedCode)

		plainAddonResponse, err := v1.GetAddonWithResponse(t.Context(), plainAddon.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, plainAddonResponse.StatusCode(), "body: %s", string(plainAddonResponse.Body))
		require.NotNil(t, plainAddonResponse.JSON200)
	})

	runRequired(t, "v1 LIST excludes unrepresentable currencies with exact totals", func(t *testing.T) {
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
	})

	runRequired(t, "v1 lifecycle mutations reject unrepresentable currencies without side effects", func(t *testing.T) {
		archivePlanResponse, err := v1.ArchivePlanWithResponse(t.Context(), customPlan.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, archivePlanResponse.StatusCode(), "body: %s", string(archivePlanResponse.Body))
		assert.Contains(t, string(archivePlanResponse.Body), expectedCode)

		publishPlanResponse, err := v1.PublishPlanWithResponse(t.Context(), customPlan.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, publishPlanResponse.StatusCode(), "body: %s", string(publishPlanResponse.Body))
		assert.Contains(t, string(publishPlanResponse.Body), expectedCode)

		nextPlanResponse, err := v1.NextPlanWithResponse(t.Context(), customPlan.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, nextPlanResponse.StatusCode(), "body: %s", string(nextPlanResponse.Body))
		assert.Contains(t, string(nextPlanResponse.Body), expectedCode)

		archiveAddonResponse, err := v1.ArchiveAddonWithResponse(t.Context(), customAddon.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, archiveAddonResponse.StatusCode(), "body: %s", string(archiveAddonResponse.Body))
		assert.Contains(t, string(archiveAddonResponse.Body), expectedCode)

		publishAddonResponse, err := v1.PublishAddonWithResponse(t.Context(), customAddon.ID)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, publishAddonResponse.StatusCode(), "body: %s", string(publishAddonResponse.Body))
		assert.Contains(t, string(publishAddonResponse.Body), expectedCode)

		planAfter, err := v3.Plans.Get(t.Context(), customPlan.ID)
		v3.requireStatus(http.StatusOK, err)
		require.NotNil(t, planAfter)
		assert.Equal(t, v3sdk.PlanStatusActive, planAfter.Status)

		planVersions, err := v3.Plans.List(t.Context(), v3sdk.PlanListParams{
			Page: &v3sdk.PageParams{Size: lo.ToPtr(1000)},
			Filter: &v3sdk.PlanFilter{
				Key: &v3sdk.StringFilter{Eq: lo.ToPtr(customPlan.Key)},
			},
		})
		v3.requireStatus(http.StatusOK, err)
		require.NotNil(t, planVersions)
		assert.Equal(t, 1, planVersions.Meta.Page.Total, "rejected v1 next must not create a plan version")

		addonAfter, err := v3.Addons.Get(t.Context(), customAddon.ID)
		v3.requireStatus(http.StatusOK, err)
		require.NotNil(t, addonAfter)
		assert.Equal(t, v3sdk.AddonStatusActive, addonAfter.Status)
	})
}

func validV1FlatRateCard(t *testing.T, keyPrefix string) api.RateCard {
	t.Helper()

	rateCard := api.RateCard{}
	require.NoError(t, rateCard.FromRateCardFlatFee(api.RateCardFlatFee{
		Key:  uniqueKey(keyPrefix),
		Name: "Test Rate Card " + keyPrefix,
		Price: &api.FlatPriceWithPaymentTerm{
			Amount:      "10",
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
		},
		BillingCadence: lo.ToPtr("P1M"),
		Type:           api.RateCardFlatFeeTypeFlatFee,
	}))

	return rateCard
}

func validV1PlanCreate(t *testing.T, keyPrefix string, currency api.CurrencyCode) api.PlanCreate {
	t.Helper()

	return api.PlanCreate{
		Key:            uniqueKey(keyPrefix),
		Name:           "Test Plan " + keyPrefix,
		Currency:       currency,
		BillingCadence: "P1M",
		Phases: []api.PlanPhase{{
			Key:       uniqueKey("phase"),
			Name:      "Test Phase",
			RateCards: []api.RateCard{validV1FlatRateCard(t, "plan_fee")},
		}},
	}
}

func validV1AddonCreate(t *testing.T, keyPrefix string, currency api.CurrencyCode) api.AddonCreate {
	t.Helper()

	return api.AddonCreate{
		Key:          uniqueKey(keyPrefix),
		Name:         "Test Addon " + keyPrefix,
		Currency:     currency,
		InstanceType: api.AddonInstanceTypeSingle,
		RateCards:    []api.RateCard{validV1FlatRateCard(t, "addon_fee")},
	}
}

func validV1CustomPlanInput(t *testing.T, keyPrefix string, currency api.CurrencyCode) api.CustomPlanInput {
	t.Helper()

	plan := validV1PlanCreate(t, keyPrefix, currency)

	return api.CustomPlanInput{
		Name:           plan.Name,
		Currency:       plan.Currency,
		BillingCadence: plan.BillingCadence,
		Phases:         plan.Phases,
	}
}
