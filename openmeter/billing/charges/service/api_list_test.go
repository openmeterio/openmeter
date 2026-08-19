package service

import (
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/subscription"
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

	// The subscription expand is exercised against the side-loader directly:
	// a charge-borne subscription reference is FK-constrained to real
	// subscription rows, which this suite does not provision.
	subscriptionID := ulid.Make().String()
	s.FakeSubscriptionService.subscriptions = []subscription.Subscription{{
		NamespacedID: models.NamespacedID{Namespace: namespace, ID: subscriptionID},
		Name:         "api-list-subscription",
	}}
	defer func() { s.FakeSubscriptionService.subscriptions = nil }()

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

	s.Run("service period filters window the listing half-open", func() {
		// given:
		// - a second charge one month after the first (March vs May)
		// when:
		// - listing with the wire contract's operators: gte on the period
		//   start, lt on the period end
		// then:
		// - only charges inside the half-open [from, to) window match, and a
		//   period ending exactly on the lt bound is excluded
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

		marchOnly := newListInput(meta.ExpandNone)
		marchOnly.ServicePeriodFrom = &filter.FilterTime{Gte: lo.ToPtr(servicePeriod.From)}
		marchOnly.ServicePeriodTo = &filter.FilterTime{Lt: lo.ToPtr(mayPeriod.From)}

		result, err := s.Charges.ListCustomerCharges(ctx, marchOnly)
		require.NoError(s.T(), err)
		require.Len(s.T(), result.Charges.Items, 1)
		s.Equal(created[0].GetID(), result.Charges.Items[0].GetID())

		laterOnly := newListInput(meta.ExpandNone)
		laterOnly.ServicePeriodFrom = &filter.FilterTime{Gte: lo.ToPtr(servicePeriod.To.Add(14 * 24 * time.Hour))}

		result, err = s.Charges.ListCustomerCharges(ctx, laterOnly)
		require.NoError(s.T(), err)
		require.Len(s.T(), result.Charges.Items, 1)
		s.Equal(mayCharges[0].GetID(), result.Charges.Items[0].GetID())

		emptyWindow := newListInput(meta.ExpandNone)
		emptyWindow.ServicePeriodTo = &filter.FilterTime{Lt: lo.ToPtr(servicePeriod.To)}

		result, err = s.Charges.ListCustomerCharges(ctx, emptyWindow)
		require.NoError(s.T(), err)
		s.Empty(result.Charges.Items, "the window end is exclusive: a period ending exactly on the lt bound does not match")
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
