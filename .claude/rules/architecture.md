## Components

### cmd/server
- **Location:** `cmd/server`
- **Responsibility:** Main HTTP API binary. Parses Viper config, calls Wire-generated initializeApplication, runs DB migration, registers LedgerNamespaceHandler then KafkaIngestNamespaceHandler on NamespaceManager before initNamespace, provisions sandbox app and default billing profile, initialises meters from config, constructs router.Config with ~40 domain services, and runs an oklog/run group with API server, telemetry server, Kafka producer, notification event handler, and termination checker.
- **Depends on:** app/common, app/config, openmeter/server, openmeter/server/router, openmeter/billing, openmeter/namespace, openmeter/notification
- **Key interfaces:** `initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error)`, `Application struct with ~40 typed service/registry fields`

### cmd/billing-worker
- **Location:** `cmd/billing-worker`
- **Responsibility:** Background worker subscribing to Kafka system events via Watermill. Processes subscription sync events, invoice creation events, and charge advancement events. After migration, provisions ledger business accounts (EnsureBusinessAccounts) and sandbox app before calling app.Run() to start the Watermill consumer loop.
- **Depends on:** app/common, app/config, openmeter/billing/worker, openmeter/billing/charges, openmeter/ledger, openmeter/watermill
- **Key interfaces:** `Application.Run()`

### cmd/balance-worker
- **Location:** `cmd/balance-worker`
- **Responsibility:** Subscribes to Kafka system and ingest topics via Watermill, recalculates metered entitlement grant burn-down using the credit engine and ClickHouse usage queries, dispatches balance threshold notifications after recalculation.
- **Depends on:** app/common, app/config, openmeter/entitlement/balanceworker, openmeter/entitlement, openmeter/notification, openmeter/watermill
- **Key interfaces:** `Application.Run()`

### cmd/sink-worker
- **Location:** `cmd/sink-worker`
- **Responsibility:** Consumes raw CloudEvents from Kafka ingest topic via confluent-kafka-go, deduplicates using Redis or in-memory deduplicator, batch-inserts into ClickHouse via streaming.Connector.BatchInsert, then publishes ingest flush notifications (EventBatchedIngest) to the balance-worker Kafka topic.
- **Depends on:** app/common, app/config, openmeter/sink, openmeter/streaming/clickhouse, openmeter/ingest/kafkaingest, openmeter/watermill, openmeter/dedupe
- **Key interfaces:** `sink.Sink.Run(ctx context.Context) error`, `sink.Sink.Close() error`

### cmd/notification-service
- **Location:** `cmd/notification-service`
- **Responsibility:** Standalone notification dispatcher. Subscribes to system events Kafka topic via Watermill, receives entitlement balance threshold and invoice events, and delivers webhook payloads via Svix.
- **Depends on:** app/common, app/config, openmeter/notification/consumer, openmeter/watermill
- **Key interfaces:** `notification.EventHandler.Start() error`, `notification.EventHandler.Close() error`

### cmd/jobs
- **Location:** `cmd/jobs`
- **Responsibility:** Cobra CLI grouping administrative one-off commands: billing advance/collect, entitlement backfill, ledger migrations, LLM cost sync, DB migration, and quickstart provisioning. Wires the full application once via PersistentPreRunE.
- **Depends on:** app/common, app/config, openmeter/billing, openmeter/entitlement, openmeter/ledger, openmeter/llmcost, tools/migrate

### cmd/benthos-collector
- **Location:** `cmd/benthos-collector`
- **Responsibility:** Runs a Redpanda Benthos/Connect service extended with custom OpenMeter bloblang plugins, input plugins, and output plugins. Thin launcher: blank-imports plugin packages then calls service.RunCLI.
- **Depends on:** collector/benthos

### app/common
- **Location:** `app/common`
- **Responsibility:** Houses all Google Wire provider sets and constructor functions that wire domain services, adapters, Kafka clients, DB clients, telemetry, and configuration into application structs. One file per domain area; openmeter_*.go files define Wire sets for each binary. Registers cross-domain hooks and validators as side-effects of provider functions. Returns noop implementations when features are disabled (credits.enabled=false).
- **Depends on:** all openmeter/* domain packages, pkg/*, app/config
- **Key interfaces:** `BillingRegistry{Billing billing.Service, Charges *ChargesRegistry}`, `BillingRegistry.ChargesServiceOrNil() charges.Service`, `AppRegistry`, `SubscriptionServiceWithWorkflow`

### app/config
- **Location:** `app/config`
- **Responsibility:** Viper-based configuration structs and defaults for all application concerns. Single shared config.Configuration type used by all binaries. Provides Configure* functions that set Viper defaults and pflag bindings, plus Validate() methods on every sub-struct. SetViperDefaults is the single registration point.
- **Depends on:** openmeter/meter, pkg/errorsx, pkg/models
- **Key interfaces:** `Configuration.Validate() error`

### openmeter/billing
- **Location:** `openmeter/billing`
- **Responsibility:** Core billing domain contract layer. Defines composite billing.Service interface (12 sub-interfaces: ProfileService, CustomerOverrideService, InvoiceService, GatheringInvoiceService, StandardInvoiceService, InvoiceLineService, SequenceService, LockableService, InvoiceAppService, ConfigService, LineEngineService, SplitLineGroupService) and billing.Adapter interface. Owns all billing domain model types including InvoiceLine tagged-union with private discriminator and NewStandardInvoiceLine/NewGatheringInvoiceLine constructors.
- **Depends on:** openmeter/app, openmeter/customer, openmeter/productcatalog/feature, pkg/pagination, pkg/models, pkg/framework/entutils
- **Key interfaces:** `billing.Service (composite of 12 sub-interfaces)`, `billing.Adapter`, `billing.InvoicingApp (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice)`

### openmeter/billing/service
- **Location:** `openmeter/billing/service`
- **Responsibility:** Concrete billing.Service implementation. Drives invoice lifecycle state machine via stateless library (stdinvoicestate.go builds *stateless.StateMachine pooled in sync.Pool). Implements multi-step charge advancement, invoice gathering, line snapshotting, profile management, and sequence numbering. Acquires pg_advisory_lock per customer via lockr.Locker before any invoice/charge mutation.
- **Depends on:** openmeter/billing, openmeter/app, openmeter/customer, openmeter/productcatalog/feature, openmeter/meter, openmeter/streaming, openmeter/watermill/eventbus, pkg/framework/lockr, pkg/framework/entutils

### openmeter/billing/adapter
- **Location:** `openmeter/billing/adapter`
- **Responsibility:** Ent/PostgreSQL implementation of billing.Adapter. All methods wrap Ent access with entutils.TransactingRepo so ctx-carried transactions are honored. Handles invoice line diff computation, schema-level migrations, entity-diff-based upsert for line hierarchies, and gathering-invoice materialized logic. Implements Tx()/WithTx()/Self() triad.
- **Depends on:** openmeter/ent/db, pkg/framework/entutils, openmeter/billing

### openmeter/billing/charges
- **Location:** `openmeter/billing/charges`
- **Responsibility:** Charges sub-domain defining tagged-union Charge/ChargeIntent types (private discriminator accessed via AsFlatFeeCharge/AsUsageBasedCharge/AsCreditPurchaseCharge), composite charges.Service interface (ChargeService + CreditPurchaseFacadeService), and charges.Adapter. Charge lifecycle flows through Create/AdvanceCharges/ApplyPatches.
- **Depends on:** openmeter/billing/charges/meta, openmeter/billing/charges/flatfee, openmeter/billing/charges/usagebased, openmeter/billing/charges/creditpurchase, openmeter/customer, pkg/models, pkg/pagination
- **Key interfaces:** `charges.Service (GetByID, GetByIDs, Create, AdvanceCharges, ListCustomersToAdvance, ApplyPatches, ListCharges, HandleCreditPurchaseExternalPaymentStateTransition)`, `charges.Adapter (ChargesSearchAdapter + entutils.TxCreator)`

### openmeter/billing/worker
- **Location:** `openmeter/billing/worker`
- **Responsibility:** Billing worker sub-packages: advance/ runs invoice auto-advance loop, collect/ runs payment collection loop, subscriptionsync/ reconciles subscription views into invoice lines via SynchronizeSubscription, asyncadvance/ handles async invoice advancement events. Worker struct composes the Watermill router and all billing sub-services.
- **Depends on:** openmeter/billing, openmeter/billing/charges, openmeter/subscription, openmeter/productcatalog, openmeter/watermill, pkg/framework/entutils
- **Key interfaces:** `subscriptionsync.Service (SynchronizeSubscription, SynchronizeSubscriptionAndInvoiceCustomer, HandleCancelledEvent, HandleSubscriptionSyncEvent, HandleInvoiceCreation, GetSyncStates)`

### openmeter/customer
- **Location:** `openmeter/customer`
- **Responsibility:** Customer lifecycle management (ListCustomers, ListCustomerUsageAttributions, CreateCustomer, DeleteCustomer, GetCustomer, GetCustomerByUsageAttribution, UpdateCustomer) with soft-delete semantics. Provides RequestValidatorRegistry (pre-mutation cross-domain guards) and ServiceHooks[Customer] for post-lifecycle callbacks.
- **Depends on:** openmeter/streaming, openmeter/ent/db, openmeter/watermill/eventbus, pkg/models, pkg/pagination, pkg/framework/entutils
- **Key interfaces:** `customer.Service (ListCustomers, ListCustomerUsageAttributions, CreateCustomer, DeleteCustomer, GetCustomer, GetCustomerByUsageAttribution, UpdateCustomer, RegisterRequestValidator, RegisterHooks)`, `customer.RequestValidator (ValidateDeleteCustomer, ValidateCreateCustomer, ValidateUpdateCustomer)`

### openmeter/entitlement
- **Location:** `openmeter/entitlement`
- **Responsibility:** Feature entitlement management across three sub-types: metered (credit grant burn-down via credit engine + ClickHouse usage queries), boolean (on/off), and static (JSON config value). Service composes all sub-types and provides scheduling, override, supersede, and balance history.
- **Depends on:** openmeter/productcatalog/feature, openmeter/customer, openmeter/streaming, openmeter/credit, openmeter/ent/db, openmeter/watermill/eventbus, pkg/framework/lockr, pkg/models
- **Key interfaces:** `entitlement.Service (CreateEntitlement, OverrideEntitlement, ScheduleEntitlement, SupersedeEntitlement, GetEntitlement, GetEntitlementValue, GetEntitlementsOfCustomer, ListEntitlements, GetAccess, DeleteEntitlement, RegisterHooks)`

### openmeter/entitlement/balanceworker
- **Location:** `openmeter/entitlement/balanceworker`
- **Responsibility:** Kafka-driven worker recalculating entitlement balances on lifecycle events. Subscribes to three topics (system, ingest, balance-worker). Converts direct lifecycle events into RecalculateEvent on the balance-worker topic; a second handler consumes RecalculateEvent and calls handleEntitlementEvent for filter-fetch-snapshot pipeline. Uses LRU caches and high-watermark filter.
- **Depends on:** openmeter/entitlement, openmeter/entitlement/metered, openmeter/credit/grant, openmeter/customer, openmeter/notification, openmeter/watermill, pkg/framework/lockr
- **Key interfaces:** `balanceworker.Worker (AddHandler, Run(ctx context.Context) error)`

### openmeter/subscription
- **Location:** `openmeter/subscription`
- **Responsibility:** Manages subscription lifecycle (Create, Update, Delete, Cancel, Continue, UpdateAnnotations) against a versioned plan-phase-RateCard model. Uses SubscriptionSpec manipulated via AppliesToSpec patch interface. Workflow service orchestrates higher-level operations. Addon service manages quantity-based addons.
- **Depends on:** openmeter/productcatalog, openmeter/customer, openmeter/entitlement, openmeter/ent/db, openmeter/watermill/eventbus, pkg/framework/lockr, pkg/models
- **Key interfaces:** `subscription.Service (Get, GetView, List, ExpandViews, Create, Update, Delete, Cancel, Continue, UpdateAnnotations, RegisterHook)`, `subscriptionworkflow.Service (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity)`

### openmeter/notification
- **Location:** `openmeter/notification`
- **Responsibility:** Defines all notification domain types (Channel, Rule, Event, EventPayload union dispatch, EventDeliveryStatus) and service interfaces (ChannelService + RuleService + EventService + FeatureService). EventHandler interface drives dispatch and reconciliation loop. Consumer sub-package contains Watermill consumer dispatching to Svix.
- **Depends on:** openmeter/productcatalog/feature, openmeter/entitlement, openmeter/billing, openmeter/ent/db, openmeter/watermill, pkg/models
- **Key interfaces:** `notification.Service (ListChannels, CreateChannel, DeleteChannel, GetChannel, UpdateChannel, ListRules, CreateRule, DeleteRule, GetRule, UpdateRule, ListEvents, GetEvent, CreateEvent, ResendEvent, ListEventsDeliveryStatus, GetEventDeliveryStatus, UpdateEventDeliveryStatus, ListFeature)`, `notification.EventHandler (Start() error, Close() error, Dispatch(ctx, Event) error, Reconcile(ctx) error)`

### openmeter/ledger
- **Location:** `openmeter/ledger`
- **Responsibility:** Double-entry ledger for customer financial balances (FBO, Receivable, Accrued) and business accounts (Wash, Earnings, Brokerage). Transaction inputs constructed exclusively via transactions.ResolveTransactions with typed template structs. noop/ provides zero-value implementations when credits.enabled=false.
- **Depends on:** openmeter/customer, openmeter/ent/db, pkg/currencyx, pkg/framework/lockr, pkg/framework/entutils
- **Key interfaces:** `ledger.AccountResolver (EnsureCustomerAccounts, GetCustomerAccounts, EnsureBusinessAccounts)`, `ledger.Ledger (CommitGroup, QueryBalance)`

### openmeter/meter
- **Location:** `openmeter/meter`
- **Responsibility:** Defines meters (event aggregation rules: event type, aggregation function COUNT/SUM/MAX/UNIQUE_COUNT, optional group-by JSON paths, optional value property). ManageService extends Service with CreateMeter/UpdateMeter/DeleteMeter. Publishes MeterCreateEvent/MeterUpdateEvent/MeterDeleteEvent after mutations.
- **Depends on:** openmeter/ent/db, openmeter/watermill/eventbus, pkg/filter, pkg/pagination, pkg/models
- **Key interfaces:** `meter.Service (ListMeters, GetMeterByIDOrSlug)`, `meter.ManageService (ListMeters, GetMeterByIDOrSlug, CreateMeter, UpdateMeter, DeleteMeter, RegisterPreUpdateMeterHook)`

### openmeter/ingest
- **Location:** `openmeter/ingest`
- **Responsibility:** CloudEvent ingestion pipeline. Collector interface (Ingest, Close) receives single events and forwards to Kafka. DeduplicatingCollector wraps any Collector with Redis or in-memory deduplication. ingestadapter/ decorates with OTel telemetry. kafkaingest/ implements Collector using confluent-kafka-go.
- **Depends on:** openmeter/dedupe, pkg/kafka, openmeter/watermill/eventbus
- **Key interfaces:** `ingest.Collector (Ingest(ctx context.Context, namespace string, ev event.Event) error, Close())`

### openmeter/sink
- **Location:** `openmeter/sink`
- **Responsibility:** High-throughput Kafka-to-ClickHouse sink worker. Sink struct consumes Kafka partitions via confluent-kafka-go, buffers messages in SinkBuffer, flushes in three-phase order (ClickHouse BatchInsert -> Kafka offset commit -> Redis dedupe), then fires FlushEventHandler in a goroutine with FlushSuccessTimeout context.
- **Depends on:** openmeter/streaming, openmeter/ingest/kafkaingest, openmeter/meter, openmeter/dedupe, openmeter/watermill, pkg/kafka/metrics
- **Key interfaces:** `sink.Storage (BatchInsert(ctx context.Context, messages []SinkMessage) error)`, `sink.Sink (Run(ctx context.Context) error, Close() error)`

### openmeter/streaming
- **Location:** `openmeter/streaming`
- **Responsibility:** Defines the streaming.Connector interface for querying meter aggregations and listing raw events from ClickHouse, and for namespace lifecycle (embeds namespace.Handler). Concrete implementation in clickhouse/ uses sqlbuilder query structs with toSQL() methods. retry/ wraps with retry logic. testutils/ provides MockStreamingConnector.
- **Depends on:** openmeter/meter, openmeter/namespace, pkg/models, pkg/filter
- **Key interfaces:** `streaming.Connector (CountEvents, ListEvents, ListEventsV2, ListSubjects, ListGroupByValues, QueryMeter, BatchInsert, ValidateJSONPath, CreateNamespace, DeleteNamespace)`

### openmeter/watermill
- **Location:** `openmeter/watermill`
- **Responsibility:** Kafka-backed pub-sub abstraction. eventbus.Publisher routes typed domain events to three named Kafka topics (IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic) by event-name prefix. grouphandler.NoPublishingHandler multiplexes by CloudEvents ce_type header; unknown types silently dropped. router.NewDefaultRouter wires fixed middleware stack.
- **Depends on:** github.com/ThreeDotsLabs/watermill, github.com/ThreeDotsLabs/watermill-kafka/v3, github.com/confluentinc/confluent-kafka-go/v2
- **Key interfaces:** `eventbus.Publisher (Publish(ctx context.Context, event marshaler.Event) error, WithContext(ctx context.Context) ContextPublisher, Marshaler() marshaler.Marshaler)`

### openmeter/namespace
- **Location:** `openmeter/namespace`
- **Responsibility:** Multi-tenancy infrastructure. Manager fans out CreateNamespace/DeleteNamespace to all registered Handler implementations (ClickHouse streaming, Kafka ingest, Ledger). Handlers are registered dynamically via RegisterHandler before CreateDefaultNamespace at startup. Fan-out uses errors.Join (no short-circuit).
- **Depends on:** pkg/models
- **Key interfaces:** `namespace.Manager (CreateNamespace, DeleteNamespace, CreateDefaultNamespace, RegisterHandler, GetDefaultNamespace)`, `namespace.Handler (CreateNamespace(ctx context.Context, name string) error, DeleteNamespace(ctx context.Context, name string) error)`

### openmeter/productcatalog
- **Location:** `openmeter/productcatalog`
- **Responsibility:** Defines the product catalog: features (meter-backed usage features), plans (versioned multi-phase rate card collections), plan addons, addons, and rate cards (usage-based and flat-fee pricing). Each entity has a Service interface, Ent adapter, and httpdriver package.
- **Depends on:** openmeter/meter, openmeter/ent/db, openmeter/watermill, pkg/models, pkg/pagination
- **Key interfaces:** `feature.FeatureConnector (CreateFeature, UpdateFeature, ArchiveFeature, ListFeatures, GetFeature, ResolveFeatureMeters)`, `plan.Service (ListPlans, CreatePlan, DeletePlan, GetPlan, UpdatePlan, PublishPlan, ArchivePlan, NextPlan)`

### openmeter/app
- **Location:** `openmeter/app`
- **Responsibility:** Marketplace registry and runtime lifecycle for installed billing apps. AppFactory self-registers at constructor via RegisterMarketplaceListing. Concrete implementations under stripe/, sandbox/, and custominvoicing/ all embed AppBase and implement billing.InvoicingApp.
- **Depends on:** openmeter/customer, openmeter/secret, openmeter/billing, openmeter/ent/db, openmeter/watermill/eventbus, pkg/pagination, pkg/models
- **Key interfaces:** `app.Service (RegisterMarketplaceListing, GetMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, InstallMarketplaceListing, GetMarketplaceListingOauth2InstallURL, AuthorizeMarketplaceListingOauth2Install, CreateApp, GetApp, UpdateAppStatus, UpdateApp, ListApps, UninstallApp, ListCustomerData, EnsureCustomer, DeleteCustomer)`, `app.App (GetAppBase, GetID, GetType, GetName, GetStatus, GetListing, UpdateAppConfig, ValidateCapabilities, GetCustomerData, UpsertCustomerData, DeleteCustomerData)`

### openmeter/credit
- **Location:** `openmeter/credit`
- **Responsibility:** Manages credit grants and balance snapshots for metered entitlements. CreditConnector (= BalanceConnector + GrantConnector) is the public facade. engine/ sub-package computes grant burn-down without I/O. Granularity truncation (time.Minute) applied to all effective times.
- **Depends on:** openmeter/streaming, openmeter/credit/grant, openmeter/credit/balance, openmeter/credit/engine, openmeter/ent/db, openmeter/watermill/eventbus, pkg/framework/transaction
- **Key interfaces:** `credit.CreditConnector (GetBalanceAt, GetBalanceForPeriod, ResetUsageForOwner, CreateGrant, VoidGrant)`

### openmeter/llmcost
- **Location:** `openmeter/llmcost`
- **Responsibility:** LLM cost price management: persists global (synced) prices and per-namespace overrides in llmcostprice Ent entity. Service resolves effective prices with namespace-override precedence. sync/ sub-package runs a four-phase pipeline to synchronize prices from external sources. NormalizeModelID must be called before any price store or resolve.
- **Depends on:** openmeter/ent/db, pkg/filter, pkg/pagination, pkg/models
- **Key interfaces:** `llmcost.Service (ListPrices, GetPrice, ResolvePrice, CreateOverride, DeleteOverride, ListOverrides)`

### openmeter/ent/schema
- **Location:** `openmeter/ent/schema`
- **Responsibility:** Ent schema definitions (source of truth for database schema). 35 entity files covering billing invoices/lines, customers, entitlements, grants, features, subscriptions, plans, addons, notifications, LLM cost prices, ledger accounts/transactions, meters, subjects, secrets, charges. Used by Atlas to generate SQL migrations. All entities use shared mixins: IDMixin (ULID char(26)), NamespaceMixin, TimeMixin.
- **Depends on:** entgo.io/ent, pkg/framework/entutils

### openmeter/server/router
- **Location:** `openmeter/server/router`
- **Responsibility:** Pure v1 endpoint delegation layer implementing api.ServerInterface by calling typed domain handlers. Config struct aggregating all domain services (~40 interface fields). init() registers CloudEvents body decoders for kin-openapi.
- **Depends on:** openmeter/billing/httpdriver, openmeter/customer/httpdriver, openmeter/meter/httphandler, api (generated stubs), pkg/framework/transport/httptransport
- **Key interfaces:** `router.Config (all ~40 domain service fields)`

### api/v3/server
- **Location:** `api/v3/server`
- **Responsibility:** v3 HTTP server: validates Config, wires all domain service dependencies into typed handler structs, registers OAS validation middleware (oasmiddleware.ValidateRequest) on Chi router, and delegates every generated ServerInterface method via typed .With(params).ServeHTTP(w, r) pattern. Credits feature flag gated at both NewServer constructor level (noop wiring) and route dispatch level.
- **Depends on:** api/v3 (generated stubs), api/v3/handlers/*, openmeter/billing, openmeter/customer, openmeter/meter, openmeter/subscription, openmeter/productcatalog, openmeter/llmcost, openmeter/app, openmeter/ledger, pkg/framework/transport/httptransport
- **Key interfaces:** `api/v3/server.Server implements api.ServerInterface (all generated v3 ServerInterface methods)`

### api/v3/handlers
- **Location:** `api/v3/handlers`
- **Responsibility:** v3 API handler packages organized per resource group: meters, customers, customers/billing, customers/charges, customers/credits, customers/entitlementaccess, billingprofiles, plans, plans/planaddons, subscriptions, addons, apps, features, featurecost, llmcost, taxcodes, currencies, events. Each sub-package implements relevant ServerInterface methods using httptransport.Handler[Request,Response] pipeline.
- **Depends on:** api/v3 (generated types), pkg/framework/transport/httptransport, openmeter/* domain services

### api/spec
- **Location:** `api/spec`
- **Responsibility:** TypeSpec source files defining both v1 (packages/legacy/) and v3 (packages/aip/) OpenAPI specifications. Compilation produces api/openapi.yaml, api/openapi.cloud.yaml, api/v3/openapi.yaml. Also produces Go client (api/client/go/client.gen.go) and JavaScript SDK (api/client/javascript/). Route and tag bindings declared only in root openmeter.tsp files.
- **Depends on:** TypeSpec compiler 1.11.0

### tools/migrate
- **Location:** `tools/migrate`
- **Responsibility:** Atlas-generated SQL migration files in tools/migrate/migrations/ using golang-migrate format (timestamped .up.sql/.down.sql pairs plus atlas.sum hash chain). migrate.go wraps golang-migrate for use at app startup. Atlas config in atlas.hcl points to ent://openmeter/ent/schema as schema source.
- **Depends on:** openmeter/ent/schema (via Atlas diff), github.com/golang-migrate/migrate/v4

### pkg/framework
- **Location:** `pkg/framework`
- **Responsibility:** Shared low-level infrastructure layer. httptransport provides generic Handler[Request,Response] struct (decode -> operation -> encode with ErrorEncoder chain). entutils provides TransactingRepo/TransactingRepoWithNoValue (ctx-propagated Ent transaction reuse with savepoints), Ent schema mixins (IDMixin ULID, NamespaceMixin, TimeMixin), and ULID utilities. lockr provides pg_advisory_xact_lock wrapper. commonhttp provides RFC 7807 error encoding (GenericErrorEncoder chain).
- **Depends on:** entgo.io/ent, go.opentelemetry.io/otel
- **Key interfaces:** `httptransport.Handler[Request, Response] (ServeHTTP(w http.ResponseWriter, r *http.Request), Chain(outer operation.Middleware[Request, Response], others ...operation.Middleware[Request, Response]) Handler[Request, Response])`, `entutils.TxCreator (Tx(ctx context.Context) (context.Context, transaction.Driver, error))`, `lockr.Locker (LockForTX(ctx context.Context, key lockr.Key) error)`

### pkg/models
- **Location:** `pkg/models`
- **Responsibility:** Foundational domain primitive library with zero imports from openmeter/* domain packages. Provides: NamespacedID/NamespacedKey identity types, ManagedModel/CadencedModel base structs, GenericError typed sentinels (GenericNotFoundError->404, GenericValidationError->400, GenericConflictError->409), ValidationIssue (immutable with-chain builder), ServiceHookRegistry[T] (re-entrant loop prevention via pointer-identity context key), RFC 7807 StatusProblem, FieldDescriptor tree for structured field paths.
- **Depends on:** pkg/treex, pkg/pagination
- **Key interfaces:** `models.ServiceHooks[T] (RegisterHooks(...ServiceHook[T]), PreCreate, PostCreate, PreUpdate, PostUpdate, PreDelete, PostDelete)`, `models.Validator (Validate() error)`

## File Placement

| Component Type | Location | Naming | Example |
|---------------|----------|--------|---------|
|  | `` | `` | `cmd/server/main.go, cmd/billing-worker/wire.go` |
|  | `` | `` | `openmeter/billing/service.go, openmeter/customer/service.go` |
|  | `` | `` | `openmeter/billing/adapter.go, openmeter/customer/adapter.go` |
|  | `` | `` | `openmeter/billing/adapter/adapter.go, openmeter/customer/adapter/adapter.go` |
|  | `` | `` | `openmeter/billing/service/service.go, openmeter/customer/service/service.go` |
|  | `` | `` | `openmeter/billing/httpdriver/handler.go, openmeter/meter/httphandler/handler.go` |
|  | `` | `` | `api/v3/handlers/customers/handler.go, api/v3/handlers/meters/handler.go` |
|  | `` | `` | `app/common/billing.go, app/common/customer.go, app/common/openmeter_server.go, app/common/openmeter_billingworker.go` |
|  | `` | `` | `openmeter/ent/db/, cmd/server/wire_gen.go, api/api.gen.go` |
|  | `` | `` | `openmeter/ent/schema/customer.go, openmeter/ent/schema/billing.go` |
|  | `` | `` | `tools/migrate/migrations/20240826120919_init.up.sql` |
|  | `` | `` | `openmeter/billing/testutils/, openmeter/streaming/testutils/` |
|  | `` | `` | `api/spec/packages/aip/src/openmeter.tsp, api/spec/packages/legacy/src/main.tsp` |
|  | `` | `` | `openmeter/ledger/noop/noop.go` |

## Naming Conventions

- ****: PascalCase named Service or <Noun>Service (e.g. )
- ****: PascalCase named Adapter or <Noun>Adapter (e.g. )
- ****: PascalCase named Connector (e.g. )
- ****: Handler type in httpdriver/ or httphandler/ package (e.g. )
- ****: <Verb><Noun>Input suffix (e.g. )
- ****: PascalCase var exported from app/common (e.g. )
- ****: <Domain>Registry struct grouping related services with nil-safe accessors (e.g. )
- ****: models.Generic* typed sentinel wrappers (e.g. )
- ****: <Domain><Action>Event PascalCase implementing EventName() string method (e.g. )
- ****: <timestamp>_<name>.up.sql and <timestamp>_<name>.down.sql (e.g. )
- ****: Mixin() must return entutils.IDMixin{}, entutils.NamespaceMixin{}, entutils.TimeMixin{} as the first three mixins (e.g. )
- ****: Must implement Tx(ctx)/WithTx(ctx, tx)/Self() triad; every method body wraps with entutils.TransactingRepo (e.g. )
- ****: EventVersionSubsystem constant used as EventName() prefix for topic routing (e.g. )