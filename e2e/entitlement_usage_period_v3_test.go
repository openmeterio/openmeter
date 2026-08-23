package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	v3sdk "github.com/openmeterio/openmeter/api/v3/client"
)

// A metered entitlement with no usage_period and no billing_cadence to default
// it from converts as-is; the domain validation rejects the zero usage period
// with entitlement_template_negative_usage_period. The rejection currently
// fires at create: the entitlement_template ent field validator runs
// EntitlementTemplate.Validate severity-blind at insert, so the draft is never
// stored (unlike rate-card-level warnings, which surface on the draft).
// Covers the plan and addon converters separately — they map entitlements
// independently.
func TestV3PlanMeteredEntitlementWithoutUsagePeriod(t *testing.T) {
	c := newV3Client(t)

	f := createMeteredFeature(t, c, "plan_ent_period")

	phase := validPlanPhase("ent_period_phase", true /* isLast */)
	phase.RateCards = []v3sdk.RateCardInput{meteredEntitlementRateCardWithoutPeriod(f)}

	body := validPlanRequest("ent_period")
	body.Phases = []v3sdk.PlanPhaseInput{phase}

	_, err := c.Plans.Create(t.Context(), body)
	problem := requireProblem(t, err, http.StatusBadRequest)
	assertValidationCode(t, problem, "entitlement_template_negative_usage_period")
}

func TestV3AddonMeteredEntitlementWithoutUsagePeriod(t *testing.T) {
	c := newV3Client(t)

	f := createMeteredFeature(t, c, "addon_ent_period")

	body := validAddonRequest("ent_period")
	body.RateCards = []v3sdk.RateCardInput{meteredEntitlementRateCardWithoutPeriod(f)}

	_, err := c.Addons.Create(t.Context(), body)
	problem := requireProblem(t, err, http.StatusBadRequest)
	assertValidationCode(t, problem, "entitlement_template_negative_usage_period")
}

// createMeteredFeature provisions a sum meter and a feature bound to it, the
// prerequisite for metered entitlement templates.
func createMeteredFeature(t *testing.T, c *v3Client, keyPrefix string) v3sdk.Feature {
	t.Helper()

	meterKey := uniqueKey(keyPrefix)

	m, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
		Key:           meterKey,
		Name:          "Test Meter " + meterKey,
		Aggregation:   v3sdk.MeterAggregationSum,
		EventType:     uniqueKey(keyPrefix + "_event"),
		ValueProperty: lo.ToPtr("$.value"),
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, m)

	featureKey := uniqueKey(keyPrefix)

	f, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
		Key:  featureKey,
		Name: "Test Feature " + featureKey,
		Meter: &v3sdk.FeatureMeterReferenceInput{
			ID: m.ID,
		},
	})
	c.requireStatus(http.StatusCreated, err)
	require.NotNil(t, f)

	return *f
}

// meteredEntitlementRateCardWithoutPeriod builds a flat rate card with no
// billing cadence whose metered entitlement omits usage_period — the usage
// period cannot be defaulted and stays zero.
func meteredEntitlementRateCardWithoutPeriod(f v3sdk.Feature) v3sdk.RateCardInput {
	term := v3sdk.PricePaymentTermInAdvance

	entitlement := lo.Must(v3sdk.RateCardEntitlementFromRateCardMeteredEntitlement(v3sdk.RateCardMeteredEntitlement{
		Limit: lo.ToPtr(float64(1000)),
	}))

	return v3sdk.RateCardInput{
		Key:         f.Key,
		Name:        "Test Entitlement Rate Card " + f.Key,
		Price:       lo.Must(v3sdk.PriceFromPriceFlat(v3sdk.PriceFlat{Amount: "10"})),
		PaymentTerm: &term,
		Feature:     &v3sdk.FeatureReference{ID: f.ID},
		Entitlement: &entitlement,
	}
}
