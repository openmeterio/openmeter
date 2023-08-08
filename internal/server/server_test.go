package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/namespace"
	"github.com/openmeterio/openmeter/internal/server/router"
	"github.com/openmeterio/openmeter/internal/streaming"
	"github.com/openmeterio/openmeter/pkg/models"
)

var meters = []*models.Meter{
	{ID: ulid.Make().String(), Slug: "meter1", WindowSize: models.WindowSizeMinute, Aggregation: models.MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
	{ID: ulid.Make().String(), Slug: "meter2", WindowSize: models.WindowSizeMinute, Aggregation: models.MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
}

var values = []*models.MeterValue{
	{Subject: "s1", WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2021, 1, 1, 0, 1, 0, 0, time.UTC), Value: 100},
	{Subject: "s1", WindowStart: time.Date(2021, 1, 1, 0, 1, 0, 0, time.UTC), WindowEnd: time.Date(2021, 1, 1, 0, 2, 0, 0, time.UTC), Value: 200},
}

type MockConnector struct{}

func (c *MockConnector) CreateMeter(ctx context.Context, namespace string, meter *models.Meter) error {
	return nil
}

func (c *MockConnector) DeleteMeter(ctx context.Context, namespace string, meterSlug string) error {
	return nil
}

func (c *MockConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) ([]*models.MeterValue, *models.WindowSize, error) {
	values, err := models.AggregateMeterValues(values, meters[0].Aggregation, params.WindowSize)
	if err != nil {
		return nil, nil, err
	}

	return values, params.WindowSize, nil
}

type MockHandler struct{}

func (h MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request, params api.IngestEventsParams) {
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
			Meters:             meters,
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
		{
			name: "ingest event",
			req: testRequest{
				method:      http.MethodPost,
				path:        "/api/v1/events",
				contentType: "application/cloudevents+json",
				body: func() *cloudevents.Event {
					e := cloudevents.New()
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
			name: "create namespace",
			req: testRequest{
				method:      http.MethodPost,
				path:        "/api/v1/namespaces",
				contentType: "application/json",
				body: &models.Namespace{
					Namespace: "test",
				},
			},
			res: testResponse{
				status: http.StatusOK,
			},
		},
		{
			name: "list meters",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/meters",
			},
			res: testResponse{
				status: http.StatusOK,
				body:   meters,
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
				status: http.StatusOK,
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
		},
		{
			name: "get meter",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/meters/" + meters[0].ID,
			},
			res: testResponse{
				status: http.StatusOK,
				body:   meters[0],
			},
		},
		{
			name: "delete meter",
			req: testRequest{
				method: http.MethodDelete,
				path:   "/api/v1/meters/" + meters[0].Slug,
			},
			res: testResponse{
				status: http.StatusOK,
			},
		},
		{
			name: "get meter values",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/meters/" + meters[0].ID + "/values?windowSize=HOUR",
			},
			res: testResponse{
				status: http.StatusOK,
				body: struct {
					WindowSize models.WindowSize    `json:"windowSize"`
					Data       []*models.MeterValue `json:"data"`
				}{
					WindowSize: models.WindowSizeHour,
					Data: []*models.MeterValue{
						{Subject: "s1", WindowStart: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC), WindowEnd: time.Date(2021, 1, 1, 1, 0, 0, 0, time.UTC), Value: 300},
					},
				},
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

			resBody, _ := json.Marshal(tt.res.body)
			assert.JSONEq(t, string(resBody), w.Body.String())
		})
	}
}
