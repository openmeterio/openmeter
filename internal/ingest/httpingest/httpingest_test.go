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
)

type inMemorySink struct {
	events []event.Event

	mu sync.Mutex
}

func (s *inMemorySink) Receive(event event.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events = append(s.events, event)

	return nil
}

func TestHandler(t *testing.T) {
	sink := &inMemorySink{}
	handler := Handler{
		Collector: sink,
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

	resp, err := client.Post(server.URL, "", &buf)
	require.NoError(t, err)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	require.Len(t, sink.events, 1)

	receivedEvent := sink.events[0]

	assert.Equal(t, ev.ID(), receivedEvent.ID())
	assert.Equal(t, ev.Subject(), receivedEvent.Subject())
	assert.Equal(t, ev.Source(), receivedEvent.Source())
	assert.Equal(t, receivedEvent.Time(), ev.Time())
}
