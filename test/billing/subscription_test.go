package billing

import (
	"context"
	"log/slog"
	"testing"

	"github.com/alpacahq/alpacadecimal"
	"github.com/invopop/gobl/currency"
	"github.com/samber/lo"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/app"
	appsandbox "github.com/openmeterio/openmeter/openmeter/app/sandbox"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingworkersubscription "github.com/openmeterio/openmeter/openmeter/billing/worker/subscription"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/isodate"
	"github.com/openmeterio/openmeter/pkg/models"
)

type SubscriptionTestSuite struct {
	BaseSuite
	SubscriptionMixin

	SubscriptionSyncHandler *billingworkersubscription.Handler
}

func TestSubscription(t *testing.T) {
	suite.Run(t, new(SubscriptionTestSuite))
}

func (s *SubscriptionTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
	s.SubscriptionMixin.SetupSuite(s.T(), s.GetSubscriptionMixInDependencies())

	handler, err := billingworkersubscription.New(billingworkersubscription.Config{
		BillingService:      s.BillingService,
		Logger:              slog.Default(),
		TxCreator:           s.BillingAdapter,
		SubscriptionService: s.SubscriptionService,
	})
	s.NoError(err)
	s.NotNil(handler)

	s.SubscriptionSyncHandler = handler
}

func (s *SubscriptionTestSuite) TestDefaultProfileChange() {
	namespace := "ns-default-profile-change"
	ctx := context.Background()

	// Given we have a default profile

	defaultProfileSettings := MinimalCreateProfileInputTemplate
	defaultProfileSettings.Default = true
	defaultProfileSettings.Namespace = namespace

	s.InstallSandboxApp(s.T(), namespace)

	defaultProfile, err := s.BillingService.CreateProfile(context.Background(), defaultProfileSettings)
	s.NoError(err)
	s.NotNil(defaultProfile)

	// Given we have another non-default profile pinned to a different app
	appTypeOther := app.AppType("test-app-type-other")

	appFactoryOther, err := appsandbox.NewMockableFactory(s.T(), appsandbox.Config{
		AppService:     s.AppService,
		BillingService: s.BillingService,
	}, appsandbox.MockWithAppType(appTypeOther))
	s.NoError(err)
	s.NotNil(appFactoryOther)

	otherApp, err := s.AppService.CreateApp(ctx, app.CreateAppInput{
		Namespace: namespace,
		Name:      "test-app-other",
		Type:      appTypeOther,
	})
	s.NoError(err)
	s.NotNil(otherApp)

	otherAppReference := billing.AppReference{
		ID: otherApp.ID,
	}

	otherProfileSettings := MinimalCreateProfileInputTemplate
	otherProfileSettings.Namespace = namespace
	otherProfileSettings.Apps = billing.CreateProfileAppsInput{
		Tax:       otherAppReference,
		Invoicing: otherAppReference,
		Payment:   otherAppReference,
	}
	otherProfileSettings.Default = false

	otherProfile, err := s.BillingService.CreateProfile(context.Background(), otherProfileSettings)
	s.NoError(err)
	s.NotNil(otherProfile)

	// Given we have a paid and an unpaid plan
	paidPlan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:     "Test Plan",
				Key:      "paid-plan",
				Version:  1,
				Currency: currency.USD,
			},

			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "first-phase",
						Key:      "first-phase",
						Duration: nil,
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
								Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
									Amount:      alpacadecimal.NewFromFloat(5),
									PaymentTerm: productcatalog.InArrearsPaymentTerm,
								}),
							},
							BillingCadence: isodate.MustParse(s.T(), "P1D"),
						},
					},
				},
			},
		},
	})
	s.NoError(err)
	s.NotNil(paidPlan)

	subscriptionPaidPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, namespace, productcatalogsubscription.PlanRefInput{
		Key:     paidPlan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)
	s.NotNil(subscriptionPaidPlan)

	freePlan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{
			Namespace: namespace,
		},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:     "Test Plan",
				Key:      "free-plan",
				Version:  1,
				Currency: currency.USD,
			},

			Phases: []productcatalog.Phase{
				{
					PhaseMeta: productcatalog.PhaseMeta{
						Name:     "first-phase",
						Key:      "first-phase",
						Duration: nil,
					},
					RateCards: productcatalog.RateCards{
						&productcatalog.UsageBasedRateCard{
							RateCardMeta: productcatalog.RateCardMeta{
								Key:  "in-arrears",
								Name: "in-arrears",
							},
							BillingCadence: isodate.MustParse(s.T(), "P1D"),
						},
					},
				},
			},
		},
	})
	s.NoError(err)
	s.NotNil(freePlan)

	subscriptionFreePlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, namespace, productcatalogsubscription.PlanRefInput{
		Key:     freePlan.Key,
		Version: lo.ToPtr(1),
	})
	s.NoError(err)
	s.NotNil(subscriptionFreePlan)

	// Given we have 4 customers:
	// - 2 unpinned customers (one for paid and one for free subscription)
	// - 1 pinned customer to the default profile (paid plan)
	// - 1 pinned customer to the other profile (paid plan)

	// Unpinned paid plan customer
	unPinnedPaidPlanCustomer := s.createCustomerWithSubscription(ctx, namespace, "unPinnedPaidPlanCustomer", subscriptionPaidPlan)
	s.NotNil(unPinnedPaidPlanCustomer)

	// Unpinned free plan customer
	unPinnedFreePlanCustomer := s.createCustomerWithSubscription(ctx, namespace, "unPinnedFreePlanCustomer", subscriptionFreePlan)
	s.NotNil(unPinnedFreePlanCustomer)

	// Customer pinned to the default profile
	pinnedCustomerToDefaultProfileCustomer := s.createCustomerWithSubscription(ctx, namespace, "pinnedCustomerToDefaultProfile", subscriptionPaidPlan)
	s.NotNil(pinnedCustomerToDefaultProfileCustomer)

	override, err := s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  namespace,
		CustomerID: pinnedCustomerToDefaultProfileCustomer.ID,
		ProfileID:  defaultProfile.ID,
	})
	s.NoError(err)
	s.NotNil(override.CustomerOverride)
	s.Equal(defaultProfile.ID, override.CustomerOverride.Profile.ID)

	// Customer pinned to the other profile
	pinnedCustomerToOtherProfileCustomer := s.createCustomerWithSubscription(ctx, namespace, "pinnedCustomerToOtherProfile", subscriptionPaidPlan)
	s.NotNil(pinnedCustomerToOtherProfileCustomer)

	override, err = s.BillingService.UpsertCustomerOverride(ctx, billing.UpsertCustomerOverrideInput{
		Namespace:  namespace,
		CustomerID: pinnedCustomerToOtherProfileCustomer.ID,
		ProfileID:  otherProfile.ID,
	})
	s.NoError(err)
	s.NotNil(override.CustomerOverride)
	s.Equal(otherProfile.ID, override.CustomerOverride.Profile.ID)

	// When
	//   Changing the default profile to the "other" profile

	otherProfile.Default = true
	otherProfile.Apps = nil
	otherProfile.AppReferences = nil

	_, err = s.BillingService.UpdateProfile(ctx, billing.UpdateProfileInput(otherProfile.BaseProfile))
	s.NoError(err)

	// Then
	//   The profiles are updated properly
	otherProfile, err = s.BillingService.GetProfile(ctx, billing.GetProfileInput{
		Profile: otherProfile.ProfileID(),
	})
	s.NoError(err)
	s.NotNil(otherProfile)
	s.True(otherProfile.Default)

	oldDefaultProfile, err := s.BillingService.GetProfile(ctx, billing.GetProfileInput{
		Profile: defaultProfile.ProfileID(),
	})
	s.NoError(err)
	s.NotNil(oldDefaultProfile)
	s.False(oldDefaultProfile.Default)

	// Then
	//   unPinnedPaidPlanCustomer is pinned to the old profile
	customerOverride, err := s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: unPinnedPaidPlanCustomer.GetID(),
	})
	s.NoError(err)
	s.NotNil(customerOverride.CustomerOverride)
	s.Equal(oldDefaultProfile.ID, customerOverride.CustomerOverride.Profile.ID)

	// Then
	//  unPinnedFreePlanCustomer does not have customer overrides
	customerOverride, err = s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: unPinnedFreePlanCustomer.GetID(),
	})
	s.NoError(err)
	s.Nil(customerOverride.CustomerOverride)

	// Then
	//   pinnedCustomerToDefaultProfileCustomer is pinned to the old default profile
	customerOverride, err = s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: pinnedCustomerToDefaultProfileCustomer.GetID(),
	})
	s.NoError(err)
	s.NotNil(customerOverride.CustomerOverride)
	s.Equal(oldDefaultProfile.ID, customerOverride.CustomerOverride.Profile.ID)

	// Then
	//   pinnedCustomerToOtherProfileCustomer is pinned to the new profile
	customerOverride, err = s.BillingService.GetCustomerOverride(ctx, billing.GetCustomerOverrideInput{
		Customer: pinnedCustomerToOtherProfileCustomer.GetID(),
	})
	s.NoError(err)
	s.NotNil(customerOverride.CustomerOverride)
	s.Equal(otherProfile.ID, customerOverride.CustomerOverride.Profile.ID)
}

func (s *SubscriptionTestSuite) createCustomerWithSubscription(ctx context.Context, namespace string, customerKey string, plan subscription.Plan) *customer.Customer { //nolint:unparam
	cust, err := s.CustomerService.CreateCustomer(ctx, customer.CreateCustomerInput{
		Namespace: namespace,

		CustomerMutate: customer.CustomerMutate{
			Name: customerKey,
			UsageAttribution: customer.CustomerUsageAttribution{
				SubjectKeys: []string{customerKey},
			},
		},
	})
	s.NoError(err)
	s.NotNil(cust)

	subsView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscription.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscription.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{
				Custom: lo.ToPtr(clock.Now()),
			},
		},
		Namespace:  namespace,
		CustomerID: cust.ID,
	}, plan)
	s.NoError(err)
	s.NotNil(subsView)

	s.NoError(s.SubscriptionSyncHandler.SyncronizeSubscription(ctx, subsView, clock.Now()))

	return cust
}
