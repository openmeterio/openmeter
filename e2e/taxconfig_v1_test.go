package e2e

import (
	"encoding/json"
	"fmt"
	"maps"
	"math/rand/v2"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// These tests pin the v1 taxConfig response shape (behavior, stripe.code, taxCodeId) that v1
// clients see today, so the tax code refactor can't change it silently. Don't edit the
// expectations to make a refactor pass: a diff here is a v1 contract change.

// v1TaxConfigScenario is one legacy taxConfig input a v1 client can send, with the taxConfig
// the catalog resources (plans, addons, subscription items) return for it.
type v1TaxConfigScenario struct {
	key   string
	input *api.TaxConfig
	want  *api.TaxConfig
}

// v1TaxConfigCatalogScenarios covers the inputs v1 accepts. taxCode must have a Stripe app
// mapping: the response renders stripe.code from the tax code's mapping, not from the input.
func v1TaxConfigCatalogScenarios(taxCode v3sdk.TaxCode) []v1TaxConfigScenario {
	stripeCode := stripeAppMapping(taxCode)

	resolved := &api.TaxConfig{
		Stripe:    &api.StripeTaxConfig{Code: stripeCode},
		TaxCodeId: lo.ToPtr(taxCode.ID),
	}

	return []v1TaxConfigScenario{
		{
			key:   "stripe_code_only",
			input: &api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: stripeCode}},
			want:  resolved,
		},
		{
			key:   "tax_code_id_only",
			input: &api.TaxConfig{TaxCodeId: lo.ToPtr(taxCode.ID)},
			want:  resolved,
		},
		{
			// The ID wins, and the client's stripe.code is replaced by the tax code's mapping.
			key: "tax_code_id_wins_over_stripe_code",
			input: &api.TaxConfig{
				Stripe:    &api.StripeTaxConfig{Code: randomStripeTaxCode()},
				TaxCodeId: lo.ToPtr(taxCode.ID),
			},
			want: resolved,
		},
		{
			key:   "behavior_only",
			input: &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)},
			want:  &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)},
		},
		{
			key: "none",
		},
	}
}

// v1TaxConfigInvoiceLineWant is what invoice lines return per scenario. Unlike the catalog
// resources, a line without a tax code gets the organization's default invoicing tax code,
// which has no Stripe mapping.
func v1TaxConfigInvoiceLineWant(scenarios []v1TaxConfigScenario, defaultTaxCodeID string) map[string]*api.TaxConfig {
	want := make(map[string]*api.TaxConfig, len(scenarios))
	for _, scenario := range scenarios {
		want[scenario.key] = scenario.want
	}

	want["behavior_only"] = &api.TaxConfig{
		Behavior:  lo.ToPtr(api.TaxBehaviorExclusive),
		TaxCodeId: lo.ToPtr(defaultTaxCodeID),
	}
	want["none"] = &api.TaxConfig{TaxCodeId: lo.ToPtr(defaultTaxCodeID)}

	return want
}

// randomStripeTaxCode returns a Stripe code that is very unlikely to be mapped already, since
// all e2e tests share one namespace and a mapped Stripe code resolves to the existing tax code.
func randomStripeTaxCode() string {
	return fmt.Sprintf("txcd_%08d", rand.IntN(100_000_000))
}

func createStripeMappedTaxCode(t *testing.T, c *v3Client) v3sdk.TaxCode {
	t.Helper()

	taxCode, err := c.Tax.CreateCode(t.Context(), v3sdk.CreateTaxCodeRequest{
		Key:  uniqueKey("v1_taxconfig"),
		Name: "v1 taxConfig characterization",
		AppMappings: []v3sdk.TaxCodeAppMapping{{
			AppType: v3sdk.AppTypeStripe,
			TaxCode: randomStripeTaxCode(),
		}},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, taxCode)

	return *taxCode
}

func stripeAppMapping(taxCode v3sdk.TaxCode) string {
	mapping, _ := lo.Find(taxCode.AppMappings, func(m v3sdk.TaxCodeAppMapping) bool {
		return m.AppType == v3sdk.AppTypeStripe
	})

	return mapping.TaxCode
}

// defaultInvoicingTaxCodeID returns the organization's invoicing default tax code, which charges
// give lines that don't have a tax code.
func defaultInvoicingTaxCodeID(t *testing.T, c *v3Client) string {
	t.Helper()

	defaults, err := c.Defaults.GetOrganizationTaxCodes(t.Context())
	require.NoError(t, err)
	require.NotNil(t, defaults)

	return defaults.InvoicingTaxCode.ID
}

// createV1TaxConfigFeatures creates one metered feature per scenario, keyed by scenario, so
// every scenario can also be expressed as a usage based rate card.
func createV1TaxConfigFeatures(t *testing.T, client *api.ClientWithResponses, scenarios []v1TaxConfigScenario) map[string]string {
	t.Helper()

	meterSlug := uniqueKey("v1_taxconfig_meter")
	meterResp, err := client.CreateMeterWithResponse(t.Context(), api.MeterCreate{
		Slug:          meterSlug,
		Aggregation:   api.MeterAggregationSum,
		EventType:     meterSlug,
		ValueProperty: lo.ToPtr("$.value"),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, meterResp.StatusCode(), "create meter: %s", string(meterResp.Body))

	featureKeys := make(map[string]string, len(scenarios))
	for _, scenario := range scenarios {
		featureResp, err := client.CreateFeatureWithResponse(t.Context(), api.CreateFeatureJSONRequestBody{
			Key:       uniqueKey("usage_" + scenario.key),
			Name:      "v1 taxConfig " + scenario.key,
			MeterSlug: lo.ToPtr(meterSlug),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, featureResp.StatusCode(), "create feature: %s", string(featureResp.Body))
		require.NotNil(t, featureResp.JSON201)

		featureKeys[scenario.key] = featureResp.JSON201.Key
	}

	return featureKeys
}

// v1TaxConfigRateCards builds a flat fee rate card per scenario, keyed keyPrefix+scenario, and
// a usage based one per scenario for the features in featureKeys (keyed by the feature, as v1
// requires). It returns the rate cards and the taxConfig each one must come back with.
func v1TaxConfigRateCards(t *testing.T, keyPrefix string, scenarios []v1TaxConfigScenario, featureKeys map[string]string) ([]api.RateCard, map[string]*api.TaxConfig) {
	t.Helper()

	rateCards := make([]api.RateCard, 0, 2*len(scenarios))
	want := make(map[string]*api.TaxConfig, 2*len(scenarios))

	for _, scenario := range scenarios {
		key := keyPrefix + scenario.key

		flatFee := api.RateCard{}
		require.NoError(t, flatFee.FromRateCardFlatFee(api.RateCardFlatFee{
			Key:  key,
			Name: key,
			Type: api.RateCardFlatFeeTypeFlatFee,
			Price: &api.FlatPriceWithPaymentTerm{
				Amount:      "10",
				PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
				Type:        api.FlatPriceWithPaymentTermTypeFlat,
			},
			BillingCadence: lo.ToPtr("P1M"),
			TaxConfig:      scenario.input,
		}))

		rateCards = append(rateCards, flatFee)
		want[key] = scenario.want

		featureKey, ok := featureKeys[scenario.key]
		if !ok {
			continue
		}

		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
			Amount: "0.1",
			Type:   api.UnitPriceWithCommitmentsTypeUnit,
		}))

		usageBased := api.RateCard{}
		require.NoError(t, usageBased.FromRateCardUsageBased(api.RateCardUsageBased{
			Key:            featureKey,
			Name:           featureKey,
			Type:           api.RateCardUsageBasedTypeUsageBased,
			FeatureKey:     lo.ToPtr(featureKey),
			Price:          &price,
			BillingCadence: "P1M",
			TaxConfig:      scenario.input,
		}))

		rateCards = append(rateCards, usageBased)
		want[featureKey] = scenario.want
	}

	return rateCards, want
}

// v1RateCardTaxConfigs returns the taxConfig of each rate card by key.
func v1RateCardTaxConfigs(t *testing.T, rateCards []api.RateCard) map[string]*api.TaxConfig {
	t.Helper()

	got := make(map[string]*api.TaxConfig, len(rateCards))
	for _, rateCard := range rateCards {
		discriminator, err := rateCard.Discriminator()
		require.NoError(t, err)

		switch discriminator {
		case string(api.RateCardFlatFeeTypeFlatFee):
			flatFee, err := rateCard.AsRateCardFlatFee()
			require.NoError(t, err)

			got[flatFee.Key] = flatFee.TaxConfig
		case string(api.RateCardUsageBasedTypeUsageBased):
			usageBased, err := rateCard.AsRateCardUsageBased()
			require.NoError(t, err)

			got[usageBased.Key] = usageBased.TaxConfig
		default:
			require.Failf(t, "unexpected rate card type", "type %q", discriminator)
		}
	}

	return got
}

// assertV1TaxConfigs checks that every expected key is present in got with the expected
// taxConfig. Extra keys in got are ignored.
func assertV1TaxConfigs(t *testing.T, want, got map[string]*api.TaxConfig) {
	t.Helper()

	for key, wantTaxConfig := range want {
		gotTaxConfig, ok := got[key]
		if assert.True(t, ok, "missing %s", key) {
			assert.Equal(t, wantTaxConfig, gotTaxConfig, "taxConfig of %s", key)
		}
	}
}

func v1SubscriptionItemTaxConfigs(t *testing.T, client *api.ClientWithResponses, subscriptionID string) map[string]*api.TaxConfig {
	t.Helper()

	resp, err := client.GetSubscriptionWithResponse(t.Context(), subscriptionID, nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "get subscription: %s", string(resp.Body))
	require.NotNil(t, resp.JSON200)
	require.Len(t, resp.JSON200.Phases, 1)

	got := make(map[string]*api.TaxConfig, len(resp.JSON200.Phases[0].Items))
	for _, item := range resp.JSON200.Phases[0].Items {
		got[item.Key] = item.TaxConfig
	}

	return got
}

// v1InvoiceLineTaxConfigs returns the taxConfig of each line by name. It also checks that the
// line's rateCard.taxConfig and the taxConfig of its detailed lines match the line's own.
func v1InvoiceLineTaxConfigs(t *testing.T, lines []api.InvoiceLine) map[string]*api.TaxConfig {
	t.Helper()

	got := make(map[string]*api.TaxConfig, len(lines))
	for _, line := range lines {
		got[line.Name] = line.TaxConfig

		if assert.NotNil(t, line.RateCard, "rate card of line %s", line.Name) {
			assert.Equal(t, line.TaxConfig, line.RateCard.TaxConfig, "rateCard.taxConfig of line %s", line.Name)
		}

		for _, child := range lo.FromPtr(line.Children) {
			assert.Equal(t, line.TaxConfig, child.TaxConfig, "taxConfig of detailed line %s of %s", child.Name, line.Name)

			if assert.NotNil(t, child.RateCard, "rate card of detailed line %s of %s", child.Name, line.Name) {
				assert.Equal(t, line.TaxConfig, child.RateCard.TaxConfig, "rateCard.taxConfig of detailed line %s of %s", child.Name, line.Name)
			}
		}
	}

	return got
}

// v1BillingProfileCreateFromDefault returns a create request for a non-default profile that
// copies the default profile's apps, supplier and workflow.
func v1BillingProfileCreateFromDefault(t *testing.T, client *api.ClientWithResponses) api.BillingProfileCreate {
	t.Helper()

	expand := []api.BillingProfileExpand{api.BillingProfileExpandApps}
	resp, err := client.ListBillingProfilesWithResponse(t.Context(), &api.ListBillingProfilesParams{
		Expand:   &expand,
		PageSize: lo.ToPtr(api.PaginationPageSize(100)),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "list billing profiles: %s", string(resp.Body))
	require.NotNil(t, resp.JSON200)

	defaultProfile, found := lo.Find(resp.JSON200.Items, func(p api.BillingProfile) bool {
		return p.Default
	})
	require.True(t, found, "default billing profile is required")

	var appRefs struct {
		Invoicing struct {
			ID string `json:"id"`
		} `json:"invoicing"`
		Payment struct {
			ID string `json:"id"`
		} `json:"payment"`
		Tax struct {
			ID string `json:"id"`
		} `json:"tax"`
	}
	rawApps, err := json.Marshal(defaultProfile.Apps)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rawApps, &appRefs), "billing profile apps: %s", string(rawApps))

	invoicing := api.BillingWorkflowInvoicingSettings{}
	if defaultProfile.Workflow.Invoicing != nil {
		invoicing = *defaultProfile.Workflow.Invoicing
	}

	return api.BillingProfileCreate{
		Name: uniqueKey("v1_taxconfig_profile"),
		Apps: api.BillingProfileAppsCreate{
			Invoicing: appRefs.Invoicing.ID,
			Payment:   appRefs.Payment.ID,
			Tax:       appRefs.Tax.ID,
		},
		Supplier: defaultProfile.Supplier,
		Workflow: api.BillingWorkflowCreate{
			Collection: defaultProfile.Workflow.Collection,
			Invoicing:  &invoicing,
			Payment:    defaultProfile.Workflow.Payment,
			Tax:        defaultProfile.Workflow.Tax,
		},
	}
}

func createV1TaxConfigCustomer(t *testing.T, client *api.ClientWithResponses, keyPrefix string) api.Customer {
	t.Helper()

	resp, err := client.CreateCustomerWithResponse(t.Context(), api.CustomerCreate{
		Name:     "v1 taxConfig " + keyPrefix,
		Key:      lo.ToPtr(uniqueKey(keyPrefix)),
		Currency: lo.ToPtr(api.CurrencyCode("USD")),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode(), "create customer: %s", string(resp.Body))
	require.NotNil(t, resp.JSON201)

	return *resp.JSON201
}

func TestV1TaxConfigPlanRateCards(t *testing.T) {
	client := initClient(t)
	c := newV3Client(t)

	scenarios := v1TaxConfigCatalogScenarios(createStripeMappedTaxCode(t, c))
	featureKeys := createV1TaxConfigFeatures(t, client, scenarios)
	rateCards, want := v1TaxConfigRateCards(t, "", scenarios, featureKeys)

	planCreate := api.PlanCreate{
		Key:            uniqueKey("v1_taxconfig_plan"),
		Name:           "v1 taxConfig plan",
		Currency:       "USD",
		BillingCadence: "P1M",
		Phases: []api.PlanPhase{{
			Key:       "default",
			Name:      "Default",
			RateCards: rateCards,
		}},
	}

	createResp, err := client.CreatePlanWithResponse(t.Context(), planCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create plan: %s", string(createResp.Body))
	require.NotNil(t, createResp.JSON201)
	require.Len(t, createResp.JSON201.Phases, 1)

	plan := *createResp.JSON201

	t.Run("create", func(t *testing.T) {
		assertV1TaxConfigs(t, want, v1RateCardTaxConfigs(t, plan.Phases[0].RateCards))

		getResp, err := client.GetPlanWithResponse(t.Context(), plan.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get plan: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)
		require.Len(t, getResp.JSON200.Phases, 1)

		assertV1TaxConfigs(t, want, v1RateCardTaxConfigs(t, getResp.JSON200.Phases[0].RateCards))
	})

	t.Run("update", func(t *testing.T) {
		// given: the created plan or addon
		// when: it is updated with every scenario again, on new flat fee keys (so new rate cards
		// are written) and on the usage based cards, which are keyed by their feature
		// then: the update response and a re-read return the same tax configs as create
		updatedRateCards, updatedWant := v1TaxConfigRateCards(t, "updated_", scenarios, featureKeys)

		updateResp, err := client.UpdatePlanWithResponse(t.Context(), plan.Id, api.PlanReplaceUpdate{
			Name:           plan.Name,
			BillingCadence: plan.BillingCadence,
			Phases: []api.PlanPhase{{
				Key:       "default",
				Name:      "Default",
				RateCards: updatedRateCards,
			}},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, updateResp.StatusCode(), "update plan: %s", string(updateResp.Body))
		require.NotNil(t, updateResp.JSON200)
		require.Len(t, updateResp.JSON200.Phases, 1)

		assertV1TaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, updateResp.JSON200.Phases[0].RateCards))

		getResp, err := client.GetPlanWithResponse(t.Context(), plan.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get plan: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)
		require.Len(t, getResp.JSON200.Phases, 1)

		assertV1TaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, getResp.JSON200.Phases[0].RateCards))
	})
}

func TestV1TaxConfigAddonRateCards(t *testing.T) {
	client := initClient(t)
	c := newV3Client(t)

	scenarios := v1TaxConfigCatalogScenarios(createStripeMappedTaxCode(t, c))
	featureKeys := createV1TaxConfigFeatures(t, client, scenarios)
	rateCards, want := v1TaxConfigRateCards(t, "", scenarios, featureKeys)

	createResp, err := client.CreateAddonWithResponse(t.Context(), api.AddonCreate{
		Key:          uniqueKey("v1_taxconfig_addon"),
		Name:         "v1 taxConfig addon",
		Currency:     "USD",
		InstanceType: api.AddonInstanceTypeSingle,
		RateCards:    rateCards,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create addon: %s", string(createResp.Body))
	require.NotNil(t, createResp.JSON201)

	addon := *createResp.JSON201

	t.Run("create", func(t *testing.T) {
		assertV1TaxConfigs(t, want, v1RateCardTaxConfigs(t, addon.RateCards))

		getResp, err := client.GetAddonWithResponse(t.Context(), addon.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get addon: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		assertV1TaxConfigs(t, want, v1RateCardTaxConfigs(t, getResp.JSON200.RateCards))
	})

	t.Run("update", func(t *testing.T) {
		// given: the created plan or addon
		// when: it is updated with every scenario again, on new flat fee keys (so new rate cards
		// are written) and on the usage based cards, which are keyed by their feature
		// then: the update response and a re-read return the same tax configs as create
		updatedRateCards, updatedWant := v1TaxConfigRateCards(t, "updated_", scenarios, featureKeys)

		updateResp, err := client.UpdateAddonWithResponse(t.Context(), addon.Id, api.AddonReplaceUpdate{
			Name:         addon.Name,
			InstanceType: addon.InstanceType,
			RateCards:    updatedRateCards,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, updateResp.StatusCode(), "update addon: %s", string(updateResp.Body))
		require.NotNil(t, updateResp.JSON200)

		assertV1TaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, updateResp.JSON200.RateCards))

		getResp, err := client.GetAddonWithResponse(t.Context(), addon.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get addon: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		assertV1TaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, getResp.JSON200.RateCards))
	})
}

func TestV1TaxConfigSubscriptions(t *testing.T) {
	client := initClient(t)
	c := newV3Client(t)

	scenarios := v1TaxConfigCatalogScenarios(createStripeMappedTaxCode(t, c))
	featureKeys := createV1TaxConfigFeatures(t, client, scenarios)
	rateCards, want := v1TaxConfigRateCards(t, "", scenarios, featureKeys)

	timing := api.SubscriptionTiming{}
	require.NoError(t, timing.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

	// The addon is attached to the plan before publishing, so subscriptions can add it.
	addonRateCards, addonWant := v1TaxConfigRateCards(t, "addon_", scenarios, nil)

	addonResp, err := client.CreateAddonWithResponse(t.Context(), api.AddonCreate{
		Key:          uniqueKey("v1_taxconfig_sub_addon"),
		Name:         "v1 taxConfig subscription addon",
		Currency:     "USD",
		InstanceType: api.AddonInstanceTypeSingle,
		RateCards:    addonRateCards,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, addonResp.StatusCode(), "create addon: %s", string(addonResp.Body))
	require.NotNil(t, addonResp.JSON201)

	publishAddonResp, err := client.PublishAddonWithResponse(t.Context(), addonResp.JSON201.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, publishAddonResp.StatusCode(), "publish addon: %s", string(publishAddonResp.Body))

	planResp, err := client.CreatePlanWithResponse(t.Context(), api.PlanCreate{
		Key:            uniqueKey("v1_taxconfig_sub_plan"),
		Name:           "v1 taxConfig subscription plan",
		Currency:       "USD",
		BillingCadence: "P1M",
		SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
		Phases: []api.PlanPhase{{
			Key:       "default",
			Name:      "Default",
			RateCards: rateCards,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, planResp.StatusCode(), "create plan: %s", string(planResp.Body))
	require.NotNil(t, planResp.JSON201)

	planAddonResp, err := client.CreatePlanAddonWithResponse(t.Context(), planResp.JSON201.Id, api.PlanAddonCreate{
		AddonId:       addonResp.JSON201.Id,
		FromPlanPhase: "default",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, planAddonResp.StatusCode(), "attach addon to plan: %s", string(planAddonResp.Body))

	publishPlanResp, err := client.PublishPlanWithResponse(t.Context(), planResp.JSON201.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, publishPlanResp.StatusCode(), "publish plan: %s", string(publishPlanResp.Body))
	require.NotNil(t, publishPlanResp.JSON200)

	customer := createV1TaxConfigCustomer(t, client, "v1_taxconfig_sub_customer")

	subscriptionCreate := api.SubscriptionCreate{}
	require.NoError(t, subscriptionCreate.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
		Timing:     &timing,
		CustomerId: lo.ToPtr(customer.Id),
		Plan: api.PlanReferenceInput{
			Key:     publishPlanResp.JSON200.Key,
			Version: lo.ToPtr(publishPlanResp.JSON200.Version),
		},
	}))

	subscriptionResp, err := client.CreateSubscriptionWithResponse(t.Context(), subscriptionCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, subscriptionResp.StatusCode(), "create subscription: %s", string(subscriptionResp.Body))
	require.NotNil(t, subscriptionResp.JSON201)

	subscriptionID := subscriptionResp.JSON201.Id
	t.Logf("customer=%s subscription=%s", customer.Id, subscriptionID)

	t.Run("plan subscription items", func(t *testing.T) {
		assertV1TaxConfigs(t, want, v1SubscriptionItemTaxConfigs(t, client, subscriptionID))
	})

	t.Run("invoice lines generated from subscription items", func(t *testing.T) {
		// given: a subscription whose flat fee and usage based items carry every scenario
		// when: billing syncs the subscription into gathering invoice lines
		// then: each line has its item's tax config, with the default tax code filled in
		lineWant := v1TaxConfigInvoiceLineWant(scenarios, defaultInvoicingTaxCodeID(t, c))

		// Gathering lines are named after their rate card: the flat fee key, or the feature key
		// for usage based ones.
		subscriptionLineWant := make(map[string]*api.TaxConfig, 2*len(scenarios))
		for _, scenario := range scenarios {
			subscriptionLineWant[scenario.key] = lineWant[scenario.key]
			subscriptionLineWant[featureKeys[scenario.key]] = lineWant[scenario.key]
		}

		customers := api.InvoiceListParamsCustomers{customer.Id}
		expand := api.InvoiceListParamsExpand{api.InvoiceExpandLines}

		var got map[string]*api.TaxConfig
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			resp, err := client.ListInvoicesWithResponse(t.Context(), &api.ListInvoicesParams{
				Customers: &customers,
				Expand:    &expand,
				PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
			})
			require.NoError(collect, err)
			require.Equal(collect, http.StatusOK, resp.StatusCode(), "list invoices: %s", string(resp.Body))
			require.NotNil(collect, resp.JSON200)

			got = map[string]*api.TaxConfig{}
			for _, invoice := range resp.JSON200.Items {
				for _, line := range lo.FromPtr(invoice.Lines) {
					if line.Subscription != nil {
						got[line.Name] = line.TaxConfig
					}
				}
			}

			for key := range subscriptionLineWant {
				require.Contains(collect, got, key)
			}
		}, time.Minute, time.Second)

		assertV1TaxConfigs(t, subscriptionLineWant, got)
	})

	t.Run("edit adds items", func(t *testing.T) {
		// given: the plan subscription
		// when: an edit adds a flat fee item per scenario
		// then: the added items return the same tax configs as plan items
		addedRateCards, addedWant := v1TaxConfigRateCards(t, "added_", scenarios, nil)

		operations := lo.Map(addedRateCards, func(rateCard api.RateCard, _ int) api.SubscriptionEditOperation {
			operation := api.SubscriptionEditOperation{}
			require.NoError(t, operation.FromEditSubscriptionAddItem(api.EditSubscriptionAddItem{
				Op:       api.EditSubscriptionAddItemOpAddItem,
				PhaseKey: "default",
				RateCard: rateCard,
			}))

			return operation
		})

		editResp, err := client.EditSubscriptionWithResponse(t.Context(), subscriptionID, api.SubscriptionEdit{
			Customizations: operations,
			Timing:         &timing,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, editResp.StatusCode(), "edit subscription: %s", string(editResp.Body))
		require.NotNil(t, editResp.JSON200)

		assertV1TaxConfigs(t, addedWant, v1SubscriptionItemTaxConfigs(t, client, editResp.JSON200.Id))
	})

	t.Run("addon rate cards", func(t *testing.T) {
		// given: the plan subscription and an addon attached to its plan
		// when: the addon is added to the subscription
		// then: the subscription addon's rate cards and the items created from them return the
		// addon's tax configs
		createResp, err := client.CreateSubscriptionAddonWithResponse(t.Context(), subscriptionID, api.SubscriptionAddonCreate{
			Addon: struct {
				Id string `json:"id"`
			}{Id: addonResp.JSON201.Id},
			Name:     "v1 taxConfig subscription addon",
			Quantity: 1,
			Timing:   timing,
		})
		require.NoError(t, err)
		// The v1 subscription addon create answers 200 instead of the documented 201, so the
		// generated client doesn't decode the body.
		require.Equal(t, http.StatusOK, createResp.StatusCode(), "add addon to subscription: %s", string(createResp.Body))

		var subscriptionAddon api.SubscriptionAddon
		require.NoError(t, json.Unmarshal(createResp.Body, &subscriptionAddon))

		subscriptionAddonRateCards := func(rateCards []api.SubscriptionAddonRateCard) []api.RateCard {
			return lo.Map(rateCards, func(rc api.SubscriptionAddonRateCard, _ int) api.RateCard {
				return rc.RateCard
			})
		}

		assertV1TaxConfigs(t, addonWant, v1RateCardTaxConfigs(t, subscriptionAddonRateCards(subscriptionAddon.RateCards)))

		getResp, err := client.GetSubscriptionAddonWithResponse(t.Context(), subscriptionID, subscriptionAddon.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get subscription addon: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		assertV1TaxConfigs(t, addonWant, v1RateCardTaxConfigs(t, subscriptionAddonRateCards(getResp.JSON200.RateCards)))

		// The addon's rate cards become subscription items of their own, since no plan item
		// shares their keys.
		assertV1TaxConfigs(t, addonWant, v1SubscriptionItemTaxConfigs(t, client, subscriptionID))
	})

	t.Run("custom subscription items", func(t *testing.T) {
		// given: a new customer
		// when: it subscribes to a custom plan with every scenario
		// then: the subscription items return the same tax configs as plan items
		customRateCards, customWant := v1TaxConfigRateCards(t, "", scenarios, featureKeys)
		customCustomer := createV1TaxConfigCustomer(t, client, "v1_taxconfig_custom_sub_customer")

		create := api.SubscriptionCreate{}
		require.NoError(t, create.FromCustomSubscriptionCreate(api.CustomSubscriptionCreate{
			Timing:     &timing,
			CustomerId: lo.ToPtr(customCustomer.Id),
			CustomPlan: api.CustomPlanInput{
				Name:           "v1 taxConfig custom plan",
				Currency:       "USD",
				BillingCadence: "P1M",
				Phases: []api.PlanPhase{{
					Key:       "default",
					Name:      "Default",
					RateCards: customRateCards,
				}},
			},
		}))

		createResp, err := client.CreateSubscriptionWithResponse(t.Context(), create)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create custom subscription: %s", string(createResp.Body))
		require.NotNil(t, createResp.JSON201)

		assertV1TaxConfigs(t, customWant, v1SubscriptionItemTaxConfigs(t, client, createResp.JSON201.Id))
	})
}

// Billing profiles no longer accept a tax code in defaultTaxConfig: create rejects it, and update
// only lets clients echo or remove a stored one. The e2e stack has no profile with a stored tax
// code, so only rejection and behavior round-trips are reachable here; echo and removal are
// covered by test/billing/taxcode_dual_write_test.go.
func TestV1TaxConfigBillingProfileDefaultTaxConfig(t *testing.T) {
	client := initClient(t)
	c := newV3Client(t)

	taxCode := createStripeMappedTaxCode(t, c)

	profileCreate := func(defaultTaxConfig *api.TaxConfig) api.BillingProfileCreate {
		create := v1BillingProfileCreateFromDefault(t, client)
		create.Workflow.Invoicing.DefaultTaxConfig = defaultTaxConfig

		return create
	}

	const deprecationDetail = "defaultTaxConfig is deprecated"

	t.Run("create rejects a tax code", func(t *testing.T) {
		for name, taxConfig := range map[string]*api.TaxConfig{
			"stripe_code_only": {Stripe: &api.StripeTaxConfig{Code: stripeAppMapping(taxCode)}},
			"tax_code_id_only": {TaxCodeId: lo.ToPtr(taxCode.ID)},
		} {
			t.Run(name, func(t *testing.T) {
				resp, err := client.CreateBillingProfileWithResponse(t.Context(), profileCreate(taxConfig))
				require.NoError(t, err)
				require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "create billing profile: %s", string(resp.Body))
				assert.Contains(t, string(resp.Body), deprecationDetail)
			})
		}
	})

	t.Run("create without a default tax config", func(t *testing.T) {
		resp, err := client.CreateBillingProfileWithResponse(t.Context(), profileCreate(nil))
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "create billing profile: %s", string(resp.Body))
		require.NotNil(t, resp.JSON201)
		require.NotNil(t, resp.JSON201.Workflow.Invoicing)

		assert.Nil(t, resp.JSON201.Workflow.Invoicing.DefaultTaxConfig)
	})

	t.Run("behavior round-trips through create and update", func(t *testing.T) {
		// given: a profile created with an exclusive default behavior
		// when: it is read through a customer and updated
		// then: the behavior can be read, changed and removed, but a tax code can't be added
		createResp, err := client.CreateBillingProfileWithResponse(t.Context(), profileCreate(&api.TaxConfig{
			Behavior: lo.ToPtr(api.TaxBehaviorExclusive),
		}))
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create billing profile: %s", string(createResp.Body))
		require.NotNil(t, createResp.JSON201)
		require.NotNil(t, createResp.JSON201.Workflow.Invoicing)

		profile := *createResp.JSON201
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, profile.Workflow.Invoicing.DefaultTaxConfig)

		profileUpdate := func(defaultTaxConfig *api.TaxConfig) api.BillingProfileReplaceUpdateWithWorkflow {
			workflow := profile.Workflow
			invoicing := *workflow.Invoicing
			invoicing.DefaultTaxConfig = defaultTaxConfig
			workflow.Invoicing = &invoicing

			return api.BillingProfileReplaceUpdateWithWorkflow{
				Name:     profile.Name,
				Supplier: profile.Supplier,
				Workflow: workflow,
			}
		}

		t.Run("customer profile shows the default tax config", func(t *testing.T) {
			// given: a customer pinned to the profile
			// when: the customer's billing profile is read
			// then: it returns the profile's default tax config
			customer := createV1TaxConfigCustomer(t, client, "v1_taxconfig_profile_customer")

			overrideResp, err := client.UpsertBillingProfileCustomerOverrideWithResponse(t.Context(), customer.Id, api.BillingProfileCustomerOverrideCreate{
				BillingProfileId: lo.ToPtr(profile.Id),
			})
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, overrideResp.StatusCode(), "pin customer to billing profile: %s", string(overrideResp.Body))

			getResp, err := client.GetBillingProfileCustomerOverrideWithResponse(t.Context(), customer.Id, nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, getResp.StatusCode(), "get customer billing profile: %s", string(getResp.Body))
			require.NotNil(t, getResp.JSON200)
			require.NotNil(t, getResp.JSON200.CustomerProfile)
			require.NotNil(t, getResp.JSON200.CustomerProfile.Workflow.Invoicing)

			assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, getResp.JSON200.CustomerProfile.Workflow.Invoicing.DefaultTaxConfig)
		})

		t.Run("update rejects adding a tax code", func(t *testing.T) {
			for name, taxConfig := range map[string]*api.TaxConfig{
				"stripe_code_only": {
					Behavior: lo.ToPtr(api.TaxBehaviorExclusive),
					Stripe:   &api.StripeTaxConfig{Code: stripeAppMapping(taxCode)},
				},
				"tax_code_id_only": {
					Behavior:  lo.ToPtr(api.TaxBehaviorExclusive),
					TaxCodeId: lo.ToPtr(taxCode.ID),
				},
			} {
				t.Run(name, func(t *testing.T) {
					resp, err := client.UpdateBillingProfileWithResponse(t.Context(), profile.Id, profileUpdate(taxConfig))
					require.NoError(t, err)
					require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "update billing profile: %s", string(resp.Body))
					assert.Contains(t, string(resp.Body), deprecationDetail)
				})
			}
		})

		t.Run("update changes the behavior", func(t *testing.T) {
			resp, err := client.UpdateBillingProfileWithResponse(t.Context(), profile.Id, profileUpdate(&api.TaxConfig{
				Behavior: lo.ToPtr(api.TaxBehaviorInclusive),
			}))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode(), "update billing profile: %s", string(resp.Body))
			require.NotNil(t, resp.JSON200)
			require.NotNil(t, resp.JSON200.Workflow.Invoicing)

			assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorInclusive)}, resp.JSON200.Workflow.Invoicing.DefaultTaxConfig)
		})

		t.Run("update removes the default tax config", func(t *testing.T) {
			resp, err := client.UpdateBillingProfileWithResponse(t.Context(), profile.Id, profileUpdate(nil))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode(), "update billing profile: %s", string(resp.Body))
			require.NotNil(t, resp.JSON200)
			require.NotNil(t, resp.JSON200.Workflow.Invoicing)

			assert.Nil(t, resp.JSON200.Workflow.Invoicing.DefaultTaxConfig)
		})
	})
}

func TestV1TaxConfigInvoiceLines(t *testing.T) {
	client := initClient(t)
	c := newV3Client(t)

	taxCode := createStripeMappedTaxCode(t, c)
	scenarios := v1TaxConfigCatalogScenarios(taxCode)
	featureKeys := createV1TaxConfigFeatures(t, client, scenarios)
	lineWant := v1TaxConfigInvoiceLineWant(scenarios, defaultInvoicingTaxCodeID(t, c))

	// Without auto-advance the standard invoice stays in draft, so its lines can be edited. The
	// profile's default behavior is copied onto the invoice's workflow settings.
	profileCreate := v1BillingProfileCreateFromDefault(t, client)
	profileCreate.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)
	profileCreate.Workflow.Invoicing.DefaultTaxConfig = &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}

	profileResp, err := client.CreateBillingProfileWithResponse(t.Context(), profileCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, profileResp.StatusCode(), "create billing profile: %s", string(profileResp.Body))
	require.NotNil(t, profileResp.JSON201)

	customer := createV1TaxConfigCustomer(t, client, "v1_taxconfig_line_customer")

	overrideResp, err := client.UpsertBillingProfileCustomerOverrideWithResponse(t.Context(), customer.Id, api.BillingProfileCustomerOverrideCreate{
		BillingProfileId: lo.ToPtr(profileResp.JSON201.Id),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, overrideResp.StatusCode(), "pin customer to billing profile: %s", string(overrideResp.Body))

	periodEnd := time.Now().UTC().Truncate(time.Second)
	period := api.Period{From: periodEnd.Add(-24 * time.Hour), To: periodEnd}

	flatPrice := func() *api.RateCardUsageBasedPrice {
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromFlatPriceWithPaymentTerm(api.FlatPriceWithPaymentTerm{
			Amount:      "10",
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
		}))

		return &price
	}

	unitPrice := func() *api.RateCardUsageBasedPrice {
		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
			Amount: "0.1",
			Type:   api.UnitPriceWithCommitmentsTypeUnit,
		}))

		return &price
	}

	// v1 accepts a line's tax config in rateCard.taxConfig, or in the deprecated top-level
	// taxConfig next to a top-level price. lineInput holds either shape.
	type lineInput struct {
		name      string
		invoiceAt time.Time
		rateCard  *api.InvoiceUsageBasedRateCard
		price     *api.RateCardUsageBasedPrice
		taxConfig *api.TaxConfig
	}

	// lineInputs returns, per scenario, a flat fee line, a usage based line and a flat fee line
	// in the deprecated shape, named prefix+{flat,usage,legacy}_+scenario, with the taxConfig
	// each one returns on a gathering invoice.
	lineInputs := func(prefix string) ([]lineInput, map[string]*api.TaxConfig) {
		inputs := make([]lineInput, 0, 3*len(scenarios))
		want := make(map[string]*api.TaxConfig, 3*len(scenarios))

		for _, scenario := range scenarios {
			flat := lineInput{
				name:      prefix + "flat_" + scenario.key,
				invoiceAt: period.From,
				rateCard:  &api.InvoiceUsageBasedRateCard{Price: flatPrice(), TaxConfig: scenario.input},
			}
			// Usage is billed in arrears, at the end of the period.
			usage := lineInput{
				name:      prefix + "usage_" + scenario.key,
				invoiceAt: period.To,
				rateCard: &api.InvoiceUsageBasedRateCard{
					FeatureKey: lo.ToPtr(featureKeys[scenario.key]),
					Price:      unitPrice(),
					TaxConfig:  scenario.input,
				},
			}
			legacy := lineInput{
				name:      prefix + "legacy_" + scenario.key,
				invoiceAt: period.From,
				price:     flatPrice(),
				taxConfig: scenario.input,
			}

			for _, input := range []lineInput{flat, usage, legacy} {
				inputs = append(inputs, input)
				want[input.name] = lineWant[scenario.key]
			}
		}

		return inputs, want
	}

	pendingLines := func(inputs []lineInput) []api.InvoicePendingLineCreate {
		return lo.Map(inputs, func(in lineInput, _ int) api.InvoicePendingLineCreate {
			return api.InvoicePendingLineCreate{
				Name:      in.name,
				InvoiceAt: in.invoiceAt,
				Period:    period,
				RateCard:  in.rateCard,
				Price:     in.price,
				TaxConfig: in.taxConfig,
			}
		})
	}

	updateLines := func(inputs []lineInput) []api.InvoiceLineReplaceUpdate {
		return lo.Map(inputs, func(in lineInput, _ int) api.InvoiceLineReplaceUpdate {
			return api.InvoiceLineReplaceUpdate{
				Name:      in.name,
				InvoiceAt: in.invoiceAt,
				Period:    period,
				RateCard:  in.rateCard,
				Price:     in.price,
				TaxConfig: in.taxConfig,
			}
		})
	}

	// Standard invoices merge the invoice's defaultTaxConfig into their lines, so lines without
	// a behavior get the profile's. Gathering invoices don't.
	withProfileBehavior := func(want map[string]*api.TaxConfig) map[string]*api.TaxConfig {
		merged := make(map[string]*api.TaxConfig, len(want))
		for key, taxConfig := range want {
			mergedTaxConfig := lo.FromPtr(taxConfig)
			if mergedTaxConfig.Behavior == nil {
				mergedTaxConfig.Behavior = lo.ToPtr(api.TaxBehaviorExclusive)
			}

			merged[key] = &mergedTaxConfig
		}

		return merged
	}

	expand := []api.InvoiceExpand{api.InvoiceExpandLines}

	getInvoice := func(t *testing.T, invoiceID string) api.Invoice {
		t.Helper()

		resp, err := client.GetInvoiceWithResponse(t.Context(), invoiceID, &api.GetInvoiceParams{Expand: &expand})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "get invoice: %s", string(resp.Body))
		require.NotNil(t, resp.JSON200)
		require.NotNil(t, resp.JSON200.Lines)

		return *resp.JSON200
	}

	updateInvoice := func(t *testing.T, invoice api.Invoice, lines []api.InvoiceLineReplaceUpdate, workflow api.InvoiceWorkflowReplaceUpdate) api.Invoice {
		t.Helper()

		resp, err := client.UpdateInvoiceWithResponse(t.Context(), invoice.Id, api.InvoiceReplaceUpdate{
			Customer:    billingPartyReplaceUpdateFromInvoiceCustomer(invoice.Customer),
			Supplier:    billingPartyReplaceUpdateFromParty(invoice.Supplier),
			Description: invoice.Description,
			Metadata:    invoice.Metadata,
			Lines:       lines,
			Workflow:    workflow,
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "update invoice: %s", string(resp.Body))
		require.NotNil(t, resp.JSON200)
		require.NotNil(t, resp.JSON200.Lines)

		return *resp.JSON200
	}

	t.Run("create rejects different taxConfig and rateCard.taxConfig", func(t *testing.T) {
		resp, err := client.CreatePendingInvoiceLineWithResponse(t.Context(), customer.Id, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{{
				Name:      "mismatch",
				InvoiceAt: period.From,
				Period:    period,
				RateCard:  &api.InvoiceUsageBasedRateCard{Price: flatPrice(), TaxConfig: &api.TaxConfig{TaxCodeId: lo.ToPtr(taxCode.ID)}},
				TaxConfig: &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)},
			}},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "create pending line: %s", string(resp.Body))
		assert.Contains(t, string(resp.Body), "taxConfig must be equal to rateCard.taxConfig")
	})

	createdInputs, createdWant := lineInputs("")

	createResp, err := client.CreatePendingInvoiceLineWithResponse(t.Context(), customer.Id, api.InvoicePendingLineCreateInput{
		Currency: "USD",
		Lines:    pendingLines(createdInputs),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create pending lines: %s", string(createResp.Body))
	require.NotNil(t, createResp.JSON201)

	gatheringInvoiceID := createResp.JSON201.Invoice.Id

	t.Run("create response", func(t *testing.T) {
		assertV1TaxConfigs(t, createdWant, v1InvoiceLineTaxConfigs(t, createResp.JSON201.Lines))
	})

	t.Run("gathering invoice", func(t *testing.T) {
		assertV1TaxConfigs(t, createdWant, v1InvoiceLineTaxConfigs(t, *getInvoice(t, gatheringInvoiceID).Lines))
	})

	// Edits run at the top level, so every subtest only reads and can run on its own.
	gatheringEditedInputs, gatheringEditedWant := lineInputs("gathering_edited_")

	gatheringInvoice := getInvoice(t, gatheringInvoiceID)

	gatheringLines := lo.Map(*gatheringInvoice.Lines, func(line api.InvoiceLine, _ int) api.InvoiceLineReplaceUpdate {
		return invoiceLineReplaceUpdateFromLine(t, line)
	})
	gatheringLines = append(gatheringLines, updateLines(gatheringEditedInputs)...)

	gatheringEdited := updateInvoice(t, gatheringInvoice, gatheringLines, invoiceWorkflowReplaceUpdateFromInvoice(gatheringInvoice))

	t.Run("gathering invoice edit adds lines", func(t *testing.T) {
		// given: the gathering invoice with the created lines
		// when: the edit above echoes those lines and adds a line per scenario and shape
		// then: the edit response and a re-read return the created and added lines' tax configs
		got := v1InvoiceLineTaxConfigs(t, *gatheringEdited.Lines)
		assertV1TaxConfigs(t, createdWant, got)
		assertV1TaxConfigs(t, gatheringEditedWant, got)

		got = v1InvoiceLineTaxConfigs(t, *getInvoice(t, gatheringInvoiceID).Lines)
		assertV1TaxConfigs(t, createdWant, got)
		assertV1TaxConfigs(t, gatheringEditedWant, got)
	})

	invoice := getInvoice(t, invoicePendingLinesNow(t, client, customer.Id, nil).Id)
	t.Logf("customer=%s invoice=%s status=%s", customer.Id, invoice.Id, invoice.StatusDetails.ExtendedStatus)

	standardWant := withProfileBehavior(createdWant)
	standardGatheringEditedWant := withProfileBehavior(gatheringEditedWant)

	t.Run("standard invoice", func(t *testing.T) {
		for _, line := range *invoice.Lines {
			// Usage based lines without usage have no detailed lines.
			if !strings.Contains(line.Name, "usage_") {
				assert.NotEmpty(t, lo.FromPtr(line.Children), "detailed lines of %s", line.Name)
			}
		}

		got := v1InvoiceLineTaxConfigs(t, *invoice.Lines)
		assertV1TaxConfigs(t, standardWant, got)
		assertV1TaxConfigs(t, standardGatheringEditedWant, got)

		require.NotNil(t, invoice.Workflow.Workflow.Invoicing)
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, invoice.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	})

	// Lines created through the pending lines API are managed by charges in credits mode, so
	// their tax config can't be changed through an invoice edit.
	t.Run("standard invoice edit rejects changing a line's tax config", func(t *testing.T) {
		// given: the standard invoice
		// when: an edit gives the flat_none line a tax code
		// then: the edit is rejected
		lines := lo.Map(*invoice.Lines, func(line api.InvoiceLine, _ int) api.InvoiceLineReplaceUpdate {
			update := invoiceLineReplaceUpdateFromLine(t, line)

			if line.Name == "flat_none" {
				taxConfig := &api.TaxConfig{TaxCodeId: lo.ToPtr(taxCode.ID)}
				rateCard := *update.RateCard
				rateCard.TaxConfig = taxConfig

				update.TaxConfig = taxConfig
				update.RateCard = &rateCard
			}

			return update
		})

		resp, err := client.UpdateInvoiceWithResponse(t.Context(), invoice.Id, api.InvoiceReplaceUpdate{
			Customer:    billingPartyReplaceUpdateFromInvoiceCustomer(invoice.Customer),
			Supplier:    billingPartyReplaceUpdateFromParty(invoice.Supplier),
			Description: invoice.Description,
			Metadata:    invoice.Metadata,
			Lines:       lines,
			Workflow:    invoiceWorkflowReplaceUpdateFromInvoice(invoice),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "update invoice: %s", string(resp.Body))
		assert.Contains(t, string(resp.Body), "tax config editing is not supported for charge-managed invoice lines")
	})

	// The edit also sends workflow.invoicing.defaultTaxConfig, which the server accepts but
	// doesn't apply, so the invoice keeps the profile's default.
	existingWant := maps.Clone(standardWant)
	maps.Copy(existingWant, standardGatheringEditedWant)

	editLines := lo.Map(*invoice.Lines, func(line api.InvoiceLine, _ int) api.InvoiceLineReplaceUpdate {
		return invoiceLineReplaceUpdateFromLine(t, line)
	})

	editedInputs, editedWant := lineInputs("edited_")
	editLines = append(editLines, updateLines(editedInputs)...)
	standardEditedWant := withProfileBehavior(editedWant)

	workflow := invoiceWorkflowReplaceUpdateFromInvoice(invoice)
	workflow.Workflow.Invoicing.DefaultTaxConfig = &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorInclusive)}

	updated := updateInvoice(t, invoice, editLines, workflow)

	t.Run("standard invoice edit adds lines and ignores the default tax config", func(t *testing.T) {
		// given: the standard invoice, with an exclusive default behavior
		// when: the edit above echoes its lines, adds a line per scenario and shape, and sets an
		// inclusive default behavior
		// then: every line gets the exclusive behavior, and the invoice default stays exclusive
		got := v1InvoiceLineTaxConfigs(t, *updated.Lines)
		assertV1TaxConfigs(t, existingWant, got)
		assertV1TaxConfigs(t, standardEditedWant, got)

		require.NotNil(t, updated.Workflow.Workflow.Invoicing)
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, updated.Workflow.Workflow.Invoicing.DefaultTaxConfig)

		reread := getInvoice(t, invoice.Id)

		got = v1InvoiceLineTaxConfigs(t, *reread.Lines)
		assertV1TaxConfigs(t, existingWant, got)
		assertV1TaxConfigs(t, standardEditedWant, got)

		require.NotNil(t, reread.Workflow.Workflow.Invoicing)
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, reread.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	})
}
