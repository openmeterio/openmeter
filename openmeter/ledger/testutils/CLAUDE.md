# testutils

<!-- archie:ai-start -->

> Reusable test fixtures for ledger integration tests: Deps wires concrete account/resolver/historical-ledger adapters directly (no app/common), and IntegrationEnv spins up a real Postgres DB, runs migrations, creates a test customer, provisions all account types, and exposes typed sub-account helpers.

## Patterns

**Build deps from constructors, not Wire** — InitDeps wires accountadapter.NewRepo, accountservice.New, historicaladapter.NewRepo, resolversadapter.NewRepo, and resolvers.NewAccountResolver directly — never via app/common — to avoid import cycles. (`repo := accountadapter.NewRepo(db); accountService := accountservice.New(repo, locker); historicalLedger := historical.NewLedger(historicalRepo, accountService, accountService, routingrules.DefaultValidator)`)
**IntegrationEnv as the single test harness entry point** — All ledger integration tests call NewIntegrationEnv(t, namespacePrefix) to get a provisioned namespace, customer, and account set. Never replicate DB setup, migration, or account provisioning inline. (`env := testutils.NewIntegrationEnv(t, "my-feature"); fbo := env.FBOSubAccount(t, 1)`)
**Use t.Context() throughout** — Every Ent call and service method in IntegrationEnv and Deps passes t.Context(), not context.Background(), tying cancellation and DB connections to the test harness. (`db.Customer.Create().SetNamespace(namespace).SetID(customerID.ID).SetName("Test Customer").Save(t.Context())`)
**Freeze clock in NewIntegrationEnv** — Time is frozen with clock.FreezeTime(now) and t.Cleanup(clock.UnFreeze) so temporal ledger queries (balance snapshots, period windows) are deterministic. (`clock.FreezeTime(now); t.Cleanup(clock.UnFreeze)`)
**Typed sub-account helpers per account type** — FBOSubAccount, ReceivableSubAccount, AccruedSubAccount, WashSubAccount, EarningsSubAccount, BrokerageSubAccount call GetSubAccountForRoute with the correct typed params. Never construct SubAccount directly. (`subAccount, _ := e.CustomerAccounts.FBOAccount.GetSubAccountForRoute(t.Context(), ledger.CustomerFBORouteParams{Currency: e.Currency, CreditPriority: priority})`)
**SumBalance reads settled balance only** — SumBalance calls HistoricalLedger.GetSubAccountBalance and returns .Settled(). Tests needing pending/unsettled components must call GetSubAccountBalance directly. (`sum := env.SumBalance(t, fboSubAccount) // settled balance as alpacadecimal.Decimal`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `deps.go` | Wires the three concrete ledger services (AccountService, ResolversService, HistoricalLedger) from a raw *entdb.Client and *slog.Logger; exported Deps struct exposes all three for direct test use. | The single accountservice.New call before historicalLedger construction is load-bearing — historical.NewLedger receives accountService as both the internal service and Querier. Do not collapse or reorder these calls. |
| `integration.go` | Full integration harness: DB via omtestutils.InitPostgresDB, migrations via migrate.New+Up, customer row inserted via raw Ent, account provisioning via ResolversService.CreateCustomerAccounts and EnsureBusinessAccounts. | Customer row is inserted via raw Ent (not customer.Service) so the package stays independent of app/common and customer/service. Do not switch to customer.Service here. |

## Anti-Patterns

- Importing app/common or any Wire provider set — causes import cycles and couples helpers to the full DI graph.
- Constructing ledger.SubAccount literals directly instead of the typed IntegrationEnv helper methods.
- Using context.Background() instead of t.Context() — leaks connections and severs cancellation.
- Skipping clock.FreezeTime in tests with temporal ledger queries — non-deterministic balance windows.
- Calling migrate.Up more than once per IntegrationEnv instance — reuse the single env per test.

## Decisions

- **Single accountservice.New call before historicalLedger construction (no two-phase pattern).** — historical.NewLedger receives accountService as both its internal service and Querier; one instance passed twice is correct and avoids circular dependency.
- **Direct Ent customer row insertion instead of customer.Service in NewIntegrationEnv.** — Keeps testutils independent of app/common and customer/service, preventing import cycles per the test-003 enforcement rule.
- **Typed sub-account helper methods per account type.** — Wrapping GetSubAccountForRoute in named helpers ensures routing-rule params are always correct per account class and keeps tests readable.

## Example: Set up and use a full ledger integration test

```
import ledgertestutils "github.com/openmeterio/openmeter/openmeter/ledger/testutils"

func TestMyLedgerFeature(t *testing.T) {
  env := ledgertestutils.NewIntegrationEnv(t, "my-feature")
  fbo := env.FBOSubAccount(t, 1)
  recv := env.ReceivableSubAccount(t)
  // post transactions via env.Deps.HistoricalLedger ...
  bal := env.SumBalance(t, fbo)
  require.Equal(t, expectedDecimal, bal)
}
```

<!-- archie:ai-end -->
