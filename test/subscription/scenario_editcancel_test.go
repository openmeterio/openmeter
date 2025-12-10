package subscription_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pcsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription/patch"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestEditingAndCanceling(t *testing.T) {
	// Let's declare our variables
	// note: this namespace is hardcoded in the test framework
	namespace := "test-namespace"

	currentTime := testutils.GetRFC3339Time(t, "2025-01-20T13:11:07Z")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tDeps := setup(t, setupConfig{})
	defer tDeps.cleanup(t)

	clock.SetTime(currentTime)

	// First, let's create the features
	f, err := tDeps.FeatureConnector.CreateFeature(ctx, feature.CreateFeatureInputs{
		Name:      "Example Feature",
		Key:       "test_feature_1",
		Namespace: namespace,
		MeterSlug: lo.ToPtr(subscriptiontestutils.ExampleFeatureMeterSlug),
	})
	require.NoError(t, err)

	// Second, let's create the plan
	p, err := tDeps.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test_plan",
				Currency:       "USD",
				BillingCadence: datetime.MustParseDuration(t, "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:      "default",
						Name:     "Default Phase",
						Duration: nil,
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        "test_feature_1",
								Name:       "Test Rate Card",
								FeatureKey: lo.ToPtr(f.Key),
								FeatureID:  lo.ToPtr(f.ID),
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromInt(100),
								}),
								TaxConfig: &productcatalog.TaxConfig{
									Stripe: &productcatalog.StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
							},
							BillingCadence: datetime.MustParseDuration(t, "P1M"),
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	p, err = tDeps.PlanService.PublishPlan(ctx, plan.PublishPlanInput{
		NamespacedID: p.NamespacedID,
		EffectivePeriod: productcatalog.EffectivePeriod{
			EffectiveFrom: lo.ToPtr(currentTime),
		},
	})
	require.NoError(t, err)

	// Then create the customer
	c, err := tDeps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,
		CustomerMutate: customer.CustomerMutate{
			Name: "Test Customer",
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject_1"},
			},
		},
	})
	require.NoError(t, err)

	pi := &pcsubscription.PlanInput{}
	pi.FromRef(&pcsubscription.PlanRefInput{
		Key:     p.Key,
		Version: &p.Version,
	})

	// And let's create extra customers (with their subjects)
	custs := []*customer.Customer{}
	for i := 0; i < 10; i++ {
		// Then create customer
		c, err := tDeps.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
			Namespace: namespace,
			CustomerMutate: customer.CustomerMutate{
				Name: "Test Customer",
				UsageAttribution: &customer.CustomerUsageAttribution{
					SubjectKeys: []string{fmt.Sprintf("subject_%d", i+2)},
				},
			},
		})
		require.NoError(t, err)
		custs = append(custs, c)
	}

	// Fourth, let's create the subscription
	s, err := tDeps.pcSubscriptionService.Create(ctx, pcsubscription.CreateSubscriptionRequest{
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			Namespace:  namespace,
			CustomerID: c.ID,
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &currentTime,
				},
				Name: "Test Subscription",
			},
		},
		PlanInput: *pi,
	})
	require.NoError(t, err)
	require.NotNil(t, s)

	// And let's subscribe the extra customers
	for _, cust := range custs {
		s, err := tDeps.pcSubscriptionService.Create(ctx, pcsubscription.CreateSubscriptionRequest{
			WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
				Namespace:  namespace,
				CustomerID: cust.ID,
				ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
					Timing: subscription.Timing{
						Custom: &currentTime,
					},
					Name: "Test Subscription",
				},
			},
			PlanInput: *pi,
		})
		require.NoError(t, err)
		require.NotNil(t, s)
	}

	currentTime = currentTime.Add(time.Minute)
	clock.SetTime(currentTime)

	// Fifth, let's edit the subscription
	_, err = tDeps.subscriptionWorkflowService.EditRunning(ctx, s.NamespacedID, []subscription.Patch{
		patch.PatchRemoveItem{
			ItemKey:  "test_feature_1",
			PhaseKey: "default",
		},
		patch.PatchAddItem{
			PhaseKey: "default",
			ItemKey:  "test_feature_1",
			CreateInput: subscription.SubscriptionItemSpec{
				CreateSubscriptionItemInput: subscription.CreateSubscriptionItemInput{
					CreateSubscriptionItemPlanInput: subscription.CreateSubscriptionItemPlanInput{
						PhaseKey: "default",
						ItemKey:  "test_feature_1",
						RateCard: &productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Name:                "Test Rate Card",
								FeatureKey:          lo.ToPtr("test_feature_1"),
								Key:                 "test_feature_1",
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.BooleanEntitlementTemplate{}),
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromInt(101),
								}),
								TaxConfig: &productcatalog.TaxConfig{
									Stripe: &productcatalog.StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
							},
							BillingCadence: datetime.MustParseDuration(t, "P1M"),
						},
					},
					CreateSubscriptionItemCustomerInput: subscription.CreateSubscriptionItemCustomerInput{},
				},
			},
		},
	}, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	})
	require.NoError(t, err)

	currentTime = currentTime.Add(time.Minute)
	clock.SetTime(currentTime)

	// Sixth, let's cancel the subscription
	_, err = tDeps.subscriptionService.Cancel(ctx, s.NamespacedID, subscription.Timing{
		Enum: lo.ToPtr(subscription.TimingImmediate),
	})
	require.NoError(t, err)
}
