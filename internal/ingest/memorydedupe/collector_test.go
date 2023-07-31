package memorydedupe

import (
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
)

// We need to define a counter to be able to change intenral state as Collector.Ingest is not a pointer receiver
type testCounter struct {
	counter int
}

func (c *testCounter) inc() {
	c.counter++
}

func newTestCounter() testCounter {
	return testCounter{
		counter: 0,
	}
}

type testEvent struct {
	id        string
	source    string
	namespace string
}

type testCollector struct {
	ingestCount *testCounter
}

func newTestCollector(ingestCount *testCounter) testCollector {
	return testCollector{
		ingestCount: ingestCount,
	}
}

func (c testCollector) Ingest(ev event.Event, namespace string) error {
	c.ingestCount.inc()
	return nil
}

func (c testCollector) Close() {
}

func TestIsUnique(t *testing.T) {
	tests := []struct {
		name   string
		events []testEvent
		want   int
	}{
		{
			name:   "Unique id",
			events: []testEvent{{"id1", "source1", "namespace1"}, {"id2", "source1", "namespace1"}},
			want:   2,
		},
		{
			name:   "Unique source",
			events: []testEvent{{"id1", "source1", "namespace1"}, {"id1", "source2", "namespace1"}},
			want:   2,
		},
		{
			name:   "Unique namespace",
			events: []testEvent{{"id1", "source1", "namespace1"}, {"id1", "source1", "namespace2"}},
			want:   2,
		},
		{
			name:   "Same",
			events: []testEvent{{"id1", "source1", "namespace1"}, {"id1", "source1", "namespace1"}},
			want:   1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ingestCount := newTestCounter()
			downstreamCollector := newTestCollector(&ingestCount)
			dedupeCollector, err := NewCollector(CollectorConfig{
				Collector: downstreamCollector,
				Size:      128,
			})
			if err != nil {
				t.Error(err)
			}
			for _, te := range tt.events {
				ev := event.New()
				ev.SetID(te.id)
				ev.SetSource(te.source)
				err := dedupeCollector.Ingest(ev, te.namespace)
				if err != nil {
					t.Error(err)
				}
			}

			got := downstreamCollector.ingestCount.counter

			assert.Equal(t, tt.want, got)
		})
	}
}
