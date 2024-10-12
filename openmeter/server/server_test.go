package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/openmeter/app"
	appentity "github.com/openmeterio/openmeter/openmeter/app/entity"
	appentitybase "github.com/openmeterio/openmeter/openmeter/app/entity/base"
	appobserver "github.com/openmeterio/openmeter/openmeter/app/observer"
	appstripe "github.com/openmeterio/openmeter/openmeter/app/stripe"
	appstripeentity "github.com/openmeterio/openmeter/openmeter/app/stripe/entity"
	appstripeentityapp "github.com/openmeterio/openmeter/openmeter/app/stripe/entity/app"
	"github.com/openmeterio/openmeter/openmeter/customer"
	customerentity "github.com/openmeterio/openmeter/openmeter/customer/entity"
	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/ingestdriver"
	"github.com/openmeterio/openmeter/openmeter/meter"
	"github.com/openmeterio/openmeter/openmeter/namespace"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/openmeter/notification"
	"github.com/openmeterio/openmeter/openmeter/productcatalog/feature"
	"github.com/openmeterio/openmeter/openmeter/server/authenticator"
	"github.com/openmeterio/openmeter/openmeter/server/router"
	"github.com/openmeterio/openmeter/openmeter/streaming"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
	"github.com/openmeterio/openmeter/pkg/pagination"
)

var mockEvent = event.New()

var mockMeters = []models.Meter{
	{
		ID:            ulid.Make().String(),
		Slug:          "meter1",
		WindowSize:    models.WindowSizeMinute,
		Aggregation:   models.MeterAggregationSum,
		EventType:     "event",
		ValueProperty: "$.value",
		GroupBy:       map[string]string{"path": "$.path", "method": "$.method"},
	},
	{
		ID:            ulid.Make().String(),
		Slug:          "meter2",
		WindowSize:    models.WindowSizeMinute,
		Aggregation:   models.MeterAggregationSum,
		EventType:     "event",
		ValueProperty: "$.value",
	},
}

var (
	mockSubject    = "s1"
	mockQueryValue = models.MeterQueryRow{
		Subject:     &mockSubject,
		WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		WindowEnd:   time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
		Value:       300,
	}
)

type MockStreamingConnector struct{}

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

func (c *MockStreamingConnector) CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error {
	return nil
}

func (c *MockStreamingConnector) DeleteMeter(ctx context.Context, namespace string, meterSlug string) error {
	return nil
}

func (c *MockStreamingConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]models.MeterQueryRow, error) {
	value := mockQueryValue

	if params.FilterSubject == nil {
		value.Subject = nil
	}

	return []models.MeterQueryRow{value}, nil
}

func (c *MockStreamingConnector) ListMeterSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error) {
	return []string{"s1"}, nil
}

type MockDebugHandler struct{}

func (h MockDebugHandler) GetDebugMetrics(ctx context.Context, namespace string) (string, error) {
	return `openmeter_events_total{subject="customer-1",error="true"} 2.0`, nil
}

type MockHandler struct{}

func (h MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, namespace string) {
	w.WriteHeader(http.StatusOK)
}

func makeRequest(r *http.Request) (*httptest.ResponseRecorder, error) {
	namespaceManager, err := namespace.NewManager(namespace.ManagerConfig{
		DefaultNamespace: "test",
	})
	if err != nil {
		return nil, err
	}

	portalTokenStrategy, err := authenticator.NewPortalTokenStrategy("12345", time.Hour)
	if err != nil {
		return nil, err
	}

	server, _ := NewServer(&Config{
		RouterConfig: router.Config{
			Meters:             meter.NewInMemoryRepository(mockMeters),
			StreamingConnector: &MockStreamingConnector{},
			DebugConnector:     MockDebugHandler{},
			IngestHandler: ingestdriver.NewIngestEventsHandler(func(ctx context.Context, request ingest.IngestEventsRequest) (bool, error) {
				return true, nil
			}, namespacedriver.StaticNamespaceDecoder("test"), nil, errorsx.NewContextHandler(errorsx.NopHandler{})),
			NamespaceManager:    namespaceManager,
			PortalTokenStrategy: portalTokenStrategy,
			ErrorHandler:        errorsx.NopHandler{},
			Notification:        &NoopNotificationService{},
			App:                 &NoopAppService{},
			AppStripe:           &NoopAppStripeService{},
			Customer:            &NoopCustomerService{},
		},
		RouterHook: func(r chi.Router) {},
	})
	writer := httptest.NewRecorder()
	server.ServeHTTP(writer, r)
	return writer, nil
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
				body:   mockMeters,
			},
		},
		{
			name: "create meter",
			req: testRequest{
				method:      http.MethodPost,
				path:        "/api/v1/meters",
				contentType: "application/json",
				body: &models.Meter{
					Slug:          "meter3",
					Description:   "API Network Traffic",
					ValueProperty: "$.bytes",
					EventType:     "api-calls",
					Aggregation:   models.MeterAggregationSum,
					GroupBy:       map[string]string{"path": "$.path", "method": "$.method"},
					WindowSize:    models.WindowSizeHour,
				},
			},
			res: testResponse{
				status: http.StatusNotImplemented,
			},
		},
		{
			name: "get meter",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/meters/" + mockMeters[0].Slug,
			},
			res: testResponse{
				status: http.StatusOK,
				body:   mockMeters[0],
			},
		},
		{
			name: "delete meter",
			req: testRequest{
				method: http.MethodDelete,
				path:   "/api/v1/meters/" + mockMeters[0].Slug,
			},
			res: testResponse{
				status: http.StatusNotImplemented,
			},
		},
		{
			name: "query meter",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Slug + "/query",
			},
			res: testResponse{
				status: http.StatusOK,
				body: struct {
					Data []models.MeterQueryRow `json:"data"`
				}{
					Data: []models.MeterQueryRow{
						{Subject: nil, WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC), Value: 300},
					},
				},
			},
		},
		{
			name: "query meter with valid group by",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Slug + "/query?groupBy=path&groupBy=method",
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
				path:        "/api/v1/meters/" + mockMeters[0].Slug + "/query?groupBy=subject",
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
				path:        "/api/v1/meters/" + mockMeters[0].Slug + "/query?groupBy=foo",
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
				path:        "/api/v1/meters/" + mockMeters[0].Slug + "/query?subject=s1",
			},
			res: testResponse{
				status: http.StatusOK,
				body: struct {
					Data []models.MeterQueryRow `json:"data"`
				}{
					Data: []models.MeterQueryRow{
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
				path:        "/api/v1/meters/" + mockMeters[0].Slug + "/query?filterGroupBy[method]=GET",
			},
			res: testResponse{
				status: http.StatusOK,
				body: struct {
					Data []models.MeterQueryRow `json:"data"`
				}{
					Data: []models.MeterQueryRow{
						{Subject: nil, WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC), Value: 300},
					},
				},
			},
		},
		{
			name: "query meter with invalid group by filter",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].Slug + "/query?filterGroupBy[invalid]=abcd",
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
				path:        "/api/v1/meters/" + mockMeters[0].Slug + "/query",
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
				path:        "/api/v1/meters/" + mockMeters[0].Slug + "/query?subject=s1",
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
				path:   fmt.Sprintf("/api/v1/meters/%s/subjects", mockMeters[0].Slug),
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
			name: "invalidate portal token",
			req: testRequest{
				method:      http.MethodPost,
				path:        "/api/v1/portal/tokens/invalidate",
				contentType: "application/json",
				body:        api.InvalidatePortalTokensJSONRequestBody{},
			},
			res: testResponse{
				status: http.StatusNotImplemented,
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
		// Subjects
		{
			name: "upsert subjects",
			req: testRequest{
				method:      http.MethodPost,
				path:        "/api/v1/subjects",
				contentType: "application/json",
				body: api.UpsertSubjectJSONRequestBody{{
					Key: "customer",
				}},
			},
			res: testResponse{
				status: http.StatusNotImplemented,
			},
		},
		{
			name: "get subject",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/subjects/abcd",
			},
			res: testResponse{
				status: http.StatusNotImplemented,
			},
		},
		{
			name: "list subjects",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/subjects",
			},
			res: testResponse{
				status: http.StatusNotImplemented,
			},
		},
		{
			name: "delete subject",
			req: testRequest{
				method: http.MethodDelete,
				path:   "/api/v1/subjects/abcd",
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
				body:        `openmeter_events_total{subject="customer-1",error="true"} 2.0`,
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
			w, err := makeRequest(req)
			assert.NoError(t, err)
			res := w.Result()

			defer res.Body.Close()

			// status
			assert.Equal(t, tt.res.status, res.StatusCode, w.Body.String())

			// body
			if tt.res.body == nil {
				return
			}

			switch tt.req.accept {
			case "text/csv":
				assert.Equal(t, tt.res.body, w.Body.String())
			default:
				// Handle default as "application/json"
				resBody, _ := json.Marshal(tt.res.body)
				assert.JSONEq(t, string(resBody), w.Body.String())
			}
		})
	}
}

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

func (n NoopAppService) RegisterMarketplaceListing(input appentity.RegisterMarketplaceListingInput) error {
	return nil
}

func (n NoopAppService) GetMarketplaceListing(ctx context.Context, input appentity.MarketplaceGetInput) (appentity.RegistryItem, error) {
	return appentity.RegistryItem{}, nil
}

func (n NoopAppService) ListMarketplaceListings(ctx context.Context, input appentity.MarketplaceListInput) (pagination.PagedResponse[appentity.RegistryItem], error) {
	return pagination.PagedResponse[appentity.RegistryItem]{}, nil
}

func (n NoopAppService) InstallMarketplaceListingWithAPIKey(ctx context.Context, input appentity.InstallAppWithAPIKeyInput) (appentity.App, error) {
	return appstripeentityapp.App{}, nil
}

func (n NoopAppService) GetMarketplaceListingOauth2InstallURL(ctx context.Context, input appentity.GetOauth2InstallURLInput) (appentity.GetOauth2InstallURLOutput, error) {
	return appentity.GetOauth2InstallURLOutput{}, nil
}

func (n NoopAppService) AuthorizeMarketplaceListingOauth2Install(ctx context.Context, input appentity.AuthorizeOauth2InstallInput) error {
	return nil
}

func (n NoopAppService) CreateApp(ctx context.Context, input appentity.CreateAppInput) (appentitybase.AppBase, error) {
	return appentitybase.AppBase{}, nil
}

func (n NoopAppService) GetApp(ctx context.Context, input appentity.GetAppInput) (appentity.App, error) {
	return appstripeentityapp.App{}, nil
}

func (n NoopAppService) GetDefaultApp(ctx context.Context, input appentity.GetDefaultAppInput) (appentity.App, error) {
	return appstripeentityapp.App{}, nil
}

func (n NoopAppService) ListApps(ctx context.Context, input appentity.ListAppInput) (pagination.PagedResponse[appentity.App], error) {
	return pagination.PagedResponse[appentity.App]{}, nil
}

func (n NoopAppService) UninstallApp(ctx context.Context, input appentity.DeleteAppInput) error {
	return nil
}

var _ appstripe.Service = (*NoopAppStripeService)(nil)

type NoopAppStripeService struct{}

func (n NoopAppStripeService) CreateCheckoutSession(ctx context.Context, input appstripeentity.CreateCheckoutSessionInput) (appstripeentity.CreateCheckoutSessionOutput, error) {
	return appstripeentity.CreateCheckoutSessionOutput{}, nil
}

func (n NoopAppStripeService) GetWebhookSecret(ctx context.Context, input appstripeentity.GetWebhookSecretInput) (appstripeentity.GetWebhookSecretOutput, error) {
	return appstripeentity.GetWebhookSecretOutput{}, nil
}

func (n NoopAppStripeService) GetStripeAppData(ctx context.Context, input appstripeentity.GetStripeAppDataInput) (appstripeentity.AppData, error) {
	return appstripeentity.AppData{}, nil
}

func (n NoopAppStripeService) SetCustomerDefaultPaymentMethod(ctx context.Context, input appstripeentity.SetCustomerDefaultPaymentMethodInput) (appstripeentity.SetCustomerDefaultPaymentMethodOutput, error) {
	return appstripeentity.SetCustomerDefaultPaymentMethodOutput{}, nil
}

func (n NoopAppStripeService) GetStripeCustomerData(ctx context.Context, input appstripeentity.GetStripeCustomerDataInput) (appstripeentity.CustomerAppData, error) {
	return appstripeentity.CustomerAppData{}, nil
}

func (n NoopAppStripeService) UpsertStripeCustomerData(ctx context.Context, input appstripeentity.UpsertStripeCustomerDataInput) error {
	return nil
}

func (n NoopAppStripeService) DeleteStripeCustomerData(ctx context.Context, input appstripeentity.DeleteStripeCustomerDataInput) error {
	return nil
}

var _ customer.Service = (*NoopCustomerService)(nil)

type NoopCustomerService struct{}

func (n NoopCustomerService) Register(observer appobserver.Observer[customerentity.Customer]) error {
	return nil
}

func (n NoopCustomerService) Deregister(observer appobserver.Observer[customerentity.Customer]) error {
	return nil
}

func (n NoopCustomerService) ListCustomers(ctx context.Context, params customerentity.ListCustomersInput) (pagination.PagedResponse[customerentity.Customer], error) {
	return pagination.PagedResponse[customerentity.Customer]{}, nil
}

func (n NoopCustomerService) CreateCustomer(ctx context.Context, params customerentity.CreateCustomerInput) (*customerentity.Customer, error) {
	return &customerentity.Customer{}, nil
}

func (n NoopCustomerService) DeleteCustomer(ctx context.Context, customer customerentity.DeleteCustomerInput) error {
	return nil
}

func (n NoopCustomerService) GetCustomer(ctx context.Context, customer customerentity.GetCustomerInput) (*customerentity.Customer, error) {
	return &customerentity.Customer{}, nil
}

func (n NoopCustomerService) UpdateCustomer(ctx context.Context, params customerentity.UpdateCustomerInput) (*customerentity.Customer, error) {
	return &customerentity.Customer{}, nil
}

func (n NoopCustomerService) UpsertAppCustomer(ctx context.Context, input customerentity.UpsertAppCustomerInput) error {
	return nil
}
