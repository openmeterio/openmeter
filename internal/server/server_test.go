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

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
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

var mockMeters = []models.Meter{
	{ID: ulid.Make().String(), Slug: "meter1", WindowSize: models.WindowSizeMinute, Aggregation: models.MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
	{ID: ulid.Make().String(), Slug: "meter2", WindowSize: models.WindowSizeMinute, Aggregation: models.MeterAggregationSum, EventType: "event", ValueProperty: "$.value"},
}

var mockValues = []models.MeterValue{
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

func (c *MockConnector) QueryMeter(ctx context.Context, namespace string, meterSlug string, params *streaming.QueryParams) (*streaming.QueryResult, error) {
	// Clone mockValues to avoid side effects between tests caused by the AggregateMeterValues
	inputValues := []*models.MeterValue{}
	for _, v := range mockValues {
		v := v
		inputValues = append(inputValues, &v)
	}

	values, err := models.AggregateMeterValues(inputValues, mockMeters[0].Aggregation, params.WindowSize)
	if err != nil {
		return nil, err
	}

	return &streaming.QueryResult{
		Values:     values,
		WindowSize: params.WindowSize,
	}, nil
}

func (c *MockConnector) ListMeterSubjects(ctx context.Context, namespace string, meterSlug string) ([]string, error) {
	return []string{"s1"}, nil
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
				body: api.Namespace{
					Namespace: "test",
				},
			},
			res: testResponse{
				status: http.StatusCreated,
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
				status: http.StatusNoContent,
			},
		},
		{
			name: "query meter",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/meters/" + mockMeters[0].ID + "/query?windowSize=HOUR",
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
		{
			name: "query meter as csv",
			req: testRequest{
				accept: "text/csv",
				method: http.MethodGet,
				path:   "/api/v1/meters/" + mockMeters[0].ID + "/query?windowSize=HOUR",
			},
			res: testResponse{
				status: http.StatusOK,
				body: strings.Join(
					[]string{
						"window_start,window_end,subject,value",
						"2021-01-01T00:00:00Z,2021-01-01T01:00:00Z,s1,300.000000",
					},
					"\n",
				),
			},
		},
		{
			name: "get meter values",
			req: testRequest{
				method: http.MethodGet,
				path:   "/api/v1/meters/" + mockMeters[0].ID + "/values?windowSize=HOUR",
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
