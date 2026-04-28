## Enforcement Rules (101 total)

Every rule the pre-edit hook (`PRE_VALIDATE_HOOK`) and the plan/commit classifier (`align_check.py`) consults. Grouped by severity.

_12 decision_violation, 5 pitfall_triggered, 42 mechanical_violation, 30 tradeoff_undermined, 12 pattern_divergence_

## Decision Violations (block)

### `dep-001` — Domain packages under openmeter/ must not import app/common. Wire wiring flows outward (app/common imports domain packages); reversing the direction creates import cycles and defeats the Wire compile-time graph.

*source: `deep_scan`* · *scope: `openmeter/`* · *check: `forbidden_import`*

**Why:** Domain packages under openmeter/ have no dependency on cmd/* or app/common (enforced by leaf-node import direction). Each cmd/<binary>/wire.go composes only the provider sets it needs. Cross-domain hook/validator registration done inside app/common avoids circular imports — if a domain package imported app/common the cycle would be unresolvable.

**Path glob:** `openmeter/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "\"github\\.com/openmeterio/openmeter/app/common\""
    ]
  }
]
```

</details>

### `dep-002` — Every new cmd/* worker binary must have a matching app/common/openmeter_<binary>.go Wire provider set file. Adding a binary without this file leaves its dependency graph unverified at compile time.

*source: `deep_scan`* · *scope: `app/common/`* · *check: `file_naming`*

**Why:** Each cmd/<binary> binary needs its own provider graph but every domain service must be wireable identically; Wire makes this compile-time checked. Workers added without matching app/common/openmeter_*worker.go Wire set are a documented violation signal.

**Example:**

```
// app/common/openmeter_billingworker.go
var BillingWorker = wire.NewSet(
    Billing,
    Charges,
    LedgerStack,
    // ...
)
```

### `dep-003` — Cross-domain hooks and request validators must be registered inside app/common provider functions, not inside domain package constructors. Domain packages must expose RegisterHooks/RegisterRequestValidator methods that app/common calls after wiring.

*source: `deep_scan`*

**Why:** Cross-domain hooks (billing → customer, ledger → customer, billing → subscription) would create circular imports if registered inside source packages. app/common/customer.go registers customerService.RegisterRequestValidator(validator) and customerService.RegisterHooks(ledgerHook, subjectHook) as side-effects of Wire provider functions to avoid circular imports.

**Example:**

```
// app/common/customer.go
func NewCustomerService(adapter customer.Adapter, ledgerHook customer.ServiceHook, ...) customer.Service {
    svc := customer.New(adapter)
    svc.RegisterHooks(ledgerHook, subjectHook)
    svc.RegisterRequestValidator(billingValidator)
    return svc
}
```

### `dep-005` — New billing backend integrations must implement billing.InvoicingApp and self-register via app.Service.RegisterMarketplaceListing() in their New() constructor. Never hardcode provider-specific logic inside billing.Service.

*source: `deep_scan`*

**Why:** The App Factory / Registry pattern keeps billing.Service decoupled from specific payment providers. Each app's New() self-registers a factory via app.Service.RegisterMarketplaceListing in service/factory.go. No billing service code changes are needed when adding a new backend.

**Example:**

```
// openmeter/app/stripe/service/factory.go
func New(appSvc app.Service, ...) (*StripeApp, error) {
    appSvc.RegisterMarketplaceListing(stripeMarketplaceListing, stripeFactory)
    return &StripeApp{}, nil
}
```

### `wm-001` — Never publish events to a Kafka topic by string literal. Always use the eventbus.Publisher from openmeter/watermill/eventbus, which routes by EventName() prefix to the correct named topic.

*source: `deep_scan`*

**Why:** Topic routing is by event-name prefix in openmeter/watermill/eventbus — events whose EventName() starts with ingestevents.EventVersionSubsystem go to IngestEventsTopic; balanceworkerevents.EventVersionSubsystem go to BalanceWorkerEventsTopic; everything else to SystemEventsTopic. Publishing directly to a Kafka topic string is a documented violation keyword.

**Example:**

```
// Correct: domain service uses eventbus.Publisher
if err := h.eventbus.Publish(ctx, &billingevents.InvoiceCreated{InvoiceID: inv.ID}); err != nil {
    return fmt.Errorf("publish invoice created: %w", err)
}
```

**Path glob:** `openmeter/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "kafka\\.NewProducer|confluent.*ProduceChannel|sarama.*SendMessage"
    ],
    "must_not_match": [
      "eventbus\\.Publisher",
      "watermill"
    ]
  }
]
```

</details>

### `wm-002` — Watermill consumer handlers must use msg.Context() to propagate the caller's context. Never substitute context.Background() inside a NoPublishingHandler or GroupEventHandler.

*source: `deep_scan`*

**Why:** Context carries the Ent transaction driver, request-scoped OTel spans, and cancellation. In Watermill consumers, msg.Context() is the only correct source of the request-scoped context. Substituting context.Background() severs traces and drops the Ent transaction. The anti-pattern 'carry ctx through the consumer via msg.Context()' is explicitly required.

**Example:**

```
// Correct: consumer uses msg.Context()
router.AddNoPublisherHandler("invoice-created", topics.System, subscriber, func(msg *message.Message) error {
    return svc.OnInvoiceCreated(msg.Context(), ev)
})
```

**Path glob:** `openmeter/watermill/**/*.go`, `openmeter/billing/worker/**/*.go`, `openmeter/notification/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "func.*\\*message\\.Message.*error"
    ],
    "must_not_match": [
      "msg\\.Context\\(\\)"
    ]
  }
]
```

</details>

### `layer-004` — The Service interface for each domain must be defined at the package root (service.go or <domain>.go). Never define the service interface inside the service/ or adapter/ sub-packages.

*source: `deep_scan`*

**Why:** Each domain exposes a Go interface (e.g. billing.Service, customer.Service) defined in <domain>/service.go. A concrete service struct in <domain>/service/ holds business logic and calls an Adapter interface for all DB access. The Adapter interface is defined alongside the Service interface and implemented by Ent-backed structs in <domain>/adapter/ sub-packages.

**Example:**

```
// Correct structure:
// openmeter/billing/service.go         — interface definition
// openmeter/billing/adapter.go         — adapter interface
// openmeter/billing/service/service.go — concrete implementation
// openmeter/billing/adapter/adapter.go — Ent-backed implementation
```

### `layer-006` — v1 API endpoints must only be added in api/spec/packages/legacy/; v3 API endpoints only in api/spec/packages/aip/. Never mix v1 and v3 TypeSpec definitions in the same package.

*source: `deep_scan`*

**Why:** Route and tag bindings are declared only in root openmeter.tsp files, not in domain sub-folder operation files. api/spec/packages/aip/src/openmeter.tsp is the v3 AIP TypeSpec entry point; api/spec/packages/legacy/src/main.tsp is the v1 TypeSpec entry point. Mixing them prevents independent versioning.

### `credits-004` — The v3 server (api/v3/server/routes.go) must check s.Credits.Enabled before dispatching to any customer credits or ledger-backed handler. Missing this guard silently enables ledger writes for credits-disabled deployments.

*source: `deep_scan`*

**Why:** api/v3/server/routes.go checks s.Credits.Enabled before dispatching to customer credits handlers. Credits feature flag is enforced at four independent wiring layers; the v3 route dispatch is one of them. Adding a new credits-related endpoint without this guard is a violation.

**Example:**

```
// api/v3/server/routes.go
func (s *Server) ListCustomerCredits(w http.ResponseWriter, r *http.Request, ...) {
    if !s.Credits.Enabled {
        models.NewStatusProblem(r.Context(), nil, http.StatusNotImplemented).Respond(w, r)
        return
    }
    // ...
}
```

### `api-001` — Never add a new endpoint only in the Go handler package without first adding it to the TypeSpec source in api/spec/. Adding endpoint only in Go handler is a documented violation keyword.

*source: `deep_scan`*

**Why:** API is authored in TypeSpec under api/spec/packages/ (aip/ for v3, legacy/ for v1) and compiled to OpenAPI YAMLs, then to Go server stubs. Drift between Go server stubs, three SDKs (Go/JS/Python), and two API versions is impossible only as long as both regen steps run. Adding endpoint only in Go handler bypasses this contract.

**Path glob:** `openmeter/**/httpdriver/**/*.go`, `api/v3/handlers/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "func.*Handler.*ServeHTTP|func.*http\\.ResponseWriter.*\\*http\\.Request"
    ]
  }
]
```

</details>

### `billing-005` — The InvoiceLine tagged-union must be constructed only via NewStandardInvoiceLine or NewGatheringInvoiceLine. Never use struct-literal InvoiceLine{} — the private discriminator t stays zero-valued and type accessors (AsStandardLine etc.) will error.

*source: `deep_scan`*

**Why:** InvoiceLine is a tagged-union type with private discriminator and NewStandardInvoiceLine/NewGatheringInvoiceLine constructors. The billing domain model enforces correct type access via AsStandardLine/AsGatheringLine/AsGenericLine. Direct struct construction bypasses the discriminator.

**Example:**

```
// Correct: use constructors
line := billing.NewStandardInvoiceLine(billing.StandardInvoiceLineInput{...})
```

**Path glob:** `openmeter/billing/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "billing\\.InvoiceLine\\{"
    ]
  }
]
```

</details>

### `wire-001` — wire.Build in cmd/<binary>/wire.go must list only provider sets from app/common (e.g. common.Billing, common.LedgerStack). Never call domain service constructors directly from wire.Build in cmd/* packages.

*source: `deep_scan`*

**Why:** Every domain package exposes plain constructors; all wiring lives in app/common/*.go. cmd/<binary>/wire.go lists the sets; cmd/<binary>/wire_gen.go is generated. Manual constructor call chains in cmd/* duplicate graphs and lose compile-time Wire graph verification.

**Example:**

```
// cmd/billing-worker/wire.go
func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
    wire.Build(
        common.BillingWorker, // correct: provider set from app/common
    )
    return Application{}, nil, nil
}
```

## Pitfalls (block)

### `ctx-001` — Never pass a raw *entdb.Tx as a struct field or function parameter in adapters. Always rebind to the context-carried transaction via entutils.TransactingRepo / TransactingRepoWithNoValue.

*source: `deep_scan`*

**Why:** Ent transactions propagate implicitly via ctx; the TransactingRepo wrapper is the only way to rebind the Ent client to the active transaction. An adapter struct that stores *entdb.Tx instead of using TransactingRepo is an explicit violation signal. Adapter structs must implement TxCreator (Tx via HijackTx + NewTxDriver) and use TransactingRepo on every method body.

**Example:**

```
// Correct adapter method pattern:
func (a *adapter) Create(ctx context.Context, in domain.CreateInput) (*domain.Entity, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*domain.Entity, error) {
        row, err := tx.db.Entity.Create().SetNamespace(in.Namespace).Save(ctx)
        if err != nil { return nil, err }
        return toDomain(row), nil
    })
}
```

**Path glob:** `openmeter/**/adapter/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "\\*entdb\\.Tx"
    ],
    "must_not_match": [
      "entutils\\.TransactingRepo",
      "entutils\\.TransactingRepoWithNoValue"
    ]
  }
]
```

</details>

### `ctx-002` — Never call a.db.Foo() directly in adapter method bodies in openmeter/billing/charges/**/adapter without wrapping in entutils.TransactingRepo. The raw client ignores the active Ent transaction in ctx.

*source: `deep_scan`* · *scope: `openmeter/billing/charges/`* · *check: `architectural_constraint`*

**Why:** Charges adapter helpers that accept a raw *entdb.Client bypass the ctx-bound Ent transaction and produce partial writes under concurrency. TransactingRepo silently falls back to repo.Self() when no tx is in ctx; a missed wrapper does not raise an error, so partial writes are correctness-fatal in the AdvanceCharges / ApplyPatches multi-step flows.

**Path glob:** `openmeter/billing/charges/**/adapter/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "a\\.db\\.\\w+\\."
    ],
    "must_not_match": [
      "entutils\\.TransactingRepo",
      "entutils\\.TransactingRepoWithNoValue"
    ]
  }
]
```

</details>

### `billing-003` — The Charge tagged-union (charges.Charge) must be constructed via NewCharge[T] and accessed via AsFlatFeeCharge/AsUsageBasedCharge/AsCreditPurchaseCharge. Never construct a charges.Charge{} struct literal — this leaves the discriminator empty and accessors return errors.

*source: `deep_scan`*

**Why:** The Charge tagged-union has a private discriminator field (meta.ChargeType) set only by NewCharge[T]. A struct-literal Charge{} leaves the discriminator empty; subsequent type-switch accessors will return errors. Tagged-union Charge prevents partial construction.

**Example:**

```
// Correct: use the constructor
charge := charges.NewCharge(flatfee.Charge{...})
fc, err := charge.AsFlatFeeCharge()
// Wrong: never do charges.Charge{...}
```

**Path glob:** `openmeter/billing/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "charges\\.Charge\\{"
    ]
  }
]
```

</details>

### `credits-003` — ChargesRegistry must not be constructed when credits.enabled=false. BillingRegistry.ChargesServiceOrNil() must be used by callers instead of accessing BillingRegistry.Charges directly.

*source: `deep_scan`*

**Why:** When config.Credits.Enabled=false, app/common/billing.go's NewBillingRegistry skips newChargesRegistry entirely — BillingRegistry.Charges stays nil and ChargesServiceOrNil() returns nil. Accessing BillingRegistry.Charges directly without nil check panics at runtime when credits are disabled.

**Example:**

```
// Correct: use the nil-safe accessor
if svc := registry.ChargesServiceOrNil(); svc != nil {
    // charges are enabled
}
```

**Path glob:** `**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "BillingRegistry\\.Charges\\."
    ],
    "must_not_match": [
      "ChargesServiceOrNil"
    ]
  }
]
```

</details>

### `lock-002` — lockr.Locker.LockForTX requires an active Postgres transaction already in context before it is called. Calling LockForTX outside a TransactingRepo-established transaction will fail or use a bare connection that does not hold the advisory lock.

*source: `deep_scan`*

**Why:** pkg/framework/lockr/locker.go calls pg_advisory_xact_lock($1) with a CRC64-based hash of the lock key. Requires an active Postgres transaction in context (from GetDriverFromContext). Lock is released automatically on tx commit/rollback. Calling LockForTX without an active transaction in ctx cannot correctly scope the lock.

**Example:**

```
// Correct: LockForTX called inside TransactingRepo
return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*domain.Invoice, error) {
    if err := a.locker.LockForTX(ctx, customerID); err != nil { return nil, err }
    // mutation work here
})
```

**Path glob:** `openmeter/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "\\.LockForTX\\("
    ],
    "must_not_match": [
      "TransactingRepo",
      "entutils\\.Transacting"
    ]
  }
]
```

</details>

## Mechanical Violations (block)

### `api-002` — Never hand-edit api/openapi.yaml, api/openapi.cloud.yaml, or api/v3/openapi.yaml. These are generated from TypeSpec source via `make gen-api`.

*source: `deep_scan`* · *scope: `api/`* · *check: `forbidden_content`*

**Why:** TypeSpec is the single source of truth for both v1 and v3 HTTP APIs. api/openapi.yaml, api/openapi.cloud.yaml, and api/v3/openapi.yaml are all generated output. Manual edits are silently overwritten on the next `make gen-api` and cause SDK drift.

**Path glob:** `api/openapi.yaml`, `api/openapi.cloud.yaml`, `api/v3/openapi.yaml`

### `infra-001` — Every pull request must have exactly one release-note label (release-note/ignore, kind/feature, release-note/bug-fix, release-note/breaking-change, etc.) before merging. The PR Checks workflow blocks merge without it.

*source: `deep_scan`*

**Why:** Always add exactly one release-note label to every PR before merging — the PR Checks workflow enforces this via mheap/github-action-required-labels. The workflow fails if the label is missing, blocking the PR from merging.

### `infra-002` — Never push untrusted (PR) Docker images to GHCR. Use the untrusted-artifacts.yaml reusable workflow for PRs, which builds but does not publish container images.

*source: `deep_scan`*

**Why:** Never push untrusted (PR) Docker images to GHCR — the untrusted-artifacts.yaml reusable workflow builds but does not publish container images for PRs. Only the artifacts.yaml workflow (triggered on verified events) publishes to GHCR.

### `gen-001` — Never manually edit files inside openmeter/ent/db/. These are generated by Ent from schema definitions in openmeter/ent/schema/ and must be regenerated with `make generate`.

*scope: `openmeter/ent/db/`* · *check: `forbidden_content`*

**Why:** openmeter/ent/db/ is the generated Ent ORM output. Manual edits are overwritten on the next `make generate` and break the schema-as-code pipeline that Atlas relies on for migration diffs. The source of truth is openmeter/ent/schema/*.go.

### `gen-002` — Never manually edit *.gen.go files, wire_gen.go files, api/api.gen.go, api/v3/api.gen.go, api/client/go/client.gen.go, api/openapi.yaml, or api/openapi.cloud.yaml. Regenerate with `make generate` or `make gen-api`.

*check: `required_pattern`*

**Why:** All generated files carry `// Code generated by X, DO NOT EDIT.` headers. Editing them produces divergence that is silently overwritten on the next generation pass, wasting effort and introducing hidden rollbacks. The five generators (TypeSpec, Ent, Wire, Goverter, Goderive) each have specific Makefile targets.

### `gen-003` — After editing TypeSpec files in api/spec/, run `make gen-api` first to regenerate OpenAPI specs and SDKs, then `make generate` to regenerate Go server stubs and Wire code. Both steps are mandatory.

*scope: `api/spec/`* · *check: `architectural_constraint`*

**Why:** TypeSpec is the single source of truth. `make gen-api` produces OpenAPI YAML and SDK artifacts; `make generate` (go generate ./...) then produces Go server stubs, Wire graphs, and Goverter converters that depend on those OpenAPI files. Partial regeneration leaves the repo in a diverged state that may compile but behave incorrectly.

### `gen-004` — TypeSpec files that add @query, @get, @post, @route, or any other HTTP decorator must import @typespec/http and include `using TypeSpec.Http;` at the top of the file.

*scope: `api/spec/`* · *check: `required_pattern`*

**Why:** Without the @typespec/http import the TypeSpec compiler cannot resolve HTTP decorators and emits an 'Unknown decorator' compilation error. This is not caught at the Go level — only at the TypeSpec compilation step inside `make gen-api`.

### `gen-005` — After editing any Ent schema file in openmeter/ent/schema/, run `make generate` to regenerate openmeter/ent/db/, then `atlas migrate --env local diff <name>` to produce timestamped .up.sql/.down.sql migration files and update atlas.sum. All three artifacts must be committed together.

*scope: `openmeter/ent/schema/`* · *check: `architectural_constraint`*

**Why:** Ent schema defines DB shape; generated code and SQL migration files must stay in sync. A schema change without a corresponding migration causes runtime schema mismatch. atlas.sum records a hash chain that CI validates; an uncommitted sum file breaks the `make migrate-check` CI step.

### `gen-006` — Never manually edit SQL migration files in tools/migrate/migrations/. They are generated by Atlas and validated by CI via `make migrate-check`. Editing them corrupts atlas.sum and breaks the migration chain.

*scope: `tools/migrate/migrations/`* · *check: `architectural_constraint`*

**Why:** Atlas maintains a cryptographic hash chain over migration files in atlas.sum. Any manual edit to a migration file invalidates the hash, fails `atlas migrate validate`, and produces a non-reproducible migration history. Use `atlas migrate --env local diff <name>` to create new migrations and `atlas migrate --env local lint` to verify them.

### `tx-001` — Every helper function in openmeter/billing/charges/**/adapter that accepts a *entdb.Client must wrap its body with entutils.TransactingRepo(...) or entutils.TransactingRepoWithNoValue(...) so it rebinds to the transaction already carried in ctx rather than using the raw client.

*scope: `openmeter/billing/charges/`* · *check: `architectural_constraint`*

**Why:** Ent transactions propagate implicitly via context. A helper that operates on the raw *entdb.Client falls off the transaction context and can produce partial writes when called inside a multi-step AdvanceCharges or ApplyPatches flow. entutils.TransactingRepo reads the *TxDriver from ctx and rebinds to it automatically.

### `tx-002` — Never introduce context.Background() or context.TODO() to work around missing context propagation in application code. Propagate the caller's context through the full call path, or remove the unused context.Context parameter if the operation is purely local.

*scope: `openmeter/`* · *check: `forbidden_content`*

**Why:** Context carries the Ent transaction driver (entutils), request-scoped telemetry (OTel spans), and cancellation signals. Replacing the caller's context with context.Background() drops the transaction, causing partial writes, and severs traces. If an operation truly does not need a context, remove the parameter rather than using a background context.

### `credits-001` — When credits.enabled is false, every wiring layer that touches ledger accounts must be independently guarded: (1) ledger-backed credit HTTP handlers in api/v3/server, (2) customer ledger hooks in app/common/customer.go, (3) namespace/default-account provisioning. Missing any one layer silently re-enables ledger writes.

*scope: `app/common/`* · *check: `architectural_constraint`*

**Why:** The credits feature is cross-cutting — ledger writes are initiated from three independent call graphs (HTTP handlers, customer hooks, namespace provisioning). No single choke point gates all paths. If any layer is missed, ledger tables are written even when credits are supposed to be off, causing data integrity issues and billing errors.

### `build-001` — Always include -tags=dynamic in all Go build and test invocations. Omitting this tag causes confluent-kafka-go to fail to link against librdkafka.

*scope: `.`* · *check: `required_pattern`*

**Why:** confluent-kafka-go uses CGo with dynamic librdkafka linking. Without -tags=dynamic the build uses a stub that errors at link time. The Makefile sets GO_BUILD_FLAGS=-tags=dynamic for this reason; manual go build or go test commands must replicate this.

### `test-001` — Tests that touch the PostgreSQL database must set POSTGRES_HOST=127.0.0.1 explicitly. Without it, test suites skip silently even when PostgreSQL is running.

*scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** The test harness uses POSTGRES_HOST to discover the database. The default value resolves to a hostname that may not match the local Docker container. Setting it to 127.0.0.1 ensures the tests connect and run rather than being silently skipped, which would hide failing tests.

### `test-003` — Domain test utility packages under openmeter/<domain>/testutils/ must not import app/common. Build test dependencies from underlying package constructors (repos, adapters, services) directly.

*scope: `openmeter/`* · *check: `forbidden_import`*

**Why:** Importing app/common from a domain testutils package creates test-only import cycles because app/common itself imports all domain packages. This forces all domains to be compiled together for any single domain test, and causes import-cycle build errors when any wiring addition creates a new transitive dependency.

### `layer-001` — Domain packages under openmeter/<domain>/ must expose a Service interface at the package root (service.go or <domain>.go). Ent/PostgreSQL access must live in the adapter/ sub-package. HTTP translation must live in httpdriver/ or httphandler/.

*scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** The layered Service/Adapter/HTTP pattern is enforced throughout the codebase to keep persistence concerns from leaking into business logic and HTTP concerns from leaking into adapters. Violating this makes the domain package difficult to test in isolation and creates transitive import cycles.

### `layer-002` — Google Wire provider sets must live exclusively in app/common/. cmd/* binaries must only call the Wire-generated initializeApplication. Business logic must not be placed in cmd/*/main.go.

*scope: `cmd/`* · *check: `architectural_constraint`*

**Why:** Wire provider sets in app/common are the single compile-time-verified wiring graph. Placing providers or constructor logic in cmd/* creates duplicate, unverifiable graphs. Business logic in cmd/*/main.go cannot be tested without running the full binary and is invisible to Wire's dependency analysis.

### `kafka-001` — Kafka topic provisioning must go through app/common's KafkaTopicProvisioner. Do not call Kafka admin APIs directly from domain code.

*scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** Topic provisioning requires specific configuration (replication factor, retention, partitions) that is centralized in app/common. Domain code calling Kafka admin APIs directly creates hidden operational dependencies, bypasses retry/error handling, and complicates testing with mocked providers.

### `kafka-002` — Cross-binary communication must use the three named Kafka topics (ingest events, system events, balance worker events) via the Watermill eventbus.Publisher. Never use HTTP calls or shared in-memory state between binaries.

*scope: `openmeter/watermill/`* · *check: `architectural_constraint`*

**Why:** The multi-binary architecture isolates failure domains. In-process communication or HTTP calls between binaries couple failure domains and eliminate independent scaling. The three-topic isolation prevents a hot ingest producer from starving billing system event consumers.

### `migration-001` — Atlas migrations must be strictly sequential by timestamp. Two parallel branches adding migrations with overlapping or identical timestamps cause atlas.sum hash chain conflicts that cannot be merged cleanly.

*scope: `tools/migrate/migrations/`* · *check: `architectural_constraint`*

**Why:** atlas.sum records a linear cryptographic hash chain over migration files. Two branches that both append migrations produce diverging chains. On merge, one chain must be discarded and the migration regenerated against the rebased schema. The /rebase skill documents the correct procedure.

### `migration-002` — atlas.sum must be updated and committed alongside every new migration. Never commit .up.sql or .down.sql files without also committing the updated atlas.sum.

*scope: `tools/migrate/migrations/`* · *check: `architectural_constraint`*

**Why:** atlas.sum is the integrity seal for the migration chain. CI runs `atlas migrate validate` which verifies that atlas.sum matches the current migration file set. Committing migration files without atlas.sum causes CI to fail and leaves the migration in an unvalidatable state.

### `security-001` — Never commit config.yaml, .env*, *.key, *.pem, or any file containing credentials or secrets to the repository. config.yaml is gitignored and generated from config.example.yaml.

*scope: `.`* · *check: `file_naming`*

**Why:** CI runs Trufflehog with fail_on_findings=true on every PR and main push. Any committed secret triggers an immediate CI failure and requires a secret rotation. config.yaml is intentionally gitignored because it contains database passwords and API keys for local development.

### `charges-001` — Charge lifecycle operations must be driven through the charges.Service entry points: Create, AdvanceCharges, ListCustomersToAdvance, ApplyPatches, and HandleCreditPurchaseExternalPaymentStateTransition. Never reach into the charges adapter directly from outside the charges domain.

*scope: `openmeter/billing/`* · *check: `architectural_constraint`*

**Why:** charges.Service enforces state machine transitions, advisory locking (pg_advisory_xact_lock per customer), and Watermill event publishing. Bypassing the service and calling the adapter directly skips all invariant enforcement, can produce invalid state machine transitions, and silently drops billing events.

### `namespace-001` — Register all namespace.Handler implementations in cmd/server/main.go before the call to initNamespace(). Handlers registered after initNamespace() do not receive CreateNamespace for the default namespace during startup.

*scope: `cmd/server/`* · *check: `architectural_constraint`*

**Why:** The namespace.Manager fans out CreateNamespace to all registered handlers when initNamespace() is called. Handlers registered after this point miss the default-namespace provisioning call, leaving their subsystem (ClickHouse table, Kafka topic, Ledger account) uninitialized for the default tenant.

### `billing-002` — Invoice and charge state transitions must go through the stateless-backed InvoiceStateMachine or generic Machine[CHARGE,BASE,STATUS]. Never manipulate invoice or charge status fields directly.

*scope: `openmeter/billing/`* · *check: `architectural_constraint`*

**Why:** The state machine enforces valid transition sequences and fires actions (DB save, event publish, external app calls) atomically during each transition. Direct field manipulation bypasses transition validation, skips post-transition actions, and produces inconsistent invoice records that downstream workers cannot process.

### `di-001` — When an optional feature is disabled (e.g., Svix not configured, credits.enabled=false), the Wire provider function must return a noop implementation rather than nil or a conditional check scattered through business logic.

*scope: `app/common/`* · *check: `architectural_constraint`*

**Why:** Returning noop implementations keeps the rest of the DI graph uniform — callers never need to nil-check the injected interface. nil interfaces cause panics at call sites that are distant from the configuration check. The noop pattern (ledgernoop.AccountService{}, webhooknoop.New()) is consistently applied for credits and Svix throughout app/common.

### `servicehook-001` — Cross-domain lifecycle callbacks (e.g., billing reacting to subscription events, entitlement validators blocking customer deletion) must use the ServiceHookRegistry (models.ServiceHooks) or SubscriptionCommandHook pattern. Do not introduce direct package imports between domains for lifecycle callbacks.

*scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** Direct cross-domain imports for lifecycle callbacks create circular import cycles (billing imports customer imports billing). The ServiceHookRegistry is registered at Wire time in app/common and avoids compile-time circular imports while enabling multiple listeners without modifying the originating service.

### `lock-001` — Per-customer billing mutations (invoice creation, charge advancement) must acquire the advisory lock via billing.Service.WithLock before any DB writes. The lockr.Locker requires an active Postgres transaction in context.

*scope: `openmeter/billing/`* · *check: `architectural_constraint`*

**Why:** Without the advisory lock, concurrent billing-worker goroutines processing the same customer can race to create duplicate invoice lines or advance charges redundantly. pg_advisory_xact_lock($1) is automatically released on transaction commit/rollback, preventing stale lock accumulation.

### `erosion-god-function` — God-function: CC>15. Complexity concentrating here — SlopCodeBench shows this is the #1 agent failure mode. Split before it grows further.

*check: `complexity_threshold`*

### `decay-empty-catch` — Empty catch/except block — error silently swallowed. SlopCodeBench shows error handling degrades first while core functionality stays.

*check: `forbidden_content`*

### `security-hardcoded-secret` — Possible hardcoded secret/API key in source code.

*check: `forbidden_content`*

### `security-debug-left-behind` — Debug breakpoint left in code. Will halt execution in production.

*check: `forbidden_content`*

### `android-layer-viewmodel-context` — 

*check: `architectural_constraint`*

**Why:** ViewModel must be lifecycle-independent. Referencing Context/View from ViewModel creates memory leaks and breaks testability.

### `android-layer-fragment-network` — 

*check: `architectural_constraint`*

**Why:** Fragments must not make network calls directly. All data flows through Repository → ViewModel → Fragment.

### `android-layer-fragment-db` — 

*check: `architectural_constraint`*

**Why:** Fragments must not access persistence directly. Data layer is the repository's responsibility.

### `android-layer-activity-db` — 

*check: `architectural_constraint`*

**Why:** Activities must not access persistence directly.

### `android-lifecycle-globalscope` — GlobalScope ignores lifecycle — coroutines leak on configuration change or process death. Use viewModelScope, lifecycleScope, or inject a supervised scope.

*check: `forbidden_content`*

### `swift-layer-view-network` — 

*check: `architectural_constraint`*

**Why:** SwiftUI Views must not make network calls. Data fetching belongs in ViewModel or Repository.

### `swift-layer-view-userdefaults` — 

*check: `architectural_constraint`*

**Why:** Views must not access persistence directly. Use a repository or data manager.

### `typescript-react-dom-manipulation` — 

*check: `architectural_constraint`*

**Why:** Direct DOM manipulation in React components breaks the virtual DOM model and causes subtle rendering bugs.

### `python-safety-bare-except` — Bare except catches SystemExit, KeyboardInterrupt, and GeneratorExit. Use except Exception: at minimum.

*check: `forbidden_content`*

### `python-safety-eval-exec` — eval/exec executes arbitrary code — critical security risk if input is user-controlled.

*check: `forbidden_content`*

## Tradeoff Signals (warn)

### `dep-004` — Never spawn goroutines outside the oklog/run.Group in worker and server binaries. Goroutines spawned outside run.Group bypass graceful shutdown and can leak resources.

*source: `deep_scan`*

**Why:** Goroutine spawned outside run.Group (bypasses graceful shutdown) is an explicit violation signal for the multi-binary orchestration trade-off. cmd/server/main.go and all worker main.go files orchestrate lifecycle through an oklog/run.Group with explicit Start and Interrupt functions.

**Path glob:** `cmd/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "^\\s*go func\\("
    ],
    "must_not_match": [
      "run\\.Add",
      "run\\.Group"
    ]
  }
]
```

</details>

### `ent-001` — Never write hand-crafted SQL queries (db.Exec, db.Query) alongside Ent queries in the same adapter. Mixing raw SQL with Ent breaks the single-schema-source guarantee and undermines Atlas diffing.

*source: `deep_scan`* · *scope: `openmeter/`* · *check: `forbidden_content`*

**Why:** Hand-written SQL added alongside Ent queries is an explicit violation signal. Ent ORM + Atlas-generated migrations as the schema pipeline means Atlas diffs the Ent schema against migration history to produce deterministic SQL; introducing raw SQL outside this pipeline creates schema drift invisible to Atlas.

**Path glob:** `openmeter/**/adapter/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "\\.ExecContext\\(|\\bdb\\.QueryContext\\("
    ],
    "must_not_match": [
      "// clickhouse",
      "// ClickHouse"
    ]
  }
]
```

</details>

### `ent-002` — Every new database table or relation must have a corresponding openmeter/ent/schema/*.go file. Never create tables via ad-hoc DDL outside tools/migrate/migrations/ and the Atlas pipeline.

*source: `deep_scan`*

**Why:** New table created without corresponding openmeter/ent/schema/*.go is an explicit violation signal. Ad-hoc DDL outside tools/migrate/migrations/ is another violation signal. Atlas can only diff and validate migrations it knows about through the Ent schema pipeline.

### `gen-007` — When adding a new Ent view entity (schema using ent.View), verify whether Atlas picks it up via `atlas migrate --env local diff`. If no changes are reported, add an explicit SQL migration for the view DDL — Atlas generator support for views is incomplete in this repo.

*scope: `openmeter/ent/schema/`* · *check: `architectural_constraint`*

**Why:** Ent view entities generate query code under openmeter/ent/db/ but do not appear in openmeter/ent/db/migrate/schema.go or migrate.Tables. Atlas therefore cannot detect them via schema diff and reports no changes, silently leaving the view undeployed in production databases.

### `credits-002` — When credits are disabled and a ledger account backfill is needed, construct concrete ledger account + resolver adapters directly instead of relying on the default Wire DI outputs — which return noop implementations when credits.enabled=false.

*scope: `openmeter/ledger/`* · *check: `architectural_constraint`*

**Why:** app/common wires ledger account services/resolvers to noop implementations when credits.enabled=false. Any code that must write real ledger_accounts or ledger_customer_accounts rows cannot use the default DI outputs and must bypass them by constructing the concrete adapters directly.

### `test-002` — In tests, prefer t.Context() over context.Background() when a *testing.T or testing.TB is available. This ties cancellation and lifecycle to the test harness.

*scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** t.Context() returns a context that is cancelled when the test ends, ensuring goroutines and DB connections spawned by the test are cleaned up. context.Background() outlives the test and can cause resource leaks or cross-test interference in parallel test runs.

### `test-004` — Usage-based billing lifecycle tests must drive behavior through charges.Service.Create, AdvanceCharges, and ApplyPatches rather than calling lower-level charge adapter methods directly. Use MockStreamingConnector with explicit StoredAt values to model late-arriving usage.

*scope: `openmeter/billing/charges/`* · *check: `architectural_constraint`*

**Why:** The charges service enforces state machine transitions and transaction boundaries. Calling adapter methods directly bypasses these guards, making tests pass in artificial conditions that would fail in production. MockStreamingConnector with explicit StoredAt exercises the real stored-at cutoff logic in finalization.

### `layer-003` — Service input types must use the <Verb><Noun>Input naming pattern (e.g. CreateCustomerInput, ListInvoicesInput) and implement models.Validator.

*scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** Consistent input type naming enables code search, review, and tooling to locate all inputs for a domain. Implementing models.Validator ensures validation is collocated with the type and runs before service logic, preventing invalid inputs from reaching the adapter or database.

### `notification-001` — Notification event payload types must be versioned. Never hardcode a version constant without providing a migration path for existing payloads.

*scope: `openmeter/notification/`* · *check: `architectural_constraint`*

**Why:** Notification payloads are delivered to external systems via Svix webhooks. Breaking payload changes without versioning cause consumer failures. The Watermill consumer may replay old messages after a deploy; unversioned payloads cannot be safely decoded if the struct changes.

### `api-v3-001` — List endpoints in the v3 API must use the shared cursor pagination and filter types defined in the AIP TypeSpec packages. Do not introduce custom pagination or filter structs per endpoint.

*scope: `api/v3/`* · *check: `architectural_constraint`*

**Why:** Consistent cursor/filter types across all v3 list endpoints enable the shared cursor parsing infrastructure and ensure clients can use the same pagination pattern everywhere. Custom per-endpoint types cause divergent client code and break the generic list handler infrastructure.

### `billing-001` — New billing backend integrations (e.g., a new payment provider) must implement the billing.InvoicingApp interface and register via app.Service.RegisterMarketplaceListing(). Do not hardcode provider-specific logic inside billing.Service.

*scope: `openmeter/billing/`* · *check: `architectural_constraint`*

**Why:** The App Factory / Registry pattern keeps billing.Service decoupled from specific payment providers. RegisterMarketplaceListing allows dynamic plugging of Stripe, Sandbox, CustomInvoicing, and future providers without modifying billing core. Hardcoding provider logic creates untestable conditional branches and makes provider swap impossible without service changes.

### `di-002` — Binary-specific Wire sets must follow the openmeter_<binary>.go naming convention in app/common/. Each file combines domain Wire sets relevant to that binary.

*scope: `app/common/`* · *check: `file_naming`*

**Why:** The naming convention makes it immediately clear which file governs each binary's DI composition. Deviating from it requires searching all app/common files to locate where a binary's Wire set is defined, adding friction to onboarding and debugging.

### `http-001` — HTTP handlers in domain packages must use pkg/framework/transport/httptransport.Handler[Request,Response] with a RequestDecoder and ResponseEncoder. Do not write raw http.Handler implementations that directly parse requests or write responses.

*scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** httptransport.Handler provides a uniform decode/operation/encode pipeline with chained error encoders and operation.Middleware support. Raw http.Handler implementations bypass the GenericErrorEncoder chain (which maps domain errors to RFC 7807 Problem Details), the OTel trace instrumentation, and the request validation middleware.

### `http-002` — Domain-level validation errors must use models.GenericValidationError, models.GenericNotFoundError, models.GenericConflictError and related types so GenericErrorEncoder maps them to the correct HTTP status codes. Do not write errors.New() with HTTP status embedded in the message string.

*scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** GenericErrorEncoder uses type matching (HandleErrorIfTypeMatches) to map domain error types to HTTP status codes (404, 400, 409, 403, etc.) and render RFC 7807 Problem Details. Plain errors.New() strings fall through to 500 Internal Server Error regardless of intent, causing incorrect API responses.

### `erosion-growing-complexity` — Complex function approaching god-function territory (CC>10). Track this — if CC grows between scans, it's eroding.

*check: `complexity_threshold`*

### `erosion-god-class` — Class has 20+ methods — likely accumulating responsibilities. Agents patch new logic here instead of creating focused classes.

*check: `size_threshold`*

### `erosion-monster-file` — File exceeds 600 lines. In agent-assisted codebases, large files grow because agents append rather than refactor.

*check: `size_threshold`*

### `erosion-many-params` — Function with 7+ parameters. Signal of a function doing too much or missing a data class/struct.

*check: `forbidden_content`*

### `decay-disabled-test` — Disabled/skipped test. Tests get disabled when agents can't fix them — this hides regressions. The paper shows error-mode tests fail first.

*check: `forbidden_content`*

### `decay-todo-fixme-hack` — FIXME/HACK/XXX marker — acknowledged technical debt. Track these: if count grows between scans, quality is degrading.

*check: `forbidden_content`*

### `decay-catch-log-only` — Catch block only logs the error without re-throwing or propagating. Error is visible in logs but callers don't know it failed.

*check: `forbidden_content`*

### `android-di-service-locator` — Service locator pattern (KoinJavaComponent.get) breaks constructor injection and makes dependencies invisible. Use constructor injection via Koin modules.

*check: `forbidden_content`*

### `swift-safety-force-unwrap` — Force unwrap (!) crashes at runtime if nil. Use guard let, if let, or ?? default.

*check: `forbidden_content`*

### `swift-safety-force-try` — Force try crashes at runtime if throwing function fails. Use do/catch or try?.

*check: `forbidden_content`*

### `typescript-layer-component-fetch` — 

*check: `architectural_constraint`*

**Why:** Components should not fetch data directly. Use hooks, services, or state management (React Query, SWR, etc.).

### `typescript-safety-any-type` — Type escape via 'any' — defeats TypeScript's type system. Use unknown, generics, or proper types.

*check: `forbidden_content`*

### `typescript-react-index-key` — Array index as React key causes rendering bugs when list items are reordered, inserted, or deleted.

*check: `forbidden_content`*

### `python-safety-mutable-default` — Mutable default argument (list/dict) — shared across all calls. Use None default with assignment in body.

*check: `forbidden_content`*

### `python-layer-star-import` — Star import pollutes namespace and hides dependencies. Makes it impossible to trace where a symbol comes from.

*check: `forbidden_content`*

### `python-layer-circular-import` — TYPE_CHECKING guard suggests circular import was encountered. The cycle should be resolved structurally, not worked around.

*check: `forbidden_content`*

## Pattern Divergence (inform)

### `dep-006` — New charge type engines must be registered with billing.Service.RegisterLineEngine() in app/common/charges.go. Do not call RegisterLineEngine from domain packages or cmd/* binaries.

*source: `deep_scan`*

**Why:** billingservice.engineRegistry stores a map[LineEngineType]LineEngine under a RWMutex. Each charge type implements its own Engine and registers it at startup via app/common/charges.go. The service.New() constructor also pre-registers the standard invoice line engine. All engine registration is a side-effect of Wire provider functions.

**Example:**

```
// app/common/charges.go
func NewFlatFeeChargesService(billingService billing.Service, ...) flatfee.Service {
    svc := flatfee.New(...)
    billingService.RegisterLineEngine(svc) // side-effect registration
    return svc
}
```

### `wm-003` — Watermill consumer routers must be built with openmeter/watermill/router.NewDefaultRouter. Do not construct a bare Watermill router without the fixed middleware stack (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout, HandlerMetrics).

*source: `deep_scan`*

**Why:** openmeter/watermill/router/router.go defines NewDefaultRouter with a fixed middleware stack: PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout+RestoreContext, HandlerMetrics. Bypassing this loses dead-letter routing, retries, OTel span propagation, and processing-timeout enforcement across all consumer binaries.

### `layer-005` — v3 API handlers must be organized under api/v3/handlers/<resource>/ sub-packages and implement the generated ServerInterface methods using httptransport.Handler[Request,Response]. Do not place v3 handlers in openmeter/*/httpdriver/.

*source: `deep_scan`*

**Why:** v3 API handler packages are organized per resource group under api/v3/handlers/ (meters, customers, customers/billing, customers/charges, billingprofiles, plans, subscriptions, etc.). Each sub-package implements relevant ServerInterface methods using the httptransport.Handler[Request,Response] pipeline and delegates to domain services. v1 handlers live in openmeter/*/httpdriver/ or openmeter/*/httphandler/.

**Example:**

```
// v3 handler in api/v3/handlers/customers/handler.go
func (h *Handler) ListCustomers(ctx context.Context, req api.ListCustomersRequestObject) (api.ListCustomersResponseObject, error) {
    items, err := h.svc.ListCustomers(ctx, customer.ListCustomersInput{Namespace: req.Params.Namespace})
    if err != nil { return nil, err }
    return api.ListCustomers200JSONResponse{Items: toAPI(items)}, nil
}
```

### `billing-004` — Billing line differences and invoice sync must go through openmeter/billing/worker/subscriptionsync.Service. Do not implement subscription-to-invoice line mapping outside this service.

*source: `deep_scan`*

**Why:** openmeter/billing/worker/subscriptionsync bridges subscription lifecycle events to invoice line generation via SynchronizeSubscription / SynchronizeSubscriptionAndInvoiceCustomer / HandleCancelledEvent / HandleSubscriptionSyncEvent / HandleInvoiceCreation. The reconciler runs this idempotently on crash-recovery to catch missed events.

### `sink-001` — The sink worker must maintain strict three-phase flush ordering: ClickHouse BatchInsert first, then Kafka offset commit, then Redis deduplication update. Reordering these phases breaks exactly-once semantics.

*source: `deep_scan`*

**Why:** Flush ordering is strict: ClickHouse insert then Kafka offset commit then Redis dedupe. After flush, FlushEventHandlers are called post-flush in a goroutine with timeout for downstream notifications. Deviating from this order risks double-counting on consumer restart or losing events before offset commit.

### `reg-001` — Typed registry structs (BillingRegistry, AppRegistry, SubscriptionServiceWithWorkflow) must be used to group cohesive services. Do not add individual service fields to router.Config that already belong to an existing registry.

*source: `deep_scan`*

**Why:** Typed registry structs group logically cohesive services (BillingRegistry exposes ChargesServiceOrNil(), AppRegistry, SubscriptionServiceWithWorkflow) and let ChargesServiceOrNil() encapsulate the credits-disabled nil case. router.Config field for a service already inside a registry is a documented violation keyword.

**Example:**

```
// Correct: access charges via registry
cmd.billingRegistry.ChargesServiceOrNil()

// Wrong: exposing charges separately in router.Config when it belongs to BillingRegistry
```

### `watermill-001` — Unknown Watermill event types received by a NoPublishingHandler must be silently dropped, not returned as errors. Returning errors for unknown event types poisons the DLQ during rolling deploys.

*source: `deep_scan`*

**Why:** Silent drop of unknown event types lets producers and consumers deploy in any order without poisoning DLQs. grouphandler.NoPublishingHandler drops unknown event types keyed on CloudEvents ce_type. Returning error for unknown event type is a documented violation keyword.

**Example:**

```
// Correct: handler drops unknown types
handler := grouphandler.NewNoPublishingHandler(marshaler, handlers...)
// Unknown ce_types not in handlers list are silently ignored
```

### `error-001` — Validation errors with structured field paths must use pkg/models.ValidationIssue with With* copy-on-write methods and WithPathString for field location. Do not create plain error strings for field-level validation failures.

*source: `deep_scan`*

**Why:** pkg/models/validationissue.go defines ValidationIssue as an immutable value type with copy-on-write With* methods. The HTTP layer reads the httpStatusCodeErrorAttribute attribute set via commonhttp.WithHTTPStatusCodeAttribute to produce the correct HTTP status. ValidationIssue carries field paths, component attribution, severity, and HTTP status through service layer boundaries.

**Example:**

```
// Correct: structured validation error
err := models.NewValidationIssue("invalid_amount", "amount must be positive").
    WithPathString("amount").
    WithComponent("charges")
return nil, commonhttp.WithHTTPStatusCodeAttribute(err, http.StatusBadRequest)
```

### `otel-001` — Service methods that initiate significant work must create an OTel span from the injected trace.Tracer and defer span.End(). Do not skip tracing in long-running operations.

*source: `deep_scan`*

**Why:** Every entry point (HTTP handlers, Kafka consumers, Ent adapters) is instrumented with OpenTelemetry. trace.Tracer is injected via Wire into service constructors. OTel metric.Meter is used in grouphandler and sink worker. All domain calls must propagate ctx (no context.Background()/TODO()).

**Example:**

```
func (s *svc) DoWork(ctx context.Context, id string) error {
    ctx, span := s.tracer.Start(ctx, "svc.DoWork")
    defer span.End()
    return s.adapter.Write(ctx, id)
}
```

### `cfg-001` — All Viper config defaults must be registered through app/config/config.go's SetViperDefaults function which calls every Configure* sub-function. Do not call viper.SetDefault directly in cmd/* binaries.

*source: `deep_scan`*

**Why:** SetViperDefaults is the single registration point calling every Configure* function. config.Configuration is the single shared config type for all binaries. Scattering viper.SetDefault calls across cmd/* binaries creates divergence between binary configuration models.

**Path glob:** `cmd/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "viper\\.SetDefault\\("
    ]
  }
]
```

</details>

### `llmcost-001` — All LLM model IDs must be normalized via llmcost.NormalizeModelID before being stored or resolved. Storing unnormalized IDs causes mismatches during price resolution because provider aliases and version suffixes produce different strings.

*source: `deep_scan`*

**Why:** llmcost/normalize.go defines NormalizeModelID: strips version/region suffixes and normalises provider aliases. NormalizeModelID must be called before any price store or resolve. Unnormalized model IDs break the namespace-override precedence lookup.

**Example:**

```
normalizedID := llmcost.NormalizeModelID(rawModelID)
price, err := svc.ResolvePrice(ctx, llmcost.ResolvePriceInput{ModelID: normalizedID})
```

### `subject-001` — Subject lifecycle events must be routed via subject.Service.RegisterHooks. Do not implement subject-related side-effects by directly calling subject adapter methods from other domains.

*source: `deep_scan`*

**Why:** openmeter/subject exposes ServiceHooks for lifecycle events via subject.Service.RegisterHooks. Direct calls from balance-worker (openmeter/entitlement/balanceworker) go through the service interface. Bypassing the service creates coupling between balance-worker and the subject adapter, making the subject domain untestable in isolation.