## Components

### cmd/server
- **Location:** `cmd/server`
- **Responsibility:** Main HTTP API server binary. Parses Viper config, calls Wire-generated initializeApplication, runs DB migration, registers LedgerNamespaceHandler then KafkaIngestNamespaceHandler on NamespaceManager (in that order, before initNamespace), provisions sandbox app and default billing profile, initialises meters from config, constructs router.Config with all ~25 domain services, starts an oklog/run group with API HTTP server, telemetry server, Kafka producer, notification event handler, and termination checker.
- **Depends on:** app/common, app/config, openmeter/server, openmeter/server/router, openmeter/billing, openmeter/ingest/kafkaingest, openmeter/namespace, openmeter/notification, openmeter/debug
- **Key interfaces:** `initializeApplication` (initializeApplication(ctx, conf) (Application, cleanup, error))

### cmd/billing-worker
- **Location:** `cmd/billing-worker`
- **Responsibility:** Background worker binary that subscribes to Kafka system events via Watermill, processes subscription sync events, invoice creation events, and charge advancement events. After migration, provisions ledger business accounts (EnsureBusinessAccounts) and sandbox app before calling app.Run() to start the Watermill consumer loop.
- **Depends on:** app/common, app/config, openmeter/billing/worker, openmeter/billing/worker/subscriptionsync, openmeter/billing/charges, openmeter/ledger, openmeter/watermill
- **Key interfaces:** `Application.Run` (Run())

### cmd/balance-worker
- **Location:** `cmd/balance-worker`
- **Responsibility:** Subscribes to Kafka system and ingest topics via Watermill, recalculates metered entitlement grant burn-down using the credit engine and ClickHouse usage queries, dispatches balance threshold notifications after recalculation.
- **Depends on:** app/common, app/config, openmeter/entitlement/balanceworker, openmeter/entitlement, openmeter/notification, openmeter/watermill
- **Key interfaces:** `Application.Run` (Run())

### cmd/sink-worker
- **Location:** `cmd/sink-worker`
- **Responsibility:** Consumes raw CloudEvents from Kafka ingest topic via confluent-kafka-go, deduplicates using Redis or in-memory deduplicator, batch-inserts into ClickHouse via streaming.Connector.BatchInsert, then publishes ingest flush notifications (EventBatchedIngest) to the balance-worker Kafka topic via Watermill.
- **Depends on:** app/common, app/config, openmeter/sink, openmeter/streaming/clickhouse, openmeter/ingest/kafkaingest, openmeter/watermill, openmeter/dedupe
- **Key interfaces:** `sink.Sink` (Run(ctx) error, Close() error)

### cmd/notification-service
- **Location:** `cmd/notification-service`
- **Responsibility:** Standalone notification dispatcher. Subscribes to system events Kafka topic via Watermill, receives entitlement balance threshold and invoice events, and delivers webhook payloads via Svix. Constructs its Watermill Kafka subscriber in main.go (not Wire) so consumer group name can come from config.
- **Depends on:** app/common, app/config, openmeter/notification/consumer, openmeter/watermill
- **Key interfaces:** `notification.EventHandler` (Start() error, Close() error)

### cmd/jobs
- **Location:** `cmd/jobs`
- **Responsibility:** Cobra CLI grouping administrative one-off commands: billing advance/collect, entitlement backfill, ledger migrations, LLM cost sync, DB migration, and quickstart provisioning. Wires the full application once via PersistentPreRunE; sub-commands use package-level globals from cmd/jobs/internal.
- **Depends on:** app/common, app/config, openmeter/billing, openmeter/entitlement, openmeter/ledger, openmeter/llmcost, tools/migrate

### cmd/benthos-collector
- **Location:** `cmd/benthos-collector`
- **Responsibility:** Runs a Redpanda Benthos/Connect service extended with custom OpenMeter bloblang plugins, input plugins, and output plugins. Thin launcher: blank-imports plugin packages then calls service.RunCLI. No Wire DI.
- **Depends on:** collector/benthos

### app/common
- **Location:** `app/common`
- **Responsibility:** Houses all Google Wire provider sets and constructor functions that wire domain services, adapters, Kafka clients, DB clients, telemetry, and configuration into application structs. One file per domain area (billing.go, customer.go, subscription.go, ledger.go, etc.); openmeter_*.go files define Wire sets for each binary. Registers cross-domain hooks (customer ledger hook, billing customer validator, subscription entitlement validator) as side-effects of provider functions. Returns noop implementations when features are disabled (credits.enabled=false).
- **Depends on:** all openmeter/* domain packages, pkg/*, app/config
- **Key interfaces:** `BillingRegistry` (ChargesServiceOrNil() charges.Service), `AppRegistry` (SandboxProvisioner(ctx, namespace) error), `SubscriptionServiceWithWorkflow`

### app/config
- **Location:** `app/config`
- **Responsibility:** Viper-based configuration structs and defaults for all application concerns. Single shared config.Configuration type used by all binaries. Provides Configure* functions that set Viper defaults and pflag bindings, plus Validate() methods on every sub-struct. SetViperDefaults is the single registration point calling every Configure* function.
- **Depends on:** openmeter/meter, pkg/errorsx, pkg/models
- **Key interfaces:** `Configuration` (Validate() error)

### openmeter/billing
- **Location:** `openmeter/billing`
- **Responsibility:** Core billing domain: defines composite billing.Service interface (ProfileService, CustomerOverrideService, InvoiceService, GatheringInvoiceService, StandardInvoiceService, InvoiceLineService, SequenceService, LockableService, InvoiceAppService, ConfigService, LineEngineService, SplitLineGroupService) and billing.Adapter interface. Owns all billing domain model types including InvoiceLine tagged-union (private discriminator t InvoiceLineType, accessed via AsStandardLine/AsGatheringLine/AsGenericLine), WorkflowConfig validation chain, InvoicingApp plugin interface for external billing backends, UpsertStandardInvoiceResult/FinalizeStandardInvoiceResult builder types.
- **Depends on:** openmeter/app, openmeter/customer, openmeter/productcatalog/feature, pkg/pagination, pkg/models, pkg/framework/entutils
- **Key interfaces:** `Service` (CreateProfile, GetDefaultProfile, GetProfile, ListProfiles, DeleteProfile, UpdateProfile, ProvisionDefaultBillingProfile, UpsertCustomerOverride, DeleteCustomerOverride, GetCustomerOverride, ListCustomerOverrides, InvoicePendingLines, ListInvoices, GetInvoiceById, AdvanceInvoice, SnapshotQuantities, ApproveInvoice, PaymentAuthorized, RetryInvoice, DeleteInvoice, UpdateInvoice, SimulateInvoice, CreatePendingInvoiceLines, RegisterLineEngine, DeregisterLineEngine, WithLock, RegisterStandardInvoiceHooks), `InvoicingApp` (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice), `Adapter`

### openmeter/billing/service
- **Location:** `openmeter/billing/service`
- **Responsibility:** Concrete billing.Service implementation. Drives invoice lifecycle state machine via stateless library (stdinvoicestate.go builds *stateless.StateMachine from sync.Pool). Implements multi-step charge advancement, invoice gathering, line snapshotting, profile management, and sequence numbering. Acquires pg_advisory_lock per customer via lockr.Locker before any invoice/charge mutation.
- **Depends on:** openmeter/billing, openmeter/app, openmeter/customer, openmeter/productcatalog/feature, openmeter/meter, openmeter/streaming, openmeter/watermill/eventbus, pkg/framework/lockr, pkg/framework/entutils

### openmeter/billing/adapter
- **Location:** `openmeter/billing/adapter`
- **Responsibility:** Ent/PostgreSQL implementation of billing.Adapter. All methods wrap Ent access with entutils.TransactingRepo so ctx-carried transactions are honored. Handles invoice line diff computation (stdinvoicelinediff.go), schema-level migrations, and gathering-invoice materialized logic.
- **Depends on:** openmeter/ent/db, pkg/framework/entutils, openmeter/billing

### openmeter/billing/httpdriver
- **Location:** `openmeter/billing/httpdriver`
- **Responsibility:** v1 HTTP handler package for billing domain. Implements httptransport.Handler[Request,Response] pipeline for billing endpoints: invoice listing/get/advance/approve/retry/delete, profile CRUD, customer override CRUD. Handler struct receives billing.Service and delegates via typed operation functions.
- **Depends on:** openmeter/billing, pkg/framework/transport/httptransport, pkg/framework/commonhttp

### openmeter/billing/charges
- **Location:** `openmeter/billing/charges`
- **Responsibility:** Charges sub-domain: defines tagged-union Charge/ChargeIntent types (discriminated by private meta.ChargeType field, accessed via AsFlatFeeCharge/AsUsageBasedCharge/AsCreditPurchaseCharge), composite charges.Service interface (ChargeService + CreditPurchaseFacadeService), charges.Adapter (ChargesSearchAdapter + entutils.TxCreator). Sub-packages flatfee, usagebased, creditpurchase own type-specific logic. Charge lifecycle flows through Create/AdvanceCharges/ApplyPatches; never via sub-package adapters directly.
- **Depends on:** openmeter/billing/charges/meta, openmeter/billing/charges/flatfee, openmeter/billing/charges/usagebased, openmeter/billing/charges/creditpurchase, openmeter/customer, pkg/models, pkg/pagination
- **Key interfaces:** `Service` (GetByID, GetByIDs, Create, AdvanceCharges, ListCustomersToAdvance, ApplyPatches, ListCharges, HandleCreditPurchaseExternalPaymentStateTransition)

### openmeter/billing/worker
- **Location:** `openmeter/billing/worker`
- **Responsibility:** Billing worker sub-packages: advance/ runs invoice auto-advance loop, collect/ runs payment collection loop, subscriptionsync/ reconciles subscription views into invoice lines via SynchronizeSubscription/SynchronizeSubscriptionAndInvoiceCustomer/HandleCancelledEvent/HandleSubscriptionSyncEvent/HandleInvoiceCreation. reconciler/ periodically re-syncs subscriptions that missed events. Worker struct composes the Watermill router and all billing sub-services.
- **Depends on:** openmeter/billing, openmeter/billing/charges, openmeter/subscription, openmeter/productcatalog, openmeter/watermill, pkg/framework/entutils
- **Key interfaces:** `subscriptionsync.Service` (SynchronizeSubscription, SynchronizeSubscriptionAndInvoiceCustomer, HandleCancelledEvent, HandleSubscriptionSyncEvent, HandleInvoiceCreation, GetSyncStates)

### openmeter/billing/rating
- **Location:** `openmeter/billing/rating`
- **Responsibility:** Rating service: ResolveBillablePeriod computes the billable period for a subscription, GenerateDetailedLines generates detailed billing lines for a subscription view against ClickHouse usage. Service sub-package provides the concrete implementation.
- **Depends on:** openmeter/billing, openmeter/subscription, openmeter/productcatalog, openmeter/streaming, openmeter/meter
- **Key interfaces:** `rating.Service` (ResolveBillablePeriod, GenerateDetailedLines)

### openmeter/customer
- **Location:** `openmeter/customer`
- **Responsibility:** Customer lifecycle management (ListCustomers, ListCustomerUsageAttributions, CreateCustomer, DeleteCustomer, GetCustomer, GetCustomerByUsageAttribution, UpdateCustomer) with soft-delete semantics. Provides RequestValidatorRegistry (pre-mutation cross-domain guards: ValidateDeleteCustomer, ValidateCreateCustomer, ValidateUpdateCustomer) and ServiceHooks[Customer] for post-lifecycle callbacks. Service layer wraps all mutations in transaction.Run and fans out to registered validators/hooks.
- **Depends on:** openmeter/streaming, openmeter/ent/db, openmeter/watermill/eventbus, pkg/models, pkg/pagination, pkg/framework/entutils
- **Key interfaces:** `Service` (ListCustomers, ListCustomerUsageAttributions, CreateCustomer, DeleteCustomer, GetCustomer, GetCustomerByUsageAttribution, UpdateCustomer, RegisterRequestValidator, RegisterHooks), `RequestValidator` (ValidateDeleteCustomer, ValidateCreateCustomer, ValidateUpdateCustomer)

### openmeter/entitlement
- **Location:** `openmeter/entitlement`
- **Responsibility:** Feature entitlement management across three sub-types: metered (credit grant burn-down via credit engine + ClickHouse usage queries), boolean (on/off), and static (JSON config value). Service composes all sub-types and provides scheduling, override, supersede, and balance history. Dispatches to sub-type connectors (metered/boolean/static) via type discriminator. Acquires pg_advisory_lock per customer before operations that modify multiple entitlement rows.
- **Depends on:** openmeter/productcatalog/feature, openmeter/customer, openmeter/streaming, openmeter/credit, openmeter/ent/db, openmeter/watermill/eventbus, pkg/framework/lockr, pkg/models
- **Key interfaces:** `Service` (CreateEntitlement, OverrideEntitlement, ScheduleEntitlement, SupersedeEntitlement, GetEntitlement, GetEntitlementValue, GetEntitlementsOfCustomer, ListEntitlements, ListEntitlementsWithCustomer, GetEntitlementOfCustomerAt, GetAccess, DeleteEntitlement, GetEntitlementWithCustomer, RegisterHooks), `meteredentitlement.Connector` (GetEntitlementBalance, GetEntitlementBalanceHistory, ResetEntitlementUsage)

### openmeter/entitlement/balanceworker
- **Location:** `openmeter/entitlement/balanceworker`
- **Responsibility:** Kafka-driven worker that recalculates entitlement balances on lifecycle events. Subscribes to three topics (system, ingest, balance-worker). Converts direct lifecycle events (grant created/voided, entitlement created/reset, batched ingest) into RecalculateEvent on the balance-worker topic; a second handler consumes RecalculateEvent and calls handleEntitlementEvent for filter->fetch->snapshot pipeline. Uses LRU caches and high-watermark filter to avoid redundant ClickHouse queries.
- **Depends on:** openmeter/entitlement, openmeter/entitlement/metered, openmeter/credit/grant, openmeter/customer, openmeter/notification, openmeter/sink/flushhandler/ingestnotification/events, openmeter/watermill, pkg/framework/lockr
- **Key interfaces:** `Worker` (AddHandler(grouphandler.GroupEventHandler), Run(ctx) error)

### openmeter/subscription
- **Location:** `openmeter/subscription`
- **Responsibility:** Manages subscription lifecycle (Create, Update, Delete, Cancel, Continue, UpdateAnnotations) against a versioned plan-phase-RateCard model. Uses SubscriptionSpec (in-memory mutable spec) manipulated exclusively via the AppliesToSpec patch interface (ApplyTo). Workflow service (subscription/workflow/) orchestrates higher-level operations (CreateFromPlan, EditRunning, ChangeToPlan). Addon service (subscription/addon/) manages quantity-based addons. Per-customer pg_advisory_lock enforced before writes.
- **Depends on:** openmeter/productcatalog, openmeter/customer, openmeter/entitlement, openmeter/ent/db, openmeter/watermill/eventbus, pkg/framework/lockr, pkg/models
- **Key interfaces:** `Service` (Get, GetView, List, ExpandViews, Create, Update, Delete, Cancel, Continue, UpdateAnnotations, RegisterHook), `subscriptionworkflow.Service` (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity)

### openmeter/notification
- **Location:** `openmeter/notification`
- **Responsibility:** Defines all notification domain types (Channel, Rule, Event, EventPayload with union dispatch, EventDeliveryStatus) and service interfaces (Service = ChannelService + RuleService + EventService + FeatureService). EventHandler interface (EventDispatcher + EventReconciler + Start/Close) drives the dispatch and reconciliation loop. consumer/ sub-package contains the Watermill consumer that subscribes to system events topic and dispatches to Svix.
- **Depends on:** openmeter/productcatalog/feature, openmeter/entitlement, openmeter/billing, openmeter/ent/db, openmeter/watermill, pkg/models
- **Key interfaces:** `Service` (ListChannels, CreateChannel, DeleteChannel, GetChannel, UpdateChannel, ListRules, CreateRule, DeleteRule, GetRule, UpdateRule, ListEvents, GetEvent, CreateEvent, ResendEvent, ListEventsDeliveryStatus, GetEventDeliveryStatus, UpdateEventDeliveryStatus, ListFeature), `EventHandler` (Start() error, Close() error, Dispatch(ctx, Event) error, Reconcile(ctx) error)

### openmeter/ledger
- **Location:** `openmeter/ledger`
- **Responsibility:** Double-entry ledger for customer financial balances (FBO, Receivable, Accrued) and business accounts (Wash, Earnings, Brokerage). Transaction inputs are constructed exclusively via transactions.ResolveTransactions with typed template structs. account/ sub-package provides account CRUD, historical/ runs the engine, chargeadapter/ bridges charge events to ledger postings. noop/ provides zero-value implementations of all interfaces when credits.enabled=false.
- **Depends on:** openmeter/customer, openmeter/ent/db, pkg/currencyx, pkg/framework/lockr, pkg/framework/entutils
- **Key interfaces:** `AccountResolver` (EnsureCustomerAccounts, GetCustomerAccounts, EnsureBusinessAccounts), `Ledger` (CommitGroup, QueryBalance)

### openmeter/credit
- **Location:** `openmeter/credit`
- **Responsibility:** Manages credit grants and balance snapshots for metered entitlements. CreditConnector (= BalanceConnector + GrantConnector) is the public facade. engine/ sub-package computes grant burn-down without I/O. Granularity truncation (time.Minute) applied to all effective times. Period cache built once in buildEngineForOwner; used by UsageQuerier closure to avoid per-query DB calls. Acquires LockOwnerForTx before any grant/snapshot write.
- **Depends on:** openmeter/streaming, openmeter/credit/grant, openmeter/credit/balance, openmeter/credit/engine, openmeter/ent/db, openmeter/watermill/eventbus, pkg/framework/transaction
- **Key interfaces:** `CreditConnector` (GetBalanceAt, GetBalanceForPeriod, ResetUsageForOwner, CreateGrant, VoidGrant)

### openmeter/productcatalog
- **Location:** `openmeter/productcatalog`
- **Responsibility:** Defines the product catalog: features (meter-backed usage features), plans (versioned multi-phase rate card collections), plan addons, addons, and rate cards (usage-based and flat-fee pricing). Each entity has a Service interface, Ent adapter, and httpdriver package. subscription/ sub-package provides PlanSubscriptionService that orchestrates plan-aware subscription lifecycle.
- **Depends on:** openmeter/meter, openmeter/ent/db, openmeter/watermill, pkg/models, pkg/pagination
- **Key interfaces:** `feature.FeatureConnector` (CreateFeature, UpdateFeature, ArchiveFeature, ListFeatures, GetFeature, ResolveFeatureMeters), `plan.Service` (ListPlans, CreatePlan, DeletePlan, GetPlan, UpdatePlan, PublishPlan, ArchivePlan, NextPlan), `plansubscription.PlanSubscriptionService` (Create, Migrate, Change)

### openmeter/app
- **Location:** `openmeter/app`
- **Responsibility:** Marketplace registry and runtime lifecycle for installed billing apps. Service manages app install/uninstall, OAuth2 flows, and customer data delegation. AppFactory self-registers at constructor via RegisterMarketplaceListing. Concrete implementations under stripe/, sandbox/, and custominvoicing/ all embed AppBase and implement InvoicingApp. In-memory registry (not DB) for marketplace listings.
- **Depends on:** openmeter/customer, openmeter/secret, openmeter/billing, openmeter/ent/db, openmeter/watermill/eventbus, pkg/pagination, pkg/models
- **Key interfaces:** `Service` (RegisterMarketplaceListing, GetMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, InstallMarketplaceListing, GetMarketplaceListingOauth2InstallURL, AuthorizeMarketplaceListingOauth2Install, CreateApp, GetApp, UpdateAppStatus, UpdateApp, ListApps, UninstallApp, ListCustomerData, EnsureCustomer, DeleteCustomer), `App` (GetAppBase, GetID, GetType, GetName, GetStatus, GetListing, UpdateAppConfig, ValidateCapabilities, GetCustomerData, UpsertCustomerData, DeleteCustomerData)

### openmeter/meter
- **Location:** `openmeter/meter`
- **Responsibility:** Defines meters (event aggregation rules: event type, aggregation function COUNT/SUM/MAX/UNIQUE_COUNT, optional group-by JSON paths, optional value property). ManageService extends Service with CreateMeter/UpdateMeter/DeleteMeter and RegisterPreUpdateMeterHook. Soft-delete via DeletedAt. Publishes MeterCreateEvent/MeterUpdateEvent/MeterDeleteEvent after mutations. ParseEvent extracts value and group-by fields from CloudEvent JSON at ingest time.
- **Depends on:** openmeter/ent/db, openmeter/watermill/eventbus, pkg/filter, pkg/pagination, pkg/models
- **Key interfaces:** `Service` (ListMeters, GetMeterByIDOrSlug), `ManageService` (ListMeters, GetMeterByIDOrSlug, CreateMeter, UpdateMeter, DeleteMeter, RegisterPreUpdateMeterHook)

### openmeter/streaming
- **Location:** `openmeter/streaming`
- **Responsibility:** Defines the streaming.Connector interface for querying meter aggregations and listing raw events from ClickHouse, and for namespace lifecycle (embeds namespace.Handler). Concrete implementation in clickhouse/ uses sqlbuilder query structs with toSQL() methods. retry/ wraps with retry logic. testutils/ provides MockStreamingConnector for unit tests.
- **Depends on:** openmeter/meter, openmeter/namespace, pkg/models, pkg/filter
- **Key interfaces:** `Connector` (CountEvents, ListEvents, ListEventsV2, ListSubjects, ListGroupByValues, QueryMeter, BatchInsert, ValidateJSONPath, CreateNamespace, DeleteNamespace)

### openmeter/ingest
- **Location:** `openmeter/ingest`
- **Responsibility:** CloudEvent ingestion pipeline. Collector interface (Ingest, Close) receives single events and forwards to Kafka. DeduplicatingCollector wraps any Collector with Redis or in-memory deduplication. ingestadapter/ decorates with OTel telemetry. kafkaingest/ implements Collector using confluent-kafka-go with JSON serializer; provisions Kafka topics on each Ingest call. httpdriver/ translates multi-format HTTP requests (application/cloudevents+json, application/cloudevents-batch+json) into Service calls.
- **Depends on:** openmeter/dedupe, pkg/kafka, openmeter/watermill/eventbus
- **Key interfaces:** `Collector` (Ingest(ctx, namespace, event) error, Close())

### openmeter/sink
- **Location:** `openmeter/sink`
- **Responsibility:** High-throughput Kafka-to-ClickHouse sink worker. Sink struct consumes Kafka partitions via confluent-kafka-go, buffers messages in SinkBuffer, flushes in three-phase order (ClickHouse BatchInsert -> Kafka offset commit -> Redis dedupe), then fires FlushEventHandler in a goroutine with FlushSuccessTimeout context. NamespacedMeterCache caches meter definitions and refreshes periodically to avoid per-message DB queries.
- **Depends on:** openmeter/streaming, openmeter/ingest/kafkaingest, openmeter/meter, openmeter/dedupe, openmeter/watermill, pkg/kafka/metrics
- **Key interfaces:** `Storage` (BatchInsert(ctx, []SinkMessage) error), `Sink` (Run(ctx) error, Close() error)

### openmeter/watermill
- **Location:** `openmeter/watermill`
- **Responsibility:** Kafka-backed pub-sub abstraction. eventbus.Publisher routes typed domain events to three named Kafka topics (IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic) by event-name prefix (EventVersionSubsystem prefix determines routing). grouphandler.NoPublishingHandler multiplexes by CloudEvents ce_type header; unknown types silently dropped. router.NewDefaultRouter wires fixed middleware stack (PoisonQueue, DLQ telemetry, CorrelationID, Recoverer, Retry, ProcessingTimeout+RestoreContext, HandlerMetrics). CloudEvents 1.0 wire format via marshaler.CloudEventMarshaler.
- **Depends on:** github.com/ThreeDotsLabs/watermill, github.com/ThreeDotsLabs/watermill-kafka/v3, github.com/confluentinc/confluent-kafka-go/v2
- **Key interfaces:** `eventbus.Publisher` (Publish(ctx, Event) error, WithContext(ctx) ContextPublisher, Marshaler() marshaler.Marshaler)

### openmeter/namespace
- **Location:** `openmeter/namespace`
- **Responsibility:** Multi-tenancy infrastructure. Manager fans out CreateNamespace/DeleteNamespace to all registered Handler implementations (ClickHouse streaming, Kafka ingest, Ledger). Handlers are registered dynamically via RegisterHandler before CreateDefaultNamespace at startup. Fan-out uses errors.Join (no short-circuit on failure). namespacedriver sub-package provides StaticNamespaceDecoder (named string type, not struct) for self-hosted single-namespace deployments.
- **Depends on:** pkg/models
- **Key interfaces:** `Manager` (CreateNamespace, DeleteNamespace, CreateDefaultNamespace, RegisterHandler, GetDefaultNamespace), `Handler` (CreateNamespace(ctx, name) error, DeleteNamespace(ctx, name) error)

### openmeter/ent/schema
- **Location:** `openmeter/ent/schema`
- **Responsibility:** Ent schema definitions (source of truth for database schema). ~30 entity files covering billing invoices/lines, customers, entitlements, grants, features, subscriptions, plans, addons, notifications, LLM cost prices, ledger accounts/transactions, meters, subjects, secrets, charges. Used by Atlas to generate SQL migrations. All entities use shared mixins: IDMixin (ULID char(26)), NamespaceMixin, TimeMixin.
- **Depends on:** entgo.io/ent

### openmeter/server
- **Location:** `openmeter/server`
- **Responsibility:** Chi-based HTTP server assembling v1 and v3 REST APIs behind a shared middleware stack (auth, OpenAPI validation via kin-openapi OapiRequestValidatorWithOptions with NoopAuthenticationFunc, CORS for portal paths). Mounts v3 API (via v3server.NewServer + RegisterRoutes) first in its own Chi group, then v1 API (via api.HandlerWithOptions) in a separate Chi group. router/ sub-package is the pure v1 endpoint delegation layer implementing api.ServerInterface by calling typed domain handlers.
- **Depends on:** openmeter/server/router, api/v3/server, api (generated stubs), openmeter/* httpdriver packages, pkg/framework/transport/httptransport
- **Key interfaces:** `router.Config`

### api/v3/server
- **Location:** `api/v3/server`
- **Responsibility:** v3 HTTP server: validates Config, wires all domain service dependencies into typed handler structs (metersHandler, customersHandler, billingProfilesHandler, chargesHandler, etc.), registers OAS validation middleware (oasmiddleware.ValidateRequest) on Chi router, and delegates every generated ServerInterface method via typed .With(params).ServeHTTP(w, r) pattern. Credits feature flag gated at both NewServer constructor level (noop wiring) and route dispatch level (explicit enabled check).
- **Depends on:** api/v3 (generated stubs), api/v3/handlers/*, openmeter/billing, openmeter/customer, openmeter/meter, openmeter/subscription, openmeter/productcatalog, openmeter/llmcost, openmeter/app, openmeter/ledger, pkg/framework/transport/httptransport

### api/spec
- **Location:** `api/spec`
- **Responsibility:** TypeSpec source files defining both v1 (packages/legacy/) and v3 (packages/aip/) OpenAPI specifications. Compilation produces api/openapi.yaml, api/openapi.cloud.yaml, api/v3/openapi.yaml. Also produces Go client (api/client/go/client.gen.go) and JavaScript SDK (api/client/javascript/). Route and tag bindings are declared only in root openmeter.tsp files, not in domain sub-folder operation files.

### api (generated stubs)
- **Location:** `api`
- **Responsibility:** Generated OpenAPI YAML specs and Go server stubs. api/api.gen.go contains v1 Go server stubs (types, request/response objects, ServerInterface) generated from api/openapi.yaml via oapi-codegen. api/v3/api.gen.go contains v3 stubs with nullable-type=true and always-prefix-enum-values from api/v3/openapi.yaml. api/client/go/client.gen.go is the external Go SDK generated from api/openapi.cloud.yaml. DO NOT EDIT any of these files.
- **Depends on:** api/spec (TypeSpec source)

### api/v3/handlers
- **Location:** `api/v3/handlers`
- **Responsibility:** v3 API handler packages organized per resource group: meters, customers, customers/billing, customers/charges, customers/credits, customers/entitlementaccess, billingprofiles, plans, plans/planaddons, subscriptions, addons, apps, features, featurecost, llmcost, taxcodes, currencies, events. Each sub-package implements the relevant ServerInterface methods using httptransport.Handler[Request,Response] pipeline and delegates to domain services.
- **Depends on:** api/v3 (generated types), pkg/framework/transport/httptransport, openmeter/* domain services

### tools/migrate
- **Location:** `tools/migrate`
- **Responsibility:** Atlas-generated SQL migration files in tools/migrate/migrations/ using golang-migrate format (timestamped .up.sql/.down.sql pairs plus atlas.sum hash chain). migrate.go wraps golang-migrate for use at app startup. viewgen sub-command generates ClickHouse view SQL. Atlas config in atlas.hcl points to ent://openmeter/ent/schema as schema source and file://tools/migrate/migrations as migrations dir.
- **Depends on:** openmeter/ent/schema (via Atlas diff), github.com/golang-migrate/migrate/v4

### pkg/framework
- **Location:** `pkg/framework`
- **Responsibility:** Shared low-level infrastructure layer used by all domain packages. httptransport provides generic Handler[Request,Response] struct (decode -> operation -> encode with ErrorEncoder chain). entutils provides TransactingRepo/TransactingRepoWithNoValue (ctx-propagated Ent transaction reuse with savepoints), Ent schema mixins (IDMixin ULID, NamespaceMixin, TimeMixin, MetadataMixin), and ULID/PGULID type utilities. lockr provides pg_advisory_xact_lock and connection-scoped advisory locks. commonhttp provides RFC 7807 error encoding (GenericErrorEncoder chain). clickhouseotel/tracex provide OTel instrumentation helpers.
- **Depends on:** entgo.io/ent, go.opentelemetry.io/otel
- **Key interfaces:** `httptransport.Handler[Request, Response]` (ServeHTTP(w, r), Chain(outer, ...others) Handler[Request, Response]), `entutils.TxCreator` (Tx(ctx) (TxContext, error)), `lockr.Locker` (LockForTX(ctx, key) error)

### pkg/models
- **Location:** `pkg/models`
- **Responsibility:** Foundational domain primitive library with zero imports from openmeter/* domain packages. Provides: NamespacedID/NamespacedKey identity types, ManagedModel/CadencedModel base structs, GenericError typed sentinels (GenericNotFoundError -> 404, GenericValidationError -> 400, GenericConflictError -> 409, GenericForbiddenError -> 403, GenericUnauthorizedError -> 401, GenericNotImplementedError -> 501, GenericPreConditionFailedError -> 412, GenericStatusFailedDependencyError -> 424), ValidationIssue (immutable with-chain builder, private constructor), ServiceHookRegistry[T] (re-entrant loop prevention via pointer-identity context key), RFC 7807 StatusProblem, FieldDescriptor tree (WithPrefix composition, JSONPath generation), CadenceList[T] with GetOverlaps/IsContinuous.
- **Depends on:** pkg/treex, pkg/pagination
- **Key interfaces:** `ServiceHooks[T]` (RegisterHooks(...ServiceHook[T]), PreCreate, PostCreate, PreUpdate, PostUpdate, PreDelete, PostDelete), `Validator` (Validate() error)

### openmeter/portal
- **Location:** `openmeter/portal`
- **Responsibility:** Issues and validates short-lived HS256 JWT portal tokens scoped to namespace, subject, and optional meter slug allowlist. ListTokens and InvalidateToken are intentionally unimplemented (noop). Meter slug validation happens in the httphandler operation against meter.Service, not inside portal.Service itself.
- **Depends on:** openmeter/ent/db, pkg/pagination, pkg/models
- **Key interfaces:** `Service` (CreateToken, Validate, ListTokens, InvalidateToken)

### openmeter/llmcost
- **Location:** `openmeter/llmcost`
- **Responsibility:** LLM cost price management: persists global (synced) prices and per-namespace overrides in llmcostprice Ent entity. Service resolves effective prices with namespace-override precedence. sync/ sub-package runs a four-phase pipeline (Fetch -> Normalize -> Deduplicate -> Reconcile) to synchronize prices from external sources (models.dev) with multi-source agreement threshold. All monetary values use alpacadecimal.Decimal. NormalizeModelID must be called before any price store or resolve.
- **Depends on:** openmeter/ent/db, pkg/filter, pkg/pagination, pkg/models
- **Key interfaces:** `Service` (ListPrices, GetPrice, ResolvePrice, CreateOverride, DeleteOverride, ListOverrides)

### openmeter/subject
- **Location:** `openmeter/subject`
- **Responsibility:** Manages subjects (usage measurement subjects, analogous to users/devices). Provides Create, Update, GetByKey, GetById, GetByIdOrKey, List, Delete backed by an Ent adapter. Exposes ServiceHooks for lifecycle events.
- **Depends on:** openmeter/ent/db, openmeter/watermill/eventbus, pkg/models
- **Key interfaces:** `Service` (Create, Update, GetByKey, GetById, GetByIdOrKey, List, Delete, RegisterHooks)

### openmeter/progressmanager
- **Location:** `openmeter/progressmanager`
- **Responsibility:** Tracks progress of long-running operations (e.g. ClickHouse query export). Provides GetProgress and UpsertProgress backed by an Ent adapter.
- **Depends on:** openmeter/ent/db, pkg/models
- **Key interfaces:** `Service` (GetProgress, UpsertProgress)

### openmeter/taxcode
- **Location:** `openmeter/taxcode`
- **Responsibility:** Manages tax codes used during invoice line processing. Provides a Service interface backed by a repository.
- **Depends on:** openmeter/ent/db

### openmeter/secret
- **Location:** `openmeter/secret`
- **Responsibility:** Stores and retrieves encrypted secrets (e.g. Stripe API keys) for installed apps. Backed by an Ent adapter.
- **Depends on:** openmeter/ent/db

### openmeter/cost
- **Location:** `openmeter/cost`
- **Responsibility:** Computes feature cost by querying meter usage. Adapter interface has a single QueryFeatureCost method backed by ClickHouse. Used for the feature cost API endpoint.
- **Depends on:** openmeter/streaming, openmeter/productcatalog/feature
- **Key interfaces:** `Adapter` (QueryFeatureCost(ctx, namespace, feature, params) ([]FeatureCostRow, error))

### openmeter/currencies
- **Location:** `openmeter/currencies`
- **Responsibility:** Manages custom currencies and cost bases for billing. Backed by Ent with transaction support.
- **Depends on:** openmeter/ent/db, pkg/framework/entutils
- **Key interfaces:** `Adapter` (ListCustomCurrencies, CreateCurrency, CreateCostBasis, ListCostBases, Tx)

### e2e
- **Location:** `e2e`
- **Responsibility:** End-to-end test suite that runs against a locally started OpenMeter instance (docker-compose based). Tests cover plans, subscriptions, entitlements, addons, ledger backfill, multi-subject scenarios. Uses the generated Go SDK client.
- **Depends on:** api/client/go (Go SDK)

### deploy/charts
- **Location:** `deploy/charts`
- **Responsibility:** Helm charts for Kubernetes deployment. openmeter/ chart covers the main API server and all workers. benthos-collector/ chart covers the Benthos collector binary. Validated by helm lint in CI.

## File Placement

| Component Type | Location | Naming | Example |
|---------------|----------|--------|---------|
| binary entrypoint | `cmd/<binary>/` | `main.go + wire.go + wire_gen.go + version.go` | `cmd/server/main.go, cmd/billing-worker/wire.go` |
| domain service interface | `openmeter/<domain>/` | `service.go or <domain>.go` | `openmeter/billing/service.go, openmeter/customer/service.go, openmeter/meter/service.go` |
| domain adapter interface | `openmeter/<domain>/` | `adapter.go` | `openmeter/billing/adapter.go, openmeter/customer/adapter.go, openmeter/charges/adapter.go` |
| Ent/PostgreSQL adapter implementation | `openmeter/<domain>/adapter/` | `adapter.go + domain-specific files` | `openmeter/billing/adapter/adapter.go, openmeter/customer/adapter/adapter.go` |
| service implementation | `openmeter/<domain>/service/` | `service.go` | `openmeter/billing/service/service.go, openmeter/customer/service/service.go` |
| v1 HTTP handler | `openmeter/<domain>/httpdriver/ or openmeter/<domain>/httphandler/` | `handler.go + domain-specific files` | `openmeter/billing/httpdriver/handler.go, openmeter/meter/httphandler/` |
| v3 API handler | `api/v3/handlers/<resource>/` | `handler.go` | `api/v3/handlers/customers/, api/v3/handlers/meters/, api/v3/handlers/billingprofiles/` |
| Wire DI provider set | `app/common/` | `<domain>.go` | `app/common/billing.go, app/common/customer.go, app/common/subscription.go` |
| generated code | `openmeter/ent/db/, **/wire_gen.go, **/*.gen.go, api/api.gen.go, api/v3/api.gen.go` | `*.gen.go or wire_gen.go or ent/db/` | `openmeter/ent/db/, cmd/server/wire_gen.go, openmeter/billing/derived.gen.go, api/api.gen.go` |
| Ent entity schema | `openmeter/ent/schema/` | `<entity>.go` | `openmeter/ent/schema/customer.go, openmeter/ent/schema/billing.go` |
| Atlas SQL migrations | `tools/migrate/migrations/` | `<timestamp>_<name>.up.sql / <timestamp>_<name>.down.sql` | `tools/migrate/migrations/20240826120919_init.up.sql` |
| test utilities | `openmeter/<domain>/testutils/` | `testutils/ sub-package` | `openmeter/billing/testutils/, openmeter/customer/testutils/, openmeter/streaming/testutils/` |
| domain unit tests | `openmeter/<domain>/` | `*_test.go alongside source files` | `openmeter/billing/stdinvoice_test.go, openmeter/meter/service_test.go` |
| integration test helpers | `test/<domain>/` | `per-domain sub-directory` | `test/billing/, test/customer/, test/subscription/` |
| TypeSpec API source | `api/spec/packages/aip/ (v3) and api/spec/packages/legacy/ (v1)` | `*.tsp files` | `api/spec/packages/aip/src/openmeter.tsp` |

## Naming Conventions

- **service interface**: PascalCase named Service or <Noun>Service (e.g. `billing.Service`, `customer.Service`, `meter.ManageService`, `notification.ChannelService`)
- **adapter interface**: PascalCase named Adapter or <Noun>Adapter (e.g. `billing.Adapter`, `customer.Adapter`, `charges.Adapter`, `currencies.Adapter`)
- **legacy connector interface**: PascalCase named Connector (e.g. `streaming.Connector`, `credit.CreditConnector`, `meteredentitlement.Connector`)
- **HTTP handler interface**: Handler type in httpdriver/ or httphandler/ package (e.g. `billing/httpdriver.Handler`, `customer/httpdriver.Handler`)
- **service input types**: <Verb><Noun>Input suffix (e.g. `CreateCustomerInput`, `ListInvoicesInput`, `GetProfileInput`, `DeleteMeterInput`)
- **Wire provider sets**: PascalCase var exported from app/common (e.g. `var Billing = wire.NewSet(...)`, `var Customer = wire.NewSet(...)`, `var Subscription = wire.NewSet(...)`)
- **registry structs**: <Domain>Registry struct grouping related services (e.g. `BillingRegistry{Billing billing.Service, Charges *ChargesRegistry}`, `AppRegistry{Service, Stripe, CustomInvoicing}`)
- **Kafka topic constants**: SystemEventsTopic, IngestEventsTopic, BalanceWorkerEventsTopic (e.g. `WorkerOptions.SystemEventsTopic`, `WorkerOptions.BalanceWorkerEventsTopic`)
- **domain errors**: models.Generic* typed sentinel wrappers (e.g. `models.GenericNotFoundError`, `models.GenericValidationError`, `models.GenericConflictError`, `billing.NotFoundError`)
- **event types**: <Domain><Action>Event PascalCase with EventName() string method (e.g. `MeterCreateEvent`, `InvoiceCreatedEvent`, `AdvanceChargesEvent`, `RecalculateEvent`)
- **test file naming**: *_test.go alongside source or testutils/ sub-package (e.g. `openmeter/billing/stdinvoice_test.go`, `openmeter/customer/testutils/`)
- **migration files**: <timestamp>_<name>.up.sql and <timestamp>_<name>.down.sql (e.g. `20240826120919_init.up.sql`, `20240917172257_billing-entities.up.sql`)