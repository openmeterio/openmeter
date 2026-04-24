# noop

<!-- archie:ai-start -->

> No-op implementations of all ledger interfaces (ledger.Ledger, ledger.Querier, ledger.AccountResolver, ledgeraccount.Service, namespace.Handler) used when credits.enabled=false. All methods return zero values and nil errors without touching any database.

## Patterns

**Compile-time interface assertions for every type** — Each noop type has a var _ <interface> = <type>{} assertion to guarantee it stays in sync when interface methods change. (`var _ ledger.Ledger = Ledger{}
var _ ledger.AccountResolver = AccountResolver{}`)
**Deterministic noop return values** — Methods return minimal valid non-nil objects (e.g. CustomerAccounts with all three account types populated) so callers can dereference without nil checks. (`return ledger.CustomerAccounts{FBOAccount: customerFBOAccount{...}, ReceivableAccount: ..., AccruedAccount: ...}, nil`)
**normalizeID / normalizeNamespace for empty-string safety** — Helper functions substitute a fallback string ('noop', 'noop-account') when namespace or ID is empty to prevent downstream panics in callers that call .ID(). (`normalizeID(id, "noop-account")`)
**accountTypeForRoute for route-driven type inference** — newSubAccount infers the AccountType from the Route fields (TransactionAuthorizationStatus → Receivable, CreditPriority → FBO, else Accrued) to produce structurally valid sub-account objects. (`accountType := accountTypeForRoute(normalizedRoute)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `noop.go` | Single file containing all noop types: Ledger, AccountResolver, AccountService, NamespaceHandler, plus internal helper types (balance, subAccount, postingAddress, customerAccount, businessAccount) and constructors (newAccount, newSubAccount). | When ledger interfaces gain new methods, add matching noop stubs here or the var _ assertion will fail to compile. |

## Anti-Patterns

- Removing the var _ interface assertion lines — they are the only compile-time proof the noop stays compliant.
- Adding any database calls or side-effects to noop methods — they must remain pure no-ops.
- Returning nil for interface-typed fields in CustomerAccounts or BusinessAccounts — callers dereference without checking nil.
- Using this package when credits.enabled=true — app/common must route to the real implementations in that case.

## Decisions

- **All noop types are value types (struct{}) rather than pointers, making them trivially constructable with zero-value syntax.** — Simplifies Wire provider functions: return noop.Ledger{} and noop.AccountResolver{} without allocation.

<!-- archie:ai-end -->
