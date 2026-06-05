# entitydiff

<!-- archie:ai-start -->

> Generic three-way diff (Create/Update/Delete) of entity slices keyed by ID, with soft-delete awareness. Backs billing adapter reconciliation of persisted-vs-expected state for entities with and without child entities.

## Patterns

**Entity interface contract** — Diffed types must implement Entity (GetID() string, IsDeleted() bool). Empty GetID() means a not-yet-persisted item; IsDeleted() drives delete vs skip decisions. (`type Entity interface { GetID() string; IsDeleted() bool }`)
**DiffByID for entities with children** — DiffByID(DiffByIDInput[T]) compares only by ID (never field equality) and dispatches HandleCreate/HandleUpdate/HandleDelete callbacks, joining their errors with errors.Join. Use when entities own children that need recursive handling. (`entitydiff.DiffByID(entitydiff.DiffByIDInput[Line]{DBState: db, ExpectedState: exp, HandleUpdate: ...})`)
**DiffByIDEqualer for leaf entities** — DiffByIDEqualer[T EqualerEntity[T]](expected, db) returns a Diff[T] and only marks an update when PersistedState.Equal(ExpectedState) is false. Use for entities with no children. (`diff := entitydiff.DiffByIDEqualer(expected, dbState)`)
**Soft-delete-aware correlation** — diffByID treats IsDeleted()+no-DB-row as skip, expected-deleted+live-DB-row as Delete (carrying the expected/target state so co-edited fields persist), and DB rows absent from expected as Delete. (`if expected.IsDeleted() && !dbState.IsDeleted() { diff.Delete = append(diff.Delete, expected) }`)
**NestedEntity wrappers carry a parent** — NestedEntity/EqualerNestedEntity wrap a child with its Parent while delegating GetID/IsDeleted/Equal to the child; build with NewEqualersWithParent to diff children while retaining parent context. (`wrapped := entitydiff.NewEqualersWithParent(lines, invoice)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `diff.go` | Diff/DiffUpdate types, diffByID core, DiffByID + DiffByIDEqualer public entry points, Union | The internal diffByID returns UpdateCandidates (unfiltered); only DiffByIDEqualer applies Equal() to drop no-op updates — DiffByID emits every candidate as an update |
| `parent.go` | NestedEntity/EqualerNestedEntity parent-carrying wrappers + NewEqualersWithParent | Equal on the wrapper compares only the embedded Entity, not the Parent |

## Anti-Patterns

- Using DiffByIDEqualer for entities with child entities — field equality skips updates whose only change is in children
- Assuming an empty GetID() item will be correlated to a DB row — it is always treated as Create (or skipped if deleted)
- Expecting DiffByID to compare fields; it correlates by ID only and relies on caller callbacks

## Decisions

- **Split DiffByID (callback, children) from DiffByIDEqualer (Equal-based, leaf)** — Parent entities need recursive child handling that field equality cannot express, while leaf entities benefit from automatic no-op-update suppression
- **Delete entries carry the expected/target state, not the DB state, for expected-deleted rows** — Co-edited fields (e.g. managedBy flipping to manual alongside deleted_at) must be persisted in the same change

## Example: Reconcile leaf entities, applying only real updates

```
import "github.com/openmeterio/openmeter/pkg/entitydiff"

diff := entitydiff.DiffByIDEqualer(expectedLines, dbLines)
for _, c := range diff.Create { /* insert */ }
for _, u := range diff.Update { /* update u.ExpectedState */ }
for _, d := range diff.Delete { /* delete */ }
```

<!-- archie:ai-end -->
