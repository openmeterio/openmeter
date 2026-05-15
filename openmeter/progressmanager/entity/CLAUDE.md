# entity

<!-- archie:ai-start -->

> Pure domain types for the progressmanager domain: ProgressID, Progress, GetProgressInput, UpsertProgressInput. No persistence logic — every type implements Validate() with errors.Join for multi-error aggregation.

## Patterns

**Validate() collects all errors via errors.Join** — Each Validate() method appends all failures into a []error slice and returns errors.Join(...). Never return early on first error. (`var errs []error
if a.ID == "" { errs = append(errs, errors.New("id is required")) }
return errors.Join(errs...)`)
**Delegate validation to embedded types** — Composite types call Validate() on embedded structs first and wrap with fmt.Errorf for context before appending to errs. (`if err := a.ProgressID.Validate(); err != nil { errs = append(errs, fmt.Errorf("progress id: %w", err)) }`)
**Business invariants enforced in Progress.Validate()** — Progress.Validate() enforces: Success+Failed <= Total, and non-zero counts require non-zero Total, and UpdatedAt must not be zero. (`if a.Success+a.Failed > a.Total { errs = append(errs, errors.New("success and failed must be less than or equal to total")) }`)
**Input types embed domain types directly** — GetProgressInput embeds ProgressID; UpsertProgressInput embeds Progress. Input types are thin validation wrappers — no extra fields. (`type GetProgressInput struct { ProgressID }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `progressmanager.go` | Single file containing all entity types: ProgressID, Progress, GetProgressInput, UpsertProgressInput. Source of truth for domain invariants. | Adding new fields to Progress requires updating Validate() to enforce any new invariants; JSON tags must match the generated API schema. |

## Anti-Patterns

- Returning on first validation error instead of collecting all errors with errors.Join
- Adding persistence, Redis, or HTTP logic to entity types — this package is pure domain types only
- Omitting UpdatedAt zero-check in Progress.Validate() — callers rely on this guard before persisting
- Embedding extra adapter-specific fields into entity types — keep entity types decoupled from storage details

## Decisions

- **Entity types in a separate sub-package (entity/) rather than the root progressmanager package** — Avoids import cycles: adapter and httpdriver both need the entity types but neither should import the other; a shared entity sub-package breaks the cycle cleanly.

<!-- archie:ai-end -->
