package subscription_test

import (
	"context"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	pcsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	subscription "github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestSubWithMeteredEntitlement(t *testing.T) {
	// Let's declare our variables
	// note: this namespace is hardcoded in the test framework
	namespace := "test-namespace"

	startOfSub := testutils.GetRFC3339Time(t, "2025-06-15T12:00:00Z")
	currentTime := startOfSub
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tDeps := setup(t, setupConfig{})
	defer tDeps.cleanup(t)

	clock.SetTime(currentTime)

	// 1st, let's create the features
	feats := tDeps.FeatureConnector.CreateExampleFeatures(t)
	require.Len(t, feats, 3)

	// 2nd, let's create the plan
	p, err := tDeps.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "Test Plan",
				Key:            "test_plan",
				Currency:       "USD",
				BillingCadence: datetime.MustParseDuration(t, "P1M"), // Let's do monthly billing
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:      "first",
						Name:     "First Phase",
						Duration: lo.ToPtr(datetime.MustParseDuration(t, "P1W")),
					},
					RateCards: productcatalog.RateCards{
						// Let's have an in-arrears monthly entitlement ratecard
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        feats[0].Key,
								Name:       "Test Rate Card",
								FeatureKey: lo.ToPtr(feats[0].Key),
								FeatureID:  lo.ToPtr(feats[0].ID),
								TaxConfig: &productcatalog.TaxConfig{
									Stripe: &productcatalog.StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
									UsagePeriod:     datetime.MustParseDuration(t, "P1M"), // compatible with the billing cadence
									IssueAfterReset: lo.ToPtr(10.0),
								}),
							},
							BillingCadence: datetime.MustParseDuration(t, "P1M"),
						},
					},
				},
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Key:  "second",
						Name: "Second Phase",
					},
					RateCards: productcatalog.RateCards{
						// Let's have an in-arrears monthly entitlement ratecard
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:        feats[0].Key,
								Name:       "Test Rate Card",
								FeatureKey: lo.ToPtr(feats[0].Key),
								FeatureID:  lo.ToPtr(feats[0].ID),
								Price: productcatalog.NewPriceFrom(productcatalog.UnitPrice{
									Amount: alpacadecimal.NewFromInt(100),
								}),
								TaxConfig: &productcatalog.TaxConfig{
									Stripe: &productcatalog.StripeTaxConfig{
										Code: "txcd_10000000",
									},
								},
								EntitlementTemplate: productcatalog.NewEntitlementTemplateFrom(productcatalog.MeteredEntitlementTemplate{
									UsagePeriod:     datetime.MustParseDuration(t, "P1M"), // compatible with the billing cadence
									IssueAfterReset: lo.ToPtr(100.0),
								}),
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

	// 3rd, let's create the billing profile
	_, err = tDeps.billingService.CreateProfile(ctx, minimalCreateProfileInputTemplate(tDeps.sandboxApp.GetID()))
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

	// 5th, let's create the subscription
	s, err := tDeps.pcSubscriptionService.Create(ctx, pcsubscription.CreateSubscriptionRequest{
		WorkflowInput: subscriptionworkflow.CreateSubscriptionWorkflowInput{
			Namespace:  namespace,
			CustomerID: c.ID,
			ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
				Timing: subscription.Timing{
					Custom: &startOfSub,
				},
				Name: "Test Subscription",
			},
			BillingAnchor: nil, // We align billing to subscription start (this is somewhat problematic with trials...)
		},
		PlanInput: *pi,
	})
	require.NoError(t, err) // THIS IS THE TEST, it used to fail
	require.NotNil(t, s)
}
