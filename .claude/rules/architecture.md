## Components

### Server entrypoint & DI bootstrap
- **Location:** `cmd/server`
- **Responsibility:** main() loads config via Viper (with config.DecodeHook), validates it, calls Wire-generated initializeApplication(ctx, conf) to build the full Application graph, then runs app.Migrate(ctx) BEFORE provisioning the default namespace. It registers namespace handlers (app.LedgerNamespaceHandler, app.KafkaIngestNamespaceHandler) prior to initNamespace, constructs a debug.NewDebugConnector wrapping the StreamingConnector, and starts the HTTP server plus telemetry server through oklog/run. Panics are funneled through log.PanicLogger(log.WithExit).
- **Depends on:** app/common, app/config, openmeter/server, openmeter/server/router, openmeter/namespace, openmeter/ingest/kafkaingest, openmeter/debug, pkg/errorsx, pkg/log
- **Key interfaces:** `main` (main)

### Worker & job entrypoints
- **Location:** `cmd`
- **Responsibility:** Five additional binaries beyond the API server, each with its own main.go repeating the same Viper config-load/validate boilerplate (cmd/server/main.go pattern) and then building a worker-specific Wire application: billing-worker (invoice advancement/collection + subscription→billing reconciliation), balance-worker (entitlement balance snapshotting from usage events), sink-worker (Kafka→ClickHouse usage event sink), notification-service (notification event delivery), and jobs (one-off operational jobs). All share app/config and pkg/log.PanicLogger.
- **Depends on:** app/common, app/config, pkg/log
- **Key interfaces:** `billing-worker main` (main), `balance-worker main` (main), `sink-worker main` (main), `notification-service main` (main), `jobs main` (main)

### Application wiring (app/common)
- **Location:** `app/common`
- **Responsibility:** The dependency-injection layer. Each file is a Google Wire provider set (wire.NewSet) for one subsystem: billing.go declares BillingRegistry/ChargesRegistry bundling billing.Service plus flatfee/usagebased/creditpurchase/recognizer charge services; ledger.go wires concrete-or-noop ledger account/resolver services gated on credits.enabled; database.go, clickhouse.go, kafka.go, streaming.go provide infra clients; openmeter_server.go / openmeter_billingworker.go / openmeter_sinkworker.go / openmeter_balanceworker.go / openmeter_notification.go assemble per-binary Application structs. Constructor functions take explicit *slog.Logger and config inputs (never slog.Default()).
- **Depends on:** app/config, openmeter/billing, openmeter/billing/charges, openmeter/billing/rating, openmeter/customer, openmeter/ledger, openmeter/meter, openmeter/productcatalog/feature, openmeter/streaming, openmeter/taxcode, openmeter/watermill/eventbus, openmeter/ent/db, pkg/framework/lockr, pkg/featuregate
- **Key interfaces:** `BillingRegistry` (ChargesServiceOrNil), `Billing (wire.NewSet)` (BillingAdapter, NewBillingRatingService, NewLedgerBreakageService, NewBillingRegistry, NewBillingCustomerOverrideService)

### Billing domain
- **Location:** `openmeter/billing`
- **Responsibility:** Largest domain (41 files in root, 95 in-edges). Root package declares the billing.Service and billing.Adapter interfaces plus invoice/line/profile/customer-override/discount/tax models and the StandardInvoiceHooks. service/ implements billing.Service (struct Service holds adapter, customerService, appService, taxCodeService, ratingService, featureService, meterService, streamingConnector, eventbus.Publisher, an engineRegistry of lineEngines, and an invoicecalc.Calculator) orchestrating the invoice state machine, gathering→standard invoice conversion, line splitting, sequence numbering, and quantity snapshots. adapter/ implements billing.Adapter over Ent with HijackTx/WithTx transaction hijacking. Sub-areas: charges/ (usage-based & flatfee & credit-purchase charge lifecycle state machines), rating/ (pricing engine), worker/ (auto-advance, collect, subscriptionsync reconciliation), lineengine/, validators/, httpdriver/ (legacy v1 HTTP). Uses goderive (derived.gen.go) and goverter.
- **Depends on:** openmeter/app, openmeter/customer, openmeter/meter, openmeter/productcatalog/feature, openmeter/streaming, openmeter/taxcode, openmeter/billing/rating, openmeter/billing/lineengine, openmeter/billing/service/invoicecalc, openmeter/watermill/eventbus, openmeter/ent/db, pkg/framework/transaction, pkg/framework/entutils, pkg/models
- **Key interfaces:** `billing.Service` ((invoice lifecycle, profile, customer-override, gathering & standard invoice, line, sequence, quantity-snapshot operations declared across openmeter/billing/*.go)), `billing.Adapter` (Tx, WithTx, Self), `Config.Validate` (Validate)

### Charges sub-system
- **Location:** `openmeter/billing/charges`
- **Responsibility:** Usage-based billing charge engine layered under billing. meta/ holds charge-meta queries (39 in-edges); service/ drives charge creation/advancement and the realization runs; usagebased/, flatfee/, creditpurchase/ implement per-charge-type lifecycle state machines; invoiceupdater/, lineage/ link charges back to invoice lines; models/ subpackages (chargemeta, creditrealization, invoicedusage, ledgertransaction, payment) carry value types. Adapter helpers are kept transaction-aware via entutils.TransactingRepo even when handed a raw *entdb.Client. Lifecycle tests drive charges.Service.Create/AdvanceCharges/ApplyPatches.
- **Depends on:** openmeter/billing, openmeter/billing/charges/meta, openmeter/ledger, openmeter/credit, openmeter/ent/db, pkg/framework/entutils
- **Key interfaces:** `charges.Service` (Create, AdvanceCharges, ApplyPatches)

### Customer domain
- **Location:** `openmeter/customer`
- **Responsibility:** High-fan-in domain (103 in-edges). Root declares customer.Service plus customer model, errors, events, and a requestvalidator. service/ implements the service (with service/hooks for lifecycle hooks); adapter/ is the Ent persistence layer; app/ holds app-integration types; httpdriver/ is the legacy v1 HTTP layer; testutils/ provides shared test fixtures. Consumed by billing, subscription, ledger and the v3 customers handlers.
- **Depends on:** openmeter/ent/db, pkg/framework/entutils, pkg/framework/transaction, pkg/models, pkg/clock
- **Key interfaces:** `customer.Service` ((customer CRUD + lifecycle declared in openmeter/customer/customer.go))

### Product catalog domain
- **Location:** `openmeter/productcatalog`
- **Responsibility:** Highest non-pkg fan-in (104 in-edges). Holds plan, addon, planaddon, ratecard, price, discount, tax, feature, and pro-rating value types and rules at the root, with adapter/ (Ent), feature/ (FeatureConnector + FeatureRepo, itself 67 in-edges), featureresolver/, plan/ (service+adapter), addon/, planaddon/, subscription/ (plan→subscription bridge), driver/ + http/ (HTTP), and testutils/. Defines the catalog primitives that subscription, billing, and entitlement build on.
- **Depends on:** openmeter/ent/db, openmeter/meter, pkg/models, pkg/currencyx, pkg/datetime
- **Key interfaces:** `feature.FeatureConnector` ((feature CRUD/query declared in openmeter/productcatalog/feature)), `feature.FeatureRepo` ((feature persistence contract))

### Subscription domain
- **Location:** `openmeter/subscription`
- **Responsibility:** 31-file domain (37 in-edges, 25 out-edges). Root holds the subscription spec model (subscriptionspec.go), views (subscriptionview.go), patch system (patch.go + patch/), apply/sync logic (apply.go), phase/item/timing models, uniqueness rules, billing bridge (billing.go), and entitlement bridge (entitlement.go, entitlement/). repo/ persists; service/ + workflow/ orchestrate creation/edit/cancel/plan-change; addon/ (with diff/, repo/, service/) is the subscription-addon sub-system; hooks/ and validators/ enforce invariants. Bridges product catalog plans to live customer subscriptions and into billing + entitlement.
- **Depends on:** openmeter/productcatalog, openmeter/customer, openmeter/entitlement, openmeter/billing, openmeter/ent/db, pkg/datetime, pkg/models
- **Key interfaces:** `subscription.Service` ((create/list/get/cancel/change declared in openmeter/subscription/service.go)), `Subscription patch system` ((patch application declared in subscription/patch.go))

### Entitlement domain
- **Location:** `openmeter/entitlement`
- **Responsibility:** Access-control domain (40 in-edges, 19 out-edges). Root declares entitlement.Service, EntitlementRepo, the access model (access.go), entitlement types (metered/boolean/static), grant linkage (entitlement_grant.go), and usage-period logic (usageperiod.go). metered/ holds the metered-entitlement connector tying entitlements to credit grants + usage; balanceworker/ computes balance snapshots from usage events (driven by cmd/balance-worker); driver/ + driver/v2 expose legacy v1 + v2 HTTP; hooks/subscription wires entitlement provisioning to subscription lifecycle; snapshot/, static/, boolean/, validators/ round it out.
- **Depends on:** openmeter/credit, openmeter/credit/grant, openmeter/productcatalog/feature, openmeter/meter, openmeter/streaming, openmeter/ent/db, pkg/models, pkg/timeutil
- **Key interfaces:** `entitlement.Service` ((entitlement CRUD + access checks declared in openmeter/entitlement/connector.go)), `meteredentitlement.Connector` ((metered balance/reset operations))

### Credit & grant domain
- **Location:** `openmeter/credit`
- **Responsibility:** Credit-grant accounting feeding entitlements. Root declares BalanceConnector, GrantConnector, balance/ (balance computation), engine/ (grant-burn-down engine), grant/ (grant model + Repo + OwnerConnector), hook/ (lifecycle hooks). The whole credit stack is feature-gated by credits.enabled at the wiring layer. Used by the metered-entitlement connector and the registry.
- **Depends on:** openmeter/credit/grant, openmeter/credit/balance, openmeter/credit/engine, openmeter/ent/db, pkg/models, pkg/timeutil
- **Key interfaces:** `credit.BalanceConnector` ((credit balance computation)), `credit.GrantConnector` ((grant CRUD)), `grant.OwnerConnector` ((grant owner resolution))

### Ledger domain
- **Location:** `openmeter/ledger`
- **Responsibility:** Double-entry-style ledger for customer credit accounts (35 in-edges, 17 out-edges), feature-gated by credits.enabled. Root holds account/transaction/balance/routing primitives, impact analysis (impact.go), routing rules + validator (routing.go, routing_validator.go), and a noop/ implementation used when credits are off. account/ has its own service/+adapter/ (account_business.go, account_customer.go, subaccount.go); transactions/, historical/ (+adapter), resolvers/ (+adapter), routingrules/, recognizer/, breakage/, chargeadapter/, customerbalance/, collector/ are sub-areas. When credits are disabled app/common wires noop services; real backfill must construct concrete account+resolver adapters directly.
- **Depends on:** openmeter/customer, openmeter/ent/db, pkg/framework/entutils, pkg/models, pkg/datetime
- **Key interfaces:** `ledger account service` ((account create/resolve/balance declared in openmeter/ledger/account)), `ledger routing` ((routing rule evaluation in routing.go / routing_validator.go))

### Meter domain
- **Location:** `openmeter/meter`
- **Responsibility:** Meter definition domain (70 in-edges). Root declares meter.Service, the Meter model, MeterQueryRow, and event parsing (parse.go). service/ implements the service; adapter/ persists via Ent; mockadapter/ is a test double; httphandler/ exposes v1 meter HTTP. Meters define how raw usage events are aggregated; consumed by streaming, billing, and entitlement.
- **Depends on:** openmeter/namespace, openmeter/ent/db, pkg/models, pkg/slicesx
- **Key interfaces:** `meter.Service` ((meter CRUD/query declared in openmeter/meter/service.go)), `Meter.parse` (parse)

### Streaming / usage query (ClickHouse)
- **Location:** `openmeter/streaming`
- **Responsibility:** Defines the streaming.Connector interface (extends namespace.Handler) for reading metered usage out of ClickHouse: CountEvents, ListEvents, ListEventsV2, ListSubjects, ListGroupByValues, QueryMeter, BatchInsert, ValidateJSONPath. clickhouse/ holds the concrete ClickHouse implementation; retry/ adds retry behavior; usageattribution.go maps events to customers; testutils/ provides a MockStreamingConnector with SetSimpleEvents and explicit StoredAt for late-arriving-usage tests. The RawEvent struct carries CloudEvents-shaped fields with both `ch` (ClickHouse) and `json` tags.
- **Depends on:** openmeter/meter, openmeter/namespace, pkg/models
- **Key interfaces:** `streaming.Connector` (CountEvents, ListEvents, ListEventsV2, ListSubjects, ListGroupByValues, QueryMeter, BatchInsert, ValidateJSONPath)

### Ingest & sink pipeline
- **Location:** `openmeter/ingest`
- **Responsibility:** Usage event intake. ingest/ defines the ingest service, in-memory and dedupe collectors (dedupe.go, inmemory.go), kafkaingest/ publishes events to Kafka, ingestadapter/ adapts, httpdriver/ exposes the v1 ingest HTTP endpoint. openmeter/sink (sink-worker side) consumes from Kafka: buffer.go batches events, partition.go handles partition assignment, storage.go + meters.go write to ClickHouse via the streaming connector, flushhandler/ flushes buffers. Together they form the event-time metering ingest→storage path.
- **Depends on:** openmeter/streaming, openmeter/meter, openmeter/namespace, pkg/kafka
- **Key interfaces:** `ingest service` ((event ingest + dedupe in openmeter/ingest/ingest.go, dedupe.go)), `sink` ((buffer, partition, storage flush in openmeter/sink))

### Notification domain
- **Location:** `openmeter/notification`
- **Responsibility:** Notification event pipeline. Root declares notification.Service + notification.Repository, channel/rule/event/eventpayload/deliverystatus models, and entitlement-specific event types (entitlements.go, invoice.go). service/ implements the service (struct Service holds feature connector, adapter repository, webhook.Handler, logger); consumer/ is the Kafka consumer side (driven by cmd/notification-service); eventhandler/ processes events; webhook/ integrates with Svix for webhook delivery; httpdriver/ exposes v1 HTTP; internal/ holds private helpers.
- **Depends on:** openmeter/notification/webhook, openmeter/productcatalog/feature, openmeter/ent/db, openmeter/watermill/eventbus, pkg/models
- **Key interfaces:** `notification.Service` ((rule/channel/event CRUD + delivery declared in openmeter/notification/service.go)), `notification.Repository` ((notification persistence)), `webhook.Handler` ((Svix webhook delivery))

### App / marketplace integrations
- **Location:** `openmeter/app`
- **Responsibility:** Third-party app/marketplace framework (53 in-edges). Root declares app.Service, the app/marketplace registry (registry.go, marketplace.go), webhook handling, and an appbase. stripe/ implements the Stripe billing integration; custominvoicing/ implements a custom-invoicing app; sandbox/ is a test app; httpdriver/ + service/ + adapter/ provide HTTP/service/persistence. Apps plug into billing (invoice delivery, payment).
- **Depends on:** openmeter/customer, openmeter/secret, openmeter/ent/db, pkg/models
- **Key interfaces:** `app.Service` ((app install/list/marketplace declared in openmeter/app/service.go))

### Ent schema (DB source of truth)
- **Location:** `openmeter/ent/schema`
- **Responsibility:** Hand-written Ent entity definitions that are the source of truth for the PostgreSQL schema. ~35 schema files spanning customer, billing (billing.go), charges (charges*.go), subscription (+addon, +billingsync), entitlement, grant, feature, ledger_* (account, customer_account, entry, transaction, transaction_group, breakage_record), llmcostprice, notification, plan/addon, ratecard, meter, taxcode, app/app_stripe, balance_snapshot, usage_reset. Schemas use shared Mixins from pkg/framework/entutils (ResourceMixin, CustomerAddressMixin, AnnotationsMixin). Generated into openmeter/ent/db (407 files, DO NOT EDIT) via `make generate`; migrations generated from schema diffs via Atlas.
- **Depends on:** pkg/framework/entutils, pkg/clock, pkg/currencyx
- **Key interfaces:** `Customer (ent.Schema)` (Mixin, Fields, Edges, Indexes)

### Generated Ent client
- **Location:** `openmeter/ent/db`
- **Responsibility:** Ent-generated ORM client (407 files, 60 in-edges, 108 out-edges). Single shared *entdb.Client used by every domain adapter for PostgreSQL access; supports transaction hijacking (HijackTx) consumed by adapters' Tx/WithTx. predicate/ subpackage (94 in-edges) holds query predicates. All files carry `// Code generated ... DO NOT EDIT` headers; regenerated from openmeter/ent/schema via `make generate`. Its breadth makes it the dominant node in the large generated-code import cycle (cycle 0).
- **Depends on:** openmeter/ent/schema
- **Key interfaces:** `entdb.Client` (HijackTx, NewTxClientFromRawConfig, Client)

### Registry & builder
- **Location:** `openmeter/registry`
- **Responsibility:** Aggregation structs that bundle related connectors for downstream consumers. registry.Entitlement groups Feature/FeatureRepo connectors, EntitlementOwner (grant.OwnerConnector), CreditBalance/Grant connectors, GrantRepo, MeteredEntitlement connector, Entitlement service, and EntitlementRepo into one struct. builder/ assembles these registries. This is the in-domain counterpart to app/common's Wire registries.
- **Depends on:** openmeter/credit, openmeter/credit/grant, openmeter/entitlement, openmeter/entitlement/metered, openmeter/productcatalog/feature
- **Key interfaces:** `registry.Entitlement`

### Legacy v1 HTTP router
- **Location:** `openmeter/server/router`
- **Responsibility:** Assembles the legacy v1 HTTP API. router.go validates requests against the embedded OpenAPI 3 spec (kin-openapi openapi3/openapi3filter) and wires per-domain httpdriver/httphandler packages (customer, billing, credit, entitlement v1+v2, meter, meterevent, ingest, ledger/customerbalance, llmcost, notification, portal, productcatalog/addon/plan/feature, app/stripe/custominvoicing, subject, subscription, debug, info) into Chi routes implementing the generated api.ServerInterface from api/api.gen.go. The sibling openmeter/server holds CORS + server lifecycle.
- **Depends on:** api, api/v3/handlers/currencies, app/config, openmeter/customer, openmeter/billing, openmeter/entitlement, openmeter/meter, openmeter/notification, openmeter/productcatalog, openmeter/ledger, (all domain httpdriver packages)
- **Key interfaces:** `Router` ((implements api.ServerInterface generated from legacy TypeSpec))

### v3 API layer (AIP-style)
- **Location:** `api/v3`
- **Responsibility:** Newer Google-AIP-style HTTP API. api.gen.go is the oapi-codegen-generated ServerInterface; server/routes.go implements every operation as a thin method delegating to a per-resource handler (e.g. s.metersHandler.CreateMeter().With(meterId).ServeHTTP). handlers/ holds one package per resource (meters, events, customers + nested billing/charges/credits/entitlementaccess, subscriptions, apps, billingprofiles, taxcodes, currencies, features, featurecost, llmcost, plans, addons) each defining a Handler interface + struct with New(resolveNamespace, service, options...) returning httptransport handlers. filters/ implements AIP query-param filtering; apierrors/, render/, request/, response/, oasmiddleware/ are shared HTTP plumbing. Not all generated operations are implemented: credits operations are gated on Credits.Enabled and delegate to api.Unimplemented otherwise; CreateCreditAdjustment and QueryGovernanceAccess are always unimplemented.
- **Depends on:** api/v3 (generated), openmeter/customer, openmeter/meter, openmeter/subscription, openmeter/billing, openmeter/billing/charges, openmeter/credit, openmeter/ledger, pkg/framework/transport/httptransport, pkg/framework/commonhttp
- **Key interfaces:** `v3 Server (meters)` (CreateMeter, GetMeter, ListMeters, UpdateMeter, DeleteMeter, QueryMeter (JSON + text/csv content negotiation)), `v3 Server (customers)` (CreateCustomer, GetCustomer, ListCustomers, UpsertCustomer, DeleteCustomer, ListCustomerEntitlementAccess), `v3 Server (subscriptions)` (CreateSubscription, ListSubscriptions, GetSubscription, CancelSubscription, UnscheduleCancelation, ChangeSubscription), `v3 Server (plans)` (ListPlans, CreatePlan, GetPlan, UpdatePlan, DeletePlan, ArchivePlan, PublishPlan), `v3 Server (credits, gated)` (GetCustomerCreditBalance, ListCreditGrants, CreateCreditGrant, GetCreditGrant, UpdateCreditGrantExternalSettlement, ListCreditTransactions), `customers.Handler` (ListCustomers, CreateCustomer, DeleteCustomer, GetCustomer, UpsertCustomer)

### Generated API contract & v1 server interface
- **Location:** `api`
- **Responsibility:** Top-level api package (43 in-edges): api.gen.go is the oapi-codegen-generated legacy v1 ServerInterface + types; convert.gen.go/convert.go are goverter type converters; openapi.yaml + openapi.cloud.yaml are the generated OpenAPI specs (from TypeSpec); types/ holds shared API types. Consumed by both the legacy router and v3. This package is the Go-side boundary of the TypeSpec→OpenAPI→Go generation chain.
- **Depends on:** pkg/models
- **Key interfaces:** `api.ServerInterface` ((all v1 operations generated from legacy TypeSpec))

### TypeSpec API specification
- **Location:** `api/spec`
- **Responsibility:** Node/pnpm workspace authoring the API contract in TypeSpec, the source of truth for OpenAPI. Two packages under packages/: legacy/ (src/main.tsp + auth/errors/debug/events/meters/filter/portal/subjects/rest/query/types.tsp and an app/ subtree) drives the v1 API; aip/ (src/main.tsp, openmeter.tsp, konnect.tsp, test.tsp) drives the Google-AIP-style v3 API. Each package also ships compiled JS libs (lib/index.js with custom AIP/legacy lint rules under lib/rules). Generation orchestrated by Makefile + pnpm scripts (generate/format/lint).
- **Key interfaces:** `legacy TypeSpec`, `aip TypeSpec`

### JavaScript/TypeScript SDK
- **Location:** `api/client/javascript`
- **Responsibility:** Published @openmeter/sdk npm package. src/client/index.ts builds an openapi-fetch typed client (paths from generated schemas.js, encodeDates helper) and composes per-resource wrapper classes (Addons, Apps, Billing, Customers, Debug, Entitlements + EntitlementsV2, Events, Features, Info, Meters, Notifications, Plans, Portal, Subjects, SubscriptionAddons, Subscriptions). src/portal/ is a separate portal client; src/zod/ provides Zod schemas; src/react/context.tsx provides a React context (react is a peerDependency). Generated via orval (orval.config.ts) from the OpenAPI spec; built/tested with biome + vitest.
- **Key interfaces:** `OpenMeter client` (Addons, Apps, Billing, Customers, Debug, Entitlements, EntitlementsV2, Events, Features, Info, Meters, Notifications, Plans, Portal, Subjects, SubscriptionAddons, Subscriptions)

### Python SDK
- **Location:** `api/client/python`
- **Responsibility:** Published `openmeter` Poetry package (Apache-2.0), a corehttp-based client supporting sync + async usage (examples/sync, examples/async). Depends on isodate, corehttp[requests,aiohttp], cloudevents, urllib3. Generated client code from the OpenAPI spec; not imported by the Go backend.
- **Key interfaces:** `OpenMeter Python client` ((sync + async usage-metering client operations))

### Collector (Benthos/Redpanda Connect)
- **Location:** `collector`
- **Responsibility:** Separate Go module + binary that streams external usage events into OpenMeter. cmd/main.go runs a Benthos (Redpanda Connect) CLI assembled from registered plugins: benthos/input, benthos/output, benthos/bloblang (custom bloblang mapping plugins), benthos/services/leaderelection (Kubernetes-style leader election CLI opts), plus benthos/presets and benthos/internal. Pulls in redpanda-data/benthos v4 + connect free bundle. Independent go.mod, so not part of the root module's import graph.
- **Key interfaces:** `collector main` (main)

### Shared utility packages (pkg/)
- **Location:** `pkg`
- **Responsibility:** Cross-cutting Go utilities, several of which are the codebase's biggest dependency magnets: pkg/models (229 in-edges) holds shared domain models + the NewNillableGenericValidationError error aggregation used by Validate() methods; pkg/pagination (119 in-edges); pkg/clock (100 in-edges, FreezeTime/UnFreeze test helpers); pkg/currencyx (91, currency.Code type used in ent schema); pkg/timeutil (82); pkg/framework/entutils (74, Ent mixins + TransactingRepo + TxDriver transaction helpers); pkg/framework/transaction (70, transaction.Driver); pkg/datetime (65); pkg/framework/commonhttp (63, GetMediaType); pkg/filter (53); pkg/framework/transport/httptransport (47, the .With().ServeHTTP handler abstraction used by v3). Also kafka, redis, log, otelx, lockr, models, set, slicesx, sortx, strcase, treex, etc.
- **Key interfaces:** `models` (NewNillableGenericValidationError), `entutils` (TransactingRepo, TransactingRepoWithNoValue, NewTxDriver, ResourceMixin, CustomerAddressMixin, AnnotationsMixin), `httptransport` ((.With(params).ServeHTTP handler abstraction + HandlerOption)), `clock` (FreezeTime, UnFreeze)

### Database migrations
- **Location:** `tools/migrate`
- **Responsibility:** Migration tooling and SQL files. migrations/ holds golang-migrate-format timestamped .up.sql/.down.sql pairs plus atlas.sum, generated from Ent schema diffs via `atlas migrate --env local diff <name>` (atlas.hcl points schema source at ent://openmeter/ent/schema). cmd/viewgen/main.go is a generator for Ent views (which don't auto-appear in generated migrate metadata). Migrations run on startup when postgres.autoMigrate is ent or migration.
- **Depends on:** openmeter/ent/schema
- **Key interfaces:** `viewgen` (main)

## File Placement

| Component Type | Location | Naming | Example |
|---------------|----------|--------|---------|
| Domain service interface | `openmeter/<domain>/` | `<domain>.go / service.go / connector.go in the domain root package` | `openmeter/billing/invoice.go declares billing.Service; openmeter/streaming/connector.go declares streaming.Connector` |
| Service implementation | `openmeter/<domain>/service/` | `service/service.go with a Service struct + Config{...} + Config.Validate() + New(Config)` | `openmeter/billing/service/service.go (Service struct, Config with per-field non-nil Validate); openmeter/notification/service/service.go follows the same shape` |
| Persistence adapter | `openmeter/<domain>/adapter/` | `adapter/adapter.go with an adapter struct holding *entdb.Client + Tx/WithTx/Self` | `openmeter/billing/adapter/adapter.go implements billing.Adapter with HijackTx/WithTx; openmeter/customer/adapter follows suit` |
| Legacy v1 HTTP layer | `openmeter/<domain>/httpdriver/ (or httphandler/)` | `httpdriver/ or httphandler/ co-located inside the domain package` | `openmeter/customer/httpdriver, openmeter/meter/httphandler, openmeter/notification/httpdriver — all imported by openmeter/server/router/router.go` |
| v3 HTTP handler | `api/v3/handlers/<resource>/` | `handlers/<resource>/{handler.go,<operation>.go,convert.gen.go} with Handler interface + New(resolveNamespace, service, options...)` | `api/v3/handlers/customers/handler.go (Handler interface, handler struct, New); per-operation files create.go/get.go/list.go/delete.go/upsert.go` |
| Type conversion | `alongside the package that owns the types (api/, openmeter/billing/, api/v3/handlers/customers/)` | `convert.go (goverter interface) → convert.gen.go (generated); derived.gen.go (goderive)` | `api/convert.go + api/convert.gen.go; openmeter/billing/derived.gen.go; api/v3/handlers/customers/convert.gen.go` |
| Unit / integration tests | `co-located in each package AND top-level test/ (test/billing, test/customer, test/entitlement, test/subscription, test/credits, test/notification, test/app)` | `<name>_test.go co-located with source; cross-domain integration suites under test/<domain>/` | `openmeter/billing/charges/service/usagebased_test.go (co-located); test/billing (shared billing integration harness imported by charges/service tests)` |
| End-to-end tests | `e2e/` | `<feature>_test.go + docker-compose.*.yaml in e2e/` | `e2e/e2e_test.go, e2e/addons_v3_test.go, e2e/customer_credits_v3_test.go, e2e/entitlement_parity_test.go, e2e/docker-compose.infra.yaml` |
| Ent schema (DB source of truth) | `openmeter/ent/schema/` | `<entity>.go with a struct embedding ent.Schema + Mixin()/Fields()/Edges()` | `openmeter/ent/schema/customer.go (Customer ent.Schema, ResourceMixin + CustomerAddressMixin + AnnotationsMixin)` |
| DI / wiring providers | `app/common/` | `one wire.NewSet provider-set file per subsystem` | `app/common/billing.go (var Billing = wire.NewSet(...)), app/common/ledger.go, app/common/openmeter_server.go` |
| Service entrypoints | `cmd/` | `cmd/<service>/main.go` | `cmd/server/main.go, cmd/billing-worker/main.go, cmd/sink-worker/main.go` |
| SQL migrations | `tools/migrate/migrations/` | `<timestamp>_<name>.up.sql / .down.sql + atlas.sum` | `tools/migrate/migrations/*.up.sql generated by `atlas migrate --env local diff`` |

## Naming Conventions

- **Go domain root packages**: single lowercase word matching the domain (e.g. `billing`, `customer`, `subscription`, `entitlement`, `streaming`, `notification`, `ledger`, `meter`)
- **Go implementation subpackages with import aliases**: package named service/adapter but imported with a domain-prefixed alias (e.g. `package billingservice in service/service.go imported as billingservice`, `package billingadapter in adapter/adapter.go`, `package customers in api/v3/handlers/customers`)
- **Go interface-satisfaction assertions**: var _ <Interface> = (*<Struct>)(nil) (e.g. `var _ billing.Service = (*Service)(nil)`, `var _ billing.Adapter = (*adapter)(nil)`, `var _ notification.Service = (*Service)(nil)`)
- **Constructor + config**: type Config struct {...}; func (c Config) Validate() error; func New(config Config) (Iface, error) (e.g. `billingadapter.New(Config)`, `billingservice Config.Validate() returning errors.New per nil field`, `notification service New(Config)`)
- **Generated Go files**: *.gen.go and wire_gen.go and ent/db/ with `// Code generated ... DO NOT EDIT` header (e.g. `api/api.gen.go`, `api/v3/api.gen.go`, `openmeter/billing/derived.gen.go`, `openmeter/billing/service/convert.gen.go`, `wire_gen.go`)
- **Go test files**: <source>_test.go co-located (e.g. `openmeter/billing/service/invoice_test.go`, `openmeter/subscription/patch_test.go`, `openmeter/entitlement/usageperiod_test.go`)
- **TypeScript SDK files**: kebab-case modules, PascalCase exported wrapper classes (e.g. `src/client/subscription-addons.js exporting class SubscriptionAddons`, `src/client/index.ts`, `src/react/context.tsx`)
- **TypeSpec spec files**: lowercase <feature>.tsp under packages/<legacy|aip>/src/ (e.g. `api/spec/packages/legacy/src/meters.tsp`, `api/spec/packages/aip/src/openmeter.tsp`, `api/spec/packages/legacy/src/main.tsp`)
- **Ent schema files**: snake_case or compound lowercase <entity>.go matching the entity (e.g. `openmeter/ent/schema/ledger_account.go`, `openmeter/ent/schema/balance_snapshot.go`, `openmeter/ent/schema/chargesusagebased.go`, `openmeter/ent/schema/customer.go`)