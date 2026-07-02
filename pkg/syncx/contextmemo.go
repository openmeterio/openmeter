package syncx

import (
	"context"
	"sync"
)

// ContextMemo is a request-scoped, keyed memoizer.
//
// Install it into a context (Install) to enable caching for that context's lifetime; without
// installation GetOrLoad falls back to a direct call, so callers that do not opt in are
// unaffected. Each ContextMemo value carries its own private context key (pointer identity), so
// several memos — even of the same K/V types — coexist in one context without colliding. Entries
// are computed at most once per key (sync.Once) and shared across concurrent goroutines.
//
// Typical use: dedup repeated lookups of the same immutable entity within one request/operation
// (e.g. the same customer fetched once per entitlement across a fan-out).
//
// INVARIANT — install ONLY around read-only operations. Cached values are never invalidated
// within a scope, so installing a memo around a call stack that MUTATES the cached entity would
// make reads after the write return the stale, pre-write value.
type ContextMemo[K comparable, V any] struct {
	key *contextMemoKey
}

// contextMemoKey is an unexported context-key type. It carries a field so it is NOT zero-sized:
// Go does not guarantee distinct addresses for zero-size allocations, so an empty struct would let
// two memos share a key and collide. A non-zero size makes each &contextMemoKey{} address unique,
// giving every ContextMemo its own context key even across identical K/V type parameters.
type contextMemoKey struct{ _ byte }

type contextMemoEntry[V any] struct {
	once sync.Once
	val  V
	err  error
}

// NewContextMemo returns a ContextMemo with its own private context key. Create one per logical
// cache (e.g. package-level var) and reuse it; the value is safe for concurrent use.
func NewContextMemo[K comparable, V any]() *ContextMemo[K, V] {
	return &ContextMemo[K, V]{key: &contextMemoKey{}}
}

// Install returns a context with this memo's store enabled. Nested installs are a no-op: the
// outermost scope wins and its store is shared by everything below it.
func (m *ContextMemo[K, V]) Install(ctx context.Context) context.Context {
	if _, ok := ctx.Value(m.key).(*sync.Map); ok {
		return ctx
	}

	return context.WithValue(ctx, m.key, &sync.Map{})
}

// GetOrLoad returns the memoized value for k when a store is installed in ctx; otherwise it calls
// load directly (no-cache fallback). load runs at most once per key within an installed scope,
// even under concurrency; both the value and the error are cached for the scope's lifetime.
func (m *ContextMemo[K, V]) GetOrLoad(ctx context.Context, k K, load func(context.Context) (V, error)) (V, error) {
	store, ok := ctx.Value(m.key).(*sync.Map)
	if !ok {
		return load(ctx)
	}

	v, _ := store.LoadOrStore(k, &contextMemoEntry[V]{})
	entry := v.(*contextMemoEntry[V])

	entry.once.Do(func() {
		entry.val, entry.err = load(ctx)
	})

	return entry.val, entry.err
}
