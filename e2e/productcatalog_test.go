package e2e

import (
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	api "github.com/openmeterio/openmeter/api/client/go"
)

const (
	PlanFeatureKey        = "plan_feature_1"
	PlanMeteredFeatureKey = "plan_metered_feature_1"
	PlanKey               = "test_plan"
)

func TestPlan(t *testing.T) {
	client := initClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Let's set up a customer
	customerAPIRes, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
		Name:         "Test Customer",
		Currency:     lo.ToPtr(api.CurrencyCode("USD")),
		Description:  lo.ToPtr("Test Customer Description"),
		PrimaryEmail: lo.ToPtr("customer@mail.com"),
		Timezone:     lo.ToPtr(time.UTC.String()),
		BillingAddress: &api.Address{
			City:        lo.ToPtr("City"),
			Country:     lo.ToPtr("US"),
			Line1:       lo.ToPtr("Line 1"),
			Line2:       lo.ToPtr("Line 2"),
			State:       lo.ToPtr("State"),
			PhoneNumber: lo.ToPtr("1234567890"),
			PostalCode:  lo.ToPtr("12345"),
		},
		UsageAttribution: api.CustomerUsageAttribution{
			SubjectKeys: []string{"test_customer_subject"},
		},
	})
	require.Nil(t, err)

	customer := customerAPIRes.JSON201
	require.NotNil(t, customer)

	// Now, let's create dedicated features for the plan
	featureAPIRes, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
		Key:  PlanFeatureKey,
		Name: "Test Plan Feature",
	})
	require.Nil(t, err)

	feature := featureAPIRes.JSON201
	require.NotNil(t, feature)

	meteredFeatureAPIRes, err := client.CreateFeatureWithResponse(ctx, api.CreateFeatureJSONRequestBody{
		Key:       PlanMeteredFeatureKey,
		Name:      "Test Plan Metered Feature",
		MeterSlug: lo.ToPtr("plan_meter"),
	})
	require.Nil(t, err)

	meteredFeature := meteredFeatureAPIRes.JSON201
	require.NotNil(t, meteredFeature)

	var planId string

	t.Run("Should create a plan on happy path", func(t *testing.T) {
		p1RC1 := api.RateCard{}
		err := p1RC1.FromRateCardFlatFee(api.RateCardFlatFee{
			Name:        "Test Plan Phase 1 Rate Card 1",
			Description: lo.ToPtr("Has a one time flat price like an installation fee"),
			Key:         "test_plan_phase_1_rate_card_1",
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
			BillingCadence: nil,
			Type:           api.RateCardFlatFeeType("flat"),
		})
		require.Nil(t, err)

		p1RC2 := api.RateCard{}
		err = p1RC2.FromRateCardFlatFee(api.RateCardFlatFee{
			Name:        "Test Plan Phase 1 Rate Card 2",
			Description: lo.ToPtr("Has a monthly recurring price to grant access to a feature"),
			Key:         PlanFeatureKey,
			FeatureKey:  lo.ToPtr(PlanFeatureKey),
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

		p2RC1 := api.RateCard{}
		err = p2RC1.FromRateCardFlatFee(api.RateCardFlatFee{
			Name:        "Test Plan Phase 2 Rate Card 1",
			Description: lo.ToPtr("Keeps access to the same feature as in phase 1"),
			Key:         PlanFeatureKey,
			FeatureKey:  lo.ToPtr(PlanFeatureKey),
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

		p2RC2P := api.RateCardUsageBasedPrice{}
		err = p2RC2P.FromUnitPriceWithCommitments(api.UnitPriceWithCommitments{
			Amount: "0.1",
			Type:   api.UnitPriceWithCommitmentsType("unit"),
		})
		require.Nil(t, err)

		p2RC2 := api.RateCard{}
		err = p2RC2.FromRateCardUsageBased(api.RateCardUsageBased{
			Name:        "Test Plan Phase 2 Rate Card 2",
			Description: lo.ToPtr("Adds a usage based price for the metered feature"),
			Key:         PlanMeteredFeatureKey,
			FeatureKey:  lo.ToPtr(PlanMeteredFeatureKey),
			TaxConfig: &api.TaxConfig{
				Stripe: &api.StripeTaxConfig{
					Code: "txcd_10000000",
				},
			},
			BillingCadence: "P1M",
			Price:          &p2RC2P,
			Type:           api.RateCardUsageBasedType("usage_based"),
		})
		require.Nil(t, err)

		planAPIRes, err := client.CreatePlanWithResponse(ctx, api.CreatePlanJSONRequestBody{
			Currency:    api.CurrencyCode("USD"),
			Name:        "Test Plan",
			Description: lo.ToPtr("Test Plan Description"),
			Key:         PlanKey,
			Phases: []api.PlanPhase{
				{
					Name:        "Test Plan Phase 1",
					Key:         "test_plan_phase_1",
					Description: lo.ToPtr("Test Plan Phase 1 Description"),
					StartAfter:  nil, // Ommittable for first phase
					RateCards:   []api.RateCard{p1RC1, p1RC2},
				},
				{
					Name:        "Test Plan Phase 2",
					Key:         "test_plan_phase_2",
					Description: lo.ToPtr("Test Plan Phase 1 Description"),
					StartAfter:  lo.ToPtr("P2M"),
					RateCards:   []api.RateCard{p2RC1, p2RC2},
				},
			},
		})
		require.Nil(t, err)
		require.Equal(t, 201, planAPIRes.StatusCode())

		plan := planAPIRes.JSON201
		require.NotNil(t, plan, "received the following body: %s", planAPIRes.Body)

		assert.Equal(t, PlanKey, plan.Key)
		require.NotNil(t, plan.Version)
		assert.Equal(t, 1, *plan.Version)

		require.NotNil(t, plan.Id)
		planId = *plan.Id
	})

	t.Run("Should publish the plan", func(t *testing.T) {
		require.NotEmpty(t, planId)
		apiRes, err := client.PublishPlanWithResponse(ctx, planId)
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode())

		body := apiRes.JSON200
		require.NotNil(t, body)
	})

	startTime := time.Now()

	var subscriptionId string

	t.Run("Should create a subscription based on the plan", func(t *testing.T) {
		require.NotNil(t, customer)
		require.NotNil(t, customer.Id)

		create := api.SubscriptionCreate{}
		err := create.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
			ActiveFrom:  startTime,
			CustomerId:  *customer.Id,
			Name:        "Test Subscription",
			Description: lo.ToPtr("Test Subscription Description"),
			Plan: api.PlanReferenceInput{
				Key:     PlanKey,
				Version: lo.ToPtr(1),
			},
		})
		require.Nil(t, err)

		// FIXME: why is this the generated type CreateSubscriptionJSONRequestBody?
		apiRes, err := client.CreateSubscriptionWithResponse(ctx, create)
		require.Nil(t, err)

		assert.Equal(t, 201, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		subscription := apiRes.JSON201
		require.NotNil(t, subscription)
		require.NotNil(t, subscription.Id)
		assert.Equal(t, api.SubscriptionStatusActive, *subscription.Status)

		subscriptionId = *subscription.Id
	})

	t.Run("Should retrieve the subscription", func(t *testing.T) {
		require.NotEmpty(t, subscriptionId)

		apiRes, err := client.GetSubscriptionWithResponse(ctx, subscriptionId, nil)
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		subscription := apiRes.JSON200
		require.NotNil(t, subscription)
		require.NotNil(t, subscription.Id)

		assert.Equal(t, subscriptionId, *subscription.Id)
	})

	t.Run("Should edit the subscription", func(t *testing.T) {
		require.NotEmpty(t, subscriptionId)

		o1 := api.SubscriptionEditOperation{}
		err := o1.FromEditSubscriptionAddPhase(api.EditSubscriptionAddPhase{
			Op: "add_phase",
			Phase: api.SubscriptionPhaseCreate{
				Key:        "test_plan_phase_3",
				Name:       "Test Plan Phase 3",
				StartAfter: lo.ToPtr("P5M"),
			},
		})
		require.Nil(t, err)

		rc := api.RateCard{}
		err = rc.FromRateCardFlatFee(api.RateCardFlatFee{
			Key:  "test_plan_phase_3_rate_card_1",
			Name: "Test Plan Phase 3 Rate Card 1",
		})
		require.Nil(t, err)

		o2 := api.SubscriptionEditOperation{}
		err = o2.FromEditSubscriptionAddItem(api.EditSubscriptionAddItem{
			Op:       "add_item",
			PhaseKey: "test_plan_phase_3",
			RateCard: rc,
		})
		require.Nil(t, err)

		// TODO: test all patches

		apiRes, err := client.EditSubscriptionWithResponse(ctx, subscriptionId, api.EditSubscriptionJSONRequestBody{
			Customizations: []api.SubscriptionEditOperation{o1, o2},
		})
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)
	})
}
