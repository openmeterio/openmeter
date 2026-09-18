package ingest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
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

// transientFailCollector fails the first Ingest call and succeeds after,
// modelling a transient downstream (e.g. broker) failure.
type transientFailCollector struct {
	collector *ingest.InMemoryCollector
	failed    bool
}

func (c *transientFailCollector) Ingest(ctx context.Context, namespace string, ev event.Event) error {
	if !c.failed {
		c.failed = true

		return errors.New("transient ingest failure")
	}

	return c.collector.Ingest(ctx, namespace, ev)
}

func (c *transientFailCollector) Close() { c.collector.Close() }

// Regression test for https://github.com/openmeterio/openmeter/issues/5118:
// a failed ingest must release the dedupe claim so the documented client retry
// is metered instead of being dropped as a duplicate.
func TestDeduplicatingCollector_ReleasesClaimOnIngestFailure(t *testing.T) {
	inMemory := ingest.NewInMemoryCollector()
	collector := &transientFailCollector{collector: inMemory}
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

	// The first attempt claims the dedupe key but fails to ingest.
	err = dedupeCollector.Ingest(context.Background(), namespace, ev)
	require.Error(t, err)
	assert.Empty(t, inMemory.Events(namespace))

	// The client's retry must not be dropped as a duplicate: the event is metered exactly once.
	err = dedupeCollector.Ingest(context.Background(), namespace, ev)
	require.NoError(t, err)
	assert.Equal(t, []event.Event{ev}, inMemory.Events(namespace))

	// A genuine duplicate afterwards is still deduplicated.
	err = dedupeCollector.Ingest(context.Background(), namespace, ev)
	require.NoError(t, err)
	assert.Equal(t, []event.Event{ev}, inMemory.Events(namespace))
}

// failRemoveDeduplicator fails Remove calls, modelling a dedupe store outage
// during claim release.
type failRemoveDeduplicator struct {
	dedupe.Deduplicator
	err error
}

func (d failRemoveDeduplicator) Remove(ctx context.Context, items ...dedupe.Item) error {
	return d.err
}

// If releasing the claim fails after a failed ingest, both failures must be
// surfaced: dropping the event silently is the bug being fixed.
func TestDeduplicatingCollector_ReleaseFailureIsLoud(t *testing.T) {
	inMemory := ingest.NewInMemoryCollector()
	collector := &transientFailCollector{collector: inMemory}
	deduplicator, err := memorydedupe.NewDeduplicator(0)
	require.NoError(t, err)

	removeErr := errors.New("dedupe store unavailable")

	dedupeCollector := ingest.DeduplicatingCollector{
		Collector:    collector,
		Deduplicator: failRemoveDeduplicator{Deduplicator: deduplicator, err: removeErr},
	}

	const namespace = "default"

	ev := event.New()
	ev.SetID("id")
	ev.SetSource("source")
	ev.SetType("some-type")

	err = dedupeCollector.Ingest(context.Background(), namespace, ev)
	require.Error(t, err)
	assert.ErrorIs(t, err, removeErr)
	assert.ErrorContains(t, err, "transient ingest failure")
	assert.ErrorContains(t, err, "releasing dedupe claim")
}
