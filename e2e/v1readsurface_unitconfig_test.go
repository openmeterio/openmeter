package e2e

import (
	"net/http"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
	apiv3 "github.com/openmeterio/openmeter/api/v3"
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

	var ucPlan *apiv3.BillingPlan
	var plainPlan *apiv3.BillingPlan

	// given:
	// - a unit_config plan (usage-based rate card, divide-by-1000 ceiling) and a plain plan, both
	//   authored and published via v3 (v1 cannot author unit_config).
	runRequired(t, "creates a unit_config plan and a plain plan", func(t *testing.T) {
		status, meter, problem := c.CreateMeter(apiv3.CreateMeterRequest{
			Key:           meterKey,
			Name:          "UC Read Meter " + uniq,
			Aggregation:   apiv3.MeterAggregationSum,
			EventType:     eventType,
			ValueProperty: lo.ToPtr("$.value"),
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, meter)

		status, feature, problem := c.CreateFeature(apiv3.CreateFeatureRequest{
			Key:   featureKey,
			Name:  "UC Read Feature " + uniq,
			Meter: &apiv3.FeatureMeterReference{Id: meter.Id},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, feature)

		cadence := apiv3.ISO8601Duration("P1M")
		term := apiv3.BillingPricePaymentTermInArrears
		price := apiv3.BillingPrice{}
		require.NoError(t, price.FromBillingPriceUnit(apiv3.BillingPriceUnit{
			Type:   apiv3.BillingPriceUnitTypeUnit,
			Amount: "0.10",
		}))
		ucRateCard := apiv3.BillingRateCard{
			Key:            feature.Key,
			Name:           "UC Read Rate Card " + uniq,
			Price:          price,
			BillingCadence: &cadence,
			PaymentTerm:    &term,
			Feature:        &apiv3.FeatureReference{Id: feature.Id},
			UnitConfig: &apiv3.BillingUnitConfig{
				Operation:        apiv3.BillingUnitConfigOperationDivide,
				ConversionFactor: "1000",
				Rounding:         lo.ToPtr(apiv3.BillingUnitConfigRoundingModeCeiling),
				Precision:        lo.ToPtr(0),
			},
		}

		status, createdUC, problem := c.CreatePlan(apiv3.CreatePlanRequest{
			Key:            ucPlanKey,
			Name:           "UC Read Plan " + uniq,
			Currency:       "USD",
			BillingCadence: apiv3.ISO8601Duration("P1M"),
			Phases: []apiv3.BillingPlanPhase{{
				Key:       "phase_1",
				Name:      "UC Phase",
				RateCards: []apiv3.BillingRateCard{ucRateCard},
			}},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, createdUC)
		status, ucPlan, problem = c.PublishPlan(createdUC.Id)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, ucPlan)

		status, createdPlain, problem := c.CreatePlan(apiv3.CreatePlanRequest{
			Key:            plainPlanKey,
			Name:           "Plain Read Plan " + uniq,
			Currency:       "USD",
			BillingCadence: apiv3.ISO8601Duration("P1M"),
			Phases:         []apiv3.BillingPlanPhase{validPlanPhase("plain_phase", true /* isLast */)},
		})
		require.Equal(t, http.StatusCreated, status, "problem: %+v", problem)
		require.NotNil(t, createdPlain)
		status, plainPlan, problem = c.PublishPlan(createdPlain.Id)
		require.Equal(t, http.StatusOK, status, "problem: %+v", problem)
		require.NotNil(t, plainPlan)
	})

	// then:
	// - v1 GET on the unit_config plan is rejected with the typed code (not a silently-stripped 200).
	runRequired(t, "v1 plan GET rejects the unit_config plan", func(t *testing.T) {
		resp, err := v1.GetPlanWithResponse(t.Context(), ucPlan.Id, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode(), "body: %s", string(resp.Body))
		require.NotNil(t, resp.ApplicationproblemJSON400)
		assert.Contains(t, string(resp.Body), unitConfigNotRepresentableCode, "the 400 must carry the typed unit_config code")
	})

	// and:
	// - the plain plan is still gettable via v1 (the exclusion is content-derived, not a blanket block).
	runRequired(t, "v1 plan GET returns the plain plan", func(t *testing.T) {
		resp, err := v1.GetPlanWithResponse(t.Context(), plainPlan.Id, nil)
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
