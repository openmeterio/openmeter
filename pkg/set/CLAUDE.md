# set

<!-- archie:ai-start -->

> Thread-safe generic set backed by map[T]struct{} with RWMutex; provides Add, Remove, AsSlice, IsEmpty, Subtract, and Union for any domain package needing concurrent set semantics over comparable types.

## Patterns

**Construct via New[T](items...)** — Always use New[T]() or New(item1, item2). Never construct Set{} directly — the content map will be nil and any write panics. (`s := set.New("a", "b")`)
**RWMutex read/write separation** — Mutating methods (Add, Remove) take write locks; read methods (AsSlice, IsEmpty) take read locks. Subtract and Union take read locks on all inputs before building the result. (`s.mu.Lock(); defer s.mu.Unlock() // writes
s.mu.RLock(); defer s.mu.RUnlock() // reads`)
**Package-level Subtract and Union** — Subtract and Union are package-level functions (not methods) to support multi-set operations. Subtract(a, b) removes items in b from a — argument order matches lo.Difference(base, toRemove). (`result := set.Subtract(set.New(1,2,3), set.New(2,3)) // {1}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `set.go` | Full set implementation; all operations in this single file. | AsSlice returns items in random map-iteration order — never assume stability; it is a snapshot, not a live view. Subtract(a, b): a is base, b items removed from it. |

## Anti-Patterns

- Constructing Set{} directly — nil content map panics on first write.
- Assuming stable ordering from AsSlice — map iteration is random.
- Expecting an AsSlice result to reflect future Add/Remove — it is a copy.
- Using this for large sets under high contention — RWMutex suits small sets; sync.Map may be preferable for high-read-concurrency large sets.

## Decisions

- **RWMutex with a plain map over sync.Map.** — Simpler and lower-overhead for the small sets typical in domain use; sync.Map optimizes for high-concurrency read-heavy workloads.

<!-- archie:ai-end -->
