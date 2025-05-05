package ingestdriver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ingest"
	"github.com/openmeterio/openmeter/openmeter/ingest/ingestdriver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
	"github.com/openmeterio/openmeter/pkg/errorsx"
	"github.com/openmeterio/openmeter/pkg/models"
)

func TestIngestEvents(t *testing.T) {
	collector := ingest.NewInMemoryCollector()

	service := ingest.Service{
		Collector: collector,
		Logger:    slog.Default(),
	}

	handler := ingestdriver.NewIngestEventsHandler(
		service.IngestEvents,
		namespacedriver.StaticNamespaceDecoder("test"),
		nil,
		errorsx.NewNopHandler(),
	)

	server := httptest.NewServer(handler)
	client := server.Client()

	now := time.Date(2023, 0o6, 15, 14, 33, 0o0, 0o0, time.UTC)

	ev := event.New()
	ev.SetID("id")
	ev.SetTime(now)
	ev.SetSubject("sub")
	ev.SetSource("test")

	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(ev)
	require.NoError(t, err)

	resp, err := client.Post(server.URL, "application/cloudevents+json", &buf)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	events := collector.Events("test")

	require.Len(t, events, 1)

	receivedEvent := events[0]

	assert.Equal(t, ev.ID(), receivedEvent.ID())
	assert.Equal(t, ev.Subject(), receivedEvent.Subject())
	assert.Equal(t, ev.Source(), receivedEvent.Source())
	assert.Equal(t, receivedEvent.Time(), ev.Time())
}

func TestIngestEvents_InvalidEvent(t *testing.T) {
	collector := ingest.NewInMemoryCollector()

	service := ingest.Service{
		Collector: collector,
		Logger:    slog.Default(),
	}

	handler := ingestdriver.NewIngestEventsHandler(
		service.IngestEvents,
		namespacedriver.StaticNamespaceDecoder("test"),
		nil,
		errorsx.NewNopHandler(),
	)

	server := httptest.NewServer(handler)
	client := server.Client()

	// Invalid JSON
	resp, err := client.Post(server.URL, "application/cloudevents+json", bytes.NewBuffer([]byte(`invalid`)))
	require.NoError(t, err)

	resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// NaN is not allowed
	resp, err = client.Post(server.URL, "application/cloudevents+json", getMockEventPayload(t, "NaN"))
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	problem := toProblem(t, resp)
	assert.Equal(t, problem.Detail, "invalid event: invalid data: property NaN is not allowed")

	// Inf is not allowed
	resp, err = client.Post(server.URL, "application/cloudevents+json", getMockEventPayload(t, "Inf"))
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	problem = toProblem(t, resp)
	assert.Equal(t, problem.Detail, "invalid event: invalid data: property Inf is not allowed")

	// -Inf is not allowed
	resp, err = client.Post(server.URL, "application/cloudevents+json", getMockEventPayload(t, "-Inf"))
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	problem = toProblem(t, resp)
	assert.Equal(t, problem.Detail, "invalid event: invalid data: property -Inf is not allowed")

	// Nested NaN is not allowed
	resp, err = client.Post(server.URL, "application/cloudevents+json", getMockEventPayload(t, map[string]interface{}{
		"nested": map[string]interface{}{
			"value": []interface{}{1, "NaN"},
		},
	}))
	require.NoError(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	problem = toProblem(t, resp)
	assert.Equal(t, problem.Detail, `invalid event: invalid data at "nested.value.[1]": property NaN is not allowed`)
}

// toProblem converts a response body to a StatusProblem.
func toProblem(t *testing.T, resp *http.Response) models.StatusProblem {
	defer resp.Body.Close()

	var body []byte

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var problem models.StatusProblem

	err = json.Unmarshal(body, &problem)
	require.NoError(t, err)

	return problem
}

// getMockEventPayload returns a new event with the given data and returns the payload as a bytes.Buffer.
func getMockEventPayload(t *testing.T, data interface{}) *bytes.Buffer {
	ev := event.New()
	ev.SetID("id")
	ev.SetSource("test")
	ev.SetSubject("sub")
	ev.SetTime(time.Now())

	err := ev.SetData(event.ApplicationJSON, data)
	require.NoError(t, err)

	b, err := ev.MarshalJSON()
	require.NoError(t, err)

	return bytes.NewBuffer(b)
}

func TestBatchHandler(t *testing.T) {
	collector := ingest.NewInMemoryCollector()

	service := ingest.Service{
		Collector: collector,
		Logger:    slog.Default(),
	}

	handler := ingestdriver.NewIngestEventsHandler(
		service.IngestEvents,
		namespacedriver.StaticNamespaceDecoder("test"),
		nil,
		errorsx.NewNopHandler(),
	)

	server := httptest.NewServer(handler)
	client := server.Client()

	now := time.Date(2023, 0o6, 15, 14, 33, 0o0, 0o0, time.UTC)

	var events []event.Event
	for i := 1; i <= 10; i++ {
		id := strconv.Itoa(i)

		event := event.New()
		event.SetID(fmt.Sprintf("id%s", id))
		event.SetSubject(fmt.Sprintf("sub%s", id))
		event.SetSource(fmt.Sprintf("test%s", id))
		event.SetTime(now.Add(time.Duration(i) * time.Second))
		events = append(events, event)
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(events)
	require.NoError(t, err)

	resp, err := client.Post(server.URL, "application/cloudevents-batch+json", &buf)
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	collectedEvents := collector.Events("test")

	require.Len(t, collectedEvents, 10)
	for i, event := range collectedEvents {
		event := event
		assert.Equal(t, events[i].ID(), event.ID())
		assert.Equal(t, events[i].Subject(), event.Subject())
		assert.Equal(t, events[i].Source(), event.Source())
		assert.Equal(t, event.Time(), events[i].Time())
	}
}
