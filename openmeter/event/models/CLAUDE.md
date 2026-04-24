# models

<!-- archie:ai-start -->

> Holds shared payload model types for events that cross subsystem boundaries (e.g. FeatureKeyAndID, NamespaceID). These are minimal value types with inline Validate() methods used inside event payloads, not domain entities.

## Patterns

**Inline Validate() on every model** — Each struct exported from this package implements `Validate() error` that checks required fields. Callers must invoke Validate() before using the value in an event payload. (`if err := f.Validate(); err != nil { return fmt.Errorf("invalid feature ref: %w", err) }`)
**Minimal shared types only** — Only cross-subsystem value types belong here. Domain-specific structs belong in their own domain package, not in this shared models package. (`FeatureKeyAndID{Key: "storage", ID: "feat_01"} — used in both entitlement and billing event payloads`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Declares FeatureKeyAndID and NamespaceID — small value types with Validate() used as embedded fields inside event payloads across subsystems. | Do not add domain-specific structs here; this package is imported by many subsystems so adding heavy types creates wide coupling. |

## Anti-Patterns

- Adding domain-specific types (e.g. Invoice, Subscription) to this package — they belong in their own domain packages
- Skipping Validate() calls on values received in event payloads
- Importing openmeter domain packages from this package (creates circular dependencies)

## Decisions

- **Separate package for cross-subsystem event model types** — Prevents import cycles: both entitlement and billing packages can import event/models without pulling in each other's full domain.

<!-- archie:ai-end -->
