package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
)

var (
	MultiSubjectFeatureKey        = "multi_subject_feature_1"
	MultiSubjectMeteredFeatureKey = "multi_subject_metered_feature_1"
	MultiSubjectPlanKey           = "multi_subject_plan"
	MultiSubjectMeterSlug         = "multi_subject_meter"
)

func TestMultiSubject(t *testing.T) {
	client := initClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var customerId string
	subjectKeys := []string{"multi_subject_1", "multi_subject_2"}

	// Let's set up a customer with multiple subjects
	t.Run("Should create a customer with multiple subjects", func(t *testing.T) {
		customerAPIRes, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
			Name:         "Test Customer Multi Subject",
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			Description:  lo.ToPtr("Test Customer Description"),
			PrimaryEmail: lo.ToPtr("customer-multi-subject@mail.com"),
			BillingAddress: &api.Address{
				City:        lo.ToPtr("City"),
				Country:     lo.ToPtr("US"),
				Line1:       lo.ToPtr("Line 1"),
				Line2:       lo.ToPtr("Line 2"),
				State:       lo.ToPtr("State"),
				PhoneNumber: lo.ToPtr("1234567890"),
				PostalCode:  lo.ToPtr("12345"),
			},
			UsageAttribution: &api.CustomerUsageAttribution{
				SubjectKeys: subjectKeys,
			},
		})
		require.Nil(t, err)

		require.Equal(t, 201, customerAPIRes.StatusCode(), "received the following body: %s", customerAPIRes.Body)
		customer1 := customerAPIRes.JSON201
		require.NotNil(t, customer1)
		require.Len(t, customer1.UsageAttribution.SubjectKeys, 2)

		customerId = customer1.Id
		require.NotEmpty(t, customerId)
	})

	var planId string

	// Now let's set up a plan (with no billables for simplicity)
	t.Run("Should be able to set up a plan as normal", func(t *testing.T) {
		// Now, let's create dedicated features for the plan
		featureAPIRes, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Key:  MultiSubjectFeatureKey,
			Name: "Test Plan Feature",
		})
		require.Nil(t, err)

		feature := featureAPIRes.JSON201
		require.NotNil(t, feature)

		// And let's create a dedicated metered feature
		meteredFeatureAPIRes, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
			Key:       MultiSubjectMeteredFeatureKey,
			Name:      "Test Plan Metered Feature",
			MeterSlug: lo.ToPtr(MultiSubjectMeterSlug),
		})
		require.Nil(t, err)

		meteredFeature := meteredFeatureAPIRes.JSON201
		require.NotNil(t, meteredFeature)

		et := &api.RateCardEntitlement{}
		err = et.FromRateCardBooleanEntitlement(api.RateCardBooleanEntitlement{
			Type: api.RateCardBooleanEntitlementType("boolean"),
		})
		require.Nil(t, err)

		met := &api.RateCardEntitlement{}
		err = met.FromRateCardMeteredEntitlement(api.RateCardMeteredEntitlement{
			Type:            api.RateCardMeteredEntitlementType("metered"),
			IssueAfterReset: lo.ToPtr(100.0),
		})
		require.Nil(t, err)

		// Lets build a PlanCreate input
		p1RC1 := api.RateCard{}
		err = p1RC1.FromRateCardFlatFee(api.RateCardFlatFee{
			Name:        "Simple Feature",
			Description: lo.ToPtr("Just a ratecard with an entitlement"),
			Key:         MultiSubjectFeatureKey,
			FeatureKey:  lo.ToPtr(MultiSubjectFeatureKey),
			TaxConfig: &api.TaxConfig{
				Stripe: &api.StripeTaxConfig{
					Code: "txcd_10000000",
				},
			},
			EntitlementTemplate: et,
			BillingCadence:      nil,
			Type:                api.RateCardFlatFeeType("flat"),
		})
		require.Nil(t, err)

		p1RC2 := api.RateCard{}
		err = p1RC2.FromRateCardFlatFee(api.RateCardFlatFee{
			Name:                "Metered Feature",
			Description:         lo.ToPtr("Has a monthly recurring price to grant access to a feature"),
			Key:                 MultiSubjectMeteredFeatureKey,
			FeatureKey:          lo.ToPtr(MultiSubjectMeteredFeatureKey),
			EntitlementTemplate: met,
			TaxConfig: &api.TaxConfig{
				Stripe: &api.StripeTaxConfig{
					Code: "txcd_10000000",
				},
			},
			Price: &api.FlatPriceWithPaymentTerm{
				Amount:      "1000",
				PaymentTerm: lo.ToPtr(api.PricePaymentTerm("in_advance")),
				Type:        api.FlatPriceWithPaymentTermType("flat"),
			},
			BillingCadence: lo.ToPtr("P1M"),
			Type:           api.RateCardFlatFeeType("flat"),
		})
		require.Nil(t, err)

		planCreate := api.PlanCreate{
			Currency:       api.CurrencyCode("USD"),
			Name:           "Test Plan",
			Description:    lo.ToPtr("Test Plan Description"),
			Key:            MultiSubjectPlanKey,
			BillingCadence: "P1M",
			Alignment: &api.Alignment{
				BillablesMustAlign: lo.ToPtr(true),
			},
			Phases: []api.PlanPhase{
				{
					Name:        "Test Plan Phase 1",
					Key:         "test_plan_phase_1",
					Description: lo.ToPtr("Test Plan Phase 1 Description"),
					Duration:    nil,
					RateCards:   []api.RateCard{p1RC1, p1RC2},
				},
			},
		}

		planAPIRes, err := client.CreatePlanWithResponse(ctx, planCreate)
		require.Nil(t, err)
		require.Equal(t, 201, planAPIRes.StatusCode())

		plan := planAPIRes.JSON201
		require.NotNil(t, plan, "received the following body: %s", planAPIRes.Body)

		assert.Equal(t, MultiSubjectPlanKey, plan.Key)
		require.NotNil(t, plan.Version)
		assert.Equal(t, 1, plan.Version)

		require.NotNil(t, plan.Id)
		planId = plan.Id

		require.NotEmpty(t, planId)
		apiRes, err := client.PublishPlanWithResponse(ctx, planId)
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		body := apiRes.JSON200
		require.NotNil(t, body)
	})

	t.Run("Should be able to create a subscription for customer with multiple subjects", func(t *testing.T) {
		immediate := &api.SubscriptionTiming{}
		require.NoError(t, immediate.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

		req := api.CreateSubscriptionJSONRequestBody{}
		require.NoError(t, req.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
			CustomerId: lo.ToPtr(customerId),
			Plan: api.PlanReferenceInput{
				Key:     MultiSubjectPlanKey,
				Version: lo.ToPtr(1),
			},
			Timing: immediate,
			Name:   lo.ToPtr("Test Subscription Multi Subject"),
		}))

		apiRes, err := client.CreateSubscriptionWithResponse(ctx, req)
		require.Nil(t, err)
		require.Equal(t, 201, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)
	})

	t.Run("Should be able to check entitlement access through each subject", func(t *testing.T) {
		for _, subjectKey := range subjectKeys {
			apiRes, err := client.GetEntitlementValueWithResponse(ctx, subjectKey, MultiSubjectFeatureKey, nil)

			require.Nil(t, err)
			require.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

			body := apiRes.JSON200
			require.NotNil(t, body)
			require.True(t, body.HasAccess)
		}
	})

	t.Run("Should aggregate usage across all subjects", func(t *testing.T) {
		// Let's ingest usage for each subject
		// Make clickhouse's job easier by sending events within a fix time range
		now := time.Now()

		for _, subjectKey := range subjectKeys {
			timestamp := gofakeit.DateRange(now, now.Add(time.Second*2))

			ev := cloudevents.New()
			ev.SetID(gofakeit.UUID())
			ev.SetSource("my-app")
			ev.SetType("multi_subject")
			ev.SetSubject(subjectKey)
			ev.SetTime(timestamp)

			resp, err := client.IngestEventWithResponse(context.Background(), ev)
			require.NoError(t, err)
			require.Equal(t, http.StatusNoContent, resp.StatusCode())
		}

		// Wait for events to be processed
		assert.EventuallyWithT(t, func(t *assert.CollectT) {
			resp, err := client.QueryMeterWithResponse(context.Background(), MultiSubjectMeterSlug, nil)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode())

			require.Len(t, resp.JSON200.Data, 1)
			assert.Equal(t, float64(len(subjectKeys)), resp.JSON200.Data[0].Value)
		}, time.Minute, time.Second)

		entResp, err := client.GetCustomerEntitlementValueV2WithResponse(ctx, customerId, MultiSubjectMeteredFeatureKey, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, entResp.StatusCode())

		ent := entResp.JSON200
		require.NotNil(t, ent)
		require.True(t, ent.HasAccess)
		require.NotNil(t, ent.Usage)
		require.Equal(t, float64(len(subjectKeys)), *ent.Usage)
	})
}
