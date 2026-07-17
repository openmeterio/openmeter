package service_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
)

func newPlanSubscriptionService(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies, logger *slog.Logger) plansubscription.PlanSubscriptionService {
	t.Helper()

	svc, err := service.New(service.Config{
		SubscriptionService: deps.SubscriptionService,
		WorkflowService:     deps.WorkflowService,
		Logger:              logger,
		PlanService:         deps.PlanService,
		CurrencyResolver:    deps.CurrencyResolver,
		CustomerService:     deps.CustomerService,
	})
	require.NoError(t, err)

	return svc
}

func TestCreateSettlementModeOverride(t *testing.T) {
	logger := testutils.NewLogger(t)

	type tDeps struct {
		subDeps subscriptiontestutils.SubscriptionDependencies
		subSvc  subscription.Service
		wfSvc   subscriptionworkflow.Service
	}

	withDeps := func(t *testing.T, f func(t *testing.T, deps tDeps)) {
		t.Helper()
		dbDeps := subscriptiontestutils.SetupDBDeps(t)
		defer dbDeps.Cleanup(t)

		deps := subscriptiontestutils.NewService(t, dbDeps)

		f(t, tDeps{
			subDeps: deps,
			subSvc:  deps.SubscriptionService,
			wfSvc:   deps.WorkflowService,
		})
	}

	// given:
	// - a custom plan input whose SettlementMode defaults to CreditThenInvoice
	// when:
	// - the request specifies SettlementMode = CreditOnly
	// then:
	// - the created subscription carries CreditOnly, not the plan default
	t.Run("AsInput: should override plan's default SettlementMode", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps tDeps) {
			now := testutils.GetRFC3339Time(t, "2021-01-01T00:01:10Z")
			clock.SetTime(now)
			defer clock.ResetTime()

			ctx := context.Background()

			svc := newPlanSubscriptionService(t, deps.subDeps, logger)

			cust := deps.subDeps.CustomerAdapter.CreateExampleCustomer(t)
			deps.subDeps.FeatureConnector.CreateExampleFeatures(t, deps.subDeps.ExampleMeterID)

			// Build a custom plan input; clear Key/Version so PlanFromPlanInput accepts it.
			planInput := subscriptiontestutils.GetExamplePlanInput(t)
			require.Equal(t, productcatalog.CreditThenInvoiceSettlementMode, planInput.Plan.SettlementMode, "precondition: plan default must be CreditThenInvoice")
			planInput.Plan.Key = ""
			planInput.Plan.Version = 0

			p1Inp := plansubscription.PlanInput{}
			p1Inp.FromInput(&planInput)

			sub, err := svc.Create(ctx, plansubscription.CreateSubscriptionRequest{
				PlanInput: p1Inp,
				WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
					ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
						Name: "test",
						Timing: subscription.Timing{
							Enum: lo.ToPtr(subscription.TimingImmediate),
						},
					},
					Namespace:  cust.Namespace,
					CustomerID: cust.ID,
				},
				SettlementMode: lo.ToPtr(productcatalog.CreditOnlySettlementMode),
			})
			require.NoError(t, err)
			require.Equal(t, productcatalog.CreditOnlySettlementMode, sub.SettlementMode)
		})
	})

	// given:
	// - a published plan whose SettlementMode defaults to CreditThenInvoice
	// when:
	// - the request references that plan and specifies SettlementMode = CreditOnly
	// then:
	// - the created subscription carries CreditOnly, not the plan's stored SettlementMode
	t.Run("AsRef: should override plan's stored SettlementMode", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps tDeps) {
			now := testutils.GetRFC3339Time(t, "2021-01-01T00:01:10Z")
			clock.SetTime(now)
			defer clock.ResetTime()

			ctx := context.Background()

			svc := newPlanSubscriptionService(t, deps.subDeps, logger)

			cust := deps.subDeps.CustomerAdapter.CreateExampleCustomer(t)
			deps.subDeps.FeatureConnector.CreateExampleFeatures(t, deps.subDeps.ExampleMeterID)

			examplePlanInput := subscriptiontestutils.GetExamplePlanInput(t)
			require.Equal(t, productcatalog.CreditThenInvoiceSettlementMode, examplePlanInput.Plan.SettlementMode, "precondition: plan default must be CreditThenInvoice")

			plan1 := deps.subDeps.PlanHelper.CreatePlan(t, examplePlanInput)

			p1Inp := plansubscription.PlanInput{}
			p1Inp.FromRef(&plansubscription.PlanRefInput{
				Key:     plan1.ToCreateSubscriptionPlanInput().Plan.Key,
				Version: &plan1.ToCreateSubscriptionPlanInput().Plan.Version,
			})

			clock.SetTime(now.Add(time.Second))

			sub, err := svc.Create(ctx, plansubscription.CreateSubscriptionRequest{
				PlanInput: p1Inp,
				WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
					ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
						Name: "test",
						Timing: subscription.Timing{
							Enum: lo.ToPtr(subscription.TimingImmediate),
						},
					},
					Namespace:  cust.Namespace,
					CustomerID: cust.ID,
				},
				SettlementMode: lo.ToPtr(productcatalog.CreditOnlySettlementMode),
			})
			require.NoError(t, err)
			require.Equal(t, productcatalog.CreditOnlySettlementMode, sub.SettlementMode)
		})
	})
}

func TestCreateInlineCustomCurrencyMaterializesManagedIdentity(t *testing.T) {
	// given:
	// - an inline plan that identifies a managed custom currency by code
	// when:
	// - the plan subscription service creates and reloads the subscription
	// then:
	// - every priced item persists the managed currency identity, not the authoring code
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	svc := newPlanSubscriptionService(t, deps, testutils.NewLogger(t))
	customer := deps.CustomerAdapter.CreateExampleCustomer(t)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)

	managedCurrency, err := deps.CurrencyService.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: subscriptiontestutils.ExampleNamespace,
		Code:      "CREDITS",
		Name:      "Credits",
		Symbol:    "CR",
	})
	require.NoError(t, err)

	planInput := subscriptiontestutils.GetExamplePlanInput(t)
	planInput.Plan.Key = ""
	planInput.Plan.Version = 0
	planInput.Plan.Currency = currencyx.Code(managedCurrency.Code)
	planInput.Plan.SettlementMode = productcatalog.CreditOnlySettlementMode

	requestPlan := plansubscription.PlanInput{}
	requestPlan.FromInput(&planInput)

	created, err := svc.Create(t.Context(), plansubscription.CreateSubscriptionRequest{
		PlanInput: requestPlan,
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Name: "inline custom currency",
				Timing: subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingImmediate),
				},
			},
			Namespace:  customer.Namespace,
			CustomerID: customer.ID,
		},
	})
	require.NoError(t, err)

	view, err := deps.SubscriptionService.GetView(t.Context(), created.NamespacedID)
	require.NoError(t, err)

	pricedItems := 0
	for _, phase := range view.Phases {
		for _, items := range phase.ItemsByKey {
			for _, item := range items {
				meta := item.Spec.RateCard.AsMeta()
				if meta.Price == nil {
					continue
				}

				pricedItems++
				require.NotNil(t, meta.Currency)
				currency, ok := meta.Currency.(currencyx.ManagedCurrency)
				require.True(t, ok)
				require.Equal(t, managedCurrency.ID, currency.GetID())
			}
		}
	}
	require.Positive(t, pricedItems)
}

func TestCreateInlinePlanValidatesCurrencyCostBasis(t *testing.T) {
	// given:
	// - a credit-only inline fiat plan with a managed custom-currency rate card
	// - no cost basis for that custom-currency to plan-fiat pair
	// when:
	// - the plan subscription service validates the inline plan
	// then:
	// - product-catalog validation rejects it before subscription validation can waive cost basis
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	dbDeps := subscriptiontestutils.SetupDBDeps(t)
	defer dbDeps.Cleanup(t)

	deps := subscriptiontestutils.NewService(t, dbDeps)
	svc := newPlanSubscriptionService(t, deps, testutils.NewLogger(t))
	customer := deps.CustomerAdapter.CreateExampleCustomer(t)

	_, err := deps.CurrencyService.CreateCurrency(t.Context(), currencies.CreateCurrencyInput{
		Namespace: subscriptiontestutils.ExampleNamespace,
		Code:      "CREDITS",
		Name:      "Credits",
		Symbol:    "CR",
	})
	require.NoError(t, err)

	planInput := subscriptiontestutils.GetExamplePlanInput(t)
	planInput.Plan.Key = ""
	planInput.Plan.Version = 0
	planInput.Plan.SettlementMode = productcatalog.CreditOnlySettlementMode
	require.NoError(t, planInput.Plan.Phases[0].RateCards[0].ChangeMeta(func(meta productcatalog.RateCardMeta) (productcatalog.RateCardMeta, error) {
		meta.Currency = currencyx.Code("CREDITS")
		return meta, nil
	}))

	requestPlan := plansubscription.PlanInput{}
	requestPlan.FromInput(&planInput)

	_, err = svc.Create(t.Context(), plansubscription.CreateSubscriptionRequest{
		PlanInput: requestPlan,
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Name: "inline custom currency without cost basis",
				Timing: subscription.Timing{
					Enum: lo.ToPtr(subscription.TimingImmediate),
				},
			},
			Namespace:  customer.Namespace,
			CustomerID: customer.ID,
		},
	})
	require.ErrorContains(t, err, productcatalog.ErrCurrencyCostBasisNotFound.Error())
}
