# ledgertransaction

<!-- archie:ai-start -->

> Provides GroupReference and TimedGroupReference value types used across all charge sub-packages to reference ledger transaction groups. GroupReference carries a TransactionGroupID string; TimedGroupReference adds the authorization/settlement timestamp.

## Patterns

**Nil-safe pointer helpers for Ent SetNillable* methods** — Use GetIDOrNull() to obtain *string and GetTimeOrNull() to obtain *time.Time from a potentially nil *TimedGroupReference. Always use these helpers when passing values to Ent SetNillable* methods. (`SetNillableAuthorizedTransactionGroupID(paymentSettlement.Authorized.GetIDOrNull())`)
**Validate returns GenericValidationError via NewNillableGenericValidationError** — Both GroupReference.Validate() and TimedGroupReference.Validate() use models.NewNillableGenericValidationError(errors.Join(errs...)) which returns nil when the error slice is empty. Callers should not special-case empty errs. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Complete package: GroupReference and TimedGroupReference structs with Validate() and nil-safe pointer getters. | GetIDOrNull() on a nil *TimedGroupReference returns nil safely — always call on pointer receiver so nil-deref is safe. GetTimeOrNull() similarly returns nil for nil receiver. |

## Anti-Patterns

- Accessing TransactionGroupID directly without calling Validate() first
- Passing GroupReference by value to SetNillable Ent methods instead of using GetIDOrNull()
- Constructing TimedGroupReference with a zero Time value — Validate() will reject it

## Decisions

- **Separate GroupReference and TimedGroupReference types instead of inline string fields** — Ledger transaction group IDs appear in multiple charge sub-packages (creditrealization, payment, invoicedusage) with consistent validation needs; a shared type prevents drift in validation logic across sub-packages.

<!-- archie:ai-end -->
