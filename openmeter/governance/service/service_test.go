package service

import (
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/governance"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/registry"
	registrybuilder "github.com/openmeterio/openmeter/openmeter/registry/builder"
	streamingtestutils "github.com/openmeterio/openmeter/openmeter/streaming/testutils"
	"github.com/openmeterio/openmeter/openmeter/subject"
	subjectadapter "github.com/openmeterio/openmeter/openmeter/subject/adapter"
	subjectservice "github.com/openmeterio/openmeter/openmeter/subject/service"
	"github.com/openmeterio/openmeter/openmeter/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/pkg/clock"
	"github.com/openmeterio/openmeter/pkg/datetime"
	"github.com/openmeterio/openmeter/pkg/framework/lockr"
	pagination "github.com/openmeterio/openmeter/pkg/pagination/v2"
	"github.com/openmeterio/openmeter/pkg/timeutil"
)

const testPageSize = 100

func newTestNamespace(t *testing.T) string {
	t.Helper()

	return ulid.Make().String()
}

type testDeps struct {
	dbClient           *testutils.TestDB
	subjectService     subject.Service
	customerService    customer.Service
	meterService       meter.ManageService
	featureRepo        feature.FeatureRepo
	registry           *registry.Entitlement
	streamingConnector *streamingtestutils.MockStreamingConnector
}

func (d *testDeps) close(t *testing.T) {
	t.Helper()

	if err := d.dbClient.EntDriver.Close(); err != nil {
		t.Errorf("close ent driver: %v", err)
	}

	if err := d.dbClient.PGDriver.Close(); err != nil {
		t.Errorf("close pg driver: %v", err)
	}
}

func setupTestDeps(t *testing.T) *testDeps {
	t.Helper()

	logger := testutils.NewDiscardLogger(t)
	testdb := testutils.InitPostgresDB(t, testutils.PostgresDBStateEntMigrated)
	dbClient := testdb.EntDriver.Client()

	meterService, err := meteradapter.NewManage(nil)
	require.NoError(t, err)

	subjectAdapter, err := subjectadapter.New(dbClient)
	require.NoError(t, err)

	subjectSvc, err := subjectservice.New(subjectAdapter)
	require.NoError(t, err)

	customerAdapter, err := customeradapter.New(customeradapter.Config{
		Client: dbClient,
		Logger: logger,
	})
	require.NoError(t, err)

	customerSvc, err := customerservice.New(customerservice.Config{
		Adapter:   customerAdapter,
		Publisher: eventbus.NewMock(t),
		Logger:    logger,
	})
	require.NoError(t, err)

	locker, err := lockr.NewLocker(&lockr.LockerConfig{Logger: logger})
	require.NoError(t, err)

	streamingConnector := streamingtestutils.NewMockStreamingConnector(t)

	reg := registrybuilder.GetEntitlementRegistry(registrybuilder.EntitlementOptions{
		DatabaseClient:     dbClient,
		StreamingConnector: streamingConnector,
		Logger:             logger,
		Tracer:             noop.NewTracerProvider().Tracer("test"),
		MeterService:       meterService,
		CustomerService:    customerSvc,
		Publisher:          eventbus.NewMock(t),
		EntitlementsConfiguration: config.EntitlementsConfiguration{
			GracePeriod: datetime.ISODurationString("P1D"),
		},
		Locker: locker,
	})

	return &testDeps{
		dbClient:           testdb,
		subjectService:     subjectSvc,
		customerService:    customerSvc,
		meterService:       meterService,
		featureRepo:        reg.FeatureRepo,
		registry:           reg,
		streamingConnector: streamingConnector,
	}
}

func newTestService(t *testing.T, deps *testDeps) governance.Service {
	t.Helper()

	svc, err := New(Config{
		Customer:    deps.customerService,
		Entitlement: deps.registry.Entitlement,
		Feature:     deps.registry.Feature,
		Tracer:      noop.NewTracerProvider().Tracer("test"),
		Meter:       metricnoop.NewMeterProvider().Meter("test"),
	})
	require.NoError(t, err)

	return svc
}

func createCustomer(t *testing.T, deps *testDeps, ns, key string, subjectKeys []string) *customer.Customer {
	t.Helper()

	for _, sk := range subjectKeys {
		_, err := deps.subjectService.Create(t.Context(), subject.CreateInput{
			Namespace: ns,
			Key:       sk,
		})
		require.NoError(t, err)
	}

	cust, err := deps.customerService.CreateCustomer(t.Context(), customer.CreateCustomerInput{
		Namespace: ns,
		CustomerMutate: customer.CustomerMutate{
			Key:  lo.ToPtr(key),
			Name: key,
			UsageAttribution: &customer.CustomerUsageAttribution{
				SubjectKeys: subjectKeys,
			},
		},
	})
	require.NoError(t, err)

	return cust
}

func createBooleanFeatureAndEntitlement(t *testing.T, deps *testDeps, ns, featureKey string, cust *customer.Customer) {
	t.Helper()

	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Key:       featureKey,
		Name:      featureKey,
		Namespace: ns,
	})
	require.NoError(t, err)

	_, err = deps.registry.Entitlement.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        ns,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       lo.ToPtr(featureKey),
		FeatureID:        lo.ToPtr(feat.ID),
		EntitlementType:  entitlement.EntitlementTypeBoolean,
	}, nil)
	require.NoError(t, err)
}

func createOrphanFeature(t *testing.T, deps *testDeps, ns, featureKey string) {
	t.Helper()

	_, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Key:       featureKey,
		Name:      featureKey,
		Namespace: ns,
	})
	require.NoError(t, err)
}

// createMeterInPG writes a meter row to ent DB (FK constraint on features.meter_id).
// The mock meter adapter only stores in memory; this must be called after CreateMeter.
func createMeterInPG(t *testing.T, dbClient *entdb.Client, mtr meter.Meter) {
	t.Helper()

	_, err := dbClient.Meter.Create().
		SetID(mtr.ID).
		SetNamespace(mtr.Namespace).
		SetName(mtr.Name).
		SetKey(mtr.Key).
		SetAggregation(mtr.Aggregation).
		SetEventType(mtr.EventType).
		SetNillableValueProperty(mtr.ValueProperty).
		Save(t.Context())
	require.NoError(t, err)
}

func createMeter(t *testing.T, deps *testDeps, ns, key string) meter.Meter {
	t.Helper()

	mtr, err := deps.meterService.CreateMeter(t.Context(), meter.CreateMeterInput{
		Namespace:     ns,
		Name:          key,
		Key:           key,
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "test",
		ValueProperty: lo.ToPtr("$.value"),
	})
	require.NoError(t, err)

	createMeterInPG(t, deps.dbClient.EntDriver.Client(), mtr)

	return mtr
}

func createMeteredFeatureAndEntitlement(t *testing.T, deps *testDeps, ns, featureKey string, mtr meter.Meter, cust *customer.Customer, issueAfterReset *float64) {
	t.Helper()

	feat, err := deps.featureRepo.CreateFeature(t.Context(), feature.CreateFeatureInputs{
		Key:       featureKey,
		Name:      featureKey,
		Namespace: ns,
		MeterID:   lo.ToPtr(mtr.ID),
	})
	require.NoError(t, err)

	_, err = deps.registry.Entitlement.CreateEntitlement(t.Context(), entitlement.CreateEntitlementInputs{
		Namespace:        ns,
		UsageAttribution: cust.GetUsageAttribution(),
		FeatureKey:       lo.ToPtr(featureKey),
		FeatureID:        lo.ToPtr(feat.ID),
		EntitlementType:  entitlement.EntitlementTypeMetered,
		UsagePeriod: lo.ToPtr(entitlement.NewUsagePeriodInputFromRecurrence(timeutil.Recurrence{
			Interval: timeutil.RecurrencePeriodDaily,
			Anchor:   clock.Now(),
		})),
		IssueAfterReset: issueAfterReset,
	}, nil)
	require.NoError(t, err)
}

// --- Tests ---

func TestQueryAccess_UnknownCustomerKey(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{"ghost"},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	assert.Empty(t, res.Customers)
	require.Len(t, res.Errors, 1)
	assert.Equal(t, governance.QueryErrorCustomerNotFound, res.Errors[0].Code)
	assert.Equal(t, "ghost", res.Errors[0].CustomerKey)
}

func TestQueryAccess_KnownCustomerNoEntitlements(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{cust.GetUsageAttribution().SubjectKeys[0]},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	require.Len(t, res.Customers, 1)
	assert.Empty(t, res.Customers[0].Features)
	assert.Empty(t, res.Errors)
}

func TestQueryAccess_BooleanEntitlement_HasAccess(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	createBooleanFeatureAndEntitlement(t, deps, ns, "premium", cust)

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{"acme"},
		FeatureKeys:  []string{"premium"},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	require.Len(t, res.Customers, 1)
	assert.Empty(t, res.Errors)

	fa := res.Customers[0].Features["premium"]

	assert.True(t, fa.HasAccess)
	assert.Nil(t, fa.Reason)
}

func TestQueryAccess_FeatureNotFound(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	createCustomer(t, deps, ns, "acme", []string{"acme"})

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{"acme"},
		FeatureKeys:  []string{"does-not-exist"},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	require.Len(t, res.Customers, 1)

	fa := res.Customers[0].Features["does-not-exist"]

	assert.False(t, fa.HasAccess)
	require.NotNil(t, fa.Reason)
	assert.Equal(t, governance.ReasonCodeFeatureNotFound, fa.Reason.Code)
}

func TestQueryAccess_FeatureUnavailable(t *testing.T) {
	// Feature exists in org but customer has no entitlement for it.
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	createCustomer(t, deps, ns, "acme", []string{"acme"})
	createOrphanFeature(t, deps, ns, "enterprise")

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{"acme"},
		FeatureKeys:  []string{"enterprise"},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	require.Len(t, res.Customers, 1)

	fa := res.Customers[0].Features["enterprise"]

	assert.False(t, fa.HasAccess)
	require.NotNil(t, fa.Reason)
	assert.Equal(t, governance.ReasonCodeFeatureUnavailable, fa.Reason.Code)
}

func TestQueryAccess_MultipleKeysSameCustomer(t *testing.T) {
	// Two input keys resolve to the same customer; result has one entry with both keys in Matched.
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	// customer key = "acme", usage attribution subject key = "acme-sub"
	createCustomer(t, deps, ns, "acme", []string{"acme-sub"})

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{"acme", "acme-sub"},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	assert.Empty(t, res.Errors)
	require.Len(t, res.Customers, 1, "two keys resolving to same customer should collapse into one result")
	assert.Len(t, res.Customers[0].Matched, 2)
	assert.ElementsMatch(t, []string{"acme", "acme-sub"}, res.Customers[0].Matched)
}

func TestQueryAccess_MixedHitsAndMisses(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	createBooleanFeatureAndEntitlement(t, deps, ns, "feature-a", cust)

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{"acme", "unknown-key"},
		FeatureKeys:  []string{"feature-a"},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	require.Len(t, res.Customers, 1)
	require.Len(t, res.Errors, 1)
	assert.Equal(t, governance.QueryErrorCustomerNotFound, res.Errors[0].Code)
	assert.True(t, res.Customers[0].Features["feature-a"].HasAccess)
}

func TestQueryAccess_NoFeatureKeysReturnsAll(t *testing.T) {
	// When no feature keys are given, all org features are returned — including ones
	// the customer has no entitlement for (marked feature-unavailable).
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	createBooleanFeatureAndEntitlement(t, deps, ns, "feat-1", cust)
	createBooleanFeatureAndEntitlement(t, deps, ns, "feat-2", cust)
	// feat-3 exists in the org but the customer has no entitlement for it.
	createOrphanFeature(t, deps, ns, "feat-3")

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{"acme"},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	require.Len(t, res.Customers, 1)
	assert.Len(t, res.Customers[0].Features, 3)
	assert.True(t, res.Customers[0].Features["feat-1"].HasAccess)
	assert.True(t, res.Customers[0].Features["feat-2"].HasAccess)

	feat3 := res.Customers[0].Features["feat-3"]

	assert.False(t, feat3.HasAccess)
	require.NotNil(t, feat3.Reason)
	assert.Equal(t, governance.ReasonCodeFeatureUnavailable, feat3.Reason.Code)
}

func TestQueryAccess_MeteredEntitlement_HasAccess(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	mtr := createMeter(t, deps, ns, "api-calls")
	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	// IssueAfterReset=10.0 → balance starts at 10, HasAccess=true
	createMeteredFeatureAndEntitlement(t, deps, ns, "premium", mtr, cust, lo.ToPtr(10.0))

	// Add an event so the streaming mock has data for the meter.
	deps.streamingConnector.AddSimpleEvent(mtr.Key, 1, now)

	clock.SetTime(now.Add(time.Hour))

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{"acme"},
		FeatureKeys:  []string{"premium"},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	require.Len(t, res.Customers, 1)
	assert.Empty(t, res.Errors)

	fa := res.Customers[0].Features["premium"]

	assert.True(t, fa.HasAccess)
	assert.Nil(t, fa.Reason)
}

func TestQueryAccess_MeteredEntitlement_Exhausted(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	mtr := createMeter(t, deps, ns, "api-calls")
	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	// No IssueAfterReset → balance=0, HasAccess=false → usage limit reached
	createMeteredFeatureAndEntitlement(t, deps, ns, "premium", mtr, cust, nil)

	deps.streamingConnector.AddSimpleEvent(mtr.Key, 1, now)

	clock.SetTime(now.Add(time.Hour))

	res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
		Namespace:    ns,
		CustomerKeys: []string{"acme"},
		FeatureKeys:  []string{"premium"},
		PageSize:     testPageSize,
	})
	require.NoError(t, err)

	require.Len(t, res.Customers, 1)
	assert.Empty(t, res.Errors)

	fa := res.Customers[0].Features["premium"]

	assert.False(t, fa.HasAccess)
	require.NotNil(t, fa.Reason)
	assert.Equal(t, governance.ReasonCodeUsageLimitReached, fa.Reason.Code)
}

func TestQueryAccess_Pagination(t *testing.T) {
	// given: 3 customers (c1, c2, c3 in creation order); pageSize=1
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	svc := newTestService(t, deps)
	ns := newTestNamespace(t)

	createCustomer(t, deps, ns, "c1", []string{"c1"})
	clock.SetTime(now.Add(time.Second))
	createCustomer(t, deps, ns, "c2", []string{"c2"})
	clock.SetTime(now.Add(2 * time.Second))
	createCustomer(t, deps, ns, "c3", []string{"c3"})

	allKeys := []string{"c1", "c2", "c3"}

	query := func(after, before *pagination.Cursor) governance.QueryResult {
		res, err := svc.QueryAccess(t.Context(), governance.QueryAccessInput{
			Namespace:    ns,
			CustomerKeys: allKeys,
			PageSize:     1,
			After:        after,
			Before:       before,
		})
		require.NoError(t, err)

		return res
	}

	// Page 1: [c1] — no previous, next set
	page1 := query(nil, nil)
	require.Len(t, page1.Customers, 1)
	assert.Equal(t, "c1", page1.Customers[0].Matched[0])
	assert.False(t, page1.HasPrev, "no previous on first page")
	require.True(t, page1.HasNext, "next must be set on page 1")
	require.NotNil(t, page1.Last)

	// Page 2: [c2] — previous and next set. Forward uses the prior page's Last cursor.
	page2 := query(page1.Last, nil)
	require.Len(t, page2.Customers, 1)
	assert.Equal(t, "c2", page2.Customers[0].Matched[0])
	require.True(t, page2.HasPrev, "previous must be set on page 2")
	require.True(t, page2.HasNext, "next must be set on page 2")
	require.NotNil(t, page2.First)
	require.NotNil(t, page2.Last)

	// Page 3 (last): [c3] — previous set, no next
	page3 := query(page2.Last, nil)
	require.Len(t, page3.Customers, 1)
	assert.Equal(t, "c3", page3.Customers[0].Matched[0])
	assert.True(t, page3.HasPrev, "previous must be set on last page")
	assert.False(t, page3.HasNext, "no next on last page")
	require.NotNil(t, page3.Last)

	// Cursor past end → empty page, no cursors
	pastEnd := query(page3.Last, nil)
	assert.Empty(t, pastEnd.Customers)
	assert.False(t, pastEnd.HasNext)
	assert.Nil(t, pastEnd.First)
	assert.Nil(t, pastEnd.Last)

	// Backward from page 2's first (previous) cursor → [c1], no previous, next set
	pageBack := query(nil, page2.First)
	require.Len(t, pageBack.Customers, 1)
	assert.Equal(t, "c1", pageBack.Customers[0].Matched[0])
	assert.False(t, pageBack.HasPrev, "no previous before c1")
	assert.True(t, pageBack.HasNext, "next must be set in backward result")
}
