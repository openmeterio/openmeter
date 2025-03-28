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
	"github.com/openmeterio/openmeter/openmeter/meterevent"
	metereventadapter "github.com/openmeterio/openmeter/openmeter/meterevent/adapter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	portaladapter "github.com/openmeterio/openmeter/openmeter/portal/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	progressmanageradapter "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
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
	// No need for DB setup with NoopBillingService
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

	// Create plan service
	planService := &NoopPlanService{}
	addonService := &NoopAddonService{}

	// Create plan subscription service
	planSubscriptionService := &NoopPlanSubscriptionService{}

	// Create subscription services
	subscriptionService := &NoopSubscriptionService{}
	subscriptionWorkflowService := &NoopSubscriptionWorkflowService{}

	// Create grant repo
	grantRepo := &NoopGrantRepo{}

	// Create billing service
	billingService := &NoopBillingService{}

	config := &Config{
		RouterConfig: router.Config{
			Addon:                       addonService,
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
			// Use the grant repo
			GrantRepo: grantRepo,
			IngestHandler: ingestdriver.NewIngestEventsHandler(func(ctx context.Context, request ingest.IngestEventsRequest) (bool, error) {
				return true, nil
			}, namespacedriver.StaticNamespaceDecoder("test"), nil, errorsx.NewNopHandler()),
			Logger:             logger,
			MeterManageService: meterManageService,
			MeterEventService:  meterEventService,
			NamespaceManager:   namespaceManager,
			Notification:       &NoopNotificationService{},
			// Use the plan service
			Plan: planService,
			// Use the plan subscription service
			PlanSubscriptionService: planSubscriptionService,
			Portal:                  portal,
			ProgressManager:         progressmanageradapter.NewNoop(),
			StreamingConnector:      mockStreamingConnector,
			// Use the subscription service
			SubscriptionService: subscriptionService,
			// Use the subscription workflow service
			SubscriptionWorkflowService: subscriptionWorkflowService,
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

func (c *MockStreamingConnector) ListEvents(ctx context.Context, namespace string, params meterevent.ListEventsParams) ([]streaming.RawEvent, error) {
	events := []streaming.RawEvent{
		{
			ID:         mockEvent.ID(),
			Type:       mockEvent.Type(),
			Source:     mockEvent.Source(),
			Subject:    mockEvent.Subject(),
			Time:       mockEvent.Time(),
			Data:       string(mockEvent.Data()),
			IngestedAt: time.Time{},
			StoredAt:   time.Time{},
		},
	}
	return events, nil
}

func (c *MockStreamingConnector) ListEventsV2(ctx context.Context, params meterevent.ListEventsV2Params) ([]streaming.RawEvent, error) {
	events := []streaming.RawEvent{
		{
			ID:         mockEvent.ID(),
			Type:       mockEvent.Type(),
			Source:     mockEvent.Source(),
			Subject:    mockEvent.Subject(),
			Time:       mockEvent.Time(),
			Data:       string(mockEvent.Data()),
			IngestedAt: time.Time{},
			StoredAt:   time.Time{},
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

func (n NoopEntitlementConnector) GetAccess(ctx context.Context, namespace string, subjectKey string) (entitlement.Access, error) {
	return entitlement.Access{}, nil
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

// NoopPlanService implements plan.Service interface with no-op operations
// for use in testing
type NoopPlanService struct{}

func (n NoopPlanService) ListPlans(ctx context.Context, params plan.ListPlansInput) (pagination.PagedResponse[plan.Plan], error) {
	return pagination.PagedResponse[plan.Plan]{}, nil
}

func (n NoopPlanService) CreatePlan(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
	return &plan.Plan{}, nil
}

func (n NoopPlanService) DeletePlan(ctx context.Context, params plan.DeletePlanInput) error {
	return nil
}

func (n NoopPlanService) GetPlan(ctx context.Context, params plan.GetPlanInput) (*plan.Plan, error) {
	return &plan.Plan{}, nil
}

func (n NoopPlanService) UpdatePlan(ctx context.Context, params plan.UpdatePlanInput) (*plan.Plan, error) {
	return &plan.Plan{}, nil
}

func (n NoopPlanService) PublishPlan(ctx context.Context, params plan.PublishPlanInput) (*plan.Plan, error) {
	return &plan.Plan{}, nil
}

func (n NoopPlanService) ArchivePlan(ctx context.Context, params plan.ArchivePlanInput) (*plan.Plan, error) {
	return &plan.Plan{}, nil
}

func (n NoopPlanService) NextPlan(ctx context.Context, params plan.NextPlanInput) (*plan.Plan, error) {
	return &plan.Plan{}, nil
}

var _ addon.Service = (*NoopAddonService)(nil)

// NoopAddonService implements addon.Service interface with no-op operations
// for use in testing
type NoopAddonService struct{}

func (n NoopAddonService) ListAddons(ctx context.Context, params addon.ListAddonsInput) (pagination.PagedResponse[addon.Addon], error) {
	return pagination.PagedResponse[addon.Addon]{}, nil
}

func (n NoopAddonService) CreateAddon(ctx context.Context, params addon.CreateAddonInput) (*addon.Addon, error) {
	return &addon.Addon{}, nil
}

func (n NoopAddonService) DeleteAddon(ctx context.Context, params addon.DeleteAddonInput) error {
	return nil
}

func (n NoopAddonService) GetAddon(ctx context.Context, params addon.GetAddonInput) (*addon.Addon, error) {
	return &addon.Addon{}, nil
}

func (n NoopAddonService) UpdateAddon(ctx context.Context, params addon.UpdateAddonInput) (*addon.Addon, error) {
	return &addon.Addon{}, nil
}

func (n NoopAddonService) PublishAddon(ctx context.Context, params addon.PublishAddonInput) (*addon.Addon, error) {
	return &addon.Addon{}, nil
}

func (n NoopAddonService) ArchiveAddon(ctx context.Context, params addon.ArchiveAddonInput) (*addon.Addon, error) {
	return &addon.Addon{}, nil
}

func (n NoopAddonService) NextAddon(ctx context.Context, params addon.NextAddonInput) (*addon.Addon, error) {
	return &addon.Addon{}, nil
}

var _ plansubscription.PlanSubscriptionService = (*NoopPlanSubscriptionService)(nil)

// NoopPlanSubscriptionService implements plansubscription.PlanSubscriptionService with no-op operations
// for use in testing
type NoopPlanSubscriptionService struct{}

func (n NoopPlanSubscriptionService) Create(ctx context.Context, request plansubscription.CreateSubscriptionRequest) (subscription.Subscription, error) {
	return subscription.Subscription{}, nil
}

func (n NoopPlanSubscriptionService) Migrate(ctx context.Context, request plansubscription.MigrateSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
	return plansubscription.SubscriptionChangeResponse{
		Current: subscription.Subscription{},
		Next:    subscription.SubscriptionView{},
	}, nil
}

func (n NoopPlanSubscriptionService) Change(ctx context.Context, request plansubscription.ChangeSubscriptionRequest) (plansubscription.SubscriptionChangeResponse, error) {
	return plansubscription.SubscriptionChangeResponse{
		Current: subscription.Subscription{},
		Next:    subscription.SubscriptionView{},
	}, nil
}

var _ subscription.Service = (*NoopSubscriptionService)(nil)

// NoopSubscriptionService implements subscription.Service with no-op operations
// for use in testing
type NoopSubscriptionService struct{}

func (n NoopSubscriptionService) RegisterValidator(validator subscription.SubscriptionValidator) error {
	return nil
}

func (n NoopSubscriptionService) Create(ctx context.Context, namespace string, spec subscription.SubscriptionSpec) (subscription.Subscription, error) {
	return subscription.Subscription{}, nil
}

func (n NoopSubscriptionService) Update(ctx context.Context, subscriptionID models.NamespacedID, target subscription.SubscriptionSpec) (subscription.Subscription, error) {
	return subscription.Subscription{}, nil
}

func (n NoopSubscriptionService) Delete(ctx context.Context, subscriptionID models.NamespacedID) error {
	return nil
}

func (n NoopSubscriptionService) Cancel(ctx context.Context, subscriptionID models.NamespacedID, timing subscription.Timing) (subscription.Subscription, error) {
	return subscription.Subscription{}, nil
}

func (n NoopSubscriptionService) Continue(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	return subscription.Subscription{}, nil
}

func (n NoopSubscriptionService) Get(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	return subscription.Subscription{}, nil
}

func (n NoopSubscriptionService) GetView(ctx context.Context, subscriptionID models.NamespacedID) (subscription.SubscriptionView, error) {
	return subscription.SubscriptionView{}, nil
}

func (n NoopSubscriptionService) List(ctx context.Context, params subscription.ListSubscriptionsInput) (subscription.SubscriptionList, error) {
	return pagination.PagedResponse[subscription.Subscription]{}, nil
}

func (n NoopSubscriptionService) GetAllForCustomerSince(ctx context.Context, customerID models.NamespacedID, at time.Time) ([]subscription.Subscription, error) {
	return []subscription.Subscription{}, nil
}

var _ subscription.WorkflowService = (*NoopSubscriptionWorkflowService)(nil)

// NoopSubscriptionWorkflowService implements subscription.WorkflowService with no-op operations
// for use in testing
type NoopSubscriptionWorkflowService struct{}

func (n NoopSubscriptionWorkflowService) CreateFromPlan(ctx context.Context, inp subscription.CreateSubscriptionWorkflowInput, plan subscription.Plan) (subscription.SubscriptionView, error) {
	return subscription.SubscriptionView{}, nil
}

func (n NoopSubscriptionWorkflowService) EditRunning(ctx context.Context, subscriptionID models.NamespacedID, customizations []subscription.Patch, timing subscription.Timing) (subscription.SubscriptionView, error) {
	return subscription.SubscriptionView{}, nil
}

func (n NoopSubscriptionWorkflowService) ChangeToPlan(ctx context.Context, subscriptionID models.NamespacedID, inp subscription.ChangeSubscriptionWorkflowInput, plan subscription.Plan) (current subscription.Subscription, new subscription.SubscriptionView, err error) {
	return subscription.Subscription{}, subscription.SubscriptionView{}, nil
}

func (n NoopSubscriptionWorkflowService) Restore(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	return subscription.Subscription{}, nil
}

var _ grant.Repo = (*NoopGrantRepo)(nil)

// NoopGrantRepo implements grant.Repo with no-op operations
// for use in testing
type NoopGrantRepo struct{}

func (n NoopGrantRepo) CreateGrant(ctx context.Context, input grant.RepoCreateInput) (*grant.Grant, error) {
	return &grant.Grant{}, nil
}

func (n NoopGrantRepo) VoidGrant(ctx context.Context, grantID models.NamespacedID, at time.Time) error {
	return nil
}

func (n NoopGrantRepo) ListGrants(ctx context.Context, params grant.ListParams) (pagination.PagedResponse[grant.Grant], error) {
	return pagination.PagedResponse[grant.Grant]{}, nil
}

func (n NoopGrantRepo) ListActiveGrantsBetween(ctx context.Context, owner models.NamespacedID, from, to time.Time) ([]grant.Grant, error) {
	return []grant.Grant{}, nil
}

func (n NoopGrantRepo) GetGrant(ctx context.Context, grantID models.NamespacedID) (grant.Grant, error) {
	return grant.Grant{}, nil
}

// NoopTransactionDriver is a no-op implementation of transaction.Driver
type NoopTransactionDriver struct{}

func (d NoopTransactionDriver) Commit() error {
	return nil
}

func (d NoopTransactionDriver) Rollback() error {
	return nil
}

func (d NoopTransactionDriver) SavePoint() error {
	return nil
}

func (n NoopGrantRepo) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
	return ctx, NoopTransactionDriver{}, nil
}

func (n NoopGrantRepo) WithTx(ctx context.Context, tx *entutils.TxDriver) grant.Repo {
	return n
}

func (n NoopGrantRepo) Self() grant.Repo {
	return n
}

var _ billing.Service = (*NoopBillingService)(nil)

// NoopBillingService implements billing.Service with no-op operations
type NoopBillingService struct{}

// ProfileService methods
func (n NoopBillingService) CreateProfile(ctx context.Context, param billing.CreateProfileInput) (*billing.Profile, error) {
	return &billing.Profile{}, nil
}

func (n NoopBillingService) GetDefaultProfile(ctx context.Context, input billing.GetDefaultProfileInput) (*billing.Profile, error) {
	return &billing.Profile{}, nil
}

func (n NoopBillingService) GetProfile(ctx context.Context, input billing.GetProfileInput) (*billing.Profile, error) {
	return &billing.Profile{}, nil
}

func (n NoopBillingService) ListProfiles(ctx context.Context, input billing.ListProfilesInput) (billing.ListProfilesResult, error) {
	return billing.ListProfilesResult{}, nil
}

func (n NoopBillingService) DeleteProfile(ctx context.Context, input billing.DeleteProfileInput) error {
	return nil
}

func (n NoopBillingService) UpdateProfile(ctx context.Context, input billing.UpdateProfileInput) (*billing.Profile, error) {
	return &billing.Profile{}, nil
}

func (n NoopBillingService) ProvisionDefaultBillingProfile(ctx context.Context, namespace string) error {
	return nil
}

func (n NoopBillingService) IsAppUsed(ctx context.Context, appID app.AppID) error {
	return nil
}

// CustomerOverrideService methods
func (n NoopBillingService) UpsertCustomerOverride(ctx context.Context, input billing.UpsertCustomerOverrideInput) (billing.CustomerOverrideWithDetails, error) {
	return billing.CustomerOverrideWithDetails{}, nil
}

func (n NoopBillingService) DeleteCustomerOverride(ctx context.Context, input billing.DeleteCustomerOverrideInput) error {
	return nil
}

func (n NoopBillingService) GetCustomerOverride(ctx context.Context, input billing.GetCustomerOverrideInput) (billing.CustomerOverrideWithDetails, error) {
	return billing.CustomerOverrideWithDetails{}, nil
}

func (n NoopBillingService) ListCustomerOverrides(ctx context.Context, input billing.ListCustomerOverridesInput) (billing.ListCustomerOverridesResult, error) {
	return billing.ListCustomerOverridesResult{}, nil
}

// InvoiceLineService methods
func (n NoopBillingService) CreatePendingInvoiceLines(ctx context.Context, input billing.CreateInvoiceLinesInput) ([]*billing.Line, error) {
	return []*billing.Line{}, nil
}

func (n NoopBillingService) GetLinesForSubscription(ctx context.Context, input billing.GetLinesForSubscriptionInput) ([]*billing.Line, error) {
	return []*billing.Line{}, nil
}

func (n NoopBillingService) SnapshotLineQuantity(ctx context.Context, input billing.SnapshotLineQuantityInput) (*billing.Line, error) {
	return &billing.Line{}, nil
}

// InvoiceService methods
func (n NoopBillingService) ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	return billing.ListInvoicesResponse{}, nil
}

func (n NoopBillingService) GetInvoiceByID(ctx context.Context, input billing.GetInvoiceByIdInput) (billing.Invoice, error) {
	return billing.Invoice{}, nil
}

func (n NoopBillingService) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.Invoice, error) {
	return []billing.Invoice{}, nil
}

func (n NoopBillingService) AdvanceInvoice(ctx context.Context, input billing.AdvanceInvoiceInput) (billing.Invoice, error) {
	return billing.Invoice{}, nil
}

func (n NoopBillingService) ApproveInvoice(ctx context.Context, input billing.ApproveInvoiceInput) (billing.Invoice, error) {
	return billing.Invoice{}, nil
}

func (n NoopBillingService) RetryInvoice(ctx context.Context, input billing.RetryInvoiceInput) (billing.Invoice, error) {
	return billing.Invoice{}, nil
}

func (n NoopBillingService) DeleteInvoice(ctx context.Context, input billing.DeleteInvoiceInput) error {
	return nil
}

func (n NoopBillingService) UpdateInvoice(ctx context.Context, input billing.UpdateInvoiceInput) (billing.Invoice, error) {
	return billing.Invoice{}, nil
}

func (n NoopBillingService) SimulateInvoice(ctx context.Context, input billing.SimulateInvoiceInput) (billing.Invoice, error) {
	return billing.Invoice{}, nil
}

func (n NoopBillingService) UpsertValidationIssues(ctx context.Context, input billing.UpsertValidationIssuesInput) error {
	return nil
}

// SequenceService methods
func (n NoopBillingService) GenerateInvoiceSequenceNumber(ctx context.Context, in billing.SequenceGenerationInput, def billing.SequenceDefinition) (string, error) {
	return "", nil
}

// InvoiceAppService methods
func (n NoopBillingService) TriggerInvoice(ctx context.Context, input billing.InvoiceTriggerServiceInput) error {
	return nil
}

func (n NoopBillingService) UpdateInvoiceFields(ctx context.Context, input billing.UpdateInvoiceFieldsInput) error {
	return nil
}

// ConfigIntrospectionService methods
func (n NoopBillingService) GetAdvancementStrategy() billing.AdvancementStrategy {
	return billing.ForegroundAdvancementStrategy
}
