package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionentitlement "github.com/openmeterio/openmeter/openmeter/subscription/entitlement"
	subscriptionservice "github.com/openmeterio/openmeter/openmeter/subscription/service"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/ffx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

// Commit a real cancellation at a read boundary instead of depending on query
// latency or goroutine scheduling. The writer uses its own transaction.
type cancellationDuringViewRead struct {
	service        subscription.Service
	repo           subscription.SubscriptionRepository
	subscriptionID models.NamespacedID
	customerID     string
	attempted      bool
	committed      bool
	serialized     bool
}

func (c *cancellationDuringViewRead) run(ctx context.Context) error {
	if c.attempted {
		return nil
	}

	c.attempted = true

	_, err := transaction.RunInNewTransaction(ctx, c.repo, func(ctx context.Context) (struct{}, error) {
		key, err := subscription.GetCustomerLock(c.customerID)
		if err != nil {
			return struct{}{}, err
		}

		driver, err := entutils.GetDriverFromContext(ctx)
		if err != nil {
			return struct{}{}, err
		}

		client := entdb.NewTxClientFromRawConfig(ctx, *driver.GetConfig())
		rows, err := client.QueryContext(ctx, "SELECT pg_try_advisory_xact_lock($1)", int64(key.Hash64()))
		if err != nil {
			return struct{}{}, err
		}
		defer rows.Close()

		if !rows.Next() {
			return struct{}{}, fmt.Errorf("customer lock probe returned no row: %w", rows.Err())
		}

		var acquired bool
		if err := rows.Scan(&acquired); err != nil {
			return struct{}{}, err
		}

		if err := rows.Close(); err != nil {
			return struct{}{}, err
		}

		// If cancellation starts locking before hydration, this writer cannot
		// intervene. Avoid blocking the reader behind its own test callback.
		if !acquired {
			c.serialized = true
			return struct{}{}, nil
		}

		_, err = c.service.Cancel(ctx, c.subscriptionID, subscription.Timing{
			Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
		})

		return struct{}{}, err
	})
	if err == nil && !c.serialized {
		c.committed = true
	}

	return err
}

type cancelAfterItemRead struct {
	subscription.SubscriptionItemRepository
	cancellation *cancellationDuringViewRead
}

func (r *cancelAfterItemRead) GetForSubscriptionsAt(ctx context.Context, input []subscription.GetForSubscriptionAtInput) ([]subscription.SubscriptionItem, error) {
	items, err := r.SubscriptionItemRepository.GetForSubscriptionsAt(ctx, input)
	if err != nil {
		return nil, err
	}

	if err := r.cancellation.run(ctx); err != nil {
		return nil, err
	}

	return items, nil
}

type cancelBeforeEntitlementRead struct {
	entitlement.Service
	cancellation *cancellationDuringViewRead
}

func (s *cancelBeforeEntitlementRead) ListEntitlementsWithCustomer(ctx context.Context, input entitlement.ListEntitlementsParams) (entitlement.ListEntitlementsWithCustomerResult, error) {
	if err := s.cancellation.run(ctx); err != nil {
		return entitlement.ListEntitlementsWithCustomerResult{}, err
	}

	return s.Service.ListEntitlementsWithCustomer(ctx, input)
}

func TestSubscriptionViewConcurrentCancellation(t *testing.T) {
	for _, operation := range []string{"list", "cancel"} {
		for _, boundary := range []string{"after_item_read", "before_entitlement_read"} {
			t.Run(operation+"/"+boundary, func(t *testing.T) {
				// given a valid subscription with one entitlement-bearing item
				start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				clock.FreezeTime(start)
				defer clock.UnFreeze()

				ctx := t.Context()
				db := subscriptiontestutils.SetupDBDeps(t)
				defer db.Cleanup(t)

				deps := subscriptiontestutils.NewService(t, db)
				deps.FeatureConnector.CreateExampleFeatures(t, deps.ExampleMeterID)
				plan := deps.PlanHelper.CreatePlan(t, subscriptiontestutils.BuildTestPlanInput(t).
					AddPhase(nil, subscriptiontestutils.ExampleRateCard3ForAddons.Clone()).Build())
				before := subscriptiontestutils.CreateSubscriptionFromPlan(t, &deps, plan, start)
				key := subscriptiontestutils.ExampleFeatureKey2
				beforeItem := before.Phases[0].ItemsByKey[key][0]
				require.NotNil(t, beforeItem.Entitlement)

				clock.FreezeTime(start.Add(10 * 24 * time.Hour))
				defer clock.UnFreeze()

				repo := subscriptiontestutils.NewSubscriptionRepo(t, db)
				writer := &cancellationDuringViewRead{
					service:        deps.SubscriptionService,
					repo:           repo,
					subscriptionID: before.Subscription.NamespacedID,
					customerID:     before.Subscription.CustomerId,
				}

				items := deps.ItemRepo
				entitlements := deps.EntitlementRegistry.Entitlement
				switch boundary {
				case "after_item_read":
					items = &cancelAfterItemRead{SubscriptionItemRepository: items, cancellation: writer}
				case "before_entitlement_read":
					entitlements = &cancelBeforeEntitlementRead{Service: entitlements, cancellation: writer}
				}

				locker, err := lockr.NewLocker(&lockr.LockerConfig{Logger: testutils.NewLogger(t)})
				require.NoError(t, err)

				reader, err := subscriptionservice.New(subscriptionservice.ServiceConfig{
					SubscriptionRepo:      repo,
					SubscriptionPhaseRepo: subscriptiontestutils.NewSubscriptionPhaseRepo(t, db),
					SubscriptionItemRepo:  items,
					CustomerService:       deps.CustomerService,
					FeatureService:        deps.FeatureConnector,
					CurrencyResolver:      deps.CurrencyResolver,
					EntitlementAdapter:    subscriptionentitlement.NewSubscriptionEntitlementAdapter(entitlements, deps.ItemRepo, repo),
					TransactionManager:    repo,
					Publisher:             eventbus.NewMock(t),
					Lockr:                 locker,
					FeatureFlags:          ffx.NewTestContextService(ffx.AccessConfig{subscription.MultiSubscriptionEnabledFF: false}),
					TaxCode:               deps.TaxCodeEnv.Service,
					CostBasisService:      deps.CurrencyService,
				})
				require.NoError(t, err)

				// when another cancellation can commit between hydration queries
				var readErr error
				var listed pagination.Result[subscription.SubscriptionView]
				var canceled subscription.Subscription
				switch operation {
				case "list":
					listed, readErr = reader.ListViews(ctx, subscription.ListSubscriptionsInput{
						Namespaces: []string{before.Subscription.Namespace},
						Page:       pagination.NewPage(1, 10),
					})
				case "cancel":
					canceled, readErr = reader.Cancel(ctx, before.Subscription.NamespacedID, subscription.Timing{
						Enum: lo.ToPtr(subscription.TimingNextBillingCycle),
					})
				}

				// then committed storage remains valid, and the reader must not mix versions
				require.True(t, writer.attempted, "the read boundary must be exercised")
				require.True(t, writer.committed || writer.serialized, "the writer must commit or be excluded by the reader's lock")

				after, err := deps.SubscriptionService.GetView(ctx, before.Subscription.NamespacedID)
				require.NoError(t, err)
				require.NoError(t, after.Validate(true))
				if writer.committed {
					afterItem := after.Phases[0].ItemsByKey[key][0]
					require.NotEqual(t, beforeItem.SubscriptionItem.ID, afterItem.SubscriptionItem.ID)
					require.NotNil(t, afterItem.Entitlement)
					require.NotEqual(t, beforeItem.Entitlement.Entitlement.ID, afterItem.Entitlement.Entitlement.ID)
				}

				require.NoError(t, readErr, "a concurrent cancellation must not produce an invalid subscription view")
				switch operation {
				case "list":
					require.Len(t, listed.Items, 1)
					require.NoError(t, listed.Items[0].Validate(true))
				case "cancel":
					end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
					require.Equal(t, &end, canceled.ActiveTo)
				}
			})
		}
	}
}
