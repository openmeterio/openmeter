package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
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
	meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
		Key:           uniqueKey("data_tokens_meter"),
		Name:          "Data Tokens Meter",
		Aggregation:   v3sdk.MeterAggregationSum,
		EventType:     uniqueKey("data_tokens_event"),
		ValueProperty: &valueProperty,
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, meter)

	// Metered feature backed by that meter.
	featureKey := uniqueKey("data_tokens")
	feature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
		Key:   featureKey,
		Name:  "Data Tokens",
		Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, feature)

	// v3 plan with a metered entitlement on the feature-linked flat rate card.
	planKey := uniqueKey("ent_repro_plan")
	created, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
		Key:            planKey,
		Name:           "Dashboard Prepaid Plus",
		Currency:       "USD",
		BillingCadence: "P1M",
		Phases: []v3sdk.PlanPhaseInput{{
			Key:       "subscription",
			Name:      "Subscription",
			RateCards: []v3sdk.RateCardInput{meteredEntitlementRateCard(*feature, 15_000_000)},
		}},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, created)

	// The entitlement should round-trip on GET (read path wired through the converter).
	got, err := c.Plans.Get(t.Context(), created.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, got)
	require.Len(t, got.Phases, 1)
	require.Len(t, got.Phases[0].RateCards, 1)
	require.NotNil(t, got.Phases[0].RateCards[0].Entitlement, "entitlement should round-trip on GET")

	rtMetered, err := got.Phases[0].RateCards[0].Entitlement.AsRateCardMeteredEntitlement()
	require.NoError(t, err, "rate card entitlement should be the metered variant")
	require.NotNil(t, rtMetered.Limit, "limit should round-trip")
	assert.Equal(t, float64(15_000_000), *rtMetered.Limit)

	// Publish so a subscription can reference it.
	_, err = c.Plans.Publish(t.Context(), created.ID)
	c.requireStatus(http.StatusOK, err)

	// Customer with a usage-attributed subject (v3 has no separate subjects API).
	subjectKey := uniqueKey("ent_repro_subj")
	customer, err := c.Customers.Create(t.Context(), v3sdk.CreateCustomerRequest{
		Key:  uniqueKey("ent_repro_cust"),
		Name: "Entitlement Repro Customer",
		UsageAttribution: &v3sdk.CustomerUsageAttribution{
			SubjectKeys: []string{subjectKey},
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, customer)

	// Subscribe to the published plan (v3 starts immediately; no timing on create).
	var subBody v3sdk.SubscriptionCreate
	subBody.Customer.Key = lo.ToPtr(customer.Key)
	subBody.Plan.Key = lo.ToPtr(planKey)
	subBody.Plan.Version = lo.ToPtr(int64(1))

	sub, err := c.Subscriptions.Create(t.Context(), subBody)
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, sub)

	// Authoritative check: the subscriber has a metered entitlement for the feature.
	access, err := c.Entitlements.ListCustomerAccess(t.Context(), customer.ID)
	c.requireStatus(http.StatusOK, err)
	require.NotNil(t, access)

	result, found := lo.Find(access.Data, func(r v3sdk.EntitlementAccessResult) bool {
		return r.FeatureKey == featureKey
	})
	require.True(t, found,
		"subscriber received no entitlement for feature %q: the entitlement on the v3 rate card was not materialized (api/v3/handlers/plans/convert.go FromAPIBillingRateCard)",
		featureKey)
	assert.True(t, result.HasAccess, "expected has_access=true for the metered entitlement")
}

// meteredEntitlementRateCard builds a flat, in-advance, feature-linked rate card
// carrying a metered entitlement template with the given per-period limit.
func meteredEntitlementRateCard(f v3sdk.Feature, limit float64) v3sdk.RateCardInput {
	cadence := "P1M"
	term := v3sdk.PricePaymentTermInAdvance

	price := lo.Must(v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{
		Amount: "3000",
	}))

	usagePeriod := "P1M"
	entitlement := lo.Must(v3sdk.RateCardEntitlementFromRateCardMeteredEntitlement(v3sdk.RateCardMeteredEntitlement{
		UsagePeriod: &usagePeriod,
		Limit:       lo.ToPtr(limit),
	}))

	return v3sdk.RateCardInput{
		Key:            f.Key,
		Name:           "Data Tokens",
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
		Feature:        &v3sdk.FeatureReference{ID: f.ID},
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
		feature, err := v3.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Cross-version static (v3 write)",
		})
		v3.requireStatus(http.StatusCreated, err)
		require.NotNil(t, feature)

		// v3 sends the config as a raw JSON object.
		var configObj map[string]any
		require.NoError(t, json.Unmarshal([]byte(configJSON), &configObj))

		ent := lo.Must(v3sdk.RateCardEntitlementFromRateCardStaticEntitlement(v3sdk.RateCardStaticEntitlement{
			Config: configObj,
		}))

		created, err := v3.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            uniqueKey("xver_v3w_plan"),
			Name:           "Cross-version static (v3 write)",
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       "subscription",
				Name:      "Subscription",
				RateCards: []v3sdk.RateCardInput{staticRateCardV3(t, *feature, ent)},
			}},
		})
		v3.requireStatus(http.StatusCreated, err)
		require.NotNil(t, created)

		// Read the same plan back through the v1 SDK.
		getResp, err := v1.GetPlanWithResponse(t.Context(), created.ID, nil)
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
		feature, err := v3.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:  featureKey,
			Name: "Cross-version static (v1 write)",
		})
		v3.requireStatus(http.StatusCreated, err)
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
		got, err := v3.Plans.Get(t.Context(), createResp.JSON201.Id)
		v3.requireStatus(http.StatusOK, err)
		require.NotNil(t, got)
		require.Len(t, got.Phases, 1)
		require.Len(t, got.Phases[0].RateCards, 1)
		require.NotNil(t, got.Phases[0].RateCards[0].Entitlement, "entitlement should round-trip on v3 GET")

		static, err := got.Phases[0].RateCards[0].Entitlement.AsRateCardStaticEntitlement()
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
func staticRateCardV3(t *testing.T, f v3sdk.Feature, ent v3sdk.RateCardEntitlement) v3sdk.RateCardInput {
	t.Helper()

	cadence := "P1M"
	term := v3sdk.PricePaymentTermInAdvance

	price := lo.Must(v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{
		Amount: "3000",
	}))

	return v3sdk.RateCardInput{
		Key:            f.Key,
		Name:           "Cross-version static",
		Price:          price,
		BillingCadence: &cadence,
		PaymentTerm:    &term,
		Feature:        &v3sdk.FeatureReference{ID: f.ID},
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
