# Archie Scan Report
> Deep scan baseline | 2026-06-05 17:18 UTC | 51,789 functions / 834,052 LOC analyzed | baseline run (comprehensive depth)

## Architecture Overview

OpenMeter is a multi-binary Go backend organized as domain packages under `openmeter/`, each following a hand-rolled **service / adapter / connector** pattern: interfaces and value models are declared in the domain root package, concrete struct constructors live in nested `service/` and `adapter/` subpackages, and everything is wired together by Google Wire in `app/common`. Persistence is Ent ORM over a single shared generated PostgreSQL client (`openmeter/ent/db`), with cross-domain atomicity achieved through a context-ambient transaction seam (`entutils.TransactingRepo` / `WithTx` / `Self`) and per-customer serialization via transaction-scoped advisory locks (`lockr`). Event-time usage metering flows through Kafka into ClickHouse via streaming connectors, while an async Watermill-over-Kafka event bus drives the billing, notification, and balance workers.

Two HTTP API surfaces coexist by design: a legacy v1 surface assembled in `openmeter/server/router` from per-domain `httpdriver`/`httphandler` packages, and a newer AIP-style v3 surface in `api/v3` whose centralized handlers delegate to the same domain services. The entire API contract is authored once in TypeSpec (`api/spec`, legacy + aip packages) and code-generated to OpenAPI, Go server stubs (oapi-codegen), and JS/Python/Go SDKs — so the two surfaces and three SDKs cannot drift from the contract by hand.

The most load-bearing decisions are: the transaction-aware Ent repository (the seam enabling subscription→billing→charges→ledger to commit atomically over one client); explicit `qmuntal/stateless` finite state machines with external storage for the invoice and per-charge lifecycles; Wire compile-time DI with feature-gated concrete-or-noop provider swaps (credits, webhooks); the `ServiceHookRegistry` for synchronous in-process cross-domain reactions; and a polymorphic `Charge` parent row with idempotent `unique_reference_id`. The only import cycles detected are test-harness-induced (production/`_test` packages reaching the shared `test/billing` fixtures), not production cycles.

## Health Scores

| Metric | Current | Previous | Trend | What it means |
|--------|--------:|---------:|------:|---------------|
| Erosion    | 0.5408 | — | flat (baseline) | 54% of code mass sits in heavy/complex functions — high, but dominated by generated `ent/db` ORM code |
| Gini       | 0.7709 | — | flat (baseline) | Complexity is very unevenly distributed; a small set of files carries most of the weight |
| Top-20%    | 0.8275 | — | flat (baseline) | The top 20% of files hold 83% of complexity mass — concentration is high |
| Verbosity  | 0.1125 | — | flat (baseline) | Exact line-clone duplication is low (~11%); 93.8k duplicate lines, much in generated/SDK code |
| LOC        | 834,052 | — | flat (baseline) | Large monorepo across six binaries + three SDKs + collector module |

Read together, the numbers describe a large, healthy-but-heavy codebase whose raw erosion/gini/top-20 figures are inflated by the generated Ent client (`openmeter/ent/db`, ~407 files) and the generated SDKs — not by hand-written domain code. Hand-written complexity is concentrated in a few well-known orchestrators (subscription sync, invoice listing, balance history, webhook reconciliation) rather than spread as pervasive rot. Verbosity is genuinely low for a project this size. The signal to watch is concentration: the billing/charges/ledger cluster is both the heaviest and the most actively changed area, so that is where erosion will show up first.

### Complexity Trajectory
Generated code dominates the raw top of the CC distribution (`ent/db/runtime.go:105 init` CC=1016, plus `assignValues`/`sqlSave` on billing invoice tables CC=118–187) and is **not actionable** — it is regenerated from the Ent schema. The top hand-written offenders are:

- `openmeter/subscription/service/sync.go:28 sync` — **CC=65** (the subscription→target-state reconciliation core)
- `openmeter/notification/eventhandler/webhook.go:30 reconcileWebhookEvent` — **CC=61**
- `openmeter/streaming/clickhouse/meter_query.go:108 toSQL` — **CC=59** (ClickHouse query builder)
- `openmeter/app/stripe/httpdriver/webhook.go:40 AppStripeWebhook` — **CC=58** (inbound Stripe webhook dispatch)
- `openmeter/app/stripe/client/checkout.go:21 CreateCheckoutSession` — **CC=47**
- `openmeter/productcatalog/ratecard.go:694 Validate` — **CC=47** (accumulating ratecard validation)
- `openmeter/billing/adapter/invoice.go:119 ListInvoices` — **CC=41** (filter-rich list query)
- `openmeter/entitlement/metered/balance.go:109 GetEntitlementBalanceHistory` — **CC=41**

These are inherent-complexity hotspots (state reconciliation, query building, webhook routing) rather than accidental sprawl, but each is a single-point-of-failure worth guarding with tests and resisting further branching.

## Findings

Ranked by severity, grouped by novelty. 268 total drift findings this scan: 33 deep architectural (AI) + 235 mechanical (8 deep errors, 25 deep warns; 230 mechanical warns mostly DI-graph annotation noise).

### NEW (first observed this scan)

**Errors (decision/contract violations):**

1. **[error] currencies adapter silently writes outside the caller's transaction.** `openmeter/currencies/adapter/currencies.go` — all four methods (incl. `CreateCurrency`, `CreateCostBasis`) run against the outer non-tx `a.db` instead of the rebound `tx.db` inside the `TransactingRepo` closure, breaking the cross-domain atomicity guarantee `dec-tx-001` exists to provide. Rename the closure param to `repo` and use `repo.db`. Confidence 0.9.
2. **[error] Credit-purchase settlement bypasses the charge state machine (two of three settlement types).** `openmeter/billing/charges/creditpurchase/service/invoice.go` and `.../external.go` — status transitions mutate `charge.Status` directly + `adapter.UpdateCharge` instead of driving the `chargestatemachine` FSM, the exact anti-pattern the folder's CLAUDE.md calls out; illegal transitions and missed side-effects (patch accumulation) become possible. Confidence 0.85.
3. **[error] Data race on shared `err` in the billing collect fan-out.** `openmeter/billing/worker/collect/collect.go` `InvoiceCollector.All()` — goroutine-per-customer closures write a package/function-scoped `err` and send it over `errChan` while the drain loop also assigns it; under `-race` a real Write/Write race, and functionally an error can be lost or misattributed to the wrong customer. Declare `localErr` inside the closure. Duplicated verbatim in the advance package — fix both. Confidence 0.85.
4. **[error] Billing customer-lock upsert swallows DB errors.** `openmeter/billing/adapter/lock.go` `UpsertCustomerLock` returns `nil` for every non-`ErrNoRows` error. This row is the load-bearing primitive for per-customer billing serialization (`FOR UPDATE` is taken after it); a swallowed error means the caller proceeds on a false premise in a financial path. Return the error (prefer `errors.Is`). Trips the universal `decay-empty-catch` rule. Confidence 0.9.
5. **[error] `WithLockedNamespaces` mutates the shared Service in place.** `openmeter/billing/service/service.go` — violates the `ConfigService` wither contract (sibling `WithAdvancementStrategy` returns a clone): a pointer receiver mutates `fsNamespaceLockdown` and returns the same shared `*Service`, so namespace lockdown leaks globally and persists. Switch to a value receiver. Confidence 0.85.
6. **[error] v3 customer-billing handler orchestrates across domains instead of delegating.** `api/v3/handlers/customers/billing/update_billing.go` — the handler resolves the payment app, switches on app type (Stripe/CustomInvoicing/Sandbox), does Stripe field validation, builds per-app `CustomerData`, and upserts app data + billing override directly. v3 handlers must be thin delegators (`dec-api-surface-001`); this belongs in a billing/customer service. Confidence 0.85.
7. **[error] Duplicated app-type orchestration across two v3 billing endpoints.** `api/v3/handlers/customers/billing/update_billing_app_data.go` duplicates the same app-type switch, Stripe validation, per-app `CustomerData` construction, and `UpsertCustomerData` call as `update_billing.go`. Extract one shared service method; the duplication guarantees the endpoints drift. Confidence 0.85.

**Warnings (pattern erosion, pitfalls manifesting, trade-offs undermined):**

8. **[warn] Adapter helpers read/write via `a.db` outside `TransactingRepo`.** `openmeter/billing/charges/creditpurchase/adapter/funded_credit_activity.go`, `openmeter/ledger/account/adapter/subaccount.go` (`resolveOrCreateRoute`) — reads use the base client, so inside a caller's tx they read a stale/uncommitted-missing snapshot, contradicting each folder's own transaction-aware-adapter rule.
9. **[warn] `slog.Default()` instead of the injected logger.** `openmeter/billing/charges/usagebased/service/statemachine.go`, `openmeter/server/server.go` (request-logger middleware + construction errors) — violates the project's no-`slog.Default()` rule; thread the injected `*slog.Logger`.
10. **[warn] `time.Now()` instead of `clock.Now()` breaks frozen-time determinism.** `openmeter/billing/worker/subscriptionsync/service/sync.go` (`HandleSubscriptionSyncEvent`, while siblings use `clock.Now()`), `openmeter/billing/service/invoicecalc/gatheringrealtime.go` (detailed-line timestamps in the otherwise-pure calc pipeline).
11. **[warn] `panic()` on recoverable config error.** `app/config/telemetry.go` `GetSampler()` panics on an invalid trace sampler ratio — violates the no-panic-in-non-test rule; validate in `Validate()` or return `(Sampler, error)`.
12. **[warn] `SimulateInvoice` hand-derives failed status outside the FSM.** `openmeter/billing/service/invoice.go` — read-only so it never corrupts durable state, but duplicates the FSM's failed-transition semantics outside the single Permit-edge source of truth; will silently drift if failed-state derivation changes.
13. **[warn] Currencies & llmcost services skip the standard Config+Validate()+New constructor.** `openmeter/currencies/service/service.go`, `openmeter/llmcost/service/service.go` — positional args, no dependency validation, no injected logger, return concrete struct not interface; a nil dep surfaces as a later nil-pointer panic.
14. **[warn] v3 handlers omit the per-handler `apierrors.GenericErrorEncoder()`.** `api/v3/handlers/currencies/create.go`, `.../create_cost_basis.go`, `api/v3/handlers/subscriptions/subscriptionaddons/get.go` — domain errors fall to the default path instead of the uniform v3 RFC7807/status mapping the rest of the surface uses.
15. **[warn] v3 read/resolve logic leaks domain rules into handlers.** `api/v3/handlers/customers/billing/get_billing.go` (per-app-type mapping), `api/v3/handlers/customers/upsert.go` (key-preservation workaround with explicit FIXME), `api/v3/handlers/subscriptions/create.go` (customer/plan ID-or-key resolution with TODOs) — each admits the logic belongs in a service.
16. **[warn] Ledger annotation propagation gap.** `openmeter/ledger/chargeadapter/usagebased.go` `OnInvoiceUsageAccrued` commits the group without stamping charge annotations on each transaction input (folder anti-pattern requires both input AND group).
17. **[warn] Test decimal/clock conventions eroded in high-leverage credits/subscription fixtures.** `test/credits/base.go`, `test/credits/sanity_test.go`, `test/credits/rating_test.go` use boolean `decimal.Equal()` instead of `require.Equal` on `InexactFloat64()` (no diff on failure, in shared `BaseSuite` helpers); `test/subscription/scenario_firstofmonth_test.go` calls `clock.SetTime` with no `defer clock.ResetTime()`, leaking frozen time to later tests.

### Data Architecture

`blueprint.data_models` carries 74 documented models — schema drift warrants elevated attention. Four schema-shaped findings emerged this scan:

1. **[warn] FK-less denormalized routing columns lack application-level existence validation.** `openmeter/ent/schema/ledger_account.go` — `LedgerSubAccountRoute` stores `currency`, `tax_code` (as `TaxCode.Key`), `tax_behavior`, `features`, `cost_basis`, `credit_priority` as immutable literal columns with no FK and no resolver-side check that `tax_code`/`currency` resolve to live canonical rows. This is the documented FK-less-denormalized-routing pitfall manifesting; add a resolver existence check + reconciliation test. (problem: integrity enforced only by app code; evidence: `ledger_account.go:116-130` immutable literals, no edge; root_cause: import-cycle avoidance between ledger and tax/catalog aggregates; fix: validate at route-create, add reconciliation test.)
2. **[warn] Destructive column-type migration on a populated routing column.** `LedgerSubAccountRoute.features` jsonb→text[] produced a DROP+ADD migration that destroys existing feature data (down-migration loses it again). `atlas lint` permits destructive changes so it passed migrate-check; prefer a `USING`-cast/data-preserving conversion. Confirm no production routes carried features.
3. **[warn] Ledger entry/transaction tables carry soft-delete against the append-only invariant.** `openmeter/ent/schema/ledger_entry.go` mixes in `TimeMixin` (mutable `deleted_at`) and even indexes `deleted_at IS NULL`, contradicting the documented append-only accounting-source-of-truth invariant. Either drop soft-delete from these double-entry rows or update the documented invariant.
4. **[warn] Customer ILIKE filter prerequisite (pg_trgm GIN) still unmet.** `openmeter/ent/schema/customer.go` ships only btree indexes while the v3 customers list contains-filters compile to leading-wildcard `ILIKE`; if the v3 list handler is live, each contains request runs a full seq scan (+ a COUNT(*) seq scan). Land the pg_trgm GIN custom SQL migration per the standing TODO before exposing the filters.

### Mechanical Findings (summary)

- **Dependency annotations (178, warn):** Almost entirely `cmd/*` entrypoints "import X but do not declare it as a dependency" — an artifact of Wire pulling the full graph into each binary's `main`. Largely benign graph-annotation noise, not real inverted dependencies.
- **Pattern divergences (52, warn):** Per-folder convention deviations (e.g. a `watermill/driver` sibling lacking the options-struct-with-`Validate()` shape, `pkg/kafka/metrics` outliers, `test/app` harness-style splits). Mostly informational; worth a glance when touching those folders.
- **Structural outliers (5, info):** `openmeter/billing` (41 files) and `pkg/models` (28 files) flagged as god-folders vs sibling averages; the three `api/client/*` SDK dirs flagged for non-`.md` file mix (expected — generated SDKs).
- **Anti-pattern clusters: 0.**

### Semantic Duplication

The mechanical verbosity score (0.1125) only catches exact line clones. The AI pass surfaced confirmed near-duplicates the metric misses:
- **`update_billing.go` ↔ `update_billing_app_data.go`** (v3 customer billing) — same app-type switch + Stripe validation + per-app `CustomerData` build + upsert, reimplemented per endpoint. Canonical fix: one shared billing/customer service method both endpoints delegate to. (Finding #7.)
- **`collect.go` ↔ advance package fan-out** — the `InvoiceCollector.All()` goroutine error-handling block is duplicated verbatim into the advance worker, propagating the same `err`-race bug. Canonical fix: a shared per-customer fan-out helper. (Finding #3.)

No other broad semantic-duplication clusters were confirmed across the reviewed frontier (879 recently-changed source files).

## Proposed Rules

No new rules were synthesized in this resumed run (Step 6 rule synthesis completed in the original run before interruption; `.claude/rules/` already carries 25 enforcement/topic files, and per-folder CLAUDE.md guidance covers the FSM, transaction-aware adapter, logger-injection, and clock conventions that the deep findings above key against). The deep findings map cleanly onto existing rules — they are erosion against documented decisions, not gaps in the rule set. Candidate rule tightening to consider: (a) a grep-guard for `\bslog\.Default\(\)` outside `_test.go`/`main.go`; (b) a guard for adapter methods using `a.db` directly inside a `TransactingRepo`-wrapped struct; (c) a guard for v3 handlers omitting `apierrors.GenericErrorEncoder()`.

---

*Archie is now active. Architecture rules will be enforced on every code change. Run `$archie-deep-scan --incremental` after code changes to update the architecture analysis.*
