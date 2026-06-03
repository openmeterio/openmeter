## Components

### cmd (binary entrypoints)
- **Location:** `cmd`
- **Responsibility:** Six binary main packages: cmd/server (HTTP API), cmd/billing-worker, cmd/balance-worker, cmd/sink-worker (Kafka/ClickHouse), cmd/notification-service (Svix webhooks), and cmd/jobs (Cobra admin CLI). Each main.go parses Viper config, calls a Wire-generated initializeApplication, runs DB migration, registers namespace handlers, and starts an oklog/run.Group lifecycle. Each has a paired wire.go (hand-written provider list) and wire_gen.go (generated). No business logic lives here.
- **Depends on:** app/common, app/config, openmeter/server, openmeter/billing, openmeter/namespace
- **Key interfaces:** `initializeApplication` (initializeApplication), `Application` (Run)

### app/common (Wire DI layer)
- **Location:** `app/common`
- **Responsibility:** Houses all Google Wire provider sets and constructor functions. One file per domain area (billing.go, customer.go, ledger.go, charges.go, etc.); openmeter_<binary>.go files (openmeter_server.go, openmeter_billingworker.go, openmeter_balanceworker.go, openmeter_sinkworker.go, openmeter_notification.go) define the composite per-binary sets. Registers cross-domain ServiceHooks and RequestValidators as construction side-effects to avoid circular imports. Returns noop implementations (not nil) when features are disabled (credits.enabled=false), guarded independently across ledger, customer hooks, charges, and namespace provisioning.
- **Depends on:** openmeter/*, pkg/*, app/config
- **Key interfaces:** `Wire provider sets` (NewBillingRegistry, NewCustomerLedgerServiceHook, NewLedgerNamespaceHandler)

### app/config (Viper configuration)
- **Location:** `app/config`
- **Responsibility:** Viper-based configuration structs, defaults, and Validate() methods for every application concern (aggregation, apps, balanceworker, billing, billingworker, credits, customer, dedupe, entitlements, events, etc.). Single shared config.Configuration type used by all binaries; SetViperDefaults is the single registration point calling each Configure* sub-function.
- **Depends on:** openmeter/meter, pkg/errorsx, pkg/models
- **Key interfaces:** `Configuration` (Validate, SetViperDefaults)

### openmeter/billing (billing domain)
- **Location:** `openmeter/billing`
- **Responsibility:** Largest domain package (487 files). Defines the composite billing.Service interface (composed of ProfileService, InvoiceService, GatheringInvoiceService, StandardInvoiceService, InvoiceLineService, LineEngineService, SplitLineGroupService and more) and billing.Adapter. Owns the InvoiceLine tagged-union (private discriminator, NewStandardInvoiceLine/NewGatheringInvoiceLine constructors). service/ drives the invoice lifecycle via a stateless.StateMachine pooled in sync.Pool; adapter/ is the Ent implementation; httpdriver/ exposes v1 HTTP. Sub-packages: charges/ (Charge/ChargeIntent tagged-union + generic charge state machine), worker/ (advance, collect, subscriptionsync, asyncadvance loops).
- **Depends on:** openmeter/app, openmeter/customer, openmeter/productcatalog, openmeter/meter, openmeter/streaming, openmeter/watermill/eventbus, pkg/framework/lockr, pkg/framework/entutils
- **Key interfaces:** `billing.Service` (CreateProfile, UpsertInvoice, AdvanceInvoice, RegisterLineEngine), `billing.InvoicingApp` (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice)

### openmeter/customer (customer domain)
- **Location:** `openmeter/customer`
- **Responsibility:** Customer lifecycle (ListCustomers, ListCustomerUsageAttributions, CreateCustomer, UpdateCustomer, DeleteCustomer, GetCustomer, GetCustomerByUsageAttribution) with soft-delete via DeletedAt. service/service.go wraps every mutation in transaction.Run so hooks and event publishes are atomic. Exposes two extension registries: RequestValidatorRegistry (pre-mutation cross-domain blocking guards, errors.Join fan-out) and ServiceHooks[Customer] (post-lifecycle callbacks). Sub-package service/hooks/ holds entitlementvalidator and subjectcustomer hooks.
- **Depends on:** openmeter/streaming, openmeter/ent/db, openmeter/watermill/eventbus, pkg/models, pkg/framework/entutils, pkg/framework/transaction
- **Key interfaces:** `customer.Service` (ListCustomers, CreateCustomer, DeleteCustomer, GetCustomer, UpdateCustomer, RegisterHooks, RegisterRequestValidator), `customer.RequestValidator` (ValidateDeleteCustomer, ValidateCreateCustomer, ValidateUpdateCustomer)

### openmeter/entitlement (entitlement domain)
- **Location:** `openmeter/entitlement`
- **Responsibility:** Feature entitlement management across metered (credit grant burn-down), boolean, and static sub-types. Service composes all sub-types with scheduling, override, supersede, and balance history. Acquires pg_advisory_lock per customer before multi-row mutations. Sub-package balanceworker/ is a Kafka-driven worker recalculating balances on lifecycle events using LRU caches and a high-watermark filter to skip redundant ClickHouse queries.
- **Depends on:** openmeter/productcatalog/feature, openmeter/customer, openmeter/streaming, openmeter/credit, openmeter/ent/db, openmeter/watermill/eventbus, pkg/framework/lockr
- **Key interfaces:** `entitlement.Service` (CreateEntitlement, OverrideEntitlement, ScheduleEntitlement, GetEntitlementValue, GetAccess, DeleteEntitlement, RegisterHooks), `balanceworker.Worker` (AddHandler, Run)

### openmeter/subscription (subscription domain)
- **Location:** `openmeter/subscription`
- **Responsibility:** Subscription lifecycle (Create, Update, Delete, Cancel, Continue, UpdateAnnotations) against a versioned plan-phase-RateCard model. Mutates an in-memory SubscriptionSpec exclusively through the AppliesToSpec patch interface (ApplyTo). A workflow service orchestrates higher-level operations (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon). Addon sub-service manages quantity-based addons.
- **Depends on:** openmeter/productcatalog, openmeter/customer, openmeter/entitlement, openmeter/ent/db, openmeter/watermill/eventbus, pkg/framework/lockr
- **Key interfaces:** `subscription.Service` (Get, GetView, List, Create, Update, Delete, Cancel, Continue, RegisterHook), `subscriptionworkflow.Service` (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity)

### openmeter/productcatalog (catalog domain)
- **Location:** `openmeter/productcatalog`
- **Responsibility:** Product catalog: features (meter-backed usage features), plans (versioned multi-phase rate-card collections), plan addons, addons, and rate cards (usage-based and flat-fee pricing). Each entity has its own Service interface, Ent adapter, and httpdriver package (161 files).
- **Depends on:** openmeter/meter, openmeter/ent/db, openmeter/watermill, pkg/models, pkg/pagination
- **Key interfaces:** `feature.FeatureConnector` (CreateFeature, UpdateFeature, ArchiveFeature, ListFeatures, GetFeature, ResolveFeatureMeters), `plan.Service` (ListPlans, CreatePlan, PublishPlan, ArchivePlan, NextPlan)

### openmeter/ledger (double-entry ledger)
- **Location:** `openmeter/ledger`
- **Responsibility:** Double-entry ledger for customer financial balances (FBO, Receivable, Accrued) and business accounts (Wash, Earnings, Brokerage). Transaction inputs are constructed exclusively via transactions.ResolveTransactions with typed template structs (enforcing debit=credit invariants). noop/ provides zero-value implementations wired when credits.enabled=false (135 files).
- **Depends on:** openmeter/customer, openmeter/ent/db, pkg/currencyx, pkg/framework/lockr, pkg/framework/entutils
- **Key interfaces:** `ledger.AccountResolver` (EnsureCustomerAccounts, GetCustomerAccounts, EnsureBusinessAccounts), `ledger.Ledger` (CommitGroup, QueryBalance)

### openmeter/credit (credit grants)
- **Location:** `openmeter/credit`
- **Responsibility:** Manages credit grants and balance snapshots for metered entitlements. CreditConnector (BalanceConnector + GrantConnector) is the public facade. engine/ computes grant burn-down without I/O. All effective times are truncated to Granularity (time.Minute) before storage or computation.
- **Depends on:** openmeter/streaming, openmeter/credit/grant, openmeter/credit/balance, openmeter/credit/engine, openmeter/ent/db, openmeter/watermill/eventbus
- **Key interfaces:** `credit.CreditConnector` (GetBalanceAt, GetBalanceForPeriod, ResetUsageForOwner, CreateGrant, VoidGrant)

### openmeter/app (marketplace apps)
- **Location:** `openmeter/app`
- **Responsibility:** Marketplace registry and runtime lifecycle for installed billing apps. AppFactory self-registers via RegisterMarketplaceListing in its constructor. Concrete implementations under stripe/, sandbox/, and custominvoicing/ embed AppBase and implement billing.InvoicingApp (103 files).
- **Depends on:** openmeter/customer, openmeter/secret, openmeter/billing, openmeter/ent/db, openmeter/watermill/eventbus
- **Key interfaces:** `app.Service` (RegisterMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, CreateApp, UninstallApp, EnsureCustomer), `app.App` (GetID, GetType, GetStatus, ValidateCapabilities, UpsertCustomerData, DeleteCustomerData)

### openmeter/notification (notification domain)
- **Location:** `openmeter/notification`
- **Responsibility:** Defines notification domain types (Channel, Rule, Event, EventPayload union, EventDeliveryStatus) and service interfaces (ChannelService + RuleService + EventService + FeatureService). EventHandler drives the dispatch + reconcile loop. consumer/ subscribes to the system events topic and delivers webhook payloads via Svix; webhook/ provides a noop fallback. A NullChannel sentinel prevents unfiltered delivery.
- **Depends on:** openmeter/productcatalog/feature, openmeter/entitlement, openmeter/billing, openmeter/ent/db, openmeter/watermill
- **Key interfaces:** `notification.Service` (CreateChannel, ListRules, CreateRule, ListEvents, CreateEvent, ResendEvent, UpdateEventDeliveryStatus), `notification.EventHandler` (Start, Close, Dispatch, Reconcile)

### openmeter/meter + ingest + sink + streaming (usage pipeline)
- **Location:** `openmeter/meter`
- **Responsibility:** The usage-metering pipeline. meter/ defines meters (event aggregation rules: type, COUNT/SUM/MAX/UNIQUE_COUNT, group-by JSON paths) and ParseEvent for value/group-by extraction. ingest/ defines the Collector interface forwarding CloudEvents to Kafka, with DeduplicatingCollector (Redis/in-memory) and OTel-decorating adapters. sink/ is the high-throughput Kafka-to-ClickHouse worker with strict three-phase flush ordering (ClickHouse BatchInsert -> Kafka offset commit -> Redis dedupe). streaming/ defines the Connector interface for ClickHouse meter aggregation queries with a concrete clickhouse/ impl and a retry/ wrapper.
- **Depends on:** openmeter/ent/db, openmeter/dedupe, openmeter/namespace, pkg/kafka, pkg/filter, openmeter/watermill/eventbus
- **Key interfaces:** `streaming.Connector` (CountEvents, ListEvents, QueryMeter, BatchInsert, CreateNamespace, DeleteNamespace), `ingest.Collector` (Ingest, Close), `sink.Sink` (Run, Close)

### openmeter/watermill (event bus)
- **Location:** `openmeter/watermill`
- **Responsibility:** Kafka-backed pub-sub abstraction. eventbus.Publisher routes typed domain events to three named Kafka topics (IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic) by EventName() prefix; unrecognized prefixes default to SystemEventsTopic. grouphandler.NoPublishingHandler multiplexes by CloudEvents ce_type header, silently dropping unknown types for rolling-deploy safety. router.NewDefaultRouter wires a fixed middleware stack (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout, HandlerMetrics).
- **Depends on:** github.com/ThreeDotsLabs/watermill, github.com/ThreeDotsLabs/watermill-kafka/v3, github.com/confluentinc/confluent-kafka-go/v2
- **Key interfaces:** `eventbus.Publisher` (Publish, WithContext, Marshaler), `grouphandler.NoPublishingHandler` (Handle)

### openmeter/namespace (multi-tenancy)
- **Location:** `openmeter/namespace`
- **Responsibility:** Multi-tenancy infrastructure. Manager fans out CreateNamespace/DeleteNamespace to all registered Handler implementations (ClickHouse streaming, Kafka ingest, Ledger) using errors.Join with no short-circuit. Handlers are registered via RegisterHandler before CreateDefaultNamespace at startup; the default namespace is protected from deletion.
- **Depends on:** pkg/models
- **Key interfaces:** `namespace.Manager` (CreateNamespace, DeleteNamespace, CreateDefaultNamespace, RegisterHandler, GetDefaultNamespace), `namespace.Handler` (CreateNamespace, DeleteNamespace)

### openmeter/ent (persistence schema + generated code)
- **Location:** `openmeter/ent`
- **Responsibility:** Persistence layer source of truth (614 files). ent/schema/ holds ~35 hand-written Ent entity definitions (billing invoices/lines, customers, entitlements, grants, features, subscriptions, plans, addons, notifications, llmcostprice, ledger accounts/transactions, meters, subjects, secrets, charges) all using shared mixins IDMixin (ULID char(26)) + NamespaceMixin + TimeMixin. ent/db/ is the fully generated Ent ORM code (DO NOT EDIT). entc.go is the single codegen driver.
- **Depends on:** entgo.io/ent, pkg/framework/entutils
- **Key interfaces:** `Ent schema entities` (Mixin, Fields, Edges)

### openmeter/server (HTTP server + v1 router)
- **Location:** `openmeter/server`
- **Responsibility:** Chi-based HTTP server assembling the v1 and v3 REST APIs behind a shared middleware stack. NewServer mounts the v3 API (v3server.NewServer + RegisterRoutes) in its own Chi Group with oasmiddleware.ValidateRequest, then the v1 API (api.HandlerWithOptions) in a separate Group with kin-openapi OapiRequestValidatorWithOptions. The router/ sub-package is the pure v1 endpoint delegation layer implementing the generated api.ServerInterface by calling typed domain httpdriver handlers; router.Config aggregates ~40 domain service fields.
- **Depends on:** api (generated v1 stubs), api/v3/server, app/config, openmeter/*/httpdriver, pkg/framework/transport/httptransport, pkg/models
- **Key interfaces:** `server.Server` (NewServer), `router.Config`

### api/v3 (v3 HTTP API)
- **Location:** `api/v3`
- **Responsibility:** v3 AIP-style HTTP API (220 files). api/v3/api.gen.go is the generated ServerInterface (DO NOT EDIT). server/ validates Config, wires domain services into typed handler structs, registers OAS validation middleware on a Chi router, and delegates each ServerInterface method via the typed .With(params).ServeHTTP pattern with credits feature-flag gating at both constructor and route-dispatch levels. handlers/ holds per-resource handler packages (meters, customers, customers/billing, customers/charges, customers/credits, billingprofiles, plans, subscriptions, addons, apps, features, llmcost, etc.) each using the httptransport.Handler[Request,Response] pipeline. filters/ provides shared AIP cursor/filter parsing.
- **Depends on:** api/v3 (generated stubs), openmeter/billing, openmeter/customer, openmeter/meter, openmeter/subscription, openmeter/productcatalog, openmeter/llmcost, openmeter/app, openmeter/ledger, pkg/framework/transport/httptransport
- **Key interfaces:** `api/v3/server.Server` (NewServer, RegisterRoutes)

### api/spec (TypeSpec source + SDKs)
- **Location:** `api/spec`
- **Responsibility:** TypeSpec source files (256 files) defining both v1 (packages/legacy/) and v3 (packages/aip/) OpenAPI specifications. Compilation (make gen-api) produces api/openapi.yaml, api/openapi.cloud.yaml, api/v3/openapi.yaml, the Go client (api/client/go), and the JavaScript SDK (api/client/javascript). Route and tag bindings are declared only in the root openmeter.tsp files. api/client/python is a separate generated Python SDK.
- **Depends on:** TypeSpec compiler 1.11.0
- **Key interfaces:** `TypeSpec definitions`

### pkg/framework (shared infrastructure)
- **Location:** `pkg/framework`
- **Responsibility:** Shared low-level infrastructure (121 files). entutils/ (82 files) provides TransactingRepo/TransactingRepoWithNoValue (ctx-propagated Ent transaction reuse with savepoints), the IDMixin/NamespaceMixin/TimeMixin schema mixins, and ULID utilities. transport/httptransport provides the generic Handler[Request,Response] decode->operate->encode pipeline. lockr/ wraps pg_advisory_xact_lock (requires an active Postgres tx in ctx). commonhttp/ provides RFC 7807 error encoding via GenericErrorEncoder. tracex/ provides OTel span helpers.
- **Depends on:** entgo.io/ent, go.opentelemetry.io/otel, pkg/models
- **Key interfaces:** `entutils.TransactingRepo` (TransactingRepo, TransactingRepoWithNoValue), `httptransport.Handler` (ServeHTTP, Chain), `lockr.Locker` (LockForTX)

### pkg/models (domain primitives)
- **Location:** `pkg/models`
- **Responsibility:** Foundational domain primitive library (32 files) with zero imports from openmeter/* domain packages. Provides NamespacedID/NamespacedKey identity types, ManagedModel/CadencedModel base structs, GenericError typed sentinels (GenericNotFoundError->404, GenericValidationError->400, GenericConflictError->409, etc.), the immutable ValidationIssue with-chain builder, ServiceHookRegistry[T] (re-entrant loop prevention via pointer-identity context key), and the RFC 7807 StatusProblem.
- **Depends on:** pkg/treex, pkg/pagination
- **Key interfaces:** `models.ServiceHooks[T]` (RegisterHooks, PreCreate, PostCreate, PreUpdate, PostUpdate, PreDelete, PostDelete), `models.Validator` (Validate)

### tools/migrate (Atlas migrations)
- **Location:** `tools/migrate`
- **Responsibility:** Atlas-generated SQL migration files in tools/migrate/migrations/ using golang-migrate format (timestamped .up.sql/.down.sql pairs plus an atlas.sum hash chain). migrate.go wraps golang-migrate for app-startup migration. tools/migrate/cmd/viewgen/ is a SQL view generator binary. Atlas config (atlas.hcl) points to ent://openmeter/ent/schema as schema source.
- **Depends on:** openmeter/ent/schema (via Atlas diff), github.com/golang-migrate/migrate/v4
- **Key interfaces:** `migrate` (Up, Down)

### collector (benthos collector module)
- **Location:** `collector`
- **Responsibility:** Separate Go module (own go.mod) running a Redpanda Benthos/Connect service extended with custom OpenMeter bloblang, input, and output plugins. collector/cmd/main.go is a thin launcher that blank-imports plugin packages then calls service.RunCLI. Built as a separate Docker image (benthos-collector.Dockerfile, CGO_ENABLED=0).
- **Depends on:** Redpanda Benthos/Connect
- **Key interfaces:** `benthos collector` (RunCLI)

## File Placement

| Component Type | Location | Naming | Example |
|---------------|----------|--------|---------|
| Service interface | `openmeter/<domain>/` | `service.go or <domain>.go at package root` | `openmeter/customer/service.go, openmeter/billing/service.go` |
| Adapter interface | `openmeter/<domain>/` | `adapter.go at package root` | `openmeter/customer/adapter.go, openmeter/billing/adapter.go` |
| Concrete service implementation | `openmeter/<domain>/service/` | `service.go inside service/ sub-package` | `openmeter/customer/service/service.go, openmeter/billing/service/service.go` |
| Ent adapter implementation | `openmeter/<domain>/adapter/` | `adapter.go inside adapter/ sub-package` | `openmeter/customer/adapter/adapter.go, openmeter/billing/adapter/adapter.go` |
| v1 HTTP handler | `openmeter/<domain>/httpdriver/` | `handler.go inside httpdriver/ or httphandler/ sub-package` | `openmeter/customer/httpdriver/handler.go, openmeter/meter/httphandler/` |
| v3 HTTP handler | `api/v3/handlers/<resource>/` | `handler.go inside per-resource sub-package` | `api/v3/handlers/customers/handler.go, api/v3/handlers/meters/` |
| Wire provider set | `app/common/` | `<domain>.go and openmeter_<binary>.go` | `app/common/billing.go, app/common/openmeter_server.go` |
| Ent entity schema | `openmeter/ent/schema/` | `<entity>.go using shared mixins` | `openmeter/ent/schema/customer.go, openmeter/ent/schema/billing.go` |
| Generated Ent code | `openmeter/ent/db/` | `generated ORM code` | `openmeter/ent/db/` |
| Generated server stubs / wire / converters | `varies` | `*.gen.go, wire_gen.go, *.gen.go` | `api/api.gen.go, api/v3/api.gen.go, cmd/server/wire_gen.go, api/v3/handlers/*/convert.gen.go` |
| SQL migrations | `tools/migrate/migrations/` | `<timestamp>_<name>.up.sql / .down.sql + atlas.sum` | `tools/migrate/migrations/20240826120919_init.up.sql` |
| TypeSpec API source | `api/spec/packages/aip/ (v3), api/spec/packages/legacy/ (v1)` | `*.tsp with route bindings only in root openmeter.tsp` | `api/spec/packages/aip/src/openmeter.tsp, api/spec/packages/legacy/src/main.tsp` |
| Test files | `alongside source files and openmeter/<domain>/testutils/` | `<name>_test.go colocated with source; shared helpers in testutils/` | `openmeter/customer/adapter/customer_test.go, openmeter/customer/testutils/env.go, e2e/` |
| Noop implementations | `openmeter/<domain>/noop/` | `noop.go inside noop/ sub-package` | `openmeter/ledger/noop/noop.go` |
| Per-folder agent context | `any package directory` | `CLAUDE.md` | `openmeter/customer/CLAUDE.md, app/common/CLAUDE.md` |

## Naming Conventions

- **Service interface**: PascalCase named Service or <Noun>Service composing fine-grained sub-interfaces (e.g. `billing.Service`, `customer.Service`, `customer.CustomerService`, `notification.EventService`)
- **Adapter interface**: PascalCase named Adapter or <Noun>Adapter composing entutils.TxCreator (e.g. `customer.Adapter`, `billing.Adapter`, `charges.Adapter`)
- **Connector interface**: PascalCase named <Noun>Connector (e.g. `streaming.Connector`, `ingest.Collector`, `feature.FeatureConnector`, `credit.CreditConnector`)
- **Service input types**: <Verb><Noun>Input suffix, implementing models.Validator (e.g. `CreateCustomerInput`, `ListCustomersInput`, `DeleteCustomerInput`, `UpdateCustomerInput`)
- **Domain events**: <Domain><Action>Event implementing EventName() prefixed by an EventVersionSubsystem constant (e.g. `MeterCreateEvent`, `InvoiceCreated`, `RecalculateEvent`)
- **Domain errors**: models.Generic* typed sentinel wrappers (e.g. `GenericNotFoundError`, `GenericValidationError`, `GenericConflictError`, `KeyConflictError (embeds GenericConflictError)`)
- **Wire provider functions**: New<Thing> exported from app/common (e.g. `NewBillingRegistry`, `NewCustomerLedgerServiceHook`, `NewLedgerNamespaceHandler`)
- **Registry structs**: <Domain>Registry grouping cohesive services with nil-safe accessors (e.g. `BillingRegistry`, `ChargesRegistry`, `AppRegistry`, `SubscriptionServiceWithWorkflow`)
- **Binary entrypoints**: cmd/<binary>/main.go + wire.go + wire_gen.go (e.g. `cmd/server/main.go`, `cmd/billing-worker/wire.go`, `cmd/sink-worker/wire_gen.go`)
- **Go packages**: lowercase single-word package directories (e.g. `billing`, `productcatalog`, `balanceworker`, `subscriptionsync`)
- **Ent schema mixins**: Mixin() returns IDMixin + NamespaceMixin + TimeMixin first (e.g. `openmeter/ent/schema/customer.go`, `openmeter/ent/schema/billing.go`)