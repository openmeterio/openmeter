package e2e

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
	"github.com/openmeterio/openmeter/e2e/internal/taxconfigv1"
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

	featureKey string // metered feature for this scenario's usage based rate card or line
}

// v1TaxConfigCatalogScenarios covers the inputs v1 accepts. taxCode must have a Stripe app
// mapping: the response renders stripe.code from the tax code's mapping, not from the input.
func v1TaxConfigCatalogScenarios(taxCode v3sdk.TaxCode) []v1TaxConfigScenario {
	stripeCode := taxconfigv1.StripeAppMapping(taxCode)

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
				Stripe:    &api.StripeTaxConfig{Code: taxconfigv1.RandomStripeTaxCode()},
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

// withV1TaxConfigFeatures creates one meter and one metered feature per scenario, so every
// scenario can also be expressed as a usage based rate card or line, and returns the scenarios
// with their feature keys filled in.
func withV1TaxConfigFeatures(t *testing.T, client *api.ClientWithResponses, scenarios []v1TaxConfigScenario) []v1TaxConfigScenario {
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

	scenarios = slices.Clone(scenarios)
	for i, scenario := range scenarios {
		featureResp, err := client.CreateFeatureWithResponse(t.Context(), api.CreateFeatureJSONRequestBody{
			Key:       uniqueKey("usage_" + scenario.key),
			Name:      "v1 taxConfig " + scenario.key,
			MeterSlug: lo.ToPtr(meterSlug),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, featureResp.StatusCode(), "create feature: %s", string(featureResp.Body))
		require.NotNil(t, featureResp.JSON201)

		scenarios[i].featureKey = featureResp.JSON201.Key
	}

	return scenarios
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

func createStripeMappedTaxCode(t *testing.T, c *v3Client) v3sdk.TaxCode {
	t.Helper()

	taxCode, err := c.Tax.CreateCode(t.Context(), v3sdk.CreateTaxCodeRequest{
		Key:  uniqueKey("v1_taxconfig"),
		Name: "v1 taxConfig characterization",
		AppMappings: []v3sdk.TaxCodeAppMapping{{
			AppType: v3sdk.AppTypeStripe,
			TaxCode: taxconfigv1.RandomStripeTaxCode(),
		}},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, taxCode)

	return *taxCode
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

// v1TaxConfigFlatFeeRateCard builds the flat fee rate card of one scenario, keyed
// keyPrefix+scenario.key, carrying the scenario's tax config.
func v1TaxConfigFlatFeeRateCard(t *testing.T, keyPrefix string, scenario v1TaxConfigScenario) api.RateCard {
	t.Helper()

	flatFee := api.RateCard{}
	require.NoError(t, flatFee.FromRateCardFlatFee(api.RateCardFlatFee{
		Key:  keyPrefix + scenario.key,
		Name: keyPrefix + scenario.key,
		Type: api.RateCardFlatFeeTypeFlatFee,
		Price: &api.FlatPriceWithPaymentTerm{
			Amount:      "10",
			PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
			Type:        api.FlatPriceWithPaymentTermTypeFlat,
		},
		BillingCadence: lo.ToPtr("P1M"),
		TaxConfig:      scenario.input,
	}))

	return flatFee
}

// v1TaxConfigFlatFeeRateCards builds one flat fee rate card per scenario, keyed
// keyPrefix+scenario, and returns the rate cards with the taxConfig each one must come back
// with.
//
// Addon rate cards and items added by a subscription edit use flat fee cards only: they can't
// reuse the plan's features, because v1 keys usage based rate cards by feature key and the
// keys would collide on the subscription.
func v1TaxConfigFlatFeeRateCards(t *testing.T, keyPrefix string, scenarios []v1TaxConfigScenario) ([]api.RateCard, map[string]*api.TaxConfig) {
	t.Helper()

	rateCards := make([]api.RateCard, 0, len(scenarios))
	want := make(map[string]*api.TaxConfig, len(scenarios))
	for _, scenario := range scenarios {
		rateCards = append(rateCards, v1TaxConfigFlatFeeRateCard(t, keyPrefix, scenario))
		want[keyPrefix+scenario.key] = scenario.want
	}

	return rateCards, want
}

// v1TaxConfigRateCards builds a flat fee rate card and a usage based one per scenario. The
// usage based card is keyed by the scenario's feature, as v1 requires, so scenarios must come
// from withV1TaxConfigFeatures. It returns the rate cards and the taxConfig each one must come
// back with.
func v1TaxConfigRateCards(t *testing.T, keyPrefix string, scenarios []v1TaxConfigScenario) ([]api.RateCard, map[string]*api.TaxConfig) {
	t.Helper()

	rateCards := make([]api.RateCard, 0, 2*len(scenarios))
	want := make(map[string]*api.TaxConfig, 2*len(scenarios))

	for _, scenario := range scenarios {
		require.NotEmpty(t, scenario.featureKey, "feature of scenario %s", scenario.key)

		rateCards = append(rateCards, v1TaxConfigFlatFeeRateCard(t, keyPrefix, scenario))
		want[keyPrefix+scenario.key] = scenario.want

		price := api.RateCardUsageBasedPrice{}
		require.NoError(t, price.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
			Amount: "0.1",
			Type:   api.UnitPriceWithCommitmentsTypeUnit,
		}))

		usageBased := api.RateCard{}
		require.NoError(t, usageBased.FromRateCardUsageBased(api.RateCardUsageBased{
			Key:            scenario.featureKey,
			Name:           scenario.featureKey,
			Type:           api.RateCardUsageBasedTypeUsageBased,
			FeatureKey:     lo.ToPtr(scenario.featureKey),
			Price:          &price,
			BillingCadence: "P1M",
			TaxConfig:      scenario.input,
		}))

		rateCards = append(rateCards, usageBased)
		want[scenario.featureKey] = scenario.want
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

// v1SubscriptionAddonRateCards unwraps the rate cards of a subscription addon.
func v1SubscriptionAddonRateCards(rateCards []api.SubscriptionAddonRateCard) []api.RateCard {
	return lo.Map(rateCards, func(rc api.SubscriptionAddonRateCard, _ int) api.RateCard {
		return rc.RateCard
	})
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

// v1ListedPlan lists plans filtered by the plan under test's key and returns it.
func v1ListedPlan(t *testing.T, client *api.ClientWithResponses, plan api.Plan) api.Plan {
	t.Helper()

	resp, err := client.ListPlansWithResponse(t.Context(), &api.ListPlansParams{
		Key:      &[]string{plan.Key},
		PageSize: lo.ToPtr(api.PaginationPageSize(100)),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "list plans: %s", string(resp.Body))
	require.NotNil(t, resp.JSON200)

	listed, found := lo.Find(resp.JSON200.Items, func(p api.Plan) bool { return p.Id == plan.Id })
	require.True(t, found, "plan %s not found in list", plan.Id)
	require.Len(t, listed.Phases, 1)

	return listed
}

func TestV1TaxConfigPlanRateCards(t *testing.T) {
	// given: a flat fee and a usage based rate card per scenario
	// when:
	//   1. a draft plan is created with them
	//   2. the plan is updated with every scenario on new flat fee keys
	// then: the create, GET, list and update responses render each scenario's tax config
	client := initClient(t)
	c := newV3Client(t)

	// given: a flat fee and a usage based rate card per scenario
	taxCode := createStripeMappedTaxCode(t, c)
	scenarios := withV1TaxConfigFeatures(t, client, v1TaxConfigCatalogScenarios(taxCode))

	rateCards, want := v1TaxConfigRateCards(t, "", scenarios)

	// when: a draft plan is created with them
	planKey := uniqueKey("v1_taxconfig_plan")
	createResp, err := client.CreatePlanWithResponse(t.Context(), api.PlanCreate{
		Key:            planKey,
		Name:           "v1 taxConfig plan",
		Currency:       "USD",
		BillingCadence: "P1M",
		Phases: []api.PlanPhase{{
			Key:       "default",
			Name:      "Default",
			RateCards: rateCards,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create plan: %s", string(createResp.Body))
	require.NotNil(t, createResp.JSON201)
	require.Len(t, createResp.JSON201.Phases, 1)

	plan := *createResp.JSON201

	t.Run("then: the create response and GET render the scenario tax configs", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, want, v1RateCardTaxConfigs(t, plan.Phases[0].RateCards))

		getResp, err := client.GetPlanWithResponse(t.Context(), plan.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get plan: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)
		require.Len(t, getResp.JSON200.Phases, 1)

		taxconfigv1.AssertTaxConfigs(t, want, v1RateCardTaxConfigs(t, getResp.JSON200.Phases[0].RateCards))
	})

	t.Run("then: listing plans by the plan key renders the created tax configs", func(t *testing.T) {
		listed := v1ListedPlan(t, client, plan)
		taxconfigv1.AssertTaxConfigs(t, want, v1RateCardTaxConfigs(t, listed.Phases[0].RateCards))
	})

	// Edits run at the top level, so every subtest only reads and can run on its own.
	// when: the plan is updated with every scenario again, on new flat fee keys (so new rate
	// cards are written) and on the usage based cards, which are keyed by their feature
	updatedRateCards, updatedWant := v1TaxConfigRateCards(t, "updated_", scenarios)

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

	t.Run("then: the update response and a re-read return the same tax configs as create", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, updateResp.JSON200.Phases[0].RateCards))

		getResp, err := client.GetPlanWithResponse(t.Context(), plan.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get plan: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)
		require.Len(t, getResp.JSON200.Phases, 1)

		taxconfigv1.AssertTaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, getResp.JSON200.Phases[0].RateCards))

		// The list response renders the plan through the same mapping as GET.
		listed := v1ListedPlan(t, client, plan)
		taxconfigv1.AssertTaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, listed.Phases[0].RateCards))
	})
}

// v1ListedAddon lists addons filtered by the addon under test's key and returns it.
func v1ListedAddon(t *testing.T, client *api.ClientWithResponses, addon api.Addon) api.Addon {
	t.Helper()

	resp, err := client.ListAddonsWithResponse(t.Context(), &api.ListAddonsParams{
		Key:      &[]string{addon.Key},
		PageSize: lo.ToPtr(api.PaginationPageSize(100)),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode(), "list addons: %s", string(resp.Body))
	require.NotNil(t, resp.JSON200)

	listed, found := lo.Find(resp.JSON200.Items, func(a api.Addon) bool { return a.Id == addon.Id })
	require.True(t, found, "addon %s not found in list", addon.Id)

	return listed
}

func TestV1TaxConfigAddonRateCards(t *testing.T) {
	// given: a flat fee and a usage based rate card per scenario
	// when:
	//   1. a draft addon is created with them
	//   2. the addon is updated with every scenario on new flat fee keys
	// then: the create, GET, list and update responses render each scenario's tax config
	client := initClient(t)
	c := newV3Client(t)

	// given: a flat fee and a usage based rate card per scenario
	taxCode := createStripeMappedTaxCode(t, c)
	scenarios := withV1TaxConfigFeatures(t, client, v1TaxConfigCatalogScenarios(taxCode))

	rateCards, want := v1TaxConfigRateCards(t, "", scenarios)

	// when: a draft addon is created with them
	addonKey := uniqueKey("v1_taxconfig_addon")
	createResp, err := client.CreateAddonWithResponse(t.Context(), api.AddonCreate{
		Key:          addonKey,
		Name:         "v1 taxConfig addon",
		Currency:     "USD",
		InstanceType: api.AddonInstanceTypeSingle,
		RateCards:    rateCards,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create addon: %s", string(createResp.Body))
	require.NotNil(t, createResp.JSON201)

	addon := *createResp.JSON201

	t.Run("then: the create response and GET render the scenario tax configs", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, want, v1RateCardTaxConfigs(t, addon.RateCards))

		getResp, err := client.GetAddonWithResponse(t.Context(), addon.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get addon: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		taxconfigv1.AssertTaxConfigs(t, want, v1RateCardTaxConfigs(t, getResp.JSON200.RateCards))
	})

	t.Run("then: listing addons by the addon key renders the created tax configs", func(t *testing.T) {
		listed := v1ListedAddon(t, client, addon)
		taxconfigv1.AssertTaxConfigs(t, want, v1RateCardTaxConfigs(t, listed.RateCards))
	})

	// Edits run at the top level, so every subtest only reads and can run on its own.
	// when: the addon is updated with every scenario again, on new flat fee keys (so new rate
	// cards are written) and on the usage based cards, which are keyed by their feature
	updatedRateCards, updatedWant := v1TaxConfigRateCards(t, "updated_", scenarios)

	updateResp, err := client.UpdateAddonWithResponse(t.Context(), addon.Id, api.AddonReplaceUpdate{
		Name:         addon.Name,
		InstanceType: addon.InstanceType,
		RateCards:    updatedRateCards,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, updateResp.StatusCode(), "update addon: %s", string(updateResp.Body))
	require.NotNil(t, updateResp.JSON200)

	t.Run("then: the update response and a re-read return the same tax configs as create", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, updateResp.JSON200.RateCards))

		getResp, err := client.GetAddonWithResponse(t.Context(), addon.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get addon: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		taxconfigv1.AssertTaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, getResp.JSON200.RateCards))

		// The list response renders the addon through the same mapping as GET.
		listed := v1ListedAddon(t, client, addon)
		taxconfigv1.AssertTaxConfigs(t, updatedWant, v1RateCardTaxConfigs(t, listed.RateCards))
	})
}

func TestV1TaxConfigSubscriptions(t *testing.T) {
	// given: scenarios covering every v1 taxConfig input, a published plan and addon with a rate
	// card per scenario, and a customer subscribed to the plan
	// when:
	//   1. billing syncs the subscription into gathering invoice lines
	//   2. an edit adds a flat fee item per scenario
	//   3. the addon is added to the subscription
	//   4. a new customer subscribes to a custom plan with every scenario
	// then: the subscription items, gathering lines, plan addons, subscription addons and custom
	// subscription items return each scenario's tax config
	client := initClient(t)
	c := newV3Client(t)

	// given: scenarios covering every v1 taxConfig input, with a metered feature each
	taxCode := createStripeMappedTaxCode(t, c)
	scenarios := withV1TaxConfigFeatures(t, client, v1TaxConfigCatalogScenarios(taxCode))

	timing := api.SubscriptionTiming{}
	require.NoError(t, timing.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

	rateCards, want := v1TaxConfigRateCards(t, "", scenarios)

	// given: a published addon with a rate card per scenario
	// The addon is attached to the plan before publishing, so subscriptions can add it.
	addonRateCards, addonWant := v1TaxConfigFlatFeeRateCards(t, "addon_", scenarios)

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

	addonID := addonResp.JSON201.Id

	// given: a published plan with a rate card per scenario and the addon attached
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

	planID := planResp.JSON201.Id

	planAddonResp, err := client.CreatePlanAddonWithResponse(t.Context(), planID, api.PlanAddonCreate{
		AddonId:       addonID,
		FromPlanPhase: "default",
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, planAddonResp.StatusCode(), "attach addon to plan: %s", string(planAddonResp.Body))

	publishPlanResp, err := client.PublishPlanWithResponse(t.Context(), planID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, publishPlanResp.StatusCode(), "publish plan: %s", string(publishPlanResp.Body))
	require.NotNil(t, publishPlanResp.JSON200)

	// given: a customer subscribed to the plan
	customer := taxconfigv1.CreateCustomer(t, client, "v1_taxconfig_sub_customer")

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

	t.Run("then: the subscription items return the plan's tax configs", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, want, v1SubscriptionItemTaxConfigs(t, client, subscriptionID))
	})

	t.Run("then: listing plan addons renders the addon's tax configs", func(t *testing.T) {
		resp, err := client.ListPlanAddonsWithResponse(t.Context(), planID, &api.ListPlanAddonsParams{
			PageSize: lo.ToPtr(api.PaginationPageSize(100)),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "list plan addons: %s", string(resp.Body))
		require.NotNil(t, resp.JSON200)

		planAddon, found := lo.Find(resp.JSON200.Items, func(planAddon api.PlanAddon) bool {
			return planAddon.Addon.Id == addonID
		})
		require.True(t, found, "plan addon of addon %s not found in list", addonID)

		taxconfigv1.AssertTaxConfigs(t, addonWant, v1RateCardTaxConfigs(t, planAddon.Addon.RateCards))
	})

	t.Run("then: gathering invoice lines get their item's tax config", func(t *testing.T) {
		// given: a subscription whose flat fee and usage based items carry every scenario
		// when: billing syncs the subscription into gathering invoice lines
		// then: each line has its item's tax config, with the default tax code filled in
		lineWant := v1TaxConfigInvoiceLineWant(scenarios, defaultInvoicingTaxCodeID(t, c))

		// Gathering lines are named after their rate card: the flat fee key, or the feature key
		// for usage based ones.
		subscriptionLineWant := make(map[string]*api.TaxConfig, 2*len(scenarios))
		for _, scenario := range scenarios {
			subscriptionLineWant[scenario.key] = lineWant[scenario.key]
			subscriptionLineWant[scenario.featureKey] = lineWant[scenario.key]
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

		taxconfigv1.AssertTaxConfigs(t, subscriptionLineWant, got)
	})

	// Edits run at the top level, so every subtest only reads and can run on its own. They stay
	// after the polling subtest, which waits for the plan's own items; the items added below are
	// extra keys to it.

	// when: an edit adds a flat fee item per scenario
	addedRateCards, addedWant := v1TaxConfigFlatFeeRateCards(t, "added_", scenarios)

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

	t.Run("then: the added items return the same tax configs as plan items", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, addedWant, v1SubscriptionItemTaxConfigs(t, client, editResp.JSON200.Id))
	})

	// when: the addon is added to the subscription
	addonAddResp, err := client.CreateSubscriptionAddonWithResponse(t.Context(), subscriptionID, api.SubscriptionAddonCreate{
		Addon: struct {
			Id string `json:"id"`
		}{Id: addonID},
		Name:     "v1 taxConfig subscription addon",
		Quantity: 1,
		Timing:   timing,
	})
	require.NoError(t, err)
	// The v1 subscription addon create answers 200 instead of the documented 201, so the
	// generated client doesn't decode the body.
	require.Equal(t, http.StatusOK, addonAddResp.StatusCode(), "add addon to subscription: %s", string(addonAddResp.Body))

	var subscriptionAddon api.SubscriptionAddon
	require.NoError(t, json.Unmarshal(addonAddResp.Body, &subscriptionAddon))

	t.Run("then: the subscription addon's rate cards and items return the addon's tax configs", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, addonWant, v1RateCardTaxConfigs(t, v1SubscriptionAddonRateCards(subscriptionAddon.RateCards)))

		getResp, err := client.GetSubscriptionAddonWithResponse(t.Context(), subscriptionID, subscriptionAddon.Id)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get subscription addon: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		taxconfigv1.AssertTaxConfigs(t, addonWant, v1RateCardTaxConfigs(t, v1SubscriptionAddonRateCards(getResp.JSON200.RateCards)))

		listResp, err := client.ListSubscriptionAddonsWithResponse(t.Context(), subscriptionID)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, listResp.StatusCode(), "list subscription addons: %s", string(listResp.Body))
		require.NotNil(t, listResp.JSON200)

		listedAddon, found := lo.Find(*listResp.JSON200, func(listed api.SubscriptionAddon) bool {
			return listed.Id == subscriptionAddon.Id
		})
		require.True(t, found, "subscription addon %s not found in list", subscriptionAddon.Id)

		taxconfigv1.AssertTaxConfigs(t, addonWant, v1RateCardTaxConfigs(t, v1SubscriptionAddonRateCards(listedAddon.RateCards)))

		// The addon's rate cards become subscription items of their own, since no plan item
		// shares their keys.
		taxconfigv1.AssertTaxConfigs(t, addonWant, v1SubscriptionItemTaxConfigs(t, client, subscriptionID))
	})

	// when: a new customer subscribes to a custom plan with every scenario
	customRateCards, customWant := v1TaxConfigRateCards(t, "", scenarios)
	customCustomer := taxconfigv1.CreateCustomer(t, client, "v1_taxconfig_custom_sub_customer")

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

	customResp, err := client.CreateSubscriptionWithResponse(t.Context(), create)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, customResp.StatusCode(), "create custom subscription: %s", string(customResp.Body))
	require.NotNil(t, customResp.JSON201)

	t.Run("then: the custom subscription items return the same tax configs as plan items", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, customWant, v1SubscriptionItemTaxConfigs(t, client, customResp.JSON201.Id))
	})
}

// v1BillingProfileCreateWithDefaultTaxConfig returns a create request for a profile with the
// given default tax config.
func v1BillingProfileCreateWithDefaultTaxConfig(t *testing.T, client *api.ClientWithResponses, defaultTaxConfig *api.TaxConfig) api.BillingProfileCreate {
	t.Helper()

	create := v1BillingProfileCreateFromDefault(t, client)
	create.Workflow.Invoicing.DefaultTaxConfig = defaultTaxConfig

	return create
}

// v1BillingProfileUpdateWithDefaultTaxConfig returns an update request that keeps the profile
// as is and sets the given default tax config.
func v1BillingProfileUpdateWithDefaultTaxConfig(profile api.BillingProfile, defaultTaxConfig *api.TaxConfig) api.BillingProfileReplaceUpdateWithWorkflow {
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

// Billing profiles no longer accept a tax code in defaultTaxConfig: create rejects it, and update
// only lets clients echo or remove a stored one. The e2e stack has no profile with a stored tax
// code, so only rejection and behavior round-trips are reachable here; echo and removal are
// covered by test/billing/taxcode_dual_write_test.go.
func TestV1TaxConfigBillingProfileDefaultTaxConfig(t *testing.T) {
	// given: a tax code with a Stripe mapping
	// when:
	//   1. billing profiles are created and updated with a tax code, no default tax config and
	//      behaviors
	//   2. a customer is pinned to the profile
	// then: tax codes are rejected with the deprecation detail, behaviors are kept and can be
	// removed, and the pinned customer's profile shows the profile's default tax config
	const deprecationDetail = "defaultTaxConfig is deprecated"

	client := initClient(t)
	c := newV3Client(t)

	// given: a tax code with a Stripe mapping
	taxCode := createStripeMappedTaxCode(t, c)

	t.Run("then: create rejects a tax code", func(t *testing.T) {
		for _, tt := range []struct {
			name      string
			taxConfig *api.TaxConfig
		}{
			{
				name:      "stripe_code_only",
				taxConfig: &api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: taxconfigv1.StripeAppMapping(taxCode)}},
			},
			{
				name:      "tax_code_id_only",
				taxConfig: &api.TaxConfig{TaxCodeId: lo.ToPtr(taxCode.ID)},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				resp, err := client.CreateBillingProfileWithResponse(t.Context(), v1BillingProfileCreateWithDefaultTaxConfig(t, client, tt.taxConfig))
				require.NoError(t, err)
				require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "create billing profile: %s", string(resp.Body))
				assert.Contains(t, string(resp.Body), deprecationDetail)
			})
		}
	})

	// when: a profile is created without a default tax config
	withoutDefaultResp, err := client.CreateBillingProfileWithResponse(t.Context(), v1BillingProfileCreateWithDefaultTaxConfig(t, client, nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, withoutDefaultResp.StatusCode(), "create billing profile: %s", string(withoutDefaultResp.Body))
	require.NotNil(t, withoutDefaultResp.JSON201)
	require.NotNil(t, withoutDefaultResp.JSON201.Workflow.Invoicing)

	t.Run("then: the create response has no default tax config", func(t *testing.T) {
		assert.Nil(t, withoutDefaultResp.JSON201.Workflow.Invoicing.DefaultTaxConfig)
	})

	// when: a profile is created with an exclusive default behavior
	createResp, err := client.CreateBillingProfileWithResponse(t.Context(), v1BillingProfileCreateWithDefaultTaxConfig(t, client, &api.TaxConfig{
		Behavior: lo.ToPtr(api.TaxBehaviorExclusive),
	}))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create billing profile: %s", string(createResp.Body))
	require.NotNil(t, createResp.JSON201)
	require.NotNil(t, createResp.JSON201.Workflow.Invoicing)

	profile := *createResp.JSON201

	t.Run("then: the create response renders the default behavior", func(t *testing.T) {
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, profile.Workflow.Invoicing.DefaultTaxConfig)
	})

	// when: a customer is pinned to the profile
	customer := taxconfigv1.CreateCustomer(t, client, "v1_taxconfig_profile_customer")

	overrideResp, err := client.UpsertBillingProfileCustomerOverrideWithResponse(t.Context(), customer.Id, api.BillingProfileCustomerOverrideCreate{
		BillingProfileId: lo.ToPtr(profile.Id),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, overrideResp.StatusCode(), "pin customer to billing profile: %s", string(overrideResp.Body))

	t.Run("then: the customer's profile shows the default tax config", func(t *testing.T) {
		getResp, err := client.GetBillingProfileCustomerOverrideWithResponse(t.Context(), customer.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get customer billing profile: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)
		require.NotNil(t, getResp.JSON200.CustomerProfile)
		require.NotNil(t, getResp.JSON200.CustomerProfile.Workflow.Invoicing)

		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, getResp.JSON200.CustomerProfile.Workflow.Invoicing.DefaultTaxConfig)
	})

	t.Run("then: update rejects adding a tax code", func(t *testing.T) {
		for _, tt := range []struct {
			name      string
			taxConfig *api.TaxConfig
		}{
			{
				name: "stripe_code_only",
				taxConfig: &api.TaxConfig{
					Behavior: lo.ToPtr(api.TaxBehaviorExclusive),
					Stripe:   &api.StripeTaxConfig{Code: taxconfigv1.StripeAppMapping(taxCode)},
				},
			},
			{
				name: "tax_code_id_only",
				taxConfig: &api.TaxConfig{
					Behavior:  lo.ToPtr(api.TaxBehaviorExclusive),
					TaxCodeId: lo.ToPtr(taxCode.ID),
				},
			},
		} {
			t.Run(tt.name, func(t *testing.T) {
				resp, err := client.UpdateBillingProfileWithResponse(t.Context(), profile.Id, v1BillingProfileUpdateWithDefaultTaxConfig(profile, tt.taxConfig))
				require.NoError(t, err)
				require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "update billing profile: %s", string(resp.Body))
				assert.Contains(t, string(resp.Body), deprecationDetail)
			})
		}
	})

	// when: the default behavior is updated to inclusive
	inclusiveResp, err := client.UpdateBillingProfileWithResponse(t.Context(), profile.Id, v1BillingProfileUpdateWithDefaultTaxConfig(profile, &api.TaxConfig{
		Behavior: lo.ToPtr(api.TaxBehaviorInclusive),
	}))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, inclusiveResp.StatusCode(), "update billing profile: %s", string(inclusiveResp.Body))
	require.NotNil(t, inclusiveResp.JSON200)
	require.NotNil(t, inclusiveResp.JSON200.Workflow.Invoicing)

	t.Run("then: the update response renders the inclusive behavior", func(t *testing.T) {
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorInclusive)}, inclusiveResp.JSON200.Workflow.Invoicing.DefaultTaxConfig)
	})

	// when: the default tax config is removed
	removedResp, err := client.UpdateBillingProfileWithResponse(t.Context(), profile.Id, v1BillingProfileUpdateWithDefaultTaxConfig(profile, nil))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, removedResp.StatusCode(), "update billing profile: %s", string(removedResp.Body))
	require.NotNil(t, removedResp.JSON200)
	require.NotNil(t, removedResp.JSON200.Workflow.Invoicing)

	t.Run("then: the update response no longer has a default tax config", func(t *testing.T) {
		assert.Nil(t, removedResp.JSON200.Workflow.Invoicing.DefaultTaxConfig)
	})
}

// v1TaxConfigLineInputs returns, per scenario, a flat fee line, a usage based line and a flat
// fee line in the deprecated shape, named prefix+{flat,usage,legacy}_+scenario, with each
// line's expected taxConfig taken from its scenario's entry in scenarioWant.
func v1TaxConfigLineInputs(t *testing.T, prefix string, scenarios []v1TaxConfigScenario, period api.Period, scenarioWant map[string]*api.TaxConfig) ([]taxconfigv1.LineInput, map[string]*api.TaxConfig) {
	t.Helper()

	inputs := make([]taxconfigv1.LineInput, 0, 3*len(scenarios))
	want := make(map[string]*api.TaxConfig, 3*len(scenarios))

	for _, scenario := range scenarios {
		flat := taxconfigv1.LineInput{
			Name:      prefix + "flat_" + scenario.key,
			InvoiceAt: period.From,
			Period:    period,
			RateCard:  &api.InvoiceUsageBasedRateCard{Price: taxconfigv1.FlatLinePrice(t), TaxConfig: scenario.input},
		}
		// Usage is billed in arrears, at the end of the period.
		usage := taxconfigv1.LineInput{
			Name:      prefix + "usage_" + scenario.key,
			InvoiceAt: period.To,
			Period:    period,
			RateCard: &api.InvoiceUsageBasedRateCard{
				FeatureKey: lo.ToPtr(scenario.featureKey),
				Price:      taxconfigv1.UnitLinePrice(t),
				TaxConfig:  scenario.input,
			},
		}
		legacy := taxconfigv1.LineInput{
			Name:      prefix + "legacy_" + scenario.key,
			InvoiceAt: period.From,
			Period:    period,
			Price:     taxconfigv1.FlatLinePrice(t),
			TaxConfig: scenario.input,
		}

		for _, input := range []taxconfigv1.LineInput{flat, usage, legacy} {
			inputs = append(inputs, input)
			want[input.Name] = scenarioWant[scenario.key]
		}
	}

	return inputs, want
}

func v1UpdateInvoice(t *testing.T, client *api.ClientWithResponses, invoice api.Invoice, lines []api.InvoiceLineReplaceUpdate, workflow api.InvoiceWorkflowReplaceUpdate) api.Invoice {
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

func TestV1TaxConfigInvoiceLines(t *testing.T) {
	// given: scenarios covering every v1 taxConfig input, and a customer pinned to a billing
	// profile that doesn't auto-advance invoices and defaults to an exclusive tax behavior
	// when:
	//   1. pending lines are created, one per scenario in every shape, on a gathering invoice
	//   2. the gathering invoice is edited with its lines echoed and new lines added
	//   3. the pending lines are billed into a standard invoice, which is edited
	// then: the responses render each line's tax config, the standard invoice merges the profile
	// behavior into its lines, changing a line's tax config is rejected, and the edit ignores
	// the invoice's default tax config
	client := initClient(t)
	c := newV3Client(t)

	// given: scenarios covering every v1 taxConfig input, with a metered feature each
	taxCode := createStripeMappedTaxCode(t, c)
	scenarios := withV1TaxConfigFeatures(t, client, v1TaxConfigCatalogScenarios(taxCode))

	// given: a customer pinned to a billing profile that doesn't auto-advance invoices and
	// defaults to an exclusive tax behavior
	// Without auto-advance the standard invoice stays in draft, so its lines can be edited. The
	// profile's default behavior is copied onto the invoice's workflow settings.
	profileCreate := v1BillingProfileCreateFromDefault(t, client)
	profileCreate.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)
	profileCreate.Workflow.Invoicing.DefaultTaxConfig = &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}

	profileResp, err := client.CreateBillingProfileWithResponse(t.Context(), profileCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, profileResp.StatusCode(), "create billing profile: %s", string(profileResp.Body))
	require.NotNil(t, profileResp.JSON201)

	profile := *profileResp.JSON201

	customer := taxconfigv1.CreateCustomer(t, client, "v1_taxconfig_line_customer")

	overrideResp, err := client.UpsertBillingProfileCustomerOverrideWithResponse(t.Context(), customer.Id, api.BillingProfileCustomerOverrideCreate{
		BillingProfileId: lo.ToPtr(profile.Id),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, overrideResp.StatusCode(), "pin customer to billing profile: %s", string(overrideResp.Body))

	periodEnd := time.Now().UTC().Truncate(time.Second)
	period := api.Period{From: periodEnd.Add(-24 * time.Hour), To: periodEnd}

	t.Run("then: create rejects different taxConfig and rateCard.taxConfig", func(t *testing.T) {
		resp, err := client.CreatePendingInvoiceLineWithResponse(t.Context(), customer.Id, api.InvoicePendingLineCreateInput{
			Currency: "USD",
			Lines: []api.InvoicePendingLineCreate{{
				Name:      "mismatch",
				InvoiceAt: period.From,
				Period:    period,
				RateCard:  &api.InvoiceUsageBasedRateCard{Price: taxconfigv1.FlatLinePrice(t), TaxConfig: &api.TaxConfig{TaxCodeId: lo.ToPtr(taxCode.ID)}},
				TaxConfig: &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)},
			}},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "create pending line: %s", string(resp.Body))
		assert.Contains(t, string(resp.Body), "taxConfig must be equal to rateCard.taxConfig")
	})

	// Charges give lines without a tax code the organization's default one.
	lineWant := v1TaxConfigInvoiceLineWant(scenarios, defaultInvoicingTaxCodeID(t, c))

	// when: pending lines are created, one per scenario in every shape, on a gathering invoice
	createdInputs, createdWant := v1TaxConfigLineInputs(t, "", scenarios, period, lineWant)

	createResp, err := client.CreatePendingInvoiceLineWithResponse(t.Context(), customer.Id, api.InvoicePendingLineCreateInput{
		Currency: "USD",
		Lines:    taxconfigv1.PendingLines(createdInputs),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create pending lines: %s", string(createResp.Body))
	require.NotNil(t, createResp.JSON201)

	gatheringInvoiceID := createResp.JSON201.Invoice.Id

	t.Run("then: the create response renders the created lines' tax configs", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, createdWant, taxconfigv1.InvoiceLineTaxConfigs(t, createResp.JSON201.Lines))
	})

	t.Run("then: the gathering invoice renders the created lines' tax configs", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, createdWant, taxconfigv1.InvoiceLineTaxConfigs(t, *taxconfigv1.Invoice(t, client, gatheringInvoiceID).Lines))
	})

	// Edits run at the top level, so every subtest only reads and can run on its own.
	// when: the edit echoes the created lines and adds a line per scenario and shape
	gatheringEditedInputs, gatheringEditedWant := v1TaxConfigLineInputs(t, "gathering_edited_", scenarios, period, lineWant)

	gatheringInvoice := taxconfigv1.Invoice(t, client, gatheringInvoiceID)

	gatheringLines := lo.Map(*gatheringInvoice.Lines, func(line api.InvoiceLine, _ int) api.InvoiceLineReplaceUpdate {
		return invoiceLineReplaceUpdateFromLine(t, line)
	})
	gatheringLines = append(gatheringLines, taxconfigv1.NewInvoiceLines(gatheringEditedInputs)...)

	gatheringEdited := v1UpdateInvoice(t, client, gatheringInvoice, gatheringLines, invoiceWorkflowReplaceUpdateFromInvoice(gatheringInvoice))

	t.Run("then: the edit response and a re-read return the created and added lines' tax configs", func(t *testing.T) {
		got := taxconfigv1.InvoiceLineTaxConfigs(t, *gatheringEdited.Lines)
		taxconfigv1.AssertTaxConfigs(t, createdWant, got)
		taxconfigv1.AssertTaxConfigs(t, gatheringEditedWant, got)

		got = taxconfigv1.InvoiceLineTaxConfigs(t, *taxconfigv1.Invoice(t, client, gatheringInvoiceID).Lines)
		taxconfigv1.AssertTaxConfigs(t, createdWant, got)
		taxconfigv1.AssertTaxConfigs(t, gatheringEditedWant, got)
	})

	// given: a standard invoice billed now from the pending lines
	invoice := taxconfigv1.Invoice(t, client, invoicePendingLinesNow(t, client, customer.Id, nil).Id)
	t.Logf("customer=%s invoice=%s status=%s", customer.Id, invoice.Id, invoice.StatusDetails.ExtendedStatus)

	standardWant := taxconfigv1.WithProfileBehavior(createdWant)
	standardGatheringEditedWant := taxconfigv1.WithProfileBehavior(gatheringEditedWant)

	t.Run("then: the standard invoice merges the profile behavior into its lines", func(t *testing.T) {
		for _, line := range *invoice.Lines {
			// Usage based lines without usage have no detailed lines.
			if !strings.Contains(line.Name, "usage_") {
				assert.NotEmpty(t, lo.FromPtr(line.Children), "detailed lines of %s", line.Name)
			}
		}

		got := taxconfigv1.InvoiceLineTaxConfigs(t, *invoice.Lines)
		taxconfigv1.AssertTaxConfigs(t, standardWant, got)
		taxconfigv1.AssertTaxConfigs(t, standardGatheringEditedWant, got)

		require.NotNil(t, invoice.Workflow.Workflow.Invoicing)
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, invoice.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	})

	// Lines created through the pending lines API are managed by charges in credits mode, so
	// their tax config can't be changed through an invoice edit.
	t.Run("then: the standard invoice edit rejects changing a line's tax config", func(t *testing.T) {
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
	// when: the edit echoes the invoice's lines, adds a line per scenario and shape, and sets an
	// inclusive default behavior
	existingWant := maps.Clone(standardWant)
	maps.Copy(existingWant, standardGatheringEditedWant)

	editLines := lo.Map(*invoice.Lines, func(line api.InvoiceLine, _ int) api.InvoiceLineReplaceUpdate {
		return invoiceLineReplaceUpdateFromLine(t, line)
	})

	editedInputs, editedWant := v1TaxConfigLineInputs(t, "edited_", scenarios, period, lineWant)
	editLines = append(editLines, taxconfigv1.NewInvoiceLines(editedInputs)...)
	standardEditedWant := taxconfigv1.WithProfileBehavior(editedWant)

	workflow := invoiceWorkflowReplaceUpdateFromInvoice(invoice)
	workflow.Workflow.Invoicing.DefaultTaxConfig = &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorInclusive)}

	updated := v1UpdateInvoice(t, client, invoice, editLines, workflow)

	t.Run("then: the standard invoice edit adds lines and ignores the default tax config", func(t *testing.T) {
		// given: the standard invoice, with an exclusive default behavior
		got := taxconfigv1.InvoiceLineTaxConfigs(t, *updated.Lines)
		taxconfigv1.AssertTaxConfigs(t, existingWant, got)
		taxconfigv1.AssertTaxConfigs(t, standardEditedWant, got)

		require.NotNil(t, updated.Workflow.Workflow.Invoicing)
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, updated.Workflow.Workflow.Invoicing.DefaultTaxConfig)

		reread := taxconfigv1.Invoice(t, client, invoice.Id)

		got = taxconfigv1.InvoiceLineTaxConfigs(t, *reread.Lines)
		taxconfigv1.AssertTaxConfigs(t, existingWant, got)
		taxconfigv1.AssertTaxConfigs(t, standardEditedWant, got)

		require.NotNil(t, reread.Workflow.Workflow.Invoicing)
		assert.Equal(t, &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}, reread.Workflow.Workflow.Invoicing.DefaultTaxConfig)
	})
}

func TestV1TaxConfigUnmappedStripeCode(t *testing.T) {
	// given: fresh unmapped Stripe codes for the plan, addon and invoice line writes, and a
	// mapped tax code for the ID-wins write
	// when:
	//   1. a plan (flat fee and usage based rate cards), an addon and a pending invoice line
	//      are written with stripe-only tax configs; the plan is published and subscribed to
	//   2. a second plan is written with the plan's Stripe code again
	//   3. an addon rate card sends a mapped tax code ID next to a fresh unmapped code
	// then: every stripe-only write resolves to a tax code rendered as
	// {stripe: {code: sent}, taxCodeId: <id>}, the reused code resolves to the same tax code,
	// and the ID wins without creating a tax code for the ignored code
	client := initClient(t)
	c := newV3Client(t)

	// given: one unmapped Stripe code per write path, and a mapped tax code for the ID-wins write
	planCode := taxconfigv1.RandomStripeTaxCode()
	addonCode := taxconfigv1.RandomStripeTaxCode()
	lineCode := taxconfigv1.RandomStripeTaxCode()
	ignoredCode := taxconfigv1.RandomStripeTaxCode()

	mappedTaxCode := createStripeMappedTaxCode(t, c)

	// given: the plan scenario's metered feature, so the plan carries the code on a flat fee
	// and a usage based rate card
	planScenarios := withV1TaxConfigFeatures(t, client, []v1TaxConfigScenario{{
		key:   "flat",
		input: &api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: planCode}},
	}})
	planRateCards, _ := v1TaxConfigRateCards(t, "", planScenarios)

	addonRateCards, _ := v1TaxConfigFlatFeeRateCards(t, "addon_", []v1TaxConfigScenario{{
		key:   "flat",
		input: &api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: addonCode}},
	}})

	reuseRateCards, _ := v1TaxConfigFlatFeeRateCards(t, "reuse_", []v1TaxConfigScenario{{
		key:   "flat",
		input: &api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: planCode}},
	}})

	winnerRateCards, _ := v1TaxConfigFlatFeeRateCards(t, "ignored_", []v1TaxConfigScenario{{
		key: "flat",
		input: &api.TaxConfig{
			Stripe:    &api.StripeTaxConfig{Code: ignoredCode},
			TaxCodeId: lo.ToPtr(mappedTaxCode.ID),
		},
	}})

	customer := taxconfigv1.CreateCustomer(t, client, "v1_taxconfig_unmapped_sub_customer")

	// given: a customer pinned to a billing profile that doesn't auto-advance invoices and
	// defaults to an exclusive tax behavior, like the invoice line test's customer
	profileCreate := v1BillingProfileCreateFromDefault(t, client)
	profileCreate.Workflow.Invoicing.AutoAdvance = lo.ToPtr(false)
	profileCreate.Workflow.Invoicing.DefaultTaxConfig = &api.TaxConfig{Behavior: lo.ToPtr(api.TaxBehaviorExclusive)}

	profileResp, err := client.CreateBillingProfileWithResponse(t.Context(), profileCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, profileResp.StatusCode(), "create billing profile: %s", string(profileResp.Body))
	require.NotNil(t, profileResp.JSON201)

	lineCustomer := taxconfigv1.CreateCustomer(t, client, "v1_taxconfig_unmapped_line_customer")

	overrideResp, err := client.UpsertBillingProfileCustomerOverrideWithResponse(t.Context(), lineCustomer.Id, api.BillingProfileCustomerOverrideCreate{
		BillingProfileId: lo.ToPtr(profileResp.JSON201.Id),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, overrideResp.StatusCode(), "pin customer to billing profile: %s", string(overrideResp.Body))

	periodEnd := time.Now().UTC().Truncate(time.Second)
	period := api.Period{From: periodEnd.Add(-24 * time.Hour), To: periodEnd}

	lineInput := taxconfigv1.LineInput{
		Name:      "flat_unmapped_code",
		InvoiceAt: period.From,
		Period:    period,
		RateCard: &api.InvoiceUsageBasedRateCard{
			Price:     taxconfigv1.FlatLinePrice(t),
			TaxConfig: &api.TaxConfig{Stripe: &api.StripeTaxConfig{Code: lineCode}},
		},
	}

	// when: a draft plan is created with the plan rate cards, then published
	planResp, err := client.CreatePlanWithResponse(t.Context(), api.PlanCreate{
		Key:            uniqueKey("v1_taxconfig_unmapped_plan"),
		Name:           "v1 taxConfig unmapped stripe plan",
		Currency:       "USD",
		BillingCadence: "P1M",
		SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
		Phases: []api.PlanPhase{{
			Key:       "default",
			Name:      "Default",
			RateCards: planRateCards,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, planResp.StatusCode(), "create plan: %s", string(planResp.Body))
	require.NotNil(t, planResp.JSON201)
	require.Len(t, planResp.JSON201.Phases, 1)

	planID := planResp.JSON201.Id

	publishPlanResp, err := client.PublishPlanWithResponse(t.Context(), planID)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, publishPlanResp.StatusCode(), "publish plan: %s", string(publishPlanResp.Body))
	require.NotNil(t, publishPlanResp.JSON200)

	// when: a draft addon is created with the addon's unmapped code
	addonResp, err := client.CreateAddonWithResponse(t.Context(), api.AddonCreate{
		Key:          uniqueKey("v1_taxconfig_unmapped_addon"),
		Name:         "v1 taxConfig unmapped stripe addon",
		Currency:     "USD",
		InstanceType: api.AddonInstanceTypeSingle,
		RateCards:    addonRateCards,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, addonResp.StatusCode(), "create addon: %s", string(addonResp.Body))
	require.NotNil(t, addonResp.JSON201)

	// when: a customer subscribes to the published plan
	timing := api.SubscriptionTiming{}
	require.NoError(t, timing.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

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

	// when: a pending invoice line is created with the line's unmapped code
	lineCreateResp, err := client.CreatePendingInvoiceLineWithResponse(t.Context(), lineCustomer.Id, api.InvoicePendingLineCreateInput{
		Currency: "USD",
		Lines:    taxconfigv1.PendingLines([]taxconfigv1.LineInput{lineInput}),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, lineCreateResp.StatusCode(), "create pending line: %s", string(lineCreateResp.Body))
	require.NotNil(t, lineCreateResp.JSON201)

	gatheringInvoiceID := lineCreateResp.JSON201.Invoice.Id
	t.Logf("line customer=%s gathering invoice=%s", lineCustomer.Id, gatheringInvoiceID)

	// when: a second plan is written with the plan's Stripe code again
	reusePlanResp, err := client.CreatePlanWithResponse(t.Context(), api.PlanCreate{
		Key:            uniqueKey("v1_taxconfig_reuse_plan"),
		Name:           "v1 taxConfig unmapped stripe reuse plan",
		Currency:       "USD",
		BillingCadence: "P1M",
		Phases: []api.PlanPhase{{
			Key:       "default",
			Name:      "Default",
			RateCards: reuseRateCards,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, reusePlanResp.StatusCode(), "create reuse plan: %s", string(reusePlanResp.Body))
	require.NotNil(t, reusePlanResp.JSON201)
	require.Len(t, reusePlanResp.JSON201.Phases, 1)

	// when: an addon rate card sends a mapped tax code ID next to a fresh unmapped code
	winnerAddonResp, err := client.CreateAddonWithResponse(t.Context(), api.AddonCreate{
		Key:          uniqueKey("v1_taxconfig_id_wins_addon"),
		Name:         "v1 taxConfig ID wins addon",
		Currency:     "USD",
		InstanceType: api.AddonInstanceTypeSingle,
		RateCards:    winnerRateCards,
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, winnerAddonResp.StatusCode(), "create ID wins addon: %s", string(winnerAddonResp.Body))
	require.NotNil(t, winnerAddonResp.JSON201)

	// The auto-created tax code IDs are only known from the create responses; every later read
	// of the same resource must render them.
	planCreateGot := v1RateCardTaxConfigs(t, planResp.JSON201.Phases[0].RateCards)
	planWant := taxconfigv1.StripeOnlyWant(t, planCode, planCreateGot["flat"])
	planCardWant := map[string]*api.TaxConfig{
		"flat":                      planWant,
		planScenarios[0].featureKey: planWant,
	}

	addonCreateGot := v1RateCardTaxConfigs(t, addonResp.JSON201.RateCards)
	addonWant := taxconfigv1.StripeOnlyWant(t, addonCode, addonCreateGot["addon_flat"])

	lineCreateGot := taxconfigv1.InvoiceLineTaxConfigs(t, lineCreateResp.JSON201.Lines)
	lineWant := taxconfigv1.StripeOnlyWant(t, lineCode, lineCreateGot["flat_unmapped_code"])

	// The ID wins, so the ignored code is replaced by the mapped tax code's own mapping.
	winnerWant := &api.TaxConfig{
		Stripe:    &api.StripeTaxConfig{Code: taxconfigv1.StripeAppMapping(mappedTaxCode)},
		TaxCodeId: lo.ToPtr(mappedTaxCode.ID),
	}

	t.Run("then: the plan create response and GET render the auto-created tax code", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, planCardWant, planCreateGot)

		getResp, err := client.GetPlanWithResponse(t.Context(), planID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get plan: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)
		require.Len(t, getResp.JSON200.Phases, 1)

		taxconfigv1.AssertTaxConfigs(t, planCardWant, v1RateCardTaxConfigs(t, getResp.JSON200.Phases[0].RateCards))
	})

	t.Run("then: the addon create response and GET render the auto-created tax code", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, map[string]*api.TaxConfig{"addon_flat": addonWant}, addonCreateGot)

		getResp, err := client.GetAddonWithResponse(t.Context(), addonResp.JSON201.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get addon: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		taxconfigv1.AssertTaxConfigs(t, map[string]*api.TaxConfig{"addon_flat": addonWant}, v1RateCardTaxConfigs(t, getResp.JSON200.RateCards))
	})

	t.Run("then: the subscription items render the plan's tax code", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, planCardWant, v1SubscriptionItemTaxConfigs(t, client, subscriptionID))
	})

	t.Run("then: the pending line create response and the gathering invoice render the auto-created tax code", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, map[string]*api.TaxConfig{"flat_unmapped_code": lineWant}, lineCreateGot)

		gathering := taxconfigv1.Invoice(t, client, gatheringInvoiceID)
		taxconfigv1.AssertTaxConfigs(t, map[string]*api.TaxConfig{"flat_unmapped_code": lineWant}, taxconfigv1.InvoiceLineTaxConfigs(t, *gathering.Lines))
	})

	t.Run("then: the auto-created tax codes are visible through v3 with the sent stripe mappings", func(t *testing.T) {
		taxconfigv1.AssertTaxCodeStripeMapping(t, c.Client, *planWant.TaxCodeId, planCode)
		taxconfigv1.AssertTaxCodeStripeMapping(t, c.Client, *addonWant.TaxCodeId, addonCode)
		taxconfigv1.AssertTaxCodeStripeMapping(t, c.Client, *lineWant.TaxCodeId, lineCode)
	})

	t.Run("then: writing the plan's stripe code again returns the same tax code", func(t *testing.T) {
		reuseGot := v1RateCardTaxConfigs(t, reusePlanResp.JSON201.Phases[0].RateCards)
		taxconfigv1.AssertTaxConfigs(t, map[string]*api.TaxConfig{"reuse_flat": planWant}, reuseGot)
	})

	t.Run("then: the tax code ID wins over the ignored stripe code and no tax code is created for it", func(t *testing.T) {
		winnerGot := v1RateCardTaxConfigs(t, winnerAddonResp.JSON201.RateCards)
		taxconfigv1.AssertTaxConfigs(t, map[string]*api.TaxConfig{"ignored_flat": winnerWant}, winnerGot)

		getResp, err := client.GetAddonWithResponse(t.Context(), winnerAddonResp.JSON201.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get ID wins addon: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		taxconfigv1.AssertTaxConfigs(t, map[string]*api.TaxConfig{"ignored_flat": winnerWant}, v1RateCardTaxConfigs(t, getResp.JSON200.RateCards))

		for taxCode, err := range c.Tax.ListCodesAll(t.Context(), v3sdk.TaxCodeListParams{}) {
			require.NoError(t, err)
			assert.NotEqual(t, ignoredCode, taxconfigv1.StripeAppMapping(taxCode), "tax code %s must not carry the ignored stripe code", taxCode.ID)
		}
	})
}

// v3TaxCodeFlatRateCard builds a v3 flat fee rate card keyed key, referencing the tax code by ID.
func v3TaxCodeFlatRateCard(key, taxCodeID string) v3sdk.RateCardInput {
	cadence := "P1M"
	paymentTerm := v3sdk.PricePaymentTermInAdvance

	return v3sdk.RateCardInput{
		Key:            key,
		Name:           key,
		Price:          lo.Must(v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{Amount: "10"})),
		BillingCadence: &cadence,
		PaymentTerm:    &paymentTerm,
		TaxConfig:      &v3sdk.TaxCodeConfig{Code: &v3sdk.TaxCodeReference{ID: taxCodeID}},
	}
}

func TestV1TaxConfigV3CreatedCatalog(t *testing.T) {
	// given: a tax code with a Stripe app mapping and one without any
	// when:
	//   1. a v3 plan and addon are created with one flat fee rate card per tax code and
	//      published, and a customer subscribes to the plan through v3
	//   2. billing syncs the subscription into gathering invoice lines
	// then: v1 renders the mapped tax code's stripe.code next to its ID and omits stripe for
	// the unmapped tax code, on the plan, the addon, the subscription items and the gathering
	// invoice lines
	client := initClient(t)
	c := newV3Client(t)

	// given: a tax code with a Stripe app mapping and one without any
	mappedCode := createStripeMappedTaxCode(t, c)

	unmappedCode, err := c.Tax.CreateCode(t.Context(), v3sdk.CreateTaxCodeRequest{
		Key:         uniqueKey("v1_taxconfig"),
		Name:        "v1 taxConfig characterization",
		AppMappings: []v3sdk.TaxCodeAppMapping{},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, unmappedCode)

	// v1 renders the mapped tax code's config from its Stripe app mapping and leaves the
	// unmapped one with its ID only.
	mappedWant := &api.TaxConfig{
		Stripe:    &api.StripeTaxConfig{Code: taxconfigv1.StripeAppMapping(mappedCode)},
		TaxCodeId: lo.ToPtr(mappedCode.ID),
	}
	unmappedWant := &api.TaxConfig{TaxCodeId: lo.ToPtr(unmappedCode.ID)}

	rateCardWant := map[string]*api.TaxConfig{
		"mapped_flat":   mappedWant,
		"unmapped_flat": unmappedWant,
	}

	// when: a v3 plan carries one flat fee rate card per tax code and is published
	plan, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
		Key:            uniqueKey("v1_taxconfig_v3_plan"),
		Name:           "v1 taxConfig v3 plan",
		Currency:       "USD",
		BillingCadence: "P1M",
		Phases: []v3sdk.PlanPhaseInput{{
			Key:  "default",
			Name: "Default",
			RateCards: []v3sdk.RateCardInput{
				v3TaxCodeFlatRateCard("mapped_flat", mappedCode.ID),
				v3TaxCodeFlatRateCard("unmapped_flat", unmappedCode.ID),
			},
		}},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, plan)

	publishedPlan, err := c.Plans.Publish(t.Context(), plan.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, publishedPlan)
	require.Equal(t, v3sdk.PlanStatusActive, publishedPlan.Status)

	// when: a v3 addon carries the same rate cards and is published
	addon, err := c.Addons.Create(t.Context(), v3sdk.CreateAddonRequest{
		Key:          uniqueKey("v1_taxconfig_v3_addon"),
		Name:         "v1 taxConfig v3 addon",
		Currency:     "USD",
		InstanceType: v3sdk.AddonInstanceTypeSingle,
		RateCards: []v3sdk.RateCardInput{
			v3TaxCodeFlatRateCard("mapped_flat", mappedCode.ID),
			v3TaxCodeFlatRateCard("unmapped_flat", unmappedCode.ID),
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, addon)

	publishedAddon, err := c.Addons.Publish(t.Context(), addon.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, publishedAddon)

	// when: a customer subscribes to the published plan through v3
	customer := taxconfigv1.CreateCustomer(t, client, "v1_taxconfig_v3_customer")

	subscription, err := c.Subscriptions.Create(t.Context(), v3sdk.SubscriptionCreate{
		Customer:       v3sdk.SubscriptionChangeCustomer{ID: lo.ToPtr(customer.Id)},
		Plan:           &v3sdk.SubscriptionChangePlan{ID: lo.ToPtr(publishedPlan.ID)},
		SettlementMode: lo.ToPtr(v3sdk.SettlementModeCreditThenInvoice),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, subscription)
	require.Equal(t, v3sdk.SubscriptionStatusActive, subscription.Status)

	t.Logf("customer=%s subscription=%s", customer.Id, subscription.ID)

	t.Run("then: v1 GET and list render the plan rate cards' tax configs", func(t *testing.T) {
		getResp, err := client.GetPlanWithResponse(t.Context(), plan.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get plan: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)
		require.Len(t, getResp.JSON200.Phases, 1)

		taxconfigv1.AssertTaxConfigs(t, rateCardWant, v1RateCardTaxConfigs(t, getResp.JSON200.Phases[0].RateCards))

		listed := v1ListedPlan(t, client, api.Plan{Id: plan.ID, Key: plan.Key})
		taxconfigv1.AssertTaxConfigs(t, rateCardWant, v1RateCardTaxConfigs(t, listed.Phases[0].RateCards))
	})

	t.Run("then: v1 GET and list render the addon rate cards' tax configs", func(t *testing.T) {
		getResp, err := client.GetAddonWithResponse(t.Context(), addon.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get addon: %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		taxconfigv1.AssertTaxConfigs(t, rateCardWant, v1RateCardTaxConfigs(t, getResp.JSON200.RateCards))

		listed := v1ListedAddon(t, client, api.Addon{Id: addon.ID, Key: addon.Key})
		taxconfigv1.AssertTaxConfigs(t, rateCardWant, v1RateCardTaxConfigs(t, listed.RateCards))
	})

	t.Run("then: the subscription items render the rate cards' tax configs", func(t *testing.T) {
		taxconfigv1.AssertTaxConfigs(t, rateCardWant, v1SubscriptionItemTaxConfigs(t, client, subscription.ID))
	})

	t.Run("then: the gathering invoice lines render the rate cards' tax configs", func(t *testing.T) {
		// given: the subscription whose items carry both tax codes
		// when: billing syncs the subscription into gathering invoice lines
		// then: each line renders its item's tax config, with or without the mapped stripe code
		customers := api.InvoiceListParamsCustomers{customer.Id}
		expand := api.InvoiceListParamsExpand{api.InvoiceExpandLines}

		var subscriptionLines []api.InvoiceLine
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			resp, err := client.ListInvoicesWithResponse(t.Context(), &api.ListInvoicesParams{
				Customers: &customers,
				Expand:    &expand,
				PageSize:  lo.ToPtr(api.PaginationPageSize(100)),
			})
			require.NoError(collect, err)
			require.Equal(collect, http.StatusOK, resp.StatusCode(), "list invoices: %s", string(resp.Body))
			require.NotNil(collect, resp.JSON200)

			subscriptionLines = nil
			for _, invoice := range resp.JSON200.Items {
				for _, line := range lo.FromPtr(invoice.Lines) {
					if line.Subscription != nil {
						subscriptionLines = append(subscriptionLines, line)
					}
				}
			}

			names := make(map[string]struct{}, len(subscriptionLines))
			for _, line := range subscriptionLines {
				names[line.Name] = struct{}{}
			}

			for key := range rateCardWant {
				require.Contains(collect, names, key)
			}
		}, time.Minute, time.Second)

		taxconfigv1.AssertTaxConfigs(t, rateCardWant, taxconfigv1.InvoiceLineTaxConfigs(t, subscriptionLines))
	})
}
