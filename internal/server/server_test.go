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
	"github.com/openmeterio/openmeter/internal/meter"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/internal/server/router"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

var mockEvent = event.New()

var mockMeters = []models.Meter{
	{ID: ulid.Make().String(), Slug: "meter1", WindowSize: models.WindowSizeMinute, Aggregation: models.MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
	{ID: ulid.Make().String(), Slug: "meter2", WindowSize: models.WindowSizeMinute, Aggregation: models.MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
}

var mockSubject = "s1"
var mockQueryValue = models.MeterQueryRow{
	Subject:     &mockSubject,
	WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	WindowEnd:   time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC),
	Value:       300,
}

type MockConnector struct{}

func (c *MockConnector) ListEvents(ctx context.Context, namespace string, params streaming.ListEventsParams) ([]api.IngestedEvent, error) {
	events := []api.IngestedEvent{
		{
			Event: mockEvent,
		},
	}
	return events, nil
}

func (c *MockConnector) CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error {
	return nil
}

func (c *MockConnector) DeleteMeter(ctx context.Context, namespace string, meterSlug string) error {
	return nil
}

func (c *MockConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]models.MeterQueryRow, error) {
	value := mockQueryValue

	if params.Subject == nil {
		value.Subject = nil
	}

	return []models.MeterQueryRow{value}, nil
}

func (c *MockConnector) ListMeterSubjects(ctx context.Context, namespace string, meterSlug string, from *time.Time, to *time.Time) ([]string, error) {
	return []string{"s1"}, nil
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

	server, _ := NewServer(&Config{
		RouterConfig: router.Config{
			Meters:             meter.NewInMemoryRepository(mockMeters),
			StreamingConnector: &MockConnector{},
			IngestHandler:      MockHandler{},
			NamespaceManager:   namespaceManager,
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
				status: http.StatusOK,
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
				path:   "/api/v1/meters/" + mockMeters[0].ID,
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
				path:        "/api/v1/meters/" + mockMeters[0].ID + "/query",
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
			name: "query meter with subject",
			req: testRequest{
				method:      http.MethodGet,
				contentType: "application/json",
				path:        "/api/v1/meters/" + mockMeters[0].ID + "/query?subject=s1",
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
			name: "query meter as csv",
			req: testRequest{
				accept:      "text/csv",
				contentType: "text/csv",
				method:      http.MethodGet,
				path:        "/api/v1/meters/" + mockMeters[0].ID + "/query",
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
				path:        "/api/v1/meters/" + mockMeters[0].ID + "/query?subject=s1",
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
				status: http.StatusNotImplemented,
			},
		},
		{
			name: "create portal token",
			req: testRequest{
				method:      http.MethodPost,
				path:        "/api/v1/portal/tokens/invalidate",
				contentType: "application/json",
			},
			res: testResponse{
				status: http.StatusNotImplemented,
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

			// status
			assert.Equal(t, tt.res.status, res.StatusCode)

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
