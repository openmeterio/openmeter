## Communication Patterns

### Layered Domain Service/Adapter/HTTP
- **Scope:** `openmeter/billing`, `openmeter/customer`, `openmeter/entitlement`, `openmeter/subscription`, `openmeter/notification`, `openmeter/meter`, `openmeter/ledger`, `openmeter/productcatalog`, `openmeter/app`, `openmeter/llmcost`, `openmeter/subject`, `openmeter/credit`
- **When:** All business-logic domains under openmeter/<domain>/. Applied whenever persistence must be separated from orchestration and HTTP translation.
- **How:** Each domain exposes a Go interface (e.g. billing.Service, customer.Service) defined in <domain>/service.go. A concrete service struct in <domain>/service/ holds business logic and calls an Adapter interface for all DB access. The Adapter interface is defined alongside the Service interface and implemented by Ent-backed structs in <domain>/adapter/ sub-packages. HTTP handlers live in <domain>/httpdriver/ or <domain>/httphandler/. Service interfaces compose fine-grained sub-interfaces (e.g. ProfileService, InvoiceService) so callers depend only on the smallest surface they need.
- **Applicable when:** Adapters implementing TxCreator (Tx via HijackTx + NewTxDriver) and TxUser[T] (WithTx + Self) — every method body must use entutils.TransactingRepo so the ctx-bound Ent transaction is honored; verified at openmeter/billing/adapter/adapter.go and openmeter/customer/adapter/ — callers that bypass this rebinding produce partial writes under concurrency.
- **Do NOT apply when:**
  - Implementing a domain adapter method that uses the raw *entdb.Client directly without TransactingRepo wrapping, as verified by the pattern at pkg/framework/entutils/transaction.go:199 — the TransactingRepo helper must be called on every body, even in shared helpers.
  - Placing business logic inside cmd/*/main.go — cmd/* must only wire and start; domain logic belongs under openmeter/.

### Google Wire Dependency Injection
- **Scope:** `app/common`, `cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`
- **When:** Assembling runtime components for each binary. Each cmd/<binary>/wire.go declares a wire.Build call; provider sets live in app/common/.
- **How:** Google Wire generates cmd/<binary>/wire_gen.go at build time. Reusable provider sets (e.g. common.Billing, common.Notification, common.LedgerStack) are declared as wire.NewSet() in app/common/ and compose individual factory functions. Wire resolves the dependency graph automatically. Each Application struct in cmd/<binary>/wire.go lists every needed service as a field and Wire auto-wires them. Hook and validator registration is done as side-effects inside provider functions in app/common to avoid circular imports.
- **Applicable when:** Binary entrypoints that must compose ~40 domain services compile-time safely — Wire provider sets in app/common are verified at wire.Build compile time, as seen in cmd/server/wire.go and cmd/billing-worker/wire.go; missing providers cause compile errors not runtime panics.
- **Do NOT apply when:**
  - Domain packages importing app/common — as enforced by the import-direction rule at openmeter/CLAUDE.md, domain packages must not import app/common because it creates import cycles.
  - Provider functions containing business logic (validation, computation, state mutation) — providers must only construct and wire, as documented in app/common/CLAUDE.md.

### Registry Structs for Multi-Service Domains
- **Scope:** `app/common`, `api/v3/server`
- **When:** When a domain exposes multiple related services that callers must access together, reducing Wire graph complexity.
- **How:** A <Domain>Registry struct groups logically cohesive services (e.g. BillingRegistry, AppRegistry, ChargesRegistry). Callers depend on the registry rather than individual services. Nil-safe accessor methods (e.g. BillingRegistry.ChargesServiceOrNil()) encapsulate optional sub-registries when features are disabled.
- **Applicable when:** Multi-service domains where one service may be conditionally nil at runtime — ChargesRegistry is nil when credits.enabled=false, so ChargesServiceOrNil() at app/common/billing.go:49 provides the nil-safe accessor; direct field access on BillingRegistry.Charges would panic.
- **Do NOT apply when:**
  - Accessing BillingRegistry.Charges directly without the nil-safe ChargesServiceOrNil() accessor — verified at app/common/billing.go:49 where ChargesServiceOrNil returns nil when credits are disabled.

### Noop Implementations for Optional Features
- **Scope:** `app/common`, `openmeter/ledger`, `openmeter/notification/webhook`
- **When:** When a feature is disabled at runtime (credits.enabled=false, Svix not configured, etc.), to avoid nil-pointer checks scattered through business logic.
- **How:** app/common provider functions check config flags and return noop structs instead of real implementations. All noop types implement the relevant interface via compile-time assertions. Callers receive a real interface and never branch on nil. Type assertions against noop types are used in some cases to conditionally skip handler registration.
- **Applicable when:** Any provider that wires ledger-backed features when credits.enabled=false — app/common/ledger.go and app/common/customer.go independently guard with creditsConfig.Enabled as verified by the four-layer rule; a single centralized guard is insufficient because credits cross-cuts multiple independent call graphs.
- **Do NOT apply when:**
  - Features that are always required (non-optional) — use real implementations directly without an enabled flag.
  - Returning nil instead of a noop struct — callers receive the interface and will panic on method calls if nil is assigned to the interface field.

### App Factory / Registry (External Billing App Protocol)
- **Scope:** `openmeter/app`, `openmeter/billing`
- **When:** Plugging Stripe, Sandbox, and CustomInvoicing billing apps into the billing state machine without hardcoding them.
- **How:** openmeter/app/registry.go defines AppFactory and RegistryItem (Listing + Factory). app.Service.RegisterMarketplaceListing is called from each app type's New() constructor (self-registration). Installed apps implement billing.InvoicingApp interface. Optional InvoicingAppPostAdvanceHook and InvoicingAppAsyncSyncer extend it. GetApp() type-asserts an installed App to InvoicingApp at runtime.
- **Applicable when:** Billing backends that implement billing.InvoicingApp (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice) — the AppFactory self-registers at constructor time via app.Service.RegisterMarketplaceListing, so no modifications to billing.Service core are needed per openmeter/app/registry.go:64.
- **Do NOT apply when:**
  - Adding provider-specific logic directly inside billing.Service — new billing backends must implement InvoicingApp and register via the factory, not add conditional branches in the core service.

### LineEngine Plugin Registry
- **Scope:** `openmeter/billing`, `app/common`
- **When:** Dispatching billing line calculation to the correct engine (standard invoice, charge flatfee, charge usagebased, charge creditpurchase) based on LineEngineType discriminator.
- **How:** billing.Service exposes RegisterLineEngine / DeregisterLineEngine. billingservice.engineRegistry stores a map[LineEngineType]LineEngine under a RWMutex. Each charge type implements its own Engine and registers it at startup via app/common/charges.go. The service.New() constructor also pre-registers the standard invoice line engine.
- **Applicable when:** Charge types that need to register their own line engine — engines must be registered before the first invoice advance, as codified in app/common/charges.go via RegisterLineEngine side-effects at Wire provider functions.
- **Do NOT apply when:**
  - Registering line engines from domain packages instead of app/common — as documented in dep-006, engine registration must happen in app/common to avoid circular imports.

### ServiceHook Registry
- **Scope:** `openmeter/customer`, `openmeter/subscription`, `openmeter/app`, `app/common`
- **When:** Cross-domain lifecycle callbacks without circular imports. Used by customer.Service, subscription.Service, and billing.Service.
- **How:** pkg/models/servicehook.go defines a generic ServiceHook[T] interface (PreUpdate, PreDelete, PostCreate, PostUpdate, PostDelete) and thread-safe ServiceHookRegistry[T] that fans out to all registered hooks. Loop prevention: a per-registry context key (pointer-identity string via fmt.Sprintf('%p', r)) prevents re-entrant invocations. Domain services embed *ServiceHookRegistry and expose RegisterHooks externally so other packages register callbacks without import cycles.
- **Applicable when:** Cross-domain lifecycle reactions where direct package imports would create circular dependencies — billing hooking into customer lifecycle must go through ServiceHookRegistry registered in app/common, not via direct import, as enforced by dep-003.
- **Do NOT apply when:**
  - Domain packages calling RegisterHooks on another domain's service inside their own constructors — this creates circular imports; always register in app/common provider functions as verified at openmeter/customer/CLAUDE.md.

### Customer RequestValidator Registry
- **Scope:** `openmeter/customer`, `app/common`
- **When:** Pre-mutation validation for customer operations where billing, subscription, or entitlement constraints must be checked before the customer is modified or deleted.
- **How:** openmeter/customer/requestvalidator.go defines RequestValidator interface (ValidateDeleteCustomer, ValidateCreateCustomer, ValidateUpdateCustomer) and a thread-safe requestValidatorRegistry that fans out to all registered validators using errors.Join. Validators are registered in app/common/customer.go to avoid circular imports.
- **Applicable when:** Pre-mutation guards from billing or entitlement domains that must block customer operations before any DB write — the registry in openmeter/customer/requestvalidator.go:41 fans out to all registered validators via errors.Join before any adapter write.
- **Do NOT apply when:**
  - Post-mutation reactions — these belong in ServiceHooks (PostCreate, PostUpdate, PostDelete), not in RequestValidatorRegistry which is exclusively for pre-mutation blocking validation.

### Invoice State Machine (stateless library)
- **Scope:** `openmeter/billing`, `openmeter/billing/service`
- **When:** Driving the invoice lifecycle transitions for StandardInvoice.
- **How:** openmeter/billing/service/stdinvoicestate.go builds a *stateless.StateMachine from sync.Pool with external storage bound to the InvoiceStateMachine struct's Invoice.Status field. Transitions trigger actions (DB save, event publish). FireAndActivate fires a trigger and persists; AdvanceUntilStateStable runs all allowed auto-transitions.
- **Applicable when:** Invoice lifecycle operations that must enforce valid transition sequences — the stateless.StateMachine in billing/service/stdinvoicestate.go prevents invalid transitions at runtime; direct Invoice.Status field mutation bypasses these guards and leaves the invoice in an inconsistent state.
- **Do NOT apply when:**
  - Directly mutating Invoice.Status fields without going through FireAndActivate — the state machine enforces valid transitions and fires post-transition actions (DB save, event publish) that cannot be replicated by direct field mutation.

### Generic Charge State Machine (Machine[CHARGE, BASE, STATUS])
- **Scope:** `openmeter/billing/charges`
- **When:** Driving charge lifecycle (flatfee, usagebased, creditpurchase) with shared mechanics (fire, activate, persist-base, refetch).
- **How:** openmeter/billing/charges/statemachine/machine.go defines generic Machine[CHARGE ChargeLike[CHARGE,BASE,STATUS], BASE any, STATUS Status]. External storage binds state to in-memory CHARGE. FireAndActivate fires a trigger and persists BASE; AdvanceUntilStateStable walks TriggerNext transitions. Each charge type instantiates Machine with concrete types.
- **Applicable when:** Charge types implementing ChargeLike[CHARGE,BASE,STATUS] — all three methods (GetStatus, WithStatus returning a copy, GetBase, WithBase returning a copy) must be pure value-returning functions; pointer-mutating implementations break the external storage pattern at statemachine/machine.go:58.
- **Do NOT apply when:**
  - Implementing WithStatus or WithBase as pointer receivers that mutate in place — they must return new value copies because Machine.Charge is updated by assignment.

### Watermill Message Bus (Kafka-backed publish/subscribe)
- **Scope:** `openmeter/watermill`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification/consumer`, `openmeter/sink`
- **When:** Async domain-event delivery between separate binaries. Used for subscription lifecycle events, billing invoice advance events, ingest flush events, and balance-worker recalculation events.
- **How:** openmeter/watermill/eventbus/eventbus.go wraps Watermill's cqrs.EventBus. Topic routing is by event-name prefix: events whose EventName() starts with ingestevents.EventVersionSubsystem go to IngestEventsTopic; balanceworkerevents.EventVersionSubsystem go to BalanceWorkerEventsTopic; everything else to SystemEventsTopic. Workers subscribe via Watermill's Kafka subscriber and dispatch to typed handlers registered in grouphandler.NewNoPublishingHandler. Unknown event types are silently dropped.
- **Applicable when:** Domain event producers that must route to one of three isolated Kafka topics — EventName() must begin with a recognized EventVersionSubsystem prefix; otherwise GeneratePublishTopic at eventbus.go:134 silently routes to SystemEventsTopic.
- **Do NOT apply when:**
  - Publishing directly to a Kafka topic string — always use eventbus.Publisher which encapsulates routing; raw producer calls bypass the three-topic isolation.
  - Returning errors for unknown event types in consumer handlers — NoPublishingHandler at grouphandler.go:48 silently drops them to support rolling deploys; returning errors causes Watermill retries and DLQ poisoning.

### Namespace Multi-tenancy via Manager + Handler Fan-out
- **Scope:** `openmeter/namespace`, `cmd/server`
- **When:** Provisioning and deprovisioning tenants across all subsystems (ClickHouse, Kafka ingest, Ledger).
- **How:** openmeter/namespace/namespace.go defines Manager which fans out CreateNamespace/DeleteNamespace to all registered Handler implementations. Handlers are registered via RegisterHandler before CreateDefaultNamespace is called at startup. Fan-out uses errors.Join (no short-circuit on partial failure). The default namespace is protected from deletion.
- **Applicable when:** Subsystems that must provision resources per namespace — namespace.Handler implementations must be registered via RegisterHandler before CreateDefaultNamespace is called at startup (namespace.go:91) to receive the default namespace creation event.
- **Do NOT apply when:**
  - Registering a Handler after CreateDefaultNamespace has been called — it will miss default namespace provisioning for the default tenant.

### entutils.TransactingRepo (Context-propagated Transactions)
- **Scope:** `openmeter/billing/adapter`, `openmeter/billing/charges/adapter`, `openmeter/customer/adapter`, `openmeter/entitlement`, `openmeter/subscription`, `openmeter/notification/adapter`, `openmeter/ledger`
- **When:** All DB adapter methods that must run inside a caller-supplied transaction or start their own.
- **How:** pkg/framework/entutils/transaction.go defines TransactingRepo[R,T] and TransactingRepoWithNoValue[T]. They read the *TxDriver from context via GetDriverFromContext. If found, the adapter's WithTx(ctx, tx) creates a txClient from raw Ent config. If none is found, the adapter runs on Self() and starts its own transaction. Savepoints enable nested calls with partial rollback.
- **Applicable when:** Adapter methods that are called both standalone and inside multi-step transactions — TransactingRepo at pkg/framework/entutils/transaction.go:199 reads *TxDriver from ctx and rebinds to the caller's transaction if present, or uses Self() to start its own; this prevents partial writes when called within AdvanceCharges or similar multi-step flows.
- **Do NOT apply when:**
  - Adapter methods that directly call a.db.Foo() without the TransactingRepo wrapper — they ignore the ctx-bound Ent transaction and produce partial writes under concurrency, as documented in ctx-002.

### Locker (pg_advisory_xact_lock)
- **Scope:** `openmeter/billing`, `openmeter/billing/charges`, `openmeter/entitlement`
- **When:** Distributed mutual exclusion for per-customer billing operations to prevent concurrent invoice generation races.
- **How:** pkg/framework/lockr/locker.go calls pg_advisory_xact_lock($1) with a CRC64-based hash of the lock key. Requires an active Postgres transaction in context (from GetDriverFromContext). Lock is released automatically on tx commit/rollback.
- **Applicable when:** Per-customer billing mutations where concurrent goroutines can race on invoice creation or charge advancement — LockForTX at lockr/locker.go:45 requires an active Postgres transaction in ctx extracted via entutils.GetDriverFromContext; calling outside a transaction returns an error.
- **Do NOT apply when:**
  - Calling LockForTX outside an active Postgres transaction — the function validates at locker.go:100 that a transaction is already in ctx and returns an error if none is found.
  - Using context.WithTimeout for lock acquisition — pgx cancels the connection on ctx cancel; use pgdriver.WithLockTimeout instead.

### httptransport.Handler Decode/Operate/Encode Pipeline
- **Scope:** `openmeter/billing/httpdriver`, `openmeter/customer/httpdriver`, `openmeter/meter/httphandler`, `api/v3/handlers`
- **When:** HTTP endpoint handlers in domain packages that separate request decoding, business logic, and response encoding.
- **How:** pkg/framework/transport/httptransport/handler.go defines Handler[Request, Response] wrapping an operation.Operation[Request, Response]. Decoding and encoding are injected via RequestDecoder and ResponseEncoder function types. ErrorEncoders form a chain; the first returning true short-circuits. Chain method wraps the operation with middleware.
- **Applicable when:** HTTP endpoints that must map domain errors to RFC 7807 problem+json responses — the GenericErrorEncoder at httptransport/handler.go:17 is always appended as a default option via defaultHandlerOptions; custom encoders passed before it take precedence.
- **Do NOT apply when:**
  - Implementing ServeHTTP directly in a handler struct — this bypasses defaultHandlerOptions (GenericErrorEncoder), OTel instrumentation, and the chain pattern.
  - Writing HTTP status codes directly in handler logic — use models.Generic* sentinel errors mapped by GenericErrorEncoder instead.

### Sink Worker (Kafka to ClickHouse Batch Flush)
- **Scope:** `openmeter/sink`, `cmd/sink-worker`
- **When:** High-throughput ingestion path: raw CloudEvents arrive via Kafka, are buffered, deduplicated, and batch-inserted into ClickHouse.
- **How:** openmeter/sink/sink.go consumes Kafka partitions via confluent-kafka-go. A SinkBuffer accumulates messages; flush is triggered by MinCommitCount or MaxCommitWait. Flush ordering is strict: ClickHouse insert then Kafka offset commit then Redis dedupe. After flush, FlushEventHandlers are called post-flush in a goroutine with timeout for downstream notifications.
- **Applicable when:** Exactly-once usage event ingestion where ClickHouse must be written before Kafka offset commit — reversing the three-phase flush order (ClickHouse → offset commit → Redis dedupe at sink.go) breaks the exactly-once guarantee on consumer restart.
- **Do NOT apply when:**
  - Calling FlushEventHandler.OnFlushSuccess synchronously inside flush() — this blocks the main sink loop and causes Kafka partition backpressure.

### Subscription Sync Reconciler
- **Scope:** `openmeter/billing/worker`, `cmd/billing-worker`
- **When:** Crash-recovery for the event-driven billing sync: periodically re-syncs subscriptions that may have missed their events.
- **How:** openmeter/billing/worker/subscriptionsync/reconciler/reconciler.go iterates customers/subscriptions in windows and calls subscriptionsync.Service.SynchronizeSubscriptionAndInvoiceCustomer for each. The reconciliation is idempotent so duplicate calls are safe.
- **Applicable when:** Subscription billing synchronization that must be resilient to missed Kafka events — SynchronizeSubscriptionAndInvoiceCustomer is idempotent so duplicate calls from both the event-driven path and the periodic reconciler are safe.

### ValidationIssue Structured Error Propagation
- **Scope:** `openmeter/billing`, `openmeter/billing/charges`, `openmeter/customer`, `openmeter/notification`, `openmeter/subscription`
- **When:** Domain-level validation errors that must carry field paths, severity (critical/warning), component names, and arbitrary attributes through service layer boundaries.
- **How:** pkg/models/validationissue.go defines ValidationIssue as an immutable value type with copy-on-write With* methods. Constructed via NewValidationIssue(code, message, opts...) or NewValidationError/NewValidationWarning. The HTTP layer reads httpStatusCodeErrorAttribute attribute (set via commonhttp.WithHTTPStatusCodeAttribute) to produce the correct HTTP status. AsValidationIssues traverses an error tree to extract all ValidationIssue nodes.
- **Applicable when:** Field-level validation errors that must survive wrapping through multiple service layers and emerge at the HTTP boundary with correct status codes — WithHTTPStatusCodeAttribute at commonhttp/errors.go:82 attaches the status as an attribute on the ValidationIssue; HandleIssueIfHTTPStatusKnown reads it at the HTTP boundary.
- **Do NOT apply when:**
  - Mutating a ValidationIssue in place — all With* methods at pkg/models/validationissue.go return new copies; direct struct field assignment bypasses the immutability contract.

### RFC 7807 Problem Details HTTP Response
- **When:** All error responses from the REST API.
- **How:** pkg/models/problem.go defines Problem interface and StatusProblem struct serialized to application/problem+json. NewStatusProblem reads the request-id from Chi middleware context, maps 'context canceled' to 408, suppresses detail on 500. Extensions map carries validationErrors array when applicable. GenericErrorEncoder chains multiple typed error matchers.
- **Applicable when:** Any HTTP error response in the API — models.NewStatusProblem at pkg/models/problem.go plus the GenericErrorEncoder chain in pkg/framework/commonhttp/errors.go ensures all domain errors render as application/problem+json with correct status codes.

### Generic Typed Domain Errors (models.Generic* Sentinels)
- **Scope:** `openmeter/billing`, `openmeter/customer`, `openmeter/entitlement`, `openmeter/subscription`, `openmeter/notification`, `openmeter/app`
- **When:** Returning domain errors from service and adapter methods so the HTTP error encoder chain maps them to correct status codes.
- **How:** pkg/models/errors.go defines typed sentinel error structs: GenericNotFoundError, GenericConflictError, GenericValidationError, GenericForbiddenError, GenericUnauthorizedError, GenericNotImplementedError, GenericPreConditionFailedError, GenericStatusFailedDependencyError. Each has New* constructor, Unwrap(), and Is* predicate. HTTP layer maps these via HandleErrorIfTypeMatches[T] in commonhttp.GenericErrorEncoder.
- **Applicable when:** Service and adapter methods that must return errors mappable to HTTP status codes — the GenericErrorEncoder at commonhttp/errors.go:60 uses HandleErrorIfTypeMatches for each Generic* type; plain fmt.Errorf falls through to 500 Internal Server Error.
- **Do NOT apply when:**
  - Returning plain fmt.Errorf for domain conditions (not-found, conflict, validation) — the HTTP error encoder chain will fall through to 500 Internal Server Error for unrecognized error types.

### credits.enabled Multi-Layer Feature Flag
- **Scope:** `app/common`, `api/v3/server`, `openmeter/ledger`
- **When:** Disabling the credits/ledger subsystem when credits.enabled=false, ensuring no ledger writes occur from any path.
- **How:** credits.enabled must be honored at four independent wiring layers: (1) app/common wires ledger services to noop; (2) api/v3/server credit handlers must skip registration; (3) customer ledger hooks must be unregistered; (4) namespace default-account provisioning must skip ledger account creation. A single guard is insufficient because credits cross-cuts multiple unrelated call graphs.
- **Applicable when:** Any new provider or route that writes to ledger_accounts or ledger_customer_accounts — each must independently check creditsConfig.Enabled and return a noop, as all four wiring layers are independently guarded in app/common/ledger.go, app/common/customer.go, app/common/billing.go, and api/v3/server/routes.go.

## Integrations

| Service | Purpose | Integration point |
|---------|---------|-------------------|
| PostgreSQL | Primary relational store for all domain entities: billing profiles, invoices, customers, subscriptions, entitlements, notification channels, ledger accounts, meters, subjects, secrets, charges. | `openmeter/ent/schema/ (source of truth); generated code in openmeter/ent/db/; Atlas migrations in tools/migrate/migrations/. Accessed via *entdb.Client injected through Wire. Transactions managed by pkg/framework/entutils.TransactingRepo.` |
| ClickHouse | Append-only analytics store for raw usage events; queried for meter aggregations (count, sum, max, unique_count) via SQL builders. | `openmeter/streaming/clickhouse/ — event_query.go and meter_query.go build ClickHouse SQL via sqlbuilder. Connector interface in openmeter/streaming/connector.go. ClickHouseStorage in openmeter/sink/storage.go for batch inserts.` |
| Kafka (confluent-kafka-go + Watermill-Kafka) | Durable event bus for domain events (subscription lifecycle, invoice advance, ingest flush notifications, balance recalculation) and raw usage event ingestion. | `openmeter/watermill/driver/kafka/ — Publisher and Subscriber wrappers. Topic provisioning via KafkaTopicProvisioner in app/common. confluent-kafka-go used directly in openmeter/sink/sink.go for high-throughput ingest consumer.` |
| Redis | Optional deduplication store for ingest events (preventing double-counting on retry). | `openmeter/dedupe/redisdedupe/redisdedupe.go — Redis-backed Deduplicator. In-memory LRU fallback in openmeter/dedupe/memorydedupe/.` |
| Svix | Outbound webhook delivery for notification events (entitlement balance thresholds, invoice events). | `openmeter/notification/webhook/svix/svix.go — Svix API client wrapper. Registered event types passed to Svix application. Handler interface in openmeter/notification/webhook/handler.go with noop fallback when Svix is unconfigured.` |
| Stripe | Invoice syncing (upsert draft, finalize, collect payment) and customer sync for billing-enabled namespaces. | `openmeter/app/stripe/app.go implements billing.InvoicingApp (UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice). Stripe REST client in openmeter/app/stripe/client/. App self-registers via app.Service.RegisterMarketplaceListing in service/factory.go.` |
| Sandbox Invoicing App | No-op invoicing app used in development/testing to drive invoice state machine without external dependencies. | `openmeter/app/sandbox/app.go implements billing.InvoicingApp + InvoicingAppPostAdvanceHook. Registered as marketplace listing with type AppTypeSandbox.` |
| CustomInvoicing App | Webhook-driven invoicing app allowing external systems to receive invoice payloads and async-confirm sync completion. | `openmeter/app/custominvoicing/ — App implements InvoicingApp + InvoicingAppAsyncSyncer; factory in custominvoicing/factory.go.` |
| GOBL | Currency and numeric type library for currency-safe arithmetic and ISO 4217 currency code validation throughout billing and subscription. | `Imported as github.com/invopop/gobl/currency and github.com/invopop/gobl/num in productcatalog, subscription, billing, cost, and currencies packages.` |
| OpenTelemetry | Distributed tracing and metrics across all services. | `trace.Tracer injected via Wire into service constructors. OTel metric.Meter used in grouphandler and sink worker. app/common/telemetry.go bootstraps OTLP exporters.` |
| TypeSpec compiler | Single source of truth for HTTP API definitions, compiling to OpenAPI YAML and downstream Go server stubs and SDKs. | `api/spec/packages/ — TypeSpec source. make gen-api runs tsp compile to produce api/openapi.yaml, api/v3/openapi.yaml, and then oapi-codegen produces api/api.gen.go, api/v3/api.gen.go, api/client/go/client.gen.go.` |
| App Marketplace Registry (extension protocol) | Runtime-pluggable billing backend mechanism allowing Stripe, Sandbox, and CustomInvoicing to register themselves without hardcoded references in billing domain. | `openmeter/app/service.go — Service.RegisterMarketplaceListing; openmeter/app/registry.go — AppFactory, RegistryItem; each app's New() self-registers.` |
| LineEngine Registry (extension protocol) | Runtime-pluggable billing line calculation dispatch by LineEngineType. | `openmeter/billing/service.go — LineEngineService.RegisterLineEngine; app/common/charges.go registers all charge type engines.` |
| ServiceHook Registries (extension protocol) | Cross-domain lifecycle callbacks without circular imports — billing hooks customer lifecycle, ledger hooks customer creation. | `pkg/models/servicehook.go — ServiceHookRegistry[T]; registered in app/common/customer.go, app/common/subscription.go.` |
| Customer RequestValidator Registry (extension protocol) | Pre-mutation validation guards for customer operations from billing and entitlement domains. | `openmeter/customer/requestvalidator.go — requestValidatorRegistry; billing validator registered in app/common/customer.go.` |

## Pattern Selection Guide

| Scenario | Pattern | Rationale |
|----------|---------|-----------|
| Adding a new domain capability (e.g., a new billing sub-feature) | Layered Domain Service/Adapter/Repository | Define a new sub-interface in <domain>/service.go, implement in <domain>/service/, add adapter methods to <domain>/adapter.go, implement in <domain>/adapter/. Wire together in app/common/<domain>.go. This keeps business logic, persistence, and HTTP separate and independently testable. |
| Triggering side-effects on domain lifecycle (e.g., sync billing on subscription update) | ServiceHook Registry or SubscriptionCommandHook | Avoids circular imports. Billing registers a hook into subscription.Service.RegisterHook() during wiring in app/common, not at compile time. The per-registry context key prevents re-entrant invocations. |
| Pre-validating a customer mutation from another domain | Customer RequestValidator Registry | billing/validators/customer implements RequestValidator and registers via customerService.RegisterRequestValidator() in app/common/customer.go, keeping billing constraints out of the customer package and preventing import cycles. |
| Processing async domain events between binaries | Watermill Message Bus (NoPublishingHandler + GroupEventHandler) | Events published to Kafka via eventbus.Publisher are consumed by worker processes registering typed closures. Unknown event types are silently dropped, making workers tolerant of schema evolution. Topic isolation matches worker topology. |
| Invoice or charge lifecycle transitions | Invoice State Machine (stateless library) or generic Machine[CHARGE,BASE,STATUS] | The state machine enforces valid transitions and fires actions (DB save, event publish, external app calls) atomically. sync.Pool reduces GC pressure on the hot billing-worker path. Generic Machine[CHARGE,BASE,STATUS] is reused for all charge types. |
| Multi-step workflow with crash recovery (e.g., subscription-to-invoice sync) | Subscription Sync Reconciler (idempotent periodic scan) | Event loss is mitigated by a periodic scan of all active subscriptions. SynchronizeSubscriptionAndInvoiceCustomer is idempotent so duplicate calls are safe. |
| New billing backend (payment processor or invoicing system) | App Factory / Registry + InvoicingApp interface | New backends implement billing.InvoicingApp and self-register a factory via app.Service.RegisterMarketplaceListing in their New() constructor. No billing service code changes needed. |
| Disabling a subsystem (credits off, no Svix) | Noop implementations for optional features | Wire provider functions check config flags and return noops. The rest of the DI graph is unaffected, avoiding nil checks scattered through business logic. Compile-time assertions keep noops in sync with interfaces. |
| Per-customer serialization for billing operations | Locker (pg_advisory_xact_lock) | Advisory locks are transactional and released automatically on commit/rollback, preventing stale locks after crashes. Requires an active Postgres transaction in context — call inside entutils.TransactingRepo. |
| HTTP error response to client | RFC 7807 Problem Details + GenericErrorEncoder chain | Domain errors (GenericNotFoundError → 404, GenericValidationError → 400, etc.) are matched by type in GenericErrorEncoder. ValidationIssues with explicit HTTP status attributes are handled by HandleIssueIfHTTPStatusKnown. All errors render as application/problem+json. |
| Batch usage event ingestion from Kafka | Sink Worker (Kafka → ClickHouse batch flush) | High-throughput events flow Kafka → SinkBuffer → ClickHouse in micro-batches. Strict three-phase flush ordering (ClickHouse → offset commit → Redis dedupe) ensures exactly-once semantics. Redis deduplication prevents double-counting on consumer restarts. |
| Outbound webhook notifications | Svix integration via webhook.Handler interface | Svix handles fan-out, retry, signature verification, and delivery status. The noop implementation runs in tests or when Svix is unconfigured. Notification consumer dispatches asynchronously from Watermill Kafka consumer. |
| Cross-domain DB operations in transactions | entutils.TransactingRepo / TransactingRepoWithNoValue | Ent transactions propagate implicitly via context. TransactingRepo reads the *TxDriver from ctx — if found it rebinds to the existing transaction; if not found it starts its own. Savepoints enable safe nesting. |
| HTTP endpoint handler | httptransport.NewHandler with RequestDecoder + Operation + ResponseEncoder | Consistent request validation, error encoding, and OTel tracing across all endpoints without duplicating boilerplate. ErrorEncoder chain maps domain errors to HTTP statuses. Chain method adds middleware. |
| Domain validation errors with field paths for API clients | ValidationIssue builder pattern (immutable with-chains) | ValidationIssue carries field paths (WithPathString), component attribution (WithComponent), severity (WithSeverity), and HTTP status (commonhttp.WithHTTPStatusCodeAttribute). AsValidationIssues traverses error trees. HandleIssueIfHTTPStatusKnown renders the correct HTTP status. |
| New binary that needs all domain services | Google Wire DI with app/common provider sets | Wire generates compile-time-verified dependency graphs. Adding a new binary requires a matching app/common/openmeter_<binary>.go provider set file and a wire.go in cmd/<binary>/. |
| Adding a new charge type engine for billing line computation | LineEngine Plugin Registry + RegisterLineEngine in app/common | Each charge type implements LineEngine and registers via billing.Service.RegisterLineEngine in app/common/charges.go. No billing service core changes needed. Engines must be registered before the first invoice advance. |

## Quick Pattern Lookup

- **new domain feature** -> Layered Domain Service/Adapter/Repository in openmeter/<domain>/  *(scope: openmeter/billing, openmeter/customer, openmeter/entitlement, openmeter/subscription)*
- **lifecycle side-effects** -> ServiceHookRegistry (models.ServiceHook[T]) or SubscriptionCommandHook  *(scope: openmeter/customer, openmeter/subscription, app/common)*
- **pre-mutation validation across domains** -> Customer RequestValidator Registry  *(scope: openmeter/customer, app/common)*
- **async domain events between binaries** -> Watermill NoPublishingHandler + GroupEventHandler on SystemEventsTopic  *(scope: openmeter/watermill, openmeter/billing/worker, openmeter/entitlement/balanceworker)*
- **invoice/charge state transitions** -> stateless-backed InvoiceStateMachine or generic Machine[CHARGE,BASE,STATUS]  *(scope: openmeter/billing, openmeter/billing/charges)*
- **crash-recovery for event-driven sync** -> Subscription Sync Reconciler (idempotent periodic scan)  *(scope: openmeter/billing/worker)*
- **new billing backend** -> Implement billing.InvoicingApp + AppFactory, register via app.Service.RegisterMarketplaceListing  *(scope: openmeter/app, openmeter/billing)*
- **optional feature disabled** -> Return noop implementation in Wire provider function when config flag is false  *(scope: app/common)*
- **per-customer serialization** -> billing.Service.WithLock -> lockr.Locker.LockForTX (pg advisory lock in tx)  *(scope: openmeter/billing, openmeter/billing/charges)*
- **DB operations in transactions** -> entutils.TransactingRepo / TransactingRepoWithNoValue  *(scope: openmeter/billing/adapter, openmeter/billing/charges/adapter, openmeter/customer/adapter)*
- **HTTP handler** -> httptransport.NewHandler with RequestDecoder + Operation + ResponseEncoder  *(scope: openmeter/billing/httpdriver, api/v3/handlers)*
- **batch usage ingestion** -> confluent-kafka-go consumer in Sink worker, ClickHouseStorage.BatchInsert  *(scope: openmeter/sink, cmd/sink-worker)*
- **outbound webhooks** -> notification.EventHandler -> webhook.Handler (Svix or noop)  *(scope: openmeter/notification, cmd/notification-service)*
- **DI wiring** -> Google Wire: wire.NewSet in app/common/, wire.Build in cmd/<binary>/wire.go  *(scope: app/common, cmd/server, cmd/billing-worker)*
- **structured validation errors** -> models.ValidationIssue with WithPathString + commonhttp.WithHTTPStatusCodeAttribute  *(scope: openmeter/billing, openmeter/billing/charges)*
- **domain error HTTP mapping** -> models.GenericXxxError wrapped in service/adapter, matched by commonhttp.GenericErrorEncoder
- **multi-language SDK contract** -> TypeSpec in api/spec/ -> make gen-api -> OpenAPI -> oapi-codegen + JS/Python generators  *(scope: api/spec)*

## Key Decisions

### TypeSpec as the single source of truth for both v1 and v3 HTTP APIs
**Chosen:** Author the HTTP API in TypeSpec (api/spec/packages/legacy for v1, api/spec/packages/aip for v3); compile via `make gen-api` to api/openapi.yaml + api/v3/openapi.yaml; then run oapi-codegen to produce api/api.gen.go, api/v3/api.gen.go, api/client/go/client.gen.go, api/client/javascript/, and Python SDK. `make generate` then propagates Go-side changes through Ent, Wire, Goverter, and Goderive.
**Rationale:** Drift between Go server stubs, three SDKs (Go/JS/Python), and two API versions is impossible as long as both regen steps run. Generated stubs are consumed by openmeter/server/router.Config and api/v3/handlers respectively, so TypeSpec changes force handler-side updates at compile time.
**Rejected:** Hand-written OpenAPI YAML, Code-first OpenAPI generated from Go handlers, gRPC/Protobuf, Single API version (skipping v3 AIP)
**Forced by:** Multi-language SDK requirement (Go, JS, Python) plus dual API versions plus runtime request validation against the same artifact.
**Enables:** ['Cross-language SDK contracts that cannot drift', 'Breaking-change detection at TypeSpec compile time', 'kin-openapi v1 + oasmiddleware v3 request validation', 'Parallel SDK evolution']

### Multi-binary deployment sharing a single domain package tree, wired via Google Wire in app/common
**Chosen:** Seven cmd/* entry points each call their own Wire-generated initializeApplication. Domain packages under openmeter/ have no dependency on cmd/* or app/common. Each binary's wire.go composes only the provider sets it needs. Hooks and validators are registered as side-effects inside Wire provider functions in app/common to break circular imports.
**Rationale:** Ingest throughput, balance recalculation, billing advancement, and notification dispatch have different scaling and failure profiles. Splitting into independent binaries while sharing types preserves the typed billing model. Wire produces compile-time-checked dependency graphs.
**Rejected:** Single monolith binary with goroutine workers, Independent microservices in separate repos, Manual constructor calls in each cmd/main.go, Reflection-based runtime DI, Domain packages registering their own hooks
**Forced by:** High-volume ingest plus strict billing correctness combined with very different scaling profiles per workload, plus need for cross-domain hooks without circular imports.
**Enables:** ['Independent horizontal scaling of sink-worker / balance-worker / billing-worker', 'Fault isolation per binary', 'Compile-time proof of completeness via Wire', 'Clean leaf-node domain packages']

### Kafka + Watermill as the async event backbone with three name-prefix-routed topics
**Chosen:** openmeter/watermill/eventbus wraps Watermill's cqrs.EventBus with TopicMapping (IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic). GeneratePublishTopic routes by EventName() prefix. Consumers build routers via openmeter/watermill/router.NewDefaultRouter (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout+RestoreContext, HandlerMetrics). Unknown event types silently dropped.
**Rationale:** Topic isolation matches worker topology: ingest bursts don't starve billing consumers. Prefix routing encapsulates topology from producers. Silent drop enables rolling deploys.
**Rejected:** Per-event explicit topic names, NATS or Redis Streams (lower replay/durability), Postgres LISTEN/NOTIFY, Pure confluent-kafka-go without Watermill (loses uniform middleware), Erroring on unknown event types (poisons DLQ during rolling deploys)
**Forced by:** Ingest bursts plus cross-worker side-effects plus need to deploy producers and consumers independently.
**Enables:** ['Replay, backpressure, decoupled producer/consumer evolution', 'Rolling deploy safety', 'Per-topic consumer scaling and DLQ semantics', 'Uniform OTel + correlation-id middleware']

### Ent ORM + Atlas-generated migrations with ctx-propagated transactions via entutils.TransactingRepo
**Chosen:** openmeter/ent/schema/ holds Go-defined entity schemas. `make generate` regenerates openmeter/ent/db/. `atlas migrate --env local diff <name>` produces timestamped .up.sql/.down.sql files plus an atlas.sum chain hash. Adapters implement TxCreator + TxUser triad and wrap every method body in entutils.TransactingRepo so the ctx-bound Ent transaction is honored.
**Rationale:** Atlas diffs Ent schema against migration history to produce deterministic SQL; Ent gives compile-time-checked relations across ~60 entities. TransactingRepo lets adapter helpers participate in caller-supplied transactions without leaking *entdb.Tx parameters. Helpers operating on the raw client fall off the transaction and produce partial writes - the most common correctness pitfall.
**Rejected:** Raw golang-migrate only (no typed Go entities), GORM (weaker typing, no native schema diff), sqlc (schema still hand-rolled SQL), Explicit *entdb.Tx parameters threaded through every call site, Global transaction middleware
**Forced by:** Billing correctness plus multi-tenant schema invariants requiring compile-time-checked relations across ~60 domain entities, plus need for ctx-propagated transaction reuse.
**Enables:** ['Deterministic reviewable SQL migrations with atlas.sum integrity', 'Typed relations across all entities', 'ctx-propagated transactions with savepoint nesting', 'Atomic charge advancement and invoice mutation']

### credits.enabled feature flag enforced at four independent wiring layers
**Chosen:** When config.Credits.Enabled=false: (1) app/common/ledger.go returns ledgernoop.* implementations from each provider; (2) NewCustomerLedgerServiceHook returns NoopCustomerLedgerHook; (3) NewBillingRegistry skips newChargesRegistry entirely; (4) v3 server credit handlers must skip registration. NewLedgerNamespaceHandler additionally type-asserts against ledgernoop.AccountResolver.
**Rationale:** Credits cross-cuts ledger writes, customer lifecycle hooks, namespace default-account provisioning, charge creation in billing/charges, and v3 HTTP handlers. There is no single choke point: a customer creation in api/v3 fans out via independent code paths.
**Rejected:** Single global runtime flag check inside ledger.Ledger, Top-level HTTP middleware blocking credits endpoints, Compile-time build tag
**Forced by:** Cross-cutting nature of credit accounting and customer/billing/ledger hook fan-out across unrelated call graphs.
**Enables:** ['Credits-disabled tenants produce zero ledger_accounts / ledger_customer_accounts rows', 'Per-deployment enabling without rebuild', 'Compile-time interface satisfaction for noop implementations']

### Charges as tagged-union domain model with explicit TransactingRepo discipline on every adapter helper
**Chosen:** openmeter/billing/charges owns the Charge tagged-union (NewCharge[T] discriminator) and Service interface (Create, AdvanceCharges, ApplyPatches). The adapter implements TxCreator + TxUser, and every method body wraps with entutils.TransactingRepo. Helpers that take a raw *entdb.Client must still wrap their bodies with TransactingRepo.
**Rationale:** Charge advancement mixes reads, realization runs, lockr advisory locks, and ledger-bound writes inside a single transaction carried via ctx. Without explicit rebinding, a helper that uses the raw client falls off the transaction and produces partial writes. Tagged-union Charge prevents partial construction.
**Rejected:** Pass *entdb.Tx explicitly through every call site, Global transaction middleware, Charge as Go interface (loses exhaustive type-dispatch)
**Forced by:** Ent transactions carried implicitly through ctx plus multi-step charge advancement that mixes reads, realization, locks, and ledger writes.
**Enables:** ['Deterministic atomic charge advancement', 'Safe nesting via Ent savepoints', 'Exhaustive charge-type dispatch', 'Per-charge advisory locking via charges.NewLockKeyForCharge']

### Dynamic build tag (-tags=dynamic) for librdkafka linking
**Chosen:** All binaries (Makefile GO_BUILD_FLAGS = -tags=dynamic) and all test invocations build with -tags=dynamic so confluent-kafka-go links against system librdkafka. CI runs through `nix develop --impure .#ci`.
**Rationale:** Dynamic linking matches production Docker image and cuts test/binary size. confluent-kafka-go's CGo dependency is enforced only at link time, not by Go modules.
**Rejected:** Static linking (large binaries, fragile builds), Pure-Go Kafka client (Sarama) at primary ingest path
**Forced by:** High-volume ingest workload plus Kafka tests in CI.
**Enables:** ['Consistent kafka behaviour across dev, CI, and production images', 'Performance parity between local and prod']

## Trade-offs Accepted

- **Accepted:** Ent-generated query friction (large openmeter/ent/db/ tree, slower compile, boilerplate Tx/WithTx/Self triad on every adapter)
  - *Benefit:* Compile-time-checked relations across ~60 entities, automatic Atlas diffing, no runtime schema surprises, ctx-propagated transactions with savepoint nesting
  - *Caused by:* Ent ORM + Atlas migration pipeline + entutils.TransactingRepo discipline
  - *Violation signal:* Hand-written SQL added alongside Ent queries
  - *Violation signal:* Direct edits inside openmeter/ent/db/
  - *Violation signal:* New table created without corresponding openmeter/ent/schema/*.go
  - *Violation signal:* Adapter struct that stores *entdb.Tx instead of using TransactingRepo
  - *Violation signal:* Helper that takes *entdb.Client and skips TransactingRepoWithNoValue
- **Accepted:** Multi-binary orchestration cost (seven Docker image variants, Helm values complexity, separate Wire graphs per binary)
  - *Benefit:* Independent horizontal scaling of sink-worker / balance-worker / billing-worker, fault isolation, isolated deploy cadence
  - *Caused by:* Multi-binary deployment of cmd/server, cmd/billing-worker, cmd/balance-worker, cmd/sink-worker, cmd/notification-service
  - *Violation signal:* Business logic added inside cmd/*/main.go beyond startup orchestration
  - *Violation signal:* Workers added without matching app/common/openmeter_*worker.go Wire set
  - *Violation signal:* Cross-binary dependencies introduced through shared global state instead of Kafka topic
  - *Violation signal:* Goroutine spawned outside run.Group
- **Accepted:** Two-step regen cadence (TypeSpec changes require `make gen-api` AND `make generate`; multiple generators write different artifacts)
  - *Benefit:* Cross-language SDK contracts cannot drift: Go server stubs, Go SDK, JS SDK, Python SDK all originate from a single TypeSpec source
  - *Caused by:* TypeSpec -> OpenAPI -> oapi-codegen + Wire/Ent/Goverter/Goderive stack
  - *Violation signal:* Hand-edits in *.gen.go files
  - *Violation signal:* PRs touching api/spec/ without regenerated openapi.yaml
  - *Violation signal:* Client SDKs under api/client/** drifting from api/spec/
  - *Violation signal:* TypeSpec edits without rerunning make generate
- **Accepted:** librdkafka C dependency (every test and binary invocation must use -tags=dynamic; CI image must ship librdkafka)
  - *Benefit:* High-throughput Kafka producer/consumer with consistent semantics across dev, CI, and production
  - *Caused by:* confluent-kafka-go + GO_BUILD_FLAGS=-tags=dynamic
  - *Violation signal:* go test invocations without -tags=dynamic (link errors)
  - *Violation signal:* CI images missing librdkafka
  - *Violation signal:* Attempts to switch primary ingest path to a pure-Go Kafka client
  - *Violation signal:* PRs that add a new test target without inheriting Makefile -tags=dynamic
- **Accepted:** Sequential Atlas migration filenames + atlas.sum hash chain (deterministic but produces guaranteed merge conflicts on long-lived branches)
  - *Benefit:* Reviewable SQL migrations with chain integrity verified by atlas.sum
  - *Caused by:* atlas migrate --env local diff filename convention + atlas.sum chain hashing
  - *Violation signal:* Multiple branches producing same-timestamp migrations
  - *Violation signal:* atlas.sum merge conflicts on long-running branches
  - *Violation signal:* Attempts to edit existing migrations after they land
  - *Violation signal:* Commits that touch tools/migrate/migrations/ without an accompanying atlas.sum update

## Out of Scope

- Frontend UI (no React/Vue in repo; React only appears in the generated JavaScript SDK under api/client/javascript/)
- Business-level auth/identity provider (portal tokens scope end-customers; tenant-level identity is delegated)
- Managed hosting control plane (config.cloud.yaml + api/openapi.cloud.yaml expose hooks but cloud orchestration logic lives separately)
- Real-time streaming queries from clients (ClickHouse is reached only via streaming.Connector inside the server process)
- Multi-region active/active replication (single PostgreSQL primary; ClickHouse cluster topology is deployment-defined)