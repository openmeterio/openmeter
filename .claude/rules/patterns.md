## Communication Patterns

### Wire compile-time dependency injection
- **When:** Wiring a new domain service, adapter, hook, or worker into a binary's object graph
- **How:** Each binary (cmd/server, cmd/billing-worker, etc.) declares a build-tagged wire.go (//go:build wireinject) listing wire.NewSet provider sets. Providers are plain constructor functions grouped per concern in app/common/*.go (e.g. var Customer = wire.NewSet(NewCustomerService)). `make generate` runs wire to emit wire_gen.go which call the constructors in dependency order. Constructors take explicit deps (logger, *entdb.Client, eventbus.Publisher) and return (Service, error); they never reach for globals.

### Service / Adapter layered package with Config-validating constructor
- **When:** Creating or extending any domain package under openmeter/ (billing, customer, entitlement, subscription, notification, ledger, charges, app)
- **How:** A package exposes a Service interface and an Adapter interface in its root. The service struct (openmeter/customer/service) holds the adapter plus collaborators; the adapter struct (openmeter/customer/adapter) holds *entdb.Client. Both have a New(config) constructor whose Config has a Validate() error checked first thing in New — returning errors instead of panicking. The service var-asserts the interface: `var _ customer.Service = (*Service)(nil)`.

### Transaction-aware Ent repository (TransactingRepo / HijackTx WithTx)
- **When:** Any adapter method that reads or writes via Ent and must participate in a caller-supplied transaction
- **How:** The adapter implements entutils.TxCreator (Tx() hijacks an Ent tx into the context) and TxUser (WithTx rebuilds the adapter bound to the tx client from raw config; Self() returns the non-tx instance). Adapter methods wrap their body in entutils.TransactingRepo(ctx, a, func(ctx, rep){...}): if a tx driver is already on the context it rebinds to it, otherwise it runs on the base client. transaction.Run(ctx, creator, cb) opens/commits/rolls-back and stores the driver on context; nested Run calls reuse the existing driver and emit savepoints.
- **Applicable when:** pkg/framework/entutils/transaction.go:199 TransactingRepo only rebinds when a *TxDriver is found on ctx, else falls back to repo.Self() — applies to adapter helpers whose backing struct holds an *entdb.Client and implements both TxUser[T] and TxCreator
- **Do NOT apply when:**
  - Helper accepts a raw *entdb.Client argument instead of reading the tx from ctx — per AGENTS.md the charges adapter convention still requires wrapping such helpers with entutils.TransactingRepo so they rebind to the ctx tx; passing a non-tx client silently writes outside the caller's transaction
  - Pure in-memory adapter with no Ent client (e.g. app marketplace registry map in openmeter/app/adapter/marketplace.go:120) — there is no transaction to join

### Postgres transaction-scoped advisory lock (lockr)
- **Scope:** `subscription`, `entitlement`, `billing`, `ledger`
- **When:** Serializing concurrent business operations that span multiple rows/tables for one logical entity, e.g. all subscription mutations for a customer or all charge advances for a customer
- **How:** lockr.NewKey(scopes...) builds a colon-joined scope string, xxh3-hashes it to a 64-bit int, and Locker.LockForTX issues SELECT pg_advisory_xact_lock($1) on the current Ent tx. The lock auto-releases on commit/rollback. getTxClient first asserts the call is inside a real postgres tx (transaction_timestamp() != statement_timestamp()) and errors otherwise. Domain code wraps the key behind a typed helper such as subscription.GetCustomerLock(customerId) = NewKey("customer", customerId, "subscription").
- **Applicable when:** The lock-key identity component must be a globally-unique row identifier. subscription.GetCustomerLock (openmeter/subscription/locks.go:6) keys on the customer id, and customer ids are PK-unique via the IDMixin `field.String("id").Unique()` + `index.Fields("namespace","id").Unique()` declared in pkg/framework/entutils/mixins.go:90-104,77 — so one ("customer", id, scope) key maps to exactly one customer and the advisory lock serializes only that customer's operations
- **Do NOT apply when:**
  - Caller would key the lock on a non-unique column — e.g. customer `key` is only unique under namespace+deleted_at IS NULL (openmeter/ent/schema/customer.go:58-62) not globally; a lock keyed on raw `key` could serialize unrelated customers across namespaces or collide a live row with a soft-deleted one
  - Caller is not inside a postgres transaction — lockr.getTxClient (pkg/framework/lockr/locker.go:134) hard-errors when transaction_timestamp()==statement_timestamp(), so LockForTX outside transaction.Run always fails
  - Operation mutates exactly one row and needs no cross-row invariant — Ent SELECT ... FOR UPDATE row locking suffices and avoids two distinct scope strings colliding into the same 64-bit advisory slot

### Service hook registry (ServiceHookRegistry)
- **When:** Letting one domain react to another domain's lifecycle events in-process (e.g. provision a ledger account when a customer is created, sync a customer when a subject is created)
- **How:** A service embeds models.ServiceHookRegistry[T] and exposes RegisterHooks(...ServiceHook[T]). Wire providers in app/common build the hook implementation and call targetService.RegisterHooks(h) at startup; when the feature is disabled the provider returns a Noop hook instead (e.g. NewCustomerLedgerServiceHook returns ledgerresolvers.NoopCustomerLedgerHook{} when creditsConfig.Enabled is false). Hooks fire synchronously inside the producing service's flow.

### In-process event bus over Watermill CQRS (eventbus.Publisher)
- **When:** Emitting domain events (ingest, balance-worker, system events) to Kafka for asynchronous workers to consume
- **How:** eventbus.New wraps a watermill cqrs.EventBus. Publish routes each marshaler.Event to one of three Kafka topics by event-name prefix: ingest-subsystem-prefixed events to IngestEventsTopic, balance-worker-prefixed to BalanceWorkerEventsTopic, everything else to SystemEventsTopic. Producers call publisher.WithContext(ctx).PublishIfNoError(event, err) to inline publish-or-propagate. nil events are intentionally dropped (a handler returning nil means 'publish nothing').

### Kafka consumer via Watermill grouphandler (type-routed event dispatch)
- **When:** Building a worker (notification, billing-worker, balance-worker) that consumes CloudEvents off Kafka and fans them to per-event-type handlers
- **How:** grouphandler.NewNoPublishingHandler builds a map[eventName][]GroupEventHandler. On each message it derives the CloudEvent type via the marshaler, looks up handlers, unmarshals once into handler[0].NewEvent(), and runs all matching handlers joining their errors. Unknown event types are counted as 'ignored' and ack'd (return nil), not failed. Per-message processing time and status counters are emitted to OpenTelemetry.

### Explicit finite state machine (qmuntal/stateless) with external storage
- **Scope:** `billing`, `charges`
- **When:** Modeling the invoice lifecycle and per-charge UBP lifecycle where transitions are guarded and have side effects
- **How:** stateless.NewStateMachineWithExternalStorage stores the current state on the domain aggregate (the invoice/charge), not in the FSM. Each state is Configure'd with Permit(trigger, targetState) edges, guarded variants PermitDynamic(trigger, target, guardFn), and OnActive side-effect callbacks composed via statelessx.AllOf/BoolFn. The invoice machine is pooled (sync.Pool, invoiceStateMachineCache) and reset per use. The charges machine is a generic Machine[CHARGE, BASE, STATUS] driven by AdvanceUntilStateStable and a Persistence{UpdateBase, Refetch} struct, accumulating invoiceupdater.Patch side effects.
- **Applicable when:** openmeter/billing/charges/statemachine/machine.go:16 the STATUS type parameter is constrained `~string` + Validate() error, and machine.go:39 requires CHARGE to implement ChargeLike (GetStatus/WithStatus/GetBase/WithBase) — applies to charge aggregates that carry their status as a string enum and can rebuild themselves with a new status
- **Do NOT apply when:**
  - State is not externally owned by the aggregate — these machines use NewStateMachineWithExternalStorage (stdinvoicestate.go:48) and read/write status on the domain object; an in-memory-state FSM would not persist transitions across the request
  - A transition has no guard and no side effect and the type has only two states — a plain bool/enum field is simpler than configuring a stateless machine

### In-memory map registry with type-asserted factory dispatch (app marketplace)
- **Scope:** `app`
- **When:** Pluggable app integrations (Stripe, custom-invoicing, sandbox) registered at startup and instantiated on demand by type
- **How:** The app adapter holds registry map[AppType]RegistryItem. Each integration's service constructor calls AppService.RegisterMarketplaceListing(RegistryItem{Listing, Factory}) once at wiring time; RegisterMarketplaceListing rejects duplicate types and validates the listing. Install operations look up the RegistryItem and type-assert the Factory to the capability interface actually requested (Factory.(app.AppFactoryInstallWithAPIKey) vs Factory.(app.AppFactoryInstall)), returning a GenericValidationError if the app doesn't support that install method.
- **Applicable when:** openmeter/app/adapter/marketplace.go:121 RegisterMarketplaceListing rejects a second registration of the same AppType — applies to integration factories that register exactly once per process at wiring time; the map is the single source of installable app types and capability support is discovered by interface type-assertion on Factory
- **Do NOT apply when:**
  - Registration happens after the listing surface is already serving requests — the map has no locking around late registration, so listings must be registered during DI wiring before the HTTP/worker surface is live

### Per-subtype connector dispatch behind a single Service (entitlement)
- **Scope:** `entitlement`
- **When:** A service whose behavior branches by an enum subtype (metered / static / boolean entitlements) where each subtype has its own logic implementation
- **How:** entitlement.SubTypeConnector is implemented three times (metered/static/boolean connector.go). The aggregate service holds all three as fields and getTypeConnector(typed) switches on EntitlementType to return the right connector, returning an error in the default case. Sub-type-specific create/value logic is delegated to the selected connector while the umbrella service owns shared concerns.
- **Applicable when:** openmeter/entitlement/service/service.go:424 getTypeConnector switches over the closed EntitlementType set declared in openmeter/entitlement/entitlement.go:331 (Values()={metered,static,boolean}) with a default-error arm — applies when adding behavior keyed on an entitlement subtype; a new EntitlementType value MUST also get a SubTypeConnector field + switch arm or it falls through to the default error
- **Do NOT apply when:**
  - The branching value is open/user-defined rather than the fixed EntitlementType enum — a closed switch with a default-error arm would reject legitimate values

### Generic HTTP handler pipeline (httptransport.Handler[Request,Response])
- **When:** Implementing a v1 HTTP endpoint: decode request -> call service -> encode response, with consistent error encoding
- **How:** httptransport.NewHandler[Request,Response](decode, service-op, encode, ...opts) builds a typed handler. Per-domain httpdriver packages compose a decoder (request -> typed Request), the service call, and an encoder, plus an errorEncoder that maps domain error types to status codes. Handlers can be Chain'd with operation.Middleware.

### Domain-error to HTTP-status mapping via typed error encoders
- **When:** Translating service-layer errors into HTTP responses (RFC7807 problem documents)
- **How:** Two complementary mechanisms. (1) Per-handler errorEncoder chains commonhttp.HandleErrorIfTypeMatches[T](ctx, status, err, w) — ordered short-circuit on errors.As against concrete domain error types (e.g. notification.NotFoundError->404, GenericValidationError->400, UpdateAfterDeleteError->409). (2) Generic errors carry an HTTP status as a ValidationIssue attribute (openmeter.http.status_code); commonhttp.HandleIssueIfHTTPStatusKnown maps it but only when all issues agree on a single status ('singular' behavior), else declines. The v3 surface mirrors the same singular logic in apierrors.NewV3ErrorHandlerFunc, rendering v3 BaseAPIError shapes.

### Accumulating Validate() with NillableGenericValidationError
- **When:** Validating any input struct or domain config
- **How:** Validate() methods collect issues into var errs []error, wrap each with field context (fmt.Errorf("field: %w", err)), and return models.NewNillableGenericValidationError(errors.Join(errs...)) so a nil join yields nil. Simpler single-field checks return models.NewGenericValidationError(...) directly. The generic validation error maps to HTTP 400 at the edge.

### Goverter / Goderive code-generated type converters
- **When:** Translating between domain, API, and DB representations without hand-writing boilerplate mappers
- **How:** convert.gen.go files are generated from goverter converter interfaces declared in convert.go; billing/derived.gen.go is generated from goderive annotations. Hand-written conversion files/functions follow the FromAPI.../ToAPI.../FromDB.../ToDB... naming convention (the /go-types-conversion skill). Never edit *.gen.go (Code generated ... DO NOT EDIT header).

### Svix-backed webhook delivery behind a Handler interface
- **When:** Delivering notification events to customer-configured HTTP endpoints
- **How:** notification/webhook defines a Handler interface (CreateWebhook, UpdateWebhook, endpoint secret/header management) with a Svix implementation (webhook/svix) and a noop implementation (webhook/noop) for when webhooks are disabled. Svix calls are wrapped in OpenTelemetry tracex spans. The notification event pipeline reconciles delivery state via eventhandler/reconcile.go.

### Ent schema mixins for cross-cutting columns (ResourceMixin/IDMixin/TimeMixin/AnnotationsMixin)
- **When:** Defining a new persisted entity that needs the standard id/namespace/metadata/timestamps and soft-delete columns
- **How:** Schema structs compose entutils mixins in Mixin(). ResourceMixin pulls in IDMixin (ULID char(26) PK, unique), NamespaceMixin, MetadataMixin (jsonb), TimeMixin (created/updated/deleted_at) and a unique (namespace,id) index. UniqueResourceMixin adds a (namespace,key,deleted_at) unique index approximating partial-unique-on-not-deleted. Soft delete is via deleted_at; unique indexes are scoped with deleted_at to avoid resurrect collisions.
- **Applicable when:** pkg/framework/entutils/mixins.go:47 UniqueResourceMixin's (namespace,key,deleted_at) unique index only approximates partial uniqueness — same-microsecond create/delete/create can collide because Ent cannot emit WHERE deleted_at IS NULL without a manual migration; applies to entities relying on key-uniqueness for soft-deleted rows
- **Do NOT apply when:**
  - Entity needs true partial-unique (namespace,key) WHERE deleted_at IS NULL — must add a custom SQL migration with IndexWhere, as the customer schema does at openmeter/ent/schema/customer.go:58-62, rather than relying on the mixin's deleted_at-in-key approximation

## Integrations

| Service | Purpose | Integration point |
|---------|---------|-------------------|
| PostgreSQL (Ent ORM + Atlas + pgx) | Primary transactional datastore for all domain entities; schema generated from ent schema, migrated via Atlas/golang-migrate | `openmeter/ent/schema/, pkg/framework/entutils/transaction.go, app/common/database.go, tools/migrate/migrations/` |
| ClickHouse | Analytics store for metered usage events; streaming queries (CountEvents, QueryMeter) feed entitlement balance and billing | `openmeter/streaming/clickhouse/, openmeter/streaming/connector.go, app/common/clickhouse.go` |
| Kafka (confluent-kafka-go + Watermill CQRS) | Event backbone: ingest events, system events, balance-worker events; consumed by notification/billing/balance/sink workers | `openmeter/watermill/eventbus/eventbus.go, openmeter/watermill/grouphandler/grouphandler.go, app/common/kafka.go, openmeter/notification/consumer/consumer.go` |
| Svix | Outbound webhook delivery to customer endpoints for notification events | `openmeter/notification/webhook/svix/webhook.go, openmeter/notification/webhook/svix/svix.go` |
| Stripe | Payment/invoicing app integration registered in the app marketplace; handles inbound Stripe webhooks | `openmeter/app/stripe/service/service.go, openmeter/app/stripe/service/factory.go, openmeter/app/stripe/httpdriver/webhook.go` |
| GOBL | Invoice document format/representation for billing invoices | `openmeter/billing/service/` |
| OpenTelemetry | Tracing and metrics across services, Kafka consumers, and Svix calls | `app/common/telemetry.go, openmeter/watermill/grouphandler/grouphandler.go, openmeter/notification/webhook/svix/webhook.go` |
| Redis | Dedupe and supporting caches (optional) | `pkg/redis/, openmeter/dedupe/` |
| Viper + Cobra | Configuration loading and CLI command structure for the binaries | `app/config/, app/common/config.go` |

## Pattern Selection Guide

| Scenario | Pattern | Rationale |
|----------|---------|-----------|
| Adding a new domain package with persistence | Service/Adapter layered package with Config.Validate() constructors + Ent adapter implementing TxCreator/TxUser | Matches every existing domain (customer, billing, entitlement); transaction-awareness via entutils.TransactingRepo lets the adapter join caller transactions and keeps namespace/multi-tenancy handling uniform. |
| Wiring a new service/hook/worker into a binary | Add a wire.NewSet provider in app/common/*.go and reference it from the binary's wire.go, then run make generate | Wire is the single DI mechanism; constructors take explicit deps and return errors, so feature flags (e.g. credits.enabled) gate wiring by returning Noop implementations. |
| Serializing all operations for one logical entity across multiple tables | lockr advisory lock keyed on a globally-unique id inside transaction.Run | pg_advisory_xact_lock auto-releases with the tx and serializes business flows the DB's row locks can't span; only safe when the key's id component is PK-unique (see lockr precondition). |
| Reacting to another domain's lifecycle event in-process | ServiceHookRegistry hook registered at wiring time (Noop when feature disabled) | Synchronous, transactional in-process coupling without a Kafka round-trip; used for ledger provisioning, subject->customer sync, entitlement validation. |
| Reacting to events asynchronously across workers | eventbus.Publish -> Kafka topic by event-name prefix -> grouphandler type-routed consumer | Decouples producers from workers; unknown event types are safely ignored/ack'd, and per-type handlers unmarshal once. |
| Modeling a guarded multi-step lifecycle with side effects (invoice, charge) | qmuntal/stateless FSM with external storage on the aggregate | Permit/PermitDynamic/OnActive make legal transitions and guards explicit and testable; state lives on the persisted aggregate so transitions survive across requests. |
| Pluggable third-party app integration (Stripe-like) | RegisterMarketplaceListing into the in-memory app registry; capabilities discovered by Factory type-assertion | Single registration point with duplicate rejection; install methods are validated against the listing and the factory's implemented interfaces. |
| Behavior that branches by a fixed enum subtype | Per-subtype connector behind one service with a getTypeConnector switch (default-error arm) | Keeps shared concerns in the umbrella service while delegating subtype logic; the default-error arm forces a new enum value to be wired explicitly. |
| Returning errors to HTTP clients with correct status | Typed error encoders (HandleErrorIfTypeMatches) plus ValidationIssue http.status_code attribute mapping | Centralizes RFC7807 problem rendering; the 'singular' rule avoids ambiguous multi-status responses, and v3 mirrors the same logic in apierrors. |
| Validating input structs | Accumulate errs []error and return NewNillableGenericValidationError(errors.Join(...)) | Reports all field errors at once with field-prefixed wrapping and maps cleanly to HTTP 400. |
| Converting between domain/API/DB types | Goverter/Goderive generated converters; hand-written FromAPI/ToAPI/FromDB/ToDB helpers | Generated mappers cut boilerplate; the naming convention (per /go-types-conversion) keeps translation direction unambiguous. |

## Quick Pattern Lookup

- **New persisted domain package** -> Service/Adapter + Config.Validate() constructor + Ent TxCreator/TxUser adapter
- **Wire a service/worker into a binary** -> wire.NewSet provider in app/common + binary wire.go + make generate
- **Run repository logic inside/optionally-inside a transaction** -> transaction.Run + entutils.TransactingRepo
- **Serialize all ops for one entity across tables** -> lockr advisory lock on globally-unique id inside transaction.Run  *(scope: subscription, entitlement, billing, ledger)*
- **React to another domain's lifecycle in-process** -> ServiceHookRegistry hook (Noop when disabled)
- **Async cross-worker reaction** -> eventbus.Publish -> Kafka by prefix -> grouphandler type-routed consumer
- **Guarded multi-step lifecycle (invoice/charge)** -> qmuntal/stateless FSM with external storage  *(scope: billing, charges)*
- **Pluggable app integration** -> RegisterMarketplaceListing + Factory type-assertion dispatch  *(scope: app)*
- **Behavior branching by fixed enum subtype** -> Per-subtype connector + getTypeConnector switch (default error)  *(scope: entitlement)*
- **Map domain errors to HTTP status** -> HandleErrorIfTypeMatches chain / http.status_code ValidationIssue attribute
- **Validate input struct** -> errs []error + NewNillableGenericValidationError(errors.Join)

## Decision Chain

**Root constraint:** Provide event-time usage metering AND ACID usage-based billing on the same multi-tenant platform: high-volume append-only usage events must aggregate cheaply, while subscriptions/invoices/credits/ledger demand cross-row transactional invariants — over one shared codebase.

- **Multi-binary Go control plane with layered service/adapter domains, code-generated API contract, and an event-time usage-metering data plane**: Read-heavy ingest/aggregation and transaction-heavy billing have opposing scaling and failure profiles, so usage events go to a columnar append-only store and a streaming pipeline while control-plane state stays in OLTP Postgres — split into independently scalable binaries sharing one module.
  - *Violation keyword:* `usage events in postgres`
  - *Violation keyword:* `single monolith binary`
  - *Violation keyword:* `microservice per domain with own db`
  - *Violation keyword:* `mongodb`
  - *Violation keyword:* `dynamodb for control plane`
  - **Ent schema mixins as the cross-cutting column contract (ResourceMixin / IDMixin / NamespaceMixin / TimeMixin / AnnotationsMixin)**: One shared Postgres schema across ~70 multi-tenant tables needs a uniform namespace/ULID/audit/soft-delete shape, defined once via mixins with Ent as source of truth and Atlas-generated migrations.
    - *Violation keyword:* `gorm`
    - *Violation keyword:* `prisma`
    - *Violation keyword:* `sqlalchemy`
    - *Violation keyword:* `hard delete`
    - *Violation keyword:* `DROP without deleted_at`
    - *Violation keyword:* `uuid v4 pk instead of ulid`
    - *Violation keyword:* `per-table id column duplication`
  - **Transaction-aware Ent repository (TransactingRepo / HijackTx / WithTx)**: Cross-domain atomicity over a single shared Ent client requires adapters that join a caller-supplied transaction carried on context, rather than each owning its own connection.
    - *Violation keyword:* `context.Background()`
    - *Violation keyword:* `context.TODO()`
    - *Violation keyword:* `raw *entdb.Client into a writing helper`
    - *Violation keyword:* `separate db connection per repo`
    - *Violation keyword:* `two-phase commit`
    - **Postgres transaction-scoped advisory locking (lockr)**: Multi-replica workers mutating multi-row per-customer aggregates need a serialization point tied to the transaction; pg_advisory_xact_lock keyed on a unique id auto-releases on commit.
      - *Violation keyword:* `sync.Mutex`
      - *Violation keyword:* `pg_advisory_lock`
      - *Violation keyword:* `redis lock for in-tx serialization`
      - *Violation keyword:* `lock on customer key`
      - *Violation keyword:* `LockForTX outside transaction.Run`
    - **Explicit finite state machines with external storage (qmuntal/stateless) for invoice and per-charge UBP lifecycles**: Guarded, side-effecting invoice/charge transitions must be durable across requests, so the FSM stores status on the persisted aggregate inside the transaction.
      - *Violation keyword:* `NewStateMachine (in-memory)`
      - *Violation keyword:* `invoice.status = ... direct mutation`
      - *Violation keyword:* `if status == switch transitions`
      - *Violation keyword:* `ad-hoc lifecycle flags`
    - **Polymorphic Charge parent row with idempotent unique_reference_id and a Postgres UNION-ALL search view**: Three charge subtypes with distinct run/lineage children still need one idempotency key and one read surface, written transactionally with the rest of billing.
      - *Violation keyword:* `single wide charges table`
      - *Violation keyword:* `charge create without unique_reference_id`
      - *Violation keyword:* `ent.View expected in migrate.Tables`
      - *Violation keyword:* `separate charge tables without parent`
    - **Service hook registry (models.ServiceHookRegistry) for in-process cross-domain reactions**: Effects that must commit in the same transaction as the triggering write (ledger provisioning on customer create) are registered as in-process hooks, not routed through Kafka.
      - *Violation keyword:* `kafka publish for transactional side effect`
      - *Violation keyword:* `static import of ledger from customer`
      - *Violation keyword:* `hard-coded cross-domain call in service body`
  - **Google Wire compile-time dependency injection in app/common, per-binary provider sets**: Six binaries over one module each need a distinct object graph; compile-time DI assembles only the needed providers and fails the build on a missing one, and provider swaps (concrete vs noop) gate features.
    - *Violation keyword:* `reflection DI container`
    - *Violation keyword:* `runtime service locator`
    - *Violation keyword:* `manual wiring in main()`
    - *Violation keyword:* `if feature.enabled inside service logic`
    - **In-memory map registry with type-asserted factory dispatch for app/marketplace integrations**: Pluggable Stripe/custom-invoicing integrations register exactly once at wiring time into a duplicate-rejecting map that is the single source of installable app types.
      - *Violation keyword:* `central switch over app type`
      - *Violation keyword:* `runtime plugin load`
      - *Violation keyword:* `late registration after server start`
      - *Violation keyword:* `duplicate AppType registration`
  - **TypeSpec-first API contract with full codegen fan-out (OpenAPI → oapi-codegen + 3 SDKs + goverter/goderive)**: Two API surfaces and three published SDKs cannot be kept consistent by hand, so a single TypeSpec source generates OpenAPI, both Go server stubs, the SDKs, and the type converters.
    - *Violation keyword:* `edit api.gen.go`
    - *Violation keyword:* `hand-edit openapi.yaml`
    - *Violation keyword:* `edit convert.gen.go`
    - *Violation keyword:* `hand-written SDK method`
    - *Violation keyword:* `edit wire_gen.go`
    - **Two coexisting HTTP surfaces: legacy v1 httptransport drivers + thin AIP v3 delegators, both over the same domain services**: An installed v1 client base plus a new AIP direction means both surfaces front the identical domain services, sharing typed error-to-status mapping.
      - *Violation keyword:* `duplicate domain logic per surface`
      - *Violation keyword:* `ad-hoc status code per handler`
      - *Violation keyword:* `drop v1 router`
      - *Violation keyword:* `bypass httptransport.Handler`
    - **Accumulating Validate() returning NillableGenericValidationError as the uniform input-contract gate**: Multiple surfaces and domains need one multi-issue validation contract that maps to a single 400 ValidationIssue.
      - *Violation keyword:* `return on first invalid field`
      - *Violation keyword:* `panic on invalid input`
      - *Violation keyword:* `custom validation error per domain`
      - *Violation keyword:* `slog.Default() fallback`
  - **Two-tier async messaging: eventbus.Publisher over Watermill CQRS out, grouphandler type-routed dispatch in**: Decoupled worker binaries consume high-volume usage events and lower-volume domain events with different topic/retention needs, fanned to per-type handlers via a shared dispatch.
    - *Violation keyword:* `direct confluent-kafka produce in worker`
    - *Violation keyword:* `single topic for all events`
    - *Violation keyword:* `synchronous cross-binary call`
    - *Violation keyword:* `no DLQ for poison messages`
    - **Per-subtype connector dispatch behind one entitlement Service**: Entitlement (and analogously charge) behavior branches by a closed enum subtype consumed by workers and services, isolated behind a uniform SubTypeConnector with a default-error arm.
      - *Violation keyword:* `inline switch over entitlement type everywhere`
      - *Violation keyword:* `open/registry subtype dispatch`
      - *Violation keyword:* `default arm that accepts unknown subtype`

## Key Decisions

### Transaction-aware Ent repository (TransactingRepo / HijackTx / WithTx)
**Chosen:** Every domain adapter holds a *entdb.Client and implements entutils.TxCreator (Tx() hijacks an Ent tx onto the context) and TxUser[T] (WithTx rebinds the adapter to the tx client; Self() returns the non-tx instance). Adapter methods wrap their body in entutils.TransactingRepo(ctx, a, func(ctx, rep){...}), which rebinds to the *TxDriver already on the context if one exists and otherwise falls back to repo.Self() (pkg/framework/entutils/transaction.go:199).
**Rationale:** This is the seam that lets one domain's service participate in another domain's transaction over the single shared Ent client (Generated Ent client component, openmeter/ent/db). Without it, subscription→billing→charges→ledger composition could not be atomic. AGENTS.md elevates this to a hard convention for the charges adapter: even helpers that accept a raw *entdb.Client must still wrap their body in entutils.TransactingRepo so they rebind to the ctx tx rather than silently writing outside the caller's transaction. Implemented in pkg/framework/entutils/transaction.go and pkg/framework/transaction/transaction.go, consumed by every adapter (openmeter/customer/adapter/adapter.go, openmeter/billing/adapter/adapter.go).
**Rejected:** Passing an explicit *sql.Tx or tx-bound client through every method signature — rejected because it pollutes every domain interface and is easy to forget; the context-carried *TxDriver makes the tx ambient and the rebind automatic., A repository-per-aggregate with its own connection pool — rejected because cross-domain atomic writes need one connection/transaction, not coordinated commits.
**Forced by:** Single shared Ent client + the requirement that subscription, billing, charges, and ledger mutations be atomic across domains.
**Enables:** Composable service-in-service transactions, the lockr advisory-lock pattern (which asserts it runs inside a real Postgres tx), and the subscription→billing sync writing invoice lines, split-line groups, and charges in one commit.

### Postgres transaction-scoped advisory locking (lockr)
**Chosen:** lockr.NewKey(scopes...) joins a colon-separated scope, xxh3-hashes it to a 64-bit int, and Locker.LockForTX issues SELECT pg_advisory_xact_lock($1) on the current Ent tx, auto-releasing on commit/rollback. getTxClient first hard-asserts the caller is inside a real Postgres transaction (transaction_timestamp() != statement_timestamp()) and errors otherwise (pkg/framework/lockr/locker.go:134).
**Rationale:** Subscription mutations, billing mutations, and charge advances each touch many rows/tables for one customer and must be serialized per logical entity without serializing the whole table. lockr keys on a globally-unique row id (subscription.GetCustomerLock keys on customer id, openmeter/subscription/locks.go:6; customer ids are PK-unique via IDMixin). It is deliberately Postgres-tx-scoped so the lock cannot leak past the transaction boundary. Implemented in pkg/framework/lockr/{locker.go,key.go}; used across subscription, billing, ledger, entitlement. Note there is also a row-based BillingCustomerLock table (openmeter/ent/schema/billing.go:1334) used via SELECT ... FOR UPDATE for the same per-customer serialization in billing.
**Rejected:** Application-level mutexes — rejected because the binaries are horizontally scaled; an in-process lock would not serialize across replicas., Redis/distributed lock (the repo has cirello.io/pglock available) — rejected for the in-transaction case because pg_advisory_xact_lock ties lock lifetime to the exact transaction that does the writes, eliminating lock-then-die orphan windows., Keying the lock on customer.key — rejected because key is only unique under namespace + deleted_at IS NULL, so a key-based lock could serialize unrelated namespaces or collide a live row with a soft-deleted one.
**Forced by:** Multi-replica workers mutating multi-row per-customer aggregates that need a serialization point.
**Enables:** Safe concurrent subscription edits, billing finalization, and charge advancement for different customers in parallel while serializing operations on the same customer.

### Explicit finite state machines with external storage (qmuntal/stateless) for invoice and per-charge UBP lifecycles
**Chosen:** stateless.NewStateMachineWithExternalStorage stores the current state on the domain aggregate (invoice.status / charge.status), not inside the FSM. States are Configure'd with Permit / PermitDynamic(guard) edges and OnActive side-effect callbacks. The charge machine is generic over a STATUS ~string + Validate() type and a CHARGE implementing ChargeLike (GetStatus/WithStatus/GetBase/WithBase) (openmeter/billing/charges/statemachine/machine.go:16,39).
**Rationale:** Invoice and usage-based-charge lifecycles have guarded transitions with side effects (calculation, finalization, voiding) that must be auditable and persisted across requests. Externalizing state onto the row means a transition computed in one request is durable and the next request resumes from the stored status. Implemented in openmeter/billing/service/stdinvoicestate.go:48 and openmeter/billing/charges/statemachine/machine.go.
**Rejected:** Ad-hoc if/switch transition code on a status field — rejected because guards and side effects scatter and illegal transitions become possible; the FSM centralizes the legal edge set., In-memory-state FSM — rejected because transitions must survive across requests; external storage on the aggregate is mandatory., A stateless machine for trivially-two-state fields — explicitly avoided; a plain bool/enum is simpler where there is no guard and no side effect.
**Forced by:** Guarded, side-effecting, auditable invoice and charge lifecycles that span multiple requests.
**Enables:** A single legal-transition source of truth per aggregate, reusable across flat-fee/usage-based/credit-purchase charge subtypes via the generic ChargeLike constraint.

### TypeSpec-first API contract with full codegen fan-out (OpenAPI → oapi-codegen + 3 SDKs + goverter/goderive)
**Chosen:** The API is authored in TypeSpec under api/spec (two pnpm packages: legacy and aip). make gen-api compiles it to api/openapi.yaml / api/v3/openapi.yaml, then oapi-codegen generates api/api.gen.go (v1 ServerInterface) and api/v3/api.gen.go (v3 ServerInterface), orval generates the JS SDK, poetry/corehttp the Python SDK, and goverter/goderive generate type converters (convert.gen.go, billing/derived.gen.go). All carry a 'DO NOT EDIT' header.
**Rationale:** Two server surfaces and three published SDKs cannot be kept consistent by hand. TypeSpec is the single source of truth; the codegen chain enforces it. The Makefile even patches the oapi-codegen chi-middleware template (patch-oapi-templates) for custom AIP filter parsing. Components: TypeSpec API specification, Generated API contract & v1 server interface, v3 API layer, JavaScript/TypeScript SDK, Python SDK.
**Rejected:** OpenAPI YAML as the hand-edited source of truth — rejected because TypeSpec is more composable/typed and avoids drift between the two API packages., Hand-written SDKs — rejected; orval/poetry generation keeps JS/Python/Go clients lock-step with the spec., Hand-written domain↔API↔DB mappers — rejected for goverter/goderive generation following the FromAPI/ToAPI/FromDB/ToDB naming convention (go-types-conversion skill).
**Forced by:** Two coexisting API surfaces (v1 + AIP v3) plus three published SDKs that must not diverge from one contract.
**Enables:** Adding an endpoint by editing .tsp and re-running make gen-api / make generate; compile-time detection of contract drift.

### Google Wire compile-time dependency injection in app/common, per-binary provider sets
**Chosen:** Each binary has a build-tagged wire.go (//go:build wireinject) listing wire.NewSet provider sets; constructors are plain functions grouped per concern in app/common/*.go (e.g. var Customer = wire.NewSet(NewCustomerService); BillingRegistry/ChargesRegistry aggregation structs in app/common/billing.go). make generate runs wire to emit wire_gen.go. Feature flags switch concrete vs noop providers at wiring time (app/common/ledger.go picks concrete or noop ledger services on credits.enabled; NewCustomerLedgerService returns a Noop hook when disabled).
**Rationale:** Six binaries share one module but each needs a different object graph. Compile-time DI means a missing provider fails the build, and each binary only links the providers it declares. Feature-gating at the provider level (credits.enabled, webhooks enabled) is the single seam where a whole subsystem becomes a no-op without touching call sites. Components: Application wiring (app/common), Server entrypoint & DI bootstrap.
**Rejected:** Runtime/reflection DI container — rejected because Wire surfaces wiring errors at build time and generates inspectable code., Manual constructor wiring in each main() — rejected; the graph (billing→charges→ledger→customer→productcatalog) is too large to wire by hand per binary without divergence., Feature flags checked deep in service logic — rejected in favor of swapping provider implementations (concrete vs noop) at the wiring seam so disabled features short-circuit cleanly (AGENTS.md: credits.enabled noop wiring).
**Forced by:** Six binaries over one module, each needing a distinct subset of the domain graph, plus feature flags that must disable whole subsystems.
**Enables:** Per-binary minimal object graphs, build-time wiring validation, and clean feature-flag disablement via noop provider swaps.

### Service hook registry (models.ServiceHookRegistry) for in-process cross-domain reactions
**Chosen:** A service embeds models.ServiceHookRegistry[T] and exposes RegisterHooks(...ServiceHook[T]). Wire providers in app/common construct the hook and call targetService.RegisterHooks(h) at startup; when a feature is off the provider registers a Noop hook instead (e.g. customer-created → provision a ledger account; subject-created → sync customer). Implemented in openmeter/customer/service/service.go, wired in app/common/customer.go.
**Rationale:** Some cross-domain effects must be synchronous and in-process (provisioning a ledger account when a customer is created), unlike the asynchronous Kafka event bus used for billing/notification. The hook registry is the in-process counterpart to the event bus, and registration at wiring time keeps the coupling explicit and feature-gatable.
**Rejected:** Routing every cross-domain reaction through Kafka — rejected for effects that must commit in the same transaction as the triggering write (ledger provisioning), where eventual consistency is unacceptable., Hard-coding the dependent call inside the triggering service — rejected because it would create a compile-time dependency from customer onto ledger and prevent the noop-when-disabled swap.
**Forced by:** Cross-domain effects that must be synchronous/transactional and feature-gatable, alongside the async event bus for the rest.
**Enables:** Provisioning ledger/customer state on lifecycle events without a static dependency edge, switchable to Noop when the feature is disabled.

### Two-tier async messaging: eventbus.Publisher over Watermill CQRS out, grouphandler type-routed dispatch in
**Chosen:** Outbound: eventbus.New wraps a watermill cqrs.EventBus; Publish routes each marshaler.Event to one of three Kafka topics by event-name prefix (ingest → IngestEventsTopic, balance-worker → BalanceWorkerEventsTopic, everything else → SystemEventsTopic) (openmeter/watermill/eventbus/eventbus.go). Inbound: grouphandler.NewNoPublishingHandler builds map[eventName][]GroupEventHandler, derives the CloudEvent type per message, unmarshals once into handler[0].NewEvent(), runs all matching handlers joining errors, and counts unknown types as ignored (openmeter/watermill/grouphandler/grouphandler.go, openmeter/notification/consumer/consumer.go).
**Rationale:** Usage ingest and domain events have different durability/throughput needs, so they are routed to distinct topics; workers (notification, billing, balance) consume CloudEvents and fan them to per-type handlers without each worker re-implementing topic/dispatch plumbing. CloudEvents is the wire format (cloudevents/sdk-go/v2). There is also a notification DLQ topic om_sys.notification_service_dlq.
**Rejected:** Direct confluent-kafka-go produce/consume in every worker — rejected; Watermill CQRS + grouphandler centralizes marshaling, topic routing, and type dispatch., A single Kafka topic for all events — rejected because ingest volume and system events have different scaling/retention profiles; prefix-based topic routing separates them., Synchronous cross-binary calls — rejected; workers are decoupled by Kafka so the API path does not block on billing/notification work.
**Forced by:** Decoupled worker binaries consuming high-volume usage events and lower-volume domain events with different topic/retention needs.
**Enables:** Adding a new event handler by registering it in the grouphandler map; routing a new event type by prefix; independent worker scaling and a DLQ for poison notification messages.

### In-memory map registry with type-asserted factory dispatch for app/marketplace integrations
**Chosen:** The app adapter holds registry map[AppType]RegistryItem. Each integration's constructor calls AppService.RegisterMarketplaceListing(RegistryItem{Listing, Factory}) exactly once at wiring time; RegisterMarketplaceListing rejects duplicate AppTypes and validates the listing (openmeter/app/adapter/marketplace.go:121). Install operations look up the factory by AppType. Implementations: stripe, custominvoicing, sandbox.
**Rationale:** Third-party billing/invoicing integrations are pluggable and discovered by type. A startup-time registration map is the single source of installable app types; capability support is discovered by type assertion on the constructed instance. The map has no late-registration locking, so all listings must register during DI before the HTTP/worker surface is live. Component: App / marketplace integrations.
**Rejected:** A hard-coded switch over app types — rejected; the registry lets a new integration register itself from its own package without editing a central switch., Runtime/dynamic plugin loading — rejected; registration is compile-time-known and validated once at wiring, avoiding late-registration races.
**Forced by:** Pluggable Stripe / custom-invoicing / sandbox integrations that must be addable without editing a central dispatch.
**Enables:** Adding a marketplace app by registering a RegistryItem at wiring time and implementing the capability interfaces it advertises.

### Per-subtype connector dispatch behind one entitlement Service
**Chosen:** entitlement.SubTypeConnector is implemented three times (metered/static/boolean, each in its own connector.go). The aggregate service holds all three and getTypeConnector(typed) switches over the closed EntitlementType set with a default-error arm (openmeter/entitlement/service/service.go:424; values metered/static/boolean from openmeter/entitlement/entitlement.go:331).
**Rationale:** Entitlement behavior branches by a fixed enum subtype, each with its own create/value logic. A closed switch with a default-error arm keeps the type set sealed and forces every new subtype to be handled explicitly. This mirrors the charges subtype split (flat_fee / usage_based / credit_purchase) where the Charge row carries exactly one of three subtype FK columns.
**Rejected:** One mega-service with inline branching everywhere — rejected; the per-subtype connector isolates each subtype's logic behind a uniform interface., Open/registry-based subtype dispatch — rejected because the subtype set is fixed (a closed enum) and a default-error arm should reject unknown values rather than accept user-defined ones.
**Forced by:** A fixed enum of entitlement subtypes each needing distinct create/value logic.
**Enables:** Adding a subtype by implementing SubTypeConnector and adding one switch arm; the compiler/default-arm flags any unhandled subtype.

### Polymorphic Charge parent row with idempotent unique_reference_id and a Postgres UNION-ALL search view
**Chosen:** Charge is a polymorphic parent row where exactly one of three subtype FK columns (flat_fee / credit_purchase / usage_based) points to the subtype table, with UNIQUE(namespace, unique_reference_id) WHERE unique_reference_id IS NOT NULL AND deleted_at IS NULL for idempotent creation (openmeter/ent/schema/charges.go:167). Reads go through the ChargesSearchV1 Postgres VIEW (UNION ALL of the three subtype tables). The view DDL is shipped via an explicit SQL migration because Ent views do not appear in generated migrate.Tables (AGENTS.md ent-view caveat).
**Rationale:** Charges share a common base but have type-specific lifecycle/run tables (ChargeFlatFeeRun, ChargeUsageBasedRuns, CreditRealizationLineage). The parent-plus-subtype layout keeps the shared columns in one place and the idempotency key makes charge creation safe under retries. The search view denormalizes reads across subtypes. Components: Charges sub-system; data models Charge, ChargesSearchV1, ChargeFlatFee, ChargeUsageBased, ChargeCreditPurchase.
**Rejected:** Single wide charges table with nullable type-specific columns — rejected; type-specific run/lineage children would not fit and the table would be sparse., Separate top-level tables per charge type with no shared parent — rejected; loses the shared base, the single idempotency key, and the unified search surface., Relying on Ent to emit the view in migrate.Tables — rejected/impossible: ent.View schemas generate query code but not migration metadata in this repo, so the view DDL needs a hand-written SQL migration.
**Forced by:** Three charge subtypes with distinct run/lineage child tables that still need one idempotency key and one read surface.
**Enables:** Idempotent charge creation under retries and unified cross-subtype charge search, while keeping subtype lifecycles isolated.

### Ent schema mixins as the cross-cutting column contract (ResourceMixin / IDMixin / NamespaceMixin / TimeMixin / AnnotationsMixin)
**Chosen:** Persisted entities compose entutils mixins in Mixin(). ResourceMixin pulls in IDMixin (ULID char(26) PK, unique), NamespaceMixin, MetadataMixin (jsonb), TimeMixin (created/updated/deleted_at) and a unique (namespace,id) index; UniqueResourceMixin adds a (namespace,key,deleted_at) unique index that only approximates partial uniqueness (pkg/framework/entutils/mixins.go:47). The Ent schema (openmeter/ent/schema/*.go) is the single source of truth; Atlas diffs it into golang-migrate SQL under tools/migrate/migrations.
**Rationale:** ~70 tables share the same multi-tenant, soft-delete, ULID-id, audit, metadata-jsonb shape; mixins enforce it once instead of per-table. The (namespace,key,deleted_at) approximation is a known sharp edge: same-microsecond create/delete/create can collide because Ent cannot emit WHERE deleted_at IS NULL without a manual migration, so entities needing true partial-uniqueness (Customer) ship a custom IndexWhere SQL migration (openmeter/ent/schema/customer.go:58-62). The data_overview confirms almost every Postgres table is namespace-scoped, ULID-id'd, and deleted_at soft-deleted.
**Rejected:** Repeating id/namespace/timestamp columns in every schema file — rejected; mixins keep the multi-tenant contract uniform and DRY., Hard deletes — rejected in favor of deleted_at soft-deletes so historical billing/usage references survive., Trusting the mixin's deleted_at-in-key index for real partial uniqueness — rejected where correctness matters; a custom partial-unique SQL migration is required.
**Forced by:** A ~70-table multi-tenant schema needing uniform id/namespace/audit/soft-delete columns across every domain.
**Enables:** Adding an entity by composing mixins and running make generate + atlas diff; consistent namespace scoping and soft-delete semantics everywhere.

### Two coexisting HTTP surfaces: legacy v1 httptransport drivers + thin AIP v3 delegators, both over the same domain services
**Chosen:** v1: openmeter/server/router validates each request against the embedded OpenAPI 3 spec (kin-openapi openapi3filter) and wires per-domain httpdriver packages, each composing httptransport.NewHandler[Request,Response](decode, service-op, encode) with a typed errorEncoder that maps domain errors to RFC7807 problem documents. v3: api/v3/server/routes.go implements the oapi-codegen ServerInterface as thin methods delegating to per-resource handlers (api/v3/handlers/*), with explicitly Unimplemented / feature-gated operations and AIP query-parameter filtering (api/v3/filters). Both call the same domain services.
**Rationale:** v3 is the forward AIP-style surface; v1 cannot be dropped without breaking clients, so both surfaces front the identical domain services and share error rendering (commonhttp.HandleErrorIfTypeMatches, api/v3/apierrors). The generic httptransport.Handler[Req,Resp] gives v1 consistent decode→service→encode→error-encode plumbing; v3 leans on generated stubs plus thin delegators.
**Rejected:** Forcing a hard cutover from v1 to v3 — rejected; existing clients depend on v1, so the surfaces coexist over shared services., Duplicating domain logic per surface — rejected; both surfaces are thin transport adapters over the same services, so behavior cannot diverge., Per-handler ad-hoc error-to-status mapping — rejected for the ordered typed errorEncoder chain (errors.As short-circuit) so domain error types map to status codes consistently.
**Forced by:** An installed v1 client base plus a new AIP-style API direction, both needing the same domain behavior.
**Enables:** Migrating endpoints to v3 incrementally while v1 keeps working, with one set of domain services and one error-mapping convention behind both.

### Accumulating Validate() returning NillableGenericValidationError as the uniform input-contract gate
**Chosen:** Validate() methods collect issues into var errs []error, wrap each with field context (fmt.Errorf("field: %w", err)), and return models.NewNillableGenericValidationError(errors.Join(errs...)) so a nil join yields nil; single-field checks use models.NewGenericValidationError(...) directly (pkg/models/errors.go, openmeter/app/marketplace.go). Service and adapter Config structs also carry Validate() and constructors reject invalid config at New().
**Rationale:** Validation must report all problems at once (not fail-fast) and surface as a single 400/ValidationIssue at the HTTP boundary. Centralizing on NewNillableGenericValidationError + errors.Join makes the nil-vs-error contract uniform across every domain and lets the error encoders map one error type to 400. AGENTS.md mandates this shape for Validate() methods. pkg/models is the 229-in-edge dependency magnet that carries it.
**Rejected:** Returning on the first invalid field — rejected; callers and API consumers want all issues at once., A bespoke validation library per domain — rejected; one shared error aggregation type keeps HTTP error mapping uniform., Validating only at the HTTP layer — rejected; service/adapter Config.Validate() catches misconfiguration at construction, before the binary serves traffic.
**Forced by:** Multiple API surfaces and many domains all needing a consistent multi-issue validation contract that maps to one HTTP status.
**Enables:** Aggregated 400 ValidationIssue responses, fail-at-construction config validation, and one error type for the encoders to match.

## Trade-offs Accepted

- **Accepted:** Context-ambient transactions: the active tx is carried on context.Context and rebound by entutils.TransactingRepo, not visible in method signatures.
  - *Benefit:* Cross-domain service composition can be atomic over one Ent client without threading a tx parameter through every interface; subscription→billing→charges→ledger commit together.
  - *Caused by:* Transaction-aware Ent repository (TransactingRepo / HijackTx / WithTx)
  - *Violation signal:* context.Background()
  - *Violation signal:* context.TODO()
  - *Violation signal:* func(... db *entdb.Client ...) without entutils.TransactingRepo wrapper
  - *Violation signal:* passing *entdb.Client into a helper that writes
  - *Violation signal:* bypassing WithTx / Self()
- **Accepted:** Advisory locks must run inside a real Postgres transaction and must key on a globally-unique id; misuse (no tx, or keying on a non-unique column) is only caught at runtime.
  - *Benefit:* Per-customer serialization of multi-row subscription/billing/charge operations across replicas, with the lock auto-released on commit/rollback (no orphan locks).
  - *Caused by:* Postgres transaction-scoped advisory locking (lockr)
  - *Violation signal:* lockr.NewKey on customer key
  - *Violation signal:* LockForTX outside transaction.Run
  - *Violation signal:* sync.Mutex for cross-replica serialization
  - *Violation signal:* pg_advisory_lock (session-scoped, not xact)
  - *Violation signal:* locking on a namespace-non-unique column
- **Accepted:** Six binaries over one module: shared code is convenient but a change to a high-fan-in magnet (pkg/models 229 in-edges, productcatalog 104, customer 103) ripples across every binary, and all binaries must be rebuilt/redeployed together.
  - *Benefit:* One codebase, one Ent client, one transaction boundary for cross-domain atomicity; workers scale and fail independently while sharing types.
  - *Caused by:* Multi-binary Go control plane with layered service/adapter domains, code-generated API contract, and an event-time usage-metering data plane
  - *Violation signal:* go.mod split per binary
  - *Violation signal:* duplicating pkg/models types per service
  - *Violation signal:* separate database per domain
  - *Violation signal:* distributed transaction / saga between binaries
- **Accepted:** TypeSpec-first codegen: any contract change requires regenerating OpenAPI, two Go server stubs, three SDKs, and goverter/goderive converters; generated files must never be hand-edited.
  - *Benefit:* Two API surfaces and three SDKs can never drift from one source of truth; contract drift becomes a build failure.
  - *Caused by:* TypeSpec-first API contract with full codegen fan-out (OpenAPI → oapi-codegen + 3 SDKs + goverter/goderive)
  - *Violation signal:* editing api/api.gen.go
  - *Violation signal:* editing api/openapi.yaml by hand
  - *Violation signal:* editing convert.gen.go
  - *Violation signal:* hand-writing SDK client methods
  - *Violation signal:* skipping make gen-api after .tsp change
- **Accepted:** A separate ClickHouse store and Kafka pipeline for usage events: two more stateful systems to operate, and ClickHouse tables are created idempotently at connector startup outside Atlas migration discipline.
  - *Benefit:* High-volume append-only usage ingest and aggregation scale independently of OLTP Postgres; meters and usage-based billing read aggregated quantities efficiently.
  - *Caused by:* Multi-binary Go control plane with layered service/adapter domains, code-generated API contract, and an event-time usage-metering data plane
  - *Violation signal:* storing usage events in Postgres
  - *Violation signal:* ALTER TABLE om_ events via Atlas
  - *Violation signal:* removing the sink-worker
  - *Violation signal:* joining RawEvent against Postgres tables
- **Accepted:** External-storage state machines put the source-of-truth status on the aggregate row; the FSM definition and the persisted status must stay in sync, and every legal transition must be declared as a Permit edge.
  - *Benefit:* Invoice/charge transitions are durable across requests, auditable, and illegal transitions are rejected at the single edge-set definition.
  - *Caused by:* Explicit finite state machines with external storage (qmuntal/stateless) for invoice and per-charge UBP lifecycles
  - *Violation signal:* mutating invoice.status directly
  - *Violation signal:* if/switch on status instead of Permit edges
  - *Violation signal:* NewStateMachine (in-memory) for a persisted aggregate
  - *Violation signal:* adding a status without a Permit edge
- **Accepted:** Feature subsystems are wired as concrete-or-noop at the DI seam (credits.enabled, webhooks), so disabling a feature requires every layer's provider to honor the flag (api/v3 handlers, customer ledger hooks, namespace provisioning).
  - *Benefit:* Whole subsystems (ledger, Svix webhooks) cleanly become no-ops without runtime conditionals scattered through service logic.
  - *Caused by:* Google Wire compile-time dependency injection in app/common, per-binary provider sets
  - *Violation signal:* if credits.enabled inside service method
  - *Violation signal:* constructing ledger adapters when credits disabled
  - *Violation signal:* Svix client built when webhooks disabled
  - *Violation signal:* missing noop provider for a gated subsystem
- **Accepted:** The (namespace,key,deleted_at) UniqueResourceMixin index only approximates partial uniqueness; entities needing true WHERE deleted_at IS NULL uniqueness must ship a hand-written IndexWhere SQL migration.
  - *Benefit:* Most entities get key-uniqueness for free from the mixin; the few that need strict partial uniqueness opt into a custom migration.
  - *Caused by:* Ent schema mixins as the cross-cutting column contract (ResourceMixin / IDMixin / NamespaceMixin / TimeMixin / AnnotationsMixin)
  - *Violation signal:* relying on UniqueResourceMixin for strict partial uniqueness
  - *Violation signal:* deleting customer.go IndexWhere migration
  - *Violation signal:* same-key create after soft-delete without partial index
- **Accepted:** Two HTTP surfaces (v1 + v3) front the same services indefinitely: every behavior change must be checked against both transport layers and both error-rendering paths.
  - *Benefit:* Existing v1 clients keep working while new endpoints land on the AIP v3 surface; no forced client migration.
  - *Caused by:* Two coexisting HTTP surfaces: legacy v1 httptransport drivers + thin AIP v3 delegators, both over the same domain services
  - *Violation signal:* v3 handler with its own domain logic
  - *Violation signal:* duplicating a service call in both surfaces with diverging behavior
  - *Violation signal:* dropping the v1 router
  - *Violation signal:* v3 endpoint not delegating to a shared domain service

## Out of Scope

- {'item': 'A bundled web UI / dashboard application — the frontend surface is published SDKs (JS/TS with a React context provider, Python) and a Cloud product, not an in-repo web app.', 'made_out_of_scope_by': 'Multi-binary Go control plane with code-generated API contract'}
- {'item': 'Distributed transactions / sagas across services — cross-domain consistency is achieved with one shared Ent client, one Postgres transaction, and lockr advisory locks, not a coordination protocol.', 'made_out_of_scope_by': 'Transaction-aware Ent repository (TransactingRepo / HijackTx / WithTx)'}
- {'item': 'Storing or migrating usage events in PostgreSQL — usage events live only in ClickHouse (append-only MergeTree), created idempotently at startup, outside Atlas/Ent migration discipline.', 'made_out_of_scope_by': 'Multi-binary control plane with an event-time usage-metering data plane'}
- {'item': 'Hand-written OpenAPI specs or hand-written SDK client methods — the contract is generated from TypeSpec; editing generated files is forbidden.', 'made_out_of_scope_by': 'TypeSpec-first API contract with full codegen fan-out'}
- {'item': 'Runtime/reflection-based dependency injection or dynamic plugin loading — DI is compile-time via Wire; app integrations register at wiring time into an in-memory map, not via runtime plugins.', 'made_out_of_scope_by': 'Google Wire compile-time dependency injection in app/common'}
- {'item': 'Hard deletes of control-plane rows — entities soft-delete via deleted_at so historical billing/usage references survive.', 'made_out_of_scope_by': 'Ent schema mixins as the cross-cutting column contract'}
- {'item': 'In-process/in-memory event delivery as the only mechanism — asynchronous cross-binary work goes through Kafka (Watermill CQRS); the in-process hook registry is reserved for synchronous/transactional cross-domain effects.', 'made_out_of_scope_by': 'Two-tier async messaging and the Service hook registry'}
- {'item': 'panic-based error handling and slog.Default() fallbacks in production code — AGENTS.md forbids panics in non-test paths and requires explicitly injected *slog.Logger.', 'made_out_of_scope_by': 'Accumulating Validate() returning NillableGenericValidationError as the uniform input-contract gate'}
- {'item': 'External payment/invoice engines hard-wired into the core — Stripe and custom-invoicing are pluggable marketplace apps registered through the app registry, and invoice formatting uses GOBL behind the billing service.', 'made_out_of_scope_by': 'In-memory map registry with type-asserted factory dispatch for app/marketplace integrations'}