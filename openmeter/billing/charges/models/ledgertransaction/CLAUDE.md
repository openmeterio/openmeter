# ledgertransaction

<!-- archie:ai-start -->

> Tiny shared value-type package providing GroupReference and TimedGroupReference — references to a ledger transaction group (with optional timestamp) used across charges to track payment authorization/settlement and credit realization linkage.

## Patterns

**Nil-safe optional-reference accessors** — GetIDOrNull / GetTimeOrNull are pointer-receiver methods that tolerate a nil receiver and an empty TransactionGroupID, returning nil instead of panicking — designed for direct use in Ent SetNillable* calls. (`func (r *GroupReference) GetIDOrNull() *string { if r == nil || r.TransactionGroupID == "" { return nil }; return &r.TransactionGroupID }`)
**Embedded timed variant** — TimedGroupReference embeds GroupReference and adds a required Time; its Validate delegates to the embedded Validate then requires a non-zero time. (`type TimedGroupReference struct { GroupReference; Time time.Time }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Sole file defining GroupReference, TimedGroupReference, their Validate and nil-safe Get*OrNull accessors. | Accessors must stay nil-safe — callers pass possibly-nil references straight into SetNillable* builders. |

## Anti-Patterns

- Dereferencing a TransactionGroupID without the nil/empty guard the GetIDOrNull helpers provide.
- Adding fields here that belong to a specific charge type — this package is intentionally minimal and broadly imported.

## Decisions

- **Keep ledger references as a dependency-light leaf package** — It is imported by nearly every charge sub-package and ledger adapter; minimal imports (only pkg/models) avoid import cycles.

<!-- archie:ai-end -->
