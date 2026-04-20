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
	lineageadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/adapter"
	lineageservice "github.com/openmeterio/openmeter/openmeter/billing/charges/lineage/service"
	chargemeta "github.com/openmeterio/openmeter/openmeter/billing/charges/meta"
	metaadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/meta/adapter"
	"github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased"
	usagebasedadapter "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/adapter"
	usagebasedservice "github.com/openmeterio/openmeter/openmeter/billing/charges/usagebased/service"
	billingratingservice "github.com/openmeterio/openmeter/openmeter/billing/rating/service"
	"github.com/openmeterio/openmeter/openmeter/customer"
	ledgerchargeadapter "github.com/openmeterio/openmeter/openmeter/ledger/chargeadapter"
	ledgercollector "github.com/openmeterio/openmeter/openmeter/ledger/collector"
	ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
	"github.com/openmeterio/openmeter/openmeter/ledger/transactions"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/adapter"
	meterservice "github.com/openmeterio/openmeter/openmeter/meter/service"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/productcatalog"
	pcadapter "github.com/openmeterio/openmeter/openmeter/productcatalog/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/currencyx"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const (
	testFeatureKey = "api_requests"
	testMeterKey   = "api_requests"
)

type testEnv struct {
	*ledgertestutils.IntegrationEnv

	Service           *service
	flatFeeService    flatfee.Service
	usageBasedService usagebased.Service
	featureMeters     feature.FeatureMeters
	streaming         *streamingtestutils.MockStreamingConnector
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	base := ledgertestutils.NewIntegrationEnv(t, "ledger-balance")
	logger := slog.New(slog.DiscardHandler)
	streaming := streamingtestutils.NewMockStreamingConnector(t)

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

	publisher := eventbus.NewMock(t)

	namespaceManager, err := namespace.NewManager(namespace.ManagerConfig{
		DefaultNamespace: base.Namespace,
	})
	require.NoError(t, err)

	meterRepo, err := meteradapter.New(meteradapter.Config{
		Client: base.DB,
		Logger: logger,
	})
	require.NoError(t, err)

	meterQueryService := meterservice.New(meterRepo)
	meterManageService := meterservice.NewManage(
		meterRepo,
		publisher,
		namespaceManager,
		nil,
	)

	featureRepo := pcadapter.NewPostgresFeatureRepo(base.DB, logger)
	featureService := feature.NewFeatureConnector(
		featureRepo,
		meterQueryService,
		publisher,
	)

	meterEntity, err := meterManageService.CreateMeter(t.Context(), meter.CreateMeterInput{
		Namespace:     base.Namespace,
		Name:          "API Requests Meter",
		Key:           testMeterKey,
		EventType:     "api_request",
		Aggregation:   meter.MeterAggregationSum,
		ValueProperty: lo.ToPtr("$.data.value"),
	})
	require.NoError(t, err)

	featureEntity, err := featureService.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Namespace: base.Namespace,
		Name:      "API Requests",
		Key:       testFeatureKey,
		MeterID:   lo.ToPtr(meterEntity.ID),
	})
	require.NoError(t, err)

	featureMeters, err := featureService.ResolveFeatureMeters(
		t.Context(),
		base.Namespace,
		ref.IDOrKey{Key: testFeatureKey},
		ref.IDOrKey{ID: featureEntity.ID},
	)
	require.NoError(t, err)

	metaAdapter, err := metaadapter.New(metaadapter.Config{
		Client: base.DB,
		Logger: logger,
	})
	require.NoError(t, err)

	locker, err := lockr.NewLocker(&lockr.LockerConfig{
		Logger: logger,
	})
	require.NoError(t, err)

	lineageAdapter, err := lineageadapter.New(lineageadapter.Config{
		Client: base.DB,
	})
	require.NoError(t, err)

	lineageService, err := lineageservice.New(lineageservice.Config{
		Adapter: lineageAdapter,
	})
	require.NoError(t, err)

	collectorService := ledgercollector.NewService(ledgercollector.Config{
		Ledger: base.Deps.HistoricalLedger,
		Dependencies: transactions.ResolverDependencies{
			AccountService:    base.Deps.ResolversService,
			SubAccountService: base.Deps.AccountService,
		},
	})

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
		Adapter: flatFeeAdapter,
		Handler: ledgerchargeadapter.NewFlatFeeHandler(
			base.Deps.HistoricalLedger,
			transactions.ResolverDependencies{
				AccountService:    base.Deps.ResolversService,
				SubAccountService: base.Deps.AccountService,
			},
			collectorService,
		),
		Lineage:     lineageService,
		MetaAdapter: metaAdapter,
		Locker:      locker,
	})
	require.NoError(t, err)

	usageService, err := usagebasedservice.New(usagebasedservice.Config{
		Adapter: usageAdapter,
		Handler: ledgerchargeadapter.NewUsageBasedHandler(
			base.Deps.HistoricalLedger,
			transactions.ResolverDependencies{
				AccountService:    base.Deps.ResolversService,
				SubAccountService: base.Deps.AccountService,
			},
			collectorService,
		),
		Lineage:                 lineageService,
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
		AccountResolver:   base.Deps.ResolversService,
		SubAccountService: base.Deps.AccountService,
		ChargesService: chargeStore{
			search:            searchAdapter,
			flatFeeService:    flatFeeService,
			usageBasedService: usageService,
		},
		UsageBasedService: usageService,
		Ledger:            base.Deps.HistoricalLedger,
	})
	require.NoError(t, err)

	env := &testEnv{
		IntegrationEnv:    base,
		Service:           service,
		flatFeeService:    flatFeeService,
		usageBasedService: usageService,
		featureMeters:     featureMeters,
		streaming:         streaming,
	}

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

func (e *testEnv) passTimeAfterServicePeriod(t *testing.T, servicePeriod timeutil.ClosedPeriod) {
	t.Helper()

	clock.SetTime(servicePeriod.To.Add(time.Second))
	t.Cleanup(clock.ResetTime)
}

// simply currency based backing (balance doesn't care about most dimensions)
func (e *testEnv) bookFBOBalance(t *testing.T, amount alpacadecimal.Decimal) {
	e.bookFBOBalanceInCurrency(t, amount, e.Currency)
}

func (e *testEnv) bookFBOBalanceInCurrency(t *testing.T, amount alpacadecimal.Decimal, currency currencyx.Code) {
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
			Currency: currency,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *testEnv) fundOpenReceivable(t *testing.T, amount alpacadecimal.Decimal) {
	e.fundOpenReceivableInCurrency(t, amount, e.Currency)
}

func (e *testEnv) fundOpenReceivableInCurrency(t *testing.T, amount alpacadecimal.Decimal, currency currencyx.Code) {
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
		transactions.FundCustomerReceivableTemplate{
			At:       e.Now(),
			Amount:   amount,
			Currency: currency,
		},
		transactions.SettleCustomerReceivablePaymentTemplate{
			At:       e.Now(),
			Amount:   amount,
			Currency: currency,
		},
	)
	require.NoError(t, err)

	_, err = e.Deps.HistoricalLedger.CommitGroup(t.Context(), transactions.GroupInputs(e.Namespace, nil, inputs...))
	require.NoError(t, err)
}

func (e *testEnv) createUsageBasedCharge(t *testing.T, unitPrice alpacadecimal.Decimal, settlementMode productcatalog.SettlementMode, servicePeriod timeutil.ClosedPeriod) usagebased.Charge {
	return e.createUsageBasedChargeInCurrency(t, unitPrice, settlementMode, servicePeriod, e.Currency)
}

func (e *testEnv) createUsageBasedChargeInCurrency(t *testing.T, unitPrice alpacadecimal.Decimal, settlementMode productcatalog.SettlementMode, servicePeriod timeutil.ClosedPeriod, currency currencyx.Code) usagebased.Charge {
	t.Helper()

	createdCharges, err := e.usageBasedService.Create(t.Context(), usagebased.CreateInput{
		Namespace:     e.Namespace,
		FeatureMeters: e.featureMeters,
		Intents: []usagebased.Intent{
			{
				Intent: chargemeta.Intent{
					Name:              "API Requests",
					ManagedBy:         billing.SystemManagedLine,
					CustomerID:        e.CustomerID.ID,
					Currency:          currency,
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
	return e.createFlatFeeChargeInCurrency(t, amount, settlementMode, servicePeriod, e.Currency)
}

func (e *testEnv) createFlatFeeChargeInCurrency(t *testing.T, amount alpacadecimal.Decimal, settlementMode productcatalog.SettlementMode, servicePeriod timeutil.ClosedPeriod, currency currencyx.Code) flatfee.Charge {
	t.Helper()

	createdCharges, err := e.flatFeeService.Create(t.Context(), flatfee.CreateInput{
		Namespace: e.Namespace,
		Intents: []flatfee.Intent{
			{
				Intent: chargemeta.Intent{
					Name:              "Platform Fee",
					ManagedBy:         billing.SystemManagedLine,
					CustomerID:        e.CustomerID.ID,
					Currency:          currency,
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

	chargeCustomerID := customer.CustomerID{
		Namespace: charge.Namespace,
		ID:        charge.Intent.CustomerID,
	}
	require.Equal(t, e.CustomerID, chargeCustomerID, "charge scope differs from test env scope")

	_, err := e.Deps.ResolversService.GetCustomerAccounts(t.Context(), chargeCustomerID)
	require.NoError(t, err, "customer accounts should exist for charge scope before advance")

	latestCharges, err := e.flatFeeService.GetByIDs(t.Context(), flatfee.GetByIDsInput{
		Namespace: charge.Namespace,
		IDs:       []string{charge.ID},
		Expands:   chargemeta.Expands{chargemeta.ExpandRealizations},
	})
	require.NoError(t, err)
	require.Len(t, latestCharges, 1)
	require.Equal(t, e.CustomerID.ID, latestCharges[0].Intent.CustomerID, "persisted charge customer differs from test env customer")
	require.Equal(t, e.Namespace, latestCharges[0].Namespace, "persisted charge namespace differs from test env namespace")

	advancedCharge, err := e.flatFeeService.AdvanceCharge(t.Context(), flatfee.AdvanceChargeInput{
		ChargeID: charge.GetChargeID(),
	})
	require.NoError(t, err)
	require.NotNil(t, advancedCharge)

	return *advancedCharge
}

type chargeStore struct {
	search            charges.ChargesSearchAdapter
	flatFeeService    flatfee.Service
	usageBasedService usagebased.Service
}

func (l chargeStore) GetByIDs(ctx context.Context, input charges.GetByIDsInput) (charges.Charges, error) {
	searchResult, err := l.search.GetByIDs(ctx, input)
	if err != nil {
		return nil, err
	}

	flatFeeIDs := make([]string, 0, len(searchResult))
	usageBasedIDs := make([]string, 0, len(searchResult))
	for _, item := range searchResult {
		switch item.Type {
		case chargemeta.ChargeTypeFlatFee:
			flatFeeIDs = append(flatFeeIDs, item.ID.ID)
		case chargemeta.ChargeTypeUsageBased:
			usageBasedIDs = append(usageBasedIDs, item.ID.ID)
		}
	}

	flatFeeCharges, err := l.flatFeeService.GetByIDs(ctx, flatfee.GetByIDsInput{
		Namespace: input.Namespace,
		IDs:       flatFeeIDs,
		Expands:   input.Expands,
	})
	if err != nil {
		return nil, err
	}

	usageBasedCharges, err := l.usageBasedService.GetByIDs(ctx, usagebased.GetByIDsInput{
		Namespace: input.Namespace,
		IDs:       usageBasedIDs,
		Expands:   input.Expands,
	})
	if err != nil {
		return nil, err
	}

	chargesByID := make(map[string]charges.Charge, len(flatFeeCharges)+len(usageBasedCharges))
	for _, charge := range flatFeeCharges {
		chargesByID[charge.ID] = charges.NewCharge(charge)
	}
	for _, charge := range usageBasedCharges {
		chargesByID[charge.ID] = charges.NewCharge(charge)
	}

	items := make(charges.Charges, 0, len(searchResult))
	for _, item := range searchResult {
		charge, ok := chargesByID[item.ID.ID]
		if !ok {
			continue
		}

		items = append(items, charge)
	}

	return items, nil
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
			flatFeeIDs = append(flatFeeIDs, item.ID.ID)
		case chargemeta.ChargeTypeUsageBased:
			usageBasedIDs = append(usageBasedIDs, item.ID.ID)
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
		charge, ok := chargesByID[item.ID.ID]
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
