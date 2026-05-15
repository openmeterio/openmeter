# noop

<!-- archie:ai-start -->

> Provides no-op implementations of all ledger interfaces (ledger.Ledger, ledger.AccountResolver, ledger.BalanceQuerier, ledgeraccount.Service, namespace.Handler) used when credits.enabled=false; all methods return zero values and nil errors without touching any database.

## Patterns

**Compile-time interface assertions for every noop type** — Every noop type has a var _ <interface> = <type>{} assertion to guarantee it stays in sync when interface methods are added. Removing these assertions silently breaks the credits-disabled guarantee. (`var _ ledger.Ledger = Ledger{}
var _ ledger.AccountResolver = AccountResolver{}`)
**Value-type structs for trivial construction** — All noop types are value types (struct{} or struct with minimal fields) rather than pointers, making Wire providers return them with zero-value syntax without allocation. (`func NewLedgerNamespaceHandler() noop.NamespaceHandler { return noop.NamespaceHandler{} }`)
**Deterministic non-nil return values** — Methods return minimal valid non-nil objects (e.g. CustomerAccounts with all three account types populated) so callers can dereference without nil checks even when credits are disabled. (`return ledger.CustomerAccounts{FBOAccount: customerFBOAccount{...}, ReceivableAccount: ..., AccruedAccount: ...}, nil`)
**normalizeID / normalizeNamespace for empty-string safety** — Helper functions substitute a fallback string ('noop', 'noop-account') when namespace or ID is empty to prevent downstream panics in callers that call .ID() on returned objects. (`func normalizeID(id, fallback string) string { if id == "" { return fallback }; return id }`)
**accountTypeForRoute for structurally valid sub-accounts** — newSubAccount infers AccountType from Route fields (TransactionAuthorizationStatus present → Receivable, CreditPriority present → FBO, else Accrued) to produce sub-account objects that pass interface type checks. (`accountType := accountTypeForRoute(normalizedRoute)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `noop.go` | Single file containing all noop types and internal helper types. All interface compliance is asserted here. | When any ledger interface gains new methods, add matching stubs here immediately — the var _ assertions will cause compile failure if missed. |

## Anti-Patterns

- Removing the var _ interface assertion lines — they are the only compile-time proof the noop types stay compliant
- Adding any database calls or side-effects to noop methods — they must remain pure no-ops
- Returning nil for interface-typed fields inside CustomerAccounts or BusinessAccounts — callers dereference without nil checks
- Using this package when credits.enabled=true — app/common must route to real implementations

## Decisions

- **All noop types are value types (struct{}) rather than pointers.** — Simplifies Wire provider functions: return noop.Ledger{} and noop.AccountResolver{} without allocation or error handling.

## Example: Wire provider returning noop implementations when credits are disabled

```
// app/common/ledger.go
func NewLedgerAccountService(creditsConfig config.CreditsConfiguration, db *entdb.Client) (ledgeraccount.Service, error) {
    if !creditsConfig.Enabled {
        return noop.AccountService{}, nil
    }
    return ledgeraccount.NewService(db)
}
```

<!-- archie:ai-end -->
