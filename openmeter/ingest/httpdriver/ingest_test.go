package httpdriver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	"github.com/openmeterio/openmeter/openmeter/ingest/httpdriver"
	"github.com/openmeterio/openmeter/openmeter/namespace/namespacedriver"
)

func TestIngestEvents(t *testing.T) {
	collector := ingest.NewInMemoryCollector()

	ingestSvc, err := ingest.NewService(ingest.Config{
		Collector: collector,
		Logger:    slog.Default(),
	})
	require.NoError(t, err)

	handler := httpdriver.New(
		namespacedriver.StaticNamespaceDecoder("test"),
		ingestSvc,
	)

	server := httptest.NewServer(handler.IngestEvents())
	client := server.Client()

	now := time.Date(2023, 0o6, 15, 14, 33, 0o0, 0o0, time.UTC)

	ev := event.New()
	ev.SetID("id")
	ev.SetTime(now)
	ev.SetSubject("sub")
	ev.SetSource("test")

	var buf bytes.Buffer

	err = json.NewEncoder(&buf).Encode(ev)
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

	ingestSvc, err := ingest.NewService(ingest.Config{
		Collector: collector,
		Logger:    slog.Default(),
	})
	require.NoError(t, err)

	handler := httpdriver.New(
		namespacedriver.StaticNamespaceDecoder("test"),
		ingestSvc,
	)

	server := httptest.NewServer(handler.IngestEvents())
	client := server.Client()

	resp, err := client.Post(server.URL, "application/cloudevents+json", bytes.NewBuffer([]byte(`invalid`)))
	require.NoError(t, err)

	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestBatchHandler(t *testing.T) {
	collector := ingest.NewInMemoryCollector()

	ingestSvc, err := ingest.NewService(ingest.Config{
		Collector: collector,
		Logger:    slog.Default(),
	})
	require.NoError(t, err)

	handler := httpdriver.New(
		namespacedriver.StaticNamespaceDecoder("test"),
		ingestSvc,
	)

	server := httptest.NewServer(handler.IngestEvents())
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
	err = json.NewEncoder(&buf).Encode(events)
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
