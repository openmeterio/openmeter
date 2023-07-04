package httpingest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type inMemoryCollector struct {
	events []event.Event

	mu sync.Mutex
}

func (s *inMemoryCollector) Receive(event event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)

	return nil
}

func TestHandler(t *testing.T) {
	collector := &inMemoryCollector{}
	handler := Handler{
		Collector: collector,
	}
	server := httptest.NewServer(handler)
	client := server.Client()

	now := time.Date(2023, 06, 15, 14, 33, 00, 00, time.UTC)

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

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	require.Len(t, collector.events, 1)

	receivedEvent := collector.events[0]

	assert.Equal(t, ev.ID(), receivedEvent.ID())
	assert.Equal(t, ev.Subject(), receivedEvent.Subject())
	assert.Equal(t, ev.Source(), receivedEvent.Source())
	assert.Equal(t, receivedEvent.Time(), ev.Time())
}

func TestBatchHandler(t *testing.T) {
	collector := &inMemoryCollector{}
	handler := Handler{
		Collector: collector,
	}
	server := httptest.NewServer(handler)
	client := server.Client()

	var events []event.Event
	for i := 1; i <= 10; i++ {
		id := strconv.Itoa(i)

		event := event.New()
		event.SetID("id" + id)
		event.SetSubject("sub" + id)
		event.SetSource("test" + id)
		events = append(events, event)
	}

	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(events)
	require.NoError(t, err)

	resp, err := client.Post(server.URL, "application/cloudevents-batch+json", &buf)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	require.Len(t, collector.events, 10)

	lastRecivedEvent := collector.events[len(collector.events) - 1]
	comperableEvent := collector.events[len(collector.events) - 2]

	assert.Equal(t, "id10", lastRecivedEvent.ID())
	assert.Equal(t, "sub10", lastRecivedEvent.Subject())
	assert.Equal(t, "test10", lastRecivedEvent.Source())
	assert.NotEqual(t, comperableEvent.Time(), lastRecivedEvent.Time())
}
