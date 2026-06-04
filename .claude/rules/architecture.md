## Components

### cmd/server (HTTP API binary)
- **Location:** `cmd/server`
- **Responsibility:** main.go (346 lines) is a thin Wire+lifecycle launcher: it parses Viper config, calls the Wire-generated initializeApplication, runs app.Migrate(ctx), then provisions the default namespace and a default billing profile. It is the ONLY binary that registers namespace.Handler implementations — it registers the Ledger handler then the KafkaIngest handler via manager.RegisterHandler BEFORE calling initNamespace so the default namespace receives every handler's provisioning (handler order is load-bearing). It assembles router.Config from ~40 domain services and runs a multi-actor oklog/run.Group (HTTP API server, telemetry server, Kafka producer, a NotificationEventHandler goroutine, and a termination checker). No business logic lives here. wire.go (//go:build wireinject) lists only app/common provider sets; wire_gen.go is generated.
- **Depends on:** app/common (Wire DI layer), app/config (Viper configuration), openmeter/server (HTTP server + v1 router), api/v3 (v3 HTTP API), openmeter/namespace (multi-tenancy)
- **Key interfaces:** `initializeApplication` (initializeApplication), `run.Group lifecycle` (Run, Add)

### cmd/sink-worker
- **Location:** `cmd/sink-worker`
- **Responsibility:** main.go (127 lines) loads and Validate()s Viper config, Wire-initializes, then runs two run.Group actors with reverse shutdown order: the TelemetryServer.ListenAndServe and the Sink.Run loop (Kafka -> ClickHouse). All sink logic lives in openmeter/sink; this binary constructs nothing domain-level directly. Uses the root signal-aware ctx, never context.Background() in actors.
- **Depends on:** app/common (Wire DI layer), app/config (Viper configuration), openmeter/meter + ingest + sink + streaming (usage pipeline)
- **Key interfaces:** `initializeApplication` (initializeApplication)

### cmd/billing-worker
- **Location:** `cmd/billing-worker`
- **Responsibility:** main.go (100 lines) runs post-migration provisioning before the consumer loop: app.Migrate(ctx) -> EnsureBusinessAccounts(ledger wash/earnings/brokerage accounts) -> SandboxProvisioner, then wires AppRegistry (Sandbox/Stripe/CustomInvoicing) and starts the Watermill consumer loop that advances invoices and charges off the Kafka system topic. EnsureBusinessAccounts must complete before run.Group.Run() or ledger postings fail.
- **Depends on:** app/common (Wire DI layer), openmeter/billing (billing domain), openmeter/ledger (double-entry ledger), openmeter/app (marketplace apps)
- **Key interfaces:** `initializeApplication` (initializeApplication)

### cmd/balance-worker
- **Location:** `cmd/balance-worker`
- **Responsibility:** main.go (88 lines) Wire-initializes and runs the entitlement balanceworker, a Kafka consumer (on the balance-worker topic) that recalculates metered entitlement grant burn-down on lifecycle events. Binary identity is set via version.go (ldflags) + common.NewMetadata(version, "balance-worker") feeding OTel service.name.
- **Depends on:** app/common (Wire DI layer), openmeter/entitlement (entitlement domain)
- **Key interfaces:** `initializeApplication` (initializeApplication)

### cmd/notification-service
- **Location:** `cmd/notification-service`
- **Responsibility:** main.go (156 lines) Wire-assembles services, then MANUALLY constructs the Watermill Kafka subscriber and notification consumer in main.go (so the consumer group name comes from runtime config, unavailable at Wire compile time) and runs them in run.Group. Uses app.EventPublisher.Marshaler() for deserialization. Dispatches notification payloads to Svix webhooks.
- **Depends on:** app/common (Wire DI layer), openmeter/notification (notification domain), openmeter/watermill (event bus)
- **Key interfaces:** `initializeApplication` (initializeApplication)

### cmd/jobs (admin CLI)
- **Location:** `cmd/jobs`
- **Responsibility:** Cobra root command (main.go, 76 lines) whose PersistentPreRunE calls internal.InitializeApplication once before any sub-command, wiring the whole app into internal.App package-level globals. Sub-commands (entitlement.RootCommand, billing.Cmd, ledger.Cmd, llmcost.Cmd, quickstart.Cmd, migrate.RootCommand) source services from internal.App and use cmd.Context() (signal-aware root ctx). The ledger backfill sub-command intentionally bypasses Wire DI, constructing concrete ledger adapters + lockr.Locker directly because Wire provides noop ledger resolvers when credits.enabled=false. Deferred internal.AppShutdown releases resources on every exit path.
- **Depends on:** app/common (Wire DI layer), openmeter/billing (billing domain), openmeter/entitlement (entitlement domain), openmeter/ledger (double-entry ledger), openmeter/llmcost (LLM cost prices), tools/migrate (Atlas migrations)
- **Key interfaces:** `rootCmd (Cobra)` (versionCommand, entitlement.RootCommand, billing.Cmd, ledger.Cmd, llmcost.Cmd, quickstart.Cmd, migrate.RootCommand)

### app/common (Wire DI layer)
- **Location:** `app/common`
- **Responsibility:** Houses all Google Wire provider sets and constructor functions for every binary. One file per domain area (billing.go, customer.go, ledger.go, charges.go, entitlement.go, subscription.go, notification.go, meter.go, streaming.go, app.go, etc.) plus per-binary composite files (openmeter_server.go, openmeter_billingworker.go, openmeter_balanceworker.go, openmeter_sinkworker.go, openmeter_notification.go). Registers cross-domain ServiceHooks and customer RequestValidators as construction SIDE-EFFECTS (e.g. customerService.RegisterHooks, billing.Service.RegisterLineEngine) to break billing/customer/subscription/ledger import cycles — side-effects invisible to Wire's compile-time type graph. Returns noop implementations (not nil) when features are disabled; credits.enabled=false is guarded independently across ledger.go, customer.go, and the ChargesRegistry in billing.go.
- **Depends on:** openmeter/billing (billing domain), openmeter/customer (customer domain), openmeter/ledger (double-entry ledger), openmeter/entitlement (entitlement domain), openmeter/subscription (subscription domain), openmeter/notification (notification domain), openmeter/app (marketplace apps), openmeter/meter + ingest + sink + streaming (usage pipeline), app/config (Viper configuration), pkg/framework (shared infrastructure)
- **Key interfaces:** `Wire provider sets` (NewBillingRegistry, NewCustomerLedgerServiceHook, NewLedgerNamespaceHandler, NewKafkaIngestNamespaceHandler)

### app/config (Viper configuration)
- **Location:** `app/config`
- **Responsibility:** Viper-based configuration structs, defaults, and Validate() methods for every application concern (33 files: aggregation, apps, balanceworker, billing, billingworker, credits, customer, dedupe, entitlements, events, ingest, etc.). Single shared config.Configuration type used by all binaries; SetViperDefaults is the single registration point calling each Configure* sub-function — viper.SetDefault must not be called in cmd/*.
- **Depends on:** openmeter/meter + ingest + sink + streaming (usage pipeline), pkg/framework (shared infrastructure), pkg/models (domain primitives)
- **Key interfaces:** `Configuration` (Validate, SetViperDefaults)

### openmeter/billing (billing domain)
- **Location:** `openmeter/billing`
- **Responsibility:** Largest domain package. service.go defines the composite billing.Service interface composed of ProfileService (CreateProfile/GetDefaultProfile/ProvisionDefaultBillingProfile/ResolveStripeAppIDFromBillingProfile), CustomerOverrideService (UpsertCustomerOverride/GetCustomerApp), InvoiceService (InvoicePendingLines/AdvanceInvoice/ApproveInvoice/PaymentAuthorized/ForceCollectInvoice/RetryInvoice/DeleteInvoice/UpdateInvoice), InvoiceLineService (GetLinesForSubscription/SnapshotLineQuantity), LineEngineService (RegisterLineEngine/DeregisterLineEngine/OnMutableStandardLinesDeleted), and SplitLineGroupService. invoiceline.go owns the InvoiceLine tagged-union (private discriminator set only by NewStandardInvoiceLine/NewGatheringInvoiceLine). service/ drives the invoice lifecycle through a stateless.StateMachine pooled in sync.Pool (stdinvoicestate.go, sole writer of Invoice.Status); adapter/ is the Ent implementation; httpdriver/ exposes v1 HTTP. Sub-packages: charges/ (Charge/ChargeIntent tagged-union + generic charge state machine), worker/ (advance/collect/subscriptionsync loops), rating/, lineengine/, validators/.
- **Depends on:** openmeter/app (marketplace apps), openmeter/customer (customer domain), openmeter/productcatalog (catalog domain), openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/watermill (event bus), pkg/framework (shared infrastructure), openmeter/ledger (double-entry ledger)
- **Key interfaces:** `billing.Service` (CreateProfile, UpsertCustomerOverride, InvoicePendingLines, AdvanceInvoice, ApproveInvoice, PaymentAuthorized, RegisterLineEngine, SnapshotLineQuantity), `billing.InvoicingApp` (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice)

### openmeter/billing/charges (charges sub-domain)
- **Location:** `openmeter/billing/charges`
- **Responsibility:** Owns the Charge/ChargeIntent tagged-union (private meta.ChargeType discriminator set only by NewCharge[T]/NewChargeIntent[T]; accessed via AsFlatFeeCharge/AsUsageBasedCharge/AsCreditPurchaseCharge). charges.Service.Create/AdvanceCharges/ApplyPatches drive multi-step writes mixing reads, realization runs, and lockr advisory locks inside one ctx-carried transaction. statemachine/machine.go is the generic Machine[CHARGE,BASE,STATUS] with value-copy WithStatus/WithBase semantics. adapter/ reads charges via the ChargesSearchV1 ent.View union; helpers must wrap a.db in entutils.TransactingRepo. lock.go provides NewLockKeyForCharge for per-charge pg advisory locks.
- **Depends on:** openmeter/billing (billing domain), openmeter/ledger (double-entry ledger), pkg/framework (shared infrastructure), openmeter/billing/charges/meta
- **Key interfaces:** `charges.Service` (Create, AdvanceCharges, ListCustomersToAdvance, ApplyPatches, HandleCreditPurchaseExternalPaymentStateTransition), `Machine[CHARGE,BASE,STATUS]` (GetStatus, WithStatus, GetBase, WithBase)

### openmeter/customer (customer domain)
- **Location:** `openmeter/customer`
- **Responsibility:** Customer lifecycle with soft-delete via DeletedAt. service.go composes RequestValidatorService (RegisterRequestValidator) and CustomerService (ListCustomers, ListCustomerUsageAttributions, CreateCustomer, DeleteCustomer, GetCustomer, GetCustomerByUsageAttribution, UpdateCustomer). service/service.go wraps every mutation in transaction.Run so hooks and event publishes are atomic. Exposes two extension registries: a RequestValidatorRegistry (pre-mutation cross-domain blocking guards, errors.Join fan-out) and ServiceHooks[Customer] (post-lifecycle callbacks). Sub-package service/hooks/ holds entitlementvalidator and subjectcustomer hooks.
- **Depends on:** openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/ent (persistence schema + generated code), openmeter/watermill (event bus), pkg/models (domain primitives), pkg/framework (shared infrastructure)
- **Key interfaces:** `customer.Service` (ListCustomers, CreateCustomer, DeleteCustomer, GetCustomer, UpdateCustomer, GetCustomerByUsageAttribution, RegisterHooks, RegisterRequestValidator), `customer.RequestValidator` (ValidateDeleteCustomer, ValidateCreateCustomer, ValidateUpdateCustomer)

### openmeter/entitlement (entitlement domain)
- **Location:** `openmeter/entitlement`
- **Responsibility:** Feature entitlement management across metered (credit grant burn-down), boolean, and static sub-types. connector.go defines the Connector composing all sub-types with scheduling, override, supersede, and balance history (CreateEntitlement, OverrideEntitlement, ScheduleEntitlement, GetEntitlement, GetEntitlementValue, GetAccess, DeleteEntitlement, GetEntitlementsOfCustomer). Acquires pg_advisory_lock per customer before multi-row mutations. Sub-package balanceworker/ is a Kafka-driven worker recalculating balances on lifecycle events using LRU caches and a high-watermark filter to skip redundant ClickHouse queries. repository.go defines the persistence interface.
- **Depends on:** openmeter/productcatalog (catalog domain), openmeter/customer (customer domain), openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/credit (credit grants), openmeter/ent (persistence schema + generated code), openmeter/watermill (event bus), pkg/framework (shared infrastructure)
- **Key interfaces:** `entitlement.Connector` (CreateEntitlement, OverrideEntitlement, ScheduleEntitlement, GetEntitlement, GetEntitlementValue, GetAccess, DeleteEntitlement, GetEntitlementsOfCustomer, GetEntitlementOfCustomerAt)

### openmeter/subscription (subscription domain)
- **Location:** `openmeter/subscription`
- **Responsibility:** Subscription lifecycle against a versioned plan-phase-RateCard model. service.go splits into QueryService (Get, GetView, List, ExpandViews), CommandService (Create, Update, Delete, Cancel, Continue, UpdateAnnotations), and HookService. All mutation is through an in-memory SubscriptionSpec via the AppliesToSpec patch interface (ApplyTo) exclusively — never direct field mutation. A workflow service orchestrates higher-level operations (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon). Addon sub-service manages quantity-based addons.
- **Depends on:** openmeter/productcatalog (catalog domain), openmeter/customer (customer domain), openmeter/entitlement (entitlement domain), openmeter/ent (persistence schema + generated code), openmeter/watermill (event bus), pkg/framework (shared infrastructure)
- **Key interfaces:** `subscription.Service` (Get, GetView, List, ExpandViews, Create, Update, Delete, Cancel, Continue, UpdateAnnotations), `subscriptionworkflow.Service` (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity)

### openmeter/productcatalog (catalog domain)
- **Location:** `openmeter/productcatalog`
- **Responsibility:** Product catalog with per-entity Service/Adapter/httpdriver across sub-packages: feature/ (meter-backed usage features: FeatureConnector with CreateFeature, ArchiveFeature, ListFeatures, GetFeature, ResolveFeatureMeters — features are archived via archived_at, not soft-deleted), plan/ (versioned multi-phase rate-card collections: Service with ListPlans, CreatePlan, DeletePlan, GetPlan, UpdatePlan, PublishPlan, ArchivePlan, NextPlan), addon/, planaddon/, subscription/ (rate cards), featureresolver/, plus driver/ and http/ HTTP layers.
- **Depends on:** openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/ent (persistence schema + generated code), openmeter/watermill (event bus), pkg/models (domain primitives), pkg/pagination
- **Key interfaces:** `feature.FeatureConnector` (CreateFeature, ArchiveFeature, ListFeatures, GetFeature, ResolveFeatureMeters), `plan.Service` (ListPlans, CreatePlan, DeletePlan, GetPlan, UpdatePlan, PublishPlan, ArchivePlan, NextPlan)

### openmeter/ledger (double-entry ledger)
- **Location:** `openmeter/ledger`
- **Responsibility:** Double-entry ledger for customer financial balances (FBO, Receivable, Accrued) and business accounts (Wash, Earnings, Brokerage). account.go declares AccountResolver (GetCustomerAccounts, EnsureBusinessAccounts, EnsureSubAccount); primitives.go declares Ledger (CommitGroup, QueryBalance). Transaction inputs are constructed exclusively via transactions.ResolveTransactions with typed template structs enforcing debit=credit invariants. noop/ provides zero-value implementations wired when credits.enabled=false. The LedgerCustomerAccount link table is intentionally FK-less (Edges() returns nil) to avoid import cycles.
- **Depends on:** openmeter/customer (customer domain), openmeter/ent (persistence schema + generated code), pkg/currencyx, pkg/framework (shared infrastructure)
- **Key interfaces:** `ledger.AccountResolver` (GetCustomerAccounts, EnsureBusinessAccounts, EnsureSubAccount), `ledger.Ledger` (CommitGroup, QueryBalance)

### openmeter/credit (credit grants)
- **Location:** `openmeter/credit`
- **Responsibility:** Manages credit grants and balance snapshots for metered entitlements. connector.go defines CreditConnector composing BalanceConnector + GrantConnector as the public facade, configured via CreditConnectorConfig (GrantRepo, BalanceSnapshotService, OwnerConnector, StreamingConnector, Publisher, TransactionManager, Granularity=time.Minute). engine/ computes grant burn-down without I/O. All effective times are truncated to Granularity (time.Minute) before storage or computation.
- **Depends on:** openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/credit/grant, openmeter/credit/balance, openmeter/credit/engine, openmeter/ent (persistence schema + generated code), openmeter/watermill (event bus)
- **Key interfaces:** `credit.CreditConnector` (GetBalanceAt, GetBalanceForPeriod, ResetUsageForOwner, CreateGrant, VoidGrant)

### openmeter/app (marketplace apps)
- **Location:** `openmeter/app`
- **Responsibility:** Marketplace registry and runtime lifecycle for installed billing apps. service.go splits MarketplaceService (RegisterMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, InstallMarketplaceListing), AppLifecycleService (CreateApp, GetApp, UpdateAppStatus, UpdateApp, ListApps, UninstallApp), and CustomerDataService (ListCustomerData). AppFactory self-registers via RegisterMarketplaceListing in its constructor. Concrete implementations under stripe/, sandbox/, and custominvoicing/ embed AppBase and implement billing.InvoicingApp.
- **Depends on:** openmeter/customer (customer domain), openmeter/secret (secret store), openmeter/billing (billing domain), openmeter/ent (persistence schema + generated code), openmeter/watermill (event bus)
- **Key interfaces:** `app.Service` (RegisterMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, CreateApp, GetApp, UpdateAppStatus, UpdateApp, ListApps, UninstallApp, ListCustomerData)

### openmeter/notification (notification domain)
- **Location:** `openmeter/notification`
- **Responsibility:** Defines notification domain types (Channel, Rule, Event, EventPayload, EventDeliveryStatus) and service.go interfaces: ChannelService (ListChannels, CreateChannel, DeleteChannel, GetChannel, UpdateChannel), RuleService (ListRules, CreateRule, DeleteRule, GetRule, UpdateRule), EventService (ListEvents, GetEvent), and FeatureService. An EventHandler drives the dispatch + reconcile loop. consumer/ subscribes to the system events topic and delivers webhook payloads via Svix; webhook/ provides a noop fallback. A NullChannel sentinel prevents unfiltered delivery; payload version is pinned per event family.
- **Depends on:** openmeter/productcatalog (catalog domain), openmeter/entitlement (entitlement domain), openmeter/billing (billing domain), openmeter/ent (persistence schema + generated code), openmeter/watermill (event bus)
- **Key interfaces:** `notification.Service` (CreateChannel, ListChannels, UpdateChannel, ListRules, CreateRule, ListEvents, GetEvent)

### openmeter/meter + ingest + sink + streaming (usage pipeline)
- **Location:** `openmeter/meter`
- **Responsibility:** The usage-metering pipeline. meter/ defines meters (event aggregation rules) with Service (ListMeters, GetMeterByIDOrSlug) and ManageService (CreateMeter, UpdateMeter, DeleteMeter, RegisterHooks), plus ParseEvent for value/group-by extraction. ingest/ defines the Collector interface (Ingest, Close) forwarding CloudEvents to Kafka, with DeduplicatingCollector and OTel-decorating adapters. sink/ is the high-throughput Kafka-to-ClickHouse worker with strict three-phase flush ordering (ClickHouse BatchInsert -> Kafka offset commit -> Redis dedupe). streaming/ defines the Connector interface (CountEvents, ListEvents, ListSubjects, ListGroupByValues, QueryMeter, BatchInsert, ValidateJSONPath) with a clickhouse/ impl and a retry/ wrapper; it also owns the RawEvent struct kept in sync by hand with the ClickHouse DDL.
- **Depends on:** openmeter/ent (persistence schema + generated code), openmeter/dedupe (ingest dedup), openmeter/namespace (multi-tenancy), pkg/kafka, pkg/filter, openmeter/watermill (event bus)
- **Key interfaces:** `streaming.Connector` (CountEvents, ListEvents, ListSubjects, ListGroupByValues, QueryMeter, BatchInsert, ValidateJSONPath), `ingest.Collector` (Ingest, Close), `meter.Service / meter.ManageService` (ListMeters, GetMeterByIDOrSlug, CreateMeter, UpdateMeter, DeleteMeter, RegisterHooks)

### openmeter/watermill (event bus)
- **Location:** `openmeter/watermill`
- **Responsibility:** Kafka-backed pub-sub abstraction. eventbus/eventbus.go exposes the Publisher interface (Publish, WithContext, Marshaler, PublishIfNoError) that routes typed domain events to three named Kafka topics (IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic) by EventName() prefix via GeneratePublishTopic; unrecognized prefixes default to SystemEventsTopic with no error. grouphandler/grouphandler.go multiplexes by CloudEvents ce_type header, silently dropping unknown types (ACK) for rolling-deploy safety. router/router.go (NewDefaultRouter) wires a fixed middleware stack (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout, HandlerMetrics).
- **Depends on:** pkg/kafka, pkg/models (domain primitives)
- **Key interfaces:** `eventbus.Publisher` (Publish, WithContext, Marshaler, PublishIfNoError), `grouphandler.NoPublishingHandler` (Handle)

### openmeter/namespace (multi-tenancy)
- **Location:** `openmeter/namespace`
- **Responsibility:** Multi-tenancy infrastructure. namespace.go defines a Manager struct (NewManager) that fans out CreateNamespace/DeleteNamespace to all registered Handler implementations (ClickHouse streaming, Kafka ingest, Ledger) using errors.Join with no short-circuit. Handlers are registered via RegisterHandler before CreateDefaultNamespace at startup; GetDefaultNamespace returns the default name; the default namespace is protected from deletion. The Handler interface is CreateNamespace/DeleteNamespace.
- **Depends on:** pkg/models (domain primitives)
- **Key interfaces:** `namespace.Manager` (CreateNamespace, DeleteNamespace, CreateDefaultNamespace, RegisterHandler, GetDefaultNamespace), `namespace.Handler` (CreateNamespace, DeleteNamespace)

### openmeter/ent (persistence schema + generated code)
- **Location:** `openmeter/ent`
- **Responsibility:** Persistence layer source of truth. ent/schema/ holds 35 hand-written Ent entity definitions (billing invoices/lines, customers, entitlements, grants, features, subscriptions, plans, addons, notifications, llmcostprice, ledger accounts/transactions/entries, meters, subjects, secrets, charges) all using shared mixins IDMixin (ULID char(26)) + NamespaceMixin + TimeMixin. ent/db/ is 573 generated Ent ORM Go files (DO NOT EDIT). entc.go is the single codegen driver. ent.View schemas (e.g. ChargesSearchV1) generate query code but are not diffed by Atlas.
- **Depends on:** pkg/framework (shared infrastructure)
- **Key interfaces:** `Ent schema entities` (Mixin, Fields, Edges, Indexes, Annotations)

### openmeter/server (HTTP server + v1 router)
- **Location:** `openmeter/server`
- **Responsibility:** Chi-based HTTP server (server.go NewServer) assembling the v1 and v3 REST APIs behind a shared middleware stack. NewServer mounts the v3 API (v3server.NewServer + RegisterRoutes) in its own Chi Group with oasmiddleware request validation, then the v1 API (api.HandlerWithOptions) in a separate Group with kin-openapi OapiRequestValidatorWithOptions. The router/ sub-package is the pure v1 endpoint delegation layer implementing the generated api.ServerInterface by calling typed domain httpdriver handlers; router.Config aggregates ~40 domain service fields (split across router/addon.go, app.go, billing.go, credit.go, customer.go, debug.go, etc.).
- **Depends on:** api (generated v1 stubs), api/v3 (v3 HTTP API), app/config (Viper configuration), openmeter/billing (billing domain), openmeter/customer (customer domain), openmeter/meter + ingest + sink + streaming (usage pipeline), pkg/framework (shared infrastructure), pkg/models (domain primitives)
- **Key interfaces:** `server.Server` (NewServer), `router.Config`

### api/v3 (v3 HTTP API)
- **Location:** `api/v3`
- **Responsibility:** v3 AIP-style HTTP API. api.gen.go is the generated ServerInterface (DO NOT EDIT). server/server.go (NewServer + RegisterRoutes) validates Config, wires domain services into typed handler structs, registers OAS validation middleware on a Chi router, and delegates each ServerInterface method via the typed handler pattern with credits feature-flag gating at both constructor and route-dispatch levels. handlers/ holds per-resource handler packages (addons, apps, billingprofiles, currencies, customers, events, featurecost, features, llmcost, meters, plans, subscriptions, taxcodes) each using the httptransport.Handler[Request,Response] pipeline and a Goverter-generated convert.gen.go. filters/ provides shared AIP cursor/filter parsing; apierrors/ provides v3 error responses.
- **Depends on:** openmeter/billing (billing domain), openmeter/customer (customer domain), openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/subscription (subscription domain), openmeter/productcatalog (catalog domain), openmeter/llmcost (LLM cost prices), openmeter/app (marketplace apps), openmeter/ledger (double-entry ledger), pkg/framework (shared infrastructure)
- **Key interfaces:** `api/v3/server.Server` (NewServer, RegisterRoutes)

### api (generated v1 stubs + converters)
- **Location:** `api`
- **Responsibility:** Generated v1 API surface: api.gen.go is the generated v1 ServerInterface stubs (DO NOT EDIT) consumed by openmeter/server/router; convert.gen.go is Goverter-generated type conversion; types/ holds shared API types. client/ holds the generated SDKs: go/ (client.gen.go), javascript/ (@openmeter/sdk Node package, only frontend dependency — React appears as an optional context export), python/ (Poetry-managed). The v1 OpenAPI spec (openapi.yaml, openapi.cloud.yaml) is generated, never hand-edited.
- **Depends on:** api/spec (TypeSpec source + SDKs)
- **Key interfaces:** `api.ServerInterface` (HandlerWithOptions)

### api/spec (TypeSpec source + SDKs)
- **Location:** `api/spec`
- **Responsibility:** TypeSpec source files (separate Node/pnpm subproject) defining both v1 (packages/legacy/) and v3 (packages/aip/) OpenAPI specifications. Compilation (make gen-api) produces api/openapi.yaml, api/openapi.cloud.yaml, api/v3/openapi.yaml, the Go client, and the JavaScript SDK. Route and tag bindings are declared only in the root openmeter.tsp (aip) / main.tsp (legacy). packages/aip/lib/rules and packages/legacy/lib/rules hold custom TypeSpec lint rules.
- **Key interfaces:** `TypeSpec definitions`

### pkg/framework (shared infrastructure)
- **Location:** `pkg/framework`
- **Responsibility:** Shared low-level infrastructure. entutils/ provides TransactingRepo/TransactingRepoWithNoValue (ctx-propagated Ent transaction reuse with savepoints — reads *TxDriver from ctx and degrades to Self() when absent, with no error), the IDMixin/NamespaceMixin/TimeMixin schema mixins, and ULID utilities. transport/httptransport provides the generic Handler[Request,Response] decode->operate->encode pipeline. lockr/ wraps pg_advisory_xact_lock (requires an active Postgres tx in ctx; getTxClient verifies transaction_timestamp() != statement_timestamp()). commonhttp/ provides RFC 7807 error encoding via GenericErrorEncoder. transaction/ provides the Creator/Run abstraction. tracex/ provides OTel span helpers; pgdriver/ provides WithLockTimeout; operation/ and clickhouseotel/ round out the set.
- **Depends on:** pkg/models (domain primitives)
- **Key interfaces:** `entutils.TransactingRepo` (TransactingRepo, TransactingRepoWithNoValue), `httptransport.Handler` (ServeHTTP, Chain), `lockr.Locker` (LockForTX)

### pkg/models (domain primitives)
- **Location:** `pkg/models`
- **Responsibility:** Foundational domain primitive library (highest in-degree node at 229) with zero imports from openmeter/* domain packages. Provides NamespacedID/NamespacedKey identity types, ManagedModel/CadencedModel base structs, GenericError typed sentinels (GenericNotFoundError->404, GenericValidationError->400, GenericConflictError->409, GenericForbiddenError->403, GenericPreConditionFailedError->412, etc.), the immutable ValidationIssue with-chain builder, ServiceHookRegistry[T] (re-entrancy loop prevention via pointer-identity context key derived from fmt.Sprintf with %p), and the RFC 7807 StatusProblem.
- **Depends on:** pkg/treex, pkg/pagination
- **Key interfaces:** `models.ServiceHooks[T]` (RegisterHooks, PreCreate, PostCreate, PreUpdate, PostUpdate, PreDelete, PostDelete), `models.Validator` (Validate), `models.GenericError sentinels`

### openmeter/secret (secret store)
- **Location:** `openmeter/secret`
- **Responsibility:** Secret storage for app credentials (e.g. Stripe API keys). service.go SecretService manages app-scoped secrets: CreateAppSecret, UpdateAppSecret, GetAppSecret, DeleteAppSecret, returning typed SecretID handles rather than raw values.
- **Depends on:** openmeter/ent (persistence schema + generated code), pkg/framework (shared infrastructure)
- **Key interfaces:** `secret.SecretService` (CreateAppSecret, UpdateAppSecret, GetAppSecret, DeleteAppSecret)

### openmeter/portal (portal tokens)
- **Location:** `openmeter/portal`
- **Responsibility:** Issues and validates short-lived JWT portal tokens scoped to namespace, subject, and an optional meter-slug allowlist. service.go PortalTokenService provides CreateToken, Validate, ListTokens, InvalidateToken. Meter-slug allowlist existence checks are performed in the HTTP handler against meter.Service (not inside portal.Service) to avoid a portal->meter import.
- **Depends on:** pkg/framework (shared infrastructure), pkg/models (domain primitives)
- **Key interfaces:** `portal.PortalTokenService` (CreateToken, Validate, ListTokens, InvalidateToken)

### openmeter/subject (subject directory)
- **Location:** `openmeter/subject`
- **Responsibility:** Manages subjects (metering identities). service.go Service provides Create, Update, GetByKey, GetById, GetByIdOrKey, List, Delete over NamespacedID/NamespacedKey identities. Subject lifecycle side-effects flow through subject.Service.RegisterHooks; the subjectcustomer customer-hook bridges subjects to customers.
- **Depends on:** openmeter/ent (persistence schema + generated code), pkg/models (domain primitives), pkg/pagination
- **Key interfaces:** `subject.Service` (Create, Update, GetByKey, GetById, GetByIdOrKey, List, Delete)

### openmeter/llmcost (LLM cost prices)
- **Location:** `openmeter/llmcost`
- **Responsibility:** Persists and resolves LLM model prices (global synced prices + per-namespace overrides). service.go provides ListPrices, GetPrice, ResolvePrice (namespace-override precedence), CreateOverride, DeleteOverride, ListOverrides. normalize.go NormalizeModelID strips version/region suffixes and normalizes provider aliases — it must be called before storing or resolving any model ID.
- **Depends on:** openmeter/ent (persistence schema + generated code), pkg/framework (shared infrastructure), pkg/pagination
- **Key interfaces:** `llmcost.Service` (ListPrices, GetPrice, ResolvePrice, CreateOverride, DeleteOverride, ListOverrides)

### openmeter/cost (feature cost queries)
- **Location:** `openmeter/cost`
- **Responsibility:** service.go provides QueryFeatureCost(QueryFeatureCostInput) -> CostQueryResult, computing per-feature cost over usage using llmcost/feature unit-cost configuration. adapter/ holds the persistence/query layer.
- **Depends on:** openmeter/llmcost (LLM cost prices), openmeter/productcatalog (catalog domain), openmeter/meter + ingest + sink + streaming (usage pipeline)
- **Key interfaces:** `cost.Service` (QueryFeatureCost)

### openmeter/meterevent (raw event listing)
- **Location:** `openmeter/meterevent`
- **Responsibility:** service.go exposes ListEvents(ListEventsParams) -> []Event, a read path over the streaming connector for listing raw metered events (distinct from meter aggregation). httphandler/ exposes the HTTP surface.
- **Depends on:** openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/streaming
- **Key interfaces:** `meterevent.Service` (ListEvents)

### openmeter/meterexport (synthetic meter export)
- **Location:** `openmeter/meterexport`
- **Responsibility:** service.go provides GetTargetMeterDescriptor, ExportSyntheticMeterData (channel-based), and ExportSyntheticMeterDataIter (iter.Seq2 streaming) to export synthetic meter data as streaming.RawEvent streams for backfill/replay scenarios.
- **Depends on:** openmeter/meter + ingest + sink + streaming (usage pipeline), openmeter/streaming
- **Key interfaces:** `meterexport.Service` (GetTargetMeterDescriptor, ExportSyntheticMeterData, ExportSyntheticMeterDataIter)

### openmeter/dedupe (ingest dedup)
- **Location:** `openmeter/dedupe`
- **Responsibility:** Defines the Deduplicator interface and dedupe.Item composite key (namespace-source-id). redisdedupe/ implements Redis SET NX with TTL (with rawkey/keyhash/keyhash-migration modes via xxh3+base64url GetKeyHash); memorydedupe/ provides an LRU fallback when Redis is not configured. Updated as the third (last) phase of the sink flush, strictly after the Kafka offset commit.
- **Depends on:** pkg/redis, pkg/lrux
- **Key interfaces:** `dedupe.Deduplicator` (IsUnique, CheckUniqueBatch, Set)

### tools/migrate (Atlas migrations)
- **Location:** `tools/migrate`
- **Responsibility:** Atlas-generated SQL migration files in tools/migrate/migrations/ using golang-migrate format (timestamped .up.sql/.down.sql pairs plus an atlas.sum hash chain). migrate.go wraps golang-migrate (Migrate struct with Up, Down, Migrate(version), LatestVersion, filterErrNoChange) for app-startup migration, with MigrationsConfig/MigrateOptions Validate(). tools/migrate/cmd/viewgen/ is the SQL view generator binary (make generate-view-sql). Atlas config (atlas.hcl at repo root) points to ent://openmeter/ent/schema as schema source. The repo holds extensive migration data-transform tests (feature_meter_id_test.go, ledger_tax_behavior_test.go, view_parity_test.go, etc.).
- **Depends on:** openmeter/ent (persistence schema + generated code)
- **Key interfaces:** `Migrate` (Up, Down, Migrate, LatestVersion)

### collector (benthos collector module)
- **Location:** `collector`
- **Responsibility:** Separate Go module (own go.mod) running a Redpanda Benthos/Connect service extended with custom OpenMeter plugins. cmd/main.go is a thin launcher that blank-imports plugin packages (benthos/bloblang, benthos/input, benthos/output) then calls service.RunCLI with leaderelection CLI opts. benthos/ holds the custom bloblang, input, output, presets, and services (incl. leaderelection) plugins. Built as a separate Docker image (benthos-collector.Dockerfile).
- **Depends on:** Redpanda Benthos/Connect
- **Key interfaces:** `benthos collector` (RunCLI)

## File Placement

| Component Type | Location | Naming | Example |
|---------------|----------|--------|---------|
| Service interface | `openmeter/<domain>/` | `service.go or <domain>.go (or connector.go) at package root` | `openmeter/customer/service.go, openmeter/billing/service.go, openmeter/entitlement/connector.go` |
| Adapter interface | `openmeter/<domain>/` | `adapter.go (or repository.go) at package root` | `openmeter/customer/adapter.go, openmeter/entitlement/repository.go, openmeter/subject/adapter.go` |
| Concrete service implementation | `openmeter/<domain>/service/` | `service.go inside service/ sub-package` | `openmeter/customer/service/service.go, openmeter/billing/service/service.go` |
| Ent adapter implementation | `openmeter/<domain>/adapter/` | `adapter.go (or <entity>.go) inside adapter/ sub-package` | `openmeter/customer/adapter/customer.go, openmeter/billing/adapter/adapter.go` |
| v1 HTTP handler | `openmeter/<domain>/httpdriver/ or httphandler/` | `handler.go inside httpdriver/ or httphandler/ sub-package` | `openmeter/billing/httpdriver/, openmeter/meter/httphandler/, openmeter/meterevent/httphandler/` |
| v3 HTTP handler | `api/v3/handlers/<resource>/` | `handler.go inside per-resource sub-package, with Goverter convert.gen.go` | `api/v3/handlers/customers/handler.go, api/v3/handlers/meters/handler.go` |
| Wire provider set | `app/common/` | `<domain>.go and openmeter_<binary>.go` | `app/common/billing.go, app/common/openmeter_server.go, app/common/openmeter_billingworker.go` |
| Ent entity schema | `openmeter/ent/schema/` | `<entity>.go using IDMixin+NamespaceMixin+TimeMixin` | `openmeter/ent/schema/customer.go, openmeter/ent/schema/billing.go, openmeter/ent/schema/charges.go` |
| Generated Ent code | `openmeter/ent/db/` | `generated ORM code (DO NOT EDIT)` | `openmeter/ent/db/ (573 generated files)` |
| Generated server stubs / wire / converters | `varies` | `*.gen.go, wire_gen.go (DO NOT EDIT)` | `api/api.gen.go, api/v3/api.gen.go, cmd/server/wire_gen.go, api/v3/handlers/customers/convert.gen.go, api/client/go/client.gen.go` |
| SQL migrations | `tools/migrate/migrations/` | `<timestamp>_<name>.up.sql / .down.sql + atlas.sum` | `tools/migrate/migrations/ (timestamped pairs + atlas.sum hash chain)` |
| TypeSpec API source | `api/spec/packages/aip/ (v3), api/spec/packages/legacy/ (v1)` | `*.tsp with route/tag bindings only in root openmeter.tsp/main.tsp` | `api/spec/packages/aip/src/openmeter.tsp, api/spec/packages/legacy/src/main.tsp` |
| Noop implementations | `openmeter/<domain>/noop/` | `noop.go inside noop/ sub-package` | `openmeter/ledger/noop/noop.go` |
| Test files | `alongside source files, openmeter/<domain>/testutils/, openmeter/testutils/, and e2e/` | `<name>_test.go colocated with source; shared helpers in testutils/` | `openmeter/customer/adapter/customer_test.go, openmeter/customer/testutils/, openmeter/testutils/, e2e/` |
| Binary entrypoint | `cmd/<binary>/` | `main.go + wire.go (//go:build wireinject) + wire_gen.go + version.go` | `cmd/server/main.go, cmd/billing-worker/wire.go, cmd/sink-worker/wire_gen.go` |
| Per-folder agent context | `any package directory` | `CLAUDE.md` | `cmd/CLAUDE.md, cmd/jobs/CLAUDE.md, openmeter/customer/CLAUDE.md, app/common/CLAUDE.md` |

## Naming Conventions

- **Service interfaces**: PascalCase Service / <Noun>Service composing fine-grained sub-interfaces (e.g. `billing.Service`, `customer.CustomerService`, `notification.EventService`, `subscription.CommandService`, `app.MarketplaceService`)
- **Connector/Collector interfaces**: PascalCase <Noun>Connector / <Noun>Collector for pipeline/abstraction boundaries (e.g. `streaming.Connector`, `ingest.Collector`, `feature.FeatureConnector`, `credit.CreditConnector`, `entitlement.Connector`)
- **Adapter interfaces**: PascalCase Adapter / Repo / <Noun>Adapter composing entutils.TxCreator (e.g. `customer.Adapter`, `billing.Adapter`, `entitlement repository.go EntitlementRepo`, `charges.Adapter`)
- **Service input types**: <Verb><Noun>Input suffix, implementing models.Validator (e.g. `CreateCustomerInput`, `ListCustomersInput`, `DeleteCustomerInput`, `UpsertCustomerOverrideInput`, `CreateEntitlementInputs`)
- **Domain events**: <Domain><Action>Event with EventName() prefixed by an EventVersionSubsystem constant (e.g. `MeterCreateEvent`, `InvoiceCreated`, `RecalculateEvent`, `AdvanceStandardInvoiceEvent`)
- **Domain errors**: models.Generic* typed sentinel wrappers (e.g. `GenericNotFoundError`, `GenericValidationError`, `GenericConflictError`, `GenericForbiddenError`, `GenericPreConditionFailedError`)
- **Wire provider functions**: New<Thing> exported from app/common (e.g. `NewBillingRegistry`, `NewCustomerLedgerServiceHook`, `NewLedgerNamespaceHandler`, `NewKafkaIngestNamespaceHandler`)
- **Per-binary Wire sets**: openmeter_<binary>.go composite wire.NewSet (e.g. `app/common/openmeter_server.go`, `app/common/openmeter_billingworker.go`, `app/common/openmeter_balanceworker.go`, `app/common/openmeter_sinkworker.go`, `app/common/openmeter_notification.go`)
- **Registry structs**: <Domain>Registry with nil-safe accessors (e.g. `BillingRegistry (ChargesServiceOrNil())`, `AppRegistry`, `ChargesRegistry`, `SubscriptionServiceWithWorkflow`)
- **Binary entrypoints**: cmd/<binary>/main.go + wire.go + wire_gen.go + version.go (e.g. `cmd/server/main.go`, `cmd/billing-worker/wire.go`, `cmd/sink-worker/wire_gen.go`, `cmd/balance-worker/version.go`)
- **Go package directories**: lowercase concatenated single-word, no underscores or hyphens (e.g. `billing`, `productcatalog`, `balanceworker`, `subscriptionsync`, `httpdriver`, `redisdedupe`)
- **Generated files**: <name>.gen.go suffix or wire_gen.go, with DO NOT EDIT header (e.g. `api.gen.go`, `client.gen.go`, `convert.gen.go`, `filter.gen.go`, `wire_gen.go`)
- **Ent schema mixins**: Mixin() returns IDMixin + NamespaceMixin + TimeMixin first (BalanceSnapshot omits IDMixin) (e.g. `openmeter/ent/schema/customer.go`, `openmeter/ent/schema/billing.go`, `openmeter/ent/schema/grant.go`)
- **Goverter converter files**: convert.go source producing convert.gen.go (e.g. `api/v3/handlers/customers/convert.gen.go`, `api/v3/handlers/billingprofiles/convert.gen.go`, `api/convert.gen.go`)