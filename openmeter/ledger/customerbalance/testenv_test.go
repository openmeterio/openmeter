package customerbalance

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/alpacahq/alpacadecimal"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	charges "github.com/openmeterio/openmeter/openmeter/billing/charges"
	chargeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee"
	flatfeeadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/adapter"
	flatfeeservice "github.com/openmeterio/openmeter/openmeter/billing/charges/flatfee/service"
	chargemeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	chargestestutils "github.com/openmeterio/openmeter/openmeter/billing/charges/testutils"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	testFeatureKey = "api_requests"
	testMeterKey   = "api_requests"
)

type testEnv struct {
	*ledgertestutils.IntegrationEnv

	Service           *Service
	flatFeeService    flatfee.Service
	usageBasedService usagebased.Service
	streaming         *streamingtestutils.MockStreamingConnector
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	base := ledgertestutils.NewIntegrationEnv(t, "ledger-balance")
	logger := slog.New(slog.DiscardHandler)
	streaming := streamingtestutils.NewMockStreamingConnector(t)
	handlers := chargestestutils.NewMockHandlers()

	billingService := mockCustomerOverrideService{
		customer: customer.Customer{
			ManagedResource: models.ManagedResource{
				NamespacedModel: models.NamespacedModel{
					Namespace: base.CustomerID.Namespace,
				},
				ID:   base.CustomerID.ID,
				Name: "Test Customer",
			},
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: []string{"subject-1"},
			},
		},
	}

	featureService := mockFeatureConnector{
		meters: feature.FeatureMeterCollection{
			testFeatureKey: {
				Feature: feature.Feature{
					Namespace: base.Namespace,
					ID:        "feature-1",
					Name:      "API Requests",
					Key:       testFeatureKey,
					MeterID:   lo.ToPtr("meter-1"),
					CreatedAt: base.Now(),
					UpdatedAt: base.Now(),
				},
				Meter: &meter.Meter{
					ManagedResource: models.ManagedResource{
						NamespacedModel: models.NamespacedModel{
							Namespace: base.Namespace,
						},
						ID:   "meter-1",
						Name: "API Requests Meter",
					},
					Key:         testMeterKey,
					Aggregation: meter.MeterAggregationSum,
					EventType:   "api_request",
				},
			},
		},
	}

	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: base.DB,
		Logger: logger,
	})
	require.NoError(t, err)

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	usageAdapter, err := usagebasedadapter.New(usagebasedadapter.Config{
		Client:      base.DB,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	require.NoError(t, err)

	flatFeeAdapter, err := flatfeeadapter.New(flatfeeadapter.Config{
		Client:      base.DB,
		Logger:      logger,
		MetaAdapter: metaAdapter,
	})
	require.NoError(t, err)

	flatFeeService, err := flatfeeservice.New(flatfeeservice.Config{
		Adapter:     flatFeeAdapter,
		Handler:     handlers.FlatFee,
		MetaAdapter: metaAdapter,
		Locker:      locker,
	})
	require.NoError(t, err)

	usageService, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter:                 usageAdapter,
		Handler:                 handlers.UsageBased,
		Locker:                  locker,
		MetaAdapter:             metaAdapter,
		CustomerOverrideService: billingService,
		FeatureService:          featureService,
		RatingService:           billingratingservice.New(),
		StreamingConnector:      streaming,
	})
	require.NoError(t, err)

	searchAdapter, err := chargeadapter.New(chargeadapter.Config{
		Client: base.DB,
		Logger: logger,
	})
	require.NoError(t, err)

	service, err := New(Config{
		AccountResolver: base.Deps.ResolversService,
		ChargesService: chargeStore{
			search:            searchAdapter,
			flatFeeService:    flatFeeService,
			usageBasedService: usageService,
		},
		UsageBasedService: usageService,
	})
	require.NoError(t, err)

	env := &testEnv{
		IntegrationEnv:    base,
		Service:           service,
		flatFeeService:    flatFeeService,
		usageBasedService: usageService,
		streaming:         streaming,
	}

	env.createCustomer(t)

	return env
}

func (e *testEnv) addUsage(value float64, at time.Time) {
	e.streaming.AddSimpleEvent(testMeterKey, value, at)
}

func (e *testEnv) sp() timeutil.ClosedPeriod {
	return timeutil.ClosedPeriod{
		From: clock.Now().Add(-time.Hour),
		To:   clock.Now().Add(time.Hour),
	}
}

// simply currency based backing (balance doesn't care about most dimensions)
func (e *testEnv) bookFBOBalance(t *testing.T, amount alpacadecimal.Decimal) {
	t.Helper()

	inputs, err := transactions.ResolveTransactions(
		t.Context(),
		transactions.ResolverDependencies{
			AccountService:    e.Deps.ResolversService,
			SubAccountService: e.Deps.AccountService,
		},
		transactions.ResolutionScope{
			CustomerID: e.CustomerID,
			Namespace:  e.Namespace,
		},
		transactions.IssueCustomerReceivableTemplate{
			At:       e.Now(),
			Amount:   amount,
			Currency: e.Currency,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *testEnv) createUsageBasedCharge(t *testing.T, unitPrice alpacadecimal.Decimal, settlementMode productcatalog.SettlementMode, servicePeriod timeutil.ClosedPeriod) usagebased.Charge {
	t.Helper()

	createdCharges, err := e.usageBasedService.Create(t.Context(), usagebased.CreateInput{
		Namespace: e.Namespace,
		Intents: []usagebased.Intent{
			{
				Intent: chargemeta.Intent{
					Name:              "API Requests",
					ManagedBy:         billing.SystemManagedLine,
					CustomerID:        e.CustomerID.ID,
					Currency:          e.Currency,
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt:      e.Now().Add(-time.Minute),
				SettlementMode: settlementMode,
				FeatureKey:     testFeatureKey,
				Price:          *productcatalog.NewPriceFrom(productcatalog.UnitPrice{Amount: unitPrice}),
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, createdCharges, 1)

	return createdCharges[0].Charge
}

func (e *testEnv) createFlatFeeCharge(t *testing.T, amount alpacadecimal.Decimal, settlementMode productcatalog.SettlementMode, servicePeriod timeutil.ClosedPeriod) flatfee.Charge {
	t.Helper()

	createdCharges, err := e.flatFeeService.Create(t.Context(), flatfee.CreateInput{
		Namespace: e.Namespace,
		Intents: []flatfee.Intent{
			{
				Intent: chargemeta.Intent{
					Name:              "Platform Fee",
					ManagedBy:         billing.SystemManagedLine,
					CustomerID:        e.CustomerID.ID,
					Currency:          e.Currency,
					ServicePeriod:     servicePeriod,
					FullServicePeriod: servicePeriod,
					BillingPeriod:     servicePeriod,
				},
				InvoiceAt:             e.Now().Add(-time.Minute),
				SettlementMode:        settlementMode,
				PaymentTerm:           productcatalog.InAdvancePaymentTerm,
				AmountBeforeProration: amount,
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, createdCharges, 1)

	return createdCharges[0].Charge
}

func (e *testEnv) advanceFlatFeeCharge(t *testing.T, charge flatfee.Charge) flatfee.Charge {
	t.Helper()

	advancedCharge, err := e.flatFeeService.AdvanceCharge(t.Context(), flatfee.AdvanceChargeInput{
		ChargeID: charge.GetChargeID(),
	})
	require.NoError(t, err)
	require.NotNil(t, advancedCharge)

	return *advancedCharge
}

func (e *testEnv) createCustomer(t *testing.T) {
	t.Helper()

	_, err := e.DB.Customer.Create().
		SetNamespace(e.Namespace).
		SetID(e.CustomerID.ID).
		SetName("Test Customer").
		Save(t.Context())
	require.NoError(t, err)
}

type chargeStore struct {
	search            charges.ChargesSearchAdapter
	flatFeeService    flatfee.Service
	usageBasedService usagebased.Service
}

func (l chargeStore) ListCharges(ctx context.Context, input charges.ListChargesInput) (pagination.Result[charges.Charge], error) {
	searchResult, err := l.search.ListCharges(ctx, input)
	if err != nil {
		return pagination.Result[charges.Charge]{}, err
	}

	flatFeeIDs := make([]string, 0, len(searchResult.Items))
	usageBasedIDs := make([]string, 0, len(searchResult.Items))
	for _, item := range searchResult.Items {
		switch item.Type {
		case chargemeta.ChargeTypeFlatFee:
			flatFeeIDs = append(flatFeeIDs, item.ID)
		case chargemeta.ChargeTypeUsageBased:
			usageBasedIDs = append(usageBasedIDs, item.ID)
		}
	}

	flatFeeCharges, err := l.flatFeeService.GetByIDs(ctx, flatfee.GetByIDsInput{
		Namespace: input.Namespace,
		IDs:       flatFeeIDs,
		Expands:   input.Expands,
	})
	if err != nil {
		return pagination.Result[charges.Charge]{}, err
	}

	usageBasedCharges, err := l.usageBasedService.GetByIDs(ctx, usagebased.GetByIDsInput{
		Namespace: input.Namespace,
		IDs:       usageBasedIDs,
		Expands:   input.Expands,
	})
	if err != nil {
		return pagination.Result[charges.Charge]{}, err
	}

	chargesByID := make(map[string]charges.Charge, len(flatFeeCharges)+len(usageBasedCharges))
	for _, charge := range flatFeeCharges {
		chargesByID[charge.ID] = charges.NewCharge(charge)
	}
	for _, charge := range usageBasedCharges {
		chargesByID[charge.ID] = charges.NewCharge(charge)
	}

	items := make([]charges.Charge, 0, len(chargesByID))
	for _, item := range searchResult.Items {
		charge, ok := chargesByID[item.ID]
		if !ok {
			continue
		}

		items = append(items, charge)
	}

	return pagination.Result[charges.Charge]{
		Page:       searchResult.Page,
		TotalCount: searchResult.TotalCount,
		Items:      items,
	}, nil
}

type mockCustomerOverrideService struct {
	customer customer.Customer
}

func (s mockCustomerOverrideService) UpsertCustomerOverride(context.Context, billing.UpsertCustomerOverrideInput) (billing.CustomerOverrideWithDetails, error) {
	return billing.CustomerOverrideWithDetails{}, nil
}

func (s mockCustomerOverrideService) DeleteCustomerOverride(context.Context, billing.DeleteCustomerOverrideInput) error {
	return nil
}

func (s mockCustomerOverrideService) GetCustomerOverride(context.Context, billing.GetCustomerOverrideInput) (billing.CustomerOverrideWithDetails, error) {
	return billing.CustomerOverrideWithDetails{
		Customer: &s.customer,
	}, nil
}

func (s mockCustomerOverrideService) GetCustomerApp(context.Context, billing.GetCustomerAppInput) (app.App, error) {
	return nil, nil
}

func (s mockCustomerOverrideService) ListCustomerOverrides(context.Context, billing.ListCustomerOverridesInput) (billing.ListCustomerOverridesResult, error) {
	return billing.ListCustomerOverridesResult{}, nil
}

type mockFeatureConnector struct {
	meters feature.FeatureMeterCollection
}

func (c mockFeatureConnector) CreateFeature(context.Context, feature.CreateFeatureInputs) (feature.Feature, error) {
	return feature.Feature{}, nil
}

func (c mockFeatureConnector) UpdateFeature(context.Context, feature.UpdateFeatureInputs) (feature.Feature, error) {
	return feature.Feature{}, nil
}

func (c mockFeatureConnector) ArchiveFeature(context.Context, models.NamespacedID) error {
	return nil
}

func (c mockFeatureConnector) ListFeatures(context.Context, feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	return pagination.Result[feature.Feature]{}, nil
}

func (c mockFeatureConnector) GetFeature(context.Context, string, string, feature.IncludeArchivedFeature) (*feature.Feature, error) {
	return nil, nil
}

func (c mockFeatureConnector) ResolveFeatureMeters(context.Context, string, []string) (feature.FeatureMeters, error) {
	return c.meters, nil
}
