# set

<!-- archie:ai-start -->

> A tiny generic, mutex-guarded set data structure (Set[T comparable]) plus free functions for set algebra. Used by billing adapter/httpdriver to deduplicate and diff comparable keys.

## Patterns

**Constructor returns pointer** — New[T](items...) builds the internal map and returns *Set[T]; callers always hold a pointer because Set carries a sync.RWMutex which must not be copied. (`s := set.New(1, 2, 3) // *Set[int]`)
**Lock discipline by intent** — Mutating methods (Add, Remove) take s.mu.Lock(); read methods (AsSlice, IsEmpty) take s.mu.RLock(). Free functions RLock every input set under defer before reading content. (`func (s *Set[T]) Add(items ...T) { s.mu.Lock(); defer s.mu.Unlock(); ... }`)
**Variadic free functions for algebra** — Subtract(a, b...) and Union(sets...) are package-level functions that return a fresh *Set, never mutating inputs. (`res := set.Union(set.New(1,2), set.New(2,3))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `set.go` | Whole package: Set[T] type, New/Add/Remove/AsSlice/IsEmpty methods, Subtract/Union free functions. | Set embeds sync.RWMutex via field mu; never copy a Set value (always pass *Set). Subtract RLocks `a` and all `b` sets — passing the same set twice deadlocks via re-entrant RLock under contention. |
| `set_test.go` | Tests Union/Subtract/IsEmpty plus a locking smoke test using testify/assert.ElementsMatch. | AsSlice order is non-deterministic (map iteration); assert with ElementsMatch, not Equal. |

## Anti-Patterns

- Copying a Set by value — duplicates the embedded mutex and breaks locking.
- Relying on AsSlice ordering — map iteration is unordered.
- Adding methods that read content without taking at least RLock.

## Decisions

- **Mutex-guarded rather than a bare map[T]struct{}.** — Comment-documented concurrency smoke test shows the package is meant to be safe for concurrent Add/Remove.

## Example: Diff two key sets without mutating inputs

```
import "github.com/openmeterio/openmeter/pkg/set"

added := set.Subtract(set.New(newKeys...), set.New(oldKeys...))
if !added.IsEmpty() { handle(added.AsSlice()) }
```

<!-- archie:ai-end -->
