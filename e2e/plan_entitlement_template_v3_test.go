package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
)

// TestV3PlanRateCardEntitlementTemplateRepro exercises configuring an entitlement
// on a feature-linked rate card via the v3 POST /plans endpoint, and asserts a
// subscriber then receives the metered entitlement. The whole flow is v3-only —
// the same subset a Kong Konnect / v3 consumer is limited to.
//
// It started life as a reproduction of a reported defect: the v3 BillingRateCard
// schema had no entitlement field, and api/v3/handlers/plans/convert.go's
// FromAPIBillingRateCard never mapped one — so the config could not be set at
// all (local servers rejected it with 400 "property \"entitlement\" is
// unsupported"; Kong Konnect's gateway stripped it and returned a silent 201).
//
// The fix adds the `entitlement` field (simplified metered shape: `limit`,
// `is_soft_limit`, `usage_period`) to the v3 schema and wires it through the
// converter. This test asserts the corrected
// end-to-end behavior: create is accepted (201), the entitlement round-trips on
// GET, and the metered entitlement materializes on subscription, surfacing via
// GET .../entitlement-access.
//
// The flow:
//
//	create a meter -> create a metered feature
//	-> POST /plans (entitlement on rate card) -> 201
//	-> GET /plans/{id} (entitlement round-trips)
//	-> POST /plans/{id}/publish
//	-> POST /customers (subject carried via usage_attribution; no subjects API in v3)
//	-> POST /subscriptions (v3 has no timing on create; starts immediately)
//	-> GET /customers/{id}/entitlement-access -> expect metered entitlement, has_access=true
//
// Only the product-catalog + subscription path is exercised, so a plain
// `make server` is sufficient.
func TestV3PlanRateCardEntitlementTemplateRepro(t *testing.T) {
	c := newV3Client(t)

	// Create a dedicated meter so the test is hermetic — independent of which
	// meters the target server's config defines (CI uses e2e/config.yaml).
	valueProperty := "$.value"
	status, meter, problem := c.CreateMeter(apiv3.CreateMeterRequest{
		Key:           uniqueKey("data_tokens_meter"),
		Name:          "Data Tokens Meter",
		Aggregation:   apiv3.MeterAggregationSum,
		EventType:     uniqueKey("data_tokens_event"),
		ValueProperty: &valueProperty,
	})
	require.Equal(t, http.StatusCreated, status, "create meter: %+v", problem)
	require.NotNil(t, meter)

	// Metered feature backed by that meter.
	featureKey := uniqueKey("data_tokens")
	status, feature, problem := c.CreateFeature(apiv3.CreateFeatureRequest{
		Key:   featureKey,
		Name:  "Data Tokens",
		Meter: &apiv3.FeatureMeterReference{Id: meter.Id},
	})
	require.Equal(t, http.StatusCreated, status, "create feature: %+v", problem)
	require.NotNil(t, feature)

	// v3 plan with a metered entitlement on the feature-linked flat rate card.
	planKey := uniqueKey("ent_repro_plan")
	status, created, problem := c.CreatePlan(apiv3.CreatePlanRequest{
		Key:            planKey,
		Name:           "Dashboard Prepaid Plus",
		Currency:       "USD",
		BillingCadence: apiv3.ISO8601Duration("P1M"),
		Phases: []apiv3.BillingPlanPhase{{
			Key:       "subscription",
			Name:      "Subscription",
			RateCards: []apiv3.BillingRateCard{meteredEntitlementRateCard(*feature, 15_000_000)},
		}},
	})
	require.Equal(t, http.StatusCreated, status, "create plan: %+v", problem)
	require.NotNil(t, created)

	// The entitlement should round-trip on GET (read path wired through the converter).
	status, got, problem := c.GetPlan(created.Id)
	require.Equal(t, http.StatusOK, status, "get plan: %+v", problem)
	require.NotNil(t, got)
	require.Len(t, got.Phases, 1)
	require.Len(t, got.Phases[0].RateCards, 1)
	require.NotNil(t, got.Phases[0].RateCards[0].Entitlement, "entitlement should round-trip on GET")

	rtMetered, err := got.Phases[0].RateCards[0].Entitlement.AsBillingRateCardMeteredEntitlement()
	require.NoError(t, err, "rate card entitlement should be the metered variant")
	require.NotNil(t, rtMetered.Limit, "limit should round-trip")
	assert.Equal(t, float64(15_000_000), *rtMetered.Limit)

	// Publish so a subscription can reference it.
	status, _, problem = c.PublishPlan(created.Id)
	require.Equal(t, http.StatusOK, status, "publish plan: %+v", problem)

	// Customer with a usage-attributed subject (v3 has no separate subjects API).
	subjectKey := uniqueKey("ent_repro_subj")
	status, customer, problem := c.CreateCustomer(apiv3.CreateCustomerRequest{
		Key:  uniqueKey("ent_repro_cust"),
		Name: "Entitlement Repro Customer",
		UsageAttribution: &apiv3.BillingCustomerUsageAttribution{
			SubjectKeys: []apiv3.UsageAttributionSubjectKey{subjectKey},
		},
	})
	require.Equal(t, http.StatusCreated, status, "create customer: %+v", problem)
	require.NotNil(t, customer)

	// Subscribe to the published plan (v3 starts immediately; no timing on create).
	var subBody apiv3.BillingSubscriptionCreate
	subBody.Customer.Key = lo.ToPtr(customer.Key)
	subBody.Plan.Key = lo.ToPtr(planKey)
	subBody.Plan.Version = lo.ToPtr(1)

	status, sub, problem := c.CreateSubscription(subBody)
	require.Equal(t, http.StatusCreated, status, "create subscription: %+v", problem)
	require.NotNil(t, sub)

	// Authoritative check: the subscriber has a metered entitlement for the feature.
	status, access, problem := c.GetCustomerEntitlementAccess(customer.Id)
	require.Equal(t, http.StatusOK, status, "entitlement access: %+v", problem)
	require.NotNil(t, access)

	result, found := lo.Find(access.Data, func(r apiv3.BillingEntitlementAccessResult) bool {
		return r.FeatureKey == featureKey
	})
	require.True(t, found,
		"subscriber received no entitlement for feature %q: the entitlement on the v3 rate card was not materialized (api/v3/handlers/plans/convert.go FromAPIBillingRateCard)",
		featureKey)
	assert.True(t, result.HasAccess, "expected has_access=true for the metered entitlement")
}

// meteredEntitlementRateCard builds a flat, in-advance, feature-linked rate card
// carrying a metered entitlement template with the given per-period limit.
func meteredEntitlementRateCard(f apiv3.Feature, limit float64) apiv3.BillingRateCard {
	cadence := apiv3.ISO8601Duration("P1M")
	term := apiv3.BillingPricePaymentTermInAdvance

	price := apiv3.BillingPrice{}
	if err := price.FromBillingPriceFlat(apiv3.BillingPriceFlat{
		Type:   apiv3.BillingPriceFlatTypeFlat,
		Amount: "3000",
	}); err != nil {
		panic(err)
	}

	usagePeriod := apiv3.ISO8601Duration("P1M")
	entitlement := apiv3.BillingRateCardEntitlement{}
	if err := entitlement.FromBillingRateCardMeteredEntitlement(apiv3.BillingRateCardMeteredEntitlement{
		Type:        "metered",
		UsagePeriod: &usagePeriod,
		Limit:       lo.ToPtr(limit),
	}); err != nil {
		panic(err)
	}

	return apiv3.BillingRateCard{
		Key:            f.Key,
		Name:           "Data Tokens",
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
		Feature:        &apiv3.FeatureReference{Id: f.Id},
		Entitlement:    &entitlement,
	}
}

// TestV3V1StaticEntitlementConfigCrossVersion proves the v1 and v3 plan
// converters agree on the *stored domain representation* of a static
// entitlement config, by writing through one API version and reading through
// the other.
func TestV3V1StaticEntitlementConfigCrossVersion(t *testing.T) {
	v3 := newV3Client(t)
	v1 := initClient(t)

	// The canonical config object the client cares about. Compared with
	// assert.JSONEq throughout, so key ordering is irrelevant.
	const configJSON = `{"integrations":["github","slack"],"limit":10}`

	t.Run("write via v3 (JSON object) -> read via v1 (JSON string token)", func(t *testing.T) {
		featureKey := uniqueKey("xver_v3w_feat")
		status, feature, problem := v3.CreateFeature(apiv3.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Cross-version static (v3 write)",
		})
		require.Equal(t, http.StatusCreated, status, "create feature: %+v", problem)
		require.NotNil(t, feature)

		// v3 sends the config as a raw JSON object.
		var configObj map[string]any
		require.NoError(t, json.Unmarshal([]byte(configJSON), &configObj))

		ent := apiv3.BillingRateCardEntitlement{}
		require.NoError(t, ent.FromBillingRateCardStaticEntitlement(apiv3.BillingRateCardStaticEntitlement{
			Type:   "static",
			Config: configObj,
		}))

		status, created, problem := v3.CreatePlan(apiv3.CreatePlanRequest{
			Key:            uniqueKey("xver_v3w_plan"),
			Name:           "Cross-version static (v3 write)",
			Currency:       "USD",
			BillingCadence: apiv3.ISO8601Duration("P1M"),
			Phases: []apiv3.BillingPlanPhase{{
				Key:       "subscription",
				Name:      "Subscription",
				RateCards: []apiv3.BillingRateCard{staticRateCardV3(t, *feature, ent)},
			}},
		})
		require.Equal(t, http.StatusCreated, status, "create plan (v3): %+v", problem)
		require.NotNil(t, created)

		// Read the same plan back through the v1 SDK.
		getResp, err := v1.GetPlanWithResponse(t.Context(), created.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, getResp.StatusCode(), "get plan (v1): %s", string(getResp.Body))
		require.NotNil(t, getResp.JSON200)

		v1Config := v1StaticConfig(t, *getResp.JSON200)

		// The v1 contract is a JSON string token: the bytes must decode into a
		// Go string whose content is the original JSON object. (Under the old v3
		// write this would be a raw object and the unmarshal-into-string fails.)
		var inner string
		require.NoError(t, json.Unmarshal(v1Config, &inner),
			"v1 must return the static config as a JSON string token, got raw bytes %q", string(v1Config))
		assert.JSONEq(t, configJSON, inner, "decoded v1 string token must equal the config the v3 client sent")
	})

	t.Run("write via v1 (JSON string token) -> read via v3 (JSON object)", func(t *testing.T) {
		featureKey := uniqueKey("xver_v1w_feat")
		status, feature, problem := v3.CreateFeature(apiv3.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Cross-version static (v1 write)",
		})
		require.Equal(t, http.StatusCreated, status, "create feature: %+v", problem)
		require.NotNil(t, feature)

		// v1 sends the config as a JSON string token: a JSON string whose
		// content is the config object. json.Marshal of a Go string produces
		// exactly that quoted, escaped form.
		token := lo.Must(json.Marshal(configJSON))

		et := api.RateCardEntitlement{}
		require.NoError(t, et.FromRateCardStaticEntitlement(api.RateCardStaticEntitlement{
			Type:   api.RateCardStaticEntitlementTypeStatic,
			Config: json.RawMessage(token),
		}))

		rc := api.RateCard{}
		require.NoError(t, rc.FromRateCardFlatFee(api.RateCardFlatFee{
			Type:                api.RateCardFlatFeeTypeFlatFee,
			Key:                 feature.Key,
			Name:                "Cross-version static (v1 write)",
			FeatureKey:          lo.ToPtr(feature.Key),
			EntitlementTemplate: &et,
			BillingCadence:      lo.ToPtr("P1M"),
			Price: &api.FlatPriceWithPaymentTerm{
				Type:        api.FlatPriceWithPaymentTermTypeFlat,
				Amount:      "3000",
				PaymentTerm: lo.ToPtr(api.PricePaymentTermInAdvance),
			},
		}))

		createResp, err := v1.CreatePlanWithResponse(t.Context(), api.PlanCreate{
			Key:            uniqueKey("xver_v1w_plan"),
			Name:           "Cross-version static (v1 write)",
			Currency:       api.CurrencyCode("USD"),
			BillingCadence: "P1M",
			Phases: []api.PlanPhase{{
				Key:       "subscription",
				Name:      "Subscription",
				RateCards: []api.RateCard{rc},
			}},
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, createResp.StatusCode(), "create plan (v1): %s", string(createResp.Body))
		require.NotNil(t, createResp.JSON201)

		// Read the same plan back through the v3 API.
		status, got, problem := v3.GetPlan(createResp.JSON201.Id)
		require.Equal(t, http.StatusOK, status, "get plan (v3): %+v", problem)
		require.NotNil(t, got)
		require.Len(t, got.Phases, 1)
		require.Len(t, got.Phases[0].RateCards, 1)
		require.NotNil(t, got.Phases[0].RateCards[0].Entitlement, "entitlement should round-trip on v3 GET")

		static, err := got.Phases[0].RateCards[0].Entitlement.AsBillingRateCardStaticEntitlement()
		require.NoError(t, err, "rate card entitlement should be the static variant")

		// The v3 contract is a JSON object: the token must have been unwrapped.
		// (Under the old v3 read this would surface as a Go string instead.)
		_, isString := static.Config.(string)
		assert.False(t, isString, "v3 must return the static config as a JSON object, not a string token")

		raw, err := json.Marshal(static.Config)
		require.NoError(t, err)
		assert.JSONEq(t, configJSON, string(raw), "v3 object must equal the config the v1 client sent")
	})
}

// staticRateCardV3 builds a flat, in-advance, feature-linked v3 rate card
// carrying the given static entitlement template.
func staticRateCardV3(t *testing.T, f apiv3.Feature, ent apiv3.BillingRateCardEntitlement) apiv3.BillingRateCard {
	t.Helper()

	cadence := apiv3.ISO8601Duration("P1M")
	term := apiv3.BillingPricePaymentTermInAdvance

	price := apiv3.BillingPrice{}
	require.NoError(t, price.FromBillingPriceFlat(apiv3.BillingPriceFlat{
		Type:   apiv3.BillingPriceFlatTypeFlat,
		Amount: "3000",
	}))

	return apiv3.BillingRateCard{
		Key:            f.Key,
		Name:           "Cross-version static",
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
		Feature:        &apiv3.FeatureReference{Id: f.Id},
		Entitlement:    &ent,
	}
}

// v1StaticConfig pulls the static entitlement config bytes out of the single
// rate card of a v1 plan response.
func v1StaticConfig(t *testing.T, plan api.Plan) json.RawMessage {
	t.Helper()

	require.Len(t, plan.Phases, 1)
	require.Len(t, plan.Phases[0].RateCards, 1)

	flat, err := plan.Phases[0].RateCards[0].AsRateCardFlatFee()
	require.NoError(t, err, "rate card should be the flat-fee variant")
	require.NotNil(t, flat.EntitlementTemplate, "entitlement should round-trip on v1 GET")

	static, err := flat.EntitlementTemplate.AsRateCardStaticEntitlement()
	require.NoError(t, err, "entitlement should be the static variant")

	return static.Config
}
