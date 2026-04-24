# entity

<!-- archie:ai-start -->

> Defines all value types and input structs for the progressmanager domain: ProgressID, Progress, GetProgressInput, and UpsertProgressInput. Every type implements Validate() with errors.Join for multi-error aggregation.

## Patterns

**Validate() on every domain type using errors.Join** — Each struct has a Validate() method that collects all validation failures into a []error slice and returns errors.Join(...). Never return on first error. (`var errs []error
if a.ID == "" { errs = append(errs, errors.New("id is required")) }
return errors.Join(errs...)`)
**Delegate validation to embedded types** — Composite types call Validate() on embedded structs first, wrapping with fmt.Errorf for context. ProgressID validates NamespacedModel; GetProgressInput delegates to ProgressID. (`if err := a.ProgressID.Validate(); err != nil { errs = append(errs, fmt.Errorf("progress id: %w", err)) }`)
**Invariant enforcement in Progress.Validate()** — Progress enforces business invariants: Success+Failed <= Total, and non-zero counts require non-zero Total. (`if a.Success+a.Failed > a.Total { errs = append(errs, errors.New("success and failed must be less than or equal to total")) }`)
**Input types embed domain types directly** — GetProgressInput embeds ProgressID; UpsertProgressInput embeds Progress. Input types are thin validation wrappers, not separate structs. (`type GetProgressInput struct { ProgressID }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `progressmanager.go` | Single file for all entity types: ProgressID, Progress, GetProgressInput, UpsertProgressInput. Source of truth for domain invariants. | Adding new fields to Progress requires updating Validate() to enforce any new invariants; JSON tags must match the API schema. |

## Anti-Patterns

- Returning on first validation error instead of collecting all errors with errors.Join
- Adding persistence or Redis logic to entity types — this package is pure domain types only
- Omitting UpdatedAt zero-check in Progress.Validate() — callers rely on this guard before persisting

## Decisions

- **Entity types in a separate sub-package (entity/) rather than the root progressmanager package** — Avoids import cycles: adapter and httpdriver both need the entity types but neither should import the other; a shared entity sub-package breaks the cycle.

<!-- archie:ai-end -->
