# testutils

<!-- archie:ai-start -->

> Shared test-harness package for the ledger domain: constructs concrete account/historical/resolver services from real adapters and spins up a fully migrated Postgres IntegrationEnv with provisioned customer and business sub-accounts. Its constraint is to build dependencies from underlying constructors (no app/common DI) to avoid import cycles.

## Patterns

**Deps built from concrete adapters** — InitDeps wires real adapters and services directly — accountadapter.NewRepo, accountservice.New, historicaladapter.NewRepo, historical.NewLedger, resolversadapter.NewRepo, resolvers.NewAccountResolver — never importing app/common wiring. (`accountService := accountservice.New(accountadapter.NewRepo(db), locker)`)
**Historical ledger uses DefaultValidator** — historical.NewLedger is constructed with routingrules.DefaultValidator as its RoutingValidator, so integration tests exercise the production routing rules. (`historical.NewLedger(historicalRepo, accountService, accountService, routingrules.DefaultValidator)`)
**IntegrationEnv full DB lifecycle** — NewIntegrationEnv freezes clock, inits a Postgres DB via omtestutils.InitPostgresDB, runs migrate.New(...).Up() with OMMigrationsConfig, creates a Customer row, then provisions customer + business accounts via the resolver. All cleanup is registered with t.Cleanup. (`clock.FreezeTime(now); t.Cleanup(clock.UnFreeze); migrator.Up()`)
**Sub-account accessor helpers per account type** — Env exposes typed helpers (FBOSubAccount, ReceivableSubAccount*, AccruedSubAccount*, WashSubAccount, EarningsSubAccount, BrokerageSubAccount, BreakageSubAccountWithCostBasis) that call GetSubAccountForRoute with the correct *RouteParams; tests resolve sub-accounts through these rather than constructing routes by hand. (`e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(ctx, ledger.CustomerFBORouteParams{Currency: e.Currency, CreditPriority: priority})`)
**Optional-param helper laddering** — Helper variants delegate to the most general form with defaults (ReceivableSubAccount → ReceivableSubAccountWithCostBasisAndStatus(t, nil, Open)); add new knobs at the base helper, not by duplicating the full chain. (`func (e *IntegrationEnv) ReceivableSubAccount(t *testing.T) ledger.SubAccount { return e.ReceivableSubAccountWithCostBasisAndStatus(t, nil, ledger.TransactionAuthorizationStatusOpen) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `deps.go` | InitDeps(db, logger) → Deps{AccountService, ResolversService, HistoricalLedger}; the canonical example of building ledger services from raw adapters + lockr without DI. | Requires an injected *slog.Logger (no slog.Default fallback) and a lockr.NewLocker. Reuse this rather than re-wiring ledger services elsewhere in tests, and keep it independent of app/common to avoid import cycles. |
| `integration.go` | IntegrationEnv struct + NewIntegrationEnv constructor and all typed sub-account accessor helpers plus SumBalance. | Freezes clock to 2026-01-01 and registers UnFreeze cleanup — assertions inheriting frozen time. SumBalance returns sum.Settled() (settled balance, not pending). Namespace is uniquified with UnixNano; customer row is created before resolver provisioning. |

## Anti-Patterns

- Importing app/common (the DI/wiring layer) to build ledger test dependencies — creates test-only import cycles; build from accountservice.New / resolvers.NewAccountResolver instead.
- Constructing sub-accounts by hand-building ledger.Route/*RouteParams instead of using the env's typed accessor helpers.
- Passing slog.Default() instead of omtestutils.NewDiscardLogger(t) into InitDeps.
- Calling clock.FreezeTime in a test that also uses NewIntegrationEnv without expecting the env's own freeze/UnFreeze cleanup.
- Skipping migrator.Up() / using a non-migrated DB — the resolver provisioning (CreateCustomerAccounts, EnsureBusinessAccounts) needs the full schema.

## Decisions

- **Test deps are assembled from concrete package constructors rather than the application wiring layer.** — Per AGENTS.md, building from repos/adapters/services/lockr keeps ledger testutils independent of app/common and prevents test-only import cycles.
- **IntegrationEnv provisions both customer and business accounts up front via the resolver.** — Most ledger lifecycle tests need FBO/Receivable/Accrued (customer) and Wash/Earnings/Brokerage/Breakage (business) sub-accounts available, so they are created once and exposed via typed helpers.

## Example: Stand up a ledger integration environment and resolve a sub-account balance

```
env := testutils.NewIntegrationEnv(t, "ledger-test")

fbo := env.FBOSubAccount(t, ledger.DefaultCustomerFBOPriority)
// ... post entries via env.Deps.HistoricalLedger ...
balance := env.SumBalance(t, fbo) // returns settled balance
require.Equal(t, float64(0), balance.InexactFloat64())
```

<!-- archie:ai-end -->
