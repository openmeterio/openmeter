# Archie Scan Report
> Deep scan baseline | 2026-06-02 20:43 UTC | 10,381 functions / 403,320 LOC analyzed | full deep-scan run

## Architecture Overview

OpenMeter is a multi-tenant usage-metering and billing platform: it ingests CloudEvents, aggregates them into meters in ClickHouse, and drives entitlement balances and financial billing on top of that usage data. It is a **multi-binary Go modulith** — a single Go module organized as a strict layered `service` / `adapter` / `httpdriver` split across ~35 domain packages under `openmeter/`, compiled into six runnable binaries (HTTP API server, sink/balance/billing workers, notification-service, jobs CLI) plus a separate `benthos-collector` module. Each binary composes ~40 domain services through Google Wire provider sets concentrated in `app/common/`, keeping the domain packages import-cycle-free leaves.

Synchronous request handling flows through a **TypeSpec-generated v1+v3 HTTP API** into domain services that persist to PostgreSQL via Ent and `entutils.TransactingRepo`. All cross-binary communication is asynchronous over **three prefix-routed Kafka topics** behind a Watermill `eventbus` facade. The usage path streams events from the benthos collector through the ingest `Collector` into Kafka, where the sink-worker performs a strict three-phase flush (ClickHouse insert → Kafka offset commit → Redis dedupe) and the balance-worker recalculates entitlement grant burn-down.

Billing correctness is enforced through **tagged-union domain models** (`Charge`/`ChargeIntent`/`InvoiceLine` with private discriminators and constructor-only construction), stateless-library invoice/charge **state machines**, per-customer `pg_advisory_xact_lock` serialization via `lockr`, and a **double-entry ledger**. Cross-domain coupling is inverted via ServiceHook and RequestValidator registries registered as Wire provider side-effects. The `credits.enabled` feature flag is guarded at four independent wiring layers. Stripe and Svix are the primary external billing and webhook integrations.

## Health Scores

| Metric | Current | Previous | Trend | What it means |
|--------|--------:|---------:|------:|---------------|
| Erosion    | 0.4433  | 0.4453  | down (−0.0020) | Structural fragmentation stable-to-improving; no new erosion. |
| Gini       | 0.6954  | 0.6957  | flat  | Complexity stays concentrated in the top quartile of functions. |
| Top-20%    | 0.7375  | 0.7378  | flat  | The top 20% of functions own ~74% of codebase mass — unchanged. |
| Verbosity  | 0.0667  | 0.0609  | up (+0.0058) | Exact-line duplication ticked up but stays under 7%; 26,912 duplicate lines, much of it generated SDK + parallel test fixtures. |
| LOC        | 403,320 | 379,517 | +6.3% | Codebase grew ~23,800 lines this window, concentrated in `openmeter/billing/charges`, `openmeter/ledger`, and `subscriptionsync`. |

Interpreted together: the codebase remains **structurally stable** (erosion, gini, top-20% all flat-to-improving) while continuing to grow. This window's growth shifted from the v3 API handler tree (prior window) to the **billing charges, ledger, and subscription-sync** subsystems — the same areas that carry this scan's new findings. The verbosity uptick plus this scan's semantic-duplication finding point at copy-evolved settlement-mode logic in the charges patch reconciler rather than at architectural decay.

### Complexity Trajectory

CC distribution: `1-2`: 6,198 · `3-5`: 2,371 · `6-10`: 1,159 · `11-20`: 537 · `21-50`: 111 · `51-100`: 5 · `101+`: 0.
**653 functions at CC ≥ 11; 5 above CC 51, none above 100.** Risk concentration is unchanged from the prior baseline — the same hotspots dominate:

| Function | Location | CC | SLOC |
|----------|----------|---:|-----:|
| `sync` | `openmeter/subscription/service/sync.go:28` | 65 | 270 |
| `reconcileWebhookEvent` | `openmeter/notification/eventhandler/webhook.go:30` | 61 | 344 |
| `toSQL` | `openmeter/streaming/clickhouse/meter_query.go:108` | 59 | 215 |
| `AppStripeWebhook` | `openmeter/app/stripe/httpdriver/webhook.go:40` | 58 | 354 |
| `CreateCheckoutSession` | `openmeter/app/stripe/client/checkout.go:21` | 47 | 169 |
| `Validate` | `openmeter/productcatalog/ratecard.go:678` | 47 | 148 |

These are large but established orchestration/serialization functions; none is new this scan. They concentrate risk (a bug here has wide blast radius) and resist unit isolation — worth characterization-test coverage before any refactor.

## Findings

Ranked by severity, grouped by novelty. Total drift surface this scan: **254 findings** (4 deep architectural + 250 mechanical: 81 pattern divergences, 156 dependency-declaration mismatches, 13 structural outliers).

### NEW (first observed this scan)

**Errors**

_None._

**Warnings**

1. **[warn] `subscriptionsync` resync uses `time.Now()` instead of the project-wide `clock.Now()`.** In `openmeter/billing/worker/subscriptionsync/service/sync.go`, a sync event handler reads `time.Now()` while every sibling line in the same file uses `clock.Now()` — severing deterministic time freezing and making billing-sync timing untestable under `clock.SetTime`. Confidence 0.85.

2. **[warn] Reconciler as-of timestamp uses `time.Now()` not `clock.Now()`.** `openmeter/billing/worker/subscriptionsync/reconciler/reconciler.go` passes `time.Now()` as the per-subscription resync as-of timestamp while every other time read in the file uses `clock.Now()`; the inconsistency breaks deterministic-time reconciliation tests and can desync the as-of cutoff from clock-frozen scheduling windows. Confidence 0.85.

3. **[warn] Flat-fee and usage-based charge-patch collections duplicate settlement-mode branching verbatim.** `openmeter/billing/worker/subscriptionsync/service/reconciler/patchchargeusagebased.go` copies the entire AddShrink/AddExtend settlement-mode control flow from the flat-fee collection; only the charge-type getter and intent constructor differ. The reconciler `CLAUDE.md` sanctions overriding *per-type construction only*, so this shared control flow should be a generic base method to prevent the two copies drifting apart as settlement modes evolve. (`semantic_duplication`) Confidence 0.8.

4. **[warn] `f_0011` — v3 customer list exposes case-insensitive `contains` filters with no matching index.** The wired v3 customers list endpoint (`openmeter/customer/adapter/customer.go:59`) applies `ILIKE` filters on name/primary-email; the backing columns carry only a btree index, so the filter forces a sequential scan on the customers table as tenant size grows. New this scan (seen 1×). Confidence 0.8.

5. **[warn] Ledger account provisioning wraps `LockForTX` in `context.WithTimeout` — sanctioned-but-conflicting with pitfall `lock-004`.** `openmeter/ledger/resolvers/account.go` bounds the `pg_advisory_xact_lock` acquisition with `context.WithTimeout`, which the documented `lock-004` pitfall warns destroys the pgx connection on deadline (corrupting the in-flight Ent transaction) rather than merely timing out the lock wait. The local `CLAUDE.md` endorses the 5s bounded wait, so this is a deviation from the global advisory-lock rule worth reconciling (move to a pg-side lock timeout, or formally exempt this path). (`pitfall_triggered`) Confidence 0.8.

### RECURRING (previously documented, still present)

6. **[warn] `f_0005` — `context.Background()` in application code severs Ent tx propagation + OTel spans.** Manifest in the v1 `OapiRequestValidatorWithOptions` ErrorHandler and the Stripe app client (`f_0009`). Mandate documented in AGENTS.md and pre-edit hooks. Confirmed in 5 scans. Confidence 0.85.

7. **[warn] `f_0009` — `openmeter/app/stripe/client/appclient.go` performs a ledger-relevant write with `context.Background()`.** `UpdateAppStatus` runs through the Ent adapter outside the caller's transaction and trace — the live write-path instance of `f_0005`'s class. Confirmed in 2 scans. Confidence 0.8.

8. **[warn] `f_0003` — Notification event payload versioning is implicit.** Payload version constants live alongside the payload struct in `openmeter/notification/`; no machinery migrates old payloads when a struct changes. Confirmed in 5 scans. Confidence 0.75.

9. **[warn] `f_0006` — `cmd/server` is the only binary that registers `namespace.Handler` implementations.** Other binaries reaching `initNamespace` will not have ClickHouse/Kafka/Ledger handlers wired by default. Confirmed in 4 scans. Confidence 0.7.

10. **[warn] `f_0007` — Wire-generated provider sets concentrate cross-domain hook registration as side-effects.** A binary that omits a provider silently drops its hook with no compile error. Confirmed in 4 scans. Confidence 0.7.

11. **[warn] `f_0008` — `eventbus.GeneratePublishTopic` routes by an `EventName()` string-prefix switch with a default fallback to `SystemEventsTopic`.** A misnamed or new event family silently misroutes instead of erroring. Confirmed in 4 scans. Confidence 0.85.

12. **[warn] `f_0010` — `cmd/billing-worker/main.go` calls `EnsureBusinessAccounts` and `SandboxProvisioner` directly in `main.go`.** Startup-orchestration logic the architecture otherwise keeps out of `cmd/*/main.go`; correct today but a boundary the multi-binary trade-off depends on. Confirmed in 2 scans. Confidence 0.7.

### RESOLVED / DEMOTED

- **`f_0001` — Charges adapter helpers accepting a raw `*entdb.Client` can bypass the ctx-bound transaction.** **Demoted** this run by the verifier: the cited call site (`adapter.go:52`) is the correct `Tx()` interface implementation, not an un-wrapped helper. Remains a durable pitfall (`pf_0001`); the discipline is still mandated, just not currently firing at a quotable call site.
- **`f_0002` — Credits-disabled deployments can still write to the ledger if any of four wiring layers forgets to guard.** **Demoted** (risk class, no firing call site). Carried in the blueprint as `pf_0002`.
- **`f_0004` — Sequential timestamped Atlas migrations + `atlas.sum` chain produce predictable merge collisions.** **Demoted** to risk-class; the `/rebase` skill encodes the recovery procedure. Carried as `pf_0003`.
- No findings were fully **resolved** this run.

### Data Architecture

`blueprint.data_models` is non-empty (17 models across 4 stores: PostgreSQL primary, ClickHouse analytics, Redis cache/dedupe, Kafka queue). Data-shaped findings this scan:

- **`f_0011` (above)** — the `ILIKE` customer filter without a supporting index is the one active data-path finding (sequential-scan risk on the `customers` table). All other models track their migrations cleanly; **no schema drift, orphan FK, or migration-without-model-update was detected** — a positive signal worth recording. The double-entry ledger (`transactions.ResolveTransactions` as the sole construction path) and the Ent-schema-as-source-of-truth discipline held across every reviewed file.

## Pitfalls (durable, carried in blueprint)

The blueprint carries **11 architectural pitfalls** covering: charges adapter `TransactingRepo` discipline (`pf_0001`), the `credits.enabled` four-layer guard (`pf_0002`), Atlas migration collisions (`pf_0003`), eventbus prefix-routing fallback, Watermill silent-drop of unknown event types, cross-domain hook registration invisibility to Wire, `context.Background()` propagation severing, plus three new data-shaped classes (unbounded `ILIKE` scan, FK-less/migration-less integrity, in-progress billing-schema cleanup debt). These are durable — they describe *classes* of problem rooted in architectural decisions and are loaded by the AI reviewer on every plan approval and pre-commit.

## Architectural Health Assessment

| Dimension | Rating | Evidence |
|-----------|--------|----------|
| Separation of concerns | **Strong** | Strict `service`/`adapter`/`httpdriver` split enforced across all 23 components; one structural outlier (`openmeter/ledger/transactions` at 23 files vs sibling avg 4) is the only god-folder signal. |
| Dependency direction | **Strong** | Domain packages are import-cycle-free leaves; `app/common` is the sole inward composition point. The 156 "dependency violations" are blueprint-declaration coarseness (actual imports exceed declared `depends_on`), not inverted dependencies. |
| Pattern consistency | **Adequate** | Tagged-union construction, TransactingRepo, eventbus routing, and lockr are applied uniformly — but this scan found 3 local deviations (two `time.Now()` vs `clock.Now()`, one duplicated settlement-mode branch) in the actively-growing subscriptionsync subsystem. |
| Testability | **Strong** | DI via Wire, adapter interfaces, MockStreamingConnector, and the `clock` abstraction make components isolable — the two `time.Now()` findings are precisely the cases that erode this and should be fixed to restore deterministic billing tests. |
| Change impact radius | **Adequate** | Tagged unions + state machines localize charge/invoice changes, but the credits four-layer guard and Wire side-effect hook registration mean some changes ripple across non-obvious wiring layers (the recurring `f_0006`/`f_0007` class). |

## Top Risks & Recommendations

1. **Deterministic-time erosion in subscription-sync (NEW, actively growing area).** `time.Now()` leaking into `sync.go` and `reconciler.go` breaks clock-frozen billing tests. *Watch:* any new `time.Now()` in `openmeter/billing/**` — route through `clock.Now()`.
2. **Advisory-lock + `context.WithTimeout` conflict in ledger provisioning (`lock-004` class).** A deadline cancel can corrupt the in-flight Ent transaction. *Action:* replace with a pg-side lock timeout (`pgdriver.WithLockTimeout`) or formally exempt and document the path.
3. **`context.Background()` write-path leaks (`f_0005`/`f_0009`, recurring 5×).** Silent loss of transaction + trace in the Stripe client and v1 validator error handler. *Watch:* any `context.Background()`/`context.TODO()` outside `main()`/graceful-shutdown.
4. **Settlement-mode logic duplication in charge patch reconciler (NEW).** Two verbatim copies of AddShrink/AddExtend branching will drift as settlement modes evolve. *Action:* extract a generic base method; keep only per-type construction overridden.
5. **Silent event misrouting via `EventName()` prefix fallback (`f_0008`, recurring 4×).** A new/misnamed event family defaults to `SystemEventsTopic`. *Watch:* every new event type's `EventName()` must begin with a registered `EventVersionSubsystem` prefix.

## Proposed Rules

Step 6 synthesis produced **132 rules** in `.archie/rules.json` (merged with the prior set), rendered into `.claude/rules/enforcement/` (12 by-topic files + index + universal). New deep-scan coverage this run spans the four-layer credits guard, tagged-union construction, TransactingRepo discipline, eventbus routing, lockr usage, and the data-contract rules for the 17 models. Review and curate adopt/reject through the viewer's Rules section.

---
**Archie is now active. Architecture rules will be enforced on every code change. Run `$archie-deep-scan --incremental` after code changes to update the architecture analysis.**
