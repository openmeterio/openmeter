---
name: ledger
description: Work with the OpenMeter ledger package. Use when modifying ledger code, writing ledger tests, or debugging ledger issues.
user-invocable: false
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Ledger

Guidance for working with the OpenMeter ledger package (`openmeter/ledger/`).

## Package Structure

- `openmeter/ledger/` — interfaces: Ledger, Account, SubAccount, Querier, DimensionResolver, AccountResolver
- `openmeter/ledger/historical/` — concrete Ledger impl; `NewLedger(repo, accountService, locker)`
- `openmeter/ledger/account/` — Account/SubAccount domain types, DimensionResolver
- `openmeter/ledger/account/service/` — `New(repo, locker, querier)` where querier can be nil
- `openmeter/ledger/account/adapter/` — ent repo adapter
- `openmeter/ledger/resolvers/` — AccountResolver impl; `NewService(ServiceConfig{AccountService, Repo})`
- `openmeter/ledger/resolvers/adapter/` — ent repo for customer→account mapping
- `openmeter/ledger/transactions/` — template resolution; `ResolveTransactions(ctx, deps, scope, templates...)`

## Wiring Notes

- `account/service.New` needs `*lockr.Locker` (required for CommitGroup lock path); querier optional (nil OK if GetBalance not called)
- `historical.Ledger` needs account service wired for `lockAccountsForTransactionInputs` in CommitGroup
- `DimensionKey.Validate()` only accepts `"currency"` currently — CRD uses key=currency value=CRD

## Testing Gotchas

- For Postgres-backed tests run directly with `go test` (not via Make), ensure the PG env is set (e.g. `POSTGRES_HOST=localhost`). If tests rely on DB constraints/triggers, run real migrations (not just `Schema.Create`).
- For `CreateEntries`, prefer Ent `CreateBulk` path (not driver-level `ExecContext`) and map DB trigger violations to validation errors (`ledger_entries_dimension_ids_fk` / SQLSTATE 23503 path).
- When adding/adjusting historical adapter tests, follow existing repo conventions (`NewTestEnv`, `DBSchemaMigrate`, `t.Cleanup(env.Close)`), and use migration-backed schema so trigger-based constraints are actually exercised.

## Running Ledger Tests

```bash
# Run all ledger tests
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/ledger/...

# Run specific sub-package
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/ledger/historical/...
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/ledger/account/...
```
