package ingest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe/memorydedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest"
)

// failingCollector fails the first Ingest call, then succeeds, simulating a
// transient downstream delivery failure.
type failingCollector struct {
	fail   bool
	events []event.Event
}

func (c *failingCollector) Ingest(_ context.Context, namespace string, ev event.Event) error {
	if c.fail {
		c.fail = false
		return errors.New("delivery failed")
	}

	c.events = append(c.events, ev)
	return nil
}

func (c *failingCollector) Close() {}

func TestDeduplicatingCollector(t *testing.T) {
	collector := ingest.NewInMemoryCollector()
	deduplicator, err := memorydedupe.NewDeduplicator(0)
	require.NoError(t, err)

	dedupeCollector := ingest.DeduplicatingCollector{
		Collector:    collector,
		Deduplicator: deduplicator,
	}

	const namespace = "default"

	ev1 := event.New()
	ev1.SetID("id")
	ev1.SetSource("source")
	ev1.SetType("some-type")

	ev2 := event.New()
	ev2.SetID("id")
	ev2.SetSource("source")
	ev2.SetType("some-other-type")

	err = dedupeCollector.Ingest(context.Background(), namespace, ev1)
	require.NoError(t, err)

	err = dedupeCollector.Ingest(context.Background(), namespace, ev2)
	require.NoError(t, err)

	assert.Equal(t, []event.Event{ev1}, collector.Events(namespace))
}

func TestDeduplicatingCollectorRetriesFailedDelivery(t *testing.T) {
	collector := &failingCollector{fail: true}
	deduplicator, err := memorydedupe.NewDeduplicator(0)
	require.NoError(t, err)

	dedupeCollector := ingest.DeduplicatingCollector{
		Collector:    collector,
		Deduplicator: deduplicator,
	}

	const namespace = "default"

	ev := event.New()
	ev.SetID("id")
	ev.SetSource("source")
	ev.SetType("some-type")

	// First attempt fails at delivery; it must not be marked as seen.
	err = dedupeCollector.Ingest(context.Background(), namespace, ev)
	require.Error(t, err)
	assert.Empty(t, collector.events)

	// Retrying the same event must not be silently deduplicated.
	err = dedupeCollector.Ingest(context.Background(), namespace, ev)
	require.NoError(t, err)
	assert.Equal(t, []event.Event{ev}, collector.events)
}
