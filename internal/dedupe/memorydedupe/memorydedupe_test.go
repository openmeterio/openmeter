package memorydedupe_test

import (
	"context"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/internal/dedupe/memorydedupe"
)

func TestDeduplicator(t *testing.T) {
	deduplicator, err := memorydedupe.NewDeduplicator(1024)
	require.NoError(t, err)

	const namespace = "default"

	ev := event.New()
	ev.SetID("id")
	ev.SetSource("source")

	isUnique, err := deduplicator.IsUnique(context.Background(), namespace, ev)
	isUnique2, err2 := deduplicator.IsUnique(context.Background(), namespace, ev)
	require.NoError(t, err)
	require.NoError(t, err2)

	assert.True(t, isUnique)
	assert.False(t, isUnique2)
}
