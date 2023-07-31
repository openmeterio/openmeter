package httpingest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/api"
	"github.com/openmeterio/openmeter/internal/namespace"
)

type inMemoryCollector struct {
	events []event.Event

	mu sync.Mutex
}

func (s *inMemoryCollector) Ingest(event event.Event, namespace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)

	return nil
}

func (s *inMemoryCollector) Close() {}

// Wrap the handler so we can set the namespace with `httptest“
type MockHandler struct {
	handler *Handler
}

func (h MockHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	namespace := "test"
	h.handler.ServeHTTP(w, r, api.IngestEventsParams{NamespaceInput: &namespace})
}

func TestHandler(t *testing.T) {
	collector := &inMemoryCollector{}
	httpHandler, err := NewHandler(HandlerConfig{
		Collector:        collector,
		NamespaceManager: &namespace.Manager{},
	})
	require.NoError(t, err)
	handler := MockHandler{
		handler: httpHandler,
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

	err = json.NewEncoder(&buf).Encode(ev)
	require.NoError(t, err)

	resp, err := client.Post(server.URL, "", &buf)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	require.Len(t, collector.events, 1)

	receivedEvent := collector.events[0]

	assert.Equal(t, ev.ID(), receivedEvent.ID())
	assert.Equal(t, ev.Subject(), receivedEvent.Subject())
	assert.Equal(t, ev.Source(), receivedEvent.Source())
	assert.Equal(t, receivedEvent.Time(), ev.Time())
}
