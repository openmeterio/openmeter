package ingest_test

import (
	"context"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/ingest"
)

func TestInMemoryCollector(t *testing.T) {
	collector := ingest.NewInMemoryCollector()

	const namespace = "default"

	ev := event.New()
	ev.SetID("id")
	ev.SetSource("source")

	err := collector.Ingest(context.Background(), namespace, ev)
	require.NoError(t, err)

	assert.Equal(t, []string{namespace}, collector.Namespaces())
	assert.Equal(t, []event.Event{ev}, collector.Events(namespace))
}
