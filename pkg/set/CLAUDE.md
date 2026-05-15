# set

<!-- archie:ai-start -->

> Thread-safe generic set backed by map[T]struct{} with RWMutex; provides Add, Remove, AsSlice, IsEmpty, Subtract, and Union for any domain package needing concurrent set semantics over comparable types.

## Patterns

**Construct via New[T](items...)** — Always use New[T]() or New(item1, item2). Never construct Set{} directly — the content map will be nil and any write will panic. (`s := set.New("a", "b")`)
**RWMutex for read/write separation** — Mutating methods (Add, Remove) acquire write locks; read methods (AsSlice, IsEmpty) acquire read locks. Subtract and Union acquire read locks on all input sets before building the result. (`s.mu.Lock(); defer s.mu.Unlock() // writes
s.mu.RLock(); defer s.mu.RUnlock() // reads`)
**Package-level Subtract and Union functions** — Subtract and Union are package-level functions, not methods, to support multi-set operations. Subtract(a, b) removes items in b from a — argument order matches lo.Difference(base, toRemove). (`result := set.Subtract(set.New(1,2,3), set.New(2,3)) // result: {1}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `set.go` | Full set implementation. All operations are in this single file. | AsSlice returns items in random map-iteration order — never assume stability. The slice is a snapshot; future mutations are not reflected. Subtract(a, b) argument order: a is the base set, b items are removed from it. |

## Anti-Patterns

- Constructing Set{} directly — nil content map panics on first write.
- Assuming stable ordering from AsSlice — map iteration is random.
- Holding the AsSlice result and expecting it to reflect future Add/Remove calls — it is a copy.
- Using this package for large sets under high contention — RWMutex is optimized for small sets; sync.Map may be preferable for high-read-concurrency large sets.

## Decisions

- **Use RWMutex with a plain map over sync.Map** — RWMutex with map[T]struct{} is simpler and lower-overhead for the small sets typical in domain use; sync.Map optimizes for high-concurrency read-heavy workloads with minimal write contention.

<!-- archie:ai-end -->
