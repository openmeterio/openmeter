## Communication Patterns

### Layered Domain Service / Adapter / HTTP (three-layer per domain)
- **Scope:** `openmeter/billing`, `openmeter/customer`, `openmeter/entitlement`, `openmeter/subscription`, `openmeter/notification`, `openmeter/meter`, `openmeter/ledger`, `openmeter/productcatalog`, `openmeter/app`, `openmeter/llmcost`, `openmeter/subject`, `openmeter/credit`
- **When:** All business-logic domains under openmeter/<domain>/. Applied whenever persistence must be separated from orchestration and HTTP translation.
- **How:** Each domain exposes a Go interface (e.g. billing.Service, customer.Service) defined at the package root in service.go. A concrete service struct in <domain>/service/ holds business logic and calls an Adapter interface for all DB access. The Adapter interface is defined alongside Service and implemented by Ent-backed structs in <domain>/adapter/ sub-packages. HTTP handlers live in <domain>/httpdriver/ or httphandler/ sub-packages. Service interfaces compose fine-grained sub-interfaces (e.g. ProfileService, InvoiceService) so callers depend on the smallest surface.
- **Applicable when:** Adapters implementing TxCreator (Tx via HijackTx + NewTxDriver) and TxUser[T] (WithTx + Self) — every method body must use entutils.TransactingRepo so the ctx-bound Ent transaction is honored; verified at openmeter/billing/adapter/adapter.go:51-72 where Tx(), WithTx(), and Self() are all implemented.
- **Do NOT apply when:**
  - Adapter method bodies that use the raw *entdb.Client directly without TransactingRepo wrapping — the raw client ignores any Ent transaction carried in ctx, confirmed at pkg/framework/entutils/transaction.go:199-221 where TransactingRepo reads the TxDriver from context
  - Placing business logic inside cmd/*/main.go — cmd/* must only wire and start; logic belongs under openmeter/

### Google Wire Dependency Injection
- **Scope:** `app/common`, `cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`
- **When:** Assembling runtime components for each binary. Each cmd/<binary>/wire.go declares a wire.Build; provider sets live in app/common/.
- **How:** Google Wire generates cmd/<binary>/wire_gen.go at build time. Reusable provider sets (e.g. common.BillingWorker, common.LedgerStack) are declared as wire.NewSet() in app/common/ and compose individual factory functions. Wire resolves the dependency graph at compile time. Hook and validator registration is done as side-effects inside provider functions in app/common to avoid circular imports.
- **Applicable when:** Binary entrypoints that must compose ~40 domain services compile-time safely — Wire provider sets in app/common are verified at wire.Build compile time, as seen in cmd/billing-worker/wire.go:35-58; missing providers cause compile errors not runtime panics.
- **Do NOT apply when:**
  - Domain packages importing app/common — the import direction is one-way outward (app/common imports domain packages); reversing creates import cycles
  - Provider functions containing business logic (validation, computation, state mutation beyond hook/validator registration)

### Registry Structs for Multi-Service Domains
- **Scope:** `app/common`, `api/v3/server`
- **When:** When a domain exposes multiple related services that callers must access together, reducing Wire graph complexity.
- **How:** A <Domain>Registry struct groups logically cohesive services (e.g. BillingRegistry, ChargesRegistry). Callers depend on the registry rather than individual services. Nil-safe accessor methods (e.g. BillingRegistry.ChargesServiceOrNil()) encapsulate optional sub-registries when features are disabled.
- **Applicable when:** Multi-service domains where one service may be conditionally nil at runtime — ChargesRegistry is nil when credits.enabled=false, so ChargesServiceOrNil() at app/common/billing.go:48 provides the nil-safe accessor; direct BillingRegistry.Charges field access would panic.
- **Do NOT apply when:**
  - Accessing BillingRegistry.Charges directly without ChargesServiceOrNil() — confirmed at app/common/billing.go:48 that Charges is nil when credits disabled

### Noop Implementations for Optional Features
- **Scope:** `app/common`, `openmeter/ledger`, `openmeter/notification/webhook`
- **When:** When a feature is disabled at runtime (credits.enabled=false, Svix not configured) to avoid nil-pointer checks scattered through business logic.
- **How:** app/common provider functions check config flags and return noop structs instead of real implementations. All noop types implement the relevant interface. Callers receive a real interface and never branch on nil. Credits feature flag must be honored at four independent layers: (1) ledger services in app/common/ledger.go, (2) customer ledger hooks in app/common/customer.go, (3) ChargesRegistry skipped in app/common/billing.go, (4) credit handlers in api/v3/server.
- **Applicable when:** Any provider that wires ledger-backed features when credits.enabled=false — app/common/ledger.go and app/common/customer.go independently guard with creditsConfig.Enabled; a single centralized guard is insufficient because credits cross-cuts multiple independent call graphs.
- **Do NOT apply when:**
  - Features that are always required — use real implementations directly without an enabled flag
  - Returning nil instead of a noop struct — callers receive the interface and will panic if nil is assigned

### Watermill Kafka-backed Pub/Sub (three fixed topics, prefix routing)
- **Scope:** `openmeter/watermill`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification/consumer`, `openmeter/sink`
- **When:** Async domain-event delivery between separate binaries. Used for subscription lifecycle events, billing invoice advance events, ingest flush notifications, and balance-worker recalculation events.
- **How:** openmeter/watermill/eventbus/eventbus.go wraps Watermill's cqrs.EventBus with TopicMapping (IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic). GeneratePublishTopic routes by EventName() prefix: events starting with ingestevents.EventVersionSubsystem go to IngestEventsTopic; balanceworkerevents.EventVersionSubsystem go to BalanceWorkerEventsTopic; default falls through to SystemEventsTopic. Workers subscribe via openmeter/watermill/router.NewDefaultRouter with fixed middleware stack (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout, HandlerMetrics). Consumers dispatch to typed handlers registered in grouphandler.NoPublishingHandler keyed on CloudEvents ce_type. Unknown types are silently dropped (returned nil in grouphandler.go:49).
- **Applicable when:** Domain event producers that must route to one of three isolated Kafka topics — EventName() must begin with a recognized EventVersionSubsystem prefix; otherwise GeneratePublishTopic at eventbus.go:135-143 silently routes to SystemEventsTopic.
- **Do NOT apply when:**
  - Publishing directly to a Kafka topic string — always use eventbus.Publisher which encapsulates routing
  - Returning errors for unknown event types in consumer handlers — grouphandler.go:49 silently returns nil to support rolling deploys; returning errors causes Watermill retries and DLQ poisoning
  - Substituting context.Background() inside a handler instead of msg.Context()

### entutils.TransactingRepo (Context-propagated Ent Transactions)
- **Scope:** `openmeter/billing/adapter`, `openmeter/billing/charges/adapter`, `openmeter/customer/adapter`, `openmeter/entitlement`, `openmeter/subscription`, `openmeter/notification/adapter`, `openmeter/ledger`
- **When:** All DB adapter methods that must run inside a caller-supplied transaction or start their own.
- **How:** pkg/framework/entutils/transaction.go defines TransactingRepo[R,T] and TransactingRepoWithNoValue[T]. They read the *TxDriver from context via GetDriverFromContext(). If found, the adapter's WithTx(ctx, tx) creates a txClient from raw Ent config. If none is found, the adapter runs on Self() and operates without a transaction. Savepoints are created for nested calls: TxDriver.SavePoint() increments a counter; first call skips the savepoint to allow the outer transaction to close normally.
- **Applicable when:** Adapter methods called both standalone and inside multi-step transactions — TransactingRepo at pkg/framework/entutils/transaction.go:199 reads *TxDriver from ctx and rebinds to the caller's transaction if present, or uses Self() to run independently; this prevents partial writes when called within AdvanceCharges or similar multi-step flows.
- **Do NOT apply when:**
  - Adapter methods that directly call a.db.Foo() without the TransactingRepo wrapper — they ignore the ctx-bound Ent transaction and produce partial writes under concurrency

### pg_advisory_xact_lock via lockr.Locker
- **Scope:** `openmeter/billing`, `openmeter/billing/charges`, `openmeter/entitlement`
- **When:** Distributed mutual exclusion for per-customer billing operations to prevent concurrent invoice generation races.
- **How:** pkg/framework/lockr/locker.go calls pg_advisory_xact_lock($1) with a CRC64-based hash of the lock key (confirmed at locker.go:66). Requires an active Postgres transaction in context: getTxClient() calls entutils.GetDriverFromContext() and additionally queries 'SELECT transaction_timestamp() != statement_timestamp()' to verify (locker.go:100-137). Lock is released automatically on tx commit/rollback.
- **Applicable when:** Per-customer billing mutations where concurrent goroutines can race on invoice creation or charge advancement — LockForTX at lockr/locker.go:45 requires an active Postgres transaction in ctx; calling outside a transaction causes the 'lockr only works in a postgres transaction' error at locker.go:135.
- **Do NOT apply when:**
  - Calling LockForTX outside an active Postgres transaction — locker.go:135 returns an error if statement_timestamp() == transaction_timestamp()
  - Using context.WithTimeout for lock acquisition — pgx cancels the connection on ctx cancel; code comment at locker.go:91-93 explicitly warns against this

### httptransport.Handler Decode/Operate/Encode Pipeline
- **Scope:** `openmeter/billing/httpdriver`, `openmeter/customer/httpdriver`, `openmeter/meter/httphandler`, `api/v3/handlers`
- **When:** HTTP endpoint handlers in domain packages that separate request decoding, business logic, and response encoding.
- **How:** pkg/framework/transport/httptransport/handler.go defines generic Handler[Request, Response] interface. NewHandler() accepts a RequestDecoder, an operation.Operation, a ResponseEncoder, and optional HandlerOptions. defaultHandlerOptions appends GenericErrorEncoder as the last error encoder. Decode failure or operation failure triggers encodeError which iterates the error encoder chain; first matching encoder short-circuits. SelfEncodingError interface allows errors to encode themselves. Chain() wraps the operation with middleware.
- **Applicable when:** HTTP endpoints that must map domain errors to RFC 7807 problem+json responses — the GenericErrorEncoder at handler.go:17 is always appended as defaultHandlerOption; custom encoders passed before it take precedence.
- **Do NOT apply when:**
  - Implementing ServeHTTP directly in a handler struct — this bypasses defaultHandlerOptions (GenericErrorEncoder), OTel instrumentation, and the chain pattern

### ServiceHook Registry (cross-domain lifecycle callbacks)
- **Scope:** `openmeter/customer`, `openmeter/subscription`, `openmeter/app`, `app/common`
- **When:** Cross-domain lifecycle callbacks without circular imports. Used by customer.Service, subscription.Service, and app marketplace.
- **How:** pkg/models/servicehook.go defines generic ServiceHook[T] interface (PreUpdate, PreDelete, PostCreate, PostUpdate, PostDelete) and thread-safe ServiceHookRegistry[T] that fans out to all registered hooks using RWMutex. Loop prevention: a per-registry context key (pointer-identity string via fmt.Sprintf('%p', r)) prevents re-entrant invocations — confirmed at servicehook.go:46-65 where ctx is checked for the loop key. Domain services embed *ServiceHookRegistry and expose RegisterHooks() externally. Registration happens as side-effects in app/common provider functions.
- **Applicable when:** Cross-domain lifecycle reactions where direct package imports would create circular dependencies — billing hooking into customer lifecycle must go through ServiceHookRegistry registered in app/common.
- **Do NOT apply when:**
  - Domain packages calling RegisterHooks on another domain's service inside their own constructors — this creates circular imports; always register in app/common provider functions

### Customer RequestValidator Registry (pre-mutation validation guards)
- **Scope:** `openmeter/customer`, `app/common`
- **When:** Pre-mutation validation for customer operations where billing, subscription, or entitlement constraints must be checked before the customer is modified or deleted.
- **How:** openmeter/customer/requestvalidator.go defines RequestValidator interface (ValidateDeleteCustomer, ValidateCreateCustomer, ValidateUpdateCustomer) and thread-safe requestValidatorRegistry that fans out to all registered validators using errors.Join (no short-circuit on first failure). Validators are registered via customerService.RegisterRequestValidator() in app/common provider functions.
- **Applicable when:** Pre-mutation guards from billing or entitlement domains that must block customer operations before any DB write — the registry fans out to all registered validators via errors.Join before any adapter write.
- **Do NOT apply when:**
  - Post-mutation reactions — these belong in ServiceHooks (PostCreate, PostUpdate, PostDelete), not in RequestValidatorRegistry which is exclusively for pre-mutation blocking validation

### Invoice State Machine (stateless library, sync.Pool backed)
- **Scope:** `openmeter/billing`, `openmeter/billing/service`
- **When:** Driving the StandardInvoice lifecycle transitions.
- **How:** openmeter/billing/service/stdinvoicestate.go builds a *stateless.StateMachine via stateless.NewStateMachineWithExternalStorage. External storage reads/writes Invoice.Status directly on the InvoiceStateMachine struct. StateMachines are pooled in invoiceStateMachineCache (sync.Pool) to reduce GC pressure. Transitions are configured with Permit/OnActive on each state. FireAndActivate fires a trigger and persists; AdvanceUntilStateStable walks TriggerNext transitions.
- **Applicable when:** Invoice lifecycle operations that must enforce valid transition sequences — the stateless.StateMachine prevents invalid transitions at runtime; direct Invoice.Status field mutation bypasses these guards and leaves invoices in inconsistent state.
- **Do NOT apply when:**
  - Directly mutating Invoice.Status fields without going through FireAndActivate — the state machine enforces valid transitions and fires post-transition actions (DB save, event publish)

### Generic Charge State Machine (Machine[CHARGE,BASE,STATUS])
- **Scope:** `openmeter/billing/charges`
- **When:** Driving charge lifecycle (flatfee, usagebased, creditpurchase) with shared mechanics.
- **How:** openmeter/billing/charges/statemachine/machine.go defines generic Machine[CHARGE ChargeLike[CHARGE,BASE,STATUS], BASE any, STATUS Status]. ExternalStorage reads GetStatus() from CHARGE and writes via WithStatus() — both must return new value copies (value semantics, not pointer mutation). FireAndActivate fires a trigger and persists BASE via Persistence.UpdateBase; after persistence, Refetch retrieves the updated CHARGE from the database.
- **Applicable when:** Charge types implementing ChargeLike[CHARGE,BASE,STATUS] — WithStatus and WithBase must return new value copies; pointer-mutating implementations break the external storage pattern at statemachine/machine.go:58 where Machine.Charge is updated by assignment.
- **Do NOT apply when:**
  - Implementing WithStatus or WithBase as pointer receivers that mutate in place — they must return new value copies because Machine.Charge is updated by assignment

### Sink Worker Three-Phase Flush (Kafka to ClickHouse batch)
- **Scope:** `openmeter/sink`, `cmd/sink-worker`
- **When:** High-throughput ingestion path: raw CloudEvents buffered from Kafka, deduplicated, and batch-inserted into ClickHouse.
- **How:** openmeter/sink/sink.go: flush() acquires a mutex, pauses Kafka partitions, dequeues buffer, runs in-batch dedup, then (1) calls persistToStorage (ClickHouse BatchInsert), (2) calls Consumer.StoreMessage for each Kafka offset, (3) calls dedupeSet (Redis SETNX with exponential retry). After all three phases, FlushEventHandler.OnFlushSuccess is called in a goroutine with FlushSuccessTimeout-bounded context to decouple it from the hot path (sink.go:371-378).
- **Applicable when:** Exactly-once usage event ingestion where ClickHouse must be written before Kafka offset commit — reversing the three-phase flush order breaks the exactly-once guarantee on consumer restart.
- **Do NOT apply when:**
  - Calling FlushEventHandler.OnFlushSuccess synchronously inside flush() — this blocks the main sink loop and causes Kafka partition backpressure (confirmed at sink.go:371-378 where it is always in a goroutine)

### Namespace Manager Fan-out (multi-tenancy provisioning)
- **Scope:** `openmeter/namespace`, `cmd/server`
- **When:** Provisioning and deprovisioning tenants across all subsystems (ClickHouse, Kafka ingest, Ledger).
- **How:** openmeter/namespace/namespace.go: Manager holds a slice of Handler implementations. createNamespace() fans out to all registered handlers using errors.Join (no short-circuit on partial failure), confirmed at namespace.go:105-119. RegisterHandler() appends dynamically with RWMutex. CreateDefaultNamespace() calls createNamespace with the default name. The default namespace is protected from deletion by DeleteNamespace() at namespace.go:64-70.
- **Applicable when:** Subsystems that must provision resources per namespace — Handler implementations must be registered via RegisterHandler before CreateDefaultNamespace is called at startup to receive the default namespace creation event.
- **Do NOT apply when:**
  - Registering a Handler after CreateDefaultNamespace has been called — it will miss default namespace provisioning

### RFC 7807 Problem Details HTTP Error Response
- **When:** All error responses from the REST API.
- **How:** pkg/models/problem.go defines StatusProblem struct serialized as application/problem+json (ProblemContentType). NewStatusProblem() reads request-id from Chi middleware context via middleware.GetReqID(), maps 'context canceled' substring to 408, suppresses detail on 500. pkg/framework/commonhttp/errors.go provides HandleErrorIfTypeMatches[T] which uses errors.As to check type and produces the correct HTTP status. HandleIssueIfHTTPStatusKnown extracts ValidationIssue nodes with httpStatusCodeErrorAttribute attached by WithHTTPStatusCodeAttribute.
- **Applicable when:** Any HTTP error response in the API — models.NewStatusProblem plus the GenericErrorEncoder chain in httptransport ensures all domain errors render as application/problem+json with correct status codes.

### ValidationIssue Immutable Builder
- **Scope:** `openmeter/billing`, `openmeter/billing/charges`, `openmeter/customer`, `openmeter/notification`, `openmeter/subscription`
- **When:** Domain-level validation errors that must carry field paths, severity, component names, and arbitrary attributes through service layer boundaries.
- **How:** pkg/models/validationissue.go defines ValidationIssue as an immutable value type (all fields private). Clone() returns a new copy with wraps set to the original. With() accepts ValidationIssueOption functions that modify the clone. WithHTTPStatusCodeAttribute() in commonhttp/errors.go attaches an integer status code as an attribute key 'openmeter.http.status_code'. HandleIssueIfHTTPStatusKnown() reads this attribute at the HTTP boundary.
- **Applicable when:** Field-level validation errors that must survive wrapping through multiple service layers and emerge at the HTTP boundary with correct status codes — WithHTTPStatusCodeAttribute at commonhttp/errors.go:82 attaches the status as an attribute on the ValidationIssue; HandleIssueIfHTTPStatusKnown reads it at the HTTP boundary.
- **Do NOT apply when:**
  - Mutating a ValidationIssue in place — all With* methods return new copies; direct struct field assignment is impossible (all fields are unexported)

### App Factory / Registry (External Billing App Protocol)
- **Scope:** `openmeter/app`, `openmeter/billing`
- **When:** Plugging Stripe, Sandbox, and CustomInvoicing billing apps into the billing state machine without hardcoding them.
- **How:** openmeter/app/service.go defines AppService interface with RegisterMarketplaceListing. Each concrete app type (Stripe, Sandbox, CustomInvoicing) implements app.App interface and optionally billing.InvoicingApp. Self-registration happens in each app's New() or factory constructor. The App interface requires GetCustomerData, UpsertCustomerData, DeleteCustomerData for customer-data lifecycle.
- **Applicable when:** Billing backends that implement billing.InvoicingApp (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice) — new backends must implement InvoicingApp and register via the factory, not add conditional branches in the core service.
- **Do NOT apply when:**
  - Adding provider-specific logic directly inside billing.Service — new billing backends must implement InvoicingApp and register via the factory

### LineEngine Plugin Registry
- **Scope:** `openmeter/billing`, `app/common`
- **When:** Dispatching billing line calculation to the correct engine based on LineEngineType discriminator.
- **How:** billing.Service exposes RegisterLineEngine / DeregisterLineEngine / GetRegisteredLineEngines via LineEngineService interface (billing/service.go:63-67). Each charge type implements its own LineEngine and registers at startup in app/common/charges.go. The service implementation stores engines in a map under RWMutex.
- **Applicable when:** Charge types that need to register their own line engine — engines must be registered before the first invoice advance, as codified in app/common via Wire side-effects.
- **Do NOT apply when:**
  - Registering line engines from domain packages instead of app/common — this would create circular imports

## Integrations

| Service | Purpose | Integration point |
|---------|---------|-------------------|
| PostgreSQL | Primary relational store for all domain entities: billing profiles, invoices, customers, subscriptions, entitlements, notification channels, ledger accounts, meters, subjects, secrets, charges. | `openmeter/ent/schema/ (source of truth); generated code in openmeter/ent/db/; Atlas migrations in tools/migrate/migrations/. Accessed via *entdb.Client injected through Wire. Transactions managed by pkg/framework/entutils.TransactingRepo.` |
| ClickHouse | Append-only analytics store for raw usage events; queried for meter aggregations (count, sum, max, unique_count) via SQL builders. | `openmeter/streaming/clickhouse/ for batch inserts and meter queries. openmeter/sink/storage.go (Storage interface BatchInsert). openmeter/sink/sink.go:308-314 calls persistToStorage which calls BatchInsert.` |
| Kafka (confluent-kafka-go + Watermill-Kafka) | Durable event bus for domain events (subscription lifecycle, invoice advance, ingest flush notifications, balance recalculation) and raw usage event ingestion. | `openmeter/watermill/driver/kafka/ — Publisher and Subscriber wrappers. openmeter/watermill/eventbus/eventbus.go — topic routing by prefix. confluent-kafka-go used directly in openmeter/sink/sink.go for high-throughput ingest consumer.` |
| Redis | Optional deduplication store for ingest events (preventing double-counting on retry). | `openmeter/dedupe/redisdedupe/ — Redis-backed Deduplicator. In-memory LRU fallback in openmeter/dedupe/memorydedupe/. Used in openmeter/sink/sink.go:333-358 (dedupeSet phase).` |
| Svix | Outbound webhook delivery for notification events (entitlement balance thresholds, invoice events). | `openmeter/notification/webhook/svix/svix.go — Svix API client wrapper. NullChannel sentinel at svix.go:20-26 prevents unfiltered delivery. Handler interface in openmeter/notification/webhook/handler.go with noop fallback when Svix is unconfigured.` |
| Stripe | Invoice syncing (upsert draft, finalize, collect payment) and customer sync for billing-enabled namespaces. | `openmeter/app/stripe/app.go implements billing.InvoicingApp (UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice). Stripe REST client in openmeter/app/stripe/client/.` |
| Sandbox Invoicing App | No-op invoicing app used in development/testing to drive invoice state machine without external dependencies. | `openmeter/app/sandbox/app.go implements billing.InvoicingApp + billing.InvoicingAppPostAdvanceHook (confirmed at sandbox/app.go:26-27).` |
| CustomInvoicing App | Webhook-driven invoicing app allowing external systems to receive invoice payloads and async-confirm sync completion. | `openmeter/app/custominvoicing/ — App implements InvoicingApp + InvoicingAppAsyncSyncer.` |
| GOBL | Currency and numeric type library for currency-safe arithmetic and ISO 4217 currency code validation throughout billing and subscription. | `Imported as github.com/invopop/gobl/currency and github.com/invopop/gobl/num in productcatalog, subscription, billing, cost, and currencies packages.` |
| OpenTelemetry | Distributed tracing and metrics across all services. | `trace.Tracer injected via Wire into service constructors. OTel metric.Meter used in grouphandler (grouphandler.go:120-139) and sink worker. app/common/telemetry.go bootstraps OTLP exporters.` |
| TypeSpec compiler | Single source of truth for HTTP API definitions, compiling to OpenAPI YAML and downstream Go server stubs and SDKs. | `api/spec/packages/ — TypeSpec source. make gen-api runs tsp compile to produce api/openapi.yaml, api/v3/openapi.yaml, then oapi-codegen produces api/api.gen.go, api/v3/api.gen.go, api/client/go/client.gen.go.` |
| App Marketplace Registry (extension protocol) | Runtime-pluggable billing backend mechanism allowing Stripe, Sandbox, and CustomInvoicing to register themselves without hardcoded references in billing domain. | `openmeter/app/service.go — Service.RegisterMarketplaceListing; each app's New() or factory self-registers via this call.` |
| LineEngine Registry (extension protocol) | Runtime-pluggable billing line calculation dispatch by LineEngineType. | `openmeter/billing/service.go — LineEngineService.RegisterLineEngine / DeregisterLineEngine; app/common registers all charge type engines at Wire startup.` |
| ServiceHook Registries (extension protocol) | Cross-domain lifecycle callbacks without circular imports — billing hooks customer lifecycle, ledger hooks customer creation. | `pkg/models/servicehook.go — ServiceHookRegistry[T]; hooks registered in app/common/customer.go as side-effects of Wire provider functions.` |
| Customer RequestValidator Registry (extension protocol) | Pre-mutation validation guards for customer operations from billing and entitlement domains. | `openmeter/customer/requestvalidator.go — requestValidatorRegistry; billing validator registered via customerService.RegisterRequestValidator() in app/common/billing.go:207.` |

## Pattern Selection Guide

| Scenario | Pattern | Rationale |
|----------|---------|-----------|
| Adding a new domain capability (e.g. a new billing sub-feature) | Layered Domain Service/Adapter/HTTP in openmeter/<domain>/ | Define a new sub-interface in <domain>/service.go, implement in <domain>/service/, add adapter methods to <domain>/adapter.go, implement in <domain>/adapter/. Wire together in app/common/<domain>.go. Keeps business logic, persistence, and HTTP separate and independently testable. |
| Triggering side-effects on domain lifecycle (e.g. sync billing on subscription update) | ServiceHook Registry (models.ServiceHooks[T]) | Avoids circular imports. Billing registers a hook into subscription.Service.RegisterHook() during wiring in app/common. The per-registry context key (pointer-identity fmt.Sprintf('%p', r)) prevents re-entrant invocations. |
| Pre-validating a customer mutation from another domain | Customer RequestValidator Registry | billing/validators/customer implements RequestValidator and registers via customerService.RegisterRequestValidator() in app/common/customer.go, keeping billing constraints out of the customer package and preventing import cycles. |
| Processing async domain events between binaries | Watermill Message Bus (NoPublishingHandler + GroupEventHandler) | Events published to Kafka via eventbus.Publisher are consumed by worker processes registering typed closures. Unknown event types are silently dropped, making workers tolerant of schema evolution. Topic isolation matches worker topology. |
| Invoice or charge lifecycle transitions | Invoice State Machine (stateless library) or generic Machine[CHARGE,BASE,STATUS] | The state machine enforces valid transitions and fires actions (DB save, event publish, external app calls) atomically. sync.Pool reduces GC pressure on the hot billing-worker path. |
| New billing backend (payment processor or invoicing system) | App Factory / Registry + InvoicingApp interface | New backends implement billing.InvoicingApp and self-register a factory via app.Service.RegisterMarketplaceListing in their New() constructor. No billing service code changes needed. |
| Disabling a subsystem (credits off, no Svix) | Noop implementations for optional features | Wire provider functions check config flags and return noops. The rest of the DI graph is unaffected, avoiding nil checks scattered through business logic. Compile-time assertions keep noops in sync with interfaces. Credits requires four independent guards: ledger services, customer hooks, ChargesRegistry, v3 HTTP handlers. |
| Per-customer serialization for billing operations | Locker (pg_advisory_xact_lock) inside TransactingRepo | Advisory locks are transactional and released automatically on commit/rollback. Requires an active Postgres transaction in context — call inside entutils.TransactingRepo. Confirmed at lockr/locker.go:100-137. |
| HTTP error response to client | RFC 7807 Problem Details + GenericErrorEncoder chain | Domain errors (GenericNotFoundError → 404, GenericValidationError → 400, etc.) are matched by type in GenericErrorEncoder via HandleErrorIfTypeMatches. ValidationIssues with explicit HTTP status attributes are handled by HandleIssueIfHTTPStatusKnown. All errors render as application/problem+json. |
| Batch usage event ingestion from Kafka | Sink Worker (Kafka to ClickHouse three-phase batch flush) | High-throughput events flow Kafka -> SinkBuffer -> ClickHouse in micro-batches. Strict three-phase flush ordering (ClickHouse -> offset commit -> Redis dedupe at sink.go:307-358) ensures exactly-once semantics. FlushEventHandler always called in goroutine. |
| Outbound webhook notifications | Svix integration via webhook.Handler interface | Svix handles fan-out, retry, signature verification, and delivery status. NullChannel sentinel at svix/svix.go:20-26 prevents unfiltered delivery. Noop implementation runs in tests or when Svix is unconfigured. |
| Cross-domain DB operations in transactions | entutils.TransactingRepo / TransactingRepoWithNoValue | Ent transactions propagate implicitly via context. TransactingRepo reads *TxDriver from ctx — if found rebinds to existing transaction; if not found runs on Self(). Savepoints enable safe nesting via TxDriver.SavePoint() confirmed at transaction.go:148-173. |
| HTTP endpoint handler | httptransport.NewHandler with RequestDecoder + Operation + ResponseEncoder | Consistent request validation, error encoding, and OTel tracing across all endpoints without duplicating boilerplate. GenericErrorEncoder always appended as defaultHandlerOptions at handler.go:17-19. |
| New binary that needs all domain services | Google Wire DI with app/common provider sets | Wire generates compile-time-verified dependency graphs. Adding a new binary requires a matching app/common/openmeter_<binary>.go provider set file and a wire.go in cmd/<binary>/. Confirmed at cmd/billing-worker/wire.go:35-58. |
| New charge type engine for billing line computation | LineEngine Plugin Registry + RegisterLineEngine in app/common | Each charge type implements LineEngine and registers via billing.Service.RegisterLineEngine in app/common. No billing service core changes needed. Confirmed at billing/service.go:63-67. |

## Quick Pattern Lookup

- **new domain feature** -> Layered Domain Service/Adapter/HTTP in openmeter/<domain>/  *(scope: openmeter/billing, openmeter/customer, openmeter/entitlement, openmeter/subscription)*
- **lifecycle side-effects** -> ServiceHookRegistry (models.ServiceHooks[T]) or SubscriptionCommandHook  *(scope: openmeter/customer, openmeter/subscription, app/common)*
- **pre-mutation validation across domains** -> Customer RequestValidator Registry  *(scope: openmeter/customer, app/common)*
- **async domain events between binaries** -> Watermill NoPublishingHandler + GroupEventHandler on SystemEventsTopic  *(scope: openmeter/watermill, openmeter/billing/worker, openmeter/entitlement/balanceworker)*
- **invoice/charge state transitions** -> stateless-backed InvoiceStateMachine or generic Machine[CHARGE,BASE,STATUS]  *(scope: openmeter/billing, openmeter/billing/charges)*
- **new billing backend** -> Implement billing.InvoicingApp + AppFactory self-registration  *(scope: openmeter/app, openmeter/billing)*
- **optional feature disabled** -> Return noop implementation in Wire provider function when config flag is false (four-layer guard for credits)  *(scope: app/common)*
- **per-customer serialization** -> billing.Service.WithLock -> lockr.Locker.LockForTX (pg advisory lock in tx)  *(scope: openmeter/billing, openmeter/billing/charges)*
- **DB operations in transactions** -> entutils.TransactingRepo / TransactingRepoWithNoValue  *(scope: openmeter/billing/adapter, openmeter/billing/charges/adapter, openmeter/customer/adapter)*
- **HTTP handler** -> httptransport.NewHandler with RequestDecoder + Operation + ResponseEncoder  *(scope: openmeter/billing/httpdriver, api/v3/handlers)*
- **batch usage ingestion** -> confluent-kafka-go consumer in Sink worker, ClickHouseStorage.BatchInsert (three-phase flush)  *(scope: openmeter/sink, cmd/sink-worker)*
- **outbound webhooks** -> notification.EventHandler -> webhook.Handler (Svix or noop)  *(scope: openmeter/notification, cmd/notification-service)*
- **DI wiring** -> Google Wire: wire.NewSet in app/common/, wire.Build in cmd/<binary>/wire.go  *(scope: app/common, cmd/server, cmd/billing-worker)*
- **structured validation errors** -> models.ValidationIssue with WithPathString + commonhttp.WithHTTPStatusCodeAttribute  *(scope: openmeter/billing, openmeter/billing/charges)*
- **domain error HTTP mapping** -> models.GenericXxxError wrapped in service/adapter, matched by commonhttp.GenericErrorEncoder
- **multi-language SDK contract** -> TypeSpec in api/spec/ -> make gen-api -> OpenAPI -> oapi-codegen + JS/Python generators  *(scope: api/spec)*

## Decision Chain

**Root constraint:** Operate a high-volume per-tenant usage-metering platform that feeds strict financial billing correctness, while shipping stable SDKs in three languages — under a single small team that cannot maintain separate repos or hand-synchronized contracts.

- **Split the runtime into seven independently deployable binaries (cmd/server + five workers + jobs CLI + benthos-collector) that share one domain-package tree under openmeter/.**: Ingest throughput, balance recalculation, billing advancement, and webhook dispatch have incompatible scaling and failure profiles, but billing correctness needs one typed domain model — so split the processes, share the types.
  - *Violation keyword:* `business logic in cmd/*/main.go`
  - *Violation keyword:* `goroutine spawned outside run.Group`
  - *Violation keyword:* `new cmd/* binary without app/common/openmeter_<binary>.go`
  - *Violation keyword:* `domain package importing app/common`
  - *Violation keyword:* `shared in-memory state between binaries`
  - **Compose each binary with Google Wire provider sets concentrated in app/common/, keeping domain packages as import-cycle-free leaves.**: ~40 services per binary make hand-wiring error-prone; Wire gives compile-time graph verification, and keeping providers out of domain packages prevents import cycles.
    - *Violation keyword:* `wire.Build calling domain constructors directly`
    - *Violation keyword:* `provider function with validation/computation/panic/os.Exit`
    - *Violation keyword:* `domain package importing app/common`
    - *Violation keyword:* `viper.SetDefault in cmd/*`
    - **Register cross-domain ServiceHooks and RequestValidators as side-effects inside app/common provider functions.**: Billing must react to customer lifecycle and ledger must react to customer creation without billing/customer/ledger importing each other; app/common is the only place that can see all of them.
      - *Violation keyword:* `RegisterHooks called inside a domain constructor`
      - *Violation keyword:* `domain package importing another domain for a callback`
      - *Violation keyword:* `wire.Build omitting a hook provider`
      - *Violation keyword:* `RegisterRequestValidator outside app/common`
    - **Gate the credits.enabled feature flag at four independent wiring layers, each returning noop implementations.**: Credits writes fan out from HTTP handlers, customer hooks, namespace provisioning, and charge creation — there is no single choke point, so each wiring layer must guard independently.
      - *Violation keyword:* `ledger-touching Wire provider without creditsConfig.Enabled branch`
      - *Violation keyword:* `BillingRegistry.Charges accessed without ChargesServiceOrNil()`
      - *Violation keyword:* `nil returned instead of a noop struct`
      - *Violation keyword:* `v3 credit handler registered without s.Credits.Enabled check`
  - **Use Kafka via Watermill as the sole inter-binary channel, with three name-prefix-routed topics (ingest, system, balance-worker).**: Independently deployable workers need durable, replayable, backpressure-aware async delivery and topic isolation so ingest bursts cannot starve billing consumers.
    - *Violation keyword:* `kafka.NewProducer / confluent ProduceChannel / sarama SendMessage in domain code`
    - *Violation keyword:* `publishing to a topic by string literal`
    - *Violation keyword:* `EventName() without an EventVersionSubsystem prefix`
    - *Violation keyword:* `context.Background() inside a Watermill handler instead of msg.Context()`
    - *Violation keyword:* `returning an error for an unknown ce_type`
    - **Build all consumer routers via openmeter/watermill/router.NewDefaultRouter and dispatch via grouphandler.NoPublishingHandler that silently drops unknown ce_types.**: Rolling deploys mean producer and consumer event-type sets differ transiently; silent drop avoids DLQ poisoning, and the fixed middleware stack gives uniform retry/DLQ/OTel behaviour.
      - *Violation keyword:* `bare Watermill router without NewDefaultRouter`
      - *Violation keyword:* `returning error for unknown event type in NoPublishingHandler`
      - *Violation keyword:* `MaxRetries:0 assumed to mean no DLQ`
- **Author the entire HTTP surface once in TypeSpec (api/spec/) and generate both v1 and v3 OpenAPI specs, Go server stubs, and Go/JS/Python SDKs.**: Three SDK languages and two API versions cannot be hand-synchronized; a single upstream contract makes drift structurally impossible.
  - *Violation keyword:* `hand-edited api/openapi.yaml or api/v3/openapi.yaml`
  - *Violation keyword:* `endpoint added only in a Go handler package`
  - *Violation keyword:* `hand-edited *.gen.go`
  - *Violation keyword:* `@route in a domain sub-folder tsp instead of root openmeter.tsp`
  - *Violation keyword:* `TypeSpec edit without make gen-api && make generate`
  - **Run two HTTP validation surfaces — kin-openapi OapiRequestValidatorWithOptions for v1 and oasmiddleware.ValidateRequest for v3 — and route every handler through pkg/framework/transport/httptransport.Handler.**: Dual API versions need dual validation middleware; the generic decode/operate/encode pipeline keeps RFC 7807 error mapping and OTel instrumentation uniform across both.
    - *Violation keyword:* `handler implementing ServeHTTP directly`
    - *Violation keyword:* `writing http status codes in handler logic instead of models.Generic* sentinels`
    - *Violation keyword:* `chi.NewRouter without a request validator`
    - *Violation keyword:* `v3 handler placed in openmeter/*/httpdriver`
- **Persist to PostgreSQL via Ent ORM with Atlas-managed migrations, accessed through context-propagated transactions (entutils.TransactingRepo) and per-customer pg_advisory_xact_lock via lockr.**: Billing correctness needs compile-time-checked relations across ~60 entities, deterministic reviewable migrations, atomic multi-step charge/invoice mutation, and per-customer serialization against concurrent workers.
  - *Violation keyword:* `edits inside openmeter/ent/db/`
  - *Violation keyword:* `hand-written SQL alongside Ent queries`
  - *Violation keyword:* `*entdb.Tx as a struct field`
  - *Violation keyword:* `a.db.Foo() in an adapter without TransactingRepo`
  - *Violation keyword:* `LockForTX outside an active transaction`
  - *Violation keyword:* `manual edits to tools/migrate/migrations/ or atlas.sum`
  - *Violation keyword:* `context.WithTimeout around LockForTX`
  - **Model billing domain objects as tagged unions (Charge, InvoiceLine) with private discriminators, constructor-only construction, and a generic state machine + LineEngine registry.**: Multi-step charge advancement mixing reads, realization, locks, and ledger writes needs exhaustive unambiguous type dispatch and impossible partial construction.
    - *Violation keyword:* `charges.Charge{} struct literal`
    - *Violation keyword:* `charges.ChargeIntent{} struct literal`
    - *Violation keyword:* `billing.InvoiceLine{} struct literal`
    - *Violation keyword:* `direct Invoice.Status field mutation`
    - *Violation keyword:* `RegisterLineEngine called from a domain package or cmd/*`

## Key Decisions

### TypeSpec as the single source of truth for both v1 and v3 HTTP APIs and all three SDKs
**Chosen:** Author the HTTP surface in TypeSpec under api/spec/packages/legacy (v1) and api/spec/packages/aip (v3); `make gen-api` compiles to api/openapi.yaml + api/v3/openapi.yaml then oapi-codegen produces api/api.gen.go, api/v3/api.gen.go, api/client/go/client.gen.go, the JavaScript SDK and Python SDK; `make generate` then propagates Go-side changes through Ent, Wire, Goverter, and Goderive.
**Rationale:** Drift between Go server stubs, three SDKs, and two API versions is structurally impossible as long as both regen steps run. The generated v1 stubs are consumed by openmeter/server/router and v3 stubs by api/v3/handlers, so a TypeSpec change forces handler-side compile errors. Route and tag bindings are centralized in the root openmeter.tsp files.
**Rejected:** Hand-written OpenAPI YAML, Code-first OpenAPI from Go handlers, Single API version skipping v3 AIP, gRPC/Protobuf
**Forced by:** Multi-language SDK requirement (Go/JS/Python) plus dual API versions plus runtime request validation against the same artifact.
**Enables:** Cross-language SDK contracts that cannot drift; breaking-change detection at TypeSpec compile time; kin-openapi (v1) + oasmiddleware (v3) request validation against the same spec; parallel SDK evolution.

### Google Wire DI with all provider sets in app/common and cross-domain hooks registered as construction side-effects
**Chosen:** Each cmd/<binary>/wire.go declares a wire.Build over composite provider sets (common.BillingWorker, common.LedgerStack, etc.) defined in app/common/. Domain packages expose plain constructors and never import app/common. Cross-domain ServiceHooks and RequestValidators are registered inside app/common provider functions as side-effects of construction (e.g. app/common/customer.go NewCustomerLedgerServiceHook calls customerService.RegisterHooks(h); app/common/billing.go NewBillingRegistry calls customerService.RegisterRequestValidator and subscriptionServices.Service.RegisterHook).
**Rationale:** Wire produces a compile-time-checked dependency graph per binary so missing providers are build errors. Concentrating provider sets in app/common keeps the ~38 domain packages as leaf nodes with no DI-layer dependency, avoiding import cycles. Hook registration as side-effects lets billing react to customer lifecycle without billing and customer importing each other.
**Rejected:** Manual constructor calls in each cmd/main.go, Reflection-based runtime DI, Domain packages registering their own hooks, Provider functions containing business logic
**Forced by:** ~40 domain services per binary, very different per-binary provider graphs, and the need for cross-domain hooks without circular imports.
**Enables:** Compile-time proof of binary completeness; clean leaf-node domain packages; independent per-binary composition; cross-domain lifecycle reactions.

### Kafka + Watermill async backbone with three name-prefix-routed topics
**Chosen:** openmeter/watermill/eventbus wraps Watermill's cqrs.EventBus with a TopicMapping of IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic. GeneratePublishTopic (eventbus.go:135-143) routes by EventName() prefix: ingestevents.EventVersionSubsystem prefix to ingest, balanceworkerevents.EventVersionSubsystem prefix to balance-worker, everything else defaults to system. Consumers build routers via openmeter/watermill/router.NewDefaultRouter (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout, HandlerMetrics) and dispatch via grouphandler.NoPublishingHandler keyed on CloudEvents ce_type; unknown ce_types are silently dropped (grouphandler.go:54 returns nil).
**Rationale:** Topic isolation matches worker topology: ingest bursts must not starve billing system-event consumers. Prefix routing hides topology from producers. Silent drop of unknown event types enables rolling deploys where producer and consumer versions differ.
**Rejected:** Per-event explicit topic names, NATS or Redis Streams (weaker replay/durability), Postgres LISTEN/NOTIFY, Raw confluent-kafka-go without Watermill (loses uniform middleware), Erroring on unknown event types (poisons DLQ during rolling deploys)
**Forced by:** Ingest bursts plus cross-worker side-effects plus the need to deploy producers and consumers independently.
**Enables:** Replay, backpressure, decoupled producer/consumer evolution; rolling-deploy safety; per-topic consumer scaling and DLQ semantics; uniform OTel + correlation-id middleware.

### Ent ORM + Atlas migrations with context-propagated transactions via entutils.TransactingRepo
**Chosen:** openmeter/ent/schema holds Go-defined entity schemas; `make generate` regenerates openmeter/ent/db/; `atlas migrate --env local diff` produces timestamped .up.sql/.down.sql plus an atlas.sum hash chain. Every domain adapter implements the TxCreator + TxUser triad (Tx via HijackTx + NewTxDriver, WithTx via NewTxClientFromRawConfig, Self) and wraps every method body in entutils.TransactingRepo / TransactingRepoWithNoValue, which reads the *TxDriver from ctx (transaction.go:199-221) and rebinds to the caller's transaction or falls back to Self().
**Rationale:** Atlas diffs the Ent schema against migration history to produce deterministic reviewable SQL; Ent gives compile-time-checked relations across ~60 entities. TransactingRepo lets adapter helpers participate in caller-supplied transactions without threading *entdb.Tx through every signature, and supports savepoint-based nesting for multi-step flows like charge advancement and invoice mutation.
**Rejected:** Raw golang-migrate only (no typed entities), GORM (weaker typing, no native schema diff), sqlc (schema still hand-rolled), Explicit *entdb.Tx threaded through every call site, Global transaction middleware
**Forced by:** Billing correctness plus multi-tenant schema invariants requiring compile-time-checked relations across ~60 entities, plus ctx-propagated transaction reuse.
**Enables:** Deterministic reviewable SQL migrations with atlas.sum integrity; typed relations across all entities; ctx-propagated transactions with savepoint nesting; atomic charge advancement and invoice mutation.

### credits.enabled feature flag enforced at four independent wiring layers via noop implementations
**Chosen:** When config.Credits.Enabled is false: app/common/ledger.go returns ledgernoop.* implementations from each provider; app/common/customer.go NewCustomerLedgerServiceHook returns NoopCustomerLedgerHook; app/common/billing.go NewBillingRegistry skips newChargesRegistry entirely (BillingRegistry.Charges stays nil, accessed via ChargesServiceOrNil()); api/v3/server credit handlers skip registration. NewLedgerNamespaceHandler additionally type-asserts against ledgernoop.AccountResolver.
**Rationale:** Credits cross-cut ledger writes, customer lifecycle hooks, namespace default-account provisioning, charge creation in billing/charges, and v3 HTTP handlers. There is no single choke point — a customer creation in api/v3 fans out through independent code paths — so each wiring layer guards independently and returns a noop interface rather than nil to avoid scattered nil-checks.
**Rejected:** Single global runtime flag check inside ledger.Ledger, Top-level HTTP middleware blocking credits endpoints, Compile-time build tag, Returning nil instead of a noop struct
**Forced by:** The cross-cutting nature of credit accounting and the customer/billing/ledger hook fan-out across unrelated call graphs.
**Enables:** Credits-disabled tenants produce zero ledger_accounts/ledger_customer_accounts rows; per-deployment enabling without rebuild; compile-time interface satisfaction for noop implementations.

### Tagged-union domain models (Charge, InvoiceLine) with private discriminators and constructor-only construction
**Chosen:** openmeter/billing/charges owns the Charge / ChargeIntent tagged-union discriminated by a private meta.ChargeType field, constructed only via NewCharge[T] / NewChargeIntent[T] and accessed via AsFlatFeeCharge / AsUsageBasedCharge / AsCreditPurchaseCharge. openmeter/billing owns the InvoiceLine tagged-union with a private discriminator, constructed via NewStandardInvoiceLine / NewGatheringInvoiceLine and accessed via AsStandardLine / AsGatheringLine / AsGenericLine. Each charge type plugs into a generic Machine[CHARGE,BASE,STATUS] state machine and registers a LineEngine with billing.Service.RegisterLineEngine in app/common/charges.go.
**Rationale:** Exhaustive type-dispatch across charge types and invoice line types must be enforced, and partial construction must be impossible. A struct-literal Charge{} leaves the discriminator zero-valued and accessors error; the constructor-only contract makes the discriminator always correct. The generic state machine shares fire/activate/persist/refetch mechanics across all three charge types.
**Rejected:** Charge as a Go interface (loses exhaustive compile-time dispatch), Public discriminator field (allows partial/inconsistent construction), Hardcoding charge-type branches in billing.Service
**Forced by:** Multi-step charge advancement that mixes reads, realization runs, advisory locks, and ledger-bound writes, requiring exhaustive and unambiguous charge-type dispatch.
**Enables:** Deterministic atomic charge advancement; exhaustive charge-type and invoice-line-type dispatch; runtime-pluggable charge engines via the LineEngine registry; per-charge advisory locking via charges.NewLockKeyForCharge.

## Trade-offs Accepted

- **Accepted:** Ent-generated query friction: a large openmeter/ent/db/ generated tree, slower compile times, and the boilerplate Tx/WithTx/Self triad plus a TransactingRepo wrapper on every adapter method body.
  - *Benefit:* Compile-time-checked relations across ~60 entities, automatic Atlas schema diffing, no runtime schema surprises, and ctx-propagated transactions with savepoint nesting.
  - *Caused by:* Ent ORM + Atlas migration pipeline + entutils.TransactingRepo discipline.
  - *Violation signal:* Hand-written db.Exec/db.QueryContext SQL added alongside Ent queries in an adapter
  - *Violation signal:* Direct edits inside openmeter/ent/db/
  - *Violation signal:* A new table created without a corresponding openmeter/ent/schema/*.go file
  - *Violation signal:* An adapter struct storing *entdb.Tx as a field instead of using TransactingRepo
  - *Violation signal:* A helper accepting *entdb.Client that skips TransactingRepoWithNoValue
- **Accepted:** Multi-binary orchestration cost: seven Docker image variants, Helm values complexity, and a separate Wire graph per binary that must each be kept complete.
  - *Benefit:* Independent horizontal scaling of sink-worker / balance-worker / billing-worker, fault isolation per binary, and isolated deploy cadence.
  - *Caused by:* Multi-binary deployment of cmd/server, cmd/billing-worker, cmd/balance-worker, cmd/sink-worker, cmd/notification-service.
  - *Violation signal:* Business logic added inside cmd/*/main.go beyond startup orchestration
  - *Violation signal:* A new worker binary added without a matching app/common/openmeter_<binary>.go Wire set
  - *Violation signal:* Cross-binary dependencies introduced through shared in-memory state or HTTP calls instead of a Kafka topic
  - *Violation signal:* A goroutine spawned outside the oklog/run.Group
- **Accepted:** Two-step regeneration cadence: TypeSpec changes require both `make gen-api` AND `make generate`, and five independent generators (oapi-codegen, Ent, Wire, Goverter, Goderive) write different artifacts that must all stay in sync.
  - *Benefit:* Cross-language SDK contracts cannot drift — Go server stubs, Go SDK, JS SDK, Python SDK all originate from a single TypeSpec source.
  - *Caused by:* TypeSpec -> OpenAPI -> oapi-codegen + Wire/Ent/Goverter/Goderive generator stack.
  - *Violation signal:* Hand-edits inside *.gen.go, wire_gen.go, api/api.gen.go, or api/v3/api.gen.go
  - *Violation signal:* PRs touching api/spec/ without regenerated api/openapi.yaml
  - *Violation signal:* Client SDKs under api/client/** drifting from api/spec/
  - *Violation signal:* TypeSpec edits without rerunning make generate
  - *Violation signal:* A new endpoint added only in a Go handler package without a TypeSpec source change
- **Accepted:** Cross-domain wiring is invisible to the compiler: hook/validator registration and credits guards are side-effects scattered across app/common provider functions, and Kafka topic routing depends on event-name string prefixes.
  - *Benefit:* Domain packages stay import-cycle-free leaves; optional features are gated without nil-checks in business logic; the three-topic topology is hidden from producers.
  - *Caused by:* ServiceHookRegistry / RequestValidator registries + the credits.enabled four-layer guard + EventName-prefix topic routing in eventbus.GeneratePublishTopic.
  - *Violation signal:* A binary's wire.Build omitting a hook provider so the hook silently never registers
  - *Violation signal:* A new ledger-touching Wire provider added without a creditsConfig.Enabled branch
  - *Violation signal:* Direct access to BillingRegistry.Charges without ChargesServiceOrNil()
  - *Violation signal:* A new event family whose EventName() lacks a recognized EventVersionSubsystem prefix, silently routing to SystemEventsTopic
- **Accepted:** Sequential timestamped Atlas migration filenames plus an atlas.sum linear hash chain that, by construction, produces merge conflicts on any two branches that both append migrations.
  - *Benefit:* Deterministic, reviewable, linearly-ordered SQL migration history with cryptographic chain integrity verified by CI's `make migrate-check`.
  - *Caused by:* atlas migrate --env local diff filename convention + atlas.sum chain hashing.
  - *Violation signal:* Two branches producing same-timestamp migration files
  - *Violation signal:* atlas.sum merge conflicts on a long-running branch
  - *Violation signal:* Manual edits to an already-landed migration file
  - *Violation signal:* Commits touching tools/migrate/migrations/ without an accompanying atlas.sum update

## Out of Scope

- Frontend UI — no React/Vue application in the repo; React appears only inside the generated JavaScript SDK under api/client/javascript/. Forced out of scope by the API-as-product / SDK-generation decision.
- Tenant-level identity and auth provider — portal tokens scope end-customers via HS256 JWTs, but tenant identity is delegated to the deployment. Forced out by the self-hosted single-namespace deployment model.
- Managed hosting control plane — config.cloud.yaml and api/openapi.cloud.yaml expose hooks but cloud orchestration logic lives separately.
- Real-time streaming queries from clients — ClickHouse is reached only via streaming.Connector inside the server process; there is no client-facing streaming surface.
- Multi-region active/active replication — a single PostgreSQL primary is assumed; ClickHouse cluster topology is deployment-defined. Forced out by the Ent + single-primary persistence decision.
- Synchronous cross-binary RPC — all inter-binary communication goes through the three Kafka topics; there is no service mesh or gRPC surface between binaries.