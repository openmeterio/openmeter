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

// TestV3PlanRateCardEntitlementTemplateRepro covers configuring an entitlement
// on a feature-linked rate card via the v3 POST /plans endpoint, and asserts a
// subscriber then receives the metered entitlement.
//
// It started life as a reproduction of a reported defect: the v3 BillingRateCard
// schema had no entitlement field, and api/v3/handlers/plans/convert.go's
// FromAPIBillingRateCard never mapped one — so the config could not be set at
// all. Before the fix the behaviour differed by deployment:
//   - Local `make server`: the v3 OpenAPI request validator rejected the unknown
//     property outright with 400 — `property "entitlement" is unsupported`.
//   - Kong Konnect (as reported): the gateway stripped unknown properties before
//     forwarding, so OpenMeter returned 201 and silently dropped the entitlement.
//
// The fix adds the `entitlement` field (V2-aligned: structured `issue`) to the
// v3 schema and wires it through the converter. This test asserts the corrected
// end-to-end behaviour: create is accepted (201) and the metered entitlement
// materializes on subscription, surfacing via GET customer access.
//
// The full flow mirrors the issue:
//
//	create metered feature
//	-> POST /plans (v3, entitlement on rate card) -> 201
//	-> POST /plans/{id}/publish
//	-> create customer + subject
//	-> POST /subscriptions { customer, plan, timing: immediate }
//	-> GET customer access -> expect metered entitlement with has_access=true
//
// Only the product-catalog + subscription path is exercised, so a plain
// `make server` (no billing/sink workers) is sufficient. The feature is linked
// to the `tokens_total` meter defined in the local config.yaml.
func TestV3PlanRateCardEntitlementTemplateRepro(t *testing.T) {
	sdk := initClient(t)  // v1 generated SDK: features, customers, subscriptions, access
	c := newV3Client(t)   // v3 raw HTTP: plan create/get/publish
	ctx := t.Context()

	featureKey := uniqueKey("data_tokens")
	planKey := uniqueKey("ent_repro_plan")

	// Metered feature backed by the tokens_total meter (local config.yaml).
	featResp, err := sdk.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
		Key:       featureKey,
		Name:      "Data Tokens",
		MeterSlug: lo.ToPtr("tokens_total"),
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, featResp.StatusCode(), "create feature: %s", featResp.Body)
	feature := featResp.JSON201
	require.NotNil(t, feature)

	// v3 plan create, built as raw JSON exactly like the reported payload:
	// snake_case keys with entitlement_template on the feature-linked rate card.
	// The typed apiv3.BillingRateCard struct can't carry this field, which is
	// itself the bug — so we send a raw map to reproduce the wire request.
	planBody := map[string]any{
		"key":                planKey,
		"name":               "Dashboard Prepaid Plus",
		"currency":           "USD",
		"billing_cadence":    "P1M",
		"pro_rating_enabled": true,
		"phases": []any{
			map[string]any{
				"key":  "subscription",
				"name": "Subscription",
				"rate_cards": []any{
					map[string]any{
						"key":             featureKey,
						"name":            "Data Tokens",
						"billing_cadence": "P1M",
						"feature":         map[string]any{"id": feature.Id},
						"payment_term":    "in_advance",
						"price":           map[string]any{"type": "flat", "amount": "3000"},
						"entitlement": map[string]any{
							"type":                      "metered",
							"usage_period":              "P1M",
							"issue":                     map[string]any{"amount": 15000000, "priority": 1},
							"is_soft_limit":             false,
							"preserve_overage_at_reset": false,
						},
					},
				},
			},
		},
	}

	createStatus, createRaw, createProblem := c.do(http.MethodPost, "/plans", planBody)
	// EXPECTED: 201 with the entitlement persisted on the rate card.
	// Before the fix (no `entitlement` field in the v3 schema), the OpenAPI
	// request validator rejected this with 400 "property \"entitlement\" is
	// unsupported"; Kong Konnect instead stripped it and returned a silent 201.
	require.Equal(t, http.StatusCreated, createStatus, "create plan: %s / %+v", createRaw, createProblem)

	var created apiv3.BillingPlan
	require.NoError(t, json.Unmarshal(createRaw, &created), "decode created plan: %s", createRaw)
	require.NotEmpty(t, created.Id)

	// Diagnostic: with the read path wired (ToAPIBillingRateCardEntitlement), the
	// configured entitlement now round-trips on GET. Logged rather than
	// hard-asserted; the authoritative check below is subscription access.
	getStatus, getRaw, getProblem := c.do(http.MethodGet, "/plans/"+created.Id, nil)
	require.Equal(t, http.StatusOK, getStatus, "get plan: %s / %+v", getRaw, getProblem)
	t.Logf("GET /plans/{id} body (rate card should echo the entitlement): %s", getRaw)

	// Publish so a subscription can reference it.
	pubStatus, pubRaw, pubProblem := c.do(http.MethodPost, "/plans/"+created.Id+"/publish", nil)
	require.Equal(t, http.StatusOK, pubStatus, "publish plan: %s / %+v", pubRaw, pubProblem)

	// Customer + subject, then an immediate subscription to the published plan.
	customerKey := uniqueKey("ent_repro_cust")
	customer := CreateCustomerWithSubject(t, sdk, customerKey, customerKey)

	timing := &api.SubscriptionTiming{}
	require.NoError(t, timing.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

	subCreate := api.SubscriptionCreate{}
	require.NoError(t, subCreate.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
		Timing:     timing,
		CustomerId: &customer.Id,
		Name:       lo.ToPtr("Entitlement Repro Subscription"),
		Plan: api.PlanReferenceInput{
			Key:     planKey,
			Version: lo.ToPtr(1),
		},
	}))

	subResp, err := sdk.CreateSubscriptionWithResponse(ctx, subCreate)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, subResp.StatusCode(), "create subscription: %s", subResp.Body)

	// Authoritative runtime check: the subscriber should have a metered
	// entitlement for the feature. On the current server this map is empty,
	// reproducing the reported {"data": []}.
	accessResp, err := sdk.GetCustomerAccessWithResponse(ctx, customer.Id)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, accessResp.StatusCode(), "customer access: %s", accessResp.Body)
	require.NotNil(t, accessResp.JSON200)
	t.Logf("GET customer access entitlements: %s", accessResp.Body)

	ent, ok := accessResp.JSON200.Entitlements[featureKey]
	require.True(t, ok,
		"subscriber received no entitlement for feature %q: the entitlement on the v3 rate card was not materialized (api/v3/handlers/plans/convert.go FromAPIBillingRateCard)",
		featureKey)
	assert.True(t, ent.HasAccess, "expected has_access=true for the metered entitlement")
}
