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

const unitConfigNotRepresentableCode = "unit_config_not_representable"

// Flow:
//   - v1 plan GET on a unit_config plan → 400 with the typed code (not a stripped 200)
//   - v1 plan GET on a plain plan → 200 (exclusion is content-derived, not blanket)
//   - v1 plan LIST omits the unit_config plan, keeps the plain one, and reports an exact TotalCount
//   - v1 subscribe-by-plan-key from the unit_config plan still succeeds (read ≠ subscribe; OM-399)
//   - v1 subscription GET on that subscription → 400 with the same typed code
func TestV1ReadSurfaceExcludesUnitConfig(t *testing.T) {
	c := newV3Client(t)
	v1 := initClient(t)

	uniq := uniqueKey("ucread")
	meterKey := "ucread_meter_" + uniq
	eventType := "ucread_event_" + uniq
	featureKey := "ucread_feature_" + uniq
	customerKey := "ucread_customer_" + uniq
	subjectKey := "ucread_subject_" + uniq
	ucPlanKey := "ucread_uc_plan_" + uniq
	plainPlanKey := "ucread_plain_plan_" + uniq

	var ucPlan *v3sdk.Plan
	var plainPlan *v3sdk.Plan

	// given:
	// - a unit_config plan (usage-based rate card, divide-by-1000 ceiling) and a plain plan, both
	//   authored and published via v3 (v1 cannot author unit_config).
	runRequired(t, "creates a unit_config plan and a plain plan", func(t *testing.T) {
		meter, err := c.Meters.Create(t.Context(), v3sdk.CreateMeterRequest{
			Key:           meterKey,
			Name:          "UC Read Meter " + uniq,
			Aggregation:   v3sdk.MeterAggregationSum,
			EventType:     eventType,
			ValueProperty: lo.ToPtr("$.value"),
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, meter)

		feature, err := c.Features.Create(t.Context(), v3sdk.CreateFeatureRequest{
			Key:   featureKey,
			Name:  "UC Read Feature " + uniq,
			Meter: &v3sdk.FeatureMeterReferenceInput{ID: meter.ID},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, feature)

		cadence := "P1M"
		term := v3sdk.PricePaymentTermInArrears
		price := lo.Must(v3sdk.PriceFromPriceUnit(v3sdk.PriceUnit{
			Amount: "0.10",
		}))
		ucRateCard := v3sdk.RateCardInput{
			Key:            feature.Key,
			Name:           "UC Read Rate Card " + uniq,
			Price:          price,
			BillingCadence: &cadence,
			PaymentTerm:    &term,
			Feature:        &v3sdk.FeatureReference{ID: feature.ID},
			UnitConfig: &v3sdk.UnitConfig{
				Operation:        v3sdk.UnitConfigOperationDivide,
				ConversionFactor: "1000",
				Rounding:         lo.ToPtr(v3sdk.UnitConfigRoundingModeCeiling),
				Precision:        lo.ToPtr(int64(0)),
			},
		}

		createdUC, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            ucPlanKey,
			Name:           "UC Read Plan " + uniq,
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases: []v3sdk.PlanPhaseInput{{
				Key:       "phase_1",
				Name:      "UC Phase",
				RateCards: []v3sdk.RateCardInput{ucRateCard},
			}},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, createdUC)
		ucPlan, err = c.Plans.Publish(t.Context(), createdUC.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, ucPlan)

		createdPlain, err := c.Plans.Create(t.Context(), v3sdk.CreatePlanRequest{
			Key:            plainPlanKey,
			Name:           "Plain Read Plan " + uniq,
			Currency:       "USD",
			BillingCadence: "P1M",
			Phases:         []v3sdk.PlanPhaseInput{validPlanPhase("plain_phase", true /* isLast */)},
		})
		c.requireStatus(http.StatusCreated, err)
		require.NotNil(t, createdPlain)
		plainPlan, err = c.Plans.Publish(t.Context(), createdPlain.ID)
		c.requireStatus(http.StatusOK, err)
		require.NotNil(t, plainPlan)
	})

	// then:
	// - v1 GET on the unit_config plan is rejected with the typed code (not a silently-stripped 200).
	runRequired(t, "v1 plan GET rejects the unit_config plan", func(t *testing.T) {
		resp, err := v1.GetPlanWithResponse(t.Context(), ucPlan.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", string(resp.Body))
		require.NotNil(t, resp.ApplicationproblemJSON400)
		assert.Contains(t, string(resp.Body), unitConfigNotRepresentableCode, "the 400 must carry the typed unit_config code")
	})

	// and:
	// - the plain plan is still gettable via v1 (the exclusion is content-derived, not a blanket block).
	runRequired(t, "v1 plan GET returns the plain plan", func(t *testing.T) {
		resp, err := v1.GetPlanWithResponse(t.Context(), plainPlan.ID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", string(resp.Body))
		require.NotNil(t, resp.JSON200)
		assert.Equal(t, plainPlanKey, resp.JSON200.Key)
	})

	// and:
	// - v1 LIST omits the unit_config plan, keeps the plain one, and TotalCount reflects the filtered set
	//   (the exclusion runs at the query layer, before the COUNT — so the count stays exact).
	runRequired(t, "v1 plan LIST excludes the unit_config plan with an exact TotalCount", func(t *testing.T) {
		resp, err := v1.ListPlansWithResponse(t.Context(), &api.ListPlansParams{
			Key:      &[]string{ucPlanKey, plainPlanKey},
			PageSize: lo.ToPtr(api.PaginationPageSize(1000)),
		})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode(), "body: %s", string(resp.Body))
		require.NotNil(t, resp.JSON200)

		keys := lo.Map(resp.JSON200.Items, func(p api.Plan, _ int) string { return p.Key })
		assert.NotContains(t, keys, ucPlanKey, "unit_config plan must be excluded from the v1 list")
		assert.Contains(t, keys, plainPlanKey, "plain plan must remain in the v1 list")
		assert.Equal(t, 1, resp.JSON200.TotalCount, "TotalCount must reflect the filtered set, not the raw count")
	})

	// and:
	// - a v1 subscription created from the unit_config plan BY KEY still succeeds (read ≠ subscribe; the
	//   server rates it correctly per OM-399, only the read surfaces are restricted).
	var subscriptionID string
	runRequired(t, "v1 subscribe-by-plan-key from the unit_config plan succeeds", func(t *testing.T) {
		customer := CreateCustomerWithSubject(t, v1, customerKey, subjectKey)
		require.NotNil(t, customer)

		timing := &api.SubscriptionTiming{}
		require.NoError(t, timing.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

		body := api.SubscriptionCreate{}
		require.NoError(t, body.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
			Timing:     timing,
			CustomerId: lo.ToPtr(customer.Id),
			Plan: api.PlanReferenceInput{
				Key:     ucPlanKey,
				Version: lo.ToPtr(1),
			},
		}))

		resp, err := v1.CreateSubscriptionWithResponse(t.Context(), body)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode(), "body: %s", string(resp.Body))
		require.NotNil(t, resp.JSON201)
		subscriptionID = resp.JSON201.Id
	})

	// then:
	// - v1 GET on that subscription is rejected with the same typed code (its items carry the unit_config).
	runRequired(t, "v1 subscription GET rejects the unit_config subscription", func(t *testing.T) {
		require.NotEmpty(t, subscriptionID)
		resp, err := v1.GetSubscriptionWithResponse(t.Context(), subscriptionID, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", string(resp.Body))
		require.NotNil(t, resp.ApplicationproblemJSON400)
		assert.Contains(t, string(resp.Body), unitConfigNotRepresentableCode, "the 400 must carry the typed unit_config code")
	})
}
