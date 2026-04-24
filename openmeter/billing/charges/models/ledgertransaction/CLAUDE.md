# ledgertransaction

<!-- archie:ai-start -->

> Provides the GroupReference and TimedGroupReference value types used across all charge sub-packages to reference ledger transaction groups. GroupReference carries a TransactionGroupID string; TimedGroupReference adds a timestamp for when authorization or settlement occurred.

## Patterns

**Nil-safe pointer helpers** — Use GetIDOrNull() to obtain *string and GetTimeOrNull() to obtain *time.Time from a potentially nil TimedGroupReference. Always use these helpers when passing values to Ent SetNillable* methods. (`SetNillableAuthorizedTransactionGroupID(paymentSettlement.Authorized.GetIDOrNull())`)
**Validate() returns GenericValidationError** — Both GroupReference.Validate() and TimedGroupReference.Validate() use models.NewNillableGenericValidationError(errors.Join(errs...)) so they return nil when the slice is empty. Callers should not special-case empty errs. (`return models.NewNillableGenericValidationError(errors.Join(errs...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ledger.go` | Complete package: GroupReference, TimedGroupReference structs with Validate() and nil-safe getters. | GetIDOrNull() on a nil *TimedGroupReference returns nil — always call on pointer receiver so nil-deref is safe. |

## Anti-Patterns

- Accessing TransactionGroupID directly without calling Validate() first
- Passing GroupReference by value to SetNillable methods instead of using GetIDOrNull()

<!-- archie:ai-end -->
