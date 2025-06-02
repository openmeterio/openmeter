package e2e

import (
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	api "github.com/openmeterio/openmeter/api/client/go"
	"github.com/openmeterio/openmeter/pkg/models"
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

	// Let's set up two customers
	customerAPIRes, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
		Name:         "Test Customer 1",
		Currency:     lo.ToPtr(api.CurrencyCode("USD")),
		Description:  lo.ToPtr("Test Customer Description"),
		PrimaryEmail: lo.ToPtr("customer1@mail.com"),
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
			SubjectKeys: []string{"test_customer_subject_1"},
		},
	})
	require.Nil(t, err)

	require.Equal(t, 201, customerAPIRes.StatusCode(), "received the following body: %s", customerAPIRes.Body)
	customer1 := customerAPIRes.JSON201
	require.NotNil(t, customer1)

	customerAPIRes, err = client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
		Name:         "Test Customer 2",
		Key:          lo.ToPtr("test_customer_2"),
		Currency:     lo.ToPtr(api.CurrencyCode("USD")),
		Description:  lo.ToPtr("Test Customer Description"),
		PrimaryEmail: lo.ToPtr("customer2@mail.com"),
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
			SubjectKeys: []string{"test_customer_subject_2"},
		},
	})
	require.Nil(t, err)

	t.Run("Should check access of customer returning nothing", func(t *testing.T) {
		res, err := client.GetCustomerAccessWithResponse(ctx, customer1.Id)
		require.Nil(t, err)

		require.Equal(t, http.StatusOK, res.StatusCode(), "received the following body: %s", res.Body)
		require.NotNil(t, res.JSON200)
		require.NotNil(t, res.JSON200.Entitlements)
		require.Equal(t, 0, len(res.JSON200.Entitlements))
	})

	customer2 := customerAPIRes.JSON201
	require.NotNil(t, customer2)

	customerAPIRes, err = client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
		Name:         "Test Customer Abused",
		Key:          lo.ToPtr("test_customer_abused"),
		Currency:     lo.ToPtr(api.CurrencyCode("USD")),
		Description:  lo.ToPtr("Test Customer Description"),
		PrimaryEmail: lo.ToPtr("customer_abused@mail.com"),
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
			SubjectKeys: []string{"test_customer_subject_abused"},
		},
	})
	require.Nil(t, err)

	customerAbused := customerAPIRes.JSON201
	require.NotNil(t, customerAbused)

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

	// Lets build a PlanCreate input
	p1RC1 := api.RateCard{}
	err = p1RC1.FromRateCardFlatFee(api.RateCardFlatFee{
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

	et := &api.RateCardEntitlement{}
	err = et.FromRateCardBooleanEntitlement(api.RateCardBooleanEntitlement{
		Type: api.RateCardBooleanEntitlementType("boolean"),
	})
	require.Nil(t, err)

	p1RC2 := api.RateCard{}
	err = p1RC2.FromRateCardFlatFee(api.RateCardFlatFee{
		Name:                "Test Plan Phase 1 Rate Card 2",
		Description:         lo.ToPtr("Has a monthly recurring price to grant access to a feature"),
		Key:                 PlanFeatureKey,
		FeatureKey:          lo.ToPtr(PlanFeatureKey),
		EntitlementTemplate: et,
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

	p2RC2UsageDiscount := api.DiscountPercentage{
		Percentage: models.NewPercentage(10),
	}

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
		Discounts:      &api.Discounts{Percentage: &p2RC2UsageDiscount},
		Type:           api.RateCardUsageBasedType("usage_based"),
	})
	require.Nil(t, err)

	planCreate := api.PlanCreate{
		Currency:       api.CurrencyCode("USD"),
		Name:           "Test Plan",
		Description:    lo.ToPtr("Test Plan Description"),
		Key:            PlanKey,
		BillingCadence: "P1M",
		Phases: []api.PlanPhase{
			{
				Name:        "Test Plan Phase 1",
				Key:         "test_plan_phase_1",
				Description: lo.ToPtr("Test Plan Phase 1 Description"),
				Duration:    lo.ToPtr("P2M"),
				RateCards:   []api.RateCard{p1RC1, p1RC2},
			},
			{
				Name:        "Test Plan Phase 2",
				Key:         "test_plan_phase_2",
				Description: lo.ToPtr("Test Plan Phase 1 Description"),
				Duration:    nil,
				RateCards:   []api.RateCard{p2RC1, p2RC2},
			},
		},
	}

	customPlanInput := api.CustomPlanInput{
		Currency:       planCreate.Currency,
		Name:           planCreate.Name,
		Description:    planCreate.Description,
		BillingCadence: planCreate.BillingCadence,
		Phases:         planCreate.Phases,
		ProRatingConfig: &api.ProRatingConfig{
			Mode:    "prorate_prices",
			Enabled: true,
		},
	}

	t.Run("Should create a plan on happy path", func(t *testing.T) {
		planAPIRes, err := client.CreatePlanWithResponse(ctx, planCreate)
		require.Nil(t, err)
		require.Equal(t, 201, planAPIRes.StatusCode())

		plan := planAPIRes.JSON201
		require.NotNil(t, plan, "received the following body: %s", planAPIRes.Body)

		assert.Equal(t, PlanKey, plan.Key)
		require.NotNil(t, plan.Version)
		assert.Equal(t, 1, plan.Version)

		require.NotNil(t, plan.Id)
		planId = plan.Id
	})

	t.Run("Plan should have discounts correctly recorded", func(t *testing.T) {
		require.Len(t, planCreate.Phases, 2)
		require.Len(t, planCreate.Phases[1].RateCards, 2)

		rateCard, found := lo.Find(planCreate.Phases[1].RateCards, func(rc api.RateCard) bool {
			disc, err := rc.Discriminator()
			if err != nil {
				return false
			}

			if disc != string(api.RateCardUsageBasedTypeUsageBased) {
				return false
			}

			usageBased, err := rc.AsRateCardUsageBased()
			if err != nil {
				return false
			}

			return *usageBased.FeatureKey == PlanMeteredFeatureKey
		})
		require.True(t, found)

		ubpRateCard, err := rateCard.AsRateCardUsageBased()
		require.NoError(t, err)
		require.NotNil(t, ubpRateCard)

		require.NotNil(t, ubpRateCard.Discounts)
		require.NotNil(t, ubpRateCard.Discounts.Percentage)
		require.Equal(t, float64(10), ubpRateCard.Discounts.Percentage.Percentage.InexactFloat64())
	})

	t.Run("Should publish the plan", func(t *testing.T) {
		require.NotEmpty(t, planId)
		apiRes, err := client.PublishPlanWithResponse(ctx, planId)
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode())

		body := apiRes.JSON200
		require.NotNil(t, body)
	})

	t.Run("Should not allow publishing a misaligned plan", func(t *testing.T) {
		// Uses a separate plan to avoid conflicts

		maP1RC1 := api.RateCard{}
		err = maP1RC1.FromRateCardFlatFee(api.RateCardFlatFee{
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
			BillingCadence: lo.ToPtr("P1W"),
			Type:           api.RateCardFlatFeeType("flat"),
		})
		require.Nil(t, err)

		et := &api.RateCardEntitlement{}
		err = et.FromRateCardBooleanEntitlement(api.RateCardBooleanEntitlement{
			Type: api.RateCardBooleanEntitlementType("boolean"),
		})
		require.Nil(t, err)

		maP1RC2 := api.RateCard{}
		err = maP1RC2.FromRateCardFlatFee(api.RateCardFlatFee{
			Name:                "Test Plan Phase 1 Rate Card 2",
			Description:         lo.ToPtr("Has a monthly recurring price to grant access to a feature"),
			Key:                 PlanFeatureKey,
			FeatureKey:          lo.ToPtr(PlanFeatureKey),
			EntitlementTemplate: et,
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

		planKey := "test_plan_misaligned"
		misalignedCreate := planCreate
		misalignedCreate.Key = planKey

		misalignedCreate.Alignment = &api.Alignment{
			BillablesMustAlign: lo.ToPtr(true),
		}
		misalignedCreate.BillingCadence = "P1M"

		misalignedCreate.Phases = slices.Clone(planCreate.Phases)
		misalignedCreate.Phases[0].RateCards = slices.Clone(planCreate.Phases[0].RateCards)
		misalignedCreate.Phases[0].RateCards = []api.RateCard{maP1RC1, maP1RC2}

		planAPIRes, err := client.CreatePlanWithResponse(ctx, misalignedCreate)
		require.Nil(t, err)
		require.Equal(t, 201, planAPIRes.StatusCode(), "received the following body: %s", planAPIRes.Body)

		plan := planAPIRes.JSON201
		require.NotNil(t, plan, "received the following body: %s", planAPIRes.Body)

		require.NotNil(t, plan.Version)
		assert.Equal(t, 1, plan.Version)

		require.NotNil(t, plan.Id)

		// Let's try to publish it and assert it fails
		require.NotNil(t, plan.Id)
		apiRes, err := client.PublishPlanWithResponse(ctx, plan.Id)
		require.Nil(t, err)

		assert.Equal(t, 400, apiRes.StatusCode(), "should return 400, received the following body: %s", apiRes.Body)

		// Now let's update the plan to remove the alignment requirement
		updateRes, err := client.UpdatePlanWithResponse(ctx, plan.Id, api.UpdatePlanJSONRequestBody{
			Name:           plan.Name,
			BillingCadence: "P1M",
			Alignment: &api.Alignment{
				BillablesMustAlign: lo.ToPtr(false),
			},
			Phases: plan.Phases,
		})
		require.Nil(t, err)

		assert.Equal(t, 200, updateRes.StatusCode(), "received the following body: %s", updateRes.Body)

		// And let's try to publish it once again
		publishRes, err := client.PublishPlanWithResponse(ctx, plan.Id)
		require.Nil(t, err)

		assert.Equal(t, 200, publishRes.StatusCode(), "received the following body: %s", publishRes.Body)
	})

	startTime := time.Now()

	var subscriptionId string
	var customSubscriptionId string

	t.Run("Should create a custom subscription", func(t *testing.T) {
		require.NotNil(t, customer1)
		require.NotNil(t, customer1.Id)

		ct := &api.SubscriptionTiming{}
		require.NoError(t, ct.FromSubscriptionTiming1(startTime))

		create := api.SubscriptionCreate{}

		anchorTime := time.Now().Add(-time.Hour).Truncate(time.Millisecond).UTC()

		err := create.FromCustomSubscriptionCreate(api.CustomSubscriptionCreate{
			Timing:        ct,
			CustomerKey:   customer2.Key,   // Let's use the key
			CustomPlan:    customPlanInput, // For simplicity we can reuse the same plan input, we know its valid
			BillingAnchor: lo.ToPtr(anchorTime),
		})
		require.Nil(t, err)

		apiRes, err := client.CreateSubscriptionWithResponse(ctx, create)
		require.Nil(t, err)

		assert.Equal(t, 201, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		subscription := apiRes.JSON201
		require.NotNil(t, subscription)
		require.NotNil(t, subscription.Id)
		assert.Equal(t, api.SubscriptionStatusActive, subscription.Status)
		assert.Nil(t, subscription.Plan)

		customSubscriptionId = subscription.Id
		require.Equal(t, "P1M", subscription.BillingCadence)
		require.True(t, anchorTime.UTC().Equal(subscription.BillingAnchor.UTC()), "billing anchor should be %s, got %s", anchorTime, subscription.BillingAnchor)
		require.Equal(t, api.ProRatingModeProratePrices, subscription.ProRatingConfig.Mode)
		require.True(t, subscription.ProRatingConfig.Enabled)
	})

	t.Run("Should list customer subscriptions", func(t *testing.T) {
		require.NotNil(t, customer2)
		require.NotNil(t, customer2.Id)

		apiRes, err := client.ListCustomerSubscriptionsWithResponse(ctx, customer2.Id, &api.ListCustomerSubscriptionsParams{
			Page:     lo.ToPtr(1),
			PageSize: lo.ToPtr(10),
		})
		require.Nil(t, err)
		require.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		body := apiRes.JSON200
		require.NotNil(t, body)

		require.Equal(t, 1, len(body.Items))
		require.Equal(t, customSubscriptionId, body.Items[0].Id)
		require.Equal(t, 1, body.Page)
		require.Equal(t, 10, body.PageSize)
		require.Equal(t, 1, body.TotalCount)
	})

	t.Run("Should create a subscription based on the plan", func(t *testing.T) {
		require.NotNil(t, customer1)
		require.NotNil(t, customer1.Id)

		ct := &api.SubscriptionTiming{}
		require.NoError(t, ct.FromSubscriptionTiming1(startTime))

		create := api.SubscriptionCreate{}
		err := create.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
			Timing:      ct,
			CustomerId:  &customer1.Id,
			Name:        lo.ToPtr("Test Subscription"),
			Description: lo.ToPtr("Test Subscription Description"),
			Plan: api.PlanReferenceInput{
				Key:     PlanKey,
				Version: lo.ToPtr(1),
			},
		})
		require.Nil(t, err)

		apiRes, err := client.CreateSubscriptionWithResponse(ctx, create)
		require.Nil(t, err)

		assert.Equal(t, 201, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		subscription := apiRes.JSON201
		require.NotNil(t, subscription)
		require.NotNil(t, subscription.Id)
		assert.Equal(t, api.SubscriptionStatusActive, subscription.Status)
		assert.Equal(t, planId, subscription.Plan.Id)

		subscriptionId = subscription.Id
		require.Equal(t, "P1M", subscription.BillingCadence)
		require.Equal(t, api.ProRatingModeProratePrices, subscription.ProRatingConfig.Mode)
		require.True(t, subscription.ProRatingConfig.Enabled)
	})

	t.Run("Should create only ONE subscription per customer, even if we spam the API in a short period of time", func(t *testing.T) {
		require.NotNil(t, customerAbused)
		require.NotNil(t, customerAbused.Id)

		ct := &api.SubscriptionTiming{}
		require.NoError(t, ct.FromSubscriptionTiming1(startTime))

		createSubscription := func() {
			ct := &api.SubscriptionTiming{}
			require.NoError(t, ct.FromSubscriptionTiming1(startTime))

			// Let's create a custom subscription so it doesn't affect the other tests
			create := api.SubscriptionCreate{}
			err := create.FromCustomSubscriptionCreate(api.CustomSubscriptionCreate{
				Timing:      ct,
				CustomerKey: customerAbused.Key,
				CustomPlan:  customPlanInput,
			})
			require.Nil(t, err)

			apiRes, err := client.CreateSubscriptionWithResponse(ctx, create)
			require.Nil(t, err)

			// It will either succeed or fail with 4xx
			assert.Less(t, apiRes.StatusCode(), 500, "received the following status %d body: %s", apiRes.StatusCode(), apiRes.Body)
		}

		// Let's spam the API 10 times
		wg := sync.WaitGroup{}
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				createSubscription()
			}()
		}

		wg.Wait()

		// Now let's fetch the customer's subscriptions and assert there's only one
		apiRes, err := client.ListCustomerSubscriptionsWithResponse(ctx, customerAbused.Id, &api.ListCustomerSubscriptionsParams{
			Page:     lo.ToPtr(1),
			PageSize: lo.ToPtr(10),
		})
		require.Nil(t, err)
		require.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		body := apiRes.JSON200
		require.NotNil(t, body)

		require.Equal(t, 1, len(body.Items))
	})

	t.Run("Should retrieve the subscription", func(t *testing.T) {
		require.NotEmpty(t, subscriptionId)

		apiRes, err := client.GetSubscriptionWithResponse(ctx, subscriptionId, nil)
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		subscription := apiRes.JSON200
		require.NotNil(t, subscription)
		require.NotNil(t, subscription.Id)

		assert.Equal(t, subscriptionId, subscription.Id)
		assert.Equal(t, api.SubscriptionStatusActive, subscription.Status)

		// Should have the current period info
		assert.NotNil(t, subscription.Alignment)
		assert.NotNil(t, subscription.Alignment.CurrentAlignedBillingPeriod)
		assert.NotEmpty(t, subscription.Alignment.CurrentAlignedBillingPeriod.From)
		assert.NotEmpty(t, subscription.Alignment.CurrentAlignedBillingPeriod.To)

		// Should have item features filled
		require.GreaterOrEqual(t, len(subscription.Phases), 1)
		phase := subscription.Phases[0]
		require.GreaterOrEqual(t, len(phase.Items), 2)

		item, ok := lo.Find(phase.Items, func(i api.SubscriptionItem) bool {
			return i.Key == PlanFeatureKey
		})
		require.True(t, ok)
		require.NotNil(t, item.Included.Feature)
		assert.Equal(t, PlanFeatureKey, item.Included.Feature.Key)
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

		// Let's fetch the sub and see the change in the timeline
		viewRes, err := client.GetSubscriptionWithResponse(ctx, subscriptionId, nil)
		require.Nil(t, err)

		assert.Equal(t, 200, viewRes.StatusCode(), "received the following body: %s", viewRes.Body)

		require.NotNil(t, viewRes.JSON200)

		// Let's get the phase
		require.GreaterOrEqual(t, len(viewRes.JSON200.Phases), 3)
		phase := viewRes.JSON200.Phases[2]

		require.NotNil(t, phase.ItemTimelines)
		require.NotNil(t, phase.ItemTimelines["test_plan_phase_3_rate_card_1"])
		require.Equal(t, 1, len(phase.ItemTimelines["test_plan_phase_3_rate_card_1"]))
	})

	t.Run("Should schedule a cancellation for the subscription", func(t *testing.T) {
		require.NotEmpty(t, subscriptionId)

		ct := &api.SubscriptionTiming{}
		require.NoError(t, ct.FromSubscriptionTimingEnum(api.SubscriptionTimingEnum("next_billing_cycle")))

		apiRes, err := client.CancelSubscriptionWithResponse(ctx, subscriptionId, api.CancelSubscriptionJSONRequestBody{
			Timing: ct,
		})
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		require.NotNil(t, apiRes.JSON200)
		assert.Equal(t, api.SubscriptionStatusCanceled, apiRes.JSON200.Status)
	})

	t.Run("Should unschedule cancellation", func(t *testing.T) {
		require.NotEmpty(t, subscriptionId)

		apiRes, err := client.UnscheduleCancelationWithResponse(ctx, subscriptionId)
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)

		require.NotNil(t, apiRes.JSON200)
		assert.Equal(t, api.SubscriptionStatusActive, apiRes.JSON200.Status)
	})

	t.Run("Should create and publish a new version of the plan", func(t *testing.T) {
		require.NotNil(t, planId)

		newPhases := []api.PlanPhase{
			planCreate.Phases[0],
			{
				Name:      planCreate.Phases[1].Name,
				Key:       planCreate.Phases[1].Key,
				Duration:  lo.ToPtr("P7M"),
				RateCards: planCreate.Phases[1].RateCards,
			},
			{
				Name:      "Test Plan Phase 3",
				Key:       "test_plan_phase_3",
				Duration:  nil,
				RateCards: []api.RateCard{p2RC1},
			},
		}

		planAPIRes, err := client.CreatePlanWithResponse(ctx, api.CreatePlanJSONRequestBody{
			Name:           "Test Plan New Version",
			Key:            PlanKey,
			Currency:       api.CurrencyCode("USD"),
			BillingCadence: "P1M",
			// Let's add a new phase
			Phases: newPhases,
		})

		require.Nil(t, err)

		require.Equal(t, 201, planAPIRes.StatusCode(), "received the following body: %s", planAPIRes.Body)
		require.NotNil(t, planAPIRes.JSON201)
		require.NotNil(t, planAPIRes.JSON201.Id)
		require.NotNil(t, planAPIRes.JSON201.Version)
		require.NotNil(t, planAPIRes.JSON201.Key)

		assert.NotEqual(t, planId, planAPIRes.JSON201.Id)
		assert.Equal(t, PlanKey, planAPIRes.JSON201.Key)
		assert.Equal(t, 2, planAPIRes.JSON201.Version)

		// Let's publish the new version
		apiRes2, err := client.PublishPlanWithResponse(ctx, planAPIRes.JSON201.Id)
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes2.StatusCode(), "received the following body: %s", apiRes2.Body)
	})

	var migratedSubscriptionId string

	t.Run("Should migrate the subscription to a newer version", func(t *testing.T) {
		require.NotNil(t, subscriptionId)

		apiRes, err := client.MigrateSubscriptionWithResponse(ctx, subscriptionId, api.MigrateSubscriptionJSONRequestBody{
			TargetVersion: lo.ToPtr(2),
		})
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)
		require.NotNil(t, apiRes.JSON200)
		require.NotNil(t, apiRes.JSON200.Next.Id)
		require.NotNil(t, apiRes.JSON200.Current.Id)

		require.Equal(t, subscriptionId, apiRes.JSON200.Current.Id)
		require.NotEqual(t, subscriptionId, apiRes.JSON200.Next.Id)

		migratedSubscriptionId = apiRes.JSON200.Next.Id

		require.Equal(t, 3, len(apiRes.JSON200.Next.Phases))
		require.Equal(t, "test_plan_phase_3", apiRes.JSON200.Next.Phases[2].Key)
	})

	t.Run("Should change the subscription's plan", func(t *testing.T) {
		// We'll use the custom sub for this
		require.NotNil(t, customSubscriptionId)

		req := api.SubscriptionChange{}

		ct := &api.SubscriptionTiming{}
		require.NoError(t, ct.FromSubscriptionTimingEnum(api.SubscriptionTimingEnum("immediate")))

		err := req.FromCustomSubscriptionChange(api.CustomSubscriptionChange{
			Timing:     *ct,
			CustomPlan: customPlanInput, // It will functionally be the same as the old plan
		})
		require.Nil(t, err)

		// For simplicity, let's change the plan to a custom one
		apiRes, err := client.ChangeSubscriptionWithResponse(ctx, customSubscriptionId, req)
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)
		require.NotNil(t, apiRes.JSON200)
		require.NotNil(t, apiRes.JSON200.Current.Id)
		require.NotNil(t, apiRes.JSON200.Next.Id)

		require.Equal(t, customSubscriptionId, apiRes.JSON200.Current.Id)
		require.NotEqual(t, customSubscriptionId, apiRes.JSON200.Next.Id)

		require.Equal(t, 2, len(planCreate.Phases))
	})

	t.Run("Should list customers of a given plan", func(t *testing.T) {
		// Let's make sure our customer is there
		require.NotNil(t, customer1)
		require.NotEmpty(t, migratedSubscriptionId)

		// Let's create a 3rd customer that doesnt have a subscription
		// Let's set up two customers
		customerAPIRes, err := client.CreateCustomerWithResponse(ctx, api.CreateCustomerJSONRequestBody{
			Name:         "Test Customer 3",
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			Description:  lo.ToPtr("Test Customer Description"),
			PrimaryEmail: lo.ToPtr("customer3@mail.com"),
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
				SubjectKeys: []string{"test_customer_subject_3"},
			},
		})
		require.Nil(t, err)
		require.Equal(t, 201, customerAPIRes.StatusCode(), "received the following body: %s", customerAPIRes.Body)

		// Let's make sure both customers do exist!
		apiRes, err := client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{})
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)
		require.NotNil(t, apiRes.JSON200)
		require.NotNil(t, apiRes.JSON200.Items)
		require.Equal(t, 4, len(apiRes.JSON200.Items))

		// Now let's check the filtering works
		apiRes, err = client.ListCustomersWithResponse(ctx, &api.ListCustomersParams{
			PlanKey: lo.ToPtr(PlanKey),
		})
		require.Nil(t, err)

		assert.Equal(t, 200, apiRes.StatusCode(), "received the following body: %s", apiRes.Body)
		require.NotNil(t, apiRes.JSON200)
		require.NotNil(t, apiRes.JSON200.Items)

		// Only customer 1 is returned
		require.Equal(t, 1, len(apiRes.JSON200.Items))
		require.Equal(t, customer1.Id, apiRes.JSON200.Items[0].Id)
	})

	t.Run("Should check entitlement of customer", func(t *testing.T) {
		res, err := client.GetCustomerEntitlementValueWithResponse(ctx, customer1.Id, PlanFeatureKey, nil)
		require.Nil(t, err)

		require.Equal(t, http.StatusOK, res.StatusCode(), "received the following body: %s", res.Body)
		require.NotNil(t, res.JSON200)
		require.NotNil(t, res.JSON200.HasAccess)
		assert.True(t, res.JSON200.HasAccess)
	})

	t.Run("Should check access of customer", func(t *testing.T) {
		res, err := client.GetCustomerAccessWithResponse(ctx, customer1.Id)
		require.Nil(t, err)

		require.Equal(t, http.StatusOK, res.StatusCode(), "received the following body: %s", res.Body)
		require.NotNil(t, res.JSON200)
		require.NotNil(t, res.JSON200.Entitlements)
		require.NotNil(t, res.JSON200.Entitlements[PlanFeatureKey])
		require.True(t, res.JSON200.Entitlements[PlanFeatureKey].HasAccess)
	})
}
