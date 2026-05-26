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
	// When feature.keys is omitted, all entitlements are returned.
	deps := setupTestDeps(t)
	t.Cleanup(func() { deps.close(t) })

	h := newTestHandler(deps)
	ns := newTestNamespace(t)

	cust := createCustomer(t, deps, ns, "acme", []string{"acme"})
	createBooleanFeatureAndEntitlement(t, deps, ns, "feat-1", cust)
	createBooleanFeatureAndEntitlement(t, deps, ns, "feat-2", cust)

	resp, err := h.processGovernanceQuery(t.Context(), queryGovernanceAccessRequest{
		Namespace: ns,
		Body:      api.GovernanceQueryRequest{Customer: api.GovernanceQueryRequestCustomers{Keys: []string{"acme"}}},
		PageSize:  defaultPageSize,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Len(t, resp.Data[0].Features, 2)
	assert.True(t, resp.Data[0].Features["feat-1"].HasAccess)
	assert.True(t, resp.Data[0].Features["feat-2"].HasAccess)
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
