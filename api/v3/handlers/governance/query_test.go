package governance

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	api "github.com/openmeterio/openmeter/api/v3"
	"github.com/openmeterio/openmeter/app/config"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customeradapter "github.com/openmeterio/openmeter/openmeter/customer/adapter"
	customerservice "github.com/openmeterio/openmeter/openmeter/customer/service"
	entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
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

func newTestNamespace(t *testing.T) string {
	t.Helper()
	return ulid.Make().String()
}

// migrateOnce serializes schema migrations to avoid concurrent-write errors from ent.
var migrateOnce sync.Mutex

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
	testdb := testutils.InitPostgresDB(t)
	dbClient := testdb.EntDriver.Client()

	migrateOnce.Lock()
	require.NoError(t, dbClient.Schema.Create(context.Background()))
	migrateOnce.Unlock()

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

func newTestHandler(deps *testDeps) *handler {
	return &handler{
		resolveNamespace:   func(_ context.Context) (string, error) { panic("not used in direct calls") },
		customerService:    deps.customerService,
		entitlementService: deps.registry.Entitlement,
		featureConnector:   deps.registry.Feature,
	}
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

// --- Tests ---

func TestQueryGovernanceAccess_UnknownCustomerKey(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body:      api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"ghost"}}},
		PageSize:  defaultPageSize,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Data)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, api.GovernanceQueryErrorCodeCustomerNotFound, resp.Errors[0].Code)
	assert.Equal(t, lo.ToPtr("ghost"), resp.Errors[0].Customer)
}

func TestQueryGovernanceAccess_KnownCustomerNoEntitlements(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body:      api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: []string{cust.GetUsageAttribution().SubjectKeys[0]}}},
		PageSize:  defaultPageSize,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Empty(t, resp.Data[0].Features)
	assert.Empty(t, resp.Errors)
}

func TestQueryGovernanceAccess_BooleanEntitlement_HasAccess(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	createBooleanFeatureAndEntitlement(t, deps, ns, "premium", cust)

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body: api.GovernanceQueryRequest{
			Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"acme"}},
			Feature:  &api.GovernanceQueryRequestFeatures{Keys: []string{"premium"}},
		},
		PageSize: defaultPageSize,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Empty(t, resp.Errors)

	featureAccess := resp.Data[0].Features["premium"]
	assert.True(t, featureAccess.HasAccess)
	assert.Nil(t, featureAccess.Reason)
}

func TestQueryGovernanceAccess_FeatureNotFound(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	_ = cust

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body: api.GovernanceQueryRequest{
			Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"acme"}},
			Feature:  &api.GovernanceQueryRequestFeatures{Keys: []string{"does-not-exist"}},
		},
		PageSize: defaultPageSize,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	featureAccess := resp.Data[0].Features["does-not-exist"]
	assert.False(t, featureAccess.HasAccess)
	require.NotNil(t, featureAccess.Reason)
	assert.Equal(t, api.GovernanceFeatureAccessReasonCodeFeatureNotFound, featureAccess.Reason.Code)
}

func TestQueryGovernanceAccess_FeatureUnavailable(t *testing.T) {
	// Feature exists in org but customer has no entitlement for it.
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	_ = cust
	createOrphanFeature(t, deps, ns, "enterprise")

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body: api.GovernanceQueryRequest{
			Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"acme"}},
			Feature:  &api.GovernanceQueryRequestFeatures{Keys: []string{"enterprise"}},
		},
		PageSize: defaultPageSize,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	featureAccess := resp.Data[0].Features["enterprise"]
	assert.False(t, featureAccess.HasAccess)
	require.NotNil(t, featureAccess.Reason)
	assert.Equal(t, api.GovernanceFeatureAccessReasonCodeFeatureUnavailable, featureAccess.Reason.Code)
}

func TestQueryGovernanceAccess_MultipleKeysSameCustomer(t *testing.T) {
	// Two input keys resolve to the same customer; response has one entry with both keys in matched[].
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	// customer key = "acme", usage attribution subject key = "acme-sub"
	createCustomer(t, deps, ns, "acme", []string{"acme-sub"})

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body:      api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"acme", "acme-sub"}}},
		PageSize:  defaultPageSize,
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Errors)
	require.Len(t, resp.Data, 1, "two keys resolving to same customer should collapse into one result")
	assert.Len(t, resp.Data[0].Matched, 2)
	assert.ElementsMatch(t, []string{"acme", "acme-sub"}, resp.Data[0].Matched)
}

func TestQueryGovernanceAccess_MixedHitsAndMisses(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	createBooleanFeatureAndEntitlement(t, deps, ns, "feature-a", cust)

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body: api.GovernanceQueryRequest{
			Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"acme", "unknown-key"}},
			Feature:  &api.GovernanceQueryRequestFeatures{Keys: []string{"feature-a"}},
		},
		PageSize: defaultPageSize,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	require.Len(t, resp.Errors, 1)
	assert.Equal(t, api.GovernanceQueryErrorCodeCustomerNotFound, resp.Errors[0].Code)
	assert.True(t, resp.Data[0].Features["feature-a"].HasAccess)
}

func TestQueryGovernanceAccess_NoFeatureKeysReturnsAll(t *testing.T) {
	// When feature.keys is omitted, all org features are returned — including ones
	// the customer has no entitlement for (marked FEATURE_UNAVAILABLE).
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	createBooleanFeatureAndEntitlement(t, deps, ns, "feat-1", cust)
	createBooleanFeatureAndEntitlement(t, deps, ns, "feat-2", cust)
	// feat-3 exists in the org but the customer has no entitlement for it.
	createOrphanFeature(t, deps, ns, "feat-3")

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body:      api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"acme"}}},
		PageSize:  defaultPageSize,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Len(t, resp.Data[0].Features, 3)
	assert.True(t, resp.Data[0].Features["feat-1"].HasAccess)
	assert.True(t, resp.Data[0].Features["feat-2"].HasAccess)
	feat3 := resp.Data[0].Features["feat-3"]
	assert.False(t, feat3.HasAccess)
	require.NotNil(t, feat3.Reason)
	assert.Equal(t, api.GovernanceFeatureAccessReasonCodeFeatureUnavailable, feat3.Reason.Code)
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

func TestQueryGovernanceAccess_MeteredEntitlement_HasAccess(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	mtr := createMeter(t, deps, ns, "api-calls")
	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	// IssueAfterReset=10.0 → balance starts at 10, HasAccess=true
	createMeteredFeatureAndEntitlement(t, deps, ns, "premium", mtr, cust, lo.ToPtr(10.0))

	// Add an event so the streaming mock has data for the meter.
	deps.streamingConnector.AddSimpleEvent(mtr.Key, 1, now)

	clock.SetTime(now.Add(time.Hour))

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body: api.GovernanceQueryRequest{
			Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"acme"}},
			Feature:  &api.GovernanceQueryRequestFeatures{Keys: []string{"premium"}},
		},
		PageSize: defaultPageSize,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Empty(t, resp.Errors)
	featureAccess := resp.Data[0].Features["premium"]
	assert.True(t, featureAccess.HasAccess)
	assert.Nil(t, featureAccess.Reason)
}

func TestQueryGovernanceAccess_MeteredEntitlement_Exhausted(t *testing.T) {
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	mtr := createMeter(t, deps, ns, "api-calls")
	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	// No IssueAfterReset → balance=0, HasAccess=false → UsageLimitReached
	createMeteredFeatureAndEntitlement(t, deps, ns, "premium", mtr, cust, nil)

	deps.streamingConnector.AddSimpleEvent(mtr.Key, 1, now)

	clock.SetTime(now.Add(time.Hour))

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body: api.GovernanceQueryRequest{
			Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"acme"}},
			Feature:  &api.GovernanceQueryRequestFeatures{Keys: []string{"premium"}},
		},
		PageSize: defaultPageSize,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Empty(t, resp.Errors)
	featureAccess := resp.Data[0].Features["premium"]
	assert.False(t, featureAccess.HasAccess)
	require.NotNil(t, featureAccess.Reason)
	assert.Equal(t, api.GovernanceFeatureAccessReasonCodeUsageLimitReached, featureAccess.Reason.Code)
}

func TestQueryGovernanceAccess_Pagination(t *testing.T) {
	// given: 3 customers (c1, c2, c3 in creation order); pageSize=1
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	clock.SetTime(now)
	defer clock.ResetTime()

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	createCustomer(t, deps, ns, "c1", []string{"c1"})
	clock.SetTime(now.Add(time.Second))
	createCustomer(t, deps, ns, "c2", []string{"c2"})
	clock.SetTime(now.Add(2 * time.Second))
	createCustomer(t, deps, ns, "c3", []string{"c3"})

	allKeys := []string{"c1", "c2", "c3"}

	decodeCursor := func(encoded string) *pagination.Cursor {
		c, err := pagination.DecodeCursor(encoded)
		require.NoError(t, err)
		return c
	}

	// Page 1: [c1] — no previous, next points to c1
	page1, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body:      api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: allKeys}},
		PageSize:  1,
	})
	require.NoError(t, err)
	require.Len(t, page1.Data, 1)
	assert.Equal(t, "c1", page1.Data[0].Matched[0])
	assert.True(t, page1.Meta.Page.Previous.IsNull(), "no previous on first page")
	require.False(t, page1.Meta.Page.Next.IsNull(), "next must be set on page 1")

	next1, _ := page1.Meta.Page.Next.Get()

	// Page 2: [c2] — previous set, next set
	page2, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace:   ns,
		Body:        api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: allKeys}},
		PageSize:    1,
		AfterCursor: decodeCursor(next1),
	})
	require.NoError(t, err)
	require.Len(t, page2.Data, 1)
	assert.Equal(t, "c2", page2.Data[0].Matched[0])
	require.False(t, page2.Meta.Page.Previous.IsNull(), "previous must be set on page 2")
	require.False(t, page2.Meta.Page.Next.IsNull(), "next must be set on page 2")

	next2, _ := page2.Meta.Page.Next.Get()
	prev2, _ := page2.Meta.Page.Previous.Get()

	// Page 3 (last): [c3] — previous set, no next
	page3, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace:   ns,
		Body:        api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: allKeys}},
		PageSize:    1,
		AfterCursor: decodeCursor(next2),
	})
	require.NoError(t, err)
	require.Len(t, page3.Data, 1)
	assert.Equal(t, "c3", page3.Data[0].Matched[0])
	require.False(t, page3.Meta.Page.Previous.IsNull(), "previous must be set on last page")
	assert.True(t, page3.Meta.Page.Next.IsNull(), "no next on last page")

	// Cursor past end → empty data
	require.NotNil(t, page3.Meta.Page.Last, "last cursor must be set")
	last3 := *page3.Meta.Page.Last
	pastEnd, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace:   ns,
		Body:        api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: allKeys}},
		PageSize:    1,
		AfterCursor: decodeCursor(last3),
	})
	require.NoError(t, err)
	assert.Empty(t, pastEnd.Data)
	assert.True(t, pastEnd.Meta.Page.Next.IsNull())
	assert.True(t, pastEnd.Meta.Page.Previous.IsNull())

	// Backward from page 2's previous cursor → [c1], no previous, next set
	pageBack, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace:    ns,
		Body:         api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: allKeys}},
		PageSize:     1,
		BeforeCursor: decodeCursor(prev2),
	})
	require.NoError(t, err)
	require.Len(t, pageBack.Data, 1)
	assert.Equal(t, "c1", pageBack.Data[0].Matched[0])
	assert.True(t, pageBack.Meta.Page.Previous.IsNull(), "no previous before c1")
	assert.False(t, pageBack.Meta.Page.Next.IsNull(), "next must be set in backward result")
}
