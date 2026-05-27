package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apiv3 "github.com/openmeterio/openmeter/api/v3"
	sdkpkg "github.com/openmeterio/openmeter/api/spec/packages/aip-client-go"
	sdkerr "github.com/openmeterio/openmeter/api/spec/packages/aip-client-go/models/apierrors"
	sdkc "github.com/openmeterio/openmeter/api/spec/packages/aip-client-go/models/components"
	sdkop "github.com/openmeterio/openmeter/api/spec/packages/aip-client-go/models/operations"
)

// TestV3ProductCatalogSmokeSDK is the SDK-driven dogfood of the v3 plan +
// addon authoring flow, mirroring TestV3ProductCatalogSmoke but issuing
// every call through the generated Go SDK in
// api/spec/packages/aip-client-go.
//
// The fixture builders in v3helpers_test.go still produce apiv3.* types. We
// JSON-roundtrip them into the SDK's components.* types — the wire format is
// identical by construction (both are generated from the same TypeSpec), so
// any decode failure here is a signal that the SDK's typed model has drifted
// from the server schema.
func TestV3ProductCatalogSmokeSDK(t *testing.T) {
	sdk, ctx := newSDKClient(t)

	eventTypes := []string{
		uniqueKey("sanity_event"),
		uniqueKey("sanity_event"),
	}
	meterKeys := []string{
		uniqueKey("sanity_meter"),
		uniqueKey("sanity_meter"),
	}

	meters := make([]sdkc.Meter, 0, len(meterKeys))
	for i := range meterKeys {
		valueProperty := "$.value"
		req := sdkc.CreateMeterRequest{
			Key:           meterKeys[i],
			Name:          "Test Meter " + meterKeys[i],
			Aggregation:   sdkc.MeterAggregationSum,
			EventType:     eventTypes[i],
			ValueProperty: &valueProperty,
		}
		res, err := sdk.OpenMeterMeters.CreateMeter(ctx, sdkop.CreateMeterRequest{Meter: req})
		require.NoError(t, err, "CreateMeter")
		require.NotNil(t, res.Body, "CreateMeter response body")
		require.NotEmpty(t, res.Body.Id, "CreateMeter Id")
		meters = append(meters, *res.Body)
	}

	featureKeys := []string{
		uniqueKey("sanity_feature"),
		uniqueKey("sanity_feature"),
	}
	features := make([]sdkc.Feature, 0, len(featureKeys))
	for i := range featureKeys {
		req := sdkc.CreateFeatureRequest{
			Key:  featureKeys[i],
			Name: "Test Feature " + featureKeys[i],
			Meter: &sdkc.FeatureMeterReference{
				Id: meters[i].Id,
			},
		}
		res, err := sdk.OpenMeterFeatures.CreateFeature(ctx, sdkop.CreateFeatureRequest{Feature: req})
		require.NoError(t, err, "CreateFeature")
		require.NotNil(t, res.Body)
		require.NotEmpty(t, res.Body.Id)
		features = append(features, *res.Body)
	}

	var (
		planID      string
		addonID     string
		planAddonID string
		phaseKey    string
	)

	t.Run("Should create a draft plan with a single flat rate card", func(t *testing.T) {
		// Reuse the apiv3 fixture builder (wire-compatible) so the test
		// stays in sync with the canonical valid-plan shape.
		fixture := validPlanRequest("sanity_plan")
		phaseKey = fixture.Phases[0].Key

		body := toSDK[sdkc.CreatePlanRequest](t, fixture)

		res, err := sdk.OpenMeterPlans.CreatePlan(ctx, sdkop.CreatePlanRequest{Plan: body})
		require.NoError(t, err, "CreatePlan")
		require.NotNil(t, res.Body)
		assert.Equal(t, sdkc.BillingPlanStatusDraft, res.Body.Status)
		require.Len(t, res.Body.Phases, 1)
		require.Len(t, res.Body.Phases[0].RateCards, 1)

		planID = res.Body.Id
	})

	t.Run("Should update the plan to carry flat + usage + graduated rate cards", func(t *testing.T) {
		// Build the same three rate cards via the v3 fixture helpers, then
		// project the whole UpsertPlanRequest through JSON into the SDK
		// component type. Lets us test the SDK transport without
		// hand-coding the BillingRateCard union.
		flat := validFlatRateCard("sanity_flat")
		usage := validUnitRateCard(apiv3FeatureFromSDK(features[0]))
		graduated := validGraduatedRateCard(apiv3FeatureFromSDK(features[1]))

		apiUpdate := apiv3.UpsertPlanRequest{
			Name: "Sanity Plan",
			Phases: []apiv3.BillingPlanPhase{{
				Key:       phaseKey,
				Name:      "Sanity Phase",
				RateCards: []apiv3.BillingRateCard{flat, usage, graduated},
			}},
		}

		sdkUpdate := toSDK[sdkc.UpsertPlanRequest](t, apiUpdate)

		res, err := sdk.OpenMeterPlans.UpdatePlan(ctx, planID, sdkop.UpdatePlanRequest{Plan: sdkUpdate})
		require.NoError(t, err, "UpdatePlan")
		require.NotNil(t, res.Body)
		require.Len(t, res.Body.Phases, 1)
		assert.Len(t, res.Body.Phases[0].RateCards, 3, "expected three rate cards on the phase")
	})

	// Track the three valid rate cards (as raw JSON) across the
	// invalid-loop subtests so "remove defective" PUTs can rebuild the
	// phase from the same baseline using server-normalized values.
	var validRateCardsJSON []json.RawMessage

	t.Run("Should add a defective rate card and surface validation_errors", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)

		// Read the current valid rate cards back. The SDK gives us
		// strongly typed BillingRateCard values, but they're a sealed
		// union — we serialize them back to JSON and rebuild the
		// upsert body via JSON, which avoids re-implementing every
		// union variant against the SDK's typed API.
		getRes, err := sdk.OpenMeterPlans.GetPlan(ctx, planID)
		require.NoError(t, err, "GetPlan")
		require.NotNil(t, getRes.Body)
		require.Len(t, getRes.Body.Phases, 1)
		require.Len(t, getRes.Body.Phases[0].RateCards, 3)

		for _, rc := range getRes.Body.Phases[0].RateCards {
			b, err := json.Marshal(rc)
			require.NoError(t, err)
			validRateCardsJSON = append(validRateCardsJSON, b)
		}

		// Defect: a flat rate card whose billing_cadence (P2W) is
		// misaligned with the plan's P1M cadence.
		defective := validFlatRateCard("defective_cadence")
		misaligned := apiv3.ISO8601Duration("P2W")
		defective.BillingCadence = &misaligned

		// Build the upsert body as JSON to splice typed and untyped
		// rate cards together, then unmarshal into the SDK type.
		defectiveJSON, err := json.Marshal(defective)
		require.NoError(t, err)

		upsertJSON := map[string]any{
			"name": "Sanity Plan",
			"phases": []map[string]any{{
				"key":        phaseKey,
				"name":       "Sanity Phase",
				"rate_cards":  append(append([]json.RawMessage{}, validRateCardsJSON...), defectiveJSON),
			}},
		}
		var update sdkc.UpsertPlanRequest
		raw, err := json.Marshal(upsertJSON)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &update))

		_, err = sdk.OpenMeterPlans.UpdatePlan(ctx, planID, sdkop.UpdatePlanRequest{Plan: update})
		require.NoError(t, err, "update with defective rate card should accept the draft")

		// GET should now show validation_errors on the draft.
		got, err := sdk.OpenMeterPlans.GetPlan(ctx, planID)
		require.NoError(t, err)
		require.NotNil(t, got.Body)
		require.NotEmpty(t, got.Body.ValidationErrors, "expected validation_errors on the draft")

		codes := make([]string, 0, len(got.Body.ValidationErrors))
		for _, e := range got.Body.ValidationErrors {
			codes = append(codes, e.Code)
		}
		assert.Contains(t, codes, "rate_card_billing_cadence_unaligned")

		// Publish should reject with 400. The SDK's BadRequestError
		// doesn't expose extensions.validationErrors — see
		// https://github.com/openmeterio/openmeter — so we only
		// assert on the surface error here.
		_, err = sdk.OpenMeterPlans.PublishPlan(ctx, planID)
		require.Error(t, err, "publish should reject the defective draft")
		var bad *sdkerr.BadRequestError
		assert.True(t, errors.As(err, &bad), "expected BadRequestError, got %T: %v", err, err)
	})

	t.Run("Should remove the defective rate card and clear validation_errors", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, phaseKey)
		require.NotEmpty(t, validRateCardsJSON)

		upsertJSON := map[string]any{
			"name": "Sanity Plan",
			"phases": []map[string]any{{
				"key":       phaseKey,
				"name":      "Sanity Phase",
				"rate_cards": validRateCardsJSON,
			}},
		}
		var update sdkc.UpsertPlanRequest
		raw, err := json.Marshal(upsertJSON)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &update))

		res, err := sdk.OpenMeterPlans.UpdatePlan(ctx, planID, sdkop.UpdatePlanRequest{Plan: update})
		require.NoError(t, err, "update removing defective rate card")
		require.NotNil(t, res.Body)
		assert.Len(t, res.Body.Phases[0].RateCards, 3, "expected three rate cards after removal")

		got, err := sdk.OpenMeterPlans.GetPlan(ctx, planID)
		require.NoError(t, err)
		require.NotNil(t, got.Body)
		assert.Empty(t, got.Body.ValidationErrors, "expected validation_errors to clear after removal")
	})

	t.Run("Should create a draft addon", func(t *testing.T) {
		body := toSDK[sdkc.CreateAddonRequest](t, validAddonRequest("sanity_addon"))

		res, err := sdk.OpenMeterAddons.CreateAddon(ctx, sdkop.CreateAddonRequest{Addon: body})
		require.NoError(t, err, "CreateAddon")
		require.NotNil(t, res.Body)
		assert.Equal(t, sdkc.AddonStatusDraft, res.Body.Status)
		addonID = res.Body.Id
	})

	t.Run("Should publish the addon", func(t *testing.T) {
		require.NotEmpty(t, addonID)

		res, err := sdk.OpenMeterAddons.PublishAddon(ctx, addonID)
		require.NoError(t, err, "PublishAddon")
		require.NotNil(t, res.Body)
		assert.Equal(t, sdkc.AddonStatusActive, res.Body.Status)
	})

	t.Run("Should attach the published addon to the plan", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, addonID)
		require.NotEmpty(t, phaseKey)

		body := toSDK[sdkc.CreatePlanAddonRequest](t, validPlanAddonRequest(phaseKey, addonID))

		res, err := sdk.OpenMeterPlanAddon.CreatePlanAddon(ctx, planID, sdkop.CreatePlanAddonRequest{PlanAddon: body})
		require.NoError(t, err, "CreatePlanAddon")
		require.NotNil(t, res.Body)
		assert.Equal(t, addonID, res.Body.Addon.Id)
		assert.Equal(t, phaseKey, res.Body.FromPlanPhase)

		planAddonID = res.Body.Id
	})

	t.Run("Should publish the plan and keep the attached addon", func(t *testing.T) {
		require.NotEmpty(t, planID)
		require.NotEmpty(t, planAddonID)

		res, err := sdk.OpenMeterPlans.PublishPlan(ctx, planID)
		require.NoError(t, err, "PublishPlan")
		require.NotNil(t, res.Body)
		assert.Equal(t, sdkc.BillingPlanStatusActive, res.Body.Status)
		require.NotNil(t, res.Body.EffectiveFrom)

		// ListPlanAddonsResponse.Body is *map[string]any because the
		// SDK's pagination wrapper isn't typed yet. Walk the map.
		listRes, err := sdk.OpenMeterPlanAddon.ListPlanAddons(ctx, planID, sdkop.ListPlanAddonsRequest{})
		require.NoError(t, err, "ListPlanAddons")
		require.NotNil(t, listRes.Body)
		found := planAddonInPage(*listRes.Body, planAddonID)
		assert.True(t, found, "attached plan-addon missing after plan publish")
	})
}

// newSDKClient returns an SDK pointed at $OPENMETER_ADDRESS + /api/v3, and a
// context wired to the test's lifecycle. Skips the test when the env var is
// unset (same convention as newV3Client).
func newSDKClient(t *testing.T) (*sdkpkg.OpenMeter, context.Context) {
	t.Helper()

	address := os.Getenv("OPENMETER_ADDRESS")
	if address == "" {
		t.Skip("OPENMETER_ADDRESS not set")
	}

	base := strings.TrimRight(address, "/") + "/api/v3"
	sdk := sdkpkg.New(sdkpkg.WithServerURL(base))

	return sdk, t.Context()
}

// toSDK projects a wire-compatible source value into the SDK's typed
// component shape by JSON round-trip. Use it when the apiv3 fixture builders
// produce data that should match a particular components.* type.
func toSDK[T any](t *testing.T, src any) T {
	t.Helper()
	b, err := json.Marshal(src)
	require.NoError(t, err)
	var dst T
	require.NoError(t, json.Unmarshal(b, &dst))
	return dst
}

// apiv3FeatureFromSDK adapts an SDK Feature into the apiv3 shape so we can
// reuse the v3 fixture builders that take apiv3.Feature.
func apiv3FeatureFromSDK(f sdkc.Feature) apiv3.Feature {
	b, _ := json.Marshal(f)
	var out apiv3.Feature
	_ = json.Unmarshal(b, &out)
	return out
}

// planAddonInPage walks an untyped paginated response and returns true when
// the given plan-addon id appears under data[*].id. The SDK doesn't yet emit
// typed list responses (ListPlanAddonsResponse.Body is *map[string]any).
func planAddonInPage(page map[string]any, id string) bool {
	data, _ := page["data"].([]any)
	for _, row := range data {
		obj, _ := row.(map[string]any)
		if obj == nil {
			continue
		}
		if got, _ := obj["id"].(string); got == id {
			return true
		}
	}
	return false
}
