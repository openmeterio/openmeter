# Archie Scan Report
> Deep scan baseline (comprehensive) | 2026-06-04 08:43 UTC | 51,710 functions / 831,426 LOC analyzed | full-history run

## Architecture Overview

OpenMeter is a multi-tenant usage-metering and billing platform: it ingests CloudEvents usage data, aggregates it in ClickHouse, and drives entitlements, credit grants, a double-entry ledger, and invoice/charge billing through a versioned v1+v3 REST API. It is built as a **multi-binary Go modulith** — a single Go module whose business logic lives in ~35 layered domain packages under `openmeter/`, each splitting Service/Adapter interfaces (package root), concrete logic (`service/`), Ent/PostgreSQL persistence (`adapter/`), and HTTP translation (`httpdriver/` v1, `api/v3/handlers/` v3). Six runnable binaries under `cmd/` (server, sink-worker, billing-worker, balance-worker, notification-service, jobs) plus a separate Benthos collector module each compose a different subset of the domain tree through Google Wire provider sets concentrated in `app/common/`.

The binaries never call each other in-process or over HTTP; **all cross-binary communication is asynchronous over three name-prefix-routed Kafka topics** behind a Watermill `eventbus` facade, and cross-domain coupling is inverted through `ServiceHook`/`RequestValidator` registries wired as `app/common` provider side-effects to keep domain packages import-cycle-free leaves. Persistence is one PostgreSQL database managed by Ent schemas + Atlas migrations, a single shared append-only ClickHouse MergeTree events table written by the sink worker's strict three-phase flush (ClickHouse → Kafka offset → Redis dedupe), and optional Redis ingest deduplication.

The entire v1+v3 API surface and the Go/JavaScript/Python SDKs are generated from a **single TypeSpec source**. Billing uses tagged-union charge/invoice-line models (private discriminators, constructor-only construction) driven by stateless-backed state machines and a `LineEngine` registry; per-customer mutations are serialized with `pg_advisory` locks (`lockr`) held inside the ctx-bound Ent transaction; and the `credits.enabled` feature flag is guarded at four independent wiring layers via noop implementations.

## Health Scores

| Metric | Current | Previous | Trend | What it means |
|--------|--------:|---------:|------:|---------------|
| Erosion    | 0.5406 | 0.4433 | ▲ up | Share of code mass in heavy (high-CC/large) functions. Apparent jump is dominated by scope change (see note). |
| Gini       | 0.7708 | 0.6954 | ▲ up | Size inequality across files — a few very large generated stubs dominate. |
| Top-20%    | 0.8275 | — | — | Top 20% of files hold ~83% of the mass — heavy concentration in generated `api.gen.go`/`ent/db`. |
| Verbosity  | 0.1128 | 0.0667 | ▲ up | Exact line-clone ratio; rose mainly because generated code (Ent ORM, SDK stubs) was included this run. |
| LOC        | 831,426 | 403,320 | ▲ up | **Scope artifact:** the `--comprehensive` flag widened the scanned set to include large generated files (`api.gen.go` ≈ 970 KB, `ent/db/`, three SDKs). |

**Read the trend with care.** This is the first `--comprehensive` baseline; prior deep scans measured a narrower (default-depth) file set. The near-doubling of LOC and the erosion/gini/verbosity rises are overwhelmingly explained by generated code entering the measurement window, **not** by a real regression in hand-written code. The hand-written core remains well-bounded: the highest-CC *hand-written* function is CC 65, versus CC 1016 / 187 / 146 in generated `ent/db`. Treat this run as a new baseline for comprehensive-mode trending; do not compare its raw numbers against the four prior default-depth runs.

### Complexity Trajectory

Top hand-written complexity offenders (generated `ent/db`/`*.gen.go` excluded as noise):

- CC 65 — `openmeter/subscription/service/sync.go:28` `sync` — the subscription→spec reconciliation core; inherently branchy (patch routing) but the single hottest hand-written function.
- CC 61 — `openmeter/notification/eventhandler/webhook.go:30` `reconcileWebhookEvent` — webhook dispatch + reconcile loop.
- CC 59 — `openmeter/streaming/clickhouse/meter_query.go:108` `toSQL` — meter→ClickHouse SQL compiler (aggregation/group-by fan-out).
- CC 58 — `openmeter/app/stripe/httpdriver/webhook.go:40` `AppStripeWebhook` — Stripe webhook type switch.
- CC 47 — `openmeter/app/stripe/client/checkout.go:21` `CreateCheckoutSession`.
- CC 47 — `openmeter/productcatalog/ratecard.go:694` `Validate`.
- CC 43 — `openmeter/productcatalog/http/mapping.go:565` `AsPrice`.
- CC 41 — `openmeter/billing/adapter/invoice.go:119` `ListInvoices` (filter fan-out).

These cluster in genuinely high-fan-out compilation/validation/dispatch code (SQL builders, webhook switches, spec sync). None is a god-function emergency, but `subscription/service/sync.go:sync` and `meter_query.go:toSQL` are the two worth watching as features accrete.

## Findings

Ranked by severity, grouped by novelty. **Total drift surface this scan: 365 findings** — 37 deep architectural (4 error / 10 high / 13 warn / 10 medium-low) + 328 mechanical (81 pattern divergences, 234 dependency-declaration mismatches, 13 structural outliers, 0 naming/anti-pattern clusters). The comprehensive full-source review surfaced substantially more deep findings than the prior default-depth run (4 deep).

### NEW (first observed this scan)

**Errors**

1. **[error] Exported gathering-invoice mutator assumes the caller already holds the per-customer lock + tx.** `openmeter/billing/service/gatheringinvoicependinglines.go` — an exported customer-mutating `Service` method relies entirely on its caller (`InvoicePendingLines`) already holding the `lockr` advisory lock and the Ent tx; any direct external caller bypasses both `UpsertCustomerLock` and the SELECT-FOR-UPDATE serialization. *(abstraction_bypass; violates "Per-customer advisory lock inside an Ent transaction".)* Confidence 0.85.
2. **[error] Customer-scoped gathering-invoice mutation runs without the per-customer advisory lock.** `openmeter/billing/service/invoice.go` — a gathering-invoice mutation can race concurrent `CreatePendingInvoiceLines`/`InvoicePendingLines` on the same customer that DO take the lock. *(trade_off_undermined; "billing.Service.WithLock" serialization.)* Confidence 0.8.
3. **[error] `subscriptionsync` reconciler uses `time.Now()` instead of `clock.Now()`.** `openmeter/billing/worker/subscriptionsync/reconciler/reconciler.go` — `ReconcileSubscription` is the single outlier passing `time.Now()` as the sync as-of time, defeating `clock.FreezeTime` in deterministic billing reconciler tests. *(pattern_erosion.)* Confidence 0.85.
4. **[error] `HandleSubscriptionSyncEvent` passes `time.Now()` as the sync as-of timestamp.** `openmeter/billing/worker/subscriptionsync/service/sync.go` — diverges from the sibling handlers/rest of the file which use `clock.Now()`. *(pattern_erosion.)* Confidence 0.85.

**High**

5. **[high] Feature Ent adapter never rebinds to the ctx-bound transaction.** `openmeter/productcatalog/adapter/feature.go` — every feature write runs on the raw client and silently falls off any caller-supplied transaction (no `entutils.TransactingRepo`); `ArchiveFeature` is a multi-step read-then-write executed without a surrounding transaction. *(abstraction_bypass + trade_off_undermined; `ctx-001`/`ent-004`.)* Confidence 0.85.
6. **[high] Grant adapter does not honor the ctx-bound Ent transaction in its own method bodies.** `openmeter/credit/adapter/grant.go` — the only repo in `openmeter/credit/adapter` not wrapping in `TransactingRepo`; works today only because the connector manually opens a tx. *(abstraction_bypass.)* Confidence 0.8.
7. **[high] Subscription `Update`/`Delete`/`Continue` write multiple rows without the per-customer `pg_advisory_lock`.** `openmeter/subscription/service/service.go` — `Create` and `Cancel` lock, but `Update`/`Delete`/`Continue` do not, violating the documented "lock before any multi-row customer-scoped write" decision. *(decision_violation.)* Confidence 0.85.
8. **[high] Subscription workflow ops mutate without acquiring the per-customer lock.** `openmeter/subscription/workflow/service/subscription.go` — `EditRunning`, `Restore`, `AddAddon`, `ChangeAddonQuantity` delegate to unlocked `Service.Update`/`Delete`. *(decision_violation.)* Confidence 0.8.
9. **[high] Entitlement soft-delete and grant cleanup are not atomic.** `openmeter/credit/hook/entitlement_hook.go` — the `PreDelete` hook deletes the owner's grants on a separate autocommit connection while the entitlement delete + event publish run in their own tx; a crash between them orphans grants or loses the delete. *(trade_off_undermined; atomic multi-step mutation.)* Confidence 0.8.
10. **[high] Duplicate "no-delete-with-active-entitlements" guard, one in the wrong layer.** `openmeter/customer/service/hooks/entitlementvalidator.go` — the guard exists as both a `RequestValidator` and a `PreDelete` `ServiceHook`, both registered in `cmd/server`; the hook copy also violates the documented pre-mutation-guards-go-in-RequestValidator separation (`customer-001`). *(semantic_duplication + pattern_erosion.)* Confidence 0.8.
11. **[high] `pf_0010` realized — FK-less `LedgerCustomerAccount` has provisioning but no cleanup/integrity path.** `openmeter/ledger/resolvers/hooks.go` — the customer→ledger link table is wired on `PostCreate` but has no deletion/reconciliation guard the FK-less design depends on. *(pitfall_triggered.)* Confidence 0.8.
12. **[high] Stripe client `providerError` uses `context.Background()` on a ledger-relevant write.** `openmeter/app/stripe/client/appclient.go` — the live write-path instance of the `pf-008`/`f_0005` class; thread `ctx` through `providerError`. *(pitfall_triggered.)* Confidence 0.85. *(Also tracked as `f_0009` in the findings store — recurring.)*

**Warnings / Medium-Low (selected NEW)**

13. **[warn] `DeduplicatingCollector.Ingest` drops events when the deduplicator errors.** `openmeter/ingest/dedupe.go` — contradicts its documented "forward on `(false, err)`" data-loss-avoidance contract; on a Redis outage `redisdedupe.IsUnique` errors and events are silently dropped instead of forwarded. *(responsibility_leak — correctness.)* Confidence 0.8.
14. **[warn] `ingest.processEvent` stamps timestamp-less events with `time.Now()`.** `openmeter/ingest/service.go:80` — persisted as the ClickHouse event `time` used for meter aggregation, bypassing `clock.Now()`. *(pattern_erosion.)* Confidence 0.8.
15. **[warn] Billing auto-advancer / collect loop / charges advancer use `time.Now()` for eligibility cutoffs.** `openmeter/billing/worker/advance/advance.go`, `worker/collect/collect.go`, `worker/worker.go`, `billing/charges/worker/advance/advance.go` — a frozen test clock cannot control which invoices/charges are picked up. *(pattern_erosion — clock.Now() class.)* Confidence 0.8.
16. **[warn] Default-profile provisioning is check-then-act without a transaction.** `openmeter/billing/service/profile.go` — two concurrent provisioners can both pass the nil check and create competing default profiles. *(pitfall_triggered.)* Confidence 0.75.
17. **[warn] Billing profile domain errors fall through to HTTP 500.** `openmeter/billing/service/profile.go` — plain `fmt.Errorf` / unmatched `models.Generic*` errors bypass the billing `errorEncoder` mapping. *(responsibility_leak.)* Confidence 0.75.
18. **[warn] `creditgrant.go` provider returns `nil` instead of the established noop when disabled.** `app/common/creditgrant.go` — violates the "noop, never nil" (`di-001`) contract at the Wire layer. *(decision_violation.)* Confidence 0.8.
19. **[medium] Credit `CreateGrant`/`ResetUsageForOwner` hardcode `time.Minute` instead of the connector's configured `Granularity`.** `openmeter/credit/grant.go`, `openmeter/credit/balance.go` — latent: both happen to equal `time.Minute` today, but the configurable field is bypassed. *(pattern_erosion.)* Confidence 0.75.
20. **[medium] `balanceworker` batch recalculator never marks deleted entitlements in the high-watermark cache** (unlike the live worker) and uses `time.Now()` for domain timestamps. `openmeter/entitlement/balanceworker/recalculate.go`. *(pattern_erosion.)* Confidence 0.75.
21. **[medium] `subjectCustomerHook.PostDelete` calls `UpdateCustomer` without the re-entrancy guard.** `openmeter/customer/service/hooks/subjectcustomer.go` — risks customer↔subject hook re-entry when both reverse hooks are registered. *(pattern_erosion.)* Confidence 0.75.
22. **[medium] Feature connector publishes Watermill events outside the `transaction.Run` closure.** `openmeter/productcatalog/feature/connector.go` — a publish failure leaves the feature persisted but the event lost. *(pattern_erosion.)* Confidence 0.75.
23. **[medium] Portal authenticator loses the nil-vs-empty `AllowedMeterSlugs` distinction across the JWT boundary.** `openmeter/portal/authenticator/authenticator.go` — an empty allowlist may permit all meters instead of denying. *(trade_off_undermined.)* Confidence 0.7.
24. **[low] Two customer v1 handlers + llmcost reconciler use `time.Now()`.** `openmeter/customer/httpdriver/customer.go`, `openmeter/llmcost/sync/reconciler.go`. *(pattern_erosion.)* Confidence 0.7.

### RECURRING (previously documented, still present)

- **`f_0005` — `context.Background()` in application code severs Ent tx + OTel spans.** Re-confirmed this scan at `openmeter/server/server.go` (the canonical `pf-008` instance) and the Stripe client (`f_0009`). Mandated in AGENTS.md + pre-edit hooks. Confirmed 6 scans. Confidence 0.85.
- **`f_0009` — Stripe `appclient.go` ledger write on `context.Background()`** (see NEW #12 — re-surfaced with a concrete fix direction). Confirmed 3 scans. Confidence 0.85.
- **`f_0008` — `eventbus.GeneratePublishTopic` default-prefix fallback to `SystemEventsTopic`** silently misroutes a misnamed/new event family. Confirmed 5 scans. Confidence 0.85.
- **`f_0006` / `f_0010` — `cmd/billing-worker` does namespace-scoped provisioning on an implicit cross-binary boot-order contract** (`cmd/server` is the sole `namespace.Handler` registrant). Re-confirmed at `cmd/billing-worker/main.go`. Confirmed 5/3 scans. Confidence 0.7.
- **`f_0007` — Wire side-effect hook registration is invisible to the compile-time graph.** Confirmed 5 scans. Confidence 0.7.
- **`f_0003` — Notification payload versioning is implicit.** Confirmed 6 scans. Confidence 0.75.
- **`f_0011` — v3 customer `contains`/`ILIKE` filters lack a `pg_trgm` GIN index** (sequential-scan risk). Carried in findings store. Confidence 0.8.
- **`f_0012` / `f_0013` — billing `schema_level` dual-write cleanup debt; RawEvent ClickHouse column-sync discipline.** Carried in findings store. Confidence 0.8.

### RESOLVED / not re-surfaced

- The prior NEW finding **"flat-fee vs usage-based settlement-mode branching duplicated verbatim"** (`patchchargeusagebased.go`) was not re-flagged this scan — either consolidated or not re-surfaced; verify before closing.
- The prior NEW finding **"ledger account provisioning wraps `LockForTX` in `context.WithTimeout` (`lock-004`)"** was not re-flagged at the same site this scan; the ledger reviewer instead surfaced the FK-less cleanup gap (`pf_0010`) and a missing `Config.Validate()` nil-check. Re-confirm whether the `context.WithTimeout` path was changed.
- No findings were verified as fully **resolved**; all 10 findings-store entries were `keep` on the backward-check verifier (KEEP=10, DEMOTE=0, DROP=0).

### Data Architecture

`blueprint.data_models` is non-empty (**22 models across 4 stores**: `primary_postgres`, `clickhouse_events`, `redis_dedupe`, `kafka_topics`). Data-shaped findings this scan:

- **FK-less ledger integrity (`pf_0010`, now realized — NEW #11):** `LedgerCustomerAccount` has provisioning but no deletion/reconciliation path; the integrity burden the FK-less design places on application code is currently unmet on the customer-delete path.
- **Atomicity gaps:** entitlement soft-delete vs grant cleanup (NEW #9) and feature-event-publish-outside-tx (NEW #22) are the two write paths where a multi-store mutation is not transactionally bounded.
- **`f_0011` ILIKE customer filter** without a `pg_trgm` index remains the one active sequential-scan risk on the `customers` table.
- **No schema drift, orphan FK, or migration-without-model-update detected** — the Ent-schema-as-source-of-truth discipline and `transactions.ResolveTransactions`-only ledger construction held across every reviewed file. The in-progress billing `schema_level` dual-write (`f_0012`) remains tracked cleanup debt, not drift.

## Pitfalls (durable, carried in blueprint)

The blueprint carries **3 canonical pitfalls** this run plus the durable risk-classes from prior runs (the deep findings above are concrete instances of several). The recurring architectural-trap classes are: charges/feature/grant adapter `TransactingRepo` discipline (`ctx-001`), the `credits.enabled` four-layer noop guard (`di-001`), `eventbus` prefix-routing fallback (`f_0008`), Watermill silent-drop of unknown event types, Wire side-effect hook-registration invisibility (`f_0007`), `context.Background()` propagation severing (`pf-008`/`f_0005`), worker-namespace boot-order (`pf-006a`/`f_0006`), FK-less ledger/ClickHouse integrity (`pf_0010`), `pg_trgm` index-vs-filter gap (`f_0011`), and billing `schema_level` dual-write cleanup debt (`f_0012`). These are loaded by the AI reviewer on every plan approval and pre-commit.

## Architectural Health Assessment

| Dimension | Rating | Evidence |
|-----------|--------|----------|
| Separation of concerns | **Strong** | Strict `service`/`adapter`/`httpdriver` split holds across all 38 components; the structural outliers (`tools/migrate/migrations` 41 files, `pkg/models` 29 files) are inventories/primitives, not god-folders. The handful of responsibility leaks (billing profile error mapping, ingest dedupe drop) are localized, not systemic. |
| Dependency direction | **Strong** | Domain packages are import-cycle-free leaves; `app/common` is the sole inward composition point; `pkg/models` is the highest in-degree node (229) by design. The 234 "dependency violations" are blueprint-declaration coarseness (real imports exceed declared `depends_on`), not inverted edges. |
| Pattern consistency | **Adequate** | Tagged-union construction, eventbus routing, and the state machines are applied uniformly — but this comprehensive pass exposed two recurring *classes* of local drift: `time.Now()` vs `clock.Now()` (≥8 sites across billing workers, ingest, balanceworker, llmcost, customer httpdriver) and `TransactingRepo` omission in two adapters (feature, grant). Both are the kind of erosion that spreads if not gated. |
| Testability | **Strong** | Wire DI, adapter interfaces, `MockStreamingConnector`, and the `clock` abstraction make components isolable — the `time.Now()` findings are precisely the cases that erode deterministic billing tests and should be fixed to restore them. |
| Change impact radius | **Adequate** | Tagged unions + state machines localize charge/invoice changes, but the `credits` four-layer guard, Wire side-effect hook registration (`f_0007`), and the cross-binary namespace boot-order (`f_0006`) mean some changes ripple across non-obvious wiring/deploy layers. |

## Top Risks & Recommendations

1. **Per-customer locking is incomplete on subscription + gathering-invoice write paths (NEW, highest impact).** `subscription.Service` `Update`/`Delete`/`Continue`, the workflow ops, and the exported gathering-invoice mutators write multiple customer-scoped rows without the `pg_advisory` lock that `Create`/`Cancel` and the invoicing path take. *Action:* route every multi-row customer mutation through `billing.Service.WithLock` / `lockr.LockForTX`; *watch:* any new `Service` method writing >1 row per customer without a lock.
2. **`TransactingRepo` omitted in the feature and grant adapters (NEW).** Writes run on the raw client and silently fall off any caller tx — the exact `ctx-001` failure mode the codebase warns about. *Action:* wrap every method body in `entutils.TransactingRepo`/`...WithNoValue`; *watch:* `a.db.<Entity>` in an adapter without a `TransactingRepo` on the stack.
3. **`clock.Now()` erosion is spreading across the billing/usage workers (NEW class).** `time.Now()` now leaks into the auto-advancer, collect loop, subscriptionsync, ingest event-stamping, balanceworker, and llmcost — severing `clock.FreezeTime` determinism and (for ingest) writing wall-clock event times into ClickHouse. *Action:* mechanically replace with `clock.Now()` in `openmeter/billing/**`, `openmeter/ingest/**`, `openmeter/entitlement/balanceworker/**`; *watch:* any new `time.Now()` outside genuine wall-clock needs.
4. **Multi-store mutations that aren't transactionally atomic (NEW).** Entitlement-delete vs grant cleanup (separate autocommit connection) and feature-event-publish-outside-`transaction.Run` can leave half-applied state on a crash. *Action:* fold the cleanup/publish into the owning transaction closure.
5. **`context.Background()` write-path leaks + silent event misrouting + worker boot-order (RECURRING).** `f_0005`/`f_0009` (Stripe client + v1 validator error handler), `f_0008` (`EventName()` prefix fallback to `SystemEventsTopic`), and `f_0006`/`pf-006a` (billing-worker assumes `cmd/server` provisioned the namespace) are durable classes confirmed across 3–6 scans. *Watch:* `context.Background()` outside `main()`/shutdown; every new `EventName()` beginning with a registered `EventVersionSubsystem` prefix; fail-fast namespace precondition in workers.
6. **`DeduplicatingCollector` drops events on dedup error (NEW correctness risk).** A Redis outage causes silent event loss instead of forward-on-error. *Action:* forward the event on `(false, err)` per the documented contract.

## Semantic Duplication

The deep-drift pass surfaced **1 confirmed semantic-duplication group**: the "no-delete-with-active-entitlements" guard exists twice — as a `customer.RequestValidator` and as a `PreDelete` `ServiceHook` (`openmeter/customer/service/hooks/entitlementvalidator.go`), both registered in `cmd/server`, duplicating the entitlement-existence check. The `RequestValidator` is the canonical home (pre-mutation blocking guards belong there per `customer-001`); the `ServiceHook` copy should be removed. No other near-duplicate function groups were confirmed beyond the recurring settlement-mode branching noted under RESOLVED/not-re-surfaced (verify whether it was consolidated).

## Proposed Rules

Step 6 synthesis produced **146 rules total** (132 existing preserved byte-for-byte + 14 new), now in `.archie/rules.json` and rendered into `.claude/rules/enforcement/`. The 14 new rules fill blueprint coverage gaps: v3 `nullable.Nullable`/`apierrors` codegen discipline; `samber/lo` over local wrappers; four infrastructure rules (`.nvmrc` sync, golangci-lint v2 config, collector-as-separate-module, commitizen commits, Python SDK publish); two data-model contracts (`CustomerSubjects` FK-less link, `LedgerTransaction` balanced construction); three persistence-store contracts (Postgres-via-Ent/Atlas, Redis dedupe-cache-only, Kafka sole-cross-binary channel); and two pattern rules (`InvoicingApp` read-only snapshot, subscription-workflow orchestration). All are user-curatable via the Archie viewer.
