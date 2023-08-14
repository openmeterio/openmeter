package ingest_test

import (
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/internal/ingest"
)

func TestInMemoryCollector(t *testing.T) {
	collector := ingest.NewInMemoryCollector()

	const namespace = "default"

	// TODO: expand event fields
	ev := event.New()

	err := collector.Ingest(ev, namespace)
	require.NoError(t, err)

	assert.Equal(t, []string{namespace}, collector.Namespaces())
	assert.Equal(t, []event.Event{ev}, collector.Events(namespace))
}
