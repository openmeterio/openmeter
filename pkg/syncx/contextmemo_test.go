package syncx_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openmeterio/openmeter/pkg/syncx"
)

func TestContextMemo_NoCacheFallback(t *testing.T) {
	// Without Install, every GetOrLoad calls load directly (no memoization).
	memo := syncx.NewContextMemo[string, int]()

	var calls int
	load := func(context.Context) (int, error) {
		calls++
		return 42, nil
	}

	for range 3 {
		v, err := memo.GetOrLoad(context.Background(), "k", load)
		require.NoError(t, err)
		require.Equal(t, 42, v)
	}

	require.Equal(t, 3, calls, "no cache installed → load runs every call")
}

func TestContextMemo_MemoizesPerKeyWithinScope(t *testing.T) {
	memo := syncx.NewContextMemo[string, int]()
	ctx := memo.Install(context.Background())

	var callsA, callsB int
	loadA := func(context.Context) (int, error) { callsA++; return 1, nil }
	loadB := func(context.Context) (int, error) { callsB++; return 2, nil }

	for range 3 {
		a, err := memo.GetOrLoad(ctx, "a", loadA)
		require.NoError(t, err)
		require.Equal(t, 1, a)

		b, err := memo.GetOrLoad(ctx, "b", loadB)
		require.NoError(t, err)
		require.Equal(t, 2, b)
	}

	require.Equal(t, 1, callsA, "same key loads once within an installed scope")
	require.Equal(t, 1, callsB, "distinct keys are cached independently")
}

func TestContextMemo_CachesError(t *testing.T) {
	memo := syncx.NewContextMemo[string, int]()
	ctx := memo.Install(context.Background())

	sentinel := errors.New("boom")

	var calls int
	load := func(context.Context) (int, error) { calls++; return 0, sentinel }

	for range 2 {
		_, err := memo.GetOrLoad(ctx, "k", load)
		require.ErrorIs(t, err, sentinel)
	}

	require.Equal(t, 1, calls, "error result is cached for the scope, not retried")
}

func TestContextMemo_DistinctMemosDoNotCollide(t *testing.T) {
	// Two memos of identical K/V types must use separate stores.
	a := syncx.NewContextMemo[string, string]()
	b := syncx.NewContextMemo[string, string]()

	ctx := b.Install(a.Install(context.Background()))

	av, err := a.GetOrLoad(ctx, "k", func(context.Context) (string, error) { return "from-a", nil })
	require.NoError(t, err)
	bv, err := b.GetOrLoad(ctx, "k", func(context.Context) (string, error) { return "from-b", nil })
	require.NoError(t, err)

	require.Equal(t, "from-a", av)
	require.Equal(t, "from-b", bv)
}

func TestContextMemo_NestedInstallIsNoOp(t *testing.T) {
	memo := syncx.NewContextMemo[string, int]()

	outer := memo.Install(context.Background())

	var calls int
	load := func(context.Context) (int, error) { calls++; return 7, nil }

	_, err := memo.GetOrLoad(outer, "k", load)
	require.NoError(t, err)

	// Re-installing must reuse the outer store, so the cached entry survives.
	inner := memo.Install(outer)
	_, err = memo.GetOrLoad(inner, "k", load)
	require.NoError(t, err)

	require.Equal(t, 1, calls, "nested Install reuses the outer store")
}

func TestContextMemo_ConcurrentSingleLoad(t *testing.T) {
	memo := syncx.NewContextMemo[string, int]()
	ctx := memo.Install(context.Background())

	var calls atomic.Int64
	load := func(context.Context) (int, error) {
		calls.Add(1)
		return 99, nil
	}

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			v, err := memo.GetOrLoad(ctx, "k", load)
			require.NoError(t, err)
			require.Equal(t, 99, v)
		}()
	}
	wg.Wait()

	require.Equal(t, int64(1), calls.Load(), "concurrent GetOrLoad for one key loads exactly once")
}
