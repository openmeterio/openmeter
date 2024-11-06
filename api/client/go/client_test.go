package openmeter

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"

	"github.com/openmeterio/openmeter/pkg/models"
)

func TestIngest(t *testing.T) {
	ctx := context.Background()

	// Create a mock server to test the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/events", r.URL.Path)
		assert.Equal(t, "application/cloudevents+json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a client with the mock server
	om, err := NewClientWithResponses(server.URL)
	assert.NoError(t, err)

	resp, err := om.IngestEventWithResponse(ctx, mockEvent())
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func TestIngestBatch(t *testing.T) {
	ctx := context.Background()

	// Create a mock server to test the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/events", r.URL.Path)
		assert.Equal(t, "application/cloudevents-batch+json", r.Header.Get("Content-Type"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a client with the mock server
	om, err := NewClientWithResponses(server.URL)
	assert.NoError(t, err)

	resp, err := om.IngestEventBatchWithResponse(ctx, []cloudevents.Event{mockEvent()})
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func TetsListEvents(t *testing.T) {
	ctx := context.Background()

	event := mockEvent()

	// Create a mock server to test the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/events", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode([]Event{event})
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a client with the mock server
	om, err := NewClientWithResponses(server.URL)
	assert.NoError(t, err)

	resp, err := om.ListEventsWithResponse(ctx, &ListEventsParams{})
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, lo.ToPtr([]Event{event}), resp.JSON200)
}

func TestAuth(t *testing.T) {
	ctx := context.Background()

	// Create a mock server to test the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/events", r.URL.Path)
		assert.Equal(t, "application/cloudevents+json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer test-api-token", r.Header.Get("Authorization"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a client with the mock server
	om, err := NewAuthClientWithResponses(server.URL, "test-api-token")
	assert.NoError(t, err)

	resp, err := om.IngestEventWithResponse(ctx, mockEvent())
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func TestGetMeter(t *testing.T) {
	ctx := context.Background()

	meter := models.Meter{
		Slug:          "meter-1",
		Description:   "Test Meter",
		Aggregation:   models.MeterAggregationSum,
		ValueProperty: "$.tokens",
		GroupBy:       map[string]string{"model": "$.model", "type": "$.type"},
	}

	// Create a mock server to test the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/meters/meter-1", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(meter)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a client with the mock server
	om, err := NewClientWithResponses(server.URL)
	assert.NoError(t, err)

	resp, err := om.GetMeterWithResponse(ctx, "meter-1")
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, &meter, resp.JSON200)
}

func TestListMeters(t *testing.T) {
	ctx := context.Background()

	meters := []models.Meter{
		{
			Slug:          "meter-1",
			Description:   "Test Meter",
			Aggregation:   models.MeterAggregationSum,
			ValueProperty: "$.tokens",
			GroupBy:       map[string]string{"model": "$.model", "type": "$.type"},
		},
		{
			Slug:          "meter-2",
			Description:   "Test Meter 2",
			Aggregation:   models.MeterAggregationSum,
			ValueProperty: "$.tokens",
			GroupBy:       map[string]string{"model": "$.model", "type": "$.type"},
		},
	}

	// Create a mock server to test the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/meters", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(meters)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a client with the mock server
	om, err := NewClientWithResponses(server.URL)
	assert.NoError(t, err)

	resp, err := om.ListMetersWithResponse(ctx)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, &meters, resp.JSON200)
}

func TestMeterQuery(t *testing.T) {
	ctx := context.Background()

	result := MeterQueryResult{
		Data: []models.MeterQueryRow{
			{
				Subject:     lo.ToPtr("customer-1"),
				WindowStart: time.Now().UTC(),
				WindowEnd:   time.Now().UTC(),
				Value:       123,
			},
		},
	}

	// Create a mock server to test the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/meters/meter-1/query", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(result)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a client with the mock server
	om, err := NewClientWithResponses(server.URL)
	assert.NoError(t, err)

	subjectFilter := []string{"customer-1"}

	resp, err := om.QueryMeterWithResponse(ctx, "meter-1", &QueryMeterParams{
		Subject: &subjectFilter,
	})
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, &result, resp.JSON200)
}

func TestListSubjects(t *testing.T) {
	ctx := context.Background()

	subjects := []Subject{
		{
			Key:         "customer-1",
			DisplayName: lo.ToPtr("Customer 1"),
		},
		{
			Key:         "customer-2",
			DisplayName: lo.ToPtr("Customer 2"),
		},
	}

	// Create a mock server to test the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/subjects", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode(subjects)
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a client with the mock server
	om, err := NewClientWithResponses(server.URL)
	assert.NoError(t, err)

	resp, err := om.ListSubjectsWithResponse(ctx)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, &subjects, resp.JSON200)
}

func TestUpsertSubject(t *testing.T) {
	ctx := context.Background()

	subject := Subject{
		Key:         "customer-1",
		DisplayName: lo.ToPtr("Customer 1"),
	}

	// Create a mock server to test the client
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/v1/subjects", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err := json.NewEncoder(w).Encode([]Subject{subject})
		assert.NoError(t, err)
	}))
	defer server.Close()

	// Create a client with the mock server
	om, err := NewClientWithResponses(server.URL)
	assert.NoError(t, err)

	resp, err := om.UpsertSubjectWithResponse(ctx, []SubjectUpsert{
		{
			Key:         "customer-1",
			DisplayName: lo.ToPtr("Customer 1"),
		},
	})
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, lo.ToPtr([]Subject{subject}), resp.JSON200)
}

// mockEvent creates a mock CloudEvent for testing
func mockEvent() cloudevents.Event {
	e := cloudevents.New()
	eventTime, _ := time.Parse(time.RFC3339, "2024-11-05T22:35:52.457Z")
	e.SetTime(eventTime)
	e.SetID("ec2672e8-458d-4c5e-8a3c-f3235dd38ba5")
	e.SetSource("my-app")
	e.SetType("usage-reports")
	e.SetSubject("customer-1")
	_ = e.SetData("application/json", map[string]string{
		"reports": "123",
		"type":    "type",
	})
	return e
}
