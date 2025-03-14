package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	billingadapter "github.com/openmeterio/openmeter/openmeter/billing/adapter"
	billingservice "github.com/openmeterio/openmeter/openmeter/billing/service"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/ingestdriver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meterhttphandler "github.com/openmeterio/openmeter/openmeter/meter/httphandler"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	metereventadapter "github.com/openmeterio/openmeter/openmeter/meterevent/adapter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	portaladapter "github.com/openmeterio/openmeter/openmeter/portal/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	progressmanageradapter "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	subscriptiontestutils "github.com/openmeterio/openmeter/openmeter/subscription/testutils"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/log"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
)

var DefaultNamespace = "test"

var mockEvent = event.New()

var mockMeters = []meter.Meter{
	{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: DefaultNamespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Meter 1",
		},
		Key:           "meter1",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "event",
		ValueProperty: lo.ToPtr("$.value"),
		GroupBy:       map[string]string{"path": "$.path", "method": "$.method"},
	},
	{
		ManagedResource: models.ManagedResource{
			ID: ulid.Make().String(),
			NamespacedModel: models.NamespacedModel{
				Namespace: DefaultNamespace,
			},
			ManagedModel: models.ManagedModel{
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			Name: "Meter 2",
		},
		Key:           "meter2",
		Aggregation:   meter.MeterAggregationSum,
		EventType:     "event",
		ValueProperty: lo.ToPtr("$.value"),
	},
}

var (
	mockSubject    = "s1"
	mockQueryValue = meter.MeterQueryRow{
		Subject:     &mockSubject,
		WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
		Value:       300,
	}
)

type MockHandler struct{}

func (h MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, namespace string) {
	w.WriteHeader(http.StatusOK)
}

type testRequest struct {
	method      string
	path        string
	accept      string
	contentType string
	body        interface{}
}

type testResponse struct {
	status int
	body   interface{}
}

func TestRoutes(t *testing.T) {
	testServer := getTestServer(t)

	tests := []struct {
		name string
		req  testRequest
		res  testResponse
	}{
		// Events
		{
			name: "ingest event",
			req: testRequest{
				method:      http.MethodPost,
				path:        "/api/v1/events",
				contentType: "application/cloudevents+json",
				body: func() *api.Event {
					e := event.New()
					e.SetID("test-1")
					e.SetType("type")
					e.SetSubject("subject")
					e.SetSource("source")
					return &e
				}(),
			},
			res: testResponse{
				status: http.StatusNoContent,
			},
		},
		{
			name: "query events",
			req: testRequest{
				method:      http.MethodGet,
				path:        "/api/v1/events",
				contentType: "application/json",
			},
			res: testResponse{
				status: http.StatusOK,
				body: []api.IngestedEvent{
					{Event: mockEvent},
				},
			},
		},
		// Meters
		{
			name: "list meters",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/meters",
			},
			res: testResponse{
				status: http.StatusOK,
				body: []api.Meter{
					meterhttphandler.ToAPIMeter(mockMeters[0]),
					meterhttphandler.ToAPIMeter(mockMeters[1]),
				},
			},
		},
		{
			name: "get meter",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/meters/" + mockMeters[0].Key,
			},
			res: testResponse{
				status: http.StatusOK,
				body:   meterhttphandler.ToAPIMeter(mockMeters[0]),
			},
		},
		{
			name: "query meter",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Key + "/query",
			},
			res: testResponse{
				status: http.StatusOK,
				body: struct {
					Data []api.MeterQueryRow `json:"data"`
				}{
					Data: []api.MeterQueryRow{
						{WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC), Value: 300},
					},
				},
			},
		},
		{
			name: "query meter with valid group by",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Key + "/query?groupBy=path&groupBy=method",
			},
			res: testResponse{
				status: http.StatusOK,
			},
		},
		{
			name: "query meter with subject group by",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Key + "/query?groupBy=subject",
			},
			res: testResponse{
				status: http.StatusOK,
			},
		},
		{
			name: "query meter with invalid group by",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Key + "/query?groupBy=foo",
			},
			res: testResponse{
				status: http.StatusBadRequest,
			},
		},
		{
			name: "query meter with subject",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Key + "/query?subject=s1",
			},
			res: testResponse{
				status: http.StatusOK,
				body: struct {
					Data []api.MeterQueryRow `json:"data"`
				}{
					Data: []api.MeterQueryRow{
						{Subject: mockQueryValue.Subject, WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC), Value: 300},
					},
				},
			},
		},
		{
			name: "query meter with filter",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Key + "/query?filterGroupBy[method]=GET",
			},
			res: testResponse{
				status: http.StatusOK,
				body: struct {
					Data []api.MeterQueryRow `json:"data"`
				}{
					Data: []api.MeterQueryRow{
						{WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC), Value: 300},
					},
				},
			},
		},
		{
			name: "query meter with invalid group by filter",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Key + "/query?filterGroupBy[invalid]=abcd",
			},
			res: testResponse{
				status: http.StatusBadRequest,
			},
		},
		{
			name: "query meter as csv",
			req: testRequest{
				accept:      "text/csv",
				contentType: "text/csv",
				method:      http.MethodGet,
				path:        "/api/v1/meters/" + mockMeters[0].Key + "/query",
			},
			res: testResponse{
				status: http.StatusOK,
				body: strings.Join(
					[]string{
						"window_start,window_end,subject,value",
						"2021-01-01T00:00:00Z,2021-01-01T01:00:00Z,,300.000000",
						"",
					},
					"\n",
				),
			},
		},
		{
			name: "query meter as csv with subject",
			req: testRequest{
				accept:      "text/csv",
				contentType: "text/csv",
				method:      http.MethodGet,
				path:        "/api/v1/meters/" + mockMeters[0].Key + "/query?subject=s1",
			},
			res: testResponse{
				status: http.StatusOK,
				body: strings.Join(
					[]string{
						"window_start,window_end,subject,value",
						"2021-01-01T00:00:00Z,2021-01-01T01:00:00Z,s1,300.000000",
						"",
					},
					"\n",
				),
			},
		},
		{
			name: "list meter subjects",
			req: testRequest{
				method: http.MethodGet,
				path:   fmt.Sprintf("/api/v1/meters/%s/subjects", mockMeters[0].Key),
			},
			res: testResponse{
				status: http.StatusOK,
				body:   []string{"s1"},
			},
		},
		// Portal
		{
			name: "create portal token",
			req: testRequest{
				method:      http.MethodPost,
				path:        "/api/v1/portal/tokens",
				contentType: "application/json",
				body: api.PortalToken{
					Subject: "customer-1",
				},
			},
			res: testResponse{
				status: http.StatusOK,
			},
		},
		{
			name: "list portal tokens",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/portal/tokens",
			},
			res: testResponse{
				status: http.StatusNotImplemented,
			},
		},
		// Debug
		{
			name: "get debug metrics",
			req: testRequest{
				method:      http.MethodGet,
				path:        "/api/v1/debug/metrics",
				contentType: "text/plain",
				body:        `openmeter_events_total{subject="customer-1"} 2.0`,
			},
			res: testResponse{
				status: http.StatusOK,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			if tt.req.body != nil {
				reqBody, _ = json.Marshal(tt.req.body)
			}
			req := httptest.NewRequest(tt.req.method, tt.req.path, bytes.NewReader(reqBody))
			if tt.req.accept != "" {
				req.Header.Set("Accept", tt.req.accept)
			}
			if tt.req.contentType != "" {
				req.Header.Set("Content-Type", tt.req.contentType)
			}

			// Make request
			writer := httptest.NewRecorder()
			testServer.ServeHTTP(writer, req)
			res := writer.Result()

			defer res.Body.Close()

			// status
			assert.Equal(t, tt.res.status, res.StatusCode, writer.Body.String())

			// body
			if tt.res.body == nil {
				return
			}

			switch tt.req.accept {
			case "text/csv":
				assert.Equal(t, tt.res.body, writer.Body.String())
			default:
				// Handle default as "application/json"
				resBody, _ := json.Marshal(tt.res.body)
				assert.JSONEq(t, string(resBody), writer.Body.String())
			}
		})
	}
}

// getTestServer returns a test server
func getTestServer(t *testing.T) *Server {
	// Initialize postgres driver
	dbDeps := subscriptiontestutils.SetupDBDeps(t)

	namespaceManager, err := namespace.NewManager(namespace.ManagerConfig{
		DefaultNamespace: DefaultNamespace,
	})
	assert.NoError(t, err, "failed to create namespace manager")

	portal, err := portaladapter.New(portaladapter.Config{
		Secret: "12345",
		Expire: time.Hour,
	})
	assert.NoError(t, err, "failed to create portal")

	mockStreamingConnector := &MockStreamingConnector{}

	meterManageService, err := meteradapter.NewManage(mockMeters)
	assert.NoError(t, err, "failed to create meter service")

	meterEventService := metereventadapter.New(mockStreamingConnector, meterManageService)

	logger := slog.New(log.NewMockHandler())

	// Create feature service
	featureService := &NoopFeatureConnector{}

	// Create app service
	appService := &NoopAppService{}
	appStripeService := &NoopAppStripeService{}

	// Create customer service
	customerService := &NoopCustomerService{}

	// Create billing service
	billingAdapter, err := billingadapter.New(billingadapter.Config{
		Client: dbDeps.DBClient,
		Logger: logger,
	})
	assert.NoError(t, err, "failed to create billing adapter")

	billingService, err := billingservice.New(billingservice.Config{
		Adapter:             billingAdapter,
		AppService:          appService,
		AdvancementStrategy: billing.ForegroundAdvancementStrategy,
		CustomerService:     customerService,
		Logger:              logger,
		FeatureService:      featureService,
		MeterService:        meterManageService,
		StreamingConnector:  mockStreamingConnector,
		Publisher:           NewNoopPublisher(),
	})
	assert.NoError(t, err, "failed to create billing service")

	config := &Config{
		RouterConfig: router.Config{
			App:                         appService,
			AppStripe:                   appStripeService,
			Billing:                     billingService,
			Customer:                    customerService,
			DebugConnector:              MockDebugHandler{},
			EntitlementConnector:        &NoopEntitlementConnector{},
			EntitlementBalanceConnector: &NoopEntitlementBalanceConnector{},
			ErrorHandler:                errorsx.NopHandler{},
			FeatureConnector:            featureService,
			GrantConnector:              &NoopGrantConnector{},
			IngestHandler: ingestdriver.NewIngestEventsHandler(func(ctx context.Context, request ingest.IngestEventsRequest) (bool, error) {
				return true, nil
			}, namespacedriver.StaticNamespaceDecoder("test"), nil, errorsx.NewNopHandler()),
			MeterManageService: meterManageService,
			MeterEventService:  meterEventService,
			NamespaceManager:   namespaceManager,
			Notification:       &NoopNotificationService{},
			Portal:             portal,
			ProgressManager:    progressmanageradapter.NewNoop(),
			StreamingConnector: mockStreamingConnector,
		},
		RouterHook: func(r chi.Router) {},
	}

	// Create server
	server, err := NewServer(config)
	assert.NoError(t, err, "failed to create server")
	return server
}

// NoopPublisher is a publisher that does nothing (no-operation)
// Useful for testing or when publishing needs to be disabled
type NoopPublisher struct {
	marshaler marshaler.Marshaler
}

// NewNoopPublisher creates a new NoopPublisher
func NewNoopPublisher() *NoopPublisher {
	return &NoopPublisher{
		marshaler: marshaler.New(nil),
	}
}

// Publish implements Publisher.Publish but does nothing and always returns nil
func (p *NoopPublisher) Publish(ctx context.Context, event marshaler.Event) error {
	return nil
}

// Marshaler returns the marshaler associated with this publisher
func (p *NoopPublisher) Marshaler() marshaler.Marshaler {
	return p.marshaler
}

// WithContext returns a no-op context publisher
func (p *NoopPublisher) WithContext(ctx context.Context) eventbus.ContextPublisher {
	return noopContextPublisher{ctx: ctx}
}

// noopContextPublisher is a no-op implementation of ContextPublisher
type noopContextPublisher struct {
	ctx context.Context
}

// PublishIfNoError implements ContextPublisher.PublishIfNoError
// It returns the provided error if not nil, otherwise returns nil
func (p noopContextPublisher) PublishIfNoError(event marshaler.Event, err error) error {
	return err
}

// MockDebugHandler
type MockDebugHandler struct{}

func (h MockDebugHandler) GetDebugMetrics(ctx context.Context, namespace string) (string, error) {
	return `openmeter_events_total{subject="customer-1"} 2.0`, nil
}

// MockStreamingConnector
type MockStreamingConnector struct{}

func (c *MockStreamingConnector) CreateNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (c *MockStreamingConnector) DeleteNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (c *MockStreamingConnector) CountEvents(ctx context.Context, namespace string, params streaming.CountEventsParams) ([]streaming.CountEventRow, error) {
	return []streaming.CountEventRow{}, nil
}

func (c *MockStreamingConnector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]api.IngestedEvent, error) {
	events := []api.IngestedEvent{
		{
			Event: mockEvent,
		},
	}
	return events, nil
}

func (c *MockStreamingConnector) CreateMeter(ctx context.Context, namespace string, meter meter.Meter) error {
	return nil
}

func (c *MockStreamingConnector) UpdateMeter(ctx context.Context, namespace string, meter meter.Meter) error {
	return nil
}

func (c *MockStreamingConnector) DeleteMeter(ctx context.Context, namespace string, meter meter.Meter) error {
	return nil
}

func (c *MockStreamingConnector) QueryMeter(ctx context.Context, namespace string, m meter.Meter, params streaming.QueryParams) ([]meter.MeterQueryRow, error) {
	value := mockQueryValue

	if params.FilterSubject == nil {
		value.Subject = nil
	}

	return []meter.MeterQueryRow{value}, nil
}

func (c *MockStreamingConnector) ListMeterSubjects(ctx context.Context, namespace string, meter meter.Meter, params streaming.ListMeterSubjectsParams) ([]string, error) {
	return []string{"s1"}, nil
}

func (c *MockStreamingConnector) BatchInsert(ctx context.Context, events []streaming.RawEvent) error {
	return nil
}

func (c *MockStreamingConnector) ValidateJSONPath(ctx context.Context, jsonPath string) (bool, error) {
	return true, nil
}

// NoopFeatureConnector
var _ feature.FeatureConnector = (*NoopFeatureConnector)(nil)

type NoopFeatureConnector struct{}

func (n NoopFeatureConnector) ListFeatures(ctx context.Context, params feature.ListFeaturesParams) (pagination.PagedResponse[feature.Feature], error) {
	return pagination.PagedResponse[feature.Feature]{}, nil
}

func (n NoopFeatureConnector) GetFeature(ctx context.Context, namespace string, idOrKey string, includeArchived feature.IncludeArchivedFeature) (*feature.Feature, error) {
	return &feature.Feature{}, nil
}

func (n NoopFeatureConnector) CreateFeature(ctx context.Context, input feature.CreateFeatureInputs) (feature.Feature, error) {
	return feature.Feature{}, nil
}

func (n NoopFeatureConnector) DeleteFeature(ctx context.Context, namespace string, key string) error {
	return nil
}

func (n NoopFeatureConnector) ArchiveFeature(ctx context.Context, featureID models.NamespacedID) error {
	return nil
}

// NoopEntitlementConnector
var _ entitlement.Connector = (*NoopEntitlementConnector)(nil)

type NoopEntitlementConnector struct{}

func (n NoopEntitlementConnector) CreateEntitlement(ctx context.Context, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) ScheduleEntitlement(ctx context.Context, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) OverrideEntitlement(ctx context.Context, subject string, entitlementIdOrFeatureKey string, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) SupersedeEntitlement(ctx context.Context, entitlementId string, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) GetEntitlement(ctx context.Context, namespace string, id string) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) DeleteEntitlement(ctx context.Context, namespace string, id string, at time.Time) error {
	return nil
}

func (n NoopEntitlementConnector) GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (entitlement.EntitlementValue, error) {
	return nil, nil
}

func (n NoopEntitlementConnector) GetEntitlementsOfSubject(ctx context.Context, namespace string, subjectKey string, at time.Time) ([]entitlement.Entitlement, error) {
	return []entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) ListEntitlements(ctx context.Context, params entitlement.ListEntitlementsParams) (pagination.PagedResponse[entitlement.Entitlement], error) {
	return pagination.PagedResponse[entitlement.Entitlement]{}, nil
}

func (n NoopEntitlementConnector) GetEntitlementOfSubjectAt(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

// NoopEntitlementBalanceConnector

var _ meteredentitlement.Connector = (*NoopEntitlementBalanceConnector)(nil)

type NoopEntitlementBalanceConnector struct{}

func (n NoopEntitlementBalanceConnector) GetEntitlementBalance(ctx context.Context, entitlementID models.NamespacedID, at time.Time) (*meteredentitlement.EntitlementBalance, error) {
	return nil, nil
}

func (n NoopEntitlementBalanceConnector) GetEntitlementBalanceHistory(ctx context.Context, entitlementID models.NamespacedID, params meteredentitlement.BalanceHistoryParams) ([]meteredentitlement.EntitlementBalanceHistoryWindow, engine.GrantBurnDownHistory, error) {
	return []meteredentitlement.EntitlementBalanceHistoryWindow{}, engine.GrantBurnDownHistory{}, nil
}

func (n NoopEntitlementBalanceConnector) ResetEntitlementUsage(ctx context.Context, entitlementID models.NamespacedID, params meteredentitlement.ResetEntitlementUsageParams) (balanceAfterReset *meteredentitlement.EntitlementBalance, err error) {
	return nil, nil
}

func (n NoopEntitlementBalanceConnector) ResetEntitlementsWithExpiredUsagePeriod(ctx context.Context, namespace string, highwatermark time.Time) ([]models.NamespacedID, error) {
	return []models.NamespacedID{}, nil
}

func (n NoopEntitlementBalanceConnector) CreateGrant(ctx context.Context, namespace string, subjectKey string, entitlementIdOrFeatureKey string, inputGrant meteredentitlement.CreateEntitlementGrantInputs) (meteredentitlement.EntitlementGrant, error) {
	return meteredentitlement.EntitlementGrant{}, nil
}

func (n NoopEntitlementBalanceConnector) ListEntitlementGrants(ctx context.Context, namespace string, subjectKey string, entitlementIdOrFeatureKey string) ([]meteredentitlement.EntitlementGrant, error) {
	return []meteredentitlement.EntitlementGrant{}, nil
}

func (n NoopEntitlementBalanceConnector) GetValue(ctx context.Context, entitlement *entitlement.Entitlement, at time.Time) (entitlement.EntitlementValue, error) {
	return nil, nil
}

func (n NoopEntitlementBalanceConnector) BeforeCreate(entitlement entitlement.CreateEntitlementInputs, feature feature.Feature) (*entitlement.CreateEntitlementRepoInputs, error) {
	return nil, nil
}

func (n NoopEntitlementBalanceConnector) AfterCreate(ctx context.Context, entitlement *entitlement.Entitlement) error {
	return nil
}

// NoopGrantConnector

var _ credit.GrantConnector = (*NoopGrantConnector)(nil)

type NoopGrantConnector struct{}

func (n NoopGrantConnector) CreateGrant(ctx context.Context, owner models.NamespacedID, input credit.CreateGrantInput) (*grant.Grant, error) {
	return &grant.Grant{}, nil
}

func (n NoopGrantConnector) VoidGrant(ctx context.Context, grantID models.NamespacedID) error {
	return nil
}

// NoopNotificationService
var _ notification.Service = (*NoopNotificationService)(nil)

type NoopNotificationService struct{}

func (n NoopNotificationService) ListFeature(_ context.Context, _ string, _ ...string) ([]feature.Feature, error) {
	return []feature.Feature{}, nil
}

func (n NoopNotificationService) ListChannels(_ context.Context, _ notification.ListChannelsInput) (notification.ListChannelsResult, error) {
	return notification.ListChannelsResult{}, nil
}

func (n NoopNotificationService) CreateChannel(_ context.Context, _ notification.CreateChannelInput) (*notification.Channel, error) {
	return &notification.Channel{}, nil
}

func (n NoopNotificationService) DeleteChannel(_ context.Context, _ notification.DeleteChannelInput) error {
	return nil
}

func (n NoopNotificationService) GetChannel(_ context.Context, _ notification.GetChannelInput) (*notification.Channel, error) {
	return &notification.Channel{}, nil
}

func (n NoopNotificationService) UpdateChannel(_ context.Context, _ notification.UpdateChannelInput) (*notification.Channel, error) {
	return &notification.Channel{}, nil
}

func (n NoopNotificationService) ListRules(_ context.Context, _ notification.ListRulesInput) (notification.ListRulesResult, error) {
	return notification.ListRulesResult{}, nil
}

func (n NoopNotificationService) CreateRule(_ context.Context, _ notification.CreateRuleInput) (*notification.Rule, error) {
	return &notification.Rule{}, nil
}

func (n NoopNotificationService) DeleteRule(_ context.Context, _ notification.DeleteRuleInput) error {
	return nil
}

func (n NoopNotificationService) GetRule(_ context.Context, _ notification.GetRuleInput) (*notification.Rule, error) {
	return &notification.Rule{}, nil
}

func (n NoopNotificationService) UpdateRule(_ context.Context, _ notification.UpdateRuleInput) (*notification.Rule, error) {
	return &notification.Rule{}, nil
}

func (n NoopNotificationService) ListEvents(_ context.Context, _ notification.ListEventsInput) (notification.ListEventsResult, error) {
	return notification.ListEventsResult{}, nil
}

func (n NoopNotificationService) GetEvent(_ context.Context, _ notification.GetEventInput) (*notification.Event, error) {
	return &notification.Event{}, nil
}

func (n NoopNotificationService) CreateEvent(_ context.Context, _ notification.CreateEventInput) (*notification.Event, error) {
	return &notification.Event{}, nil
}

func (n NoopNotificationService) ListEventsDeliveryStatus(_ context.Context, _ notification.ListEventsDeliveryStatusInput) (notification.ListEventsDeliveryStatusResult, error) {
	return notification.ListEventsDeliveryStatusResult{}, nil
}

func (n NoopNotificationService) GetEventDeliveryStatus(_ context.Context, _ notification.GetEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	return &notification.EventDeliveryStatus{}, nil
}

func (n NoopNotificationService) UpdateEventDeliveryStatus(_ context.Context, _ notification.UpdateEventDeliveryStatusInput) (*notification.EventDeliveryStatus, error) {
	return &notification.EventDeliveryStatus{}, nil
}

func (n NoopNotificationService) Close() error {
	return nil
}

var _ app.Service = (*NoopAppService)(nil)

type NoopAppService struct{}

func (n NoopAppService) RegisterMarketplaceListing(input app.RegisterMarketplaceListingInput) error {
	return nil
}

func (n NoopAppService) GetMarketplaceListing(ctx context.Context, input app.MarketplaceGetInput) (app.RegistryItem, error) {
	return app.RegistryItem{}, nil
}

func (n NoopAppService) ListMarketplaceListings(ctx context.Context, input app.MarketplaceListInput) (pagination.PagedResponse[app.RegistryItem], error) {
	return pagination.PagedResponse[app.RegistryItem]{}, nil
}

func (n NoopAppService) InstallMarketplaceListingWithAPIKey(ctx context.Context, input app.InstallAppWithAPIKeyInput) (app.App, error) {
	return appstripeentityapp.App{}, nil
}

func (n NoopAppService) GetMarketplaceListingOauth2InstallURL(ctx context.Context, input app.GetOauth2InstallURLInput) (app.GetOauth2InstallURLOutput, error) {
	return app.GetOauth2InstallURLOutput{}, nil
}

func (n NoopAppService) AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input app.AuthorizeOauth2InstallInput) error {
	return nil
}

func (n NoopAppService) CreateApp(ctx context.Context, input app.CreateAppInput) (app.AppBase, error) {
	return app.AppBase{}, nil
}

func (n NoopAppService) UpdateAppStatus(ctx context.Context, input app.UpdateAppStatusInput) error {
	return nil
}

func (n NoopAppService) GetApp(ctx context.Context, input app.GetAppInput) (app.App, error) {
	return appstripeentityapp.App{}, nil
}

func (n NoopAppService) GetDefaultApp(ctx context.Context, input app.GetDefaultAppInput) (app.App, error) {
	return appstripeentityapp.App{}, nil
}

func (n NoopAppService) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	return appstripeentityapp.App{}, nil
}

func (n NoopAppService) ListApps(ctx context.Context, input app.ListAppInput) (pagination.PagedResponse[app.App], error) {
	return pagination.PagedResponse[app.App]{}, nil
}

func (n NoopAppService) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	return nil
}

func (n NoopAppService) ListCustomerData(ctx context.Context, input app.ListCustomerInput) (pagination.PagedResponse[app.CustomerApp], error) {
	return pagination.PagedResponse[app.CustomerApp]{}, nil
}

func (n NoopAppService) EnsureCustomer(ctx context.Context, input app.EnsureCustomerInput) error {
	return nil
}

func (n NoopAppService) DeleteCustomer(ctx context.Context, input app.DeleteCustomerInput) error {
	return nil
}

var _ appstripe.Service = (*NoopAppStripeService)(nil)

type NoopAppStripeService struct{}

func (n NoopAppStripeService) UpdateAPIKey(ctx context.Context, input appstripeentity.UpdateAPIKeyInput) error {
	return nil
}

func (n NoopAppStripeService) CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error) {
	return appstripeentity.CreateCheckoutSessionOutput{}, nil
}

func (n NoopAppStripeService) GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error) {
	return appstripeentity.GetWebhookSecretOutput{}, nil
}

func (n NoopAppStripeService) GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error) {
	return appstripeentity.AppData{}, nil
}

func (n NoopAppStripeService) HandleSetupIntentSucceeded(ctx context.Context, input appstripeentity.HandleSetupIntentSucceededInput) (appstripeentity.HandleSetupIntentSucceededOutput, error) {
	return appstripeentity.HandleSetupIntentSucceededOutput{}, nil
}

func (n NoopAppStripeService) GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerData, error) {
	return appstripeentity.CustomerData{}, nil
}

func (n NoopAppStripeService) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	return nil
}

func (n NoopAppStripeService) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	return nil
}

func (n NoopAppStripeService) GetSupplierContact(ctx context.Context, input appstripeentity.GetSupplierContactInput) (billing.SupplierContact, error) {
	return billing.SupplierContact{}, nil
}

func (n NoopAppStripeService) GetMaskedSecretAPIKey(ctx context.Context, secretAPIKeyID secretentity.SecretID) (string, error) {
	return "", nil
}

// Factory methods
func (n NoopAppStripeService) NewApp(ctx context.Context, appBase app.AppBase) (app.App, error) {
	return nil, nil
}

func (n NoopAppStripeService) InstallAppWithAPIKey(ctx context.Context, input app.AppFactoryInstallAppWithAPIKeyInput) (app.App, error) {
	return nil, nil
}

func (n NoopAppStripeService) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	return nil
}

// Billing parts
func (n NoopAppStripeService) HandleInvoiceStateTransition(ctx context.Context, input appstripeentity.HandleInvoiceStateTransitionInput) error {
	return nil
}

func (n NoopAppStripeService) HandleInvoiceSentEvent(ctx context.Context, input appstripeentity.HandleInvoiceSentEventInput) error {
	return nil
}

var _ customer.Service = (*NoopCustomerService)(nil)

type NoopCustomerService struct{}

func (n NoopCustomerService) ListCustomers(ctx context.Context, params customer.ListCustomersInput) (pagination.PagedResponse[customer.Customer], error) {
	return pagination.PagedResponse[customer.Customer]{}, nil
}

func (n NoopCustomerService) CreateCustomer(ctx context.Context, params customer.CreateCustomerInput) (*customer.Customer, error) {
	return &customer.Customer{}, nil
}

func (n NoopCustomerService) DeleteCustomer(ctx context.Context, input customer.DeleteCustomerInput) error {
	return nil
}

func (n NoopCustomerService) GetCustomer(ctx context.Context, input customer.GetCustomerInput) (*customer.Customer, error) {
	return &customer.Customer{}, nil
}

func (n NoopCustomerService) UpdateCustomer(ctx context.Context, params customer.UpdateCustomerInput) (*customer.Customer, error) {
	return &customer.Customer{}, nil
}

func (n NoopCustomerService) GetEntitlementValue(ctx context.Context, input customer.GetEntitlementValueInput) (entitlement.EntitlementValue, error) {
	return nil, nil
}

func (n NoopCustomerService) FindCustomer(ctx context.Context, namespace string, customerRef ref.IDOrKey) (*customer.Customer, error) {
	return nil, nil
}

func (n NoopCustomerService) RegisterRequestValidator(validator customer.RequestValidator) {
}

func (n NoopCustomerService) CustomerExists(ctx context.Context, customer customer.CustomerID) error {
	return nil
}
