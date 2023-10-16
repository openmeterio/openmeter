package memorydedupe_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/internal/dedupe"
	"github.com/openmeterio/openmeter/internal/dedupe/memorydedupe"
)

func TestDeduplicator(t *testing.T) {
	deduplicator, err := memorydedupe.NewDeduplicator(1024)
	require.NoError(t, err)

	item := dedupe.Item{
		Namespace: "default",
		ID:        "id",
		Source:    "source",
	}

	isUnique, err := deduplicator.IsUnique(context.Background(), item)
	errSet := deduplicator.Set(context.Background(), item)
	isUnique2, err2 := deduplicator.IsUnique(context.Background(), item)
	require.NoError(t, errSet)
	require.NoError(t, err)
	require.NoError(t, err2)

	assert.True(t, isUnique)
	assert.False(t, isUnique2)
}
