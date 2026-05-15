# models

<!-- archie:ai-start -->

> Holds minimal cross-subsystem value types (FeatureKeyAndID, NamespaceID) that appear inside event payloads shared between multiple domain packages (entitlement, billing, productcatalog). Each type implements Validate() to enforce required fields before payload construction.

## Patterns

**Inline Validate() on every exported struct** — Every struct in this package must implement Validate() error that checks all required fields. Callers must invoke Validate() before embedding the value in an event payload. No silent zero-value structs should propagate into event buses. (`f := models.FeatureKeyAndID{Key: featureKey, ID: featureID}
if err := f.Validate(); err != nil {
    return fmt.Errorf("invalid feature ref: %w", err)
}`)
**Cross-subsystem value types only** — Only add types here if they appear in event payloads used by two or more domain packages. Domain-specific structs belong in their own domain package. This package is imported by many subsystems; heavy additions create wide compilation coupling. (`// FeatureKeyAndID is used in both entitlement and billing event payloads — correct placement
// Invoice{} belongs in openmeter/billing — do not add here`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `models.go` | Declares FeatureKeyAndID (Key+ID pair for feature references) and NamespaceID (namespace identifier) — both are embedded in event payloads across entitlement and billing subsystems. | Do not import openmeter domain packages (billing, entitlement, customer, etc.) from this file — it is imported by all of them and circular imports will result. Keep types minimal: string fields + Validate() only. |

## Anti-Patterns

- Adding domain-specific types (Invoice, Subscription, Entitlement) to this package — they belong in their respective domain packages
- Skipping Validate() calls on values received in event payloads from this package
- Importing openmeter/* domain packages from models.go — this package must remain a leaf with no domain imports to avoid circular dependencies
- Adding business logic or computed fields to these value types — they are pure data containers

## Decisions

- **Separate package for cross-subsystem event value types rather than duplicating in each domain** — Both entitlement and billing packages need FeatureKeyAndID in their event payloads. A shared package prevents import cycles: neither billing nor entitlement imports the other, but both can import event/models.

<!-- archie:ai-end -->
