# models

<!-- archie:ai-start -->

> Minimal cross-subsystem value types (FeatureKeyAndID, NamespaceID) that appear inside event payloads shared across multiple domain packages (entitlement, billing, productcatalog). Each type implements Validate() to enforce required fields before payload construction. A leaf package with no domain imports.

## Patterns

**Inline Validate() on every exported struct** — Every struct implements Validate() error checking all required fields; callers must call it before embedding the value in an event payload. (`f := models.FeatureKeyAndID{Key: featureKey, ID: featureID}
if err := f.Validate(); err != nil { return fmt.Errorf("invalid feature ref: %w", err) }`)
**Cross-subsystem value types only** — Add a type here only if it appears in event payloads used by two or more domain packages; domain-specific structs belong in their own package. (`// FeatureKeyAndID used in both entitlement and billing payloads — correct; Invoice{} belongs in openmeter/billing`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Declares FeatureKeyAndID (Key+ID feature ref) and NamespaceID, embedded in event payloads across entitlement and billing. | Do not import openmeter domain packages — this package is imported by all of them and would cause circular imports; keep types to string fields + Validate(). |

## Anti-Patterns

- Adding domain-specific types (Invoice, Subscription, Entitlement) — they belong in their domain packages.
- Skipping Validate() on values received in event payloads from this package.
- Importing openmeter/* domain packages from models.go — must remain a leaf to avoid cycles.
- Adding business logic or computed fields — these are pure data containers.

## Decisions

- **Separate package for cross-subsystem event value types rather than duplicating per domain.** — Both entitlement and billing need FeatureKeyAndID in payloads; a shared leaf package lets both import it without importing each other, preventing cycles.

<!-- archie:ai-end -->
