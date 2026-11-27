package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/billing/charges"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	billingfeaturemeter "github.com/openmeterio/openmeter/openmeter/billing/featuremeter"
	billingfeaturemeterservice "github.com/openmeterio/openmeter/openmeter/billing/featuremeter/service"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
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

type customerChargeFeatureServiceMock struct {
	mock.Mock
}

func (m *customerChargeFeatureServiceMock) ListFeatures(ctx context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	args := m.Called(ctx, params)

	return args.Get(0).(pagination.Result[feature.Feature]), args.Error(1)
}

type customerChargeMeterServiceStub struct{}

func (customerChargeMeterServiceStub) ListMeters(context.Context, meter.ListMetersParams) (pagination.Result[meter.Meter], error) {
	return pagination.Result[meter.Meter]{}, nil
}

func (s *CustomerChargeAPIListTestSuite) TestFeatureExpansionValidationPolicy() {
	const namespace = "namespace"

	s.Run("meterless feature remains expandable", func() {
		// given:
		// - a usage-based charge references an existing feature without a meter
		featureEntity := feature.Feature{ID: "feature-id", Key: "meterless-feature"}
		service := newCustomerChargeFeatureTestService(s.T(), []feature.Feature{featureEntity}, nil)
		reference := charges.NewCharge(usagebased.Charge{ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{ID: "charge-id"},
			Intent:          usagebased.Intent{FeatureKey: featureEntity.Key}.AsOverridableIntent(),
			Status:          usagebased.StatusCreated,
		}})

		// when:
		// - the customer charge facade expands the feature
		references, err := collectCustomerChargeReferences(charges.Charges{reference})
		require.NoError(s.T(), err)
		entities, err := service.loadCustomerChargeEntities(
			s.T().Context(),
			namespace,
			references,
			meta.Expands{meta.ExpandFeature},
		)

		// then:
		// - feature-only resolution does not require a meter
		require.NoError(s.T(), err)
		featureMeter, err := entities.featureMeters.Get(billingfeaturemeter.WithoutMeters(reference))
		require.NoError(s.T(), err)
		require.Equal(s.T(), featureEntity, featureMeter.Feature)
	})

	s.Run("catalog system error aborts expansion", func() {
		// given:
		// - the feature catalog fails while resolving a charge reference
		resolverErr := errors.New("feature service unavailable")
		service := newCustomerChargeFeatureTestService(s.T(), nil, resolverErr)
		reference := charges.NewCharge(usagebased.Charge{ChargeBase: usagebased.ChargeBase{
			ManagedResource: meta.ManagedResource{ID: "charge-id"},
			Intent:          usagebased.Intent{FeatureKey: "feature-key"}.AsOverridableIntent(),
			Status:          usagebased.StatusCreated,
		}})

		// when:
		// - the customer charge facade expands the feature
		references, err := collectCustomerChargeReferences(charges.Charges{reference})
		require.NoError(s.T(), err)
		_, err = service.loadCustomerChargeEntities(
			s.T().Context(),
			namespace,
			references,
			meta.Expands{meta.ExpandFeature},
		)

		// then:
		// - the operational failure remains an error instead of being treated as best-effort validation
		require.ErrorIs(s.T(), err, resolverErr)
	})
}

func newCustomerChargeFeatureTestService(t *testing.T, features []feature.Feature, featureServiceErr error) *service {
	t.Helper()

	featureService := &customerChargeFeatureServiceMock{}
	featureService.Test(t)
	featureService.On("ListFeatures", mock.Anything, mock.Anything).
		Return(pagination.Result[feature.Feature]{Items: features}, featureServiceErr).
		Once()
	t.Cleanup(func() {
		featureService.AssertExpectations(t)
	})

	resolver, err := billingfeaturemeterservice.New(billingfeaturemeterservice.Config{
		FeatureService: featureService,
		MeterService:   customerChargeMeterServiceStub{},
		Logger:         slog.Default(),
	})
	require.NoError(t, err)

	return &service{
		logger:               slog.New(slog.DiscardHandler),
		featureMeterResolver: resolver,
	}
}

func (s *CustomerChargeAPIListTestSuite) TestListCustomerChargesRealtimeUsage() {
	t := s.T()
	ctx := t.Context()

	// Given a mix of persisted charge statuses and overridden service periods.
	now := time.Date(2027, 3, 15, 12, 0, 0, 0, time.UTC)
	// Create the charges before any period starts so create-time advancement leaves them untouched.
	clock.FreezeTime(now.Add(-3 * time.Hour))
	defer clock.UnFreeze()

	namespace := s.GetUniqueNamespace("charges-realtime-usage")
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(t, namespace)
	cust := s.CreateTestCustomer(namespace, "charges-realtime-usage")
	sandboxApp := s.InstallSandboxApp(t, namespace)
	s.ProvisionBillingProfile(ctx, namespace, sandboxApp.GetID())
	feat := s.SetupApiRequestsTotalFeature(ctx, namespace)
	defer feat.Cleanup()

	current := timeutil.ClosedPeriod{From: now.Add(-time.Hour), To: now.Add(time.Hour)}
	starting := timeutil.ClosedPeriod{From: now, To: now.Add(time.Hour)}
	ending := timeutil.ClosedPeriod{From: now.Add(-time.Hour), To: now}
	past := timeutil.ClosedPeriod{From: now.Add(-2 * time.Hour), To: now.Add(-time.Hour)}
	future := timeutil.ClosedPeriod{From: now.Add(time.Hour), To: now.Add(2 * time.Hour)}
	overrideCurrent := timeutil.ClosedPeriod{From: now.Add(-30 * time.Minute), To: now.Add(30 * time.Minute)}

	type chargeFixture struct {
		name              string
		status            usagebased.Status
		period            timeutil.ClosedPeriod
		override          *timeutil.ClosedPeriod
		overrideDeleted   bool
		wantRealtimeUsage bool
		// quantity is distinct per eligible fixture so a result written back to
		// the wrong charge fails the assertions.
		quantity float64
	}
	fixtures := []chargeFixture{
		{name: "active", status: usagebased.StatusActive, period: current, wantRealtimeUsage: true, quantity: 3},
		{name: "awaiting payment after period", status: usagebased.StatusActiveAwaitingPaymentSettlement, period: past, wantRealtimeUsage: true, quantity: 5},
		{name: "created at start", status: usagebased.StatusCreated, period: starting, wantRealtimeUsage: true, quantity: 7},
		{name: "created at end", status: usagebased.StatusCreated, period: ending},
		{name: "created in future", status: usagebased.StatusCreated, period: future},
		{name: "override moves into period", status: usagebased.StatusCreated, period: future, override: &overrideCurrent, wantRealtimeUsage: true, quantity: 11},
		{name: "override moves out of period", status: usagebased.StatusCreated, period: current, override: &future},
		{name: "final", status: usagebased.StatusFinal, period: current},
		{name: "deleted", status: usagebased.StatusDeleted, period: current},
		{name: "effective intent deleted", status: usagebased.StatusActive, period: current, override: &current, overrideDeleted: true},
	}

	chargeIDs := make(map[string]meta.ChargeID, len(fixtures))
	fixturesByID := make(map[string]chargeFixture, len(fixtures))
	var wantPeriods []timeutil.ClosedPeriod
	for _, fixture := range fixtures {
		created, err := s.Charges.Create(ctx, charges.CreateInput{
			ChangeSource: billing.ChangeSourceSystem,
			Namespace:    namespace,
			Intents: charges.NewCreateChargeIntents(s.createMockChargeIntent(createMockChargeIntentInput{
				customer:          cust.GetID(),
				currency:          USD,
				servicePeriod:     fixture.period,
				price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
				featureKey:        feat.Feature.Key,
				name:              fixture.name,
				managedBy:         billing.SubscriptionManagedLine,
				uniqueReferenceID: fixture.name,
			})),
		})
		require.NoError(t, err, fixture.name)
		require.Len(t, created, 1, fixture.name)
		charge, err := created[0].AsUsageBasedCharge()
		require.NoError(t, err, fixture.name)
		chargeIDs[fixture.name] = charge.GetChargeID()
		fixturesByID[charge.ID] = fixture

		// Set the lifecycle state directly so the fixture isolates what the read observes.
		if fixture.status == usagebased.StatusDeleted {
			require.NoError(t, s.UsageBasedAdapter.DeleteCharge(ctx, charge), fixture.name)
		} else if charge.Status != fixture.status {
			charge.Status = fixture.status
			charge.State.FeatureID = feat.Feature.ID
			charge.ChargeBase, err = s.UsageBasedAdapter.UpdateCharge(ctx, charge.ChargeBase)
			require.NoError(t, err, fixture.name)
		}

		effectivePeriod := fixture.period
		if fixture.override != nil {
			effectivePeriod = *fixture.override
			override := charge.Intent.GetBaseIntent().IntentMutableFields
			override.ServicePeriod = effectivePeriod
			override.FullServicePeriod = effectivePeriod
			override.BillingPeriod = effectivePeriod
			override.InvoiceAt = effectivePeriod.To
			if fixture.overrideDeleted {
				override.IntentDeletedAt = lo.ToPtr(clock.Now())
			}

			_, err := s.UsageBasedAdapter.CreateChargeOverride(ctx, charge.ChargeBase, override)
			require.NoError(t, err, fixture.name)
		}

		if fixture.wantRealtimeUsage {
			wantPeriods = append(wantPeriods, effectivePeriod)
			s.MockStreamingConnector.AddRow(feat.Feature.Key, meter.MeterQueryRow{
				Value:       fixture.quantity,
				WindowStart: effectivePeriod.From,
				WindowEnd:   effectivePeriod.To,
			})
		}
	}

	// When the customer list requests realtime usage for the mixed batch.
	clock.FreezeTime(now)
	defer clock.UnFreeze()

	expands := meta.Expands{meta.ExpandRealtimeUsage}
	queryCountBeforeList := len(s.MockStreamingConnector.MeterQueries())
	listed, err := s.Charges.ListCustomerCharges(ctx, charges.ListCustomerChargesInput{
		ListChargesInput: charges.ListChargesInput{
			Namespace:      namespace,
			Page:           pagination.NewPage(1, len(fixtures)),
			CustomerIDs:    []string{cust.ID},
			ChargeTypes:    []meta.ChargeType{meta.ChargeTypeUsageBased},
			IncludeDeleted: true,
			Expands:        expands,
		},
	})
	require.NoError(t, err)
	require.Len(t, listed.Charges.Items, len(fixtures))

	// Then only eligible effective periods reach the meter and populate live fields.
	var queriedPeriods []timeutil.ClosedPeriod
	for _, query := range s.MockStreamingConnector.MeterQueries()[queryCountBeforeList:] {
		require.NotNil(t, query.From)
		require.NotNil(t, query.To)
		require.NotNil(t, query.FilterStoredAt)
		require.NotNil(t, query.FilterStoredAt.Lt)
		require.True(t, query.FilterStoredAt.Lt.Equal(now))
		queriedPeriods = append(queriedPeriods, timeutil.ClosedPeriod{From: query.From.UTC(), To: query.To.UTC()})
	}

	require.ElementsMatch(t, wantPeriods, queriedPeriods)

	for _, item := range listed.Charges.Items {
		charge, err := item.AsUsageBasedCharge()
		require.NoError(t, err)
		fixture, ok := fixturesByID[charge.ID]
		require.True(t, ok, charge.ID)
		delete(fixturesByID, charge.ID)
		require.Equal(t, fixture.status, charge.Status, fixture.name)

		if fixture.wantRealtimeUsage {
			require.NotNil(t, charge.Expands.RealtimeUsage, fixture.name)
			require.Equal(t, fixture.quantity, charge.Expands.RealtimeUsage.Total.InexactFloat64(), fixture.name)
			require.NotNil(t, charge.Expands.RealtimeQuantity, fixture.name)
			require.Equal(t, fixture.quantity, charge.Expands.RealtimeQuantity.InexactFloat64(), fixture.name)
			require.Len(t, item.UsageBasedRealizations, 1, fixture.name)
			require.NotNil(t, item.UsageBasedRealizations[0].Quantity, fixture.name)
			require.Equal(t, fixture.quantity, item.UsageBasedRealizations[0].Quantity.InexactFloat64(), fixture.name)
		} else {
			require.Nil(t, charge.Expands.RealtimeUsage, fixture.name)
			require.Nil(t, charge.Expands.RealtimeQuantity, fixture.name)
			for _, realization := range item.UsageBasedRealizations {
				if realization.Run == nil {
					require.Nil(t, realization.Quantity, fixture.name)
				}
			}
		}
	}

	require.Empty(t, fixturesByID)

	s.Run("direct single reads apply the same period boundary", func() {
		t := s.T()
		reader, ok := s.UsageBasedService.(interface {
			GetByID(context.Context, usagebased.GetByIDInput) (usagebased.Charge, error)
		})
		require.True(t, ok)

		before := len(s.MockStreamingConnector.MeterQueries())
		eligible, err := reader.GetByID(t.Context(), usagebased.GetByIDInput{ChargeID: chargeIDs["created at start"], Expands: expands})
		require.NoError(t, err)
		require.NotNil(t, eligible.Expands.RealtimeQuantity)
		require.Len(t, s.MockStreamingConnector.MeterQueries(), before+1)

		skipped, err := reader.GetByID(t.Context(), usagebased.GetByIDInput{ChargeID: chargeIDs["created at end"], Expands: expands})
		require.NoError(t, err)
		require.Nil(t, skipped.Expands.RealtimeUsage)
		require.Nil(t, skipped.Expands.RealtimeQuantity)
		require.Len(t, s.MockStreamingConnector.MeterQueries(), before+1)
	})
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
	s.TaxCodeEnv.ProvisionDefaultTaxCodes(s.T(), namespace)
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
		ChangeSource: billing.ChangeSourceSystem,
		Namespace:    namespace,
		Intents: charges.NewCreateChargeIntents(
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
		),
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

	s.Run("a stale charge feature does not fail expansion", func() {
		// given:
		// - a persisted charge whose snapshotted feature ID no longer resolves
		usageCharge, err := created[0].AsUsageBasedCharge()
		require.NoError(s.T(), err)
		usageCharge.Status = usagebased.StatusActive
		usageCharge.State.FeatureID = "missing-feature-id"
		staleCharge := charges.NewCharge(usageCharge)

		// when:
		// - feature expansion resolves that charge
		references, err := collectCustomerChargeReferences(charges.Charges{staleCharge})
		require.NoError(s.T(), err)
		entities, err := s.Charges.loadCustomerChargeEntities(ctx, namespace, references, meta.Expands{meta.ExpandFeature})
		require.NoError(s.T(), err)
		customerCharge, err := s.Charges.buildCustomerCharge(ctx, staleCharge, entities, meta.Expands{meta.ExpandFeature})

		// then:
		// - the missing feature is omitted so the facade can retain its ID-only fallback
		require.NoError(s.T(), err)
		require.Nil(s.T(), customerCharge.Feature)
		resolvedCharge, err := customerCharge.AsUsageBasedCharge()
		require.NoError(s.T(), err)
		require.Equal(s.T(), usageCharge.State.FeatureID, resolvedCharge.State.FeatureID)
	})

	s.Run("the subscription side-loader serves the facade's bulk lookup", func() {
		full, err := s.Charges.listCustomerChargeSubscriptions(ctx, namespace, []string{cust.ID}, []string{subscriptionID})
		require.NoError(s.T(), err)
		require.Contains(s.T(), full, subscriptionID)
		s.Equal("api-list-subscription", full[subscriptionID].Name)
	})

	s.Run("a namespace-wide listing resolves each row's own customer", func() {
		// given:
		// - a second customer with its own charge
		other := s.CreateTestCustomer(namespace, "api-list-other")
		otherCharges, err := s.Charges.Create(ctx, charges.CreateInput{
			ChangeSource: billing.ChangeSourceSystem,
			Namespace:    namespace,
			Intents: charges.NewCreateChargeIntents(
				s.createMockChargeIntent(createMockChargeIntentInput{
					customer:          other.GetID(),
					currency:          USD,
					servicePeriod:     servicePeriod,
					settlementMode:    productcatalog.CreditThenInvoiceSettlementMode,
					price:             productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: alpacadecimal.NewFromInt(1)}),
					featureKey:        feat.Feature.Key,
					name:              "api-list-other-usage-based",
					managedBy:         billing.SubscriptionManagedLine,
					uniqueReferenceID: "api-list-other-usage-based",
				}),
			),
		})
		require.NoError(s.T(), err)
		require.Len(s.T(), otherCharges, 1)

		unscoped := newListInput(meta.Expands{meta.ExpandCustomer})
		unscoped.CustomerIDs = nil

		bothCustomers := newListInput(meta.Expands{meta.ExpandCustomer})
		bothCustomers.CustomerIDs = []string{cust.ID, other.ID}

		filtered := newListInput(meta.Expands{meta.ExpandCustomer})
		filtered.CustomerIDs = nil
		filtered.CustomerID = &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(other.ID)}}

		cases := []struct {
			name      string
			input     charges.ListCustomerChargesInput
			customers []string
		}{
			{name: "no customer scoping", input: unscoped, customers: []string{cust.ID, other.ID}},
			{name: "several customer IDs", input: bothCustomers, customers: []string{cust.ID, other.ID}},
			{name: "customer filter", input: filtered, customers: []string{other.ID}},
		}

		for _, tc := range cases {
			// when:
			// - the facade lists charges across customers with the customer expand
			result, err := s.Charges.ListCustomerCharges(ctx, tc.input)
			require.NoError(s.T(), err, tc.name)

			// then:
			// - every row carries the customer it belongs to
			require.Len(s.T(), result.Charges.Items, len(tc.customers), tc.name)

			var rowCustomers []string
			for _, item := range result.Charges.Items {
				chargeCustomer, err := item.GetCustomerID()
				require.NoError(s.T(), err)
				require.NotNil(s.T(), item.Customer, tc.name)
				s.Equal(chargeCustomer.ID, item.Customer.ID, tc.name)
				rowCustomers = append(rowCustomers, item.Customer.ID)
			}

			s.ElementsMatch(tc.customers, rowCustomers, tc.name)
		}
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
			ChangeSource: billing.ChangeSourceSystem,
			Namespace:    namespace,
			Intents: charges.NewCreateChargeIntents(
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
			),
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
		// - both charges reference the api-requests feature by key without a pinned ID
		// when:
		// - listing filtered by feature id and by feature key
		// then:
		// - the ID does not match before activation, while the key returns every charge
		byID := newListInput(meta.ExpandNone)
		byID.FeatureID = &filter.FilterULID{FilterString: filter.FilterString{Eq: lo.ToPtr(feat.Feature.ID)}}

		result, err := s.Charges.ListCustomerCharges(ctx, byID)
		require.NoError(s.T(), err)
		s.Empty(result.Charges.Items)

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
