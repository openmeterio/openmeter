## Communication Patterns

### Layered Domain Service/Adapter/Repository
- **When:** All business-logic domains under openmeter/<domain>/. The pattern applies whenever you need to separate persistence from orchestration.
- **How:** Each domain exposes a Go interface (e.g. billing.Service, customer.Service, notification.Service). A concrete service struct in <domain>/service/ holds business logic and calls an Adapter interface for all DB access. The Adapter interface is defined in <domain>/adapter.go alongside the Service interface and is implemented by Ent-backed structs in <domain>/adapter/ sub-packages. Service interfaces are composed of fine-grained sub-interfaces (ProfileService, InvoiceService, etc.) so callers depend on the smallest surface they need.

### Wire-based Dependency Injection
- **When:** Wiring together all runtime components of each binary. Each cmd/<binary>/wire.go declares a wire.Build call; provider sets live in app/common/.
- **How:** Google Wire generates cmd/<binary>/wire_gen.go at build time. Reusable provider sets (e.g. common.Billing, common.Notification, common.LedgerStack) are declared as wire.NewSet() in app/common/ files and compose individual factory functions. Wire resolves the dependency graph automatically. Each Application struct in cmd/<binary>/wire.go lists every needed service as a field and Wire auto-wires them.

### Watermill Message Bus (Kafka-backed publish/subscribe)
- **When:** Async domain-event delivery between services. Used for subscription lifecycle events, billing invoice advance events, ingest flush events, and balance-worker recalculation events.
- **How:** openmeter/watermill/eventbus/eventbus.go wraps the Watermill CQRS EventBus. Events are Go structs implementing marshaler.Event (carrying EventName() and EventMetadata()). The Publisher routes events to Kafka topics by inspecting event-name prefixes. Worker processes subscribe via Watermill's Kafka subscriber and dispatch to typed handlers registered in a NoPublishingHandler (openmeter/watermill/grouphandler/grouphandler.go). Handlers are closures matching event type via marshaler; unmatched events are silently ignored. Three topic channels exist: IngestEventsTopic, BalanceWorkerEventsTopic, SystemEventsTopic.

### ServiceHook Registry (observer chain)
- **When:** Cross-domain lifecycle callbacks without circular imports. Used by customer.Service, subscription.Service, and billing.Service (StandardInvoiceHooks).
- **How:** pkg/models/servicehook.go defines a generic ServiceHook[T] interface (PreUpdate, PreDelete, PostCreate, PostUpdate, PostDelete) and a thread-safe ServiceHookRegistry[T] that fans out to all registered hooks. Loop prevention: a context key unique to the registry prevents re-entrant invocations. Subscription uses a separate SubscriptionCommandHook interface (BeforeCreate, AfterCreate, etc.) in openmeter/subscription/hook.go that is registered via Service.RegisterHook().

### Customer RequestValidator Registry
- **When:** Pre-request validation for customer mutation operations where billing or subscription constraints must be checked before the customer is modified or deleted.
- **How:** openmeter/customer/requestvalidator.go defines RequestValidator interface with ValidateDeleteCustomer, ValidateCreateCustomer, ValidateUpdateCustomer. A requestValidatorRegistry fans out to all registered validators using errors.Join. billing.validators/customer constructs a billingcustomer.Validator that checks billing pre-conditions and registers it via customerService.RegisterRequestValidator().

### Invoice State Machine (stateless library)
- **When:** Driving the invoice lifecycle transitions for StandardInvoice and for charge-level state machines.
- **How:** openmeter/billing/service/stdinvoicestate.go builds a *stateless.StateMachine instance from sync.Pool with external storage bound to the InvoiceStateMachine struct's Invoice.Status field. Transitions trigger actions (DB save, event publish). openmeter/billing/charges/statemachine/machine.go provides a generic Machine[CHARGE, BASE, STATUS] wrapping stateless for charge lifecycles. Both use FireAndActivate + AdvanceUntilStateStable to run all allowed auto-transitions.

### LineEngine Plugin Registry
- **When:** Dispatching billing line calculation to the correct engine (standard invoice, charge flatfee, charge usagebased, charge creditpurchase) based on LineEngineType discriminator.
- **How:** billing.LineEngine interface declares GetLineEngineType() + OnCollectionCompleted + OnStandardInvoiceCreated + OnInvoiceIssued + OnPaymentAuthorized + OnPaymentSettled. billing.Service exposes RegisterLineEngine / DeregisterLineEngine. billingservice.engineRegistry (openmeter/billing/service/lineengine.go) stores a map[LineEngineType]LineEngine under a RWMutex. Each charge type implements its own Engine and registers it at startup.

### App Factory / Registry (external billing app protocol)
- **When:** Plugging Stripe, Sandbox, and CustomInvoicing billing apps into the billing state machine without hardcoding them.
- **How:** openmeter/app/registry.go defines AppFactory (NewApp, UninstallApp) and RegistryItem (Listing + Factory). App service (openmeter/app/service.go) exposes RegisterMarketplaceListing / InstallMarketplaceListing. Installed apps must implement openmeter/billing/app.go InvoicingApp interface (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice). Optionally implement InvoicingAppPostAdvanceHook and InvoicingAppAsyncSyncer for post-advance callbacks and async sync support. app.GetApp() type-asserts installed App to InvoicingApp at runtime.

### Entutils TransactingRepo (context-propagated transactions)
- **When:** All DB adapter methods that must run inside a caller-supplied transaction or start their own.
- **How:** pkg/framework/entutils/transaction.go defines TransactingRepo[R,T] and TransactingRepoWithNoValue[T]. They read the *TxDriver from context (GetDriverFromContext). If one is found, the adapter's WithTx(ctx, tx) method creates a txClient from the raw Ent config. If none is found the adapter runs on its Self() client and starts a new transaction via the Creator interface. Savepoints are used for nested calls so partial rollback is possible.

### Locker (pg_advisory_xact_lock)
- **When:** Distributed mutual exclusion for per-customer billing operations to prevent concurrent invoice generation races.
- **How:** pkg/framework/lockr/locker.go calls pg_advisory_xact_lock($1) with a CRC64-based hash of the lock key. Requires an active Postgres transaction in context (fails otherwise). billing.Service.WithLock acquires the lock per CustomerID before any invoice or charge mutation.

### httptransport Operation/Handler pattern
- **When:** HTTP endpoint handlers in domain httpdriver packages that separate request decoding, business logic, and response encoding.
- **How:** pkg/framework/transport/httptransport/handler.go defines Handler[Request, Response] wrapping an operation.Operation[Request, Response] (a func(ctx, req) (resp, err)). Decoding and encoding are injected via RequestDecoder and ResponseEncoder function types. ErrorEncoders form a chain; the first one to return true short-circuits. The Chain method wraps the operation with operation.Middleware for cross-cutting concerns.

### Subscription Sync Reconciler
- **When:** Crash-recovery for the event-driven billing sync: periodically re-syncs subscriptions that may have missed their events.
- **How:** openmeter/billing/worker/subscriptionsync/reconciler/reconciler.go iterates customers/subscriptions in windows and calls subscriptionsync.Service.SynchronizeSubscriptionAndInvoiceCustomer for each. BillingWorker.Worker.AddHandler allows external code to attach additional event handlers after construction.

### Namespace multi-tenancy via context + static decoder
- **When:** Tenant isolation in every API handler. All service calls require a namespace string.
- **How:** openmeter/namespace/namespace.go defines Manager which fans out CreateNamespace / DeleteNamespace to all registered Handler implementations. Each HTTP server uses namespacedriver.StaticNamespaceDecoder to inject the default namespace for self-hosted deployments. Namespace enforcement is structural (namespace field on every input type) rather than middleware-enforced.

### Chi middleware chain + OpenAPI validator
- **When:** HTTP request validation and authentication on the v1 REST API.
- **How:** openmeter/server/server.go builds a Chi router with middleware stacks: RealIP, RequestID, request logger, Recoverer, then Authenticator middleware (portal token or API key), then oapi-codegen's OapiRequestValidatorWithOptions against the generated swagger spec (authentication delegated to Nooop so it runs after auth middleware). The v3 API uses a parallel stack with oasmiddleware.ValidateRequest for schema validation.

### Noop implementations for optional features
- **When:** When a feature is disabled (credits.enabled=false, svix not configured) production code still wires a real interface, but backed by no-op implementations.
- **How:** app/common/ledger.go and app/common/notification.go check configuration flags and return noop structs (ledgernoop.Ledger{}, ledgernoop.AccountResolver{}, webhooknoop.New()) so the rest of the wiring does not need to branch. app/common/ledger.go also uses type assertion against noop types to skip namespace handler registration.

### Sink Worker (Kafka → ClickHouse batch flush)
- **When:** High-throughput ingestion path: raw CloudEvents arrive via Kafka, are buffered, deduplicated (Redis or in-memory), and batch-inserted into ClickHouse.
- **How:** openmeter/sink/sink.go consumes Kafka partitions via confluent-kafka-go. A SinkBuffer accumulates messages; flush is triggered by MinCommitCount or MaxCommitWait. After flush, openmeter/sink/storage.go ClickHouseStorage.BatchInsert writes to ClickHouse via streaming.Connector.BatchInsert. FlushEventHandlers (openmeter/sink/flushhandler/) are called post-flush for downstream notifications.

### ValidationIssue structured error propagation
- **When:** Domain-level validation errors that must carry field paths, severity (critical/warning), component names, and arbitrary attributes through service layer boundaries.
- **How:** pkg/models/validationissue.go defines ValidationIssue as a value type implementing error. ValidationIssues ([]ValidationIssue) can be converted to/from joined errors. The HTTP layer (commonhttp.HandleIssueIfHTTPStatusKnown) reads the httpStatusCodeErrorAttribute attribute to produce the correct HTTP status. Billing-domain uses billing.ValidationIssue (a different struct) in the invoicing state machine for storing issues on the invoice record.

### RFC 7807 Problem Details HTTP response
- **When:** All error responses from the REST API.
- **How:** pkg/models/problem.go defines Problem interface and StatusProblem struct serialized to application/problem+json. NewStatusProblem reads the request-id from context and sets it as instance URI. Context cancellation is mapped to 408. InternalServerError suppresses the detail string. Extensions map is used by the validation error encoder to attach validationErrors array.

## Integrations

| Service | Purpose | Integration point |
|---------|---------|-------------------|
| PostgreSQL (Ent ORM + pgx + Atlas) | Primary relational store for all domain entities: billing profiles, invoices, customers, subscriptions, entitlements, notification channels, ledger accounts, etc. | `openmeter/ent/schema/ (source of truth); generated code in openmeter/ent/db/; Atlas migrations in tools/migrate/migrations/. Accessed via entdb.Client injected through Wire. Transactions managed by pkg/framework/entutils.TransactingRepo.` |
| ClickHouse | Append-only analytics store for raw usage events; queried for meter aggregations (count, sum, max, unique_count) via SQL builders. | `openmeter/streaming/clickhouse/ – event_query.go and meter_query.go build ClickHouse SQL via sqlbuilder. Connector interface in openmeter/streaming/connector.go. ClickHouseStorage in openmeter/sink/storage.go for batch inserts.` |
| Kafka (confluent-kafka-go + Watermill-Kafka) | Durable event bus for domain events (subscription lifecycle, invoice advance, ingest flush notifications, balance recalculation) and raw usage event ingestion. | `openmeter/watermill/driver/kafka/ – Publisher and Subscriber wrappers. Topic provisioning via KafkaTopicProvisioner in app/common. confluent-kafka-go used directly in openmeter/sink/sink.go for high-throughput ingest consumer.` |
| Redis | Optional deduplication store for ingest events (preventing double-counting on retry). | `openmeter/dedupe/redisdedupe/redisdedupe.go – Redis-backed Deduplicator. In-memory fallback in openmeter/dedupe/memorydedupe/.` |
| Svix | Outbound webhook delivery for notification events (entitlement balance thresholds, invoice events). | `openmeter/notification/webhook/svix/svix.go – Svix API client wrapper. Registered event types passed to Svix application; messages routed by channel filter (NullChannel sentinel prevents unfiltered delivery). Handler interface in openmeter/notification/webhook/handler.go with noop fallback when Svix is unconfigured.` |
| Stripe (via app/stripe) | Invoice syncing (upsert draft, finalize, collect payment) and customer sync for billing-enabled namespaces. | `openmeter/app/stripe/app.go implements billing.InvoicingApp (UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice). Stripe REST client in openmeter/app/stripe/client/. App registered via app.Service.RegisterMarketplaceListing at startup.` |
| Sandbox invoicing app | No-op invoicing app used in development/testing to drive invoice state machine without external dependencies. | `openmeter/app/sandbox/app.go implements billing.InvoicingApp + InvoicingAppPostAdvanceHook. Registered as marketplace listing with type AppTypeSandbox.` |
| CustomInvoicing app | Webhook-driven invoicing app allowing external systems to receive invoice payloads and async-confirm sync completion. | `openmeter/app/custominvoicing/ – App implements InvoicingApp + InvoicingAppAsyncSyncer; factory in custominvoicing/factory.go.` |
| GOBL (invopop/gobl) | Currency and numeric type library used throughout billing and subscription for currency-safe arithmetic and ISO 4217 currency code validation. | `Imported as github.com/invopop/gobl/currency and github.com/invopop/gobl/num in productcatalog, subscription, billing, cost, and currencies packages.` |
| OpenTelemetry | Distributed tracing and metrics across all services. | `trace.Tracer injected via Wire into service constructors. OTel metric.Meter used in grouphandler (watermill.grouphandler.*) and sink worker. app/common/telemetry.go bootstraps OTLP exporters.` |

## Pattern Selection Guide

| Scenario | Pattern | Rationale |
|----------|---------|-----------|
| Adding a new domain capability (e.g. a new billing sub-feature) | Layered Domain Service/Adapter/Repository | Define a new sub-interface in <domain>/service.go, implement it in <domain>/service/<file>.go, add corresponding adapter methods to <domain>/adapter.go and implement in <domain>/adapter/. Wire together in app/common/<domain>.go. |
| Triggering side-effects on domain object lifecycle (e.g. sync billing on subscription update) | ServiceHook Registry or SubscriptionCommandHook | Avoids circular imports. Billing registers a hook into subscription.Service.RegisterHook() during wiring, not at compile time. |
| Pre-validating a customer mutation from another domain | Customer RequestValidator Registry | billing/validators/customer implements RequestValidator and registers via customerService.RegisterRequestValidator(), keeping billing constraints out of the customer package. |
| Processing async domain events between services | Watermill Message Bus (NoPublishingHandler + GroupEventHandler) | Events are published to Kafka via eventbus.Publisher and consumed by worker processes that register typed closures. Unknown event types are silently dropped, so workers are tolerant of schema evolution. |
| Invoice or charge lifecycle transitions | Invoice State Machine (stateless library) | The state machine enforces valid transitions and fires actions (DB save, event publish, external app calls) atomically within the transition. Generic Machine[CHARGE,BASE,STATUS] is reused for all charge types. |
| Crash-recovery for event-driven billing sync | Subscription Sync Reconciler | Event loss is mitigated by a periodic scan of all active subscriptions; SynchronizeSubscriptionAndInvoiceCustomer is idempotent. |
| Multiple billing backend implementations (Stripe, Sandbox, CustomInvoicing) | App Factory / Registry + InvoicingApp interface | New billing backends implement billing.InvoicingApp and register a factory with app.Service.RegisterMarketplaceListing(). No billing service code changes needed. |
| Disabling a subsystem (credits off, no Svix) | Noop implementations for optional features | Wire provider functions check config flags and return noops; the rest of the DI graph is unaffected, avoiding nil checks scattered through business logic. |
| Distributed lock for per-customer serialization | Locker (pg_advisory_xact_lock) | Advisory locks are transactional and released automatically on commit/rollback, avoiding stale lock cleanup. lockr.Locker requires an active Postgres transaction in context. |
| HTTP error response to client | RFC 7807 Problem Details + GenericErrorEncoder chain | Domain errors (GenericNotFoundError → 404, GenericValidationError → 400, etc.) are matched by type in GenericErrorEncoder. ValidationIssues with explicit HTTP status attributes are handled first. All errors render as application/problem+json. |
| Batch usage event ingestion | Sink Worker (Kafka → ClickHouse batch flush) | High-throughput events flow Kafka → SinkBuffer → ClickHouse in micro-batches. Redis deduplication prevents double-counting on consumer restarts. |
| Outbound webhook notifications | Svix integration via webhook.Handler interface | Svix handles fan-out, retry, signature verification, and delivery status. The noop implementation runs in tests or when Svix is unconfigured. |

## Quick Pattern Lookup

- **new domain feature** -> Layered Domain Service/Adapter/Repository in openmeter/<domain>/
- **lifecycle side-effects** -> ServiceHookRegistry (models.ServiceHook) or SubscriptionCommandHook
- **pre-mutation validation across domains** -> Customer RequestValidator Registry
- **async domain events** -> Watermill NoPublishingHandler + GroupEventHandler on SystemEventsTopic
- **invoice/charge state transitions** -> stateless-backed InvoiceStateMachine or generic Machine[CHARGE,BASE,STATUS]
- **new billing backend** -> Implement billing.InvoicingApp + AppFactory, register via app.Service.RegisterMarketplaceListing
- **optional feature disabled** -> Return noop implementation in Wire provider function when config flag is false
- **per-customer serialization** -> billing.Service.WithLock → lockr.Locker.LockForTX (pg advisory lock in tx)
- **DB operations in transactions** -> entutils.TransactingRepo / TransactingRepoWithNoValue
- **HTTP handler** -> httptransport.NewHandler with RequestDecoder + Operation + ResponseEncoder
- **batch usage ingestion** -> confluent-kafka-go consumer in Sink worker, ClickHouseStorage.BatchInsert
- **outbound webhooks** -> notification.EventHandler → webhook.Handler (Svix or noop)
- **DI wiring** -> Google Wire: wire.NewSet in app/common/, wire.Build in cmd/<binary>/wire.go

## Key Decisions

### TypeSpec as the single source of truth for the HTTP API
**Chosen:** API is authored in TypeSpec under api/spec/packages/ (aip/ for v3, legacy/ for v1) and compiled to OpenAPI YAMLs, Go server stubs (api/api.gen.go, api/v3/api.gen.go), Go client (api/client/go/client.gen.go), and the JavaScript/Python SDKs. `make gen-api` regenerates specs and SDKs; `make generate` then regenerates Go code that depends on them.
**Rationale:** One spec feeds three SDKs and two server versions. Drift between languages is eliminated and the two-step pipeline (TypeSpec -> OpenAPI -> generators) is deterministic.
**Rejected:** Hand-written OpenAPI, Code-first (generating OpenAPI from Go handlers), gRPC/Protobuf (would not match REST-first customer expectations and Svix webhook payloads)
**Forced by:** Multiple SDK languages (Go, JavaScript, Python) + dual API versions (v1 + v3)
**Enables:** Contract-stable breaking-change detection, shared request/response validation via kin-openapi middleware, parallel SDK evolution

### Ent ORM + Atlas migrations as the schema pipeline
**Chosen:** Ent schemas under openmeter/ent/schema/ are the single source of truth for DB shape. `make generate` produces openmeter/ent/db/. `atlas migrate --env local diff <name>` produces timestamped up/down SQL into tools/migrate/migrations/ plus atlas.sum. Migrations run at server startup when postgres.autoMigrate is set.
**Rationale:** Atlas diffs the Ent schema against the migration history to produce deterministic SQL; Ent gives typed queries and auto-wires relations. Combined they prevent hand-rolled migration drift while keeping Go-native schema definitions.
**Rejected:** sqlc (schema still hand-rolled SQL), GORM (weaker typing, no native schema diff), Raw golang-migrate only (no typed Go entities)
**Forced by:** Billing correctness + multi-tenant schema invariants that need compile-time checks
**Enables:** Typed relations across ~60 domain entities, deterministic migration review in PRs

### Multi-binary deployment sharing a single domain package tree
**Chosen:** Seven cmd/* entry points each call their own Wire-generated initializeApplication from app/common. Domain packages under openmeter/ have no dependency on cmd/* or app/common.
**Rationale:** Ingest throughput, balance recalculation, billing advancement, and notification dispatch have different scaling profiles and failure characteristics. Splitting them into independent binaries while sharing types keeps operational flexibility without fracturing the domain model.
**Rejected:** Single monolith binary with goroutine workers (couples failure domains), Independent microservices with separate repos (fragments shared types and migrations)
**Forced by:** High-volume per-tenant usage metering with strict billing correctness and independent scaling needs for ingest vs invoicing
**Enables:** Horizontal scaling of sink-worker independent of billing-worker; isolated deploy cadence

### Kafka + Watermill as the asynchronous event backbone
**Chosen:** openmeter/watermill wraps confluent-kafka-go behind a Watermill Publisher with three named topics (ingest events, system events, balance worker events). Producers call eventbus.Publisher; consumers (billing-worker, balance-worker, notification-service, sink-worker) subscribe via watermill/router with OTel tracing.
**Rationale:** Topic isolation matches the worker topology. Watermill gives per-message retries, dead-lettering, and a uniform router abstraction across four independent binaries. confluent-kafka-go is chosen for librdkafka throughput despite the dynamic-build cost.
**Rejected:** Pure confluent-kafka-go without Watermill (loses uniform routing), NATS/Redis Streams (lower replay semantics, less ecosystem for billing patterns), Postgres LISTEN/NOTIFY (wrong durability/backpressure model for ingest volumes)
**Forced by:** Ingest bursts + cross-worker side-effects (ingest -> balance recalc -> notification)
**Enables:** Replay, backpressure, decoupled producer/consumer evolution

### Google Wire DI concentrated in app/common
**Chosen:** Every domain package exposes plain constructors; all wiring lives in app/common/*.go with Wire provider sets (billing.go, customer.go, entitlement.go, subscription.go, database.go, app.go, notification.go, openmeter_server.go, openmeter_billingworker.go, etc.). cmd/<binary>/wire.go lists the sets; cmd/<binary>/wire_gen.go is generated.
**Rationale:** Declarative wiring with compile-time verification. Changing a constructor signature causes the Wire regeneration step to surface missing providers before runtime.
**Rejected:** Manual DI in each cmd (duplicated graphs), Reflection DI (runtime failure mode incompatible with billing correctness), Functional options everywhere (no compile-time proof that the full app is buildable)
**Forced by:** Seven binaries sharing dozens of providers
**Enables:** Single edit point to add a new dependency to any binary

### credits.enabled feature flag enforced at multiple independent wiring layers
**Chosen:** When credits.enabled is false, app/common wires ledger account services/resolvers to noop implementations AND v3 server ledger-backed customer credit handlers AND customer ledger hooks AND namespace default-account provisioning must each be guarded independently.
**Rationale:** Credits touches several unrelated call graphs (HTTP handlers, customer hooks, namespace provisioning, charge creation). No single choke point can gate it.
**Rejected:** Single global flag branch (does not stop ledger writes originating from hooks), Dynamic runtime check inside ledger.Ledger (performance + correctness risk; writes still attempted)
**Forced by:** Cross-cutting nature of credit accounting
**Enables:** Credits-disabled tenants genuinely skip ledger writes when every layer is guarded correctly

### Charges realization with explicit TransactingRepo discipline
**Chosen:** openmeter/billing/charges owns usage-based, flat-fee, and credit-purchase charges. Lifecycle is driven through charges.Service.Create / AdvanceCharges / ApplyPatches. Adapter helpers that accept a raw *entdb.Client must wrap their body with entutils.TransactingRepo / TransactingRepoWithNoValue so the ctx-bound transaction is honored.
**Rationale:** Charge advancement mixes reads, realization runs, and ledger-bound writes. Without explicit TransactingRepo rebinding in every helper, a helper can fall off the transaction and cause partial writes under concurrency.
**Rejected:** Pass *entdb.Tx explicitly (leaks tx plumbing into every call site), Global tx middleware (cannot enforce per-helper without compiler help)
**Forced by:** Ent transactions carried implicitly in ctx
**Enables:** Deterministic multi-step charge advancement without partial writes

### Dynamic build tag for librdkafka (GO_BUILD_FLAGS=-tags=dynamic)
**Chosen:** All binaries and test invocations build with -tags=dynamic so confluent-kafka-go links against system librdkafka.
**Rationale:** Dynamic linking cuts binary size and speeds tests; matches production librdkafka availability. CI uses nix develop --impure .#ci to pin the toolchain.
**Rejected:** Static link (pure go kafka client is lower throughput; static librdkafka is hard to ship), Sarama (lower Kafka-protocol coverage at the throughput OpenMeter needs)
**Forced by:** High-volume ingest + tests that touch Kafka
**Enables:** Consistent library behavior across dev, CI, and production images

## Trade-offs Accepted

- **Accepted:** Ent-generated query friction (longer compile on schema change, boilerplate in openmeter/ent/db/)
  - *Benefit:* Compile-time-checked relations across ~60 entities, automatic Atlas diffing, no runtime schema surprises
  - *Caused by:* Ent ORM + Atlas migration pipeline
  - *Violation signal:* Hand-written SQL added alongside Ent queries
  - *Violation signal:* Direct edits inside openmeter/ent/db/
  - *Violation signal:* New table created without corresponding openmeter/ent/schema/*.go
- **Accepted:** Multi-binary orchestration cost (Docker images, Helm values, local docker-compose) and operational complexity
  - *Benefit:* Horizontal scaling of sink-worker / balance-worker / billing-worker independent of HTTP traffic; fault isolation per binary
  - *Caused by:* Multi-binary deployment of cmd/server, cmd/billing-worker, cmd/balance-worker, cmd/sink-worker, cmd/notification-service
  - *Violation signal:* Business logic added inside cmd/*/main.go
  - *Violation signal:* Workers added without matching app/common/openmeter_*worker.go Wire set
  - *Violation signal:* Cross-binary dependencies introduced through shared global state instead of Kafka topic
- **Accepted:** Two-step regen cadence: TypeSpec -> `make gen-api`, then `make generate` for Go server/wire/ent
  - *Benefit:* Stable cross-language SDK contracts; contract drift is impossible as long as both steps run
  - *Caused by:* TypeSpec -> OpenAPI -> oapi-codegen + Wire/Ent/Goverter stack
  - *Violation signal:* Hand-edits in *.gen.go files
  - *Violation signal:* PRs touching api/spec/ without regenerated api/openapi.yaml or api/v3/openapi.yaml
  - *Violation signal:* Client SDKs under api/client/** drifting from api/spec/
- **Accepted:** librdkafka C dependency (dynamic linking at build time)
  - *Benefit:* High-throughput Kafka producer/consumer, consistent semantics with Kafka ecosystem
  - *Caused by:* confluent-kafka-go + GO_BUILD_FLAGS=-tags=dynamic
  - *Violation signal:* go test invocations without -tags=dynamic (link errors)
  - *Violation signal:* CI images missing librdkafka
  - *Violation signal:* Attempts to switch to a pure-Go Kafka client
- **Accepted:** Sequential Atlas migration filenames (timestamped .up.sql/.down.sql) can collide on long-running branches
  - *Benefit:* Deterministic, reviewable SQL migrations with atlas.sum verifying the chain
  - *Caused by:* atlas migrate --env local diff naming + atlas.sum chain hashing
  - *Violation signal:* Multiple branches producing same-timestamp migrations
  - *Violation signal:* atlas.sum merge conflicts
  - *Violation signal:* Attempts to edit existing migrations after they land

## Out of Scope

- Frontend UI (frontend_ratio = 0; React only appears in the generated JavaScript SDK under api/client/javascript/)
- Business-level auth / identity provider (portal tokens handle end-customer scoping; core tenant auth is outside this repo)
- Managed hosting control plane (config.cloud.yaml and api/openapi.cloud.yaml exist but the hosted-platform logic is not in this monorepo)