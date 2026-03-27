---
name: ledger
description: Work with the OpenMeter ledger package. Use when modifying ledger code, writing ledger tests, or debugging ledger issues.
user-invocable: false
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Ledger

Guidance for working with the OpenMeter ledger package (`openmeter/ledger/`).

## Package Structure

- `openmeter/ledger/` — core interfaces and primitives: `Ledger`, `Account`, `SubAccount`, `Querier`, routing, validation, account type definitions
- `openmeter/ledger/historical/` — current concrete ledger implementation; books immutable transactions and computes balances by summing entries
- `openmeter/ledger/account/` — account and sub-account domain types, posting addresses, route-backed sub-account identity
- `openmeter/ledger/account/service/` — account service; `New(repo, liveServices)` self-wires `SubAccountService` into `AccountLiveServices`
- `openmeter/ledger/account/adapter/` — ent repo adapter
- `openmeter/ledger/resolvers/` — `AccountResolver` implementation; provisions per-customer accounts and shared business accounts
- `openmeter/ledger/resolvers/adapter/` — ent repo for customer→account mapping
- `openmeter/ledger/transactions/` — transaction templates, resolution, and prioritized customer-credit collection helpers

## Business Domain Model

- `customer_fbo` currently means customer credit/stored value, not a literal regulated FBO bank account. It includes prepaid, promotional, externally purchased, and similar credit sources.
- `customer_receivable` tracks value the customer owes but has not yet paid.
- `customer_accrued` is a staging account for acknowledged usage/spend that is not yet recognized as earnings for reporting.
- Business accounts are namespace-scoped shared accounts:
  - `wash` represents the outside world / external cash boundary and is expected to run negative
  - `earnings` receives recognized revenue from accrued
  - `brokerage` is the business-side offset used in FX flows

## Routing Model

- Accounts express ownership and business purpose; sub-accounts are the concrete posting addresses.
- Sub-accounts are identified by canonical route dimensions stored in `ledger.Route` and encoded into a routing key.
- Active route dimensions today:
  - `currency` for all account types
  - `credit_priority` for customer credit (`customer_fbo`) routing and collection order
  - `cost_basis` for FX-related routing
- `tax_code` and `features` are wired through routing and query code but are deferred from the current business flows.
- Use account-specific route params instead of constructing generic `Route` values in higher-level domain code:
  - `CustomerFBORouteParams`
  - `CustomerReceivableRouteParams`
  - `CustomerAccruedRouteParams`
  - `BusinessRouteParams`

## Transaction Semantics

- Transaction templates in `openmeter/ledger/transactions/` encode posting mechanics, not settlement-mode orchestration.
- Current customer/business posting flows:
  - `IssueCustomerReceivableTemplate`: customer credit `+`, customer receivable `-`
  - `FundCustomerReceivableTemplate`: wash `-`, customer receivable `+`
  - `CoverCustomerReceivableTemplate`: customer credit `-`, customer receivable `+`
  - `TransferCustomerFBOToAccruedTemplate`: collect from prioritized customer credit sub-accounts into accrued
  - `TransferCustomerReceivableToAccruedTemplate`: receivable `-`, accrued `+`
  - `RecognizeEarningsFromAccruedTemplate`: accrued `-`, earnings `+`
  - `ConvertCurrencyTemplate`: customer credit and brokerage postings on both source and target legs
- `TransferCustomerFBOToAccruedTemplate` currently supports partial collection. If no value can be collected it returns `nil`.
- Longer-term settlement logic is expected to live above templates; collection logic will likely move out of `transactions/` into a dedicated package.

## Wiring Notes

- `account/service.New` takes `account.AccountLiveServices`; `Locker` is required for customer-account locking and `Querier` is required for balance lookups.
- The usual composition is account repo + locker + lazy/historical querier + resolver repo + historical repo.
- `historical.NewLedger(repo, accountService, locker)` is the concrete runtime ledger.
- `historical.Ledger.CommitGroup` validates balanced transactions, locks affected customer accounts, creates a transaction group, then books transactions.
- Current account locking in `CommitGroup` applies to customer FBO and customer receivable accounts. Balance-consistency validation is still largely TODO territory.
- `resolvers.AccountResolver` provisions:
  - three per-customer accounts (`customer_fbo`, `customer_receivable`, `customer_accrued`)
  - three lazily created shared business accounts per namespace (`wash`, `earnings`, `brokerage`)

## Testing Gotchas

- Ledger tests are Postgres-backed. Use real migrations, not bare schema creation, when tests rely on route/account/entry integrity.
- For direct `go test` runs, set `POSTGRES_HOST=127.0.0.1` so Postgres-backed tests are not skipped.
- `openmeter/ledger/testutils/integration.go` is the main integration fixture for the ledger domain. It sets up:
  - migrated Postgres schema
  - account service
  - account resolver
  - historical ledger
  - pre-created customer and business accounts
- Transaction tests in `openmeter/ledger/transactions/` are a good source of business-domain examples and expected balances.
- Historical adapter tests in `openmeter/ledger/historical/adapter/ledger_test.go` are the best reference for query/filter behavior and migration-backed setup patterns.

## Running Ledger Tests

Prefer direct command execution for ledger verification. Do not wrap commands in `sh -lc`, `bash -lc`, or similar helper shells when a direct invocation works. If a Nix shell is required, use `nix develop --impure .#ci -c env POSTGRES_HOST=127.0.0.1 go test ...` rather than an extra shell wrapper.

```bash
# Run all ledger tests
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/ledger/...

# Run specific sub-packages
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/ledger/historical/...
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/ledger/account/...
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/ledger/transactions/...
```
