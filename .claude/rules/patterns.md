## Communication Patterns

### Layered Domain Service / Adapter / HTTP
- **Scope:** `openmeter/billing`, `openmeter/billing/service`, `openmeter/billing/adapter`, `openmeter/customer`, `openmeter/entitlement`, `openmeter/subscription`, `openmeter/notification`, `openmeter/meter`, `openmeter/ledger`, `openmeter/productcatalog`, `openmeter/app`, `openmeter/llmcost`
- **When:** Adding any business-logic capability to a domain under openmeter/<domain>/, separating orchestration from persistence and HTTP translation
- **How:** Each domain declares a Service interface and an Adapter interface at the package root (service.go / adapter.go). The concrete service lives in <domain>/service/, the Ent-backed adapter in <domain>/adapter/, and HTTP handlers in <domain>/httpdriver/ (v1) or api/v3/handlers/<resource>/ (v3). Service calls only the Adapter interface; the adapter is the single DB boundary. Composite service interfaces are assembled from fine-grained sub-interfaces so callers depend on the narrowest slice.

### entutils.TransactingRepo context-propagated Ent transactions
- **Scope:** `openmeter/billing/adapter`, `openmeter/billing/charges/adapter`, `openmeter/customer/adapter`, `openmeter/notification/adapter`, `openmeter/ledger`, `openmeter/entitlement`, `openmeter/subscription`
- **When:** Every adapter method body that reads or writes via Ent and must compose with a caller-supplied transaction or start its own
- **How:** TransactingRepo(ctx, a, func(ctx, tx *adapter)(T,error)) reads the *TxDriver from ctx; if present it rebinds a.db to the caller's transaction via WithTx(), otherwise it runs on Self(). The adapter must implement the Tx/WithTx/Self triad. lockr.LockForTX queries entutils.GetDriverFromContext to find the tx driver — so any lock or nested adapter call inherits the same transaction.

### Per-customer advisory lock via lockr inside an Ent transaction
- **Scope:** `openmeter/billing`, `openmeter/billing/service`, `openmeter/billing/adapter`, `openmeter/billing/charges`, `openmeter/entitlement`
- **When:** Serializing concurrent invoice/charge mutations for the same customer so concurrent billing-worker goroutines and API requests cannot race on invoice or line creation
- **How:** billing.Service.WithLock wraps transaction.RunWithNoValue and transactionForInvoiceManipulation: it UpsertCustomerLock (idempotent insert with OnConflict DoNothing) then LockCustomerForUpdate, which inside the ctx-bound transaction issues a SELECT ... FOR UPDATE on the single BillingCustomerLock row keyed (namespace, customer_id). lockr.Locker.LockForTX (pg_advisory_xact_lock) is the generic equivalent for per-charge locks; getTxClient verifies transaction_timestamp() != statement_timestamp() so it errors outside a real transaction. The lock auto-releases on commit/rollback.
- **Applicable when:** openmeter/ent/schema/billing.go:1356 declares a UNIQUE index on (namespace, customer_id) for BillingCustomerLock — the (namespace, customerID) tuple maps to at most one row, so the SELECT FOR UPDATE / advisory lock serializes exactly that customer and nothing else. BillingCustomerOverride mirrors this with UNIQUE (namespace, customer_id) at openmeter/ent/schema/billing.go:258.
- **Do NOT apply when:**
  - Lock-key columns lack a UNIQUE index on the locked entity — pkg/framework/lockr/key.go hashes scopes to a uint64, so charges.NewLockKeyForCharge(openmeter/billing/charges/lock.go:15) keys on (namespace, charge, id) which is the charge primary key; reusing this shape for a non-unique key (e.g. a status or type column) would silently serialize unrelated rows under one hash
  - Caller is outside an active Postgres transaction — pkg/framework/lockr/locker.go:134 returns 'lockr only works in a postgres transaction' when statement_timestamp()==transaction_timestamp(), so LockForTX from an autocommit connection acquires nothing
  - Acquisition is wrapped in context.WithTimeout — pkg/framework/lockr/locker.go:91-92 documents that pgx cancels the connection on ctx cancel, corrupting the tx; use pgdriver.WithLockTimeout instead

### Kafka + Watermill pub/sub with three prefix-routed topics
- **Scope:** `openmeter/watermill`, `openmeter/watermill/eventbus`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification`, `openmeter/sink`, `openmeter/ingest`
- **When:** Any async domain-event delivery between the seven binaries (subscription lifecycle, invoice advance, ingest flush, balance recalculation)
- **How:** eventbus.New wraps cqrs.NewEventBusWithConfig; GeneratePublishTopic does strings.HasPrefix on the EventName against ingestevents.EventVersionSubsystem+'.' and balanceworkerevents.EventVersionSubsystem+'.' and routes to IngestEventsTopic / BalanceWorkerEventsTopic respectively, defaulting all other prefixes to SystemEventsTopic with no error. Producers call Publisher.Publish or WithContext(ctx).PublishIfNoError. Consumers build routers via router.NewDefaultRouter and dispatch through grouphandler.NoPublishingHandler.
- **Applicable when:** openmeter/watermill/eventbus/eventbus.go:141-142 has a default switch case that returns SystemEventsTopic for any unrecognized EventName prefix — so the routing invariant holds ONLY for event families whose EventName() begins with a registered EventVersionSubsystem constant (ingest or balance-worker); everything else is silently treated as a system event.
- **Do NOT apply when:**
  - Event family intended for the ingest or balance-worker topic whose EventName() lacks the matching EventVersionSubsystem prefix — openmeter/watermill/eventbus/eventbus.go:141 silently routes it to SystemEventsTopic, bypassing topic isolation
  - Producer in a binary other than balance-worker emitting balanceworkerevents.* — the BalanceWorkerEventsTopic is a dedicated recalculation queue consumed only by the balance-worker

### NoPublishingHandler silent-drop dispatch by CloudEvents ce_type
- **Scope:** `openmeter/watermill/grouphandler`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification/consumer`
- **When:** Consumer-side dispatch of a Kafka message to typed handlers in a worker router
- **How:** NoPublishingHandler.Handle reads the ce_type via marshaler.NameFromMessage, looks it up in typeHandlerMap; if no handler is registered it increments an 'ignored' metric and returns nil (ACK, silent drop). For a matched type it unmarshals once and fans out to all registered handlers via errors.Join(lo.Map(...)) so any handler failure surfaces and triggers Watermill retry. Handlers must use msg.Context().
- **Applicable when:** openmeter/watermill/grouphandler/grouphandler.go:48-54 returns nil for any ce_type not in typeHandlerMap — the silent-drop contract is correct ONLY for consumers that must tolerate producer/consumer version skew during rolling deploys; it cannot distinguish 'unknown event type' from 'known type, payload version the consumer cannot decode'.
- **Do NOT apply when:**
  - Multiple handlers registered for the same ce_type that mutate the shared event pointer — grouphandler.go:57-66 unmarshals one instance and passes the same pointer to every handler via errors.Join, so concurrent mutation races
  - Handler that needs unknown/undecodable versions surfaced for DLQ — grouphandler.go:54 ACKs and drops; returning an error here instead would poison the DLQ for valid messages of other families on the same topic

### Sink worker three-phase flush (ClickHouse -> Kafka offset -> Redis dedupe)
- **Scope:** `openmeter/sink`, `cmd/sink-worker`
- **When:** High-throughput batch ingestion of raw CloudEvents from Kafka into ClickHouse with exactly-once semantics
- **How:** Sink.flush dedupes in-batch, then executes strictly: (1) persistToStorage -> Storage.BatchInsert into ClickHouse, (2) Consumer.StoreMessage for each Kafka offset (sorted so the largest offset is stored last), (3) dedupeSet (Redis SETNX with retry) only when a Deduplicator is configured. After all three phases FlushEventHandler.OnFlushSuccess is invoked in a goroutine bounded by FlushSuccessTimeout so the post-flush balance-recalculation notification never blocks the consumer loop.
- **Applicable when:** openmeter/sink/sink.go:327-372 orders persistToStorage (phase 1) strictly before Consumer.StoreMessage (phase 2) and dedupeSet (phase 3); the exactly-once guarantee holds ONLY while ClickHouse is written before the Kafka offset is committed — on consumer restart an uncommitted offset re-delivers messages not yet in ClickHouse.
- **Do NOT apply when:**
  - Reordering so Redis dedupe is set before the Kafka offset commit — openmeter/sink/sink.go:350-372 sets dedupe last; a crash after dedupe but before offset commit would mark events processed while ClickHouse re-reads from the uncommitted offset, dropping them
  - Calling FlushEventHandler.OnFlushSuccess synchronously — openmeter/sink/sink.go:391-399 always wraps it in a goroutine with FlushSuccessTimeout; a synchronous call blocks the main sink loop and causes Kafka partition backpressure

### Namespace Manager fan-out (multi-tenancy provisioning)
- **Scope:** `openmeter/namespace`, `cmd/server`
- **When:** Provisioning/deprovisioning a tenant across all subsystems (ClickHouse streaming, Kafka ingest, Ledger)
- **How:** namespace.Manager holds a slice of registered Handler implementations. createNamespace and DeleteNamespace iterate every handler and aggregate failures with errors.Join (no short-circuit). RegisterHandler appends handlers; CreateDefaultNamespace calls createNamespace with the default name. The default namespace is protected from deletion.
- **Applicable when:** openmeter/namespace/namespace.go:92 CreateDefaultNamespace fans out only over handlers already present in the slice — so a Handler is provisioned for the default namespace ONLY if RegisterHandler was called before CreateDefaultNamespace at startup; handlers registered afterward miss default-namespace provisioning.
- **Do NOT apply when:**
  - Registering a namespace.Handler after CreateDefaultNamespace has already run — openmeter/namespace/namespace.go:92-103 iterates only the handlers present at call time, so a late handler's subsystem is never initialized for the default tenant

### ServiceHook Registry for cross-domain lifecycle callbacks
- **Scope:** `openmeter/customer`, `openmeter/subscription`, `openmeter/app`, `app/common`
- **When:** Reacting to another domain's entity lifecycle (billing reacting to subscription/customer events, ledger reacting to customer creation) without importing that domain's service
- **How:** pkg/models.ServiceHookRegistry[T] fans out PreCreate/PostCreate/PreUpdate/PostUpdate/PreDelete/PostDelete to all registered hooks under an RWMutex. Re-entrancy is prevented by a per-registry loop key derived from pointer identity (fmt.Sprintf('service-hook-registry-%p', r)) stored in ctx. Domain services embed the registry and expose RegisterHooks; registration happens as a side-effect inside app/common provider functions.
- **Applicable when:** pkg/models/servicehook.go:42 derives the loop-prevention key from the registry's own pointer (fmt.Sprintf('...%p', r)) — so the re-entrancy guard is correct ONLY while the registry is shared by pointer; copying the registry value produces a different %p and defeats loop prevention.
- **Do NOT apply when:**
  - Registering a hook inside a domain package's own constructor instead of an app/common provider — Wire models types not side-effects, so omitting the provider from a binary's wire.Build silently drops the hook (no compile error)
  - Pre-mutation blocking validation — that belongs in the customer RequestValidator registry (openmeter/customer/requestvalidator.go), not in post-lifecycle ServiceHooks

### httptransport decode/operate/encode pipeline with GenericErrorEncoder chain
- **Scope:** `openmeter/billing/httpdriver`, `openmeter/customer/httpdriver`, `openmeter/meter/httphandler`, `api/v3/handlers`
- **When:** Every v1 (httpdriver) and v3 (api/v3/handlers) HTTP endpoint
- **How:** httptransport.NewHandler composes a RequestDecoder, an operation.Operation, and a ResponseEncoder, and appends commonhttp.GenericErrorEncoder as the last error encoder. Encoders return bool (first match wins, short-circuiting double-writes). Domain errors are models.Generic* sentinels matched by type; ValidationIssue HTTP status is carried as an attribute read by HandleIssueIfHTTPStatusKnown.

### Google Wire DI with app/common provider sets and noop-for-disabled-features
- **Scope:** `app/common`, `cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`
- **When:** Composing the dependency graph for each of the seven binaries
- **How:** Each cmd/<binary>/wire.go declares wire.Build over composite provider sets defined in app/common/ (per-domain files plus openmeter_<binary>.go). Domain packages expose plain constructors and never import app/common. Optional features (credits.enabled=false, Svix unconfigured) are gated by returning noop interface implementations rather than nil. Credits is guarded independently at four wiring layers.

### Tagged-union domain models with constructor-only construction (Charge, ChargeIntent, InvoiceLine)
- **Scope:** `openmeter/billing`, `openmeter/billing/charges`
- **When:** Modeling billing entities that have several mutually exclusive sub-types requiring exhaustive dispatch
- **How:** Charge/ChargeIntent carry a private meta.ChargeType discriminator set only by NewCharge[T]/NewChargeIntent[T]; InvoiceLine carries a private InvoiceLineType set only by NewStandardInvoiceLine/NewGatheringInvoiceLine. Typed accessors (AsFlatFeeCharge / AsUsageBasedCharge / AsCreditPurchaseCharge / AsStandardLine / AsGatheringLine) return an error on type mismatch. A struct literal leaves the discriminator zero-valued and all accessors error.

### Invoice / Charge state machine (stateless library, sync.Pool backed)
- **Scope:** `openmeter/billing`, `openmeter/billing/service`, `openmeter/billing/charges`
- **When:** Driving StandardInvoice or charge lifecycle transitions
- **How:** billing/service/stdinvoicestate.go builds a *stateless.StateMachine bound to Invoice.Status via external storage, pooled in invoiceStateMachineCache (sync.Pool). FireAndActivate fires a trigger and persists; advancementStrategy switches between inline AdvanceUntilStateStable and publishing AdvanceStandardInvoiceEvent for the billing-worker. The generic charges Machine[CHARGE,BASE,STATUS] uses value-copy WithStatus/WithBase semantics.

### App Factory / Marketplace Registry (InvoicingApp protocol)
- **Scope:** `openmeter/app`, `openmeter/billing`
- **When:** Plugging a billing backend (Stripe, Sandbox, CustomInvoicing) into the invoice state machine without hardcoding it
- **How:** app.Service exposes RegisterMarketplaceListing; each concrete app self-registers a factory in its constructor and implements billing.InvoicingApp (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice). Invoices passed to app callbacks are read-only snapshots; external IDs are returned via the UpsertResults/FinalizeStandardInvoiceResult builder whose MergeIntoInvoice is applied under billing-service control.

### TypeSpec single-source API generation (v1 + v3 + three SDKs)
- **Scope:** `api/spec`, `openmeter/server/router`, `api/v3/server`, `api/v3/handlers`
- **When:** Adding or changing any HTTP endpoint, request/response type, or SDK contract
- **How:** Endpoints are authored in TypeSpec under api/spec/packages/legacy (v1) or api/spec/packages/aip (v3), with route/tag bindings only in the root openmeter.tsp. make gen-api compiles to api/openapi.yaml + api/v3/openapi.yaml then oapi-codegen produces api/api.gen.go, api/v3/api.gen.go, and the Go/JS/Python SDKs; make generate then propagates to Ent/Wire/Goverter/Goderive. Both regen steps are mandatory.

## Integrations

| Service | Purpose | Integration point |
|---------|---------|-------------------|
| PostgreSQL | Primary relational store for all billing/customer/entitlement/subscription/notification/ledger/meter/secret entities via Ent ORM | `openmeter/ent/schema/ (source of truth), generated openmeter/ent/db/, Atlas migrations in tools/migrate/migrations/; accessed via *entdb.Client through entutils.TransactingRepo` |
| ClickHouse | Append-only analytics store for raw usage events; queried for meter aggregations and batch-inserted by the sink worker | `openmeter/streaming/clickhouse/ (queries); openmeter/sink/storage.go Storage.BatchInsert called from openmeter/sink/sink.go:330 persistToStorage` |
| Kafka (confluent-kafka-go + Watermill) | Durable event bus for domain events and raw usage ingestion, isolated into three named topics | `openmeter/watermill/eventbus/eventbus.go (prefix routing); openmeter/watermill/router/router.go; confluent-kafka-go used directly in openmeter/sink/sink.go for the ingest consumer` |
| Redis | Optional deduplication store for ingest events to prevent double-counting on retry | `openmeter/dedupe/redisdedupe/ (in-memory LRU fallback in openmeter/dedupe/memorydedupe/); used in openmeter/sink/sink.go:354 dedupeSet phase` |
| Svix | Outbound webhook delivery for notification events (balance thresholds, invoice events) | `openmeter/notification/webhook/svix/svix.go; noop webhook.Handler fallback when Svix is unconfigured` |
| Stripe | Invoice syncing (upsert draft, finalize, collect payment) and customer sync for billing-enabled namespaces | `openmeter/app/stripe/ implements billing.InvoicingApp; Stripe REST client under openmeter/app/stripe/client/` |
| Sandbox Invoicing App | No-op invoicing app to drive the invoice state machine in dev/test without external dependencies | `openmeter/app/sandbox/ implements billing.InvoicingApp (and InvoicingAppPostAdvanceHook)` |
| CustomInvoicing App | Webhook-driven invoicing allowing external systems to receive invoice payloads and async-confirm sync | `openmeter/app/custominvoicing/ implements InvoicingApp + InvoicingAppAsyncSyncer` |
| GOBL | Currency-safe numeric arithmetic and ISO 4217 currency validation in billing and subscription | `github.com/invopop/gobl imported across productcatalog, subscription, billing, currencies` |
| OpenTelemetry | Distributed tracing and metrics across all binaries | `trace.Tracer injected via Wire; pkg/framework/tracex span helpers; metric.Meter in grouphandler and sink; app/common/telemetry.go bootstraps exporters` |
| TypeSpec compiler | Single source of truth for HTTP API definitions compiling to OpenAPI and Go/JS/Python SDKs | `api/spec/packages/; make gen-api runs tsp compile then oapi-codegen` |
| App Marketplace Registry (extension protocol) | Runtime-pluggable billing backends self-registering without hardcoded references in billing | `openmeter/app/service.go Service.RegisterMarketplaceListing; each app self-registers in its constructor` |
| LineEngine Registry (extension protocol) | Runtime-pluggable billing line calculation dispatch by LineEngineType | `billing.Service.RegisterLineEngine (LineEngineService); registered in app/common/charges.go at Wire startup` |
| ServiceHook & RequestValidator Registries (extension protocol) | Cross-domain lifecycle callbacks and pre-mutation guards without circular imports | `pkg/models/servicehook.go ServiceHookRegistry[T]; openmeter/customer/requestvalidator.go; registered as side-effects in app/common provider functions` |
| namespace.Handler fan-out (extension protocol) | Per-namespace resource provisioning across ClickHouse, Kafka ingest, and Ledger | `openmeter/namespace/namespace.go Manager.RegisterHandler; handlers must register before CreateDefaultNamespace` |

## Pattern Selection Guide

| Scenario | Pattern | Rationale |
|----------|---------|-----------|
| Adding a new domain capability (a new billing sub-feature) | Layered Domain Service/Adapter/HTTP | Define the sub-interface in <domain>/service.go, implement in <domain>/service/, add adapter methods in <domain>/adapter/, wire in app/common/<domain>.go; keeps business logic, persistence, and HTTP independently testable |
| Any adapter DB read/write that may run inside a multi-step transaction | entutils.TransactingRepo / TransactingRepoWithNoValue | Rebinds to the ctx-bound Ent transaction if present (or Self() otherwise), preventing partial writes during AdvanceCharges or invoice-mutation flows |
| Serializing per-customer invoice or charge mutation | billing.Service.WithLock -> lockr / SELECT FOR UPDATE on BillingCustomerLock | Advisory/row lock keyed on the UNIQUE (namespace, customer_id) row auto-releases on commit/rollback and serializes exactly one customer |
| Delivering a domain event to another binary | eventbus.Publisher (prefix-routed to one of three topics) | Topic isolation matches worker topology; producers stay decoupled from consumer topology by routing on the EventVersionSubsystem prefix |
| Consuming events in a worker | router.NewDefaultRouter + grouphandler.NoPublishingHandler | Inherits the fixed DLQ/retry/OTel middleware stack and silently drops unknown ce_types for rolling-deploy tolerance |
| Batch ingesting usage events from Kafka into ClickHouse | Sink three-phase flush | ClickHouse insert -> Kafka offset commit -> Redis dedupe ordering preserves exactly-once on consumer restart; FlushEventHandler runs in a goroutine |
| Reacting to another domain's entity lifecycle without import cycles | ServiceHookRegistry[T] registered in app/common | Avoids circular imports between billing/customer/subscription/ledger; pointer-identity loop key prevents re-entrancy |
| Blocking a customer mutation based on another domain's constraints | Customer RequestValidator registry | Pre-mutation guards fan out via errors.Join before any DB write, keeping billing/entitlement constraints out of the customer package |
| Invoice or charge lifecycle transition | stateless InvoiceStateMachine or generic Machine[CHARGE,BASE,STATUS] | Enforces valid transition sequences and fires post-transition actions atomically; sync.Pool reduces GC pressure on the hot path |
| Adding a new billing backend (payment processor/invoicing system) | Implement billing.InvoicingApp + self-register via RegisterMarketplaceListing | No core billing.Service changes; the read-only invoice snapshot plus UpsertResults builder limit the app's writable surface |
| Disabling an optional subsystem (credits off, no Svix) | Return a noop implementation from the Wire provider | Keeps the DI graph uniform with no nil-checks; credits requires four independent guards (ledger services, customer hooks, ChargesRegistry, v3 handlers) |
| Returning an error to an HTTP client | models.Generic* sentinel + GenericErrorEncoder chain | Type-matched mapping to RFC 7807 problem+json; plain fmt.Errorf falls through to 500 |
| Adding or changing an HTTP endpoint | Author TypeSpec then make gen-api && make generate | Single source of truth makes drift between v1/v3 stubs and three SDKs structurally impossible |

## Quick Pattern Lookup

- **new domain feature** -> Layered Domain Service/Adapter/HTTP in openmeter/<domain>/  *(scope: openmeter/billing, openmeter/customer, openmeter/entitlement, openmeter/subscription)*
- **adapter DB access in a transaction** -> entutils.TransactingRepo / TransactingRepoWithNoValue  *(scope: openmeter/billing/adapter, openmeter/billing/charges/adapter, openmeter/customer/adapter)*
- **per-customer serialization** -> billing.Service.WithLock -> lockr.LockForTX (pg advisory lock in tx)  *(scope: openmeter/billing, openmeter/billing/charges, openmeter/entitlement)*
- **async domain events between binaries** -> eventbus.Publisher prefix-routed to ingest/system/balance-worker topics  *(scope: openmeter/watermill, openmeter/billing/worker, openmeter/entitlement/balanceworker)*
- **consuming worker events** -> router.NewDefaultRouter + grouphandler.NoPublishingHandler (silent drop)  *(scope: openmeter/watermill/grouphandler, openmeter/billing/worker, openmeter/notification/consumer)*
- **batch usage ingestion** -> Sink three-phase flush (ClickHouse -> offset -> Redis dedupe)  *(scope: openmeter/sink, cmd/sink-worker)*
- **lifecycle side-effects across domains** -> ServiceHookRegistry[T] registered in app/common  *(scope: openmeter/customer, openmeter/subscription, app/common)*
- **pre-mutation validation across domains** -> Customer RequestValidator registry  *(scope: openmeter/customer, app/common)*
- **invoice/charge state transitions** -> stateless InvoiceStateMachine or Machine[CHARGE,BASE,STATUS]  *(scope: openmeter/billing, openmeter/billing/charges)*
- **new billing backend** -> Implement billing.InvoicingApp + AppFactory self-registration  *(scope: openmeter/app, openmeter/billing)*
- **optional feature disabled** -> Return noop implementation in Wire provider (four-layer guard for credits)  *(scope: app/common)*
- **HTTP handler** -> httptransport.NewHandler decode/operate/encode + GenericErrorEncoder  *(scope: openmeter/billing/httpdriver, api/v3/handlers)*
- **domain error HTTP mapping** -> models.Generic* sentinel matched by GenericErrorEncoder
- **multi-tenant provisioning** -> namespace.Manager fan-out; register Handlers before CreateDefaultNamespace  *(scope: openmeter/namespace, cmd/server)*
- **DI wiring a binary** -> wire.NewSet in app/common/, wire.Build in cmd/<binary>/wire.go  *(scope: app/common, cmd/server, cmd/billing-worker)*
- **API contract change** -> TypeSpec in api/spec/ -> make gen-api -> make generate  *(scope: api/spec)*

## Decision Chain

**Root constraint:** Operate a high-volume per-tenant usage-metering platform feeding strict financial billing correctness, while shipping stable SDKs in three languages — under a small team that cannot maintain separate repos or hand-synchronized contracts.

- **Multi-binary modular monolith: one shared Go domain tree, seven independently deployable binaries, Kafka as the sole inter-binary channel**: Ingest throughput, balance recalculation, billing advancement, and webhook dispatch have incompatible scaling/failure profiles, but billing correctness needs one typed domain model — so split the processes, share the openmeter/ types.
  - *Violation keyword:* `business logic in cmd/*/main.go`
  - *Violation keyword:* `goroutine spawned outside run.Group`
  - *Violation keyword:* `new cmd/* binary without app/common/openmeter_<binary>.go`
  - *Violation keyword:* `domain package importing app/common`
  - *Violation keyword:* `shared in-memory state between binaries`
  - *Violation keyword:* `HTTP call between binaries`
  - **Google Wire DI with all provider sets in app/common and cross-domain hooks registered as construction side-effects**: ~40 services per binary make hand-wiring error-prone; Wire gives compile-time graph verification, and keeping providers out of domain packages prevents import cycles.
    - *Violation keyword:* `wire.Build calling domain constructors directly`
    - *Violation keyword:* `provider function with validation/computation/panic/os.Exit`
    - *Violation keyword:* `domain package importing app/common`
    - *Violation keyword:* `viper.SetDefault in cmd/*`
    - **credits.enabled feature flag enforced at four independent wiring layers via noop implementations**: Credits writes fan out from HTTP handlers, customer hooks, namespace provisioning, and charge creation — no single choke point — so each wiring layer must guard independently and return a noop, not nil.
      - *Violation keyword:* `ledger-touching Wire provider without creditsConfig.Enabled branch`
      - *Violation keyword:* `BillingRegistry.Charges accessed without ChargesServiceOrNil()`
      - *Violation keyword:* `nil returned instead of a noop struct`
      - *Violation keyword:* `v3 credit handler registered without s.Credits.Enabled check`
  - **Kafka + Watermill async backbone with three name-prefix-routed topics**: Independently deployable workers need durable, replayable, backpressure-aware async delivery and topic isolation so ingest bursts cannot starve billing consumers.
    - *Violation keyword:* `kafka.NewProducer in domain code`
    - *Violation keyword:* `confluent ProduceChannel`
    - *Violation keyword:* `sarama SendMessage`
    - *Violation keyword:* `publishing to a topic by string literal`
    - *Violation keyword:* `EventName() without an EventVersionSubsystem prefix`
    - *Violation keyword:* `context.Background() inside a Watermill handler instead of msg.Context()`
    - *Violation keyword:* `fourth Kafka topic without updating TopicMapping`
    - **Sink worker exactly-once via strict three-phase flush; ingest dedup upstream of an un-deduplicated ClickHouse MergeTree**: Exactly-once ingestion needs ClickHouse written before the Kafka offset commits, and because the MergeTree does not deduplicate, Redis dedupe must be the last phase.
      - *Violation keyword:* `Redis dedupe set before Kafka offset commit`
      - *Violation keyword:* `Kafka offset committed before ClickHouse BatchInsert`
      - *Violation keyword:* `OnFlushSuccess called synchronously`
      - *Violation keyword:* `ReplacingMergeTree dedup in ClickHouse write path`
- **TypeSpec as the single source of truth for both v1 and v3 HTTP APIs and all three SDKs**: Three SDK languages and two API versions cannot be hand-synchronized; a single upstream TypeSpec contract makes drift structurally impossible.
  - *Violation keyword:* `hand-edited api/openapi.yaml`
  - *Violation keyword:* `hand-edited api/v3/openapi.yaml`
  - *Violation keyword:* `endpoint added only in a Go handler package`
  - *Violation keyword:* `hand-edited *.gen.go`
  - *Violation keyword:* `@route in a domain sub-folder tsp instead of root openmeter.tsp`
  - *Violation keyword:* `TypeSpec edit without make gen-api && make generate`
  - **httptransport decode/operate/encode pipeline with dual request validation and GenericErrorEncoder chain**: Dual API versions need dual validation middleware (kin-openapi for v1, oasmiddleware for v3); the generic pipeline keeps RFC 7807 error mapping and OTel uniform across both.
    - *Violation keyword:* `handler implementing ServeHTTP directly`
    - *Violation keyword:* `http status codes written in handler logic instead of models.Generic* sentinels`
    - *Violation keyword:* `chi.NewRouter without a request validator`
    - *Violation keyword:* `v3 handler placed in openmeter/*/httpdriver`
- **Ent ORM + Atlas migrations with context-propagated transactions (entutils.TransactingRepo) and per-customer pg locks (lockr)**: Billing correctness needs compile-time-checked relations across ~35 entities, deterministic reviewable migrations, atomic multi-step mutation, and per-customer serialization against concurrent workers.
  - *Violation keyword:* `edits inside openmeter/ent/db/`
  - *Violation keyword:* `hand-written SQL alongside Ent queries`
  - *Violation keyword:* `*entdb.Tx as a struct field`
  - *Violation keyword:* `a.db.Foo() in an adapter without TransactingRepo`
  - *Violation keyword:* `LockForTX outside an active transaction`
  - *Violation keyword:* `manual edits to tools/migrate/migrations/ or atlas.sum`
  - *Violation keyword:* `context.WithTimeout around LockForTX`
  - **Tagged-union billing models (Charge, ChargeIntent, InvoiceLine) with private discriminators, constructor-only construction, and a generic state machine + LineEngine registry**: Multi-step charge advancement mixing reads, realization, locks, and ledger writes needs exhaustive unambiguous type dispatch and impossible partial construction.
    - *Violation keyword:* `charges.Charge{} struct literal`
    - *Violation keyword:* `charges.ChargeIntent{} struct literal`
    - *Violation keyword:* `billing.InvoiceLine{} struct literal`
    - *Violation keyword:* `direct Invoice.Status field mutation`
    - *Violation keyword:* `RegisterLineEngine called from a domain package or cmd/*`

## Key Decisions

### TypeSpec as the single source of truth for both v1 and v3 HTTP APIs and all three SDKs
**Chosen:** Endpoints are authored only in TypeSpec under api/spec/packages/legacy (v1) and api/spec/packages/aip (v3), with route/tag bindings confined to the root openmeter.tsp. `make gen-api` compiles to api/openapi.yaml + api/v3/openapi.yaml then oapi-codegen emits api/api.gen.go, api/v3/api.gen.go, api/client/go/client.gen.go, plus the JS and Python SDKs; `make generate` then propagates to Ent/Wire/Goverter/Goderive. Handlers implement the generated ServerInterface in openmeter/<domain>/httpdriver (v1) or api/v3/handlers/<resource> (v3).
**Rationale:** Three SDK languages and two API versions cannot be hand-synchronized. The blueprint's TypeSpec single-source pattern (api/spec/packages/aip/src/openmeter.tsp, api/spec/packages/legacy/src/main.tsp) makes drift structurally impossible as long as both regen steps run — a TypeSpec change forces handler-side compile errors in api/v3/handlers and openmeter/server/router. oapi-codegen v2.6.1 (pinned pseudo-version), kin-openapi v0.139.0 (v1 validation) and oasmiddleware v1.1.2 (v3 validation) all validate against the same generated spec.
**Rejected:** Hand-written OpenAPI YAML — rejected; the YAML files carry generated headers and are overwritten by make gen-api., Code-first OpenAPI from Go handlers — rejected; would not produce the JS/Python SDKs from one source., Skipping v3 / single API version — rejected; the AIP-style v3 surface coexists with legacy v1 and both regenerate from the same compiler.
**Forced by:** Multi-language SDK requirement (Go/JS/Python) plus dual API versions plus runtime request validation against the same artifact.
**Enables:** Cross-language SDK contracts that cannot drift; kin-openapi (v1) + oasmiddleware (v3) request validation against the same spec; breaking-change detection at TypeSpec compile time.

### Google Wire DI with all provider sets in app/common and cross-domain hooks registered as construction side-effects
**Chosen:** Each cmd/<binary>/wire.go declares a wire.Build over composite provider sets (per-domain files plus openmeter_<binary>.go) defined in app/common/. Domain packages under openmeter/ expose plain constructors and never import app/common. Cross-domain ServiceHooks and customer RequestValidators are registered inside app/common provider functions as side-effects of construction (e.g. customerService.RegisterHooks and customerService.RegisterRequestValidator), invisible to Wire's type graph.
**Rationale:** Wire produces a compile-time-checked dependency graph per binary so missing providers are build errors, not runtime panics (cmd/billing-worker/wire.go). Concentrating providers in app/common keeps the ~38 domain packages as leaf nodes with no DI-layer dependency, avoiding import cycles. Registering hooks as side-effects in app/common (pkg/models/servicehook.go ServiceHookRegistry, openmeter/customer/requestvalidator.go) lets billing react to customer lifecycle without billing and customer importing each other.
**Rejected:** Manual constructor calls in each cmd/main.go — rejected; ~40 services per binary make hand-wiring error-prone and unverifiable., Reflection-based runtime DI — rejected; loses Wire's compile-time graph verification., Domain packages registering their own hooks — rejected; creates circular imports between billing, customer, subscription, ledger., Provider functions containing business logic — rejected; providers only construct and wire (the wire-002 enforcement rule blocks panic/log.Fatal/os.Exit in app/common).
**Forced by:** ~40 domain services per binary with very different per-binary provider graphs, plus the need for cross-domain hooks without circular imports.
**Enables:** Compile-time proof of binary completeness; import-cycle-free leaf domain packages; cross-domain lifecycle reactions; per-binary composition.

### credits.enabled feature flag enforced at four independent wiring layers via noop implementations
**Chosen:** When config.Credits.Enabled is false: app/common/ledger.go returns ledgernoop.* implementations from each provider; app/common/customer.go NewCustomerLedgerServiceHook returns a NoopCustomerLedgerHook; app/common/billing.go NewBillingRegistry skips the ChargesRegistry entirely (BillingRegistry.Charges stays nil, accessed only via ChargesServiceOrNil()); and api/v3/server credit/ledger handlers skip registration. NewLedgerNamespaceHandler additionally type-asserts against the noop AccountResolver.
**Rationale:** Credits cross-cut ledger writes, customer lifecycle hooks, namespace default-account provisioning, charge creation in billing/charges, and v3 HTTP handlers — there is no single choke point (a customer creation in api/v3 fans out through independent call graphs). The blueprint's noop-for-disabled-features pattern requires each wiring layer to guard independently and return a noop interface rather than nil so callers never branch on nil. AGENTS.md codifies this: 'credits.enabled needs explicit guarding at multiple layers ... wired separately'.
**Rejected:** Single global runtime flag check inside ledger.Ledger — rejected; ledger writes are initiated from three independent call graphs, so one check cannot gate all paths., Top-level HTTP middleware blocking credits endpoints — rejected; does not stop ledger writes triggered by customer hooks or namespace provisioning., Returning nil instead of a noop struct — rejected; callers receive the interface and would panic on nil (the di-001 enforcement rule).
**Forced by:** The cross-cutting nature of credit accounting and the customer/billing/ledger hook fan-out across unrelated call graphs.
**Enables:** Credits-disabled tenants produce zero ledger_accounts/ledger_customer_accounts rows; per-deployment enabling without rebuild; compile-time interface satisfaction for noop implementations.

### Ent ORM + Atlas migrations with context-propagated transactions (entutils.TransactingRepo) and per-customer pg locks (lockr)
**Chosen:** openmeter/ent/schema holds ~35 Go-defined entity schemas (each with IDMixin + NamespaceMixin + TimeMixin); `make generate` regenerates openmeter/ent/db/; `atlas migrate --env local diff` produces timestamped .up.sql/.down.sql plus an atlas.sum hash chain. Every domain adapter implements the Tx/WithTx/Self triad and wraps every method body in entutils.TransactingRepo / TransactingRepoWithNoValue (pkg/framework/entutils/transaction.go), which rebinds to any ctx-bound transaction or runs on Self(). Per-customer serialization uses a SELECT FOR UPDATE on the BillingCustomerLock row (UNIQUE (namespace, customer_id)) plus generic pg_advisory_xact_lock via pkg/framework/lockr.
**Rationale:** Billing correctness needs compile-time-checked relations across ~35 entities, deterministic reviewable migrations, atomic multi-step charge/invoice mutation, and per-customer serialization against concurrent workers. TransactingRepo reads the *TxDriver from ctx and rebinds (transaction.go), supporting savepoint nesting for multi-step flows like AdvanceCharges. The lock invariant is grounded: openmeter/ent/schema/billing.go declares UNIQUE (namespace, customer_id) on BillingCustomerLock, so the lock serializes exactly one customer.
**Rejected:** Raw golang-migrate only (no typed entities) — rejected; loses compile-checked relations across ~35 entities., GORM — rejected; weaker typing and no native Atlas-style schema diff., Explicit *entdb.Tx threaded through every signature — rejected; ctx-propagation via TransactingRepo avoids signature churn and supports savepoint nesting., Hand-written SQL alongside Ent — rejected; breaks Atlas's single-schema-source diffing (the ent-001 enforcement rule).
**Forced by:** Billing correctness plus multi-tenant schema invariants requiring compile-time-checked relations and ctx-propagated transaction reuse with savepoints.
**Enables:** Deterministic reviewable SQL migrations with atlas.sum integrity; typed relations across all entities; atomic charge advancement and invoice mutation; per-customer advisory locking.

### Tagged-union billing models (Charge, ChargeIntent, InvoiceLine) with private discriminators, constructor-only construction, and a generic state machine + LineEngine registry
**Chosen:** openmeter/billing/charges/charge.go declares Charge/ChargeIntent with a private meta.ChargeType discriminator set only by NewCharge[T]/NewChargeIntent[T] and accessed via AsFlatFeeCharge/AsUsageBasedCharge/AsCreditPurchaseCharge; openmeter/billing/invoiceline.go declares InvoiceLine with a private discriminator set only by NewStandardInvoiceLine/NewGatheringInvoiceLine. Each charge type plugs into a generic Machine[CHARGE,BASE,STATUS] (value-copy WithStatus/WithBase) and registers a LineEngine with billing.Service.RegisterLineEngine in app/common/charges.go. StandardInvoice lifecycle runs through a stateless.StateMachine pooled in sync.Pool (openmeter/billing/service/stdinvoicestate.go).
**Rationale:** Multi-step charge advancement mixes reads, realization runs, advisory locks, and ledger-bound writes, so it needs exhaustive unambiguous charge-type dispatch and impossible partial construction — a struct literal leaves the discriminator zero-valued and accessors error. The blueprint's tagged-union and state-machine patterns confirm this: qmuntal/stateless v1.8.0 drives the lifecycle, and the LineEngine registry (registered in app/common, never from domain packages) lets new charge types plug in without editing billing core.
**Rejected:** Charge as a Go interface — rejected; loses exhaustive compile-time dispatch., Public discriminator field — rejected; allows partial/inconsistent construction via struct literals., Hardcoding charge-type branches in billing.Service — rejected; the LineEngine registry plus App Factory keep the core decoupled., Mutating Invoice.Status directly — rejected; the stateless state machine enforces valid transitions and fires post-transition actions atomically.
**Forced by:** Multi-step charge advancement requiring exhaustive, unambiguous charge-type dispatch and atomic state transitions.
**Enables:** Deterministic atomic charge advancement; exhaustive charge-type and invoice-line-type dispatch; runtime-pluggable charge engines via RegisterLineEngine; per-charge advisory locking via charges.NewLockKeyForCharge.

### Sink worker exactly-once via strict three-phase flush; ingest dedup upstream of an un-deduplicated ClickHouse MergeTree
**Chosen:** openmeter/sink/sink.go flush() executes strictly: (1) Storage.BatchInsert into the shared ClickHouse MergeTree events table, (2) Consumer.StoreMessage per Kafka offset (largest stored last), (3) Redis SET NX dedupe (only when a Deduplicator is configured), then fires FlushEventHandler.OnFlushSuccess in a goroutine bounded by FlushSuccessTimeout. The ClickHouse RawEvent table is ENGINE=MergeTree with no engine-level dedup — deduplication is entirely upstream in Redis (openmeter/dedupe/redisdedupe) keyed namespace-source-id with TTL.
**Rationale:** Exactly-once usage ingestion requires ClickHouse to be written before the Kafka offset commits — on consumer restart an uncommitted offset re-delivers messages not yet in ClickHouse (openmeter/sink/sink.go ordering). Because the MergeTree does not deduplicate (RawEvent guarantees: 'not deduplicated by the engine — dedup is upstream in Redis'), Redis dedupe being phase 3 (strictly after offset commit) is load-bearing; reversing it would mark events processed while ClickHouse re-reads from the uncommitted offset, dropping them.
**Rejected:** ReplacingMergeTree / engine-level dedup in ClickHouse — rejected; dedup is pushed to the ingest edge (Redis SET NX) so the hot analytics path stays append-only., Committing Kafka offset before the ClickHouse insert — rejected; would lose events on crash between commit and insert., Setting Redis dedupe before the offset commit — rejected; the dedupe-001 enforcement rule and sink.go ordering forbid it (breaks exactly-once on restart)., Calling OnFlushSuccess synchronously — rejected; it always runs in a goroutine so post-flush balance-recalc notification never blocks the consumer loop.
**Forced by:** High-throughput usage ingestion needing exactly-once semantics on top of an append-only analytics store that does not deduplicate.
**Enables:** Exactly-once event delivery to ClickHouse across consumer restarts; append-only hot path; decoupled post-flush balance-recalculation via Kafka.

## Trade-offs Accepted

- **Accepted:** Ent-generated query friction: a large generated openmeter/ent/db/ tree, slower compile times, and the boilerplate Tx/WithTx/Self triad plus a TransactingRepo wrapper on every adapter method body.
  - *Benefit:* Compile-time-checked relations across ~35 entities, automatic Atlas schema diffing into reviewable SQL, and ctx-propagated transactions with savepoint nesting for atomic multi-step charge/invoice flows.
  - *Caused by:* Ent ORM + Atlas migrations with context-propagated transactions (entutils.TransactingRepo) and per-customer pg locks (lockr)
  - *Violation signal:* db.ExecContext / db.QueryContext raw SQL added alongside Ent queries in an adapter
  - *Violation signal:* direct edits inside openmeter/ent/db/
  - *Violation signal:* a new table without a corresponding openmeter/ent/schema/*.go file
  - *Violation signal:* an adapter storing *entdb.Tx as a struct field instead of using TransactingRepo
  - *Violation signal:* an adapter method body calling a.db.Foo() without entutils.TransactingRepo / TransactingRepoWithNoValue
- **Accepted:** Multi-binary orchestration cost: seven Docker image variants, Helm values complexity, and a separate Wire graph per binary that must each stay complete.
  - *Benefit:* Independent horizontal scaling of sink-worker / balance-worker / billing-worker, fault isolation per binary, and isolated deploy cadence.
  - *Caused by:* Multi-binary modular monolith: one shared Go domain tree, seven independently deployable binaries, Kafka as the sole inter-binary channel
  - *Violation signal:* business logic added inside cmd/*/main.go beyond startup orchestration
  - *Violation signal:* a new cmd/* worker binary without a matching app/common/openmeter_<binary>.go Wire set
  - *Violation signal:* cross-binary dependencies introduced via shared in-memory state or HTTP calls instead of a Kafka topic
  - *Violation signal:* a goroutine spawned outside the oklog/run.Group
  - *Violation signal:* kafka.NewProducer / confluent ProduceChannel / sarama SendMessage in domain code instead of eventbus.Publisher
- **Accepted:** Two-step regeneration cadence: TypeSpec changes require both `make gen-api` AND `make generate`, and five generators (oapi-codegen, Ent, Wire, Goverter, Goderive) write different artifacts that must all stay in sync.
  - *Benefit:* Cross-language SDK contracts cannot drift — Go server stubs, Go SDK, JS SDK, Python SDK all originate from one TypeSpec source.
  - *Caused by:* TypeSpec as the single source of truth for both v1 and v3 HTTP APIs and all three SDKs
  - *Violation signal:* hand-edits inside *.gen.go, wire_gen.go, api/api.gen.go, or api/v3/api.gen.go
  - *Violation signal:* PRs touching api/spec/ without regenerated api/openapi.yaml
  - *Violation signal:* a new endpoint added only in a Go handler package without a TypeSpec source change
  - *Violation signal:* @route declared in a domain sub-folder tsp instead of the root openmeter.tsp
  - *Violation signal:* hand-edited api/openapi.yaml or api/v3/openapi.yaml
- **Accepted:** Cross-domain wiring and event routing are invisible to the compiler: hook/validator registration and credits guards are side-effects scattered across app/common, and Kafka topic routing depends on event-name string prefixes that default to SystemEventsTopic.
  - *Benefit:* Domain packages stay import-cycle-free leaves; optional features are gated without nil-checks in business logic; the three-topic topology is hidden from producers.
  - *Caused by:* Google Wire DI with all provider sets in app/common and cross-domain hooks registered as construction side-effects
  - *Violation signal:* a binary's wire.Build omitting a hook provider so the hook silently never registers
  - *Violation signal:* a new ledger-touching Wire provider added without a creditsConfig.Enabled branch
  - *Violation signal:* direct access to BillingRegistry.Charges without ChargesServiceOrNil()
  - *Violation signal:* a new event family whose EventName() lacks a recognized EventVersionSubsystem prefix (silently routes to SystemEventsTopic)
  - *Violation signal:* a fourth Kafka topic added without updating GeneratePublishTopic in eventbus.go
- **Accepted:** Sequential timestamped Atlas migration filenames plus a linear atlas.sum hash chain that, by construction, produces merge conflicts between any two branches that both append migrations.
  - *Benefit:* Deterministic, reviewable, linearly-ordered SQL migration history with cryptographic chain integrity verified by CI (`make migrate-check`).
  - *Caused by:* Ent ORM + Atlas migrations with context-propagated transactions (entutils.TransactingRepo) and per-customer pg locks (lockr)
  - *Violation signal:* two branches producing same-timestamp migration files
  - *Violation signal:* atlas.sum merge conflicts on a long-running branch
  - *Violation signal:* manual edits to an already-landed migration file in tools/migrate/migrations/
  - *Violation signal:* committing .up.sql/.down.sql without an updated atlas.sum

## Out of Scope

- Frontend UI — no React/Vue application in the repo; React appears only as an optional context export inside the generated JavaScript SDK (api/client/javascript). Out of scope by the API-as-product / SDK-generation decision.
- Tenant-level identity and auth provider — portal tokens scope end-customers via JWTs (openmeter/portal, golang-jwt v5), but tenant identity is delegated to the deployment. Out of scope by the self-hosted single-namespace deployment model.
- Managed hosting control plane — config.cloud.yaml and api/openapi.cloud.yaml expose cloud hooks, but cloud orchestration logic lives separately.
- Real-time client-facing streaming queries — ClickHouse is reached only via streaming.Connector inside server-side processes; there is no client-facing streaming surface.
- Multi-region active/active replication — a single PostgreSQL primary is assumed; ClickHouse cluster topology is deployment-defined. Out of scope by the single-primary Ent persistence decision.
- Synchronous cross-binary RPC / service mesh — all inter-binary communication goes through the three Kafka topics; there is no gRPC surface between binaries.
- LLM inference — openmeter/llmcost only persists/syncs model price tables; there is no OpenAI/Anthropic inference SDK.
- Infrastructure provisioning as code — Helm charts only (deploy/charts/); no Terraform/CloudFormation/Pulumi.