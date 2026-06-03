# ledgertransaction

<!-- archie:ai-start -->

> Provides GroupReference and TimedGroupReference value types used across all charge sub-packages to reference ledger transaction groups. GroupReference carries a TransactionGroupID string; TimedGroupReference adds the authorization/settlement timestamp.

## Patterns

**Nil-safe pointer helpers for Ent SetNillable* methods** — Use GetIDOrNull() for *string and GetTimeOrNull() for *time.Time from a possibly nil *TimedGroupReference. Always use these helpers when passing to Ent SetNillable* methods. (`SetNillableAuthorizedTransactionGroupID(paymentSettlement.Authorized.GetIDOrNull())`)
**Validate via NewNillableGenericValidationError** — Both GroupReference.Validate() and TimedGroupReference.Validate() use models.NewNillableGenericValidationError(errors.Join(errs...)), returning nil when the error slice is empty. Callers should not special-case empty errs. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Complete package: GroupReference and TimedGroupReference structs with Validate() and nil-safe pointer getters. | GetIDOrNull()/GetTimeOrNull() on a nil *TimedGroupReference return nil safely — always call on the pointer receiver so nil-deref is safe. |

## Anti-Patterns

- Accessing TransactionGroupID directly without calling Validate() first
- Passing GroupReference by value to SetNillable Ent methods instead of GetIDOrNull()
- Constructing TimedGroupReference with a zero Time value — Validate() rejects it

## Decisions

- **Separate GroupReference and TimedGroupReference types instead of inline string fields** — Ledger transaction group IDs appear across multiple charge sub-packages (creditrealization, payment, invoicedusage) with consistent validation needs; a shared type prevents validation drift.

<!-- archie:ai-end -->
