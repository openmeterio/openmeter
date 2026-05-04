# Archie Scan Report
> Deep scan baseline | 2026-05-04 10:09 UTC | 9,366 functions / 351,752 LOC analyzed | continuous run (3rd deep scan)

## Architecture Overview

OpenMeter is a Go monorepo producing seven independently deployable binaries (cmd/server, cmd/billing-worker, cmd/balance-worker, cmd/sink-worker, cmd/notification-service, cmd/jobs, cmd/benthos-collector). All binaries share domain packages under `openmeter/`, assembled via Google Wire DI in `app/common/`. The HTTP surface is authored in TypeSpec (`api/spec/`), compiled to OpenAPI YAML, and then to Go server stubs and SDKs (Go/JS/Python). v1 is served by `openmeter/server/router` (Chi + kin-openapi); v3 by `api/v3/server` (Chi + oasmiddleware). Domain packages follow strict service-interface / adapter-implementation / httpdriver-transport layering with cross-domain callbacks mediated by `ServiceHooks` and `RequestValidator` registries to avoid circular imports.

The system of record is PostgreSQL via Ent ORM with Atlas-managed migrations under `tools/migrate/migrations/`; usage events are stored in ClickHouse and queried via `streaming.Connector` for meter aggregations. Async cross-binary communication runs over Kafka via Watermill with three named topics (ingest, system, balance-worker) routed by event-name prefix in `openmeter/watermill/eventbus`. Multi-step billing operations serialise per customer via PostgreSQL advisory locks (`pkg/framework/lockr`) acquired inside ctx-propagated Ent transactions (`pkg/framework/entutils.TransactingRepo`).

The most load-bearing architectural commitments — the ones a coding agent must honour to keep the system correct — are: (1) every adapter helper under `openmeter/billing/charges/.../adapter` wraps DB access in `entutils.TransactingRepo` to honour the ctx-bound transaction; (2) the `credits.enabled` feature flag is enforced at four independent wiring layers (`app/common/ledger.go`, `app/common/customer.go`, `app/common/billing.go::NewBillingRegistry`, `api/v3/server` credit handlers); (3) cross-domain hooks/validators are registered as side-effects inside Wire provider functions in `app/common/`, never in the source domain; (4) the two-step regen cadence (`make gen-api` → `make generate`) keeps TypeSpec, OpenAPI, Go stubs, Wire, Ent, Goverter, and Goderive output in sync; (5) all builds and tests run with `-tags=dynamic` so confluent-kafka-go links against librdkafka.

## Health Scores

| Metric     | Current  | Previous | Trend | What it means |
|------------|---------:|---------:|------:|---------------|
| Erosion    | 0.4459   | 0.4459   | flat  | Stable; no new fragmentation since 2026-04-28. |
| Gini       | 0.6968   | 0.6969   | flat  | Complexity remains concentrated in the top quartile of functions. |
| Top-20%    | 0.7379   | 0.7378   | flat  | The top 20% of functions own ~74% of the codebase mass. Concentration unchanged. |
| Verbosity  | 0.0588   | 0.0586   | flat  | Exact-line duplication stays under 6%. The semantic duplication discussion (Part 6) is the meaningful complement. |
| LOC        | 351,752  | 345,136  | +1.9% | Codebase grew ~6,600 lines in the last deep-scan window — modest, in line with feature work in `openmeter/billing` and `openmeter/subscription`. |

Interpreted together, the numbers describe a stable codebase: complexity is concentrated but not growing, duplication is contained, and growth is gradual rather than burst-y. The risk surface lives in the top complexity offenders (below) and in the architectural-rule adherence reported in *Findings*.

### Complexity Trajectory

The top 10 high-complexity functions:

| CC | Function | Location |
|---:|----------|----------|
| 65 | `sync` | `openmeter/subscription/service/sync.go:28` |
| 61 | `reconcileWebhookEvent` | `openmeter/notification/eventhandler/webhook.go:30` |
| 59 | `toSQL` | `openmeter/streaming/clickhouse/meter_query.go:108` |
| 58 | `AppStripeWebhook` | `openmeter/app/stripe/httpdriver/webhook.go:40` |
| 51 | `TestOpenPeriod` | `pkg/timeutil/openperiod_test.go:8` (test only — not a production risk) |
| 47 | `CreateCheckoutSession` | `openmeter/app/stripe/client/checkout.go:21` |
| 47 | `Validate` | `openmeter/productcatalog/ratecard.go:665` |
| 43 | `AsPrice` | `openmeter/productcatalog/http/mapping.go:565` |
| 41 | `ListInvoices` | `openmeter/billing/adapter/invoice.go:119` |
| 41 | `GetEntitlementBalanceHistory` | `openmeter/entitlement/metered/balance.go:105` |

580 functions exceed CC=15. The concentration around `subscription/service/sync.go`, the Stripe webhook reconciler, and the ClickHouse query builder reflects three areas where state-machine logic, external-event reconciliation, and dynamic SQL composition each accumulate inherent complexity. They are the most likely sites for future bugs and should be the first targets for refactoring or test hardening.

## Findings

### NEW (first observed this scan)

**Warnings**

1. **[warn] `time.Now()` widely used in billing workers instead of `pkg/clock.Now()`.** Six worker files (`openmeter/billing/worker/advance/advance.go:82,97`, `openmeter/billing/worker/worker.go`, `openmeter/billing/worker/collect/collect.go`, `openmeter/billing/worker/subscriptionsync/reconciler/reconciler.go`, `openmeter/billing/worker/subscriptionsync/service/sync.go`, `openmeter/billing/service/invoicecalc/gatheringrealtime.go`) anchor cutoff windows and synthetic timestamps to wall-clock `time.Now()`. The reconciler's own per-folder CLAUDE.md flags this as an anti-pattern, and the existing rule corpus already mandates `pkg/clock.Now()` for production code so tests can `clock.FreezeTime` and reproduce billing cutoffs deterministically. The drift agent identified this in 6 distinct hot-path files, suggesting the rule has eroded post-baseline. Confidence 0.92.

2. **[warn] Structural hack in `app/common/ledger.go::NewLedgerHistoricalLedger`.** A second `accountservice` instance is constructed with a `nil` `Querier` field to break a circular dependency in the historical ledger wiring. This violates the project's "no nil fields in injected dependencies" convention and creates a latent NPE: any call path that triggers the historical ledger to look up an account through the nil Querier will panic at runtime. Confidence 0.85.

3. **[warn] Hook registration side-effects fragmented across three customer-related Wire providers.** `app/common/customer.go` registers customer hooks via three separate provider functions (ledger hook, subject hook, entitlement validator). Wire sees only types, not side-effects: a binary that omits any one provider in its `wire.Build` still compiles successfully but silently drops the hook. This is the live manifestation of pitfall `pf_0006`; it's surfaced now as a NEW finding because no compile-time enforcement protects against the regression. Confidence 0.90.

### RECURRING (previously documented, still present — `confirmed_in_scan ≥ 2`)

4. **[error] `f_0001` / `pf_0001` — Charges adapter helpers accepting raw `*entdb.Client` can silently bypass the ctx-bound transaction.** Every helper under `openmeter/billing/charges/**/adapter` that touches `a.db` must do so inside `entutils.TransactingRepo` / `TransactingRepoWithNoValue`. The mandate is documented in AGENTS.md and the per-package CLAUDE.md but no compile-time check enforces it. Confirmed in 4 scans. Confidence 0.95.

5. **[error] `f_0002` / `pf_0002` — Credits-disabled deployments can still write to the ledger if any of four wiring layers forgets to guard.** `app/common/ledger.go`, `app/common/customer.go`, `app/common/billing.go::NewBillingRegistry`, and `api/v3/server` credit handlers each independently check `creditsConfig.Enabled`. A new ledger-touching provider added without this branch re-introduces the leak. Confirmed in 4 scans. Confidence 0.95.

6. **[warn] `f_0003` / `pf_0007` — Notification event payload versioning is implicit.** Payload version constants live alongside the payload struct in `openmeter/notification/`; there is no machinery to migrate old payloads when a struct changes. Confirmed in 4 scans. Confidence 0.85.

7. **[warn] `f_0004` / `pf_0004` — Sequential timestamped Atlas migrations + `atlas.sum` chain hashing produces predictable merge collisions on long-lived branches.** Documented; the `/rebase` skill encodes the recovery procedure. Confirmed in 4 scans. Confidence 0.95.

8. **[warn] `f_0005` — `context.Background()` / `context.TODO()` introductions in application code silently sever Ent transaction propagation and OTel spans.** The mandate is documented in AGENTS.md and pre-edit hooks; reviewers must catch each occurrence. Confirmed in 4 scans. Confidence 0.90.

9. **[warn] `f_0006` — Only `cmd/server` registers `namespace.Handler` implementations; other binaries that reach `initNamespace` will not have ClickHouse/Kafka/Ledger handlers wired by default.** Confirmed in 3 scans. Confidence 0.80.

10. **[warn] `f_0007` / `pf_0006` — Wire-generated provider sets concentrate cross-domain hook registration as side-effects.** A binary that omits a provider silently drops its hook with no compile error. The new finding #3 above is a fresh manifestation in the customer hooks. Confirmed in 3 scans. Confidence 0.85.

11. **[warn] `f_0008` / `pf_0007` — `openmeter/watermill/eventbus.GeneratePublishTopic` uses a string-prefix switch on `EventName()` to route topics; default-case fallback to `SystemEventsTopic` means a misnamed event family silently misroutes instead of erroring.** Confirmed in 3 scans. Confidence 0.85.

### RESOLVED

_None resolved this run. All eight findings from the prior scan remain active and have been confirmed-in-scan again._

## Pitfalls (durable, carried in blueprint)

The blueprint carries 8 architectural pitfalls (`pf_0001`–`pf_0008`) covering: charges adapter discipline, credits multi-layer guard, multi-generator regen drift, Atlas migration collisions, `-tags=dynamic` build linking, cross-domain hook registration, eventbus prefix routing, and TypeSpec source-of-truth regen. These are intentionally durable: they describe *classes* of problem rooted in architectural decisions and are referenced from the per-finding `pitfall_id`. They do not need to be re-stated each scan; they are loaded from `blueprint.json` by the AI reviewer on every plan approval and pre-commit.

## Architectural Health Assessment

| Dimension | Rating | Evidence |
|-----------|--------|----------|
| Separation of concerns     | **Strong**   | Domain packages strictly separate Service / Adapter / HTTP. Cross-domain coupling routes through `ServiceHookRegistry` and `RequestValidatorRegistry`, not direct imports. The 40-component blueprint shows clean responsibility boundaries. |
| Dependency direction       | **Strong**   | Domain packages do not import `app/common`; Wire wiring flows outward. The dependency_graph.json reports 0 cycles across resolved directories. The four-layer credits guard is a deliberate cross-cutting case, not an inversion. |
| Pattern consistency        | **Adequate** | The TransactingRepo discipline, registry-based hooks, and noop-on-disabled patterns are applied consistently across mature domains (billing, customer, subscription). New finding #1 (clock usage) and #3 (fragmented hook registration) show the patterns can erode where compile-time enforcement is absent. |
| Testability                | **Strong**   | Test helpers are colocated under `<domain>/testutils/` independent of `app/common`. `BaseSuite + SubscriptionMixin` patterns enable integration tests against real Postgres + Svix. The `pkg/clock` indirection (when used) supports deterministic time-travel tests; finding #1 shows it's not yet uniform. |
| Change impact radius       | **Adequate** | The Wire provider graph + TypeSpec→OpenAPI→SDK pipeline localises most changes. Atlas-migration linearity intentionally creates merge friction on long branches (pitfall `pf_0004`) — the trade-off is reviewability. The two-step regen cadence (pitfall `pf_0008`) means a TypeSpec edit ripples through six generators that must all be run; missing one is a real risk. |

## Top Risks & Recommendations

1. **Clock-determinism erosion in billing workers** (NEW finding #1). The drift agent found 6 production files using `time.Now()` directly in cutoff or anchor positions. A correctness-fatal bug would be a billing-cutoff test that passes locally but produces incorrect invoice periods in production. Recommend: add a custom golangci-lint analyzer that flags `time.Now()` outside `pkg/clock` and `*_test.go`; fix the 6 sites; promote the rule into `enforcement.md` so the pre-edit hook blocks new occurrences.

2. **Charges TransactingRepo discipline (`f_0001`/`pf_0001`)**. Recurring across 4 scans with no compile-time enforcement. Failure mode is partial writes under concurrency that don't manifest until a multi-tenant production load. Recommend: write the lint analyzer specified in the existing pitfall fix-direction, plus an integration test that opens a transaction and asserts every public charges-Service method commits/rolls back atomically.

3. **Credits four-layer guard (`f_0002`/`pf_0002`)**. Recurring across 4 scans. The blast radius is "credits-disabled tenant produces ledger rows" — a data-integrity / billing-correctness regression that's silent until financial reconciliation. Recommend: the smoke integration test in `pf_0002.fix_direction` (boots with `credits.enabled=false`, asserts ledger tables stay empty under representative flows) is the highest-leverage missing piece.

4. **Customer hook registration fragility** (NEW finding #3 / recurring `pf_0006`). Three independent providers each register a hook; omitting one in a binary's `wire.Build` is a silent drop. Recommend: per the pitfall's fix-direction, promote hook bundles into named registry types so a missing registry is a compile error. Lower-cost: add an integration test per binary asserting `customerService.HookCount()` matches the expected count for that binary's role.

5. **Concentration of complexity in `subscription/service/sync.go::sync` (CC=65)**. This single function is the dominant target for both bugs and refactoring. Recommend: harvest the function's preconditions and post-conditions into a state machine driven by `qmuntal/stateless` (the same library `openmeter/billing/service/stdinvoicestate.go` uses); split per-trigger subroutines.

## Semantic Duplication

The Phase 2 deep-drift agent did not flag explicit `semantic_duplication` clusters in the strategic sample of 20 files reviewed (billing core, charges adapters, app/common wiring, framework primitives, key domain services). The mechanical verbosity score of 0.0588 (≤6%) measures only exact line clones and is the right floor; semantic duplication would manifest as similar function bodies under different names. The most plausible places for it in this codebase — converter packages (`api/v3/handlers/.../convert.go`), Stripe webhook handlers, and ratecard validation logic — were not part of the strategic sample, so a more targeted future scan over `api/v3/handlers/**/convert.go` would be the highest-leverage extension.

**No actionable semantic duplication detected in this scan's strategic sample.**

## Proposed Rules

The Step 6 rule synthesis run produced 33 new rules merged with 71 prior rules, for a total of **104 rules** in `.archie/rules.json` (52 path-glob triggers, 43 code-shape triggers, 93 classifier rules). New rules cover: TypeSpec source-of-truth gates, charges TransactingRepo at adapter sites, credits four-layer guard at each Wire provider, cross-domain hook registration patterns, Atlas migration linearity, and `-tags=dynamic` build invariants. No standalone `proposed_rules.json` is needed — the active rules are the merged set, and the pre-edit hook + plan/commit classifier consult them on every change.
