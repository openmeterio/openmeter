package ingest_test

import (
	"context"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe/memorydedupe"
	"github.com/openmeterio/openmeter/openmeter/ingest"
)

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
