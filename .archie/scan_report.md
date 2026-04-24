# Archie Scan Report
> Deep scan baseline | 2026-04-24 10:20 UTC | 48,315 functions / 742,092 LOC analyzed | baseline run (FIRST_BASELINE)

## Architecture Overview

OpenMeter is a multi-binary Go monolith built around high-volume, per-tenant usage metering that feeds strict billing correctness. Seven independent binaries — `cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`, `cmd/benthos-collector` — share the same domain packages under `openmeter/` and are assembled into runnable applications exclusively through Google Wire provider sets in `app/common`. Each domain package follows a layered service/adapter/http pattern: a `Service` (or `Connector`) interface, a hand-written Ent/PostgreSQL adapter, and an `httpdriver`/`httphandler` package that maps requests to the service via a generic `httptransport.Handler[Req,Res]`.

The HTTP surface is dual-versioned (v1 via `openmeter/server/router` and `api/api.gen.go`, v2/v3 via `api/v3/server` and `api/v3/api.gen.go`) and is generated from TypeSpec in `api/spec/` as the single source of truth — the same spec drives the Go server stubs, Go client, JavaScript SDK (`@openmeter/sdk`), and Python SDK. Schema is owned by Ent (`openmeter/ent/schema/`) and migrations are produced by Atlas into `tools/migrate/migrations/` as sequential golang-migrate `.up/.down` SQL files. PostgreSQL is the system of record; ClickHouse handles analytics and high-volume usage queries; Kafka (via Watermill and `confluent-kafka-go`) is the cross-binary event bus for ingest, balance, and system topics. Svix is the outbound webhook delivery layer.

Load-bearing constraints that shape everything: `credits.enabled` must be guarded at every independent wiring layer in `app/common`; charges-adapter helpers must always rebind to the in-context Ent transaction via `entutils.TransactingRepo(...)`; the multi-generator toolchain (TypeSpec + Ent + Goverter + Wire + Goderive) requires a specific `make gen-api` → `make generate` cadence; and all binaries must be compiled with `-tags=dynamic` because of the librdkafka C dependency.

## Health Scores

| Metric | Current | Previous | Trend | What it means |
|--------|--------:|---------:|------:|---------------|
| Erosion    | 0.5359    | — | baseline | 53.6% of codebase mass sits in "heavy" functions; substantial complexity concentration. |
| Gini       | 0.7677    | — | baseline | Size distribution is highly skewed; a minority of files hold most of the code. |
| Top-20%    | 0.8248    | — | baseline | 82.5% of total mass sits in the top 20% of files — textbook heavy tail. |
| Verbosity  | 0.1093    | — | baseline | ~11% duplicate-line ratio (81,139 / 742,092); most is generated/scaffolded code. |
| LOC        | 742,092   | — | baseline | 3,296 files total; 1,811 under `openmeter/`, 483 under `api/`. |

The skew in Gini and Top-20% is expected here: `openmeter/ent/db/` alone hosts the highest-CC functions in the repo (init at CC=965; multiple ent-generated `assignValues` / `sqlSave` in the 100-190 range), and Ent-generated code also inflates the duplicate-line count. After discounting generated code, the real picture is more benign but still shows meaningful concentration in hand-written hot paths (subscription sync, notification reconciler, ClickHouse query builder, Stripe webhook).

### Complexity Trajectory (hand-written only)
- `openmeter/subscription/service/sync.go:28 sync` — CC 65
- `openmeter/notification/eventhandler/webhook.go:30 reconcileWebhookEvent` — CC 61
- `openmeter/streaming/clickhouse/meter_query.go:108 toSQL` — CC 59
- `openmeter/app/stripe/httpdriver/webhook.go:40 AppStripeWebhook` — CC 58
- `openmeter/app/stripe/client/checkout.go:21 CreateCheckoutSession` — CC 47
- `openmeter/productcatalog/ratecard.go:665 Validate` — CC 47
- `openmeter/productcatalog/http/mapping.go:564 AsPrice` — CC 43
- `openmeter/billing/adapter/invoice.go:119 ListInvoices` — CC 41
- `openmeter/entitlement/metered/balance.go:105 GetEntitlementBalanceHistory` — CC 41

Risk is concentrated in exactly the places you'd expect: the subscription sync algorithm, the ClickHouse query generator, Stripe webhook handling, and billing invoice listing — all high-complexity orchestration paths that benefit most from tight test coverage and defensive decomposition.

## Findings

Ranked by severity, grouped by novelty.

### NEW (first observed this scan)

**Errors**

1. **[error] Non-transactional `*entdb.Client` access in charges adapter helpers.** `openmeter/billing/charges/flatfee/adapter/charge.go`'s `buildCreateFlatFeeCharge` (line 266 area) uses `a.db.ChargeFlatFee.Create()` directly. Although it is currently invoked only from inside a `TransactingRepo` closure (receiver is the tx-bound adapter), the helper's signature does not enforce this, so a future non-tx caller would silently bypass transaction rebinding and produce partial writes under concurrency. Violates pitfall `pf_0001`. Confidence 0.85.

2. **[error] `credits.enabled` guard missing in `NewCreditGrantService`.** `app/common/creditgrant.go:21` checks `billingRegistry.Charges == nil` but never checks `creditsConfig.Enabled` before constructing the real `creditgrant.Service` — unlike the neighbouring `ledger.go`, `customer.go`, and `customerbalance.go` providers. A deployment with credits explicitly disabled will still wire the credit-grant service, which is exactly the class of leak pitfall `pf_0002` warns about. Confidence 0.9.

3. **[error] `context.Background()` in production error-handling path.** `openmeter/app/stripe/client/appclient.go:240` — `(*stripeAppClient).providerError` calls `c.appService.UpdateAppStatus(context.Background(), ...)`. This drops cancellation signals and trace spans on every provider-side Stripe error; explicit AGENTS.md prohibition. Confidence 0.95.

**Warnings**

4. **[warn] Long-lived reconciler roots its context in `context.Background()`.** `openmeter/notification/eventhandler/handler.go:84` — `(*Handler).Start` uses `context.WithCancel(context.Background())` instead of accepting a lifecycle context from the caller, which prevents clean shutdown propagation via context cancellation. Confidence 0.9.

5. **[warn] Dispatch goroutine uses `context.Background()` as its trace root.** `openmeter/notification/eventhandler/dispatch.go:29` — `(*Handler).Dispatch` calls `context.WithTimeout(context.Background(), notification.DefaultDispatchTimeout)`. Even though a fresh trace root is reasonable for an async fan-out, AGENTS.md asks application paths to propagate the caller's context; if the intent is to detach, that choice should be explicit (e.g. `context.WithoutCancel(parent)` in Go 1.21+) with a comment. Confidence 0.75.

6. **[warn] E2E test helpers bypass `t.Context()`.** `e2e/helpers.go` — `CreateCustomerWithSubject` (line 25), `GetMeterIDBySlug` (line 59), and `QueryMeterV3` (line 81) all accept `*testing.T` but build contexts from `context.Background()` instead of `t.Context()`. Breaks test-scoped lifecycle and cancellation per AGENTS.md testing guidance. Confidence 0.95.

7. **[warn] Charges test helper exposes sub-services individually.** `openmeter/billing/charges/testutils/service.go:80-85` — `NewServices` returns `FlatFeeService`, `UsageBasedService`, and `CreditPurchaseService` alongside the top-level `charges.Service`. This surface tempts test authors to drive lifecycle through lower-level adapters instead of the canonical `charges.Service.Create` / `AdvanceCharges` / `ApplyPatches` path that AGENTS.md mandates for usage-based lifecycle tests. Confidence 0.7.

8. **[warn] Duplicated pagination-parsing boilerplate in v3 list handlers.** `api/v3/handlers/billingprofiles/list.go`, `api/v3/handlers/taxcodes/list.go`, and `api/v3/handlers/llmcost/list_overrides.go` each repeat the exact same `pagination.NewPage(1,20)` default, `lo.FromPtrOr(params.Page.Number/Size)`, and `page.Validate()` block verbatim. Semantic duplication; extract a shared request helper (e.g. `api/v3/request/pagination.go` or a new `pagination.ParseFromParams`). Confidence 0.9.

9. **[warn] Dual HTTP API versioning with significant resource overlap.** `openmeter/server/router/router.go` (v1) and `api/v3/server/server.go` (v3) both independently construct handler sets from the same domain services for overlapping resources (meters, customers, subscriptions, billing, apps). Two parallel wiring paths must be kept in sync; no automated drift gate exists today. Confidence 0.9.

10. **[warn] Credits subsystem requires multi-layer manual wiring discipline.** `app/common/ledger.go` wires ledger account services to noop implementations when `credits.enabled=false`, but AGENTS.md explicitly documents that any ledger-account backfill that must write real `ledger_accounts` / `ledger_customer_accounts` rows has to construct concrete adapters directly and cannot rely on DI outputs. This is the same class as finding 2; pitfall `pf_0002`. Confidence 0.85.

**Informational**

11. **[info] `app/common` is a 43-file mega-package.** `openmeter_server.go` carries an explicit `// TODO: create a separate file or package for each application instead`. Adding a new domain requires modifying shared wiring. Known interim design.

12. **[info] `feature.FeatureConnector` has not been migrated to the Service pattern.** `openmeter/productcatalog/feature/connector.go:58` carries `// TODO: refactor to service pattern`. Legacy naming (`FeatureConnector` instead of `Service`) and location (`connector.go` instead of `service.go`) diverge from the canonical layout used by newer domains. Low urgency; documented self-awareness.

13. **[info] `subscription.SubscriptionServiceWithWorkflow` initializer is known-overloaded.** `app/common/subscription.go:38` — `// TODO: break up to multiple initializers`. `NewSubscriptionServices` does too much in a single function per its own comment.

14. **[info] Ent view entities require manual migration SQL.** `ent.View` entities generate query code but don't appear in `migrate.Tables`, so Atlas reports no changes and view DDL must be added to hand-written SQL migrations. Known tooling limitation (AGENTS.md); impact is surprise-not-correctness.

15. **[info] Structural outliers (from mechanical drift).**
    - `openmeter/billing/` has 38 files vs. sibling average 7 — candidate for sub-domain splitting.
    - `pkg/models/` has 28 files vs. sibling average 4 — same.
    - Five folders (`.github/`, `quickstart/`, `api/client/{go,javascript,python}/`) use non-`.md` extensions while siblings are mostly `.md`. All of these are correct-by-design (CI configs, generated SDKs).

### RECURRING (previously documented, still present)

_Not applicable — first baseline._

### RESOLVED

_Not applicable — first baseline._

## Pitfalls (durable, carried in blueprint)

- `pf_0001` [error] Charges adapter helpers accepting raw `*entdb.Client` can silently bypass `TransactingRepo` and cause partial writes under concurrency.
- `pf_0002` [error] `credits.enabled=false` does not fully stop ledger writes unless every wiring layer is independently guarded.
- `pf_0003` [warn] Multi-generator toolchain (TypeSpec + Ent + Goverter + Wire + Goderive) is easy to leave partially regenerated, producing drifted APIs.
- `pf_0004` [warn] Sequential Atlas migration filenames + `atlas.sum` chain hashing guarantee merge collisions on long-lived feature branches.

## Architectural Health Assessment

| Dimension | Rating | Notes |
|---|---|---|
| Separation of concerns | **Strong** | The service / adapter / http layering is consistent across domains, and the `httptransport.Handler[Req,Res]` generic keeps HTTP→service translation uniform. Exceptions (legacy `FeatureConnector`) are documented. |
| Dependency direction | **Strong** | Domain packages don't import `app/common`; AGENTS.md codifies this for tests too. `cmd/*` only calls `initializeApplication`. No cycles detected at directory level (0 cycles across 7 nodes, 4 edges). |
| Pattern consistency | **Adequate** | Newer domains follow the canonical split cleanly; a handful of legacy spots (`FeatureConnector`, `SubscriptionServiceWithWorkflow` initializer) are self-flagged. Credits-guard discipline is uneven (see finding 2). |
| Testability | **Adequate** | Service interfaces support mocking; `charges.testutils` and `billing` test suites exist. Two weaknesses: e2e helpers ignoring `t.Context()`, and the charges test helper surfacing lower-level services. |
| Change impact radius | **Moderate — watch** | Two coupled sources of ripple: (a) `app/common` mega-package forces cross-domain edits for any new wiring, and (b) dual v1/v3 HTTP surfaces require parallel handler maintenance. Both are documented; neither has a mechanical guard against drift. |

## Top Risks & Recommendations

1. **Credits-disabled leak (finding 2).** High impact because a misconfigured deployment silently issues real ledger writes. Add a unit or wiring test that boots `app/common` with `credits.enabled=false` and asserts every credit-related service is a noop. Watch for new `app/common/*.go` providers that touch credits and add them to the guard.
2. **TransactingRepo signature safety (finding 1 + pitfall pf_0001).** The helper currently "works" only because callers happen to be tx-bound. Either require `TransactingRepo` wrapping in every adapter helper signature, or add a linter/test that walks `openmeter/billing/charges/**/adapter` and fails on raw `*entdb.Client` access in helper bodies.
3. **`context.Background()` creep (findings 3, 4, 5, 6).** Four occurrences across production and e2e code. Lightweight fix: add a custom golangci-lint rule (or a simple `ripgrep` check in CI) that forbids `context.Background()` / `context.TODO()` outside `main()` and test setup — with `t.Context()` required in tests.
4. **v1/v3 handler parity drift (finding 9).** No automated guard today. Consider (a) generating a matrix of `resource × version × method` and failing CI when v1 supports something v3 doesn't (or vice versa), or (b) a formal deprecation schedule for v1.
5. **`app/common` as a god-package.** Finding 11 + the self-reported TODO. Any new domain wiring adds to this weight. Consider splitting by domain (`app/common/billing/`, `app/common/notification/`, …) with thin re-exports for backwards compatibility, then chipping away over a few PRs.

## Semantic Duplication

One high-confidence group identified:

- **Pagination-parsing boilerplate** duplicated verbatim in `api/v3/handlers/billingprofiles/list.go`, `api/v3/handlers/taxcodes/list.go`, and `api/v3/handlers/llmcost/list_overrides.go`. Each file repeats the same `pagination.NewPage(1,20)` default construction, `lo.FromPtrOr(params.Page.Number/Size)` unpacking, and `page.Validate()` call. The v3 handler layer should surface a single `pagination.FromParams(params.Page)` (or equivalent in `api/v3/request`) and have each handler call that helper. This is finding 8.

Mechanical verbosity (0.109 / 10.9%) reports no additional near-duplicates that weren't already explained by generated Ent code.

## Proposed Rules

The first run synthesized 36 rules (saved to `.archie/rules.json`) spanning:

- Generated-code immutability (Ent, Wire, Goverter, Goderive, oapi-codegen outputs, OpenAPI YAML)
- TypeSpec + schema regeneration cadence (`make gen-api` before `make generate`)
- Ent schema change workflow (`make generate` then `atlas migrate --env local diff`)
- TransactingRepo discipline in charges adapters
- `context.Background()` / `context.TODO()` prohibition + `t.Context()` in tests
- `credits.enabled` multi-layer wiring guard
- `POSTGRES_HOST=127.0.0.1` for DB tests
- Domain testutils must not import `app/common`
- `-tags=dynamic` for all Go builds
- TypeSpec `@query` → requires `using TypeSpec.Http;`
- Atlas migration sequentiality + `atlas.sum` update
- Notification payload versioning
- Kafka provisioning via `app/common`'s `KafkaTopicProvisioner`
- `cmd/*` must only call `initializeApplication`, never reach into domain code directly

These are now active in `.archie/rules.json` and will be consulted by the AI reviewer on plan approval and pre-edit hooks.
