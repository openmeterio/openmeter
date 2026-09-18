package memorydedupe_test

import (
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/openmeter/dedupe"
	"github.com/openmeterio/openmeter/openmeter/dedupe/memorydedupe"
)

func TestIsUniqueDeduplicator(t *testing.T) {
	deduplicator, err := memorydedupe.NewDeduplicator(1024)
	require.NoError(t, err)

	const namespace = "default"

	ev := event.New()
	ev.SetID("id")
	ev.SetSource("source")

	isUnique, err := deduplicator.IsUnique(t.Context(), namespace, ev)
	isUnique2, err2 := deduplicator.IsUnique(t.Context(), namespace, ev)
	require.NoError(t, err)
	require.NoError(t, err2)

	assert.True(t, isUnique)
	assert.False(t, isUnique2)
}

func TestDeduplicator(t *testing.T) {
	deduplicator, err := memorydedupe.NewDeduplicator(1024)
	require.NoError(t, err)

	item := dedupe.Item{
		Namespace: "default",
		ID:        "id",
		Source:    "source",
	}

	isUnique, err := deduplicator.CheckUnique(t.Context(), item)
	_, errSet := deduplicator.Set(t.Context(), item)
	isUnique2, err2 := deduplicator.CheckUnique(t.Context(), item)
	require.NoError(t, errSet)
	require.NoError(t, err)
	require.NoError(t, err2)

	assert.True(t, isUnique)
	assert.False(t, isUnique2)
}

func TestRelease(t *testing.T) {
	deduplicator, err := memorydedupe.NewDeduplicator(1024)
	require.NoError(t, err)
	item := dedupe.Item{Namespace: "default", ID: "id", Source: "source"}
	claim, unique, err := deduplicator.Claim(t.Context(), item)
	require.NoError(t, err)
	require.True(t, unique)

	// A stale owner cannot release the current claim.
	require.NoError(t, deduplicator.Release(t.Context(), dedupe.Claim{Item: item, Token: "stale"}))
	unique, err = deduplicator.CheckUnique(t.Context(), item)
	require.NoError(t, err)
	require.False(t, unique)

	require.NoError(t, deduplicator.Release(t.Context(), claim))
	unique, err = deduplicator.CheckUnique(t.Context(), item)
	require.NoError(t, err)
	require.True(t, unique)
}

func TestReleaseDoesNotDeleteReacquiredClaim(t *testing.T) {
	deduplicator, err := memorydedupe.NewDeduplicator(1)
	require.NoError(t, err)
	item := dedupe.Item{Namespace: "default", ID: "id", Source: "source"}
	oldClaim, unique, err := deduplicator.Claim(t.Context(), item)
	require.NoError(t, err)
	require.True(t, unique)

	// Evict the old claim, then let a concurrent retry acquire a new one.
	_, unique, err = deduplicator.Claim(t.Context(), dedupe.Item{Namespace: "default", ID: "other", Source: "source"})
	require.NoError(t, err)
	require.True(t, unique)
	_, unique, err = deduplicator.Claim(t.Context(), item)
	require.NoError(t, err)
	require.True(t, unique)

	require.NoError(t, deduplicator.Release(t.Context(), oldClaim))
	unique, err = deduplicator.CheckUnique(t.Context(), item)
	require.NoError(t, err)
	require.False(t, unique)
}

