package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/ffx"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/models"
)

// Migration happens during update validation, after GetView has captured the old
// plan but before Update acquires the customer lock. This makes the interleaving
// deterministic without relying on goroutine timing or database polling.
type migrateDuringUpdateValidation struct {
	customer.Service
	reads   int
	migrate func(context.Context) error
}

func (s *migrateDuringUpdateValidation) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	result, err := s.Service.GetCustomer(ctx, input)
	if err != nil {
		return nil, err
	}
	s.reads++
	if s.reads == 2 {
		if err := s.migrate(ctx); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func TestUpdateRejectsMigrationBetweenReadAndLock(t *testing.T) {
	// given an edit that has read the old plan before a migration commits
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.FreezeTime(start)
	defer clock.UnFreeze()
	ctx := t.Context()
	db := subscriptiontestutils.SetupDBDeps(t)
	defer db.Cleanup(t)
	deps := subscriptiontestutils.NewService(t, db)
	deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
	p1 := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil, subscriptiontestutils.ExampleRateCard1.Clone()).Build())
	before := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, p1, start)
	clock.FreezeTime(start.Add(10 * 24 * time.Hour))
	p2 := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).AddPhase(nil,
		subscriptiontestutils.ExampleRateCard1.Clone(), subscriptiontestutils.ExampleRateCard2.Clone()).Build())
	var migrated subscription.SubscriptionView
	customerService := &migrateDuringUpdateValidation{Service: deps.CustomerService, migrate: func(ctx context.Context) error {
		var err error
		migrated, err = deps.WorkflowService.MigrateToPlan(ctx, subscriptionworkflow.MigrateSubscriptionWorkflowInput{
			SubscriptionID: before.Subscription.NamespacedID, Plan: p2,
			Timing: subscription.Timing{Enum: lo.ToPtr(subscription.TimingImmediate)},
		})
		return err
	}}
	logger := testutils.NewLogger(t)
	locker, err := lockr.NewLocker(&lockr.LockerConfig{Logger: logger})
	require.NoError(t, err)
	repo := subscriptiontestutils.NewSubscriptionRepo(t, db)
	service, err := subscriptionservice.New(subscriptionservice.ServiceConfig{
		SubscriptionRepo:      repo,
		SubscriptionPhaseRepo: subscriptiontestutils.NewSubscriptionPhaseRepo(t, db),
		SubscriptionItemRepo:  deps.ItemRepo,
		CustomerService:       customerService,
		FeatureService:        deps.FeatureConnector,
		CurrencyResolver:      deps.CurrencyResolver,
		EntitlementAdapter:    deps.EntitlementAdapter,
		TransactionManager:    repo,
		Publisher:             eventbus.NewMock(t),
		Lockr:                 locker,
		FeatureFlags:          ffx.NewTestContextService(ffx.AccessConfig{subscription.MultiSubscriptionEnabledFF: false}),
		TaxCode:               deps.TaxCodeService,
		CostBasisService:      deps.CurrencyService,
	})
	require.NoError(t, err)
	// when that edit resumes and obtains the customer lock
	_, err = service.Update(ctx, before.Subscription.NamespacedID, before.Spec)
	// then it cannot replace the committed migration with stale terms
	require.ErrorContains(t, err, "subscription plan changed during update")
	require.True(t, models.IsGenericConflictError(err))
	require.NotEmpty(t, migrated.Subscription.ID)
	persisted, err := deps.SubscriptionService.GetView(ctx, before.Subscription.NamespacedID)
	require.NoError(t, err)
	require.Equal(t, p2.ToCreateSubscriptionPlanInput().Plan, persisted.Subscription.PlanRef)
	require.Len(t, persisted.Phases[0].ItemsByKey["rate-card-2"], 1)
	require.Equal(t, migrated.Phases[0].ItemsByKey["rate-card-2"][0].SubscriptionItem.ID, persisted.Phases[0].ItemsByKey["rate-card-2"][0].SubscriptionItem.ID)
}
