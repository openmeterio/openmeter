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
	"github.com/oklog/ulid/v2"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appcustominvoicing "github.com/openmeterio/openmeter/openmeter/app/custominvoicing"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/credit"
	"github.com/openmeterio/openmeter/openmeter/credit/engine"
	"github.com/openmeterio/openmeter/openmeter/credit/grant"
	"github.com/openmeterio/openmeter/openmeter/currencies"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	meteredentitlement "github.com/openmeterio/openmeter/openmeter/entitlement/metered"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/meter"
	meterhttphandler "github.com/openmeterio/openmeter/openmeter/meter/httphandler"
	meteradapter "github.com/openmeterio/openmeter/openmeter/meter/mockadapter"
	metereventadapter "github.com/openmeterio/openmeter/openmeter/meterevent/adapter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/notification"
	portaladapter "github.com/openmeterio/openmeter/openmeter/portal/adapter"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/addon"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/planaddon"
	plansubscription "github.com/openmeterio/openmeter/openmeter/productcatalog/subscription"
	progressmanageradapter "github.com/openmeterio/openmeter/openmeter/progressmanager/adapter"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	subscriptionaddon "github.com/openmeterio/openmeter/openmeter/subscription/addon"
	subscriptionworkflow "github.com/openmeterio/openmeter/openmeter/subscription/workflow"
	"github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
	"github.com/openmeterio/openmeter/openmeter/watermill/marshaler"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/framework/entutils"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/log"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
	"github.com/openmeterio/openmeter/pkg/ref"
	"github.com/openmeterio/openmeter/pkg/timeutil"
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
			name: "list events v1",
			req: testRequest{
				method:      http.MethodGet,
				path:        "/api/v1/events",
				contentType: "application/json",
			},
			res: testResponse{
				status: http.StatusOK,
				body: []api.IngestedEvent{
					{
						Event: mockEvent,
						// empty string customer id because it's not set on the mock
						CustomerId: lo.ToPtr(""),
						// empty string event type in error message because it's not set on the mock event
						ValidationError: lo.ToPtr("no meter found for event type: "),
					},
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
						"window_start,window_end,value",
						"2021-01-01T00:00:00Z,2021-01-01T01:00:00Z,300.000000",
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
		{
			name: "list meter group by values",
			req: testRequest{
				method: http.MethodGet,
				path:   fmt.Sprintf("/api/v1/meters/%s/group-by/%s/values", mockMeters[0].Key, "groupByKey"),
			},
			res: testResponse{
				status: http.StatusOK,
				body:   []string{"v1"},
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
	customerService := &NoopCustomerService{}

	meterManageService, err := meteradapter.NewManage(mockMeters)
	assert.NoError(t, err, "failed to create meter service")

	meterEventService := metereventadapter.New(mockStreamingConnector, customerService, meterManageService)

	logger := slog.New(log.NewMockHandler())

	// Create feature service
	featureService := &NoopFeatureConnector{}

	// Create app service
	appService := &NoopAppService{}
	appStripeService := &NoopAppStripeService{}
	appCustomInvoicingService := &NoopAppCustomInvoicingService{}

	// Create plan service
	planService := &NoopPlanService{}
	addonService := &NoopAddonService{}
	planAddonService := &NoopPlanAddonService{}

	// Create plan subscription service
	planSubscriptionService := &NoopPlanSubscriptionService{}

	// Create subscription services
	subscriptionService := &NoopSubscriptionService{}
	subscriptionWorkflowService := &NoopSubscriptionWorkflowService{}
	subscriptionAddonService := &NoopSubscriptionAddonService{}
	// Create grant repo
	grantRepo := &NoopGrantRepo{}

	// Create billing service
	billingService := &NoopBillingService{}

	// Create subject service
	subjectService := &NoopSubjectService{}

	config := &Config{
		RouterConfig: router.Config{
			Addon:                       addonService,
			App:                         appService,
			AppStripe:                   appStripeService,
			AppCustomInvoicing:          appCustomInvoicingService,
			Billing:                     billingService,
			Customer:                    customerService,
			CurrencyService:             &NoopCurrencyService{},
			DebugConnector:              MockDebugHandler{},
			EntitlementConnector:        &NoopEntitlementConnector{},
			EntitlementBalanceConnector: &NoopEntitlementBalanceConnector{},
			ErrorHandler:                errorsx.NopHandler{},
			FeatureConnector:            featureService,
			GrantConnector:              &NoopGrantConnector{},
			// Use the grant repo
			GrantRepo:          grantRepo,
			IngestService:      &NoopIngestService{},
			Logger:             logger,
			MeterManageService: meterManageService,
			MeterEventService:  meterEventService,
			NamespaceManager:   namespaceManager,
			Notification:       &NoopNotificationService{},
			// Use the plan service
			Plan:      planService,
			PlanAddon: planAddonService,
			// Use the plan subscription service
			PlanSubscriptionService: planSubscriptionService,
			Portal:                  portal,
			ProgressManager:         progressmanageradapter.NewNoop(),
			StreamingConnector:      mockStreamingConnector,
			// Use the subscription service
			SubscriptionService: subscriptionService,
			// Use the subscription workflow service
			SubscriptionWorkflowService: subscriptionWorkflowService,
			SubscriptionAddonService:    subscriptionAddonService,
			// Use the subject service
			SubjectService: subjectService,
		},
		RouterHooks: RouterHooks{},
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
var _ streaming.Connector = (*MockStreamingConnector)(nil)

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

func (c *MockStreamingConnector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]streaming.RawEvent, error) {
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

func (c *MockStreamingConnector) ListEventsV2(ctx context.Context, params streaming.ListEventsV2Params) ([]streaming.RawEvent, error) {
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

func (c *MockStreamingConnector) QueryMeter(ctx context.Context, namespace string, m meter.Meter, params streaming.QueryParams) ([]meter.MeterQueryRow, error) {
	value := mockQueryValue

	if params.FilterSubject == nil {
		value.Subject = nil
	}

	return []meter.MeterQueryRow{value}, nil
}

func (c *MockStreamingConnector) ListSubjects(ctx context.Context, params streaming.ListSubjectsParams) ([]string, error) {
	return []string{"s1"}, nil
}

func (c *MockStreamingConnector) ListGroupByValues(ctx context.Context, params streaming.ListGroupByValuesParams) ([]string, error) {
	return []string{"v1"}, nil
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

func (n NoopFeatureConnector) ListFeatures(ctx context.Context, params feature.ListFeaturesParams) (pagination.Result[feature.Feature], error) {
	return pagination.Result[feature.Feature]{}, nil
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

func (n NoopFeatureConnector) ResolveFeatureMeters(ctx context.Context, namespace string, featureKeys []string) (feature.FeatureMeters, error) {
	return feature.FeatureMeterCollection{}, nil
}

// NoopEntitlementConnector
var _ entitlement.Service = (*NoopEntitlementConnector)(nil)

type NoopEntitlementConnector struct{}

func (n NoopEntitlementConnector) RegisterHooks(hooks ...models.ServiceHook[entitlement.Entitlement]) {
}

func (n NoopEntitlementConnector) CreateEntitlement(ctx context.Context, input entitlement.CreateEntitlementInputs, grants []entitlement.CreateEntitlementGrantInputs) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) ScheduleEntitlement(ctx context.Context, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) OverrideEntitlement(ctx context.Context, subject string, entitlementIdOrFeatureKey string, input entitlement.CreateEntitlementInputs, grants []entitlement.CreateEntitlementGrantInputs) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) SupersedeEntitlement(ctx context.Context, entitlementId string, input entitlement.CreateEntitlementInputs) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) GetEntitlement(ctx context.Context, namespace string, id string) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) GetEntitlementWithCustomer(ctx context.Context, namespace string, id string) (*entitlement.EntitlementWithCustomer, error) {
	return &entitlement.EntitlementWithCustomer{}, nil
}

func (n NoopEntitlementConnector) DeleteEntitlement(ctx context.Context, namespace string, id string, at time.Time) error {
	return nil
}

func (n NoopEntitlementConnector) GetEntitlementValue(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (entitlement.EntitlementValue, error) {
	return nil, nil
}

func (n NoopEntitlementConnector) GetEntitlementsOfCustomer(ctx context.Context, namespace string, subjectKey string, at time.Time) ([]entitlement.Entitlement, error) {
	return []entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) ListEntitlements(ctx context.Context, params entitlement.ListEntitlementsParams) (pagination.Result[entitlement.Entitlement], error) {
	return pagination.Result[entitlement.Entitlement]{}, nil
}

func (n NoopEntitlementConnector) ListEntitlementsWithCustomer(ctx context.Context, params entitlement.ListEntitlementsParams) (entitlement.ListEntitlementsWithCustomerResult, error) {
	return entitlement.ListEntitlementsWithCustomerResult{}, nil
}

func (n NoopEntitlementConnector) GetEntitlementOfCustomerAt(ctx context.Context, namespace string, subjectKey string, idOrFeatureKey string, at time.Time) (*entitlement.Entitlement, error) {
	return &entitlement.Entitlement{}, nil
}

func (n NoopEntitlementConnector) GetAccess(ctx context.Context, namespace string, subjectKey string) (entitlement.Access, error) {
	return entitlement.Access{}, nil
}

// NoopEntitlementBalanceConnector

var _ meteredentitlement.Connector = (*NoopEntitlementBalanceConnector)(nil)

type NoopEntitlementBalanceConnector struct{}

func (n NoopEntitlementBalanceConnector) RegisterHooks(hooks ...models.ServiceHook[meteredentitlement.Entitlement]) {
}

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

func (n NoopEntitlementBalanceConnector) ListEntitlementGrants(ctx context.Context, namespace string, params meteredentitlement.ListEntitlementGrantsParams) (pagination.Result[meteredentitlement.EntitlementGrant], error) {
	return pagination.Result[meteredentitlement.EntitlementGrant]{}, nil
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

func (n NoopNotificationService) ResendEvent(_ context.Context, _ notification.ResendEventInput) error {
	return nil
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

func (n NoopAppService) ListMarketplaceListings(ctx context.Context, input app.MarketplaceListInput) (pagination.Result[app.RegistryItem], error) {
	return pagination.Result[app.RegistryItem]{}, nil
}

func (n NoopAppService) InstallMarketplaceListingWithAPIKey(ctx context.Context, input app.InstallAppWithAPIKeyInput) (app.App, error) {
	return appstripeentityapp.App{}, nil
}

func (n NoopAppService) InstallMarketplaceListing(ctx context.Context, input app.InstallAppInput) (app.App, error) {
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

func (n NoopAppService) UpdateApp(ctx context.Context, input app.UpdateAppInput) (app.App, error) {
	return appstripeentityapp.App{}, nil
}

func (n NoopAppService) ListApps(ctx context.Context, input app.ListAppInput) (pagination.Result[app.App], error) {
	return pagination.Result[app.App]{}, nil
}

func (n NoopAppService) UninstallApp(ctx context.Context, input app.UninstallAppInput) error {
	return nil
}

func (n NoopAppService) ListCustomerData(ctx context.Context, input app.ListCustomerInput) (pagination.Result[app.CustomerApp], error) {
	return pagination.Result[app.CustomerApp]{}, nil
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

func (n NoopAppStripeService) CreatePortalSession(ctx context.Context, input appstripeentity.CreateStripePortalSessionInput) (appstripeentity.StripePortalSession, error) {
	return appstripeentity.StripePortalSession{}, nil
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

var _ appcustominvoicing.SyncService = (*NoopAppCustomInvoicingService)(nil)

type NoopAppCustomInvoicingService struct{}

func (n NoopAppCustomInvoicingService) SyncDraftInvoice(ctx context.Context, input appcustominvoicing.SyncDraftInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopAppCustomInvoicingService) SyncIssuingInvoice(ctx context.Context, input appcustominvoicing.SyncIssuingInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopAppCustomInvoicingService) HandlePaymentTrigger(ctx context.Context, input appcustominvoicing.HandlePaymentTriggerInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

var _ customer.Service = (*NoopCustomerService)(nil)

type NoopCustomerService struct{}

func (n NoopCustomerService) RegisterHooks(_ ...models.ServiceHook[customer.Customer]) {}

func (n NoopCustomerService) ListCustomers(ctx context.Context, params customer.ListCustomersInput) (pagination.Result[customer.Customer], error) {
	return pagination.Result[customer.Customer]{}, nil
}

func (n NoopCustomerService) ListCustomerUsageAttributions(ctx context.Context, input customer.ListCustomerUsageAttributionsInput) (pagination.Result[streaming.CustomerUsageAttribution], error) {
	return pagination.Result[streaming.CustomerUsageAttribution]{}, nil
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

func (n NoopCustomerService) GetCustomerByUsageAttribution(ctx context.Context, input customer.GetCustomerByUsageAttributionInput) (*customer.Customer, error) {
	return &customer.Customer{}, nil
}

func (n NoopCustomerService) UpdateCustomer(ctx context.Context, params customer.UpdateCustomerInput) (*customer.Customer, error) {
	return &customer.Customer{}, nil
}

func (n NoopCustomerService) GetEntitlementValue(ctx context.Context, input customer.GetEntitlementValueInput) (entitlement.EntitlementValue, error) {
	return nil, nil
}

func (n NoopCustomerService) GetCustomerAccess(ctx context.Context, input customer.GetCustomerInput) (entitlement.Access, error) {
	return entitlement.Access{}, nil
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

func (n NoopPlanService) ListPlans(ctx context.Context, params plan.ListPlansInput) (pagination.Result[plan.Plan], error) {
	return pagination.Result[plan.Plan]{}, nil
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

func (n NoopAddonService) ListAddons(ctx context.Context, params addon.ListAddonsInput) (pagination.Result[addon.Addon], error) {
	return pagination.Result[addon.Addon]{}, nil
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

var _ planaddon.Service = (*NoopPlanAddonService)(nil)

// NoopAddonService implements planaddon.Service interface with no-op operations
// for use in testing
type NoopPlanAddonService struct{}

func (n NoopPlanAddonService) ListPlanAddons(ctx context.Context, params planaddon.ListPlanAddonsInput) (pagination.Result[planaddon.PlanAddon], error) {
	return pagination.Result[planaddon.PlanAddon]{}, nil
}

func (n NoopPlanAddonService) CreatePlanAddon(ctx context.Context, params planaddon.CreatePlanAddonInput) (*planaddon.PlanAddon, error) {
	return &planaddon.PlanAddon{}, nil
}

func (n NoopPlanAddonService) DeletePlanAddon(ctx context.Context, params planaddon.DeletePlanAddonInput) error {
	return nil
}

func (n NoopPlanAddonService) GetPlanAddon(ctx context.Context, params planaddon.GetPlanAddonInput) (*planaddon.PlanAddon, error) {
	return &planaddon.PlanAddon{}, nil
}

func (n NoopPlanAddonService) UpdatePlanAddon(ctx context.Context, params planaddon.UpdatePlanAddonInput) (*planaddon.PlanAddon, error) {
	return &planaddon.PlanAddon{}, nil
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

func (n NoopSubscriptionService) UpdateAnnotations(ctx context.Context, subscriptionID models.NamespacedID, annotations models.Annotations) (*subscription.Subscription, error) {
	return nil, nil
}

func (n NoopSubscriptionService) RegisterHook(validator subscription.SubscriptionCommandHook) error {
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
	return pagination.Result[subscription.Subscription]{}, nil
}

func (n NoopSubscriptionService) GetAllForCustomer(ctx context.Context, customerID models.NamespacedID, period timeutil.StartBoundedPeriod) ([]subscription.Subscription, error) {
	return []subscription.Subscription{}, nil
}

func (n NoopSubscriptionService) ExpandViews(ctx context.Context, subs []subscription.Subscription) ([]subscription.SubscriptionView, error) {
	return []subscription.SubscriptionView{}, nil
}

var _ subscriptionworkflow.Service = (*NoopSubscriptionWorkflowService)(nil)

// NoopSubscriptionWorkflowService implements subscriptionworkflow.Service with no-op operations
// for use in testing
type NoopSubscriptionWorkflowService struct{}

func (n NoopSubscriptionWorkflowService) CreateFromPlan(ctx context.Context, inp subscriptionworkflow.CreateSubscriptionWorkflowInput, plan subscription.Plan) (subscription.SubscriptionView, error) {
	return subscription.SubscriptionView{}, nil
}

func (n NoopSubscriptionWorkflowService) EditRunning(ctx context.Context, subscriptionID models.NamespacedID, customizations []subscription.Patch, timing subscription.Timing) (subscription.SubscriptionView, error) {
	return subscription.SubscriptionView{}, nil
}

func (n NoopSubscriptionWorkflowService) ChangeToPlan(ctx context.Context, subscriptionID models.NamespacedID, inp subscriptionworkflow.ChangeSubscriptionWorkflowInput, plan subscription.Plan) (current subscription.Subscription, new subscription.SubscriptionView, err error) {
	return subscription.Subscription{}, subscription.SubscriptionView{}, nil
}

func (n NoopSubscriptionWorkflowService) Restore(ctx context.Context, subscriptionID models.NamespacedID) (subscription.Subscription, error) {
	return subscription.Subscription{}, nil
}

func (n NoopSubscriptionWorkflowService) AddAddon(ctx context.Context, subscriptionID models.NamespacedID, inp subscriptionworkflow.AddAddonWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error) {
	return subscription.SubscriptionView{}, subscriptionaddon.SubscriptionAddon{}, nil
}

func (n NoopSubscriptionWorkflowService) ChangeAddonQuantity(ctx context.Context, subscriptionID models.NamespacedID, inp subscriptionworkflow.ChangeAddonQuantityWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error) {
	return subscription.SubscriptionView{}, subscriptionaddon.SubscriptionAddon{}, nil
}

var _ subscriptionaddon.Service = (*NoopSubscriptionAddonService)(nil)

// NoopSubscriptionAddonService implements subscriptionaddon.Service with no-op operations
// for use in testing
type NoopSubscriptionAddonService struct{}

func (n NoopSubscriptionAddonService) ChangeQuantity(ctx context.Context, subscriptionID models.NamespacedID, input subscriptionaddon.CreateSubscriptionAddonQuantityInput) (*subscriptionaddon.SubscriptionAddon, error) {
	return nil, nil
}

func (n NoopSubscriptionAddonService) Get(ctx context.Context, addonId models.NamespacedID) (*subscriptionaddon.SubscriptionAddon, error) {
	return nil, nil
}

func (n NoopSubscriptionAddonService) Create(ctx context.Context, ns string, input subscriptionaddon.CreateSubscriptionAddonInput) (*subscriptionaddon.SubscriptionAddon, error) {
	return nil, nil
}

func (n NoopSubscriptionAddonService) List(ctx context.Context, ns string, input subscriptionaddon.ListSubscriptionAddonsInput) (pagination.Result[subscriptionaddon.SubscriptionAddon], error) {
	return pagination.Result[subscriptionaddon.SubscriptionAddon]{}, nil
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

func (n NoopGrantRepo) DeleteOwnerGrants(ctx context.Context, ownerID models.NamespacedID) error {
	return nil
}

func (n NoopGrantRepo) ListGrants(ctx context.Context, params grant.ListParams) (pagination.Result[grant.Grant], error) {
	return pagination.Result[grant.Grant]{}, nil
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

func (n NoopBillingService) ResolveStripeAppIDFromBillingProfile(ctx context.Context, namespace string, customerId *customer.CustomerID) (app.AppID, error) {
	return app.AppID{}, nil
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

func (n NoopBillingService) GetCustomerApp(ctx context.Context, input billing.GetCustomerAppInput) (app.App, error) {
	return nil, nil
}

func (n NoopBillingService) ListCustomerOverrides(ctx context.Context, input billing.ListCustomerOverridesInput) (billing.ListCustomerOverridesResult, error) {
	return billing.ListCustomerOverridesResult{}, nil
}

// InvoiceLineService methods
func (n NoopBillingService) CreatePendingInvoiceLines(ctx context.Context, input billing.CreatePendingInvoiceLinesInput) (*billing.CreatePendingInvoiceLinesResult, error) {
	return nil, nil
}

func (n NoopBillingService) GetLinesForSubscription(ctx context.Context, input billing.GetLinesForSubscriptionInput) ([]billing.LineOrHierarchy, error) {
	return []billing.LineOrHierarchy{}, nil
}

func (n NoopBillingService) SnapshotLineQuantity(ctx context.Context, input billing.SnapshotLineQuantityInput) (*billing.StandardLine, error) {
	return &billing.StandardLine{}, nil
}

// InvoiceSplitLineGroupService methods
func (n NoopBillingService) DeleteSplitLineGroup(ctx context.Context, input billing.DeleteSplitLineGroupInput) error {
	return nil
}

func (n NoopBillingService) UpdateSplitLineGroup(ctx context.Context, input billing.UpdateSplitLineGroupInput) (billing.SplitLineGroup, error) {
	return billing.SplitLineGroup{}, nil
}

func (n NoopBillingService) GetSplitLineGroup(ctx context.Context, input billing.GetSplitLineGroupInput) (billing.SplitLineHierarchy, error) {
	return billing.SplitLineHierarchy{}, nil
}

// InvoiceService methods
func (n NoopBillingService) ListInvoices(ctx context.Context, input billing.ListInvoicesInput) (billing.ListInvoicesResponse, error) {
	return billing.ListInvoicesResponse{}, nil
}

func (n NoopBillingService) ListStandardInvoices(ctx context.Context, input billing.ListStandardInvoicesInput) (billing.ListStandardInvoicesResponse, error) {
	return billing.ListStandardInvoicesResponse{}, nil
}

func (n NoopBillingService) CreateStandardInvoiceFromGatheringLines(ctx context.Context, input billing.CreateStandardInvoiceFromGatheringLinesInput) (*billing.StandardInvoice, error) {
	return &billing.StandardInvoice{}, nil
}

func (n NoopBillingService) GetInvoiceById(ctx context.Context, input billing.GetInvoiceByIdInput) (billing.Invoice, error) {
	return billing.Invoice{}, nil
}

func (n NoopBillingService) GetStandardInvoiceById(ctx context.Context, input billing.GetStandardInvoiceByIdInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopBillingService) AdvanceInvoice(ctx context.Context, input billing.AdvanceInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopBillingService) SnapshotQuantities(ctx context.Context, input billing.SnapshotQuantitiesInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopBillingService) ApproveInvoice(ctx context.Context, input billing.ApproveInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopBillingService) RetryInvoice(ctx context.Context, input billing.RetryInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopBillingService) DeleteInvoice(ctx context.Context, input billing.DeleteInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopBillingService) UpdateInvoice(ctx context.Context, input billing.UpdateInvoiceInput) (billing.Invoice, error) {
	return billing.Invoice{}, nil
}

func (n NoopBillingService) UpdateStandardInvoice(ctx context.Context, input billing.UpdateStandardInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopBillingService) SimulateInvoice(ctx context.Context, input billing.SimulateInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopBillingService) UpsertValidationIssues(ctx context.Context, input billing.UpsertValidationIssuesInput) error {
	return nil
}

func (n NoopBillingService) RecalculateGatheringInvoices(ctx context.Context, input billing.RecalculateGatheringInvoicesInput) error {
	return nil
}

// GatheringInvoiceService methods
func (n NoopBillingService) InvoicePendingLines(ctx context.Context, input billing.InvoicePendingLinesInput) ([]billing.StandardInvoice, error) {
	return []billing.StandardInvoice{}, nil
}

func (n NoopBillingService) PrepareBillableLines(ctx context.Context, input billing.PrepareBillableLinesInput) (*billing.PrepareBillableLinesResult, error) {
	return nil, nil
}

func (n NoopBillingService) ListGatheringInvoices(ctx context.Context, input billing.ListGatheringInvoicesInput) (pagination.Result[billing.GatheringInvoice], error) {
	return pagination.Result[billing.GatheringInvoice]{}, nil
}

func (n NoopBillingService) UpdateGatheringInvoice(ctx context.Context, input billing.UpdateGatheringInvoiceInput) error {
	return nil
}

func (n NoopBillingService) GetGatheringInvoiceById(ctx context.Context, input billing.GetGatheringInvoiceByIdInput) (billing.GatheringInvoice, error) {
	return billing.GatheringInvoice{}, nil
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

func (n NoopBillingService) SyncDraftInvoice(ctx context.Context, input billing.SyncDraftStandardInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

func (n NoopBillingService) SyncIssuingInvoice(ctx context.Context, input billing.SyncIssuingStandardInvoiceInput) (billing.StandardInvoice, error) {
	return billing.StandardInvoice{}, nil
}

// ConfigIntrospectionService methods
func (n NoopBillingService) GetAdvancementStrategy() billing.AdvancementStrategy {
	return billing.ForegroundAdvancementStrategy
}

func (n NoopBillingService) WithAdvancementStrategy(strategy billing.AdvancementStrategy) billing.Service {
	return n
}

func (n NoopBillingService) WithLockedNamespaces(namespaces []string) billing.Service {
	return n
}

// LockableService methods
func (n NoopBillingService) WithLock(ctx context.Context, customerID customer.CustomerID, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// SubjectService methods

var _ subject.Service = &NoopSubjectService{}

type NoopSubjectService struct{}

func (n NoopSubjectService) RegisterHooks(_ ...models.ServiceHook[subject.Subject]) {}

func (n NoopSubjectService) Create(ctx context.Context, input subject.CreateInput) (subject.Subject, error) {
	return subject.Subject{}, nil
}

func (n NoopSubjectService) Update(ctx context.Context, input subject.UpdateInput) (subject.Subject, error) {
	return subject.Subject{}, nil
}

func (n NoopSubjectService) GetByIdOrKey(ctx context.Context, orgId string, idOrKey string) (subject.Subject, error) {
	return subject.Subject{}, nil
}

func (n NoopSubjectService) GetById(ctx context.Context, id models.NamespacedID) (subject.Subject, error) {
	return subject.Subject{}, nil
}

func (n NoopSubjectService) GetByKey(ctx context.Context, key models.NamespacedKey) (subject.Subject, error) {
	return subject.Subject{}, nil
}

func (n NoopSubjectService) List(ctx context.Context, orgId string, params subject.ListParams) (pagination.Result[subject.Subject], error) {
	return pagination.Result[subject.Subject]{}, nil
}

func (n NoopSubjectService) Delete(ctx context.Context, id models.NamespacedID) error {
	return nil
}

// IngestService methods
var _ ingest.Service = (*NoopIngestService)(nil)

// NoopIngestService implements ingest.Service with no-op operations
// for use in testing
type NoopIngestService struct{}

func (n NoopIngestService) IngestEvents(ctx context.Context, request ingest.IngestEventsRequest) (bool, error) {
	return true, nil
}

// NoopCurrencyService implements currencies.CurrencyService with no-op operations
// for use in testing
var _ currencies.CurrencyService = (*NoopCurrencyService)(nil)

type NoopCurrencyService struct{}

func (n NoopCurrencyService) ListCurrencies(ctx context.Context, params currencies.ListCurrenciesInput) ([]currencies.Currency, int, error) {
	return []currencies.Currency{}, 0, nil
}

func (n NoopCurrencyService) CreateCurrency(ctx context.Context, params currencies.CreateCurrencyInput) (currencies.Currency, error) {
	return currencies.Currency{}, nil
}

func (n NoopCurrencyService) CreateCostBasis(ctx context.Context, params currencies.CreateCostBasisInput) (*currencies.CostBasis, error) {
	return nil, nil
}

func (n NoopCurrencyService) ListCostBases(ctx context.Context, params currencies.ListCostBasesInput) ([]currencies.CostBasis, int, error) {
	return []currencies.CostBasis{}, 0, nil
}
