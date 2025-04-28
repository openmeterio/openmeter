package service_test

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

func TestAddonServiceGet(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")
	t.Run("Should use name and description of addon", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			p, add := createPlanWithAddon(
				t,
				deps,
				subscriptiontestutils.GetExamplePlanInput(t),
				subscriptiontestutils.GetExampleAddonInput(t, productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(now),
				}),
			)

			// Let's create a subscription
			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			spec1, err := subscription.NewSpecFromPlan(p, subscription.CreateSubscriptionCustomerInput{
				CustomerId: cust.ID,
				Currency:   "USD",
				ActiveFrom: now,
				Name:       "Test Subscription",
			})
			require.Nil(t, err)

			sub, err := deps.SubscriptionService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, spec1)
			require.Nil(t, err)

			aRCIDs := lo.Map(add.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs, 1)

			// Now, let's create a SubscriptionAddon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add.ID,
				SubscriptionID: sub.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}
			subAdd, err := deps.SubscriptionAddonService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.Nil(t, err)

			require.Equal(t, add.Name, subAdd.Name)
			require.Equal(t, add.Description, subAdd.Description)
		})
	})
}

func TestAddonServiceList(t *testing.T) {
	now := testutils.GetRFC3339Time(t, "2025-04-01T00:00:00Z")

	t.Run("Should error if input is formally invalid", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			inp := subscriptionaddon.ListSubscriptionAddonsInput{}

			expectedErr := inp.Validate()
			require.Error(t, expectedErr)

			_, err := deps.SubscriptionAddonService.List(context.Background(), subscriptiontestutils.ExampleNamespace, inp)
			require.Error(t, err)
			require.ErrorContains(t, err, expectedErr.Error())
		})
	})

	t.Run("Should return all addons for a subscription", func(t *testing.T) {
		withDeps(t, func(t *testing.T, deps subscriptiontestutils.SubscriptionDependencies) {
			clock.SetTime(now)
			defer clock.ResetTime()

			ctx := context.Background()

			_ = deps.FeatureConnector.CreateExampleFeatures(t)

			// Let's create a plan

			p, err := deps.PlanService.CreatePlan(context.Background(), subscriptiontestutils.GetExamplePlanInput(t))
			require.Nil(t, err)
			require.NotNil(t, p)

			// And let's create two addons for it

			per := productcatalog.EffectivePeriod{
				EffectiveFrom: lo.ToPtr(now),
			}

			add1 := deps.AddonService.CreateTestAddon(t, subscriptiontestutils.GetExampleAddonInput(t, per))

			aRCIDs1 := lo.Map(add1.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs1, 1)

			addInp := subscriptiontestutils.GetExampleAddonInput(t, per)
			addInp.Addon.AddonMeta.Key = "addon-2"

			add2 := deps.AddonService.CreateTestAddon(t, addInp)
			require.NoError(t, err)

			aRCIDs2 := lo.Map(add2.RateCards, func(rc addon.RateCard, _ int) string {
				return rc.ID
			})
			require.Len(t, aRCIDs2, 1)

			// Now let's link both to the plan

			_, err = deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
				NamespacedModel: models.NamespacedModel{
					Namespace: subscriptiontestutils.ExampleNamespace,
				},
				PlanID:        p.ID,
				AddonID:       add1.ID,
				FromPlanPhase: p.Phases[0].Key,
			})
			require.Nil(t, err, "received error: %s", err)

			_, err = deps.PlanAddonService.CreatePlanAddon(context.Background(), planaddon.CreatePlanAddonInput{
				NamespacedModel: models.NamespacedModel{
					Namespace: subscriptiontestutils.ExampleNamespace,
				},
				PlanID:        p.ID,
				AddonID:       add2.ID,
				FromPlanPhase: p.Phases[0].Key,
			})
			require.Nil(t, err, "received error: %s", err)

			// Now let's publish the plan

			p, err = deps.PlanService.PublishPlan(context.Background(), plan.PublishPlanInput{
				NamespacedID: p.NamespacedID,
				EffectivePeriod: productcatalog.EffectivePeriod{
					EffectiveFrom: lo.ToPtr(clock.Now()),
					EffectiveTo:   lo.ToPtr(testutils.GetRFC3339Time(t, "2099-01-01T00:00:00Z")),
				},
			})
			require.Nil(t, err, "received error: %s", err)

			// Let's create a subscription

			cust := deps.CustomerAdapter.CreateExampleCustomer(t)
			spec1, err := subscription.NewSpecFromPlan(&plansubscription.Plan{
				Plan: p.AsProductCatalogPlan(),
				Ref:  &p.NamespacedID,
			}, subscription.CreateSubscriptionCustomerInput{
				CustomerId: cust.ID,
				Currency:   "USD",
				ActiveFrom: now,
				Name:       "Test Subscription",
			})
			require.Nil(t, err)

			sub, err := deps.SubscriptionService.Create(context.Background(), subscriptiontestutils.ExampleNamespace, spec1)
			require.Nil(t, err)

			// Let's create a SubscriptionAddon for the first addon
			subAddonInp := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add1.ID,
				SubscriptionID: sub.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}
			subAdd1, err := deps.SubscriptionAddonService.Create(ctx, subscriptiontestutils.ExampleNamespace, subAddonInp)
			require.NoError(t, err)

			// Let's create a SubscriptionAddon for the second addon
			subAddonInp2 := subscriptionaddon.CreateSubscriptionAddonInput{
				AddonID:        add2.ID,
				SubscriptionID: sub.ID,
				InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{
					ActiveFrom: now,
					Quantity:   1,
				},
			}

			subAdd2, err := deps.SubscriptionAddonService.Create(ctx, subscriptiontestutils.ExampleNamespace, subAddonInp2)
			require.NoError(t, err)

			t.Run("Should return all addons for a subscription", func(t *testing.T) {
				listInp := subscriptionaddon.ListSubscriptionAddonsInput{
					SubscriptionID: sub.ID,
				}
				resp, err := deps.SubscriptionAddonService.List(ctx, subscriptiontestutils.ExampleNamespace, listInp)
				require.NoError(t, err)

				require.Len(t, resp.Items, 2)
				require.Equal(t, resp.TotalCount, 2)
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd1, resp.Items[0])
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd2, resp.Items[1])
			})

			t.Run("Should paginate returned addons", func(t *testing.T) {
				listInp := subscriptionaddon.ListSubscriptionAddonsInput{
					SubscriptionID: sub.ID,
					Page:           pagination.NewPage(1, 1),
				}
				resp, err := deps.SubscriptionAddonService.List(ctx, subscriptiontestutils.ExampleNamespace, listInp)
				require.NoError(t, err)

				require.Len(t, resp.Items, 1)
				require.Equal(t, resp.TotalCount, 2)
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd1, resp.Items[0])

				// Let's fetch the next page
				listInp.Page = pagination.NewPage(2, 1)
				resp, err = deps.SubscriptionAddonService.List(ctx, subscriptiontestutils.ExampleNamespace, listInp)
				require.NoError(t, err)

				require.Len(t, resp.Items, 1)
				require.Equal(t, resp.TotalCount, 2)
				subscriptiontestutils.SubscriptionAddonsEqual(t, *subAdd2, resp.Items[0])
			})
		})
	})
}
