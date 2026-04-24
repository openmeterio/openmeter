# set

<!-- archie:ai-start -->

> Thread-safe generic set backed by map[T]struct{} with RWMutex; provides Add, Remove, AsSlice, IsEmpty, Subtract, and Union operations for use across any domain package needing concurrent set semantics.

## Patterns

**RWMutex for read/write separation** — All mutating methods (Add, Remove) acquire a write lock; read methods (AsSlice, IsEmpty) acquire a read lock. Subtract and Union acquire read locks on all input sets before building the result. (`s.mu.Lock(); defer s.mu.Unlock() // writes
s.mu.RLock(); defer s.mu.RUnlock() // reads`)
**Construct via New(items...)** — Always use New[T]() or New(item1, item2) — never construct Set directly because the content map will be nil. (`s := set.New("a", "b")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `set.go` | Full set implementation; Subtract and Union are package-level functions, not methods, to support multi-set operations. | Subtract(a, b) removes items from a that are in b — the argument order is the same as lo.Difference(base, new), not the reverse. |

## Anti-Patterns

- Constructing Set{} directly (nil map panics on write)
- Calling AsSlice and assuming a stable order — iteration over a map is random
- Holding a reference to the returned slice from AsSlice and expecting it to reflect future mutations

## Decisions

- **Use RWMutex over sync.Map** — RWMutex with a plain map is simpler and lower-overhead for the small sets typical in domain use; sync.Map is optimized for high-concurrency read-heavy workloads with minimal contention.

<!-- archie:ai-end -->
