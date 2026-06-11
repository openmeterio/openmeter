package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
// The fix adds the `entitlement` field (V2-aligned: structured `issue`) to the
// v3 schema and wires it through the converter. This test asserts the corrected
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
		Feature:        &apiv3.FeatureReferenceItem{Id: f.Id},
		Entitlement:    &entitlement,
	}
}
