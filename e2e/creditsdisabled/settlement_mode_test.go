package creditsdisabled

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	api "github.com/openmeterio/openmeter/api/client/go"
)

const creditsDisabledDetail = "credits are not enabled on this deployment of OpenMeter"

// TestSettlementMode verifies the credit_only settlement mode guard and that
// credit_then_invoice is accepted and round-tripped correctly when credits are
// disabled.
func TestSettlementMode(t *testing.T) {
	client := initClient(t)

	ctx := t.Context()

	rc := api.RateCard{}
	require.NoError(t, rc.FromRateCardFlatFee(api.RateCardFlatFee{
		Key:  gofakeit.Numerify("ratecard_####"),
		Name: "flat",
		Type: api.RateCardFlatFeeTypeFlatFee,
		Price: &api.FlatPriceWithPaymentTerm{
			Amount: "100",
			Type:   api.FlatPriceWithPaymentTermType("flat"),
		},
	}))

	defaultPhase := api.PlanPhase{
		Key:       "default",
		Name:      "Default Phase",
		RateCards: []api.RateCard{rc},
	}

	t.Run("Should reject a plan with credit_only settlement mode when credit is disabled", func(t *testing.T) {
		res, err := client.CreatePlanWithResponse(ctx, api.PlanCreate{
			Key:            gofakeit.Numerify("test_plan_settlement_credit_only_####"),
			Name:           "Credit Only Plan",
			Currency:       "USD",
			BillingCadence: "P1M",
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditOnly),
			Phases:         []api.PlanPhase{defaultPhase},
		})
		require.NoError(t, err)
		assert.Equal(t, 400, res.StatusCode(), "received the following body: %s", res.Body)
		require.NotNil(t, res.ApplicationproblemJSON400)
		assert.Contains(t, res.ApplicationproblemJSON400.Detail, creditsDisabledDetail)
	})

	t.Run("Should accept a plan with credit_then_invoice settlement mode and return it in the response", func(t *testing.T) {
		res, err := client.CreatePlanWithResponse(ctx, api.PlanCreate{
			Key:            gofakeit.Numerify("test_plan_settlement_credit_then_invoice_####"),
			Name:           "Credit Then Invoice Plan",
			Currency:       "USD",
			BillingCadence: "P1M",
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
			Phases:         []api.PlanPhase{defaultPhase},
		})
		require.NoError(t, err)
		require.Equal(t, 201, res.StatusCode(), "received the following body: %s", res.Body)
		require.NotNil(t, res.JSON201)
		assert.Equal(t, lo.ToPtr(api.BillingSettlementModeCreditThenInvoice), res.JSON201.SettlementMode)
	})

	t.Run("Should reject a custom subscription with credit_only settlement mode when credit is disabled", func(t *testing.T) {
		customer, err := client.CreateCustomerWithResponse(ctx, api.CustomerCreate{
			Key:          lo.ToPtr(gofakeit.Numerify("customer_####")),
			Name:         gofakeit.Name(),
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			PrimaryEmail: lo.ToPtr("testcustomer@example.com"),
		})
		require.NoError(t, err)

		create := api.SubscriptionCreate{}
		err = create.FromCustomSubscriptionCreate(api.CustomSubscriptionCreate{
			CustomerId: lo.ToPtr(customer.JSON201.Id),
			CustomPlan: api.CustomPlanInput{
				Name:           "Credit Only Custom Plan",
				Currency:       "USD",
				BillingCadence: "P1M",
				SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditOnly),
				Phases:         []api.PlanPhase{defaultPhase},
			},
		})
		require.NoError(t, err)

		res, err := client.CreateSubscriptionWithResponse(ctx, create)
		require.NoError(t, err)
		assert.Equal(t, 400, res.StatusCode(), "received the following body: %s", res.Body)
		assert.Contains(t, string(res.Body), creditsDisabledDetail)
	})

	t.Run("Should reject a custom subscription change with credit_only settlement mode when credit is disabled", func(t *testing.T) {
		customer, err := client.CreateCustomerWithResponse(ctx, api.CustomerCreate{
			Key:          lo.ToPtr(gofakeit.Numerify("customer_####")),
			Name:         gofakeit.Name(),
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			PrimaryEmail: lo.ToPtr("testcustomer@example.com"),
		})
		require.NoError(t, err)

		create := api.SubscriptionCreate{}
		require.NoError(t, create.FromCustomSubscriptionCreate(api.CustomSubscriptionCreate{
			CustomerId: lo.ToPtr(customer.JSON201.Id),
			CustomPlan: api.CustomPlanInput{
				Name:           "Credit Only Custom Plan",
				Currency:       "USD",
				BillingCadence: "P1M",
				SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
				Phases:         []api.PlanPhase{defaultPhase},
			},
		}))

		subscription, err := client.CreateSubscriptionWithResponse(ctx, create)
		require.NoError(t, err)
		require.Equal(t, 201, subscription.StatusCode(), "received the following body: %s", subscription.Body)

		ct := &api.SubscriptionTiming{}
		require.NoError(t, ct.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

		req := api.SubscriptionChange{}
		err = req.FromCustomSubscriptionChange(api.CustomSubscriptionChange{
			Timing: *ct,
			CustomPlan: api.CustomPlanInput{
				Name:           "Credit Only Custom Plan",
				Currency:       "USD",
				BillingCadence: "P1M",
				SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditOnly),
				Phases:         []api.PlanPhase{defaultPhase},
			},
		})
		require.NoError(t, err)

		res, err := client.ChangeSubscriptionWithResponse(ctx, subscription.JSON201.Id, req)
		require.NoError(t, err)
		assert.Equal(t, 400, res.StatusCode(), "received the following body: %s", res.Body)
		assert.Contains(t, string(res.Body), creditsDisabledDetail)
	})

	t.Run("Should reject a plan update with credit_only settlement mode when credit is disabled", func(t *testing.T) {
		createRes, err := client.CreatePlanWithResponse(ctx, api.PlanCreate{
			Key:            gofakeit.Numerify("test_plan_settlement_update_credit_only_####"),
			Name:           "Credit Only Plan Update Test",
			Currency:       "USD",
			BillingCadence: "P1M",
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
			Phases:         []api.PlanPhase{defaultPhase},
		})
		require.NoError(t, err)
		require.Equal(t, 201, createRes.StatusCode(), "received the following body: %s", createRes.Body)
		require.NotNil(t, createRes.JSON201)

		res, err := client.UpdatePlanWithResponse(ctx, createRes.JSON201.Id, api.PlanReplaceUpdate{
			Name:           "Credit Only Plan Updated",
			BillingCadence: "P1M",
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditOnly),
			Phases:         []api.PlanPhase{defaultPhase},
		})
		require.NoError(t, err)
		assert.Equal(t, 400, res.StatusCode(), "received the following body: %s", res.Body)
		require.NotNil(t, res.ApplicationproblemJSON400)
		assert.Contains(t, res.ApplicationproblemJSON400.Detail, creditsDisabledDetail)
	})

	t.Run("Should accept a plan update with credit_then_invoice settlement mode and return it in the response", func(t *testing.T) {
		createRes, err := client.CreatePlanWithResponse(ctx, api.PlanCreate{
			Key:            gofakeit.Numerify("test_plan_settlement_update_cti_####"),
			Name:           "Update Test Plan",
			Currency:       "USD",
			BillingCadence: "P1M",
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
			Phases:         []api.PlanPhase{defaultPhase},
		})
		require.NoError(t, err)
		require.Equal(t, 201, createRes.StatusCode(), "received the following body: %s", createRes.Body)
		require.NotNil(t, createRes.JSON201)

		res, err := client.UpdatePlanWithResponse(ctx, createRes.JSON201.Id, api.PlanReplaceUpdate{
			Name:           "Update Test Plan Updated",
			BillingCadence: "P1M",
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
			Phases:         []api.PlanPhase{defaultPhase},
		})
		require.NoError(t, err)
		require.Equal(t, 200, res.StatusCode(), "received the following body: %s", res.Body)
		require.NotNil(t, res.JSON200)
		assert.Equal(t, lo.ToPtr(api.BillingSettlementModeCreditThenInvoice), res.JSON200.SettlementMode)
	})

	t.Run("Should reject a plan-based subscription with credit_only settlement mode when credit is disabled", func(t *testing.T) {
		customer, err := client.CreateCustomerWithResponse(ctx, api.CustomerCreate{
			Key:          lo.ToPtr(gofakeit.Numerify("customer_####")),
			Name:         gofakeit.Name(),
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			PrimaryEmail: lo.ToPtr("testcustomer@example.com"),
		})
		require.NoError(t, err)

		plan, err := client.CreatePlanWithResponse(ctx, api.PlanCreate{
			Key:            gofakeit.Numerify("test_plan_settlement_credit_then_invoice_####"),
			Name:           "Credit Only Plan",
			Currency:       "USD",
			BillingCadence: "P1M",
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
			Phases:         []api.PlanPhase{defaultPhase},
		})
		require.NoError(t, err)

		_, err = client.PublishPlanWithResponse(ctx, plan.JSON201.Id)
		require.NoError(t, err)

		ct := &api.SubscriptionTiming{}
		require.NoError(t, ct.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

		create := api.SubscriptionCreate{}
		err = create.FromPlanSubscriptionCreate(api.PlanSubscriptionCreate{
			Timing:         ct,
			CustomerId:     lo.ToPtr(customer.JSON201.Id),
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditOnly),
			Plan: api.PlanReferenceInput{
				Key:     plan.JSON201.Key,
				Version: lo.ToPtr(1),
			},
		})
		require.NoError(t, err)

		res, err := client.CreateSubscriptionWithResponse(ctx, create)
		require.NoError(t, err)
		assert.Equal(t, 400, res.StatusCode(), "received the following body: %s", res.Body)
		assert.Contains(t, string(res.Body), creditsDisabledDetail)
	})

	t.Run("Should reject a plan-based subscription change with credit_only settlement mode when credit is disabled", func(t *testing.T) {
		customer, err := client.CreateCustomerWithResponse(ctx, api.CustomerCreate{
			Key:          lo.ToPtr(gofakeit.Numerify("customer_####")),
			Name:         gofakeit.Name(),
			Currency:     lo.ToPtr(api.CurrencyCode("USD")),
			PrimaryEmail: lo.ToPtr("testcustomer@example.com"),
		})
		require.NoError(t, err)

		plan, err := client.CreatePlanWithResponse(ctx, api.PlanCreate{
			Key:            gofakeit.Numerify("test_plan_settlement_credit_then_invoice_####"),
			Name:           "Credit Only Plan",
			Currency:       "USD",
			BillingCadence: "P1M",
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
			Phases:         []api.PlanPhase{defaultPhase},
		})
		require.NoError(t, err)

		_, err = client.PublishPlanWithResponse(ctx, plan.JSON201.Id)
		require.NoError(t, err)

		create := api.SubscriptionCreate{}
		require.NoError(t, create.FromCustomSubscriptionCreate(api.CustomSubscriptionCreate{
			CustomerId: lo.ToPtr(customer.JSON201.Id),
			CustomPlan: api.CustomPlanInput{
				Name:           "Credit Only Custom Plan",
				Currency:       "USD",
				BillingCadence: "P1M",
				SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditThenInvoice),
				Phases:         []api.PlanPhase{defaultPhase},
			},
		}))

		subscription, err := client.CreateSubscriptionWithResponse(ctx, create)
		require.NoError(t, err)
		require.Equal(t, 201, subscription.StatusCode(), "received the following body: %s", subscription.Body)

		ct := &api.SubscriptionTiming{}
		require.NoError(t, ct.FromSubscriptionTimingEnum(api.SubscriptionTimingEnumImmediate))

		req := api.SubscriptionChange{}
		err = req.FromPlanSubscriptionChange(api.PlanSubscriptionChange{
			Timing:         *ct,
			SettlementMode: lo.ToPtr(api.BillingSettlementModeCreditOnly),
			Plan: api.PlanReferenceInput{
				Key:     plan.JSON201.Key,
				Version: lo.ToPtr(1),
			},
		})
		require.NoError(t, err)

		res, err := client.ChangeSubscriptionWithResponse(ctx, subscription.JSON201.Id, req)
		require.Nil(t, err)
		assert.Equal(t, 400, res.StatusCode(), "received the following body: %s", res.Body)
		assert.Contains(t, string(res.Body), creditsDisabledDetail)
	})
}
