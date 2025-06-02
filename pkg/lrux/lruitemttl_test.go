package lrux

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/clock"
)

func TestCacheWithItemTTLSanity(t *testing.T) {
	require := require.New(t)

	cache, err := NewCacheWithItemTTL(10, func(ctx context.Context, key string) (int64, error) {
		return clock.Now().Unix(), nil
	}, WithTTL(time.Second*10))
	require.NoError(err)

	ctx := t.Context()

	baseTime := lo.Must(time.Parse(time.RFC3339, "2021-01-01T00:00:00Z"))
	clock.FreezeTime(baseTime)
	defer clock.UnFreeze()

	item, err := cache.Get(ctx, "test")
	require.NoError(err)
	require.Equal(baseTime.Unix(), item)

	// Clock advances 5 seconds => we should have a cache hit
	clock.SetTime(baseTime.Add(time.Second * 5))
	item, err = cache.Get(ctx, "test")
	require.NoError(err)
	require.Equal(baseTime.Unix(), item)

	// Clock advances 10 seconds => we should have a cache hit
	clock.SetTime(baseTime.Add(time.Second * 10))
	item, err = cache.Get(ctx, "test")
	require.NoError(err)
	require.Equal(clock.Now().Unix(), item)
	newBase := clock.Now().Unix()

	// Clock advances 15 seconds => we have a cache hit
	clock.SetTime(baseTime.Add(time.Second * 15))
	item, err = cache.Get(ctx, "test")
	require.NoError(err)
	require.Equal(newBase, item)

	// Refresh the item => we have updated value
	item, err = cache.Refresh(ctx, "test")
	require.NoError(err)
	require.Equal(clock.Now().Unix(), item)
}

func TestCacheWithItemTTLSanity_ErrorHandling(t *testing.T) {
	require := require.New(t)

	cache, err := NewCacheWithItemTTL(10, func(ctx context.Context, key string) (int64, error) {
		return 0, errors.New("test error")
	}, WithTTL(time.Second*10))
	require.NoError(err)

	_, err = cache.Get(t.Context(), "test")
	require.Error(err)

	_, err = cache.Refresh(t.Context(), "test")
	require.Error(err)
}
