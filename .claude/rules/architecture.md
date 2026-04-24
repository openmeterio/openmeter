## Components

### server
- **Location:** `cmd/server`
- **Responsibility:** Main HTTP API server. Initializes the application via Wire, runs DB migration, registers namespace handlers (Ledger, Kafka Ingest), provisions the default namespace and billing profile, then mounts both the v1 router and a telemetry server and starts Kafka producers and notification event handler goroutines in a shared oklog/run group.
- **Depends on:** app/common, openmeter/server, openmeter/billing, openmeter/ingest, openmeter/namespace, openmeter/notification, openmeter/customer, openmeter/entitlement, openmeter/subscription, openmeter/productcatalog, openmeter/ledger, openmeter/meter, openmeter/streaming, openmeter/app, api/v3/server

### billing-worker
- **Location:** `cmd/billing-worker`
- **Responsibility:** Background worker that processes billing-related Kafka system events: subscription sync events, invoice creation events, and charge advancement events. Uses Watermill router with Kafka subscriber.
- **Depends on:** app/common, openmeter/billing/worker, openmeter/billing/worker/subscriptionsync, openmeter/billing/charges, openmeter/subscription, openmeter/watermill

### balance-worker
- **Location:** `cmd/balance-worker`
- **Responsibility:** Background worker that handles entitlement balance recalculation events. Subscribes to system and ingest Kafka topics via Watermill, processes batched ingest events to update metered entitlement grant burn-down, and dispatches balance threshold notifications.
- **Depends on:** app/common, openmeter/entitlement/balanceworker, openmeter/entitlement, openmeter/notification, openmeter/watermill, openmeter/registry

### sink-worker
- **Location:** `cmd/sink-worker`
- **Responsibility:** Consumes raw CloudEvents from Kafka (written by the ingest path), deduplicates them, and batch-inserts into ClickHouse. After flushing, publishes ingest notification events to Watermill so downstream workers (balance-worker) can act on new usage data.
- **Depends on:** app/common, openmeter/sink, openmeter/streaming/clickhouse, openmeter/ingest/kafkaingest, openmeter/watermill

### notification-service
- **Location:** `cmd/notification-service`
- **Responsibility:** Standalone notification dispatcher. Subscribes to system events topic via Watermill, receives entitlement balance threshold and invoice events, and delivers webhook payloads via Svix.
- **Depends on:** app/common, openmeter/notification/consumer, openmeter/watermill

### jobs
- **Location:** `cmd/jobs`
- **Responsibility:** Cobra CLI that groups administrative one-off commands: billing advance/collect, entitlement backfill, ledger migrations, LLM cost sync, DB migration, and quickstart provisioning. Each sub-command is in a sub-package under cmd/jobs/.
- **Depends on:** app/common, openmeter/billing, openmeter/entitlement, openmeter/ledger, openmeter/llmcost, tools/migrate

### benthos-collector
- **Location:** `cmd/benthos-collector`
- **Responsibility:** Runs a Redpanda Benthos/Connect service extended with custom OpenMeter bloblang plugins, input plugins, and output plugins. Used to collect and pipeline events from external sources into OpenMeter via the ingest API. Includes leader-election functionality.
- **Depends on:** collector/benthos

### app_common_di
- **Location:** `app/common`
- **Responsibility:** Houses all Google Wire provider sets and constructor functions that wire domain services, adapters, Kafka clients, database clients, telemetry, and configuration into application structs. One file per domain area (billing.go, customer.go, entitlement.go, subscription.go, etc.). Openmeter_*.go files define the Wire sets for each binary. Does not contain business logic.
- **Depends on:** openmeter/*, pkg/*, app/config
- **Key interfaces:** `BillingRegistry` (ChargesServiceOrNil()), `AppRegistry`, `SubscriptionServiceWithWorkflow`

### app_config
- **Location:** `app/config`
- **Responsibility:** Viper-based configuration structs and defaults for all application concerns: Postgres, Kafka, ClickHouse, billing, entitlements, credits, notification, sink, balance-worker, telemetry, portal, apps, and more. Single shared config.Configuration type used by all binaries.

### v1_http_api
- **Location:** `openmeter/server`
- **Responsibility:** Chi-based HTTP server for the v1 API (api/openapi.yaml). The router package receives all domain service interfaces via router.Config and delegates each HTTP endpoint to the corresponding httpdriver or httphandler implementation. Request validation is done via kin-openapi middleware before reaching handlers.
- **Depends on:** api (generated), openmeter/billing/httpdriver, openmeter/customer/httpdriver, openmeter/notification/httpdriver, openmeter/entitlement/driver, openmeter/productcatalog/plan/httpdriver, openmeter/subscription/workflow, openmeter/meter/httphandler, openmeter/ingest/httpdriver, openmeter/app/httpdriver, pkg/framework/transport/httptransport
- **Key interfaces:** `router.Config`

### v3_http_api
- **Location:** `api/v3`
- **Responsibility:** Chi-based HTTP server for the v3 API (api/v3/openapi.yaml). Uses the same handler pattern as v1 but with a separate generated stub (api/v3/api.gen.go) and handler packages under api/v3/handlers/. Serves meters, customers, subscriptions, billing profiles, addons, plans, LLM cost, tax codes, currencies, apps, and charges.
- **Depends on:** api/v3 (generated), api/v3/handlers/*, openmeter/*, pkg/framework/transport/httptransport

### api_spec
- **Location:** `api/spec`
- **Responsibility:** TypeSpec source files defining both the v1 (legacy/) and v3 (aip/) OpenAPI specifications. Compilation produces api/openapi.yaml, api/openapi.cloud.yaml, and api/v3/openapi.yaml. Also produces JavaScript SDK (api/client/javascript/), Go SDK (api/client/go/client.gen.go), and Go server stubs.

### billing_domain
- **Location:** `openmeter/billing`
- **Responsibility:** Core billing domain implementing invoice lifecycle (gathering, standard, advance, collect), billing profiles, customer overrides, line items, split line groups, and sequence numbering. Also owns the line engine registry for pluggable line computation, a rating service for detailed line generation, and a charges sub-domain for usage-based, flat-fee, and credit-purchase charge types. Worker sub-packages handle async invoice advancement, subscription-to-invoice synchronization, and background collection.
- **Depends on:** openmeter/customer, openmeter/app, openmeter/productcatalog/feature, openmeter/meter, openmeter/streaming, openmeter/taxcode, openmeter/watermill, openmeter/ent/db, pkg/framework/entutils
- **Key interfaces:** `billing.Service` (ProfileService, CustomerOverrideService, InvoiceService, GatheringInvoiceService, StandardInvoiceService, InvoiceLineService, SequenceService, LockableService, InvoiceAppService, ConfigService, LineEngineService, SplitLineGroupService), `billing.Adapter` (ProfileAdapter, CustomerOverrideAdapter, InvoiceLineAdapter, InvoiceAdapter, GatheringInvoiceAdapter, StandardInvoiceAdapter, SequenceAdapter, InvoiceAppAdapter, CustomerSynchronizationAdapter, SchemaLevelAdapter, TxCreator), `charges.Service` (GetByID, GetByIDs, Create, AdvanceCharges, ListCustomersToAdvance, ApplyPatches, ListCharges, HandleCreditPurchaseExternalPaymentStateTransition), `rating.Service` (ResolveBillablePeriod, GenerateDetailedLines), `subscriptionsync.Service` (SynchronizeSubscription, SynchronizeSubscriptionAndInvoiceCustomer, HandleCancelledEvent, HandleSubscriptionSyncEvent, HandleInvoiceCreation, GetSyncStates)

### customer_domain
- **Location:** `openmeter/customer`
- **Responsibility:** Manages customer lifecycle (create, read, update, delete, list) and usage attributions. Provides a ServiceHooks mechanism allowing other packages (ledger, entitlement validators) to register pre/post lifecycle callbacks. Also includes a RequestValidator extension point registered by entitlement validators.
- **Depends on:** openmeter/streaming, openmeter/ent/db, openmeter/watermill, pkg/models
- **Key interfaces:** `customer.Service` (ListCustomers, ListCustomerUsageAttributions, CreateCustomer, DeleteCustomer, GetCustomer, GetCustomerByUsageAttribution, UpdateCustomer, RegisterRequestValidator, RegisterHooks), `customer.Adapter`

### entitlement_domain
- **Location:** `openmeter/entitlement`
- **Responsibility:** Manages feature entitlements for customers across three sub-types: metered (credit-based with burn-down tracking via credit engine), boolean (on/off access), and static (value-based). Provides balance history, usage reset, and snapshot capabilities. The metered sub-type integrates with the credit package for grant management and balance calculation.
- **Depends on:** openmeter/productcatalog/feature, openmeter/customer, openmeter/streaming, openmeter/credit, openmeter/ent/db, openmeter/watermill, pkg/framework/lockr
- **Key interfaces:** `entitlement.Service` (CreateEntitlement, OverrideEntitlement, ScheduleEntitlement, SupersedeEntitlement, GetEntitlement, GetEntitlementValue, GetEntitlementsOfCustomer, ListEntitlements, ListEntitlementsWithCustomer, GetEntitlementOfCustomerAt, GetAccess, DeleteEntitlement, GetEntitlementWithCustomer, RegisterHooks), `meteredentitlement.Connector` (GetEntitlementBalance, GetEntitlementBalanceHistory, ResetEntitlementUsage)

### subscription_domain
- **Location:** `openmeter/subscription`
- **Responsibility:** Manages subscription lifecycle (create, cancel, continue, update, delete) against a versioned plan-phase-ratecard model. Uses a patch system (subscription/patch/) and spec types (SubscriptionSpec) for applying incremental changes. Workflow service (subscription/workflow/) orchestrates higher-level operations like CreateFromPlan, EditRunning, and ChangeToPlan by composing the core service. Addon service (subscription/addon/) manages quantity-based subscription add-ons.
- **Depends on:** openmeter/productcatalog, openmeter/customer, openmeter/entitlement, openmeter/ent/db, openmeter/watermill, pkg/framework/lockr
- **Key interfaces:** `subscription.Service` (Get, GetView, List, ExpandViews, Create, Update, Delete, Cancel, Continue, UpdateAnnotations, RegisterHook), `subscriptionworkflow.Service` (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity), `plansubscription.PlanSubscriptionService` (Create, Migrate, Change)

### notification_domain
- **Location:** `openmeter/notification`
- **Responsibility:** Manages notification channels (webhook), rules for triggering notifications, events, and delivery status tracking. Delivers payloads via Svix (webhook service). Includes a Watermill consumer (notification/consumer/) that processes entitlement snapshot and invoice events from the system events Kafka topic. An EventHandler interface with Start/Close lifecycle manages the dispatch and reconciliation loop.
- **Depends on:** openmeter/productcatalog/feature, openmeter/entitlement, openmeter/billing, openmeter/ent/db, openmeter/watermill, external Svix
- **Key interfaces:** `notification.Service` (ListChannels, CreateChannel, DeleteChannel, GetChannel, UpdateChannel, ListRules, CreateRule, DeleteRule, GetRule, UpdateRule, ListEvents, GetEvent, CreateEvent, ResendEvent, ListEventsDeliveryStatus, GetEventDeliveryStatus, UpdateEventDeliveryStatus, ListFeature), `notification.EventHandler` (Dispatch, Reconcile, Start, Close)

### meter_domain
- **Location:** `openmeter/meter`
- **Responsibility:** Defines meters (event aggregation rules) and manages their lifecycle. ManageService extends Service with create/update/delete and a pre-update hook. Meters specify event type, aggregation function, and optional group-by JSON paths.
- **Depends on:** openmeter/ent/db, pkg/filter, pkg/pagination, pkg/sortx
- **Key interfaces:** `meter.Service` (ListMeters, GetMeterByIDOrSlug), `meter.ManageService` (ListMeters, GetMeterByIDOrSlug, CreateMeter, UpdateMeter, DeleteMeter, RegisterPreUpdateMeterHook)

### streaming_domain
- **Location:** `openmeter/streaming`
- **Responsibility:** Abstract connector interface for querying metered usage data from ClickHouse. Implementations live in openmeter/streaming/clickhouse/. Provides event listing, subject listing, group-by value listing, meter query, and batch insert. Also implements namespace.Handler so it participates in namespace lifecycle.
- **Depends on:** openmeter/meter, openmeter/namespace, external ClickHouse
- **Key interfaces:** `streaming.Connector` (CountEvents, ListEvents, ListEventsV2, ListSubjects, ListGroupByValues, QueryMeter, BatchInsert, ValidateJSONPath, CreateNamespace, DeleteNamespace)

### ingest_domain
- **Location:** `openmeter/ingest`
- **Responsibility:** Defines the Collector interface (Ingest/Close) for receiving CloudEvents and forwarding to Kafka. The kafkaingest sub-package implements the collector using confluent-kafka-go with a JSON serializer. An ingestadapter wraps the collector with OTel telemetry. Optional deduplication wraps the collector using a Redis or in-memory deduplicator.
- **Depends on:** openmeter/dedupe, openmeter/ingest/kafkaingest, pkg/kafka, external Kafka
- **Key interfaces:** `ingest.Collector` (Ingest, Close)

### productcatalog_domain
- **Location:** `openmeter/productcatalog`
- **Responsibility:** Defines the product catalog entities: features (meter-backed usage features with optional unit costs), plans (versioned multi-phase rate card collections), plan addons, addons, and rate cards (usage-based, flat-fee pricing models). Each entity has a Service interface, Ent adapter, and httpdriver package. Plan subscription service (productcatalog/subscription/) orchestrates plan-aware subscription lifecycle.
- **Depends on:** openmeter/meter, openmeter/ent/db, openmeter/watermill
- **Key interfaces:** `feature.FeatureConnector` (CreateFeature, UpdateFeature, ArchiveFeature, ListFeatures, GetFeature, ResolveFeatureMeters), `plan.Service` (ListPlans, CreatePlan, DeletePlan, GetPlan, UpdatePlan, PublishPlan, ArchivePlan, NextPlan), `addon.Service`

### ledger_domain
- **Location:** `openmeter/ledger`
- **Responsibility:** Double-entry ledger for tracking customer financial balances (FBO, receivable, accrued) and business accounts (wash, earnings, brokerage). Provides routing rules, sub-account primitives, and customer balance queries. Used for credit management in billing. When credits are disabled, app/common wires noop implementations.
- **Depends on:** openmeter/customer, openmeter/ent/db, pkg/currencyx
- **Key interfaces:** `ledger.Ledger`

### credit_domain
- **Location:** `openmeter/credit`
- **Responsibility:** Manages credit grants and balance snapshots for metered entitlements. CreditConnector combines GrantConnector and BalanceConnector. The engine sub-package implements grant burn-down calculation. Uses streaming.Connector for usage queries and grant.Repo (Ent-backed) for persistence.
- **Depends on:** openmeter/streaming, openmeter/credit/grant, openmeter/credit/balance, openmeter/credit/engine, openmeter/ent/db, openmeter/watermill
- **Key interfaces:** `credit.CreditConnector`

### app_domain
- **Location:** `openmeter/app`
- **Responsibility:** Manages installed marketplace apps (Stripe, custom invoicing, sandbox). The Service interface handles app installation, OAuth2 flows, customer data management, and marketplace listings. Sub-packages for stripe (openmeter/app/stripe/) and custominvoicing (openmeter/app/custominvoicing/) extend the base app with provider-specific logic.
- **Depends on:** openmeter/customer, openmeter/secret, openmeter/billing, openmeter/ent/db, openmeter/watermill
- **Key interfaces:** `app.Service` (RegisterMarketplaceListing, GetMarketplaceListing, ListMarketplaceListings, InstallMarketplaceListingWithAPIKey, InstallMarketplaceListing, GetMarketplaceListingOauth2InstallURL, AuthorizeMarketplaceListingOauth2Install, CreateApp, GetApp, UpdateAppStatus, UpdateApp, ListApps, UninstallApp, ListCustomerData, EnsureCustomer, DeleteCustomer), `app.App` (GetAppBase, GetID, GetType, GetName, GetStatus, GetListing, UpdateAppConfig, ValidateCapabilities, GetCustomerData, UpsertCustomerData, DeleteCustomerData)

### namespace_domain
- **Location:** `openmeter/namespace`
- **Responsibility:** Provides multi-tenancy via namespaces. The Manager holds a list of handlers (ClickHouse streaming, Kafka ingest, Ledger) and calls CreateNamespace/DeleteNamespace on each when namespaces are provisioned or deprovisioned. Used during server startup and namespace API endpoints.
- **Key interfaces:** `namespace.Manager` (CreateDefaultNamespace, RegisterHandler, GetDefaultNamespace), `namespace.Handler` (CreateNamespace, DeleteNamespace)

### sink_worker_domain
- **Location:** `openmeter/sink`
- **Responsibility:** Kafka consumer that reads raw ingest events, validates them against meter definitions, buffers them, deduplicates via Redis or in-memory, and batch-inserts into ClickHouse via the streaming.Connector. After a successful flush, fires ingest notification events via Watermill so the balance-worker can recalculate entitlements.
- **Depends on:** openmeter/streaming, openmeter/ingest/kafkaingest, openmeter/meter, openmeter/dedupe, openmeter/watermill

### watermill_event_bus
- **Location:** `openmeter/watermill`
- **Responsibility:** Wrapper around Watermill for Kafka-backed publish-subscribe. Provides a Publisher that routes messages to three topics: ingest events, system events, and balance worker events. Router, group handler, and marshaler sub-packages provide Watermill router construction with OTel tracing. Drivers include Kafka (confluent) and noop.
- **Depends on:** external Kafka via confluent-kafka-go, external Watermill
- **Key interfaces:** `eventbus.Publisher`

### portal_domain
- **Location:** `openmeter/portal`
- **Responsibility:** Issues and validates short-lived portal tokens that restrict end-customer access to specific meter slugs and subjects. Backed by JWT with optional expiry and subject scoping.
- **Depends on:** openmeter/ent/db
- **Key interfaces:** `portal.Service` (CreateToken, Validate, ListTokens, InvalidateToken)

### llmcost_domain
- **Location:** `openmeter/llmcost`
- **Responsibility:** Manages LLM model cost prices: global synced prices and per-namespace overrides. Provides price resolution logic (namespace overrides take precedence) and a sync sub-package for synchronizing prices from external sources.
- **Depends on:** openmeter/ent/db, pkg/filter, pkg/pagination
- **Key interfaces:** `llmcost.Service` (ListPrices, GetPrice, ResolvePrice, CreateOverride, DeleteOverride, ListOverrides)

### cost_domain
- **Location:** `openmeter/cost`
- **Responsibility:** Computes feature cost by querying meter usage. The Adapter interface has a single QueryFeatureCost method backed by ClickHouse. Used for the feature cost API endpoint.
- **Depends on:** openmeter/streaming, openmeter/productcatalog/feature
- **Key interfaces:** `cost.Adapter` (QueryFeatureCost)

### progressmanager_domain
- **Location:** `openmeter/progressmanager`
- **Responsibility:** Tracks progress of long-running operations (e.g., ClickHouse query export). Provides GetProgress and UpsertProgress backed by an Ent adapter.
- **Depends on:** openmeter/ent/db
- **Key interfaces:** `progressmanager.Service` (GetProgress, UpsertProgress)

### subject_domain
- **Location:** `openmeter/subject`
- **Responsibility:** Manages subjects (usage measurement subjects, analogous to users/devices). Provides CRUD and list operations backed by an Ent adapter. Exposes ServiceHooks for lifecycle events.
- **Depends on:** openmeter/ent/db, openmeter/watermill
- **Key interfaces:** `subject.Service` (Create, Update, GetByKey, GetById, GetByIdOrKey, List, Delete, RegisterHooks)

### meterevent_domain
- **Location:** `openmeter/meterevent`
- **Responsibility:** Provides event listing API backed by streaming.Connector. Enforces time window limits (32 days lookback, 100 result limit) and supports v1 and v2 listing with cursor-based pagination.
- **Depends on:** openmeter/streaming
- **Key interfaces:** `meterevent.Service` (ListEvents, ListEventsV2)

### currencies_domain
- **Location:** `openmeter/currencies`
- **Responsibility:** Manages custom currencies and cost bases for billing. Backed by Ent with transaction support.
- **Depends on:** openmeter/ent/db, pkg/framework/entutils
- **Key interfaces:** `currencies.Adapter` (ListCustomCurrencies, CreateCurrency, CreateCostBasis, ListCostBases, Tx)

### taxcode_domain
- **Location:** `openmeter/taxcode`
- **Responsibility:** Manages tax codes used during invoice line processing. Provides a Service interface backed by a repository.
- **Depends on:** openmeter/ent/db

### secret_domain
- **Location:** `openmeter/secret`
- **Responsibility:** Stores and retrieves encrypted secrets (e.g., Stripe API keys) for installed apps. Backed by an Ent adapter.
- **Depends on:** openmeter/ent/db

### ent_schema
- **Location:** `openmeter/ent/schema`
- **Responsibility:** Ent schema definitions (source of truth for database schema). One file per entity: billing invoices and lines, customers, entitlements, grants, features, subscriptions, plans, addons, notifications, LLM cost prices, ledger accounts/transactions, meters, subjects, secrets. Used by Atlas to generate SQL migrations.

### migrations
- **Location:** `tools/migrate`
- **Responsibility:** Holds Atlas-generated SQL migration files (golang-migrate format) in tools/migrate/migrations/. A Go migrate package (tools/migrate/migrate.go) wraps golang-migrate for use in app startup. The viewgen sub-command generates ClickHouse view SQL. Atlas config in atlas.hcl points to ent://openmeter/ent/schema and file://tools/migrate/migrations.
- **Depends on:** openmeter/ent/schema

### pkg_framework
- **Location:** `pkg/framework`
- **Responsibility:** Shared low-level Go patterns used across all domain packages. httptransport provides a generic Handler[Request,Response] struct that decodes requests, invokes an operation.Operation, and encodes responses. entutils provides Ent transaction helpers (TxCreator, TransactingRepo), pagination helpers, cursor helpers, and ULID/PGULID type utilities. lockr provides a distributed lock implementation. commonhttp provides error encoding and media-type negotiation. pgdriver provides pgx pool wiring.
- **Key interfaces:** `httptransport.Handler[Request, Response]` (ServeHTTP, Chain), `entutils.TxCreator` (Tx)

### pkg_models
- **Location:** `pkg/models`
- **Responsibility:** Shared domain model primitives: NamespacedID, NamespacedKey, Metadata, Annotations, Validator interface, ValidationIssue, Percentage, Cadence, ServiceHook/ServiceHooks generic lifecycle hook registry, pagination.Result, pagination.Page, and sortx. Used throughout all domain packages.
- **Key interfaces:** `models.ServiceHooks[T]` (RegisterHooks), `models.Validator` (Validate)

## File Placement

| Component Type | Location | Naming | Example |
|---------------|----------|--------|---------|
| domain service interface | `openmeter/<domain>/` | `service.go or <domain>.go at package root` | `openmeter/billing/service.go, openmeter/customer/service.go, openmeter/meter/service.go` |
| domain adapter interface | `openmeter/<domain>/` | `adapter.go at package root` | `openmeter/billing/adapter.go, openmeter/customer/adapter.go, openmeter/charges/adapter.go` |
| Ent/PostgreSQL adapter implementation | `openmeter/<domain>/adapter/` | `adapter/ sub-package with adapter.go` | `openmeter/billing/adapter/adapter.go, openmeter/customer/adapter/adapter.go` |
| service implementation | `openmeter/<domain>/service/` | `service/ sub-package with service.go` | `openmeter/billing/service/service.go, openmeter/customer/service/service.go` |
| HTTP handler | `openmeter/<domain>/httpdriver/ or openmeter/<domain>/httphandler/` | `httpdriver/ or httphandler/ sub-package with handler.go` | `openmeter/billing/httpdriver/handler.go, openmeter/customer/httpdriver/, openmeter/meter/httphandler/` |
| v3 API handler | `api/v3/handlers/<resource>/` | `handlers/<resource>/ sub-package` | `api/v3/handlers/meters/, api/v3/handlers/customers/, api/v3/handlers/billing profiles/` |
| Wire DI provider set | `app/common/` | `<domain>.go in app/common/` | `app/common/billing.go, app/common/customer.go, app/common/subscription.go` |
| binary-specific Wire set | `app/common/` | `openmeter_<binary>.go in app/common/` | `app/common/openmeter_server.go, app/common/openmeter_billingworker.go, app/common/openmeter_sinkworker.go` |
| generated code | `openmeter/ent/db/, */wire_gen.go, */*.gen.go, api/api.gen.go, api/v3/api.gen.go` | `*.gen.go or wire_gen.go or ent/db/` | `openmeter/ent/db/, cmd/server/wire_gen.go, openmeter/billing/derived.gen.go, api/api.gen.go` |
| Ent entity schema | `openmeter/ent/schema/` | `<entity>.go` | `openmeter/ent/schema/customer.go, openmeter/ent/schema/billing.go` |
| Atlas SQL migrations | `tools/migrate/migrations/` | `<timestamp>_<name>.up.sql / <timestamp>_<name>.down.sql` | `tools/migrate/migrations/20240826120919_init.up.sql` |
| test utilities | `openmeter/<domain>/testutils/` | `testutils/ sub-package` | `openmeter/billing/testutils, openmeter/customer/testutils, openmeter/subscription/testutils` |

## Naming Conventions

- **service interface**: PascalCase interface named Service or <Noun>Service in domain package (e.g. `billing.Service`, `customer.Service`, `meter.ManageService`, `notification.ChannelService`)
- **adapter interface**: PascalCase interface named Adapter or <Noun>Adapter (e.g. `billing.Adapter`, `customer.Adapter`, `charges.Adapter`, `currencies.Adapter`)
- **connector interface (legacy pattern)**: PascalCase interface named Connector (e.g. `streaming.Connector`, `credit.CreditConnector`, `meteredentitlement.Connector`)
- **HTTP handler interface**: Handler interface in httpdriver/ or httphandler/ package (e.g. `billing/httpdriver.Handler`, `customer/httpdriver.Handler`)
- **service input types**: <Verb><Noun>Input suffix for service method parameters (e.g. `CreateCustomerInput`, `ListInvoicesInput`, `GetProfileInput`, `DeleteMeterInput`)
- **Wire provider sets**: PascalCase var named after domain in app/common (e.g. `var Billing = wire.NewSet(...)`, `var Customer = wire.NewSet(...)`, `var Subscription = wire.NewSet(...)`)
- **typed service registry structs**: <Domain>Registry struct in app/common grouping related services (e.g. `BillingRegistry{Billing: billing.Service, Charges: *ChargesRegistry}`, `AppRegistry{Service, Stripe, CustomInvoicing}`)
- **Kafka topic constants**: SystemEventsTopic, IngestEventsTopic, BalanceWorkerEventsTopic in worker Options structs (e.g. `WorkerOptions.SystemEventsTopic`, `WorkerOptions.BalanceWorkerEventsTopic`)
- **generated files**: *.gen.go or wire_gen.go (e.g. `wire_gen.go`, `derived.gen.go`, `filter.gen.go`, `convert.gen.go`)