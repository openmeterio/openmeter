# Archie Scan Report
> Deep scan baseline | 2026-05-14 20:00 UTC | 9,931 functions / 379,517 LOC analyzed | full deep scan

## Architecture Overview

OpenMeter is a single Go monorepo that compiles seven independently deployable binaries (`cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`, `cmd/benthos-collector`) from a shared domain-package tree under `openmeter/`. Every domain follows a strict three-layer pattern: a Service interface at the package root, a concrete implementation in `service/`, an Ent/PostgreSQL adapter in `adapter/`, and HTTP handlers in `httpdriver/` (v1) or `api/v3/handlers/` (v3). All binary wiring is concentrated in `app/common/` via Google Wire provider sets, keeping domain packages free of any dependency on the DI layer.

The HTTP surface is contract-first: TypeSpec under `api/spec/` is the single source of truth, compiled to OpenAPI YAML and then to Go server stubs plus Go/JS/Python SDKs. PostgreSQL via Ent ORM (with Atlas-managed migrations) is the system of record; ClickHouse stores raw usage events for meter aggregation; Kafka via Watermill is the async event bus with three name-prefix-routed topics (ingest, system, balance-worker). Cross-domain coupling is mediated through `ServiceHookRegistry` and `RequestValidatorRegistry` rather than direct imports, and per-customer billing operations serialize through `pg_advisory_xact_lock` acquired inside ctx-propagated Ent transactions.

The most consequential decisions are the multi-binary split sharing one typed domain model (independent scaling of ingest vs. billing vs. balance recalculation), TypeSpec-as-source-of-truth (no SDK drift across three languages and two API versions), and the `credits.enabled` feature flag enforced at four independent wiring layers (a deliberate fan-out rather than a single choke point, because credits cross-cut HTTP handlers, customer hooks, namespace provisioning, and charge creation).

## Health Scores

| Metric | Current | Previous | Trend | What it means |
|--------|--------:|---------:|------:|---------------|
| Erosion    | 0.4453  | 0.4459  | flat  | Structural fragmentation stable; no new erosion since 2026-05-04. |
| Gini       | 0.6957  | 0.6968  | flat  | Complexity stays concentrated in the top quartile of functions. |
| Top-20%    | 0.7378  | 0.7379  | flat  | The top 20% of functions own ~74% of codebase mass — unchanged. |
| Verbosity  | 0.0609  | 0.0588  | up (+0.0021) | Exact-line duplication ticked up slightly but stays under 7%; 23,104 duplicate lines, much of it generated SDK / parallel test fixtures. |
| LOC        | 379,517 | 351,752 | +7.9% | Codebase grew ~27,800 lines this window — the largest jump in the tracked history, driven by `api/v3/handlers` build-out and `openmeter/billing`. |

Interpreted together: the codebase is structurally stable (erosion, gini, top-20% all flat) but growing faster than in prior windows, and the growth is concentrated in the v3 API handler tree. The verbosity uptick plus this scan's semantic-duplication findings (below) point at the real risk surface — copy-evolved domain↔API conversion code in `api/v3/handlers/*/convert.go` — rather than at any architectural decay.

### Complexity Trajectory

Top high-CC functions (624 functions at CC ≥ 11; 5 above CC 51, none above 100):

| Function | Location | CC | SLOC |
|----------|----------|---:|-----:|
| `sync` | `openmeter/subscription/service/sync.go:28` | 65 | 270 |
| `reconcileWebhookEvent` | `openmeter/notification/eventhandler/webhook.go:30` | 61 | 344 |
| `toSQL` | `openmeter/streaming/clickhouse/meter_query.go:108` | 59 | 215 |
| `AppStripeWebhook` | `openmeter/app/stripe/httpdriver/webhook.go:40` | 58 | 354 |
| `CreateCheckoutSession` | `openmeter/app/stripe/client/checkout.go:21` | 47 | 169 |
| `Validate` | `openmeter/productcatalog/ratecard.go:678` | 47 | 148 |
| `AsPrice` | `openmeter/productcatalog/http/mapping.go:565` | 43 | 156 |

Risk is concentrated in subscription sync, notification reconciliation, and ClickHouse query building — the same offenders as prior scans, none growing. `subscription/service/sync.go::sync` remains the single dominant refactor target.

## Findings

Ranked by severity, grouped by novelty.

### NEW (first observed this scan)

**Errors**

_None._

**Warnings**

1. **[warn] Domain↔API price/rate-card conversion is independently duplicated across three handler packages.** The full `productcatalog.RateCard ↔ api.BillingRateCard` and `productcatalog.Price ↔ api.BillingPrice` bidirectional conversion hierarchy is reimplemented in `api/v3/handlers/plans/convert.go`, `api/v3/handlers/addons/convert.go`, and `api/v3/handlers/customers/charges/convert.go` — none imports the others; the logic is copy-evolved. Adding a new price type requires parallel edits in three places; omitting one silently produces 500s or missing fields on that API surface. A shared `api/v3/handlers/productcatalogconvert/` package would collapse the drift surface. Confidence 0.9.

2. **[warn] `addons/convert.go` returns a plain `fmt.Errorf` for unsupported v3 price types where `plans/convert.go` returns a typed `models.GenericConflictError`.** `plans/convert.go:238` handles `DynamicPriceType`/`PackagePriceType` with `models.NewGenericConflictError` (→ 409 via `GenericErrorEncoder`); `addons/convert.go:385` returns a bare `fmt.Errorf` for the identical condition, which falls through to a 500. Same condition, divergent HTTP contract across sibling packages. Confidence 0.9.

3. **[warn] `addons/list.go` lacks the `hasUnsupportedV3Price` filter that `plans/list.go` applies before serialization.** `plans/list.go:115` skips plans containing `DynamicPriceType`/`PackagePriceType`; `addons/list.go` has no equivalent guard, so a single addon with an unsupported rate card fails the entire `ListAddons` response. The invariant "list never errors on unsupported price types" holds for plans but not addons. Confidence 0.85.

4. **[warn] `var _ Handler = (*handler)(nil)` compile-time check present in only 1 of 17 v3 handler packages.** `handlers/CLAUDE.md` mandates the assertion in every `handler.go`; only `api/v3/handlers/apps/handler.go:16` has it. The other 16 packages surface a missing interface method only at runtime or as a distant wiring-package compile error. Confidence 0.9.

5. **[warn] `ToAPIBillingRateCardTaxConfig` has incompatible signatures in `plans/convert.go` vs `addons/convert.go`.** The plans version takes `(*productcatalog.TaxConfig, *taxcode.TaxCode)` and gates on both being non-nil; the addons version drops the `taxcode` parameter entirely, so addons rate cards serialize without taxcode metadata that plans rate cards include — structurally different output for the same `api.BillingRateCardTaxConfig` type, with no test enforcing parity. Confidence 0.85.

6. **[warn] `ToAPIBillingSpendCommitments` has diverged input representations across `plans` and `addons`.** Plans takes a `productcatalog.Commitments` value; addons takes two separate `*alpacadecimal.Decimal` pointers. The two packages hold different mental models of the same domain concept reaching the serialization layer — a responsibility leak that cannot be unified without a canonical-source judgment call. Confidence 0.8.

7. **[warn] `get_billing.go` decoder omits the deleted-customer guard that all three sibling mutation handlers apply.** `customers/billing/CLAUDE.md` documents that every decoder calls `GetCustomer` and checks `IsDeleted()`; `update_billing.go`, `update_billing_app_data.go`, and `create_customer_stripe_checkout_session.go` do, but `GetCustomerBilling` does not — it returns a successful response for a soft-deleted customer. Lower severity (read-only) but inconsistent with the package contract. Confidence 0.85.

8. **[warn] `f_0009` — `openmeter/app/stripe/client/appclient.go` uses `context.Background()` inside request-path error handling.** `providerError()` at `appclient.go:238-240` constructs a background context during request processing, severing Ent transaction propagation and OTel spans — the live manifestation of `f_0005`'s class in the Stripe app client. Confidence 0.9.

9. **[warn] `f_0010` — `cmd/billing-worker/main.go` calls `EnsureBusinessAccounts` and `SandboxProvisioner` directly in `main.go`.** Startup-orchestration logic that the architecture otherwise keeps out of `cmd/*/main.go`; correct today but a boundary the multi-binary trade-off depends on. Confidence 0.85.

10. **[info] CLAUDE documentation drift in `customers/billing/error_encoder.go`.** The encoder handles `billing.ValidationIssue` (→ 400) in addition to the four error types listed in the package's `CLAUDE.md`. Not a functional bug — it produces more correct status codes — but the per-folder doc no longer enumerates everything the encoder maps. Confidence 0.9.

### RECURRING (previously documented, still present — `confirmed_in_scan ≥ 3`)

11. **[error] `f_0001` / `pf_0001` — Charges adapter helpers accepting a raw `*entdb.Client` can silently bypass the ctx-bound transaction.** Every helper under `openmeter/billing/charges/**/adapter` that touches `a.db` must do so inside `entutils.TransactingRepo` / `TransactingRepoWithNoValue`. Documented in AGENTS.md and per-package CLAUDE.md; no compile-time check enforces it. Confirmed in 4 scans. _Note: the Haiku verifier flagged this for demotion this run (the cited call site at `adapter.go:52` is the correct `Tx()` interface impl, not an un-wrapped helper) — hysteresis holds it active pending a second confirmation._ Confidence 0.95.

12. **[warn] `f_0003` / `pf_0007` — Notification event payload versioning is implicit.** Payload version constants live alongside the payload struct in `openmeter/notification/`; no machinery migrates old payloads when a struct changes. Confirmed in 4 scans. Confidence 0.85.

13. **[warn] `f_0004` / `pf_0004` — Sequential timestamped Atlas migrations + `atlas.sum` chain hashing produce predictable merge collisions on long-lived branches.** The `/rebase` skill encodes the recovery procedure. Confirmed in 4 scans. Confidence 0.95.

14. **[warn] `f_0005` — `context.Background()` / `context.TODO()` in application code silently severs Ent transaction propagation and OTel spans.** Mandate documented in AGENTS.md and pre-edit hooks; new finding #8 above is a fresh manifestation in the Stripe app client. Confirmed in 4 scans. Confidence 0.9.

15. **[warn] `f_0006` — `cmd/server` is the only binary that registers `namespace.Handler` implementations.** Other binaries reaching `initNamespace` will not have ClickHouse/Kafka/Ledger handlers wired by default. Confirmed in 3 scans. Confidence 0.8.

16. **[warn] `f_0007` / `pf_0006` — Wire-generated provider sets concentrate cross-domain hook registration as side-effects.** A binary that omits a provider silently drops its hook with no compile error. Confirmed in 3 scans. Confidence 0.85.

17. **[warn] `f_0008` / `pf_0008` — `eventbus.GeneratePublishTopic` routes by an `EventName()` string-prefix switch with a default fallback to `SystemEventsTopic`.** A misnamed or new event family silently misroutes instead of erroring. Confirmed in 3 scans. Confidence 0.87.

### RESOLVED / DEMOTED

- **`f_0002` — Credits-disabled deployments can still write to the ledger if any of four wiring layers forgets to guard.** **Demoted** this run by the Haiku verifier: the finding accurately describes a real architectural fragility (fan-out noop guards with no central compile-time enforcement), but no current call site demonstrates the failure firing — it is a *risk class*, not an active instance. It remains in the blueprint as pitfall `pf_0002`. No findings were fully resolved this run.

## Pitfalls (durable, carried in blueprint)

The blueprint carries 7 architectural pitfalls covering: charges adapter TransactingRepo discipline, the `credits.enabled` multi-layer guard, multi-generator regeneration drift, Atlas migration collisions, cross-domain hook registration invisibility to Wire, eventbus prefix-routing fallback, and Watermill silent-drop of unknown event types. These are durable — they describe *classes* of problem rooted in architectural decisions and are referenced by per-finding `pitfall_id`. The AI reviewer loads them from `blueprint.json` on every plan approval and pre-commit.

## Architectural Health Assessment

| Dimension | Rating | Evidence |
|-----------|--------|----------|
| Separation of concerns | **Strong** | Domain packages strictly separate Service / Adapter / HTTP. Cross-domain coupling routes through `ServiceHookRegistry` and `RequestValidatorRegistry`, never direct imports. The 38-component blueprint shows clean responsibility boundaries. Structural outliers (`openmeter/billing` 41 files, `pkg/models` 29 files) are large but cohesive, not god-folders. |
| Dependency direction | **Strong** | Domain packages do not import `app/common`; Wire wiring flows strictly outward. `dependency_graph.json` reports 0 cycles and the mechanical drift scan found 0 dependency violations. The four-layer credits guard is a deliberate cross-cutting case, not an inversion. |
| Pattern consistency | **Adequate** | TransactingRepo discipline, registry-based hooks, and noop-on-disabled are applied consistently across mature domains (billing, customer, subscription). But this scan's 9 deep drift findings show real erosion in `api/v3/handlers`: triplicated `convert.go` logic, inconsistent error typing (#2), asymmetric list filtering (#3), and a compile-time check present in only 1 of 17 handler packages (#4). |
| Testability | **Strong** | Test helpers are colocated under `<domain>/testutils/` independent of `app/common`. `BaseSuite + SubscriptionMixin` enable integration tests against real Postgres + Svix. `pkg/clock` indirection supports deterministic time-travel tests. Duplicate-line analysis shows much of the 23k duplicate lines is parallel test fixtures (e.g. `planaddon` adapter/service tests, `addon`/`plan` ratecard tests). |
| Change impact radius | **Adequate** | The Wire provider graph + TypeSpec→OpenAPI→SDK pipeline localises most changes. But the triplicated price/rate-card conversion code (finding #1) means a single new price type ripples to three packages with no shared seam, and the two-step regen cadence means a TypeSpec edit ripples through multiple generators that must all run. |

## Top Risks & Recommendations

1. **Copy-evolved domain↔API conversion in `api/v3/handlers/*/convert.go`** (NEW findings #1, #2, #3, #5, #6). The single highest-leverage problem this scan. Three packages independently maintain the `Price`/`RateCard` conversion hierarchy with already-divergent signatures and error contracts. Failure mode: adding `PackagePrice` v3 support requires finding and editing all three; missing one ships a 500 or a missing field. **Recommend:** extract a shared `api/v3/handlers/productcatalogconvert/` package exposing the canonical `ToAPIBillingPrice` / `FromAPIBillingPrice` / `ToAPIBillingRateCard` mappings, with `DynamicPriceType`/`PackagePriceType` uniformly returning `models.GenericConflictError`; add a contract test enforcing parity across the three call sites.

2. **`var _ Handler` compile-time check missing in 16 of 17 v3 handler packages** (NEW finding #4). Cheap to fix, high value: adding a method to the `Handler` interface should fail at the package, not at runtime. **Recommend:** add the one-line `var _ Handler = (*handler)(nil)` assertion to every `api/v3/handlers/*/handler.go`; consider a `make generate`-time check or a trivial lint rule.

3. **Charges TransactingRepo discipline (`f_0001` / `pf_0001`)** — recurring 4 scans, no compile-time enforcement. The verifier's demotion signal this run suggests the cited instance may have been a false positive, but the *class* of risk (a helper falling off the ctx transaction → partial writes under concurrency) is real and unguarded. **Recommend:** the golangci-lint analyzer from `pf_0001.fix_direction` that flags `a.db.` usage in adapter bodies without `TransactingRepo` on the call stack, plus a per-method atomicity integration test. If the next scan also demotes, retire the finding and keep only the pitfall.

4. **Credits four-layer guard (`pf_0002`, demoted `f_0002`)** — the finding was demoted to a pure risk class, but the architecture genuinely has no central enforcement point. **Recommend:** the smoke integration test from `pf_0002.fix_direction` (boot with `credits.enabled=false`, assert `ledger_accounts` / `ledger_customer_accounts` stay empty under representative flows) is still the highest-leverage missing safety net — it converts a "remember to guard" convention into a CI gate.

5. **Concentration of complexity in `subscription/service/sync.go::sync` (CC=65)** — unchanged across scans, the dominant single target for both bugs and refactoring. **Recommend:** drive it through a `qmuntal/stateless` state machine (the same library `openmeter/billing/service/stdinvoicestate.go` already uses) and split per-trigger subroutines.

## Proposed Rules

Step 6 rule synthesis merged the existing 104 rules with 8 new rules into `.archie/rules.json` (112 total), covering blueprint sections that previously had no enforcement rule: portal token scope (`portal-001`), the CloudEvent ingest pipeline (`ingest-001`), meter `ParseEvent` (`meter-001`), credit granularity truncation (`credit-001`), entitlement customer locking (`entitlement-001`), balance-worker high-watermark filtering (`balance-001`), Redis dedupe ordering (`dedupe-001`), and SplitLineGroup integrity (`billing-010-split-line-group`). All are already written to `rules.json` and indexed; no separate `proposed_rules.json` was produced this run.
