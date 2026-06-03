# noop

<!-- archie:ai-start -->

> Provides no-op implementations of all ledger interfaces (ledger.Ledger, ledger.AccountResolver, ledger.BalanceQuerier, ledgeraccount.Service, namespace.Handler) used when credits.enabled=false; every method returns zero values and nil errors without touching any database.

## Patterns

**Compile-time interface assertions for every noop type** — Each noop type has a var _ <interface> = <type>{} assertion so it stays in sync when interface methods are added. Removing them silently breaks the credits-disabled guarantee. (`var _ ledger.Ledger = Ledger{}
var _ ledger.AccountResolver = AccountResolver{}`)
**Value-type structs for trivial construction** — All noop types are value types (struct{} or minimal-field structs), letting Wire providers return them with zero-value syntax and no allocation. (`func NewLedgerNamespaceHandler() noop.NamespaceHandler { return noop.NamespaceHandler{} }`)
**Deterministic non-nil return values** — Methods return minimal valid non-nil objects (e.g. CustomerAccounts with all three account types populated) so callers can dereference without nil checks. (`return ledger.CustomerAccounts{FBOAccount: ..., ReceivableAccount: ..., AccruedAccount: ...}, nil`)
**normalizeID / normalizeNamespace for empty-string safety** — Helpers substitute a fallback string ('noop', 'noop-account') when namespace or ID is empty to prevent downstream panics when callers call .ID(). (`func normalizeID(id, fallback string) string { if id == "" { return fallback }; return id }`)
**accountTypeForRoute for structurally valid sub-accounts** — newSubAccount infers AccountType from Route fields (TransactionAuthorizationStatus -> Receivable, CreditPriority -> FBO, else Accrued) so sub-accounts pass interface type checks. (`accountType := accountTypeForRoute(normalizedRoute)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `noop.go` | Single file containing all noop types, internal helpers, and every interface compliance assertion. | When any ledger interface gains methods, add matching stubs here immediately — the var _ assertions cause compile failure otherwise. |

## Anti-Patterns

- Removing the var _ interface assertion lines (the only compile-time proof noop types stay compliant)
- Adding any database calls or side-effects to noop methods (must remain pure no-ops)
- Returning nil for interface-typed fields inside CustomerAccounts or BusinessAccounts (callers dereference without nil checks)
- Using this package when credits.enabled=true (app/common must route to real implementations)

## Decisions

- **All noop types are value types (struct{}) rather than pointers.** — Simplifies Wire providers: return noop.Ledger{} and noop.AccountResolver{} without allocation or error handling.

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
