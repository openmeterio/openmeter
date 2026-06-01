package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/subscription/service"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
)

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

			svc := service.New(service.Config{
				SubscriptionService: deps.subSvc,
				WorkflowService:     deps.wfSvc,
				Logger:              logger,
				PlanService:         deps.subDeps.PlanService,
				CustomerService:     deps.subDeps.CustomerService,
			})

			cust := deps.subDeps.CustomerAdapter.CreateExampleCustomer(t)

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

			svc := service.New(service.Config{
				SubscriptionService: deps.subSvc,
				WorkflowService:     deps.wfSvc,
				Logger:              logger,
				PlanService:         deps.subDeps.PlanService,
				CustomerService:     deps.subDeps.CustomerService,
			})

			cust := deps.subDeps.CustomerAdapter.CreateExampleCustomer(t)

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
