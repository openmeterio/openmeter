## Communication Patterns

### Layered Domain Service/Adapter/Repository
- **When:** All business-logic domains under openmeter/<domain>/. Applied whenever persistence must be separated from orchestration.
- **How:** Each domain exposes a Go interface (e.g. billing.Service, customer.Service) defined in <domain>/service.go. A concrete service struct in <domain>/service/ holds business logic and calls an Adapter interface for all DB access. The Adapter interface is defined alongside the Service interface and implemented by Ent-backed structs in <domain>/adapter/ sub-packages. Service interfaces compose fine-grained sub-interfaces (e.g. ProfileService, InvoiceService) so callers depend only on the smallest surface they need. HTTP handlers live in <domain>/httpdriver/ or <domain>/httphandler/.

### Google Wire Dependency Injection
- **When:** Assembling runtime components for each binary. Each cmd/<binary>/wire.go declares a wire.Build call; provider sets live in app/common/.
- **How:** Google Wire generates cmd/<binary>/wire_gen.go at build time. Reusable provider sets (e.g. common.Billing, common.Notification, common.LedgerStack) are declared as wire.NewSet() in app/common/ and compose individual factory functions. Wire resolves the dependency graph automatically. Each Application struct in cmd/<binary>/wire.go lists every needed service as a field and Wire auto-wires them. Hook and validator registration is done as side-effects inside provider functions in app/common to avoid circular imports.

### Registry structs for multi-service domains (Wire Registry pattern)
- **When:** When a domain exposes multiple related services that callers must access together, reducing Wire graph complexity.
- **How:** A <Domain>Registry struct groups logically cohesive services (e.g. BillingRegistry, AppRegistry, ChargesRegistry). Callers depend on the registry rather than individual services. Nil-safe accessor methods (e.g. BillingRegistry.ChargesServiceOrNil()) encapsulate optional sub-registries.

### Noop Implementations for Optional Features
- **When:** When a feature is disabled at runtime (credits.enabled=false, Svix not configured, etc.), to avoid nil-pointer checks scattered through business logic.
- **How:** app/common provider functions check config flags and return noop structs instead of real implementations. All noop types implement the relevant interface via compile-time assertions. Callers receive a real interface and never branch on nil. Type assertions against noop types are used in some cases to conditionally skip handler registration.

### App Factory / Registry (external billing app protocol)
- **When:** Plugging Stripe, Sandbox, and CustomInvoicing billing apps into the billing state machine without hardcoding them.
- **How:** openmeter/app/registry.go defines AppFactory and RegistryItem (Listing + Factory). app.Service.RegisterMarketplaceListing is called from each app type's New() constructor (self-registration). Installed apps implement billing.InvoicingApp interface. Optional InvoicingAppPostAdvanceHook and InvoicingAppAsyncSyncer extend it. GetApp() type-asserts an installed App to InvoicingApp at runtime.

### LineEngine Plugin Registry
- **When:** Dispatching billing line calculation to the correct engine (standard invoice, charge flatfee, charge usagebased, charge creditpurchase) based on LineEngineType discriminator.
- **How:** billing.Service exposes RegisterLineEngine / DeregisterLineEngine. billingservice.engineRegistry stores a map[LineEngineType]LineEngine under a RWMutex. Each charge type implements its own Engine and registers it at startup via app/common/charges.go. The service.New() constructor also pre-registers the standard invoice line engine.

### ServiceHook Registry
- **When:** Cross-domain lifecycle callbacks without circular imports. Used by customer.Service, subscription.Service, and billing.Service.
- **How:** pkg/models/servicehook.go defines a generic ServiceHook[T] interface (PreUpdate, PreDelete, PostCreate, PostUpdate, PostDelete) and thread-safe ServiceHookRegistry[T] that fans out to all registered hooks. Loop prevention: a per-registry context key (pointer-identity string via fmt.Sprintf('%p', r)) prevents re-entrant invocations. Domain services embed *ServiceHookRegistry and expose RegisterHooks externally so other packages register callbacks without import cycles.

### Customer RequestValidator Registry
- **When:** Pre-mutation validation for customer operations where billing, subscription, or entitlement constraints must be checked before the customer is modified or deleted.
- **How:** openmeter/customer/requestvalidator.go defines RequestValidator interface (ValidateDeleteCustomer, ValidateCreateCustomer, ValidateUpdateCustomer) and a thread-safe requestValidatorRegistry that fans out to all registered validators using errors.Join. Validators are registered in app/common/customer.go to avoid circular imports.

### Invoice State Machine (stateless library)
- **When:** Driving the invoice lifecycle transitions for StandardInvoice.
- **How:** openmeter/billing/service/stdinvoicestate.go builds a *stateless.StateMachine from sync.Pool with external storage bound to the InvoiceStateMachine struct's Invoice.Status field. Transitions trigger actions (DB save, event publish). FireAndActivate fires a trigger and persists; AdvanceUntilStateStable runs all allowed auto-transitions.

### Generic Charge State Machine (Machine[CHARGE, BASE, STATUS])
- **When:** Driving charge lifecycle (flatfee, usagebased, creditpurchase) with shared mechanics (fire, activate, persist-base, refetch).
- **How:** openmeter/billing/charges/statemachine/machine.go defines generic Machine[CHARGE ChargeLike[CHARGE,BASE,STATUS], BASE any, STATUS Status]. External storage binds state to in-memory CHARGE. FireAndActivate fires a trigger and persists BASE; AdvanceUntilStateStable walks TriggerNext transitions. Each charge type instantiates Machine with concrete types.

### Watermill Message Bus (Kafka-backed publish/subscribe)
- **When:** Async domain-event delivery between separate binaries. Used for subscription lifecycle events, billing invoice advance events, ingest flush events, and balance-worker recalculation events.
- **How:** openmeter/watermill/eventbus/eventbus.go wraps Watermill's cqrs.EventBus. Topic routing is by event-name prefix: events whose EventName() starts with ingestevents.EventVersionSubsystem go to IngestEventsTopic; balanceworkerevents.EventVersionSubsystem go to BalanceWorkerEventsTopic; everything else to SystemEventsTopic. Workers subscribe via Watermill's Kafka subscriber and dispatch to typed handlers registered in grouphandler.NewNoPublishingHandler. Unknown event types are silently dropped.

### Namespace multi-tenancy via Manager + Handler fan-out
- **When:** Provisioning and deprovisioning tenants across all subsystems (ClickHouse, Kafka ingest, Ledger).
- **How:** openmeter/namespace/namespace.go defines Manager which fans out CreateNamespace/DeleteNamespace to all registered Handler implementations. Handlers are registered via RegisterHandler before CreateDefaultNamespace is called at startup. Fan-out uses errors.Join (no short-circuit on partial failure). The default namespace is protected from deletion.

### entutils.TransactingRepo (context-propagated transactions)
- **When:** All DB adapter methods that must run inside a caller-supplied transaction or start their own.
- **How:** pkg/framework/entutils/transaction.go defines TransactingRepo[R,T] and TransactingRepoWithNoValue[T]. They read the *TxDriver from context via GetDriverFromContext. If found, the adapter's WithTx(ctx, tx) creates a txClient from raw Ent config. If none is found, the adapter runs on Self() and starts its own transaction. Savepoints enable nested calls with partial rollback.

### Locker (pg_advisory_xact_lock)
- **When:** Distributed mutual exclusion for per-customer billing operations to prevent concurrent invoice generation races.
- **How:** pkg/framework/lockr/locker.go calls pg_advisory_xact_lock($1) with a CRC64-based hash of the lock key. Requires an active Postgres transaction in context (from GetDriverFromContext). Lock is released automatically on tx commit/rollback.

### httptransport Operation/Handler pattern
- **When:** HTTP endpoint handlers in domain httpdriver packages that separate request decoding, business logic, and response encoding.
- **How:** pkg/framework/transport/httptransport/handler.go defines Handler[Request, Response] wrapping an operation.Operation[Request, Response]. Decoding and encoding are injected via RequestDecoder and ResponseEncoder function types. ErrorEncoders form a chain; the first returning true short-circuits. Chain method wraps the operation with middleware.

### Sink Worker (Kafka to ClickHouse batch flush)
- **When:** High-throughput ingestion path: raw CloudEvents arrive via Kafka, are buffered, deduplicated, and batch-inserted into ClickHouse.
- **How:** openmeter/sink/sink.go consumes Kafka partitions via confluent-kafka-go. A SinkBuffer accumulates messages; flush is triggered by MinCommitCount or MaxCommitWait. Flush ordering is strict: ClickHouse insert then Kafka offset commit then Redis dedupe. After flush, FlushEventHandlers are called post-flush in a goroutine with timeout for downstream notifications.

### Subscription Sync Reconciler
- **When:** Crash-recovery for the event-driven billing sync: periodically re-syncs subscriptions that may have missed their events.
- **How:** openmeter/billing/worker/subscriptionsync/reconciler/reconciler.go iterates customers/subscriptions in windows and calls subscriptionsync.Service.SynchronizeSubscriptionAndInvoiceCustomer for each. The reconciliation is idempotent so duplicate calls are safe.

### ValidationIssue structured error propagation
- **When:** Domain-level validation errors that must carry field paths, severity (critical/warning), component names, and arbitrary attributes through service layer boundaries.
- **How:** pkg/models/validationissue.go defines ValidationIssue as an immutable value type with copy-on-write With* methods. Constructed via NewValidationIssue(code, message, opts...) or NewValidationError/NewValidationWarning. The HTTP layer reads httpStatusCodeErrorAttribute attribute (set via commonhttp.WithHTTPStatusCodeAttribute) to produce the correct HTTP status. AsValidationIssues traverses an error tree to extract all ValidationIssue nodes.

### RFC 7807 Problem Details HTTP response
- **When:** All error responses from the REST API.
- **How:** pkg/models/problem.go defines Problem interface and StatusProblem struct serialized to application/problem+json. NewStatusProblem reads the request-id from Chi middleware context, maps 'context canceled' to 408, suppresses detail on 500. Extensions map carries validationErrors array when applicable. GenericErrorEncoder chains multiple typed error matchers.

### Generic Typed Domain Errors (models.Generic* sentinels)
- **When:** Returning domain errors from service and adapter methods so the HTTP error encoder chain maps them to correct status codes.
- **How:** pkg/models/errors.go defines typed sentinel error structs: GenericNotFoundError, GenericConflictError, GenericValidationError, GenericForbiddenError, GenericUnauthorizedError, GenericNotImplementedError, GenericPreConditionFailedError, GenericStatusFailedDependencyError. Each has New* constructor, Unwrap(), and Is* predicate. HTTP layer maps these via HandleErrorIfTypeMatches[T] in commonhttp.GenericErrorEncoder.

### credits.enabled multi-layer feature flag
- **When:** Disabling the credits/ledger subsystem when credits.enabled=false, ensuring no ledger writes occur from any path.
- **How:** credits.enabled must be honored at four independent wiring layers: (1) app/common wires ledger services to noop; (2) api/v3/server credit handlers must skip registration; (3) customer ledger hooks must be unregistered; (4) namespace default-account provisioning must skip ledger account creation. A single guard is insufficient because credits cross-cuts multiple unrelated call graphs.

## Integrations

| Service | Purpose | Integration point |
|---------|---------|-------------------|
| PostgreSQL (Ent ORM + pgx + Atlas) | Primary relational store for all domain entities: billing profiles, invoices, customers, subscriptions, entitlements, notification channels, ledger accounts, etc. | `openmeter/ent/schema/ (source of truth); generated code in openmeter/ent/db/; Atlas migrations in tools/migrate/migrations/. Accessed via entdb.Client injected through Wire. Transactions managed by pkg/framework/entutils.TransactingRepo.` |
| ClickHouse | Append-only analytics store for raw usage events; queried for meter aggregations (count, sum, max, unique_count) via SQL builders. | `openmeter/streaming/clickhouse/ — event_query.go and meter_query.go build ClickHouse SQL via sqlbuilder. Connector interface in openmeter/streaming/connector.go. ClickHouseStorage in openmeter/sink/storage.go for batch inserts.` |
| Kafka (confluent-kafka-go + Watermill-Kafka) | Durable event bus for domain events (subscription lifecycle, invoice advance, ingest flush notifications, balance recalculation) and raw usage event ingestion. | `openmeter/watermill/driver/kafka/ — Publisher and Subscriber wrappers. Topic provisioning via KafkaTopicProvisioner in app/common. confluent-kafka-go used directly in openmeter/sink/sink.go for high-throughput ingest consumer.` |
| Redis | Optional deduplication store for ingest events (preventing double-counting on retry). | `openmeter/dedupe/redisdedupe/redisdedupe.go — Redis-backed Deduplicator. In-memory LRU fallback in openmeter/dedupe/memorydedupe/.` |
| Svix | Outbound webhook delivery for notification events (entitlement balance thresholds, invoice events). | `openmeter/notification/webhook/svix/svix.go — Svix API client wrapper. Registered event types passed to Svix application. Handler interface in openmeter/notification/webhook/handler.go with noop fallback when Svix is unconfigured.` |
| Stripe (via app/stripe) | Invoice syncing (upsert draft, finalize, collect payment) and customer sync for billing-enabled namespaces. | `openmeter/app/stripe/app.go implements billing.InvoicingApp (UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice). Stripe REST client in openmeter/app/stripe/client/. App self-registers via app.Service.RegisterMarketplaceListing in service/factory.go.` |
| Sandbox invoicing app | No-op invoicing app used in development/testing to drive invoice state machine without external dependencies. | `openmeter/app/sandbox/app.go implements billing.InvoicingApp + InvoicingAppPostAdvanceHook. Registered as marketplace listing with type AppTypeSandbox.` |
| CustomInvoicing app | Webhook-driven invoicing app allowing external systems to receive invoice payloads and async-confirm sync completion. | `openmeter/app/custominvoicing/ — App implements InvoicingApp + InvoicingAppAsyncSyncer; factory in custominvoicing/factory.go.` |
| GOBL (invopop/gobl) | Currency and numeric type library for currency-safe arithmetic and ISO 4217 currency code validation throughout billing and subscription. | `Imported as github.com/invopop/gobl/currency and github.com/invopop/gobl/num in productcatalog, subscription, billing, cost, and currencies packages.` |
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

## Quick Pattern Lookup

- **new domain feature** -> Layered Domain Service/Adapter/Repository in openmeter/<domain>/
- **lifecycle side-effects** -> ServiceHookRegistry (models.ServiceHook[T]) or SubscriptionCommandHook
- **pre-mutation validation across domains** -> Customer RequestValidator Registry
- **async domain events between binaries** -> Watermill NoPublishingHandler + GroupEventHandler on SystemEventsTopic
- **invoice/charge state transitions** -> stateless-backed InvoiceStateMachine or generic Machine[CHARGE,BASE,STATUS]
- **crash-recovery for event-driven sync** -> Subscription Sync Reconciler (idempotent periodic scan)
- **new billing backend** -> Implement billing.InvoicingApp + AppFactory, register via app.Service.RegisterMarketplaceListing
- **optional feature disabled** -> Return noop implementation in Wire provider function when config flag is false
- **per-customer serialization** -> billing.Service.WithLock → lockr.Locker.LockForTX (pg advisory lock in tx)
- **DB operations in transactions** -> entutils.TransactingRepo / TransactingRepoWithNoValue
- **HTTP handler** -> httptransport.NewHandler with RequestDecoder + Operation + ResponseEncoder
- **batch usage ingestion** -> confluent-kafka-go consumer in Sink worker, ClickHouseStorage.BatchInsert
- **outbound webhooks** -> notification.EventHandler → webhook.Handler (Svix or noop)
- **DI wiring** -> Google Wire: wire.NewSet in app/common/, wire.Build in cmd/<binary>/wire.go
- **structured validation errors** -> models.ValidationIssue with WithPathString + commonhttp.WithHTTPStatusCodeAttribute
- **domain error HTTP mapping** -> models.GenericXxxError wrapped in service/adapter, matched by commonhttp.GenericErrorEncoder
- **multi-language SDK contract** -> TypeSpec in api/spec/ → make gen-api → OpenAPI → oapi-codegen + JS/Python generators

## Decision Chain

**Root constraint:** OpenMeter must provide high-volume per-tenant usage metering feeding strict billing correctness, with stable multi-language SDKs

- **Multi-binary deployment with shared domain packages**: Ingest, balance recalc, billing advancement, and notification dispatch have different throughput/latency envelopes but share the billing correctness model — independent binaries provide fault isolation and scaling without fracturing the typed domain
  - *Violation keyword:* `cmd/server owns worker`
  - *Violation keyword:* `business logic in cmd/*/main.go`
  - *Violation keyword:* `shared in-memory state across binaries`
  - *Violation keyword:* `goroutine spawned outside run.Group`
  - **Kafka + Watermill as the async event backbone**: Multi-binary needs durable message passing with replay and backpressure to connect ingest, workers, and notifications without in-memory coupling
    - *Violation keyword:* `HTTP call between workers`
    - *Violation keyword:* `direct DB poll in consumer`
    - *Violation keyword:* `shared in-memory queue`
    - *Violation keyword:* `raw confluent-kafka-go consumer outside openmeter/sink`
    - **Three named Kafka topics routed by event-name prefix in openmeter/watermill/eventbus**: Isolating ingest from billing system events from balance worker events prevents one hot producer from starving another consumer group; prefix routing inside eventbus encapsulates topic names from producers
      - *Violation keyword:* `publishing directly to a Kafka topic string`
      - *Violation keyword:* `EventName without EventVersionSubsystem prefix`
      - *Violation keyword:* `fourth Kafka topic added without updating TopicMapping`
      - *Violation keyword:* `producer importing balanceworkerevents from a non-balance-worker`
    - **Silent-drop of unknown event types in NoPublishingHandler**: Allows independent rolling deploys of producers and consumers; an outdated consumer ignores newer event types instead of poisoning its DLQ
      - *Violation keyword:* `return error for unknown event type`
      - *Violation keyword:* `panic on unrecognised ce_type`
      - *Violation keyword:* `MaxRetries=0 misuse`
  - **Google Wire DI concentrated in app/common with one provider file per domain**: Each cmd/* binary needs its own provider graph but every domain service must be wireable identically; Wire makes this compile-time checked
    - *Violation keyword:* `manual constructor call chain in cmd/`
    - *Violation keyword:* `runtime DI container`
    - *Violation keyword:* `provider in domain package`
    - *Violation keyword:* `domain package importing app/common`
    - **Hooks and validators registered inside app/common provider functions, not in domain packages**: Cross-domain hooks (billing → customer, ledger → customer, billing → subscription) would create circular imports if registered in source packages
      - *Violation keyword:* `domain package calling RegisterHooks on another domain's service`
      - *Violation keyword:* `import cycle between billing and customer`
      - *Violation keyword:* `hook registration inside domain service constructor`
    - **Typed registry structs (BillingRegistry, AppRegistry, SubscriptionServiceWithWorkflow) instead of injecting individual services**: Groups logically cohesive services and lets ChargesServiceOrNil() encapsulate the credits-disabled nil case
      - *Violation keyword:* `depending on BillingRegistry.Charges directly without ChargesServiceOrNil`
      - *Violation keyword:* `router.Config field for a service already inside a registry`
- **TypeSpec as the API source of truth for v1 and v3**: Multi-language SDKs (Go/JS/Python) plus dual API versions (v1 legacy + v3 AIP) force contract-first to avoid drift
  - *Violation keyword:* `editing api/openapi.yaml`
  - *Violation keyword:* `editing api/api.gen.go`
  - *Violation keyword:* `editing api/v3/api.gen.go`
  - *Violation keyword:* `adding endpoint only in Go handler`
  - **Two-step regen: `make gen-api` then `make generate`**: OpenAPI is an intermediate; Go server stubs, Wire, Ent, Goverter, and Goderive are downstream generators
    - *Violation keyword:* `partial regen`
    - *Violation keyword:* `committing api/spec without openapi.yaml`
    - *Violation keyword:* `hand-edit of client.gen.go`
    - *Violation keyword:* `TypeSpec change merged without `make generate``
  - **Generic httptransport.Handler[Request, Response] adapter for every endpoint**: Generated server stubs must adapt to concrete domain Service interfaces; the generic adapter centralises decode/operate/encode plus error encoding chain
    - *Violation keyword:* `implementing http.Handler.ServeHTTP directly`
    - *Violation keyword:* `bypassing httptransport.NewHandler`
    - *Violation keyword:* `writing status codes in handlers`
    - *Violation keyword:* `duplicate request validation per endpoint`
- **Ent ORM + Atlas migrations as the schema pipeline**: Billing correctness needs typed relations across ~60 entities and reviewable SQL migration diffs
  - *Violation keyword:* `hand-written SQL alongside Ent`
  - *Violation keyword:* `edit of openmeter/ent/db/`
  - *Violation keyword:* `ad-hoc DDL outside tools/migrate/migrations/`
  - **Sequential timestamped migrations with atlas.sum chain**: Linear hash chain enforces reviewable order and prevents out-of-order migration application
    - *Violation keyword:* `atlas.sum merge conflict`
    - *Violation keyword:* `duplicate migration timestamp`
    - *Violation keyword:* `editing a landed migration`
    - *Violation keyword:* `manual hand-edit of tools/migrate/migrations/*.sql`
  - **entutils.TransactingRepo discipline on every adapter helper**: Ent transactions are carried implicitly through ctx; helpers using a raw *entdb.Client fall off the transaction and produce partial writes
    - *Violation keyword:* `helper accepting *entdb.Client without TransactingRepo`
    - *Violation keyword:* `adapter struct storing *entdb.Tx`
    - *Violation keyword:* `creator.Tx() called directly`
    - *Violation keyword:* `adapter method body that calls a.db.Foo() without TransactingRepo wrapper`
- **credits.enabled guarded at four independent wiring layers**: Credits cross-cuts ledger, customer hooks, v3 handlers, charges creation, and namespace provisioning — no single guard point exists
  - *Violation keyword:* `single global credits check`
  - *Violation keyword:* `ledger write from unguarded hook`
  - *Violation keyword:* `default account provisioned when credits disabled`
  - *Violation keyword:* `ChargesRegistry constructed when credits.enabled=false`
- **Dynamic build tag (-tags=dynamic) for librdkafka**: High-throughput ingest requires librdkafka; dynamic linking matches production Alpine image and CI Nix shell
  - *Violation keyword:* `go test without -tags=dynamic`
  - *Violation keyword:* `go build without -tags=dynamic`
  - *Violation keyword:* `static linking attempt for confluent-kafka-go`
  - *Violation keyword:* `primary ingest path switching to Sarama`

## Key Decisions

### TypeSpec as the single source of truth for both v1 and v3 HTTP APIs
**Chosen:** Author the HTTP API in TypeSpec (api/spec/packages/legacy for v1, api/spec/packages/aip for v3), compile via `make gen-api` to api/openapi.yaml + api/v3/openapi.yaml, then run oapi-codegen to produce api/api.gen.go (v1 server stubs), api/v3/api.gen.go (v3 server stubs), api/client/go/client.gen.go (Go SDK), api/client/javascript/ (JS SDK), and Python SDK. `make generate` then propagates Go-side changes through Ent, Wire, Goverter, and Goderive.
**Rationale:** Drift between Go server stubs, three SDKs (Go/JS/Python), and two API versions is impossible as long as both regen steps run. Generated stubs (api/api.gen.go, api/v3/api.gen.go) are consumed by openmeter/server/router.Config (assembled in cmd/server/main.go) and api/v3/handlers respectively, so any TypeSpec change forces handler-side updates at compile time.
**Rejected:** Hand-written OpenAPI YAML, Code-first OpenAPI generated from Go handlers, gRPC/Protobuf, Single API version (skipping v3 AIP-style API)
**Forced by:** Multi-language SDK requirement (Go, JS, Python) plus dual API versions (v1 legacy + v3 AIP-style)
**Enables:** Contract-stable breaking-change detection, kin-openapi request validation middleware, parallel SDK evolution without manual reconciliation

### Ent ORM + Atlas-generated migrations as the single schema pipeline
**Chosen:** openmeter/ent/schema/ holds Go-defined entity schemas as source of truth. `make generate` regenerates openmeter/ent/db/. `atlas migrate --env local diff <name>` produces timestamped .up.sql/.down.sql files plus an atlas.sum chain hash in tools/migrate/migrations/. Adapters (e.g. openmeter/billing/charges/adapter/adapter.go) implement TxCreator + TxUser via HijackTx + NewTxClientFromRawConfig and must wrap every method body in entutils.TransactingRepo so the ctx-bound Ent transaction is honored.
**Rationale:** Atlas diffs the Ent schema against the migration history to produce deterministic SQL; Ent gives compile-time-checked relations across ~60 entities. The TxCreator/TxUser triad implemented in pkg/framework/entutils/transaction.go (TransactingRepo reads *TxDriver from ctx, calls WithTx if found, Self otherwise; savepoint-aware for nested calls) lets adapter helpers participate in caller-supplied transactions without leaking *entdb.Tx parameters.
**Rejected:** Raw golang-migrate only (no typed Go entities), GORM (weaker typing, no native schema diff), sqlc (schema still hand-rolled SQL), Explicit *entdb.Tx parameters threaded through every call site
**Forced by:** Billing correctness + multi-tenant schema invariants requiring compile-time-checked relations across ~60 domain entities
**Enables:** Deterministic reviewable SQL migrations, typed relations, ctx-propagated transaction reuse with savepoint nesting

### Multi-binary deployment sharing a single domain package tree
**Chosen:** Seven cmd/* entry points (server, billing-worker, balance-worker, sink-worker, notification-service, jobs, benthos-collector) each call their own Wire-generated initializeApplication. Domain packages under openmeter/ have no dependency on cmd/* or app/common (enforced by leaf-node import direction documented in app/CLAUDE.md). Each binary's wire.go composes only the provider sets it needs (e.g. cmd/sink-worker uses common.WatermillNoPublisher; cmd/billing-worker uses common.BillingWorker which composes the full billing+charges+ledger stack).
**Rationale:** Ingest throughput, balance recalculation, billing advancement, and notification dispatch have different scaling and failure profiles. Splitting them into independent binaries while sharing types preserves the typed billing model. cmd/server/main.go orchestrates ~40 services through router.Config; cmd/billing-worker/main.go follows a much narrower post-Migrate provisioning sequence (EnsureBusinessAccounts, SandboxProvisioner, then app.Run).
**Rejected:** Single monolith binary with goroutine workers, Independent microservices in separate repos, Per-binary domain duplication
**Forced by:** High-volume ingest with strict billing correctness combined with very different scaling profiles per workload
**Enables:** Independent horizontal scaling of sink-worker, fault isolation per binary, isolated deploy cadence

### Google Wire DI concentrated in app/common with provider sets per domain and per binary
**Chosen:** Every domain package exposes plain constructors. All wiring lives in app/common/*.go (one file per domain area: billing.go, customer.go, ledger.go, etc., plus binary-specific openmeter_server.go, openmeter_billingworker.go, openmeter_sinkworker.go). Domain services are grouped into typed registry structs (BillingRegistry exposing ChargesServiceOrNil(), AppRegistry, SubscriptionServiceWithWorkflow). Hooks and validators are registered as side-effects inside Wire provider functions to avoid circular imports (app/common/customer.go: customerService.RegisterRequestValidator(validator); app/common/billing.go: subscriptionServices.Service.RegisterHook(subscriptionValidator)).
**Rationale:** Compile-time-checked dependency graph; changing a constructor signature surfaces missing providers at Wire regen time. Cross-domain hook/validator registration done inside app/common avoids circular imports (billing depends on customer; if customer needed to register billing's validator the cycle would be unresolvable). cmd/server/wire.go references ~30 provider sets producing the largest Application struct in the codebase (~40 service fields).
**Rejected:** Manual constructor calls in each cmd/main.go (duplicated graphs), Reflection-based runtime DI (incompatible with billing correctness), Domain packages registering their own hooks (circular imports)
**Forced by:** Seven binaries sharing dozens of providers + cross-domain lifecycle hooks that cannot be registered inside the source domain without circular imports
**Enables:** Single edit point to add a dependency to any binary; compile-time proof of completeness; clean leaf-node domain packages

### Kafka + Watermill as the asynchronous event backbone with three name-prefix-routed topics
**Chosen:** openmeter/watermill/eventbus wraps Watermill's cqrs.EventBus with a topic mapping (IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic). GeneratePublishTopic in eventbus.go inspects the EventName() prefix (ingestevents.EventVersionSubsystem+'.', balanceworkerevents.EventVersionSubsystem+'.', else SystemEventsTopic) so producers never name a topic explicitly. Consumers (billing-worker, balance-worker, notification-service, sink-worker) build routers via openmeter/watermill/router.NewDefaultRouter (fixed middleware: PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout+RestoreContext, HandlerMetrics) and dispatch via grouphandler.NoPublishingHandler keyed on CloudEvents ce_type — unknown event types are silently dropped to support rolling deploys.
**Rationale:** Topic isolation matches worker topology — ingest bursts in IngestEventsTopic don't starve billing consumers on SystemEventsTopic. Prefix-based routing in eventbus encapsulates routing knowledge so producers stay unaware of topic names. Silent drop of unknown event types lets producers and consumers deploy in any order without poisoning DLQs.
**Rejected:** Per-event explicit topic names, NATS or Redis Streams (lower replay semantics), Postgres LISTEN/NOTIFY (wrong durability/backpressure model for ingest), Pure confluent-kafka-go without Watermill (loses uniform router middleware), Erroring on unknown event types (poisons DLQ during rolling deploys)
**Forced by:** Ingest bursts + cross-worker side-effects (ingest -> balance recalc -> notification) + need to deploy producers and consumers independently
**Enables:** Replay, backpressure, decoupled producer/consumer evolution, rolling deploy safety

### credits.enabled feature flag enforced at four independent wiring layers
**Chosen:** When config.Credits.Enabled=false, app/common/ledger.go returns ledgernoop.AccountService{}, ledgernoop.Ledger{}, ledgernoop.AccountResolver{}, and ledgernoop.NamespaceHandler{} from each Wire provider; app/common/customer.go's NewCustomerLedgerServiceHook returns ledgerresolvers.NoopCustomerLedgerHook{}; app/common/billing.go's NewBillingRegistry skips newChargesRegistry entirely (BillingRegistry.Charges stays nil and ChargesServiceOrNil() returns nil); v3 server credit handlers must skip registration. NewLedgerNamespaceHandler additionally type-asserts against ledgernoop.AccountResolver to skip namespace handler registration.
**Rationale:** Credits cross-cuts ledger writes, customer lifecycle hooks, namespace default-account provisioning, charge creation in billing/charges, and v3 HTTP handlers. There is no single choke point — a customer creation in api/v3 ultimately fans out to customer hooks and ledger writes via independent code paths. Centralising into one runtime check inside ledger.Ledger would still allow the writes to be attempted (performance + correctness risk).
**Rejected:** Single global runtime flag check inside ledger.Ledger, Top-level HTTP middleware blocking credits endpoints (does not stop hook-driven writes), Compile-time build tag (cannot toggle without rebuild)
**Forced by:** Cross-cutting nature of credit accounting and the customer/billing/ledger hook fan-out
**Enables:** Credits-disabled tenants genuinely produce zero ledger_accounts / ledger_customer_accounts rows when every layer is correctly guarded

### Charges realization with explicit TransactingRepo discipline on every adapter helper
**Chosen:** openmeter/billing/charges owns the Charge tagged-union (NewCharge[T] discriminator) and the Service interface (Create, AdvanceCharges, ApplyPatches, etc.). The adapter at openmeter/billing/charges/adapter/adapter.go implements TxCreator (Tx via HijackTx + NewTxDriver) and TxUser (WithTx via NewTxClientFromRawConfig, Self), and every method body must call entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter) ...). Helper methods that take a raw *entdb.Client must still wrap their bodies with TransactingRepo / TransactingRepoWithNoValue so they rebind to the ctx-bound transaction (AGENTS.md explicitly mandates this).
**Rationale:** Charge advancement (charges.Service.AdvanceCharges, ApplyPatches) mixes reads, realization runs, lockr advisory locks, and ledger-bound writes inside a single transaction carried via ctx. Without explicit rebinding, a helper that uses the raw client falls off the transaction and produces partial writes under concurrency. Tagged-union Charge prevents partial construction (struct-literal Charge{} leaves discriminator empty and accessors error).
**Rejected:** Pass *entdb.Tx explicitly through every call site (leaks tx plumbing), Global transaction middleware (cannot enforce per-helper without compiler help), Charge as Go interface (loses exhaustive type-dispatch via switch)
**Forced by:** Ent transactions carried implicitly through ctx + multi-step charge advancement that mixes reads, realization, locks, and ledger writes
**Enables:** Deterministic atomic charge advancement; safe nesting via Ent savepoints; exhaustive charge-type dispatch in service code

### Dynamic build tag for librdkafka (-tags=dynamic)
**Chosen:** All binaries (Makefile GO_BUILD_FLAGS = -tags=dynamic) and all test invocations build with -tags=dynamic so confluent-kafka-go links against system librdkafka. CI runs through nix develop --impure .#ci to pin the toolchain.
**Rationale:** Dynamic linking matches production Docker image (alpine-based with librdkafka) and cuts test/binary size, while keeping librdkafka throughput required for ingest workloads.
**Rejected:** Static linking (large binaries, fragile builds), Pure-Go Kafka client (Sarama) at primary ingest path (lower throughput)
**Forced by:** High-volume ingest workload + Kafka tests in CI
**Enables:** Consistent kafka behaviour across dev, CI, and production images

## Trade-offs Accepted

- **Accepted:** Ent-generated query friction — large openmeter/ent/db/ tree, slower compile when adding entities, boilerplate for every adapter (Tx/WithTx/Self triad)
  - *Benefit:* Compile-time-checked relations across ~60 entities, automatic Atlas diffing, no runtime schema surprises, ctx-propagated transactions with savepoint nesting
  - *Caused by:* Ent ORM + Atlas migration pipeline + entutils.TransactingRepo discipline
  - *Violation signal:* Hand-written SQL added alongside Ent queries
  - *Violation signal:* Direct edits inside openmeter/ent/db/
  - *Violation signal:* New table created without corresponding openmeter/ent/schema/*.go
  - *Violation signal:* Adapter struct that stores *entdb.Tx instead of using TransactingRepo
  - *Violation signal:* Helper that takes *entdb.Client and skips TransactingRepoWithNoValue
- **Accepted:** Multi-binary orchestration cost — seven Docker image variants, Helm values complexity, multi-service docker-compose, separate Wire graphs per binary
  - *Benefit:* Independent horizontal scaling of sink-worker / balance-worker / billing-worker; fault isolation per binary; isolated deploy cadence
  - *Caused by:* Multi-binary deployment of cmd/server, cmd/billing-worker, cmd/balance-worker, cmd/sink-worker, cmd/notification-service
  - *Violation signal:* Business logic added inside cmd/*/main.go beyond startup orchestration
  - *Violation signal:* Workers added without matching app/common/openmeter_*worker.go Wire set
  - *Violation signal:* Cross-binary dependencies introduced through shared global state instead of Kafka topic
  - *Violation signal:* Goroutine spawned outside run.Group (bypasses graceful shutdown)
- **Accepted:** Two-step regen cadence — TypeSpec changes require `make gen-api` AND `make generate`; multiple generators (TypeSpec, Ent, Wire, Goverter, Goderive) write different artifacts
  - *Benefit:* Cross-language SDK contracts cannot drift; Go server stubs, Go SDK, JS SDK, Python SDK all originate from a single TypeSpec source
  - *Caused by:* TypeSpec → OpenAPI → oapi-codegen + Wire/Ent/Goverter/Goderive stack
  - *Violation signal:* Hand-edits in *.gen.go files
  - *Violation signal:* PRs touching api/spec/ without regenerated api/openapi.yaml or api/v3/openapi.yaml
  - *Violation signal:* Client SDKs under api/client/** drifting from api/spec/
  - *Violation signal:* TypeSpec edits without rerunning `make generate` (Go types out of sync)
- **Accepted:** librdkafka C dependency — every test and binary invocation must use -tags=dynamic, CI image must ship librdkafka, dev shell needs nix .#ci
  - *Benefit:* High-throughput Kafka producer/consumer with consistent semantics across dev, CI, and production
  - *Caused by:* confluent-kafka-go + GO_BUILD_FLAGS=-tags=dynamic
  - *Violation signal:* go test invocations without -tags=dynamic (link errors)
  - *Violation signal:* CI images missing librdkafka
  - *Violation signal:* Attempts to switch primary ingest path to a pure-Go Kafka client
  - *Violation signal:* PRs that add a new test target without inheriting Makefile -tags=dynamic
- **Accepted:** Sequential Atlas migration filenames + atlas.sum hash chain — deterministic but produces guaranteed merge conflicts on long-lived branches
  - *Benefit:* Reviewable SQL migrations with chain integrity verified by atlas.sum; impossible to land out-of-order migrations
  - *Caused by:* atlas migrate --env local diff filename convention + atlas.sum chain hashing
  - *Violation signal:* Multiple branches producing same-timestamp migrations
  - *Violation signal:* atlas.sum merge conflicts on long-running branches
  - *Violation signal:* Attempts to edit existing migrations after they land
  - *Violation signal:* Commits that touch tools/migrate/migrations/ without an accompanying atlas.sum update

## Out of Scope

- Frontend UI (frontend_ratio = 0; React only appears in the generated JavaScript SDK under api/client/javascript/) — out of scope because the architectural style is REST-first multi-language SDK; UI is a customer concern
- Business-level auth / identity provider (portal tokens scope end-customers, but tenant-level identity is out of repo) — out of scope because the architectural style targets self-hosted and managed deployments where the surrounding identity provider varies
- Managed hosting control plane (config.cloud.yaml + api/openapi.cloud.yaml exist but the hosted-platform logic is not in the monorepo) — out of scope because multi-binary deployment isolates cloud-only orchestration into a separate codebase