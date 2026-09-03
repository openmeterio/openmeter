package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	productcatalogsubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/filter"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

func TestCustomerChargeAPIList(t *testing.T) {
	suite.Run(t, new(CustomerChargeAPIListTestSuite))
}

type CustomerChargeAPIListTestSuite struct {
	BaseSuite
}

func (s *CustomerChargeAPIListTestSuite) TestListCustomerChargesExpands() {
	// given:
	// - a subscription-managed usage-based charge referencing a feature by key
	//   and a subscription served by the fake subscription service
	// when:
	// - the facade lists the customer's charges with and without expands
	// then:
	// - the resolved realization view is always attached, side-loaded entities
	//   are ID-only stubs without the expand and fully loaded with it
	ctx := s.T().Context()
	servicePeriod := timeutil.ClosedPeriod{
		From: datetime.MustParseTimeInLocation(s.T(), "2027-03-01T00:00:00Z", time.UTC).AsTime(),
		To:   datetime.MustParseTimeInLocation(s.T(), "2027-04-01T00:00:00Z", time.UTC).AsTime(),
	}
	clock.FreezeTime(servicePeriod.From.Add(-time.Hour))
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("charges-service-api-list")
	s.ProvisionDefaultTaxCodes(ctx, namespace)
	cust := s.CreateTestCustomer(namespace, "api-list")
	sandboxApp := s.InstallSandboxApp(s.T(), namespace)
	_ = s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	feat := s.SetupApiRequestsTotalFeature(ctx, namespace)

	// The subscription expand is exercised against the side-loader directly
	// with a subscription created through the real subscription stack; the
	// charge intents below do not reference it.
	testPlan, err := s.PlanService.CreatePlan(ctx, plan.CreatePlanInput{
		NamespacedModel: models.NamespacedModel{Namespace: namespace},
		Plan: productcatalog.Plan{
			PlanMeta: productcatalog.PlanMeta{
				Name:           "api-list-plan",
				Key:            "api-list-plan",
				Version:        1,
				Currency:       currencies.NewCurrencyReference(USD),
				BillingCadence: datetime.MustParseDuration(s.T(), "P1M"),
				ProRatingConfig: productcatalog.ProRatingConfig{
					Enabled: true,
					Mode:    productcatalog.ProRatingModeProratePrices,
				},
			},
			Phases: []productcatalog.Phase{{
				PhaseMeta: productcatalog.PhaseMeta{Name: "first-phase", Key: "first-phase"},
				RateCards: productcatalog.RateCards{
					&productcatalog.FlatFeeRateCard{
						RateCardMeta: productcatalog.RateCardMeta{
							Key:  "flat",
							Name: "flat",
							Price: productcatalog.NewPriceFrom(productcatalog.FlatPrice{
								Amount:      alpacadecimal.NewFromInt(5),
								PaymentTerm: productcatalog.InArrearsPaymentTerm,
							}),
						},
						BillingCadence: lo.ToPtr(datetime.MustParseDuration(s.T(), "P1M")),
					},
				},
			}},
		},
	})
	require.NoError(s.T(), err)

	subscriptionPlan, err := s.SubscriptionPlanAdapter.GetVersion(ctx, namespace, productcatalogsubscription.PlanRefInput{
		Key:     testPlan.Key,
		Version: lo.ToPtr(1),
	})
	require.NoError(s.T(), err)

	subscriptionView, err := s.SubscriptionWorkflowService.CreateFromPlan(ctx, subscriptionworkflow.CreateSubscriptionWorkflowInput{
		ChangeSubscriptionWorkflowInput: subscriptionworkflow.ChangeSubscriptionWorkflowInput{
			Timing: subscription.Timing{Custom: lo.ToPtr(clock.Now())},
			Name:   "api-list-subscription",
		},
		Namespace:  namespace,
		CustomerID: cust.ID,
	}, subscriptionPlan)
	require.NoError(s.T(), err)
	subscriptionID := subscriptionView.Subscription.ID

	created, err := s.Charges.Create(ctx, charges.CreateInput{
		Namespace: namespace,
		Intents: charges.ChargeIntents{
			s.createMockChargeIntent(createMockChargeIntentInput{
				customer:          cust.GetID(),
				currency:          USD,
				servicePeriod:     servicePeriod,
				settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
				price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				featureKey:        feat.Feature.Key,
				name:              "api-list-usage-based",
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: "api-list-usage-based",
			}),
		},
	})
	require.NoError(s.T(), err)
	require.Len(s.T(), created, 1)

	newListInput := func(expands meta.Expands) charges.ListCustomerChargesInput {
		return charges.ListCustomerChargesInput{
			ListChargesInput: charges.ListChargesInput{
				Page:        pagination.NewPage(1, 10),
				Namespace:   namespace,
				CustomerIDs: []string{cust.ID},
				ChargeTypes: []meta.ChargeType{meta.ChargeTypeFlatFee, meta.ChargeTypeUsageBased},
				Expands:     expands,
			},
		}
	}
	s.Run("without expands the charges carry the resolved view and no expanded entities", func() {
		result, err := s.Charges.ListCustomerCharges(ctx, newListInput(meta.ExpandNone))
		require.NoError(s.T(), err)

		s.True(result.Expands.Has(meta.ExpandRealizations), "realizations are always loaded")
		s.True(result.Expands.Has(meta.ExpandDeletedRealizations), "deleted runs are always loaded for voided history")

		require.Len(s.T(), result.Charges.Items, 1)
		item := result.Charges.Items[0]

		require.Len(s.T(), item.UsageBasedRealizations, 1)
		s.Nil(item.UsageBasedRealizations[0].Run, "a fresh charge only has the outstanding projection")
		s.True(servicePeriod.From.Equal(item.UsageBasedRealizations[0].ServicePeriod.From))
		s.True(servicePeriod.To.Equal(item.UsageBasedRealizations[0].ServicePeriod.To))
		s.Nil(item.UsageBasedRealizations[0].Invoice, "invoices are only attached under the expand")

		s.Nil(item.Customer, "expanded members stay nil without their expand")
		s.Nil(item.Feature)
		s.Nil(item.Subscription)
	})

	s.Run("with expands the charges carry the full entities", func() {
		result, err := s.Charges.ListCustomerCharges(ctx, newListInput(meta.Expands{
			meta.ExpandCustomer,
			meta.ExpandFeature,
			meta.ExpandSubscription,
			meta.ExpandRealizationInvoice,
		}))
		require.NoError(s.T(), err)

		require.Len(s.T(), result.Charges.Items, 1)
		item := result.Charges.Items[0]

		require.NotNil(s.T(), item.Customer)
		s.Equal(cust.ID, item.Customer.ID)
		s.Equal(cust.Name, item.Customer.Name)

		require.NotNil(s.T(), item.Feature, "created charges resolve their feature by key")
		s.Equal(feat.Feature.ID, item.Feature.ID)
		s.Equal(feat.Feature.Name, item.Feature.Name)

		s.Nil(item.Subscription, "the charge references no subscription")
	})

	s.Run("the subscription side-loader serves the facade's bulk lookup", func() {
		full, err := s.Charges.listCustomerChargeSubscriptions(ctx, namespace, cust.ID, []string{subscriptionID})
		require.NoError(s.T(), err)
		require.Contains(s.T(), full, subscriptionID)
		s.Equal("api-list-subscription", full[subscriptionID].Name)
	})

	s.Run("customer scoping is required", func() {
		input := newListInput(meta.ExpandNone)
		input.CustomerIDs = nil

		_, err := s.Charges.ListCustomerCharges(ctx, input)
		s.ErrorContains(err, "customer")
	})

	s.Run("only wire-supported charge types are accepted", func() {
		input := newListInput(meta.ExpandNone)
		input.ChargeTypes = []meta.ChargeType{meta.ChargeTypeCreditPurchase}

		_, err := s.Charges.ListCustomerCharges(ctx, input)
		s.ErrorContains(err, "unsupported charge type")
	})

	s.Run("service period filters apply per column with any operator", func() {
		// given:
		// - a second charge one month after the first (March vs May)
		// when:
		// - listing with service-period filters composed by the caller
		// then:
		// - each filter applies independently to its own bound, so the
		//   caller can express containment, one-sided, or overlap queries
		mayPeriod := timeutil.ClosedPeriod{
			From: datetime.MustParseTimeInLocation(s.T(), "2027-05-01T00:00:00Z", time.UTC).AsTime(),
			To:   datetime.MustParseTimeInLocation(s.T(), "2027-06-01T00:00:00Z", time.UTC).AsTime(),
		}
		mayCharges, err := s.Charges.Create(ctx, charges.CreateInput{
			Namespace: namespace,
			Intents: charges.ChargeIntents{
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          cust.GetID(),
					currency:          USD,
					servicePeriod:     mayPeriod,
					settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					featureKey:        feat.Feature.Key,
					name:              "api-list-usage-based-may",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-list-usage-based-may",
				}),
			},
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), mayCharges, 1)

		// Containment: from >= Mar 1 and to < May 1 returns only March.
		contained := newListInput(meta.ExpandNone)
		contained.ServicePeriodFrom = &filter.FilterTime{Gte: lo.ToPtr(servicePeriod.From)}
		contained.ServicePeriodTo = &filter.FilterTime{Lt: lo.ToPtr(mayPeriod.From)}

		result, err := s.Charges.ListCustomerCharges(ctx, contained)
		require.NoError(s.T(), err)
		require.Len(s.T(), result.Charges.Items, 1)
		s.Equal(created[0].GetID(), result.Charges.Items[0].GetID())

		// One-sided: charges starting on or after mid-April return only May.
		laterOnly := newListInput(meta.ExpandNone)
		laterOnly.ServicePeriodFrom = &filter.FilterTime{Gte: lo.ToPtr(servicePeriod.To.Add(14 * 24 * time.Hour))}

		result, err = s.Charges.ListCustomerCharges(ctx, laterOnly)
		require.NoError(s.T(), err)
		require.Len(s.T(), result.Charges.Items, 1)
		s.Equal(mayCharges[0].GetID(), result.Charges.Items[0].GetID())

		// Operators are honored literally: lt on the end bound excludes a
		// period ending exactly on it.
		endExclusive := newListInput(meta.ExpandNone)
		endExclusive.ServicePeriodTo = &filter.FilterTime{Lt: lo.ToPtr(servicePeriod.To)}

		result, err = s.Charges.ListCustomerCharges(ctx, endExclusive)
		require.NoError(s.T(), err)
		s.Empty(result.Charges.Items)

		// Overlap with a window strictly inside March (Mar 15–20): neither
		// charge is contained in it, but March overlaps it.
		// Overlap of [from, to) with [windowStart, windowEnd) holds when
		// from < windowEnd and to > windowStart.
		windowStart := servicePeriod.From.Add(14 * 24 * time.Hour)
		windowEnd := servicePeriod.From.Add(19 * 24 * time.Hour)
		insideMarch := newListInput(meta.ExpandNone)
		insideMarch.ServicePeriodFrom = &filter.FilterTime{Lt: lo.ToPtr(windowEnd)}
		insideMarch.ServicePeriodTo = &filter.FilterTime{Gt: lo.ToPtr(windowStart)}

		result, err = s.Charges.ListCustomerCharges(ctx, insideMarch)
		require.NoError(s.T(), err)
		require.Len(s.T(), result.Charges.Items, 1)
		s.Equal(created[0].GetID(), result.Charges.Items[0].GetID(), "the March charge overlaps the window without being contained in it")
	})

	s.Run("feature filters scope the listing", func() {
		// given:
		// - both charges reference the api-requests feature, created by key
		//   (Create resolves and persists the feature ID)
		// when:
		// - listing filtered by feature id and by feature key
		// then:
		// - the matching feature returns every charge, a foreign one none
		byID := newListInput(meta.ExpandNone)
		byID.FeatureID = &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(feat.Feature.ID)}}

		result, err := s.Charges.ListCustomerCharges(ctx, byID)
		require.NoError(s.T(), err)
		s.Len(result.Charges.Items, 2)

		byKey := newListInput(meta.ExpandNone)
		byKey.FeatureKey = &filter.FilterString{In: lo.ToPtr([]string{feat.Feature.Key, "another-feature"})}

		result, err = s.Charges.ListCustomerCharges(ctx, byKey)
		require.NoError(s.T(), err)
		s.Len(result.Charges.Items, 2)

		foreign := newListInput(meta.ExpandNone)
		foreign.FeatureKey = &filter.FilterString{Eq: lo.ToPtr("some-other-feature")}

		result, err = s.Charges.ListCustomerCharges(ctx, foreign)
		require.NoError(s.T(), err)
		s.Empty(result.Charges.Items)
	})

	s.Run("deleted customers still expand", func() {
		// given:
		// - the charge's customer is deleted after the charge was created
		// when:
		// - the facade lists with the customer expand
		// then:
		// - the listing succeeds and the deleted customer is still expanded,
		//   which is why the loader goes through ListCustomers+IncludeDeleted
		//   instead of GetCustomer
		// A customer with an active subscription cannot be deleted, so the
		// subscription is canceled first.
		_, err := s.SubscriptionService.Cancel(ctx, models.NamespacedID{Namespace: namespace, ID: subscriptionID}, subscription.Timing{Custom: lo.ToPtr(clock.Now())})
		require.NoError(s.T(), err)
		require.NoError(s.T(), s.CustomerService.DeleteCustomer(ctx, cust.GetID()))

		result, err := s.Charges.ListCustomerCharges(ctx, newListInput(meta.Expands{meta.ExpandCustomer}))
		require.NoError(s.T(), err)

		require.NotEmpty(s.T(), result.Charges.Items)
		for _, item := range result.Charges.Items {
			require.NotNil(s.T(), item.Customer)
			s.Equal(cust.ID, item.Customer.ID)
			s.NotNil(item.Customer.DeletedAt)
		}
	})
}
