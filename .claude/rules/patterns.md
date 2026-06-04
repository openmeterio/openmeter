## Communication Patterns

### Layered Domain Service / Adapter / HTTP-driver
- **When:** Adding any business-logic capability to a domain under openmeter/<domain>/, separating orchestration (service), persistence (adapter), and HTTP translation (httpdriver/httphandler or api/v3/handlers).
- **How:** Each domain declares a Service interface and an Adapter interface at the package root (service.go / adapter.go). The concrete service lives in <domain>/service/ and calls only the Adapter interface for all DB access; the Ent-backed adapter lives in <domain>/adapter/; v1 HTTP handlers in <domain>/httpdriver/ and v3 handlers in api/v3/handlers/<resource>/. Composite service interfaces are assembled from fine-grained sub-interfaces so callers depend on the narrowest slice (e.g. billing.Service composes ProfileService+InvoiceService+InvoiceLineService+...).

### entutils.TransactingRepo context-propagated Ent transactions (TxCreator+TxUser triad)
- **Scope:** `openmeter/billing`, `openmeter/customer`, `openmeter/entitlement`, `openmeter/subscription`, `openmeter/notification`, `openmeter/ledger`, `openmeter/credit`, `openmeter/productcatalog`
- **When:** Every adapter method body that reads or writes via Ent and must either compose with a caller-supplied transaction or start its own; including multi-step charge/invoice flows that must commit atomically.
- **How:** Each adapter implements the triad: Tx(ctx) hijacks the *entdb.Client tx and wraps with entutils.NewTxDriver; WithTx(ctx, tx) rebuilds the client via entdb.NewTxClientFromRawConfig; Self() returns the raw adapter. Every method body wraps with entutils.TransactingRepo(ctx, a, fn) which reads the *TxDriver from ctx via GetDriverFromContext: if present it rebinds via repo.WithTx(ctx, tx), otherwise it runs on repo.Self(). lockr.LockForTX also reads the same ctx driver, so any lock or nested adapter call inherits the same transaction. The graceful fallback to Self() means a helper that bypasses TransactingRepo silently falls off the transaction with no error.
- **Applicable when:** pkg/framework/entutils/transaction.go:208-220 — TransactingRepo reads *TxDriver from ctx and rebinds (line 220) or falls back to repo.Self() (line 216); the adapter must implement Tx/WithTx/Self (openmeter/customer/adapter/adapter.go:52-70 implements all three). Applies to any adapter method that participates in a multi-step ctx-carried transaction.
- **Do NOT apply when:**
  - Adapter helper bodies that operate directly on a.db.Foo() or accept a raw *entdb.Client without wrapping in TransactingRepo/TransactingRepoWithNoValue — transaction.go:216 silently degrades to Self() (non-tx client) with no error, producing partial writes in flows like charges AdvanceCharges/ApplyPatches (openmeter/billing/charges/adapter/search.go)
  - Adapter structs that store a *entdb.Tx as a struct field instead of rebinding via ctx — the captured tx falls off any caller-supplied ctx transaction

### Per-customer billing lock via SELECT FOR UPDATE on BillingCustomerLock
- **Scope:** `openmeter/billing`, `openmeter/billing/charges`
- **When:** Serializing concurrent invoice/charge mutations for the same customer so concurrent billing-worker goroutines and API requests cannot race on invoice or line creation.
- **How:** billing.Service.WithLock wraps the operation in a transaction that UpsertCustomerLock (idempotent insert, OnConflict DoNothing) then LockCustomerForUpdate issues SELECT ... FOR UPDATE on the single BillingCustomerLock row keyed (namespace, customer_id). The lock auto-releases on commit/rollback. This serializes exactly one customer because the (namespace, customer_id) tuple maps to at most one row.
- **Applicable when:** openmeter/ent/schema/billing.go:1356 — BillingCustomerLock declares index.Fields("namespace", "customer_id").Unique(), so the (namespace, customerID) tuple maps to at most one row and the SELECT FOR UPDATE serializes exactly that customer. BillingCustomerOverride mirrors this with the same UNIQUE at billing.go:258.
- **Do NOT apply when:**
  - Locking on a column set that has no UNIQUE index on the locked entity — the row lock would either serialize unrelated rows (if multiple rows share the key) or not exist at all to lock
  - An operation that mutates a single row only — Ent's built-in row-level locking suffices; there is no cross-row invariant requiring the customer-scoped lock

### Generic pg advisory lock via lockr (pg_advisory_xact_lock) inside an Ent transaction
- **Scope:** `openmeter/billing`, `openmeter/billing/charges`, `openmeter/entitlement`
- **When:** Serializing concurrent mutations of the same logical key (a charge, or a feature-key+customer scope) where no dedicated single-row lock table exists; the generic equivalent of the BillingCustomerLock SELECT FOR UPDATE.
- **How:** lockr.NewKey(scopes...) builds a key whose Hash64() is an xxh3 hash of the colon-joined scopes (pkg/framework/lockr/key.go). LockForTX runs SELECT pg_advisory_xact_lock($1) with int64(key.Hash64()) (locker.go:66). getTxClient reads the tx driver from ctx via entutils.GetDriverFromContext and verifies it is a real Postgres tx by checking transaction_timestamp() != statement_timestamp(), erroring 'lockr only works in a postgres transaction' otherwise (locker.go:109-135). The advisory lock auto-releases on tx commit/rollback. Charge keys come from charges.NewLockKeyForCharge (scopes namespace+charge ID); entitlement keys from NewEntitlementUniqueScopeLock (scopes feature-key+customer).
- **Applicable when:** pkg/framework/lockr/locker.go:109-135 — getTxClient requires an active Postgres transaction in ctx (transaction_timestamp() != statement_timestamp()), returning an error otherwise; so LockForTX is correct ONLY for callers already inside an entutils.TransactingRepo-established transaction. The lock key must map to the intended serialization scope: charges.NewLockKeyForCharge (openmeter/billing/charges/lock.go:11) scopes on the charge primary key (namespace+id), which is unique.
- **Do NOT apply when:**
  - Caller is outside an active Postgres transaction (e.g. an autocommit connection) — locker.go:135 returns 'lockr only works in a postgres transaction' and the advisory lock acquires nothing useful
  - Acquisition is wrapped in context.WithTimeout — pgx cancels the connection on ctx cancel, corrupting the tx; use pgdriver.WithLockTimeout instead
  - The chosen lock-key scopes do not map 1:1 to the entity you intend to serialize — Hash64() collapses scopes to a uint64 (key.go:51), so keying on a non-unique scope (e.g. only a status or type column) would silently serialize unrelated entities under one hash. NewEntitlementUniqueScopeLock (openmeter/entitlement/service/lock.go:6) deliberately scopes on (feature-key, customer-id) to serialize that combination, not a single physical row

### Kafka + Watermill pub/sub with three prefix-routed topics
- **Scope:** `openmeter/watermill`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification`, `openmeter/sink`, `openmeter/ingest`
- **When:** Any async domain-event delivery between the binaries (subscription lifecycle, invoice advance, ingest flush, balance recalculation).
- **How:** eventbus.New wraps Watermill cqrs.NewEventBusWithConfig. GeneratePublishTopic does strings.HasPrefix on params.EventName against ingestevents.EventVersionSubsystem+'.' and balanceworkerevents.EventVersionSubsystem+'.', routing to IngestEventsTopic / BalanceWorkerEventsTopic respectively, and DEFAULTING all other prefixes to SystemEventsTopic with no error. Producers call Publisher.Publish or WithContext(ctx).PublishIfNoError. Topic isolation matches worker topology so ingest bursts cannot starve billing system-event consumers.
- **Applicable when:** openmeter/watermill/eventbus/eventbus.go:141-142 — the default switch case returns opts.TopicMapping.SystemEventsTopic for any unrecognized EventName prefix; so the routing invariant holds ONLY for event families whose EventName() begins with a registered EventVersionSubsystem constant (ingestevents at eventbus.go:137 or balanceworkerevents at eventbus.go:139). Everything else is silently treated as a system event.
- **Do NOT apply when:**
  - An ingest- or balance-worker-bound event family whose EventName() lacks the matching EventVersionSubsystem prefix — eventbus.go:142 silently routes it to SystemEventsTopic, bypassing topic isolation and the dedicated consumer
  - A producer in a binary other than balance-worker emitting balanceworkerevents.* — the BalanceWorkerEventsTopic is a dedicated recalculation queue consumed only by the balance-worker

### NoPublishingHandler silent-drop dispatch by CloudEvents ce_type
- **Scope:** `openmeter/watermill/grouphandler`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification/consumer`
- **When:** Consumer-side dispatch of a Kafka message to typed handlers in a worker router, where producer/consumer version skew during rolling deploys must be tolerated.
- **How:** NoPublishingHandler.Handle reads the ce_type via marshaler.NameFromMessage and looks it up in typeHandlerMap. If no handler is registered (or the slice is empty) it increments an 'ignored' metric and returns nil (ACK, silent drop) — grouphandler.go:48-55. For a matched type it unmarshals one event instance and fans out to all registered handlers via errors.Join(lo.Map(...)) passing msg.Context() so any handler failure surfaces and triggers Watermill retry.
- **Applicable when:** openmeter/watermill/grouphandler/grouphandler.go:49-55 — Handle returns nil for any ce_type absent from typeHandlerMap; the silent-drop contract is correct ONLY for consumers that must tolerate producer/consumer version skew during rolling deploys. It cannot distinguish 'unknown event type' from 'known type whose payload version the consumer cannot decode'.
- **Do NOT apply when:**
  - Multiple handlers registered for the same ce_type that mutate the shared event pointer — grouphandler.go:57 unmarshals one instance (groupHandler[0].NewEvent()) and passes the same pointer to every handler via errors.Join/lo.Map, so concurrent mutation races
  - A handler that needs unknown/undecodable versions surfaced to a DLQ — grouphandler.go returns nil (ACK) and drops; returning an error here would instead poison the DLQ for valid messages of other families on the same topic

### Watermill default router middleware stack (router.NewDefaultRouter)
- **Scope:** `openmeter/watermill/router`, `openmeter/billing/worker`, `openmeter/notification/consumer`, `openmeter/entitlement/balanceworker`
- **When:** Building any consumer router in a worker or server binary that consumes the three Kafka topics.
- **How:** router.NewDefaultRouter wires a fixed middleware stack (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout, HandlerMetrics) so all consumers inherit dead-letter routing, retries, OTel span propagation, and processing-timeout enforcement. Handlers are added as AddNoPublisherHandler with a NoPublishingHandler dispatcher.

### Sink worker three-phase flush (ClickHouse -> Kafka offset -> Redis dedupe)
- **Scope:** `openmeter/sink`, `cmd/sink-worker`
- **When:** High-throughput batch ingestion of raw CloudEvents from Kafka into ClickHouse with exactly-once semantics.
- **How:** Sink.flush dedupes in-batch, then executes strictly: (1) persistToStorage -> Storage.BatchInsert into the shared ClickHouse MergeTree events table (sink.go:330,464); (2) Consumer.StoreMessage per Kafka offset (sink.go:344); (3) dedupeSet -> Redis SET NX with retry, only when a Deduplicator is configured (sink.go:354). After all three phases FlushEventHandler.OnFlushSuccess fires in a goroutine bounded by FlushSuccessTimeout (sink.go:391-395) so the post-flush balance-recalc notification never blocks the consumer loop. The MergeTree is not deduplicated by the engine; dedup is entirely upstream in Redis keyed namespace-source-id with TTL.
- **Applicable when:** openmeter/sink/sink.go:330-354 orders persistToStorage (phase 1, line 330) strictly before Consumer.StoreMessage (phase 2, line 344) and dedupeSet (phase 3, line 354); exactly-once holds ONLY while ClickHouse is written before the Kafka offset commits — on consumer restart an uncommitted offset re-delivers messages not yet in ClickHouse.
- **Do NOT apply when:**
  - Reordering so Redis dedupe (sink.go:354) is set before the Kafka offset commit (sink.go:344) — a crash after dedupe but before commit marks events processed while ClickHouse re-reads the uncommitted offset, dropping them
  - Committing the Kafka offset before the ClickHouse BatchInsert (sink.go:464) — loses events on crash between commit and insert
  - Calling FlushEventHandler.OnFlushSuccess synchronously instead of in the goroutine at sink.go:391 — blocks the main sink loop and causes Kafka partition backpressure

### Ingest Collector with DeduplicatingCollector wrapper
- **Scope:** `openmeter/ingest`, `openmeter/sink`
- **When:** Receiving a single CloudEvent at the ingest edge and forwarding it to Kafka, with optional upstream deduplication.
- **How:** ingest.Collector is an interface (Ingest, Close). The concrete collector forwards CloudEvents to Kafka; DeduplicatingCollector wraps any Collector and consults a Deduplicator (Redis or in-memory LRU) before forwarding (ingest/dedupe.go). InMemoryCollector is a test/dev fallback. All ingest flows must go through Collector so the dedupe layer runs before events reach ClickHouse via the sink worker.

### Namespace Manager fan-out (multi-tenancy provisioning)
- **Scope:** `openmeter/namespace`, `cmd/server`
- **When:** Provisioning/deprovisioning a tenant across all subsystems (ClickHouse streaming, Kafka ingest, Ledger).
- **How:** namespace.Manager holds a slice of registered Handler implementations. createNamespace and DeleteNamespace iterate every handler and aggregate failures with errors.Join (no short-circuit) — namespace.go:105-118,135. RegisterHandler appends handlers (namespace.go:76); CreateDefaultNamespace calls createNamespace with the configured default name (namespace.go:92-93). The default namespace is protected from deletion (namespace.go:68).
- **Applicable when:** openmeter/namespace/namespace.go:92-93,105-118 — CreateDefaultNamespace fans out only over handlers already present in the slice at call time; so a Handler provisions the default namespace ONLY if RegisterHandler (namespace.go:76) was called before CreateDefaultNamespace. The fan-out uses errors.Join with no short-circuit, so a partial-provisioning failure does not block startup.
- **Do NOT apply when:**
  - Registering a namespace.Handler after CreateDefaultNamespace has already run — namespace.go:105-118 iterates only the handlers present at call time, so a late handler's subsystem is never initialized for the default tenant
  - A worker binary (cmd/billing-worker, cmd/balance-worker, cmd/sink-worker) relying on namespace-handler-provisioned subsystems without registering the handlers itself — only cmd/server registers Ledger/KafkaIngest handlers, so workers must fail-fast verify the default namespace exists rather than assume the fan-out ran

### ServiceHook Registry for cross-domain lifecycle callbacks
- **Scope:** `openmeter/customer`, `openmeter/subscription`, `openmeter/app`, `openmeter/ledger`, `app/common`
- **When:** Reacting to another domain's entity lifecycle (billing reacting to subscription/customer events, ledger reacting to customer creation) without importing that domain's service.
- **How:** pkg/models.ServiceHookRegistry[T] fans out PreCreate/PostCreate/PreUpdate/PostUpdate/PreDelete/PostDelete to all registered hooks under an RWMutex. Re-entrancy is prevented by a per-registry loop key derived from pointer identity (fmt.Sprintf('service-hook-registry-%p', r)) stored in ctx — servicehook.go:42. Domain services embed the registry and expose RegisterHooks; registration happens as a side-effect inside app/common provider functions (e.g. customerService.RegisterHooks) to avoid circular imports.
- **Applicable when:** pkg/models/servicehook.go:42 — the loop-prevention key is derived from the registry's own pointer (fmt.Sprintf('...%p', r)); the re-entrancy guard is correct ONLY while the registry is shared by pointer. Copying the registry value yields a different %p and defeats loop prevention.
- **Do NOT apply when:**
  - Registering a hook inside a domain package's own constructor instead of an app/common provider — Wire models types not side-effects, so omitting the provider from a binary's wire.Build silently drops the hook with no compile error (app/common/customer.go:61-62 registers via the provider, not the domain constructor)
  - Pre-mutation blocking validation — that belongs in the customer RequestValidator registry (openmeter/customer/requestvalidator.go), not in post-lifecycle ServiceHooks

### Customer RequestValidator registry (pre-mutation cross-domain guards)
- **Scope:** `openmeter/customer`, `openmeter/billing`, `openmeter/entitlement`, `app/common`
- **When:** Blocking a customer mutation (create/update/delete) based on another domain's constraints (billing unpaid invoices, entitlement existence) before any DB write.
- **How:** customer.RequestValidator is an interface (ValidateCreateCustomer, ValidateUpdateCustomer, ValidateDeleteCustomer) with a NoopRequestValidator default. The RequestValidatorRegistry fans out registered validators with errors.Join (no short-circuit) before any mutation. Validators are registered as side-effects inside app/common provider functions (customerService.RegisterRequestValidator) to keep billing/entitlement constraints out of the customer package. This is strictly pre-mutation and blocking, distinct from post-lifecycle ServiceHooks.
- **Do NOT apply when:**
  - Post-mutation reactions (e.g. provisioning ledger accounts after customer creation) — those belong in ServiceHooks (PostCreate/PostUpdate/PostDelete), not the RequestValidatorRegistry which is exclusively pre-mutation blocking validation (openmeter/customer/requestvalidator.go:13-16 declares only Validate* methods)

### Tagged-union domain models with constructor-only construction (Charge / ChargeIntent / InvoiceLine)
- **Scope:** `openmeter/billing`, `openmeter/billing/charges`
- **When:** Modeling billing entities with several mutually exclusive sub-types requiring exhaustive dispatch and impossible partial construction.
- **How:** Charge carries a private discriminator t meta.ChargeType plus three nullable sub-type pointers (charge.go:20-26); it is set ONLY by NewCharge[T] (charge.go:32-50) constrained to flatfee/usagebased/creditpurchase. Typed accessors AsFlatFeeCharge/AsCreditPurchaseCharge/AsUsageBasedCharge (charge.go:79/91/103) return an error when c.t does not match. NewChargeIntent[T] (charge.go:276) does the same for intents. InvoiceLine uses an analogous private discriminator set only by NewStandardInvoiceLine/NewGatheringInvoiceLine. A struct literal leaves the discriminator zero-valued so every accessor errors.
- **Applicable when:** openmeter/billing/charges/charge.go:20-21,32 — the discriminator field 't meta.ChargeType' is unexported and set only by NewCharge[T] (line 32); accessors at charge.go:79/91/103 return an error when the discriminator does not match the requested type. So a struct literal charges.Charge{} leaves t zero-valued and all As* accessors error — construction is correct ONLY via the generic constructor.
- **Do NOT apply when:**
  - Constructing charges.Charge{}, charges.ChargeIntent{}, or billing.InvoiceLine{} via a struct literal — the private discriminator (charge.go:21) stays zero-valued and the typed accessors error

### Invoice / Charge state machine (stateless library, sync.Pool backed)
- **Scope:** `openmeter/billing`, `openmeter/billing/service`, `openmeter/billing/charges`
- **When:** Driving StandardInvoice or charge lifecycle transitions so post-transition actions (DB save, event publish, external app calls) fire atomically.
- **How:** billing/service/stdinvoicestate.go builds a *stateless.StateMachine via stateless.NewStateMachineWithExternalStorage bound to Invoice.Status (stdinvoicestate.go:24-77), pooled in invoiceStateMachineCache (sync.Pool, line 35) to reduce GC pressure on the hot path. FireAndActivate fires a trigger and persists. The generic charges Machine[CHARGE,BASE,STATUS] mirrors this with value-copy WithStatus/WithBase semantics (charges/statemachine). Direct Invoice.Status mutation is forbidden because it skips the post-transition actions.
- **Applicable when:** openmeter/billing/charges/statemachine/machine.go — the generic Machine updates its Charge field by assignment, so ChargeLike WithStatus/WithBase must return value copies (value receivers); a pointer-mutating implementation breaks the external-storage assignment and aliases the prior state.
- **Do NOT apply when:**
  - Mutating Invoice.Status (or a charge status field) directly instead of firing the stateless transition — skips the DB save and event publish wired to the transition (openmeter/billing/service/stdinvoicestate.go)
  - Implementing ChargeLike WithStatus/WithBase as pointer receivers — Machine.Charge is updated by assignment, so a pointer mutation aliases shared state

### App Factory / Marketplace Registry (InvoicingApp protocol, self-registration in New())
- **Scope:** `openmeter/app`, `openmeter/billing`
- **When:** Plugging a billing backend (Stripe, Sandbox, CustomInvoicing) into the invoice state machine without hardcoding it in billing.Service.
- **How:** app.Service exposes RegisterMarketplaceListing(ctx, RegistryItem{Listing, Factory}) (service.go:12). Each concrete app calls it inside its own constructor New()/NewFactory() as a self-registration side-effect — stripe/service/service.go:89, sandbox/app.go:196, custominvoicing/factory.go:87. Registration failure returns an error so the listing is always present before any app instance. Each app implements billing.InvoicingApp (ValidateStandardInvoice, UpsertStandardInvoice, FinalizeStandardInvoice, DeleteStandardInvoice); invoices passed to app callbacks are read-only snapshots and external IDs are merged back via a builder under billing-service control.
- **Applicable when:** openmeter/app/stripe/service/service.go:89 (and sandbox/app.go:196, custominvoicing/factory.go:87) — the app type is registered by calling config.AppService.RegisterMarketplaceListing inside New()/NewFactory(); the listing is invisible to app.Service's marketplace catalog unless this constructor side-effect runs. So a new billing backend must self-register in its constructor, not at a separate wire step.
- **Do NOT apply when:**
  - Hardcoding provider-specific branches inside billing.Service instead of implementing billing.InvoicingApp and self-registering — the registry/factory indirection (openmeter/app/service.go:12) is the only intended extension point

### LineEngine registry (runtime-pluggable billing line calculation)
- **Scope:** `app/common`, `openmeter/billing`, `openmeter/billing/charges`
- **When:** Adding a new charge type whose line calculation must dispatch by LineEngineType without editing billing core.
- **How:** billing.Service exposes RegisterLineEngine (LineEngineService); the service holds a map[LineEngineType]LineEngine under a RWMutex. Each charge type implements its Engine and is registered at Wire startup exclusively in app/common/charges.go as a provider side-effect, never from a domain package or cmd/*.
- **Do NOT apply when:**
  - Registering a LineEngine from a domain package or a cmd/* binary instead of app/common/charges.go — registration is a Wire provider side-effect concentrated in app/common to avoid import cycles

### Google Wire DI with app/common provider sets and noop-for-disabled-features
- **Scope:** `app/common`, `cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`
- **When:** Composing the dependency graph for each binary; gating optional features (credits.enabled=false, Svix unconfigured) without nil-checks in business logic.
- **How:** Each cmd/<binary>/wire.go declares wire.Build over composite provider sets defined in app/common/ (per-domain files plus openmeter_<binary>.go). Domain packages expose plain constructors and never import app/common (the import direction is one-way outward). Optional features return noop interface implementations rather than nil: app/common/ledger.go returns ledgernoop.AccountService{}/Ledger{}/AccountResolver{} when !creditsConfig.Enabled (ledger.go:74-114), and app/common/customer.go returns NoopCustomerLedgerHook{} (customer.go:61-62). NewLedgerNamespaceHandler type-asserts against the noop AccountResolver (ledger.go:133-134). Credits is guarded independently at four wiring layers.
- **Applicable when:** app/common/ledger.go:74-114 — each ledger provider checks creditsConfig.Enabled and returns a ledgernoop.* implementation when disabled; the credits guard must be repeated at every independent provider (ledger services, customer ledger hooks at customer.go:61, ChargesRegistry skip in billing.go, v3 credit handlers) because Wire verifies types not the policy. A new ledger-writing provider added without the creditsConfig.Enabled branch silently re-enables ledger writes for credits-disabled deployments.
- **Do NOT apply when:**
  - Returning nil instead of a noop struct for a disabled optional feature — callers receive the interface and panic on nil (app/common/ledger.go:75 returns ledgernoop.AccountService{}, not nil)
  - Adding a new ledger-touching Wire provider without a creditsConfig.Enabled branch — the four-layer guard is only as complete as the most recently added provider
  - Registering hooks/validators inside a domain constructor instead of an app/common provider — omitting the provider from a binary's wire.Build silently drops the side-effect

### httptransport decode/operate/encode pipeline with GenericErrorEncoder chain
- **Scope:** `openmeter/billing/httpdriver`, `openmeter/customer/httpdriver`, `openmeter/meter/httphandler`, `api/v3/handlers`
- **When:** Every v1 (httpdriver) and v3 (api/v3/handlers) HTTP endpoint.
- **How:** httptransport.NewHandler / NewHandlerWithArgs composes a RequestDecoder, an operation.Operation, and a ResponseEncoder, appending commonhttp.GenericErrorEncoder as the last error encoder. Error encoders return bool (first match wins, short-circuiting double-writes). Domain errors are models.Generic* sentinels matched by type (encoder.go:138-146); ValidationIssue HTTP status is carried as an attribute read by HandleIssueIfHTTPStatusKnown. Unmatched errors fall through to a 500 (handler.go:132).
- **Do NOT apply when:**
  - Implementing http.Handler.ServeHTTP directly instead of via httptransport.NewHandler — bypasses the GenericErrorEncoder chain (pkg/framework/commonhttp/encoder.go:136) and the OTel/validation middleware
  - Writing HTTP status codes in handler logic instead of returning models.Generic* sentinels — encoder.go:138-146 maps the sentinel types to status codes; a plain fmt.Errorf falls through to 500

### TypeSpec single-source API generation (v1 + v3 + three SDKs)
- **Scope:** `api/spec`, `openmeter/server`, `api/v3/server`, `api/v3/handlers`
- **When:** Adding or changing any HTTP endpoint, request/response type, or SDK contract.
- **How:** Endpoints are authored in TypeSpec under api/spec/packages/legacy (v1) or api/spec/packages/aip (v3), with route/tag bindings only in the root openmeter.tsp. make gen-api compiles to api/openapi.yaml + api/v3/openapi.yaml then oapi-codegen produces api/api.gen.go, api/v3/api.gen.go, and the Go/JS/Python SDKs; make generate then propagates to Ent/Wire/Goverter/Goderive. Generated ServerInterfaces are implemented in openmeter/<domain>/httpdriver (v1) or api/v3/handlers/<resource> (v3). Both regen steps are mandatory; the .yaml and *.gen.go files carry DO NOT EDIT headers.
- **Do NOT apply when:**
  - Hand-editing api/openapi.yaml, api/v3/openapi.yaml, api/api.gen.go, or api/v3/api.gen.go — they carry generated headers and are overwritten on the next make gen-api
  - Adding an endpoint only in a Go handler package without a TypeSpec source change — the SDKs and the other API version drift
  - Declaring @route/@tag in a domain sub-folder .tsp instead of the root openmeter.tsp

### Subscription spec patch interface (AppliesToSpec.ApplyTo)
- **Scope:** `openmeter/subscription`
- **When:** Mutating a subscription's in-memory SubscriptionSpec during create/edit/change/restore/addon operations.
- **How:** All SubscriptionSpec mutations flow through the AppliesToSpec interface (apply.go:20-23): ApplyTo(spec *SubscriptionSpec, actx ApplyContext) error. NewAppliesToSpec wraps a function into the interface (apply.go:35). The workflow service composes patches and applies them via ApplyTo so spec invariants and ordering constraints are maintained; direct field mutation on the spec bypasses these guards.
- **Do NOT apply when:**
  - Mutating SubscriptionSpec fields directly (e.g. spec.Phases = append(...)) instead of applying a patch via ApplyTo (openmeter/subscription/apply.go:23) — bypasses patch validation and ordering invariants

### ValidationIssue immutable with-chain + GenericValidationError aggregation
- **Scope:** `openmeter`
- **When:** Producing structured, field-pathed validation errors that map to HTTP statuses and aggregate across collectors.
- **How:** models.ValidationIssue is an immutable value type built via copy-on-write With* methods (WithPathString, WithComponent, WithSeverity, WithHTTPStatusCodeAttribute); the HTTP layer reads the status-code attribute via HandleIssueIfHTTPStatusKnown. Validate() implementations collect issues into errs and return models.NewNillableGenericValidationError(errors.Join(errs...)). GenericErrorEncoder type-matches the resulting Generic* sentinel to a status code.
- **Do NOT apply when:**
  - Mutating a ValidationIssue's struct fields directly instead of using the copy-on-write With* methods (pkg/models/validationissue.go) — the type is designed immutable and direct assignment corrupts shared instances passed through multiple service layers

### OTel tracing via tracex helpers
- **When:** Instrumenting service/adapter methods that initiate significant work, with automatic error recording and panic recovery.
- **How:** pkg/framework/tracex.Start/Wrap wrap an injected trace.Tracer, automatically recording errors, setting span status, and recovering panics — superior to calling tracer.Start directly which requires those three steps manually. The tracer is injected via Wire into service constructors; ctx (not context.Background()) is propagated so spans nest across HTTP handlers, Kafka consumers, and Ent adapters.

## Integrations

| Service | Purpose | Integration point |
|---------|---------|-------------------|
| PostgreSQL (Ent ORM + Atlas migrations) | Authoritative relational store for all ~35 domain entities (billing invoices/lines, charges, customers, subscriptions, entitlements, credit grants, double-entry ledger, meters, features/plans, notifications, LLM cost prices, secrets, subjects). | `openmeter/ent/schema/ (source of truth) -> generated openmeter/ent/db/ -> Atlas migrations in tools/migrate/migrations/; accessed via *entdb.Client through entutils.TransactingRepo (pkg/framework/entutils/transaction.go) and pgx driver.` |
| ClickHouse | Append-only single shared MergeTree analytics store for raw usage CloudEvents; queried for meter aggregations and batch-inserted by the sink worker. DDL created by the connector at startup (CREATE TABLE IF NOT EXISTS), not by Atlas. | `openmeter/streaming/connector.go (RawEvent struct, Connector interface QueryMeter/ListEvents/BatchInsert/CreateNamespace), openmeter/streaming/clickhouse/event_query.go, openmeter/sink/storage.go (BatchInsert called from sink.go:464).` |
| Kafka (confluent-kafka-go + Watermill) | Durable cross-binary event bus isolated into three prefix-routed topics (ingest, system, balance-worker) plus the raw ingest CloudEvents stream consumed by the sink worker. | `openmeter/watermill/eventbus/eventbus.go (prefix routing at lines 137-142), openmeter/watermill/router/router.go (middleware stack), openmeter/watermill/grouphandler/grouphandler.go (ce_type dispatch); confluent-kafka-go used directly in openmeter/sink/sink.go for the ingest consumer; topic provisioning via app/common KafkaTopicProvisioner.` |
| Redis | TTL-based ingest deduplication store (SET NX on namespace-source-id keys) to prevent double-counting on retry; in-memory LRU fallback when Redis is unconfigured. | `openmeter/dedupe/redisdedupe/redisdedupe.go (IsUnique/CheckUniqueBatch/Set), openmeter/dedupe/memorydedupe/memorydedupe.go; invoked as the third sink flush phase in openmeter/sink/sink.go:354 (dedupeSet).` |
| Svix | Outbound webhook delivery for notification events (balance thresholds, invoice events); a noop webhook.Handler runs when Svix is unconfigured. | `openmeter/notification/webhook/svix/svix.go (concrete Handler), openmeter/notification/consumer (subscribes to system topic and dispatches), openmeter/notification/webhook (noop fallback).` |
| Stripe (stripe-go/v80) | Invoice syncing (upsert draft, finalize, collect payment) and customer sync for billing-enabled namespaces; implements the billing.InvoicingApp protocol. | `openmeter/app/stripe/ implements billing.InvoicingApp and self-registers via RegisterMarketplaceListing (stripe/service/service.go:89); REST client under openmeter/app/stripe/client/, adapter at openmeter/app/stripe/adapter.go.` |
| Sandbox Invoicing App | No-op invoicing app to drive the invoice state machine in dev/test without external dependencies; also implements InvoicingAppPostAdvanceHook. | `openmeter/app/sandbox/ implements billing.InvoicingApp; self-registers via RegisterMarketplaceListing (sandbox/app.go:196, sandbox/mock.go:255).` |
| CustomInvoicing App | Webhook-driven invoicing letting external systems receive invoice payloads and async-confirm sync; implements InvoicingApp + InvoicingAppAsyncSyncer. | `openmeter/app/custominvoicing/ self-registers via RegisterMarketplaceListing (custominvoicing/factory.go:87).` |
| GOBL (invopop/gobl) | Currency-safe numeric arithmetic and ISO 4217 currency validation in billing and subscription pricing. | `github.com/invopop/gobl imported across openmeter/productcatalog, openmeter/subscription, openmeter/billing, pkg/currencyx.` |
| OpenTelemetry | Distributed tracing and metrics across all binaries with automatic error recording and panic recovery. | `trace.Tracer injected via Wire; pkg/framework/tracex span helpers; metric.Meter used in openmeter/watermill/grouphandler and openmeter/sink; telemetry exporters bootstrapped in app/common.` |
| TypeSpec compiler | Single source of truth for the v1 + v3 HTTP API definitions, compiling to OpenAPI YAML and Go/JS/Python SDKs. | `api/spec/packages/ (aip v3, legacy v1); make gen-api runs tsp compile then oapi-codegen producing api/openapi.yaml, api/v3/openapi.yaml, api/api.gen.go, api/v3/api.gen.go, api/client/{go,javascript,python}.` |
| App Marketplace Registry (extension protocol) | Runtime-pluggable billing backends that self-register without hardcoded references in billing core. | `openmeter/app/service.go app.Service.RegisterMarketplaceListing; each app self-registers a RegistryItem{Listing, Factory} in its constructor (stripe/service/service.go:89, sandbox/app.go:196, custominvoicing/factory.go:87).` |
| LineEngine Registry (extension protocol) | Runtime-pluggable billing line calculation dispatch by LineEngineType. | `billing.Service.RegisterLineEngine (LineEngineService); engines registered at Wire startup in app/common/charges.go, never from domain packages or cmd/*.` |
| ServiceHook & RequestValidator Registries (extension protocol) | Cross-domain lifecycle callbacks (post-mutation) and pre-mutation blocking guards without circular imports. | `pkg/models/servicehook.go ServiceHookRegistry[T] (pointer-identity loop key at line 42); openmeter/customer/requestvalidator.go RequestValidatorRegistry; both registered as side-effects in app/common provider functions (app/common/customer.go:61, app/common/billing.go).` |
| namespace.Handler fan-out (extension protocol) | Per-namespace resource provisioning across ClickHouse streaming, Kafka ingest, and Ledger. | `openmeter/namespace/namespace.go Manager.RegisterHandler (line 76); handlers must register before CreateDefaultNamespace (line 92); fan-out aggregates with errors.Join (lines 105-118), no short-circuit.` |

## Pattern Selection Guide

| Scenario | Pattern | Rationale |
|----------|---------|-----------|
| Adding a new domain capability (a new billing/customer/entitlement sub-feature) | Layered Domain Service/Adapter/HTTP-driver | Define the sub-interface in <domain>/service.go, implement in <domain>/service/, add adapter methods in <domain>/adapter/, wire in app/common/<domain>.go; keeps business logic, persistence, and HTTP independently testable and mockable (services call the Adapter interface only). |
| Any adapter DB read/write that may run inside a multi-step transaction | entutils.TransactingRepo / TransactingRepoWithNoValue | Rebinds to the ctx-bound Ent transaction if present (transaction.go:220) or runs on Self() (line 216), preventing partial writes during AdvanceCharges/invoice-mutation flows; the silent Self() fallback makes a missing wrapper undetectable at compile time, so every helper must wrap. |
| Serializing concurrent invoice/charge mutation for one customer | billing.Service.WithLock -> SELECT FOR UPDATE on BillingCustomerLock | The UNIQUE(namespace, customer_id) index at billing.go:1356 means the row lock serializes exactly that customer; the lock auto-releases on commit/rollback. |
| Serializing per-charge or feature-key+customer mutation where no dedicated lock table exists | lockr.LockForTX (pg_advisory_xact_lock) inside a TransactingRepo transaction | Generic advisory lock keyed by an xxh3 hash of typed scopes; getTxClient verifies a real Postgres tx (locker.go:109-135). Use charges.NewLockKeyForCharge / NewEntitlementUniqueScopeLock so the hash maps to the intended scope, not a non-unique column. |
| Delivering a domain event to another binary | eventbus.Publisher (prefix-routed to one of three topics) | Topic isolation matches worker topology; producers stay decoupled from consumer topology by routing on the EventVersionSubsystem prefix (eventbus.go:137-142). Unrecognized prefixes default to SystemEventsTopic, so the EventName() must carry a registered prefix. |
| Consuming events in a worker | router.NewDefaultRouter + grouphandler.NoPublishingHandler | Inherits the fixed DLQ/retry/OTel/timeout middleware stack and silently drops unknown ce_types (grouphandler.go:49-55) for rolling-deploy tolerance; handlers use msg.Context() to keep the tx/span. |
| Batch ingesting usage events from Kafka into ClickHouse | Sink three-phase flush | ClickHouse BatchInsert -> Kafka offset commit -> Redis dedupe ordering (sink.go:330,344,354) preserves exactly-once on consumer restart; OnFlushSuccess runs in a goroutine (sink.go:391) so it never blocks the consumer loop. |
| Reacting to another domain's entity lifecycle without import cycles | ServiceHookRegistry[T] registered in app/common | Avoids circular imports between billing/customer/subscription/ledger; the pointer-identity loop key (servicehook.go:42) prevents re-entrancy. Registration is a Wire provider side-effect (app/common/customer.go:61). |
| Blocking a customer mutation based on another domain's constraints | Customer RequestValidator registry | Pre-mutation guards fan out via errors.Join before any DB write, keeping billing/entitlement constraints out of the customer package; distinct from post-lifecycle ServiceHooks. |
| Invoice or charge lifecycle transition | stateless InvoiceStateMachine or generic Machine[CHARGE,BASE,STATUS] | Enforces valid transition sequences and fires post-transition actions (DB save, event publish) atomically; sync.Pool (stdinvoicestate.go:35) reduces GC pressure on the hot path. Direct status mutation is forbidden. |
| Adding a new billing backend (payment processor / invoicing system) | Implement billing.InvoicingApp + self-register via RegisterMarketplaceListing | No core billing.Service changes; the read-only invoice snapshot plus a results builder limit the app's writable surface, and self-registration in New() (stripe/service/service.go:89) keeps registration co-located with the implementation. |
| Disabling an optional subsystem (credits off, no Svix) | Return a noop implementation from the Wire provider | Keeps the DI graph uniform with no nil-checks (app/common/ledger.go:74-114); credits requires four independent guards (ledger services, customer hooks, ChargesRegistry, v3 handlers) because Wire verifies types, not the credits policy. |
| Returning an error to an HTTP client | models.Generic* sentinel + GenericErrorEncoder chain | Type-matched mapping to RFC 7807 problem+json (encoder.go:138-146); a plain fmt.Errorf falls through to 500 (handler.go:132). |
| Adding or changing an HTTP endpoint | Author TypeSpec then make gen-api && make generate | Single source of truth makes drift between v1/v3 stubs and the three SDKs structurally impossible; the .yaml and *.gen.go outputs carry DO NOT EDIT headers. |
| Provisioning/deprovisioning a tenant across subsystems | namespace.Manager fan-out; register Handlers before CreateDefaultNamespace | Fans out to all handlers present at call time with errors.Join (namespace.go:105-118); a handler registered after CreateDefaultNamespace misses default-namespace provisioning. |
| Mutating a subscription's in-memory spec | AppliesToSpec.ApplyTo patch interface | All SubscriptionSpec mutations go through ApplyTo (apply.go:23) so spec invariants and ordering constraints hold; direct field mutation bypasses the guards. |

## Quick Pattern Lookup

- **new domain feature** -> Layered Domain Service/Adapter/HTTP-driver in openmeter/<domain>/  *(scope: openmeter/billing, openmeter/customer, openmeter/entitlement, openmeter/subscription)*
- **adapter DB access in a transaction** -> entutils.TransactingRepo / TransactingRepoWithNoValue  *(scope: openmeter/billing, openmeter/customer, openmeter/entitlement, openmeter/subscription, openmeter/notification, openmeter/ledger)*
- **per-customer billing serialization** -> billing.Service.WithLock -> SELECT FOR UPDATE on BillingCustomerLock  *(scope: openmeter/billing, openmeter/billing/charges)*
- **per-charge / scoped advisory lock** -> lockr.LockForTX (pg_advisory_xact_lock) inside a TransactingRepo tx  *(scope: openmeter/billing, openmeter/billing/charges, openmeter/entitlement)*
- **async domain events between binaries** -> eventbus.Publisher prefix-routed to ingest/system/balance-worker topics  *(scope: openmeter/watermill, openmeter/billing/worker, openmeter/entitlement/balanceworker, openmeter/notification)*
- **consuming worker events** -> router.NewDefaultRouter + grouphandler.NoPublishingHandler (silent drop)  *(scope: openmeter/watermill/grouphandler, openmeter/billing/worker, openmeter/notification/consumer, openmeter/entitlement/balanceworker)*
- **batch usage ingestion** -> Sink three-phase flush (ClickHouse -> offset -> Redis dedupe)  *(scope: openmeter/sink, cmd/sink-worker)*
- **edge ingest with dedup** -> ingest.Collector wrapped by DeduplicatingCollector  *(scope: openmeter/ingest, openmeter/sink)*
- **lifecycle side-effects across domains (post-mutation)** -> ServiceHookRegistry[T] registered in app/common  *(scope: openmeter/customer, openmeter/subscription, openmeter/app, openmeter/ledger, app/common)*
- **pre-mutation cross-domain validation** -> Customer RequestValidator registry  *(scope: openmeter/customer, openmeter/billing, openmeter/entitlement, app/common)*
- **billing tagged-union construction** -> NewCharge[T]/NewChargeIntent[T]/NewStandardInvoiceLine/NewGatheringInvoiceLine (never struct literal)  *(scope: openmeter/billing, openmeter/billing/charges)*
- **invoice/charge state transitions** -> stateless InvoiceStateMachine or Machine[CHARGE,BASE,STATUS]  *(scope: openmeter/billing, openmeter/billing/service, openmeter/billing/charges)*
- **new billing backend** -> Implement billing.InvoicingApp + AppFactory self-registration in New()  *(scope: openmeter/app, openmeter/billing)*
- **new charge line engine** -> RegisterLineEngine only in app/common/charges.go  *(scope: app/common, openmeter/billing, openmeter/billing/charges)*
- **optional feature disabled** -> Return noop implementation in Wire provider (four-layer guard for credits)  *(scope: app/common)*
- **HTTP handler** -> httptransport.NewHandler decode/operate/encode + GenericErrorEncoder  *(scope: openmeter/billing/httpdriver, openmeter/customer/httpdriver, openmeter/meter/httphandler, api/v3/handlers)*
- **domain error -> HTTP status mapping** -> models.Generic* sentinel matched by GenericErrorEncoder
- **multi-tenant provisioning** -> namespace.Manager fan-out; register Handlers before CreateDefaultNamespace  *(scope: openmeter/namespace, cmd/server)*
- **DI wiring a binary** -> wire.NewSet in app/common/, wire.Build in cmd/<binary>/wire.go  *(scope: app/common, cmd/server, cmd/billing-worker, cmd/balance-worker, cmd/sink-worker, cmd/notification-service, cmd/jobs)*
- **API contract change** -> TypeSpec in api/spec/ -> make gen-api -> make generate  *(scope: api/spec)*
- **subscription spec mutation** -> AppliesToSpec.ApplyTo patch interface  *(scope: openmeter/subscription)*
- **structured field-pathed validation error** -> ValidationIssue with-chain + NewNillableGenericValidationError(errors.Join(...))
- **tracing service/adapter work** -> tracex.Start/Wrap (auto error-record + panic recovery)

## Decision Chain

**Root constraint:** Operate a high-volume per-tenant usage-metering platform feeding strict financial billing correctness, while shipping stable SDKs in three languages — under a small team that cannot maintain separate repos or hand-synchronized contracts.

- **Multi-binary Go modulith: one shared domain tree, six Wire-composed binaries, Kafka-only inter-binary coupling, single TypeSpec API source**: Ingest throughput, balance recalculation, billing advancement, and webhook dispatch have incompatible scaling/failure profiles, but billing correctness needs one typed domain model — so split the processes, share the openmeter/ types, and make Kafka the sole inter-binary channel.
  - *Violation keyword:* `business logic in cmd/*/main.go`
  - *Violation keyword:* `goroutine spawned outside run.Group`
  - *Violation keyword:* `new cmd/* binary without app/common/openmeter_<binary>.go`
  - *Violation keyword:* `domain package importing app/common`
  - *Violation keyword:* `shared in-memory state between binaries`
  - *Violation keyword:* `HTTP call between binaries`
  - **Google Wire DI with all provider sets in app/common and cross-domain hooks registered as construction side-effects**: ~40 services per binary make hand-wiring error-prone; Wire gives compile-time graph verification, and keeping providers out of domain packages prevents import cycles.
    - *Violation keyword:* `wire.Build calling domain constructors directly`
    - *Violation keyword:* `provider function with validation/computation/panic/os.Exit`
    - *Violation keyword:* `domain package importing app/common`
    - *Violation keyword:* `viper.SetDefault in cmd/*`
    - **credits.enabled feature flag enforced at four independent wiring layers via noop implementations**: Credits writes fan out from HTTP handlers, customer hooks, namespace provisioning, and charge creation — no single choke point — so each wiring layer must guard independently and return a noop, not nil.
      - *Violation keyword:* `ledger-touching Wire provider without creditsConfig.Enabled branch`
      - *Violation keyword:* `BillingRegistry.Charges accessed without ChargesServiceOrNil()`
      - *Violation keyword:* `nil returned instead of a noop struct`
      - *Violation keyword:* `v3 credit handler registered without s.Credits.Enabled check`
  - **Kafka + Watermill async backbone with three name-prefix-routed topics and silent-drop consumer dispatch**: Independently deployable workers need durable, replayable, backpressure-aware async delivery and topic isolation so ingest bursts cannot starve billing consumers.
    - *Violation keyword:* `kafka.NewProducer in domain code`
    - *Violation keyword:* `confluent ProduceChannel`
    - *Violation keyword:* `sarama SendMessage`
    - *Violation keyword:* `publishing to a topic by string literal`
    - *Violation keyword:* `EventName() without an EventVersionSubsystem prefix`
    - *Violation keyword:* `context.Background() inside a Watermill handler instead of msg.Context()`
    - *Violation keyword:* `fourth Kafka topic without updating TopicMapping`
    - **Sink worker exactly-once via strict three-phase flush; ingest dedup upstream of an un-deduplicated ClickHouse MergeTree**: Exactly-once ingestion needs ClickHouse written before the Kafka offset commits, and because the MergeTree does not deduplicate, Redis dedupe must be the last phase.
      - *Violation keyword:* `Redis dedupe set before Kafka offset commit`
      - *Violation keyword:* `Kafka offset committed before ClickHouse BatchInsert`
      - *Violation keyword:* `OnFlushSuccess called synchronously`
      - *Violation keyword:* `ReplacingMergeTree dedup in ClickHouse write path`
      - *Violation keyword:* `ingest events written directly to ClickHouse bypassing ingest.Collector`
    - **Namespace Manager fan-out provisioning with cmd/server as the sole handler registrant**: Multi-tenancy provisioning must fan out across ClickHouse/Kafka/Ledger with no single owning store; cmd/server registers all handlers before initNamespace, and workers self-provision only what they own.
      - *Violation keyword:* `namespace.Handler registered after CreateDefaultNamespace`
      - *Violation keyword:* `worker assuming the default namespace exists without a fail-fast check`
      - *Violation keyword:* `RegisterHandler called in a worker binary instead of cmd/server`
- **TypeSpec as the single source of truth for both v1 and v3 HTTP APIs and all three SDKs**: Three SDK languages and two API versions cannot be hand-synchronized; a single upstream TypeSpec contract makes drift structurally impossible.
  - *Violation keyword:* `hand-edited api/openapi.yaml`
  - *Violation keyword:* `hand-edited api/v3/openapi.yaml`
  - *Violation keyword:* `endpoint added only in a Go handler package`
  - *Violation keyword:* `hand-edited *.gen.go`
  - *Violation keyword:* `@route in a domain sub-folder tsp instead of root openmeter.tsp`
  - *Violation keyword:* `TypeSpec edit without make gen-api && make generate`
  - **httptransport decode/operate/encode pipeline with dual request validation and GenericErrorEncoder chain**: Dual API versions need dual validation middleware (kin-openapi for v1, oasmiddleware for v3); the generic pipeline keeps RFC 7807 error mapping and OTel uniform across both.
    - *Violation keyword:* `handler implementing ServeHTTP directly`
    - *Violation keyword:* `http status codes written in handler logic instead of models.Generic* sentinels`
    - *Violation keyword:* `chi.NewRouter without a request validator`
    - *Violation keyword:* `v3 handler placed in openmeter/*/httpdriver`
- **Ent ORM + Atlas migrations with context-propagated transactions (entutils.TransactingRepo) and per-customer pg locks**: Billing correctness needs compile-time-checked relations across ~35 entities, deterministic reviewable migrations, atomic multi-step mutation, and per-customer serialization against concurrent workers.
  - *Violation keyword:* `edits inside openmeter/ent/db/`
  - *Violation keyword:* `hand-written SQL alongside Ent queries`
  - *Violation keyword:* `*entdb.Tx as a struct field`
  - *Violation keyword:* `a.db.Foo() in an adapter without TransactingRepo`
  - *Violation keyword:* `LockForTX outside an active transaction`
  - *Violation keyword:* `manual edits to tools/migrate/migrations/ or atlas.sum`
  - *Violation keyword:* `context.WithTimeout around LockForTX`
  - **Tagged-union billing models (Charge, ChargeIntent, InvoiceLine) with private discriminators and constructor-only construction, driven by stateless state machines + a LineEngine registry**: Multi-step charge advancement mixing reads, realization, locks, and ledger writes needs exhaustive unambiguous type dispatch and impossible partial construction.
    - *Violation keyword:* `charges.Charge{} struct literal`
    - *Violation keyword:* `charges.ChargeIntent{} struct literal`
    - *Violation keyword:* `billing.InvoiceLine{} struct literal`
    - *Violation keyword:* `direct Invoice.Status field mutation`
    - *Violation keyword:* `RegisterLineEngine called from a domain package or cmd/*`
  - **Double-entry ledger with FK-less cross-aggregate links and template-only transaction construction (credits-gated)**: Double-entry financial correctness requires the debit=credit invariant enforced at one seam (ResolveTransactions), and the layered import-cycle-avoidance forbids FK edges that would couple ledger to customer/account aggregates.
    - *Violation keyword:* `hand-constructed ledger.Entry{} outside ResolveTransactions`
    - *Violation keyword:* `edge.To/edge.From added to LedgerCustomerAccount`
    - *Violation keyword:* `ledger write without a creditsConfig.Enabled guard`
    - *Violation keyword:* `relying on a DB FK for ledger referential integrity`

## Key Decisions

### Google Wire DI with all provider sets in app/common and cross-domain hooks registered as construction side-effects
**Chosen:** Each cmd/<binary>/wire.go declares wire.Build over composite provider sets defined in app/common/ (per-domain files plus openmeter_<binary>.go). Domain packages under openmeter/ expose plain constructors and never import app/common (one-way outward import direction). Cross-domain ServiceHooks and customer RequestValidators are registered inside app/common provider functions as side-effects of construction (e.g. customerService.RegisterHooks at app/common/customer.go, customerService.RegisterRequestValidator and billing.Service.RegisterLineEngine at app/common/charges.go), invisible to Wire's type graph.
**Rationale:** Wire produces a compile-time-checked dependency graph per binary so missing providers are build errors, not runtime panics (cmd/billing-worker/wire.go). Concentrating providers in app/common keeps the ~35 domain packages as leaf nodes with no DI-layer dependency, avoiding the import cycles that would otherwise form between billing, customer, subscription, and ledger. Registering hooks as side-effects in app/common (pkg/models/servicehook.go ServiceHookRegistry, openmeter/customer/requestvalidator.go) lets billing react to customer lifecycle without billing and customer importing each other. The dep-001 enforcement rule forbids a domain package importing app/common precisely to preserve this direction.
**Rejected:** Manual constructor calls in each cmd/main.go — ~40 services per binary make hand-wiring error-prone and unverifiable., Reflection-based runtime DI — loses Wire's compile-time graph verification., Domain packages registering their own hooks — creates circular imports between billing, customer, subscription, ledger., Provider functions containing business logic — rejected; the wire-002 rule blocks panic/log.Fatal/os.Exit in app/common providers, which must only construct and wire.
**Forced by:** Multi-binary modulith with ~40 services per binary and very different per-binary provider graphs, plus the need for cross-domain hooks without circular imports.
**Enables:** Compile-time proof of binary completeness; import-cycle-free leaf domain packages; cross-domain lifecycle reactions; per-binary composition; uniform noop-for-disabled-feature gating.

### credits.enabled feature flag enforced at four independent wiring layers via noop implementations
**Chosen:** When config.Credits.Enabled is false: app/common/ledger.go returns ledgernoop.AccountService{}/Ledger{}/AccountResolver{} from each provider (ledger.go:74-114); app/common/customer.go NewCustomerLedgerServiceHook returns NoopCustomerLedgerHook{} (customer.go:61-62); app/common/billing.go NewBillingRegistry skips the ChargesRegistry entirely (BillingRegistry.Charges stays nil, accessed only via ChargesServiceOrNil()); and api/v3/server credit/ledger handlers skip registration. NewLedgerNamespaceHandler additionally type-asserts against the noop AccountResolver (ledger.go:133-134).
**Rationale:** Credits cross-cut ledger writes, customer lifecycle hooks, namespace default-account provisioning, charge creation in billing/charges, and v3 HTTP handlers — there is no single choke point, because a customer creation in api/v3 fans out through independent call graphs. The noop-for-disabled-features pattern requires each wiring layer to guard independently and return a noop interface rather than nil so callers never branch on nil. Wire verifies types, not the policy that every ledger-writer provider has a creditsConfig.Enabled branch, so the guard set is only as complete as the most recently added provider's author remembered.
**Rejected:** Single global runtime flag check inside ledger.Ledger — ledger writes are initiated from three independent call graphs, so one check cannot gate all paths., Top-level HTTP middleware blocking credits endpoints — does not stop ledger writes triggered by customer hooks or namespace provisioning., Returning nil instead of a noop struct — callers receive the interface and panic on nil (the di-001 rule).
**Forced by:** The cross-cutting nature of credit accounting and the customer/billing/ledger hook fan-out across unrelated call graphs.
**Enables:** Credits-disabled tenants produce zero ledger_accounts/ledger_customer_accounts rows; per-deployment enabling without rebuild; compile-time interface satisfaction for noop implementations.

### Kafka + Watermill async backbone with three name-prefix-routed topics and silent-drop consumer dispatch
**Chosen:** eventbus.New wraps Watermill cqrs.NewEventBusWithConfig; GeneratePublishTopic (openmeter/watermill/eventbus/eventbus.go:137-142) does strings.HasPrefix on EventName against ingestevents.EventVersionSubsystem and balanceworkerevents.EventVersionSubsystem, routing to IngestEventsTopic / BalanceWorkerEventsTopic and DEFAULTING everything else to SystemEventsTopic. Consumers build routers via router.NewDefaultRouter (fixed PoisonQueue/DLQ/CorrelationID/Recoverer/Retry/ProcessingTimeout/HandlerMetrics stack) and dispatch via grouphandler.NoPublishingHandler keyed on the CloudEvents ce_type, silently ACKing unknown types for rolling-deploy safety.
**Rationale:** Independently deployable workers need durable, replayable, backpressure-aware async delivery and topic isolation so ingest bursts cannot starve billing consumers. The prefix-routing default-to-SystemEventsTopic (eventbus.go:142) lets genuine system events need no explicit declaration. grouphandler.go:49-55 returns nil for any ce_type not in typeHandlerMap so producer/consumer version skew during rolling deploys does not poison the DLQ. The wm-001 rule forbids publishing by string literal; routing is always by EventName() prefix through eventbus.Publisher.
**Rejected:** kafka.NewProducer / confluent ProduceChannel / sarama SendMessage in domain code — bypasses prefix routing and topic isolation (wm-001)., Returning an error for an unknown ce_type — poisons the DLQ for valid messages of other families on the same topic during rolling deploys., A single shared topic — ingest bursts would starve billing system-event consumers; the three-topic split matches worker topology.
**Forced by:** The multi-binary decision: independently deployable workers with incompatible scaling profiles need durable async delivery with topic isolation.
**Enables:** Topic isolation between ingest, system, and balance-worker queues; rolling-deploy tolerance via silent-drop dispatch; replayable cross-binary events; uniform DLQ/retry/OTel middleware.

### Ent ORM + Atlas migrations with context-propagated transactions (entutils.TransactingRepo) and per-customer pg locks
**Chosen:** openmeter/ent/schema/ holds ~35 Go-defined entity schemas (each with IDMixin + NamespaceMixin + TimeMixin); make generate regenerates openmeter/ent/db/; atlas migrate --env local diff produces timestamped .up.sql/.down.sql plus an atlas.sum hash chain. Every domain adapter implements the Tx/WithTx/Self triad and wraps every method body in entutils.TransactingRepo / TransactingRepoWithNoValue (pkg/framework/entutils/transaction.go:208-220), which rebinds to any ctx-bound transaction (line 220) or falls back to repo.Self() (line 216). Per-customer serialization uses a SELECT FOR UPDATE on the BillingCustomerLock row (UNIQUE (namespace, customer_id), openmeter/ent/schema/billing.go:1356) plus generic pg_advisory_xact_lock via pkg/framework/lockr.
**Rationale:** Billing correctness needs compile-time-checked relations across ~35 entities, deterministic reviewable migrations, atomic multi-step charge/invoice mutation, and per-customer serialization against concurrent workers. TransactingRepo reads the *TxDriver from ctx and rebinds, supporting savepoint nesting for multi-step flows like charges AdvanceCharges/ApplyPatches. The lock invariant is grounded: billing.go:1356 declares UNIQUE (namespace, customer_id) on BillingCustomerLock so the lock serializes exactly one customer. The graceful fallback to Self() at transaction.go:216 means a helper that bypasses TransactingRepo silently falls off the transaction with no error — the ctx-001/ctx-002 rules forbid raw *entdb.Tx fields and unwrapped a.db.Foo() in adapter helpers.
**Rejected:** Raw golang-migrate only (no typed entities) — loses compile-checked relations across ~35 entities., GORM — weaker typing and no native Atlas-style schema diff., Explicit *entdb.Tx threaded through every signature — ctx-propagation via TransactingRepo avoids signature churn and supports savepoint nesting., Hand-written SQL alongside Ent — breaks Atlas's single-schema-source diffing (ent-001).
**Forced by:** Billing correctness plus multi-tenant schema invariants requiring compile-time-checked relations and ctx-propagated transaction reuse with savepoints.
**Enables:** Deterministic reviewable SQL migrations with atlas.sum integrity; typed relations across all entities; atomic charge advancement and invoice mutation; per-customer advisory locking.

### TypeSpec as the single source of truth for both v1 and v3 HTTP APIs and all three SDKs
**Chosen:** Endpoints are authored only in TypeSpec under api/spec/packages/legacy (v1) and api/spec/packages/aip (v3), with route/tag bindings confined to the root openmeter.tsp. make gen-api compiles to api/openapi.yaml + api/v3/openapi.yaml then oapi-codegen emits api/api.gen.go, api/v3/api.gen.go, api/client/go/client.gen.go, plus the JS and Python SDKs; make generate then propagates to Ent/Wire/Goverter/Goderive. Handlers implement the generated ServerInterface in openmeter/<domain>/httpdriver (v1) or api/v3/handlers/<resource> (v3) via the httptransport decode/operate/encode pipeline.
**Rationale:** Three SDK languages and two API versions cannot be hand-synchronized. The single TypeSpec source makes drift structurally impossible as long as both regen steps run — a TypeSpec change forces handler-side compile errors. oapi-codegen v2.6.1 (pinned pseudo-version), kin-openapi v0.139.0 (v1 validation) and oasmiddleware v1.1.2 (v3 validation) all validate against the same generated spec. The api-002/gen-002 rules forbid hand-editing the generated .yaml and *.gen.go files (DO NOT EDIT headers).
**Rejected:** Hand-written OpenAPI YAML — the YAML files carry generated headers and are overwritten by make gen-api., Code-first OpenAPI from Go handlers — would not produce the JS/Python SDKs from one source., Skipping v3 — the AIP-style v3 surface coexists with legacy v1 and both regenerate from the same compiler.
**Forced by:** Multi-language SDK requirement (Go/JS/Python) plus dual API versions plus runtime request validation against the same artifact.
**Enables:** Cross-language SDK contracts that cannot drift; kin-openapi (v1) + oasmiddleware (v3) request validation against the same spec; breaking-change detection at TypeSpec compile time.

### Tagged-union billing models (Charge, ChargeIntent, InvoiceLine) with private discriminators and constructor-only construction, driven by stateless state machines + a LineEngine registry
**Chosen:** openmeter/billing/charges/charge.go declares Charge/ChargeIntent with a private discriminator t meta.ChargeType plus three nullable sub-type pointers (charge.go:20-26), set only by the generic NewCharge[T]/NewChargeIntent[T] (charge.go:32,276) constrained to flatfee/usagebased/creditpurchase; typed accessors AsFlatFeeCharge/AsUsageBasedCharge/AsCreditPurchaseCharge (charge.go:79/91/103) error on mismatch. billing.InvoiceLine uses an analogous discriminator set only by NewStandardInvoiceLine/NewGatheringInvoiceLine. StandardInvoice lifecycle runs through a *stateless.StateMachine pooled in sync.Pool (openmeter/billing/service/stdinvoicestate.go:24-77); the generic charges Machine[CHARGE,BASE,STATUS] mirrors it with value-copy WithStatus/WithBase. Each charge type registers a LineEngine via billing.Service.RegisterLineEngine exclusively in app/common/charges.go.
**Rationale:** Multi-step charge advancement mixes reads, realization runs, advisory locks, and ledger-bound writes, so it needs exhaustive unambiguous charge-type dispatch and impossible partial construction — a struct literal leaves the discriminator zero-valued and accessors error (charge.go:21,32). qmuntal/stateless v1.8.0 drives lifecycle so post-transition actions (DB save, event publish) fire atomically; direct Invoice.Status mutation is forbidden (billing-002). The LineEngine registry (registered only in app/common, never from domain packages — dep-006) lets new charge types plug in without editing billing core.
**Rejected:** Charge as a Go interface — loses exhaustive compile-time dispatch., Public discriminator field — allows partial/inconsistent construction via struct literals., Hardcoding charge-type branches in billing.Service — the LineEngine registry plus App Factory keep the core decoupled., Mutating Invoice.Status directly — the stateless state machine enforces valid transitions and fires post-transition actions atomically.
**Forced by:** Multi-step charge advancement requiring exhaustive, unambiguous charge-type dispatch and atomic state transitions.
**Enables:** Deterministic atomic charge advancement; exhaustive charge-type and invoice-line-type dispatch; runtime-pluggable charge engines via RegisterLineEngine; per-charge advisory locking via charges.NewLockKeyForCharge.

### Sink worker exactly-once via strict three-phase flush; ingest dedup upstream of an un-deduplicated ClickHouse MergeTree
**Chosen:** openmeter/sink/sink.go flush() executes strictly: (1) persistToStorage -> Storage.BatchInsert into the shared ClickHouse MergeTree events table (sink.go:330,464); (2) Consumer.StoreMessage per Kafka offset, largest stored last (sink.go:344); (3) Redis SET NX dedupe with retry, only when a Deduplicator is configured (sink.go:354); then fires FlushEventHandler.OnFlushSuccess in a goroutine bounded by FlushSuccessTimeout (sink.go:391-395). The RawEvent table is ENGINE=MergeTree with no engine-level dedup — deduplication is entirely upstream in Redis (openmeter/dedupe/redisdedupe) keyed namespace-source-id with TTL.
**Rationale:** Exactly-once usage ingestion requires ClickHouse written before the Kafka offset commits — on consumer restart an uncommitted offset re-delivers messages not yet in ClickHouse (sink.go:330-344). Because the MergeTree does not deduplicate, Redis dedupe being phase 3 (strictly after offset commit) is load-bearing; reversing it would mark events processed while ClickHouse re-reads from the uncommitted offset, dropping them. OnFlushSuccess always runs in a goroutine so the post-flush balance-recalc notification never blocks the consumer loop (sink-002).
**Rejected:** ReplacingMergeTree / engine-level dedup in ClickHouse — dedup is pushed to the ingest edge (Redis SET NX) so the hot analytics path stays append-only., Committing the Kafka offset before the ClickHouse insert — would lose events on crash between commit and insert., Setting Redis dedupe before the offset commit — breaks exactly-once on restart (dedupe-001)., Calling OnFlushSuccess synchronously — blocks the main sink loop and causes Kafka partition backpressure.
**Forced by:** High-throughput usage ingestion needing exactly-once semantics on top of an append-only analytics store that does not deduplicate.
**Enables:** Exactly-once event delivery to ClickHouse across consumer restarts; append-only hot path; decoupled post-flush balance-recalculation via Kafka.

### Double-entry ledger with FK-less cross-aggregate links and template-only transaction construction (credits-gated)
**Chosen:** openmeter/ledger persists LedgerAccount, LedgerSubAccountRoute, LedgerCustomerAccount, LedgerTransaction, and LedgerEntry. Transaction inputs are constructed exclusively via transactions.ResolveTransactions with typed template structs that enforce debit=credit, then committed with ledger.CommitGroup (ledger-001). LedgerCustomerAccount.account_id/customer_id are FK-less Immutable strings (Edges() returns nil) to avoid import cycles to LedgerAccount/Customer; LedgerSubAccountRoute denormalizes routing dimensions (currency, tax_code as TaxCode.Key, tax_behavior, features, cost_basis) as plain literal columns with no FK. LedgerEntry is idempotent via UNIQUE(transaction_id, sub_account_id, identity_key). All ledger tables are written only when credits.enabled=true; noop/ provides zero-value implementations otherwise.
**Rationale:** Double-entry financial balances require the debit=credit invariant enforced in one place — transactions.ResolveTransactions with typed templates is the single seam, so hand-constructing ledger.Entry{} bypasses balance checks (ledger-001). The layered import-cycle-avoidance keeps the link tables FK-less (LedgerCustomerAccount.Edges() returns nil), pushing referential integrity to application code. The denormalized LedgerSubAccountRoute columns avoid joins to canonical tables at posting time at the cost of drift discipline. This is the data-architecture counterpart to the credits four-layer guard: the ledger is the write target those guards protect.
**Rejected:** FK-backed edges from LedgerCustomerAccount to LedgerAccount/Customer — would create import cycles between ledger, customer, and account aggregates; intentionally avoided (Edges() returns nil)., Hand-constructed ledger entries at call sites — bypasses the debit=credit invariant that ResolveTransactions centralizes., Normalized routing with FKs to tax_code/feature tables — adds joins on the posting hot path; the schema denormalizes routing dimensions instead and accepts dual-write drift risk.
**Forced by:** Double-entry financial correctness plus the layered import-cycle-avoidance decision (which forbids FK edges that would couple ledger to customer/account aggregates).
**Enables:** Single-seam debit=credit enforcement; idempotent postings via the (transaction_id, sub_account_id, identity_key) unique key; import-cycle-free ledger package; join-free routing resolution at posting time.

### Namespace Manager fan-out provisioning with cmd/server as the sole handler registrant
**Chosen:** namespace.Manager holds a slice of registered Handler implementations (ClickHouse streaming, Kafka ingest, Ledger). createNamespace and DeleteNamespace iterate every handler and aggregate failures with errors.Join (no short-circuit) — namespace.go:105-118,135. RegisterHandler appends handlers (namespace.go:76); CreateDefaultNamespace fans out only over handlers present at call time (namespace.go:92-93). cmd/server is the only binary that registers the Ledger/KafkaIngest handlers before initNamespace; worker binaries perform namespace-scoped provisioning inline assuming the default namespace already exists. The default namespace is protected from deletion (namespace.go:68).
**Rationale:** Multi-tenancy provisioning must fan out across heterogeneous subsystems (ClickHouse table, Kafka topic, ledger accounts) with no single owning store. errors.Join with no short-circuit means partial provisioning does not block startup. The namespace-001 rule requires all handlers registered before initNamespace because the fan-out iterates only handlers present at call time. Concentrating registration in cmd/server (the sole API binary) keeps workers as self-provisioners of only what they own, at the cost of an unenforced cmd/server-before-workers boot-order contract.
**Rejected:** Registering handlers in every binary — duplicates provisioning responsibility and risks divergent handler sets per binary; only cmd/server registers them., Short-circuiting on the first handler failure — would leave some subsystems provisioned and others not with no aggregated error; errors.Join reports all failures., A dedicated provisioning service binary — the modulith folds provisioning into cmd/server startup instead.
**Forced by:** Multi-tenancy across heterogeneous subsystems with no single owning store, plus the multi-binary decision that each binary self-provisions only what it owns.
**Enables:** Per-namespace fan-out across ClickHouse/Kafka/Ledger; aggregated (non-short-circuiting) provisioning errors; default-namespace deletion protection.

## Trade-offs Accepted

- **Accepted:** Ent-generated query friction: a large generated openmeter/ent/db/ tree, slower compile times, and the boilerplate Tx/WithTx/Self triad plus a TransactingRepo wrapper on every adapter method body.
  - *Benefit:* Compile-time-checked relations across ~35 entities, automatic Atlas schema diffing into reviewable SQL, and ctx-propagated transactions with savepoint nesting for atomic multi-step charge/invoice flows.
  - *Caused by:* Ent ORM + Atlas migrations with context-propagated transactions (entutils.TransactingRepo) and per-customer pg locks
  - *Violation signal:* db.ExecContext / db.QueryContext raw SQL added alongside Ent queries in an adapter
  - *Violation signal:* direct edits inside openmeter/ent/db/
  - *Violation signal:* a new table without a corresponding openmeter/ent/schema/*.go file
  - *Violation signal:* an adapter storing *entdb.Tx as a struct field instead of using TransactingRepo
  - *Violation signal:* an adapter method body calling a.db.Foo() without entutils.TransactingRepo / TransactingRepoWithNoValue
- **Accepted:** Multi-binary orchestration cost: six Docker image variants, Helm values complexity, and a separate Wire graph per binary that must each stay complete.
  - *Benefit:* Independent horizontal scaling of sink-worker / balance-worker / billing-worker, fault isolation per binary, and isolated deploy cadence.
  - *Caused by:* Multi-binary Go modulith: one shared domain tree, six Wire-composed binaries, Kafka-only inter-binary coupling, single TypeSpec API source
  - *Violation signal:* business logic added inside cmd/*/main.go beyond startup orchestration
  - *Violation signal:* a new cmd/* worker binary without a matching app/common/openmeter_<binary>.go Wire set
  - *Violation signal:* cross-binary dependencies introduced via shared in-memory state or HTTP calls instead of a Kafka topic
  - *Violation signal:* a goroutine spawned outside the oklog/run.Group
  - *Violation signal:* kafka.NewProducer / confluent ProduceChannel / sarama SendMessage in domain code instead of eventbus.Publisher
- **Accepted:** Two-step regeneration cadence: TypeSpec changes require both make gen-api AND make generate, and five generators (oapi-codegen, Ent, Wire, Goverter, Goderive) write different artifacts that must all stay in sync.
  - *Benefit:* Cross-language SDK contracts cannot drift — Go server stubs, Go SDK, JS SDK, Python SDK all originate from one TypeSpec source.
  - *Caused by:* TypeSpec as the single source of truth for both v1 and v3 HTTP APIs and all three SDKs
  - *Violation signal:* hand-edits inside *.gen.go, wire_gen.go, api/api.gen.go, or api/v3/api.gen.go
  - *Violation signal:* PRs touching api/spec/ without regenerated api/openapi.yaml
  - *Violation signal:* a new endpoint added only in a Go handler package without a TypeSpec source change
  - *Violation signal:* @route declared in a domain sub-folder tsp instead of the root openmeter.tsp
  - *Violation signal:* hand-edited api/openapi.yaml or api/v3/openapi.yaml
- **Accepted:** Cross-domain wiring and event routing are invisible to the compiler: hook/validator registration and credits guards are side-effects scattered across app/common, and Kafka topic routing depends on event-name string prefixes that default to SystemEventsTopic.
  - *Benefit:* Domain packages stay import-cycle-free leaves; optional features are gated without nil-checks in business logic; the three-topic topology is hidden from producers.
  - *Caused by:* Google Wire DI with all provider sets in app/common and cross-domain hooks registered as construction side-effects
  - *Violation signal:* a binary's wire.Build omitting a hook provider so the hook silently never registers
  - *Violation signal:* a new ledger-touching Wire provider added without a creditsConfig.Enabled branch
  - *Violation signal:* direct access to BillingRegistry.Charges without ChargesServiceOrNil()
  - *Violation signal:* a new event family whose EventName() lacks a recognized EventVersionSubsystem prefix (silently routes to SystemEventsTopic)
  - *Violation signal:* a fourth Kafka topic added without updating GeneratePublishTopic in eventbus.go
- **Accepted:** Sequential timestamped Atlas migration filenames plus a linear atlas.sum hash chain that, by construction, produces merge conflicts between any two branches that both append migrations.
  - *Benefit:* Deterministic, reviewable, linearly-ordered SQL migration history with cryptographic chain integrity verified by CI (make migrate-check).
  - *Caused by:* Ent ORM + Atlas migrations with context-propagated transactions (entutils.TransactingRepo) and per-customer pg locks
  - *Violation signal:* two branches producing same-timestamp migration files
  - *Violation signal:* atlas.sum merge conflicts on a long-running branch
  - *Violation signal:* manual edits to an already-landed migration file in tools/migrate/migrations/
  - *Violation signal:* committing .up.sql/.down.sql without an updated atlas.sum
- **Accepted:** Exactly-once ingestion depends on a hand-ordered three-phase flush and an upstream Redis dedupe rather than engine-level deduplication, so any reordering or skipped dedupe phase silently drops or double-counts events.
  - *Benefit:* An append-only ClickHouse MergeTree hot path (no ReplacingMergeTree merge cost) with exactly-once delivery preserved across consumer restarts and a non-blocking post-flush notification path.
  - *Caused by:* Sink worker exactly-once via strict three-phase flush; ingest dedup upstream of an un-deduplicated ClickHouse MergeTree
  - *Violation signal:* Redis dedupe set before Kafka offset commit
  - *Violation signal:* Kafka offset committed before ClickHouse BatchInsert
  - *Violation signal:* OnFlushSuccess called synchronously inside flush()
  - *Violation signal:* ReplacingMergeTree dedup in the ClickHouse write path
  - *Violation signal:* ingest events written directly to ClickHouse bypassing ingest.Collector
- **Accepted:** FK-less cross-aggregate links (LedgerCustomerAccount.account_id/customer_id, LedgerSubAccountRoute denormalized routing columns, ClickHouse RawEvent struct vs DDL) push referential integrity and column/struct alignment onto application code with no database-level guard.
  - *Benefit:* Import-cycle-free ledger package, join-free routing resolution on the posting hot path, and a migration-less create-if-not-exists ClickHouse table that needs no Atlas pipeline.
  - *Caused by:* Double-entry ledger with FK-less cross-aggregate links and template-only transaction construction (credits-gated)
  - *Violation signal:* adding edge.To/edge.From to LedgerCustomerAccount (re-introducing an import cycle)
  - *Violation signal:* a new ch:-tagged field on streaming.RawEvent without an ALTER TABLE migration for existing deployments
  - *Violation signal:* relying on a database FK to catch a dangling ledger account_id/customer_id
  - *Violation signal:* a tax_code/feature value read from LedgerSubAccountRoute assumed to match the canonical table without a reconcile check

## Out of Scope

- Frontend UI — no React/Vue application in the repo; React appears only as an optional context export inside the generated JavaScript SDK (api/client/javascript). Out of scope by the API-as-product / SDK-generation decision.
- Tenant-level identity and auth provider — openmeter/portal scopes end-customers via JWTs (golang-jwt v5), but tenant identity is delegated to the deployment. Out of scope by the self-hosted single-namespace deployment model.
- Managed hosting control plane — config.cloud.yaml and api/openapi.cloud.yaml expose cloud hooks, but cloud orchestration logic lives separately.
- Real-time client-facing streaming queries — ClickHouse is reached only via streaming.Connector inside server-side processes; there is no client-facing streaming surface.
- Multi-region active/active replication — a single PostgreSQL primary is assumed; ClickHouse cluster topology is deployment-defined. Out of scope by the single-primary Ent persistence decision.
- Synchronous cross-binary RPC / service mesh — all inter-binary communication goes through the three Kafka topics; there is no gRPC surface between binaries. Out of scope by the Kafka-only inter-binary coupling decision.
- LLM inference — openmeter/llmcost and openmeter/cost only persist/sync model price tables and compute feature costs; there is no OpenAI/Anthropic inference SDK.
- Infrastructure provisioning as code — Helm charts only (deploy/charts/); no Terraform/CloudFormation/Pulumi.
- ClickHouse schema migrations — the RawEvent events table is created via CREATE TABLE IF NOT EXISTS at connector startup, outside the Atlas/golang-migrate pipeline; column changes on already-provisioned tables are not automated.