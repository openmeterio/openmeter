# Archie Scan Report
> Deep scan | 2026-04-28 18:42 UTC | 9,183 functions / 345,136 LOC analyzed | second deep-scan baseline

## Architecture Overview

OpenMeter is a multi-binary Go monolith built around high-volume per-tenant usage metering feeding strict billing correctness. Seven independent binaries — `cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`, `cmd/benthos-collector` — share the same domain packages under `openmeter/` and are assembled into runnable applications exclusively through Google Wire provider sets in `app/common`. Each domain package follows a layered service / adapter / http pattern: a `Service` (or `Connector`) interface, a hand-written Ent/PostgreSQL adapter, and an `httpdriver`/`httphandler` package that maps requests to the service via the generic `pkg/framework/transport/httptransport.Handler[Req,Resp]`.

The HTTP surface is dual-versioned (v1 via `openmeter/server/router` Chi + kin-openapi implementing `api/api.gen.go`; v3 via `api/v3/server` Chi + oasmiddleware implementing `api/v3/api.gen.go`) and is generated from TypeSpec in `api/spec/` as the single source of truth — the same spec drives the Go server stubs, Go SDK, JavaScript SDK (`@openmeter/sdk`), and Python SDK. Schema is owned by Ent (`openmeter/ent/schema/`) and migrations are produced by Atlas into `tools/migrate/migrations/` as sequential golang-migrate `.up`/`.down` SQL files. PostgreSQL is the system of record; ClickHouse handles analytics and high-volume usage queries; Kafka via Watermill (with the prefix-routed publisher in `openmeter/watermill/eventbus`) is the cross-binary event bus carrying ingest, balance-worker, and system-event topics. Svix delivers outbound webhooks.

Load-bearing constraints that shape every architectural decision: `credits.enabled` must be guarded at four independent wiring layers in `app/common`; charges adapter helpers must always rebind to the in-context Ent transaction via `entutils.TransactingRepo(...)`; the multi-generator toolchain (TypeSpec + Ent + Goverter + Wire + Goderive) requires a strict `make gen-api` → `make generate` cadence; and all binaries must compile with `-tags=dynamic` because `confluent-kafka-go` links dynamically against system librdkafka.

## Health Scores

| Metric | Current | Previous | Trend | Interpretation |
|--------|--------:|---------:|:-----:|----------------|
| Erosion       | 0.4459    | 0.5359    | ↓ -0.090 | Less code mass concentrated in heavy/complex sections — the codebase is healthier per-function than the prior baseline. |
| Gini          | 0.6969    | 0.7677    | ↓ -0.071 | Complexity is more evenly distributed; fewer megacode hotspots. |
| Top-20% share | 0.7378    | 0.8248    | ↓ -0.087 | The top 20% of files now hold ~74% of complexity (was 82%) — broader code surface absorbs change. |
| Verbosity     | 0.0586    | 0.1093    | ↓ -0.051 | Exact-line clone density nearly halved. **Caveat:** AI-style semantic duplicates (different signatures, identical bodies) are NOT captured here — see Part 6 below. |
| LOC (analyzed)| 345,136   | 742,092   | ↓ -396k | Large drop is mostly improved scanner filtering of generated/bulk content (api/client/*, ent/db/, *.gen.go) — not a real code shrink. |

The numbers together: structural complexity is more evenly spread and exact-clone duplication is down, but the AI-detected semantic-duplication clusters in the charges sub-tree (deep-001..004 below) mean the `verbosity` improvement understates real maintenance pressure in that area.

### Complexity Trajectory

Top complexity offenders (CC ≥ 38, sloc ≥ 130):

- `openmeter/subscription/service/sync.go::sync` — CC 65, 270 sloc *(intent-rich; documented sync algorithm)*
- `openmeter/notification/eventhandler/webhook.go::reconcileWebhookEvent` — CC 61, 344 sloc *(reconcile loop with retry/backoff branches)*
- `openmeter/streaming/clickhouse/meter_query.go::toSQL` — CC 59, 215 sloc *(SQL builder with many query shapes)*
- `openmeter/app/stripe/httpdriver/webhook.go::AppStripeWebhook` — CC 58, 354 sloc *(Stripe event-type fan-out)*
- `openmeter/app/stripe/client/checkout.go::CreateCheckoutSession` — CC 47, 169 sloc
- `openmeter/productcatalog/ratecard.go::Validate` — CC 47, 148 sloc
- `openmeter/productcatalog/http/mapping.go::AsPrice` — CC 43, 156 sloc
- `openmeter/billing/adapter/invoice.go::ListInvoices` — CC 41, 153 sloc
- `openmeter/entitlement/metered/balance.go::GetEntitlementBalanceHistory` — CC 41, 212 sloc
- `openmeter/entitlement/adapter/entitlement.go::ListEntitlements` — CC 38, 149 sloc

CC distribution: 5,395 functions in [1-2], 2,178 in [3-5], 1,041 in [6-10], 465 in [11-20], 99 in [21-50], 5 in [51-100], 0 above. Heavy tail (CC ≥ 21) is concentrated in subscription/sync, notification/webhook, ClickHouse query building, Stripe integration, and rate-card/product-catalog validation — all areas with genuine domain branching, not boilerplate. Risk concentration is intentional, not accidental.

## Findings

Ranked by severity, grouped by novelty.

### NEW (first observed this scan)

**Errors**

_None new. The two error-class findings (`f_0001`, `f_0002`) are recurring._

**Warnings**

1. **[warn] `deep-001` — `ensureDetailedLinesLoadedForRating` duplicated across state machine and run service.** The method exists in two places: as a method on `*stateMachine` in `openmeter/billing/charges/usagebased/service/statemachine.go` (mutates `s.Charge` in place, no return value) and as a method on `*Service` in `openmeter/billing/charges/usagebased/service/run/create.go` (returns an updated charge). Both lazy-fetch detailed lines from the adapter when the charge's realizations are missing them; the state machine version calls `s.Adapter.FetchDetailedLines` directly, duplicating logic that should be owned by the run service. Confidence 0.85.

2. **[warn] `deep-002` — `FinalizeRealizationRun` (creditsonly) and `SnapshotInvoiceUsage` (creditheninvoice) share near-identical 12-step bodies.** In `openmeter/billing/charges/usagebased/service/creditsonly.go` and `.../service/creditheninvoice.go` both methods execute the same sequence (nil-check `CurrentRealizationRunID` → `ensureDetailedLinesLoadedForRating` → `GetByID` → compute `storedAtOffset` → rate via `GetDetailedLinesForUsage` → round with `CurrencyCalculator` → `run.ReconcileCredits` → `run.PersistRunDetailedLines` → `run.UpdateRealizationRun` → clear run state → `UpdateCharge` → `RefetchCharge`). The only behavioral differences are the `storedAtOffset` value (`0` vs non-zero) and the `CreditAllocationMode` flag (`Exact` vs `Available`). Highest-impact duplicate cluster in the charges sub-tree. Confidence 0.95.

3. **[warn] `deep-003` — v1/v2 detailed-line upsert pairs in the billing adapter are near-identical.** In `openmeter/billing/adapter/stdinvoicelines.go`: `upsertDetailedLines` and `upsertDetailedLinesV2` have structurally identical bodies that differ only by the Ent builder type (`*db.BillingInvoiceLineCreate` vs `*db.BillingStandardInvoiceDetailedLineCreate`). The same applies to `upsertDetailedLineAmountDiscounts` / `...V2`. The SchemaLevel branching in `UpsertInvoiceLines` calls one pair or the other; bug-fixes have to be applied twice. Confidence 0.9.

4. **[warn] `deep-004` — `OnCreditPurchasePaymentAuthorized` and `OnCreditPurchasePaymentSettled` share near-identical bodies.** In `openmeter/ledger/chargeadapter/creditpurchase.go` both methods follow the documented "Resolve→annotate→commit" pattern verbatim — only the `transactions.ResolveTransactions` template type differs. The ChargeAdapter CLAUDE.md describes this pattern as canonical; the violation is implementing it twice instead of extracting a `commitCreditPurchaseTemplate(ctx, charge, template)` helper. Confidence 0.9.

5. **[warn] `deep-005` — `createNewRealizationRun` does two adapter writes without an explicit `transaction.Run` wrapper.** In `openmeter/billing/charges/usagebased/service/run/create.go`: the helper calls `s.adapter.CreateRealizationRun` then `s.adapter.UpdateCharge` back-to-back. Atomicity works today only because the outer `withLockedCharge` in the charges service holds an open transaction in ctx via `transaction.RunWithResult` and `entutils.TransactingRepo` rebinds inside each adapter call. Implicit coupling — breaks if `run.Service.createNewRealizationRun` is ever called from a fresh ctx (e.g. in a test that drives `run.Service` directly). Confidence 0.8.

6. **[warn] `deep-006` — `lineengine.go::newStateMachineForStandardLine` reaches into charge persistence inside the billing plugin layer.** In `openmeter/billing/charges/usagebased/service/lineengine.go` the line engine (a `billing.LineEngine` plugin invoked during invoice line processing) calls `svc.GetByID` to fetch the charge entity instead of receiving it pre-resolved. Couples the billing line layer to charge persistence reads — a responsibility leak that contradicts the LineEngine plugin contract documented in the architecture rules. Confidence 0.8.

7. **[warn] `f_0006` — Only `cmd/server` registers `namespace.Handler` implementations (Ledger + KafkaIngest); other binaries that talk to the same DB don't.** Workers reach the database independently of the namespace lifecycle, meaning a namespace deletion done from one binary leaves orphaned state visible to others. Documented in the canonical findings store; `confirmed_in_scan: 1`. See AGENTS.md "Architecture section" — namespace handlers must be registered before `initNamespace`. Confidence 0.85.

8. **[warn] `f_0007` — Wire-generated provider sets concentrate cross-domain hook registration as side-effects.** Hook registration happens inside provider functions (`app/common/customer.go::NewCustomerLedgerServiceHook` calls `customerService.RegisterHooks(h)` as a side-effect; `app/common/billing.go::NewBillingRegistry` registers `RequestValidator`s and `SubscriptionCommandHook`s the same way). Wire sees only types, not side-effects: a binary that omits the provider in its `wire.Build` still compiles but silently drops the hook. Mirrors pitfall `pf_0006`. Confidence 0.9.

9. **[warn] `f_0008` — Watermill prefix-routed topic dispatch silently fans new event families to `SystemEventsTopic`.** `openmeter/watermill/eventbus.GeneratePublishTopic` switches on `EventName()` prefix — anything that doesn't start with `ingestevents.EventVersionSubsystem` or `balanceworkerevents.EventVersionSubsystem` lands in `SystemEventsTopic`. A new event family that forgets to register a prefix (or chooses a fresh subsystem) silently merges with system events and gets eaten by the wrong consumer (or none). Mirrors pitfall `pf_0007`. Confidence 0.85.

**Informational**

10. **[info] Two folders flagged as god-folders.** `openmeter/billing/` (39 files vs sibling avg 8) and `pkg/models/` (29 files vs sibling avg 5). For billing this is expected (largest domain); for `pkg/models` it's a candidate for splitting along the Service/Validator/PageResult axes.

### RECURRING (previously documented, still present — `confirmed_in_scan ≥ 2`)

11. **[error] `f_0001` / `pf_0001` — Charges adapter helpers accepting raw `*entdb.Client` can silently bypass the ctx-bound transaction.** Mandate from AGENTS.md: every helper under `openmeter/billing/charges/**/adapter` that touches `a.db` must do so inside `entutils.TransactingRepo` / `TransactingRepoWithNoValue`. Confirmed in 2 scans. Confidence 0.95.

12. **[error] `f_0002` / `pf_0002` — Credits-disabled deployments can still write to the ledger if any of four wiring layers forgets to guard.** `app/common/ledger.go`, `app/common/customer.go`, `app/common/billing.go::NewBillingRegistry`, and `api/v3/server` credit handlers each independently check `creditsConfig.Enabled`. A new ledger-touching provider added without this branch re-introduces the leak. Confirmed in 2 scans. Confidence 0.95.

13. **[warn] `f_0003` — Notification event payload versioning is implicit.** Payload version constants live alongside the producing event packages while the consumer matches on a `ce_type` string — no compile-time enforcement of version compatibility. Confirmed in 2 scans. Confidence 0.8.

14. **[warn] `f_0004` / `pf_0004` — Sequential timestamped Atlas migrations + `atlas.sum` linear hash chain produce predictable merge conflicts on long-running feature branches.** The `/rebase` skill documents the recovery procedure; the cost compounds with branch age. Confirmed in 2 scans. Confidence 0.95.

15. **[warn] `f_0005` — `context.Background()` / `context.TODO()` in application code drops cancellation, deadlines, and request-scoped values.** Multiple specific occurrences flagged in the prior scan (`openmeter/app/stripe/client/appclient.go::providerError`, notification eventhandler `Start`/`Dispatch`, several e2e helpers). AGENTS.md prohibits this in application code; tests should use `t.Context()`. Confirmed in 2 scans. Confidence 0.9.

### RESOLVED

_None confirmed resolved this scan. The deep drift agent reviewed only 20 architecturally-critical files (charges service tree, billing adapter, route wiring, server config, ledger chargeadapter), so the prior scan's findings #7 (charges test helper sub-services), #8 (pagination boilerplate duplication in v3 list handlers), #9 (v1/v3 handler parity drift) were not re-checked. Re-run a full scan or a targeted review to confirm._

## Pitfalls (durable, carried in blueprint)

- `pf_0001` [error] Charges adapter helpers accepting raw `*entdb.Client` can silently bypass `TransactingRepo` and cause partial writes under concurrency.
- `pf_0002` [error] `credits.enabled=false` does not fully stop ledger writes unless every wiring layer is independently guarded.
- `pf_0003` [warn] Multi-generator toolchain (TypeSpec + Ent + Goverter + Wire + Goderive) is easy to leave partially regenerated, producing drifted APIs / SDKs.
- `pf_0004` [warn] Sequential Atlas migration filenames + `atlas.sum` chain hashing guarantee merge collisions on long-lived feature branches.
- `pf_0005` [warn] Tests or build invocations that omit `-tags=dynamic` fail to link `confluent-kafka-go` against librdkafka with confusing errors.
- `pf_0006` [warn] Cross-domain hook/validator registration as side-effects inside Wire provider functions: omitting the provider in a binary's `wire.Build` silently drops the hook with no compile error.
- `pf_0007` [warn] EventName prefix-based topic routing in `eventbus.GeneratePublishTopic` falls through to `SystemEventsTopic` by default; new event families that forget to register a prefix silently merge with system events.
- `pf_0008` [warn] TypeSpec edits without a follow-up `make gen-api` + `make generate` produce silent drift between the API contract and Go server stubs / SDKs.

## Architectural Health Assessment

| Dimension | Rating | Notes |
|---|---|---|
| Separation of concerns | **Strong** | Service / adapter / httpdriver layering is consistent across ~50 components. ServiceHooks + RequestValidator registries break cross-domain dependencies without circular imports. The `BillingRegistry`/`AppRegistry`/`ChargesRegistry` typed groupings keep cohesive services bundled. |
| Dependency direction | **Strong** | Domain packages don't import `app/common`; AGENTS.md codifies this for tests too. `cmd/*` only calls `initializeApplication`. 0 cycles in `dependency_graph.json`. |
| Pattern consistency | **Adequate** | Newer domains follow the canonical split cleanly. The charges sub-tree shows local drift: same realization sequence implemented twice (`deep-002`), v1/v2 upsert pairs duplicated in the billing adapter (`deep-003`), and "resolve-annotate-commit" repeated in chargeadapter (`deep-004`). Credits-guard discipline is uneven across providers. |
| Testability | **Strong** | Service interfaces support mocking; `openmeter/.../testutils` packages stay independent of `app/common` to avoid import cycles; `t.Context()` convention; `pgtestdb` + parallel infrastructure carries 549 changed Go files in 30 days. |
| Change impact radius | **Moderate — watch** | Two coupled sources of ripple: (a) `app/common` is a single concentrated wiring surface (every new domain adds a file), and (b) dual v1/v3 HTTP handlers require parallel maintenance. Both are documented; neither has a mechanical guard against drift. The semantic-duplication clusters in charges add a third: a behavior change in the realization-run sequence has to be applied twice. |

## Top Risks & Recommendations

1. **Charges adapter `TransactingRepo` discipline (`f_0001` / `pf_0001`).** Highest-severity recurring finding. The `charges/**/adapter` helpers currently *work* only because their callers happen to be tx-bound. Hard recommendation: add a custom golangci-lint analyzer (or a CI ripgrep gate) that walks `openmeter/billing/charges/**/adapter/**.go` and fails on `a.db.` access outside an `entutils.TransactingRepo` callback. Pair with an integration test that asserts atomic commit/rollback under concurrent `AdvanceCharges` calls.

2. **Credits-disabled leak (`f_0002` / `pf_0002`).** Equal severity. Add a unit-or-wiring test that boots `app/common` with `Credits.Enabled=false` and asserts every credit-touching service is a noop via type assertion. Add a PR template checkbox: "If this PR adds a Wire provider that touches the ledger, does the credits-disabled path return a noop?"

3. **Realization-run state-machine duplication (`deep-001`, `deep-002`).** Highest-impact non-recurring finding. Extract a shared `executeRealizationRun(ctx, opts realizationRunOpts)` private helper on the base `stateMachine`. Both `FinalizeRealizationRun` (creditsonly) and `SnapshotInvoiceUsage` (creditheninvoice) become thin wrappers passing `storedAtOffset` and `CreditAllocationMode`. Same shape for `ensureDetailedLinesLoadedForRating` (consolidate into the run package).

4. **Multi-generator toolchain partial regen (`pf_0003`).** No mechanical gate today beyond `make migrate-check`. Recommendation: extend the dirty-tree CI gate to cover all five generators (TypeSpec → OpenAPI YAML, Ent → ent/db, Wire → wire_gen, Goverter → convert.gen, Goderive → derived.gen). A single `make generate-all` invocation followed by `git diff --exit-code` is the cleanest one-liner.

5. **Atlas `atlas.sum` chain conflicts on long branches (`f_0004` / `pf_0004`).** Recurring; recovery procedure documented. Consider a daily-rebase reminder bot for branches that touch `openmeter/ent/schema/`, or a CI message when an open PR's migration timestamp predates `main`'s most recent migration.

6. **`context.Background()` creep (`f_0005`).** Recurring across both production and e2e. Add a custom golangci-lint rule that forbids `context.Background()` / `context.TODO()` outside `main()`, top-level CLI commands, and explicitly-annotated detach points. Require `t.Context()` in tests.

7. **Watermill default-fall-through routing (`f_0008` / `pf_0007`).** Recommendation: change `eventbus.GeneratePublishTopic` to error (or log loudly with a metric counter) when an event name doesn't match any registered prefix, instead of silently routing to `SystemEventsTopic`. Coupled change: add a unit test that registers each `EventVersionSubsystem` constant and asserts it routes to the intended topic.

## Semantic Duplication

The mechanical verbosity score (0.0586) is misleadingly low — it only catches exact line clones. The AI deep drift surfaced **four high-confidence semantic-duplication groups** that the metric misses entirely:

| Group | Files | Canonical owner | Differing axis | Recommendation |
|---|---|---|---|---|
| `ensureDetailedLinesLoadedForRating` | `usagebased/service/statemachine.go`, `usagebased/service/run/create.go` | `run.Service` | receiver + return signature | Consolidate in run package; statemachine delegates. |
| `FinalizeRealizationRun` / `SnapshotInvoiceUsage` (12-step body) | `usagebased/service/creditsonly.go`, `usagebased/service/creditheninvoice.go` | base `stateMachine` (private helper) | `storedAtOffset` + `CreditAllocationMode` | Extract `executeRealizationRun(ctx, opts)`; both methods become wrappers. |
| `upsertDetailedLines{,V2}` & `upsertDetailedLineAmountDiscounts{,V2}` | `billing/adapter/stdinvoicelines.go` | shared helper or generic | Ent builder type | Strategy interface or Go generic over the builder. |
| `OnCreditPurchasePaymentAuthorized` / `OnCreditPurchasePaymentSettled` | `ledger/chargeadapter/creditpurchase.go` | `commitCreditPurchaseTemplate` | template type only | Extract private helper performing resolve→annotate→commit. |

Two patterns dominate: AI copy-paste with a tweaked signature (groups 1, 4) and "v1/v2 schema-level branching" (group 3). All four sit inside the charges/ledger/billing-adapter cluster — the same area where `f_0001` / `pf_0001` mandate strict transactional discipline. Behavior changes there have to be applied in multiple places today; consolidation directly reduces the blast radius for that pitfall as well.

## Proposed Rules

This run synthesized **35 new rules**, merged into the existing 36, total **71 in `rules.json`** (29 path globs + 13 code shapes + 76 classifier rules in `rule_index.json`).

Coverage spans:

- Generated-code immutability — Ent (`openmeter/ent/db/`), Wire (`*_wire_gen.go`), Goverter (`*.convert.gen.go`), Goderive (`billing/derived.gen.go`), oapi-codegen (`api/api.gen.go`, `api/v3/api.gen.go`, `api/client/go/client.gen.go`), TypeSpec→OpenAPI (`api/openapi*.yaml`).
- TypeSpec + schema regeneration cadence (`make gen-api` then `make generate`).
- Ent schema change workflow (`make generate` then `atlas migrate --env local diff`).
- `entutils.TransactingRepo` discipline in charges adapters (path-glob + code-shape trigger).
- `context.Background()` / `context.TODO()` prohibition outside `main()` and tests; `t.Context()` required in tests.
- `credits.enabled` four-layer wiring guard.
- `POSTGRES_HOST=127.0.0.1` for DB tests.
- Domain testutils may not import `app/common`.
- `-tags=dynamic` for all Go builds and tests.
- TypeSpec `@query` requires `using TypeSpec.Http;`.
- Atlas migration sequentiality + `atlas.sum` chain integrity.
- Notification payload versioning.
- Kafka topic provisioning via `app/common`'s `KafkaTopicProvisioner` (no direct `confluent-kafka-go.NewAdminClient` calls in domain code).
- Watermill router middleware order from `openmeter/watermill/router.NewDefaultRouter`.
- HTTP handler shape via `httptransport.NewHandler[Req,Resp]` with `commonhttp.GenericErrorEncoder`.
- `models.Generic*` sentinel errors instead of raw status codes in handlers.
- `lockr.Locker.LockForTX` for per-customer billing operations.
- ServiceHook / RequestValidator registration via `app/common` provider functions (avoids circular imports).
