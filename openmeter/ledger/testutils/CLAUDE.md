# testutils

<!-- archie:ai-start -->

> Provides reusable test fixtures for ledger integration tests: a Deps builder that wires concrete account/resolver/historical-ledger adapters without importing app/common, and an IntegrationEnv that spins up a real Postgres DB, runs migrations, creates a test customer, and exposes typed sub-account helpers.

## Patterns

**Build deps from constructors, not Wire** — InitDeps wires accountadapter.NewRepo, accountservice.New, historicaladapter.NewRepo, resolversadapter.NewRepo, and resolvers.NewAccountResolver directly — never via app/common — to avoid import cycles. (`repo := accountadapter.NewRepo(db); svc := accountservice.New(repo, ledgeraccount.AccountLiveServices{Locker: locker})`)
**IntegrationEnv as the single test harness entry point** — All ledger integration tests call NewIntegrationEnv(t, prefix) to get a fully provisioned namespace, customer, and account set. Never replicate this setup inline. (`env := testutils.NewIntegrationEnv(t, "mytest")`)
**Use t.Context() throughout** — Every method on IntegrationEnv and every Ent call passes t.Context(), not context.Background(). (`db.Customer.Create()...Save(t.Context())`)
**Freeze clock in NewIntegrationEnv** — Time is frozen with clock.FreezeTime(now) and unfrozen with t.Cleanup(clock.UnFreeze) so temporal ledger queries are deterministic. (`clock.FreezeTime(now); t.Cleanup(clock.UnFreeze)`)
**Sub-account helpers delegate to routing rules** — FBOSubAccount, ReceivableSubAccount, AccruedSubAccount, WashSubAccount, etc. call GetSubAccountForRoute with typed params (CustomerFBORouteParams, CustomerReceivableRouteParams, BusinessRouteParams) — never construct SubAccount directly. (`env.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{Currency: e.Currency, CreditPriority: priority})`)
**SumBalance reads settled balance only** — SumBalance calls subAccount.GetBalance(t.Context()) and returns .Settled() — tests should not inspect pending/unsettled components via this helper. (`sum := env.SumBalance(t, subAccount) // returns alpacadecimal.Decimal of settled balance`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `deps.go` | Wires the three concrete ledger services (AccountService, ResolversService, HistoricalLedger) from raw Ent client + logger. Two accountservice.New calls are intentional: first builds internalAccountService without Querier (needed to construct historicalLedger), second rebuilds with Querier set to historicalLedger. | The two-phase accountservice.New pattern is load-bearing — do not collapse into one call or HistoricalLedger will have a nil Querier. |
| `integration.go` | Full integration harness: DB provisioning via omtestutils.InitPostgresDB, migrations via migrate.New+Up, customer row insertion via Ent, and customer+business account creation via ResolversService. Exposes per-account-type sub-account helpers. | Customer row is inserted directly via Ent (not customer.Service) because the test package must stay independent of app/common; do not switch to a higher-level service call that would introduce a cycle. |

## Anti-Patterns

- Importing app/common or any Wire provider set — causes import cycles and couples test helpers to the full DI graph
- Constructing ledger.SubAccount literals directly instead of calling GetSubAccountForRoute
- Using context.Background() instead of t.Context() in test helpers
- Skipping clock.FreezeTime in tests that depend on temporal ledger queries
- Calling migrate.Up more than once per test DB instance (idempotent but wasteful; use the shared env pattern)

## Decisions

- **Two-phase accountservice.New in InitDeps** — historical.Ledger requires an internalAccountService without a Querier to avoid a circular dependency; the real AccountService is then built with Querier set to the fully-constructed historicalLedger.
- **Direct Ent customer row insertion instead of customer.Service** — Keeps testutils independent from app/common and customer/service packages, preventing import cycles in test-only code per AGENTS.md guidance.
- **Typed sub-account helper methods per account type** — Wrapping GetSubAccountForRoute in named helpers (FBOSubAccount, ReceivableSubAccount, etc.) makes tests readable and ensures routing-rule params are always correct for each account class.

## Example: Set up a full ledger integration test

```
import (
    ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"
)

func TestMyLedgerFeature(t *testing.T) {
    env := ledgertestutils.NewIntegrationEnv(t, "my-feature")
    fbo := env.FBOSubAccount(t, 1)
    recv := env.ReceivableSubAccount(t)
    // post transactions via env.Deps.HistoricalLedger, then:
    bal := env.SumBalance(t, fbo)
    require.Equal(t, expectedDecimal, bal)
}
```

<!-- archie:ai-end -->
