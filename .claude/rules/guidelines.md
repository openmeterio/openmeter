## Implementation Guidelines

### Author v1 + v3 HTTP endpoints from a single TypeSpec source compiled to Go server stubs and three SDKs [networking]
**Scope:** `api/spec`, `openmeter/server/router`, `api/v3/server`, `api/v3/handlers`, `openmeter/billing/httpdriver`, `openmeter/customer/httpdriver`, `openmeter/meter/httphandler`
Libraries: `TypeSpec @typespec/compiler 1.11.0`, `oapi-codegen v2.6.1 (pinned pseudo-version)`, `Chi router v5.2.5`, `kin-openapi v0.139.0`, `oasmiddleware v1.1.2`
Pattern: Author endpoints in TypeSpec under api/spec/packages/legacy (v1) or api/spec/packages/aip (v3) with route/tag bindings only in the root openmeter.tsp. `make gen-api` regenerates api/openapi.yaml + api/v3/openapi.yaml + api/api.gen.go + api/v3/api.gen.go + the Go/JS/Python SDKs; `make generate` then regenerates Wire/Ent/Goverter/Goderive. Implement the generated ServerInterface in openmeter/<domain>/httpdriver (v1) or api/v3/handlers/<resource> (v3) using httptransport.NewHandler, which separates decode -> operate -> encode and appends commonhttp.GenericErrorEncoder to map models.Generic* sentinels to RFC 7807 problem+json.
Key files: `api/spec/packages/aip/src/openmeter.tsp`, `api/spec/packages/legacy/src/main.tsp`, `api/api.gen.go`, `api/v3/api.gen.go`, `pkg/framework/transport/httptransport/handler.go`, `pkg/framework/commonhttp/encoder.go`
Example: `// 1. Edit TypeSpec under api/spec/packages/aip to add the operation
// 2. make gen-api    # regenerates api/v3/openapi.yaml + Go server stubs + SDKs
// 3. make generate   # regenerates api/v3/api.gen.go + Wire/Ent/Goverter
// 4. Implement the generated interface in api/v3/handlers/foo/handler.go:
func (h *Handler) ListFoos() http.Handler {
    return httptransport.NewHandler(
        func(ctx context.Context, r *http.Request) (ListFoosInput, error) {
            return ListFoosInput{Namespace: chi.URLParam(r, "namespace")}, nil
        },
        func(ctx context.Context, in ListFoosInput) ([]Foo, error) {
            return h.svc.List(ctx, in)
        },
        commonhttp.JSONResponseEncoder[[]Foo],
        httptransport.AppendOptions(commonhttp.GenericErrorEncoder()),
    )
}`
**Applicable when:** API surface authored once in TypeSpec and compiled to dual OpenAPI artifacts plus three SDKs — route/tag bindings live only in the root openmeter.tsp and both make gen-api and make generate are required to keep stubs and SDKs in sync (api/spec/packages/aip/src/openmeter.tsp).
**Do NOT apply when:**
  - Generated artifacts that are regenerated, not hand-maintained — editing api/openapi.yaml, api/v3/openapi.yaml, api/api.gen.go, or api/v3/api.gen.go is silently overwritten on the next make gen-api (api/v3/api.gen.go carries a DO NOT EDIT header).
  - Handlers implementing ServeHTTP directly — bypasses httptransport.NewHandler's GenericErrorEncoder + OTel chain (pkg/framework/transport/httptransport/handler.go).
  - Placing a v1 endpoint into api/spec/packages/aip or a v3 endpoint into api/spec/packages/legacy — the two version packages must not be mixed (api/spec/packages/legacy/src/main.tsp).
- TypeSpec files adding @query/@route must import @typespec/http and add `using TypeSpec.Http;` or compilation fails with Unknown decorator.
- Return models.Generic* sentinels from the service layer; GenericErrorEncoder maps them to the correct HTTP status — do not write status codes in handler logic.
- Keep v1 changes in api/spec/packages/legacy and v3 changes in api/spec/packages/aip; never mix them in one package.

### Persist domain data via Ent schema, Atlas migrations, and context-propagated transactions [persistence]
**Scope:** `openmeter/billing/adapter`, `openmeter/billing/charges/adapter`, `openmeter/customer/adapter`, `openmeter/notification/adapter`, `openmeter/ledger`, `openmeter/entitlement`, `openmeter/subscription`
Libraries: `Ent ORM v0.14.6`, `Atlas 0.36.0`, `golang-migrate v4.19.1`, `pgx v5.9.2`
Pattern: Define Ent entity schemas under openmeter/ent/schema (each with IDMixin + NamespaceMixin + TimeMixin). Run `make generate` to regenerate openmeter/ent/db/. Generate migrations with `atlas migrate --env local diff <name>`, committing the .up.sql/.down.sql pair plus the updated atlas.sum together. Each domain adapter implements the Tx/WithTx/Self triad (Tx via HijackTx + NewTxDriver, WithTx via NewTxClientFromRawConfig, Self) and wraps every method body in entutils.TransactingRepo / TransactingRepoWithNoValue so it rebinds to any ctx-bound transaction or runs on Self().
Key files: `openmeter/ent/schema/billing.go`, `openmeter/ent/schema/charges.go`, `tools/migrate/migrations`, `tools/migrate/migrations/atlas.sum`, `atlas.hcl`, `pkg/framework/entutils/transaction.go`, `openmeter/billing/charges/adapter/search.go`
Example: `type adapter struct{ db *entdb.Client }

func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
    txCtx, cfg, drv, err := a.db.HijackTx(ctx, &sql.TxOptions{})
    return txCtx, entutils.NewTxDriver(drv, cfg), err
}
func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
    return &adapter{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()}
}
func (a *adapter) Self() *adapter { return a }

func (a *adapter) Create(ctx context.Context, in CreateInput) (*Entity, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) {
        return toDomain(tx.db.Entity.Create().SetNamespace(in.Namespace).Save(ctx))
    })
}`
**Applicable when:** Adapters implementing the Tx/WithTx/Self triad — every method body must wrap with entutils.TransactingRepo so the ctx-bound Ent transaction is honored or it falls back to Self() (pkg/framework/entutils/transaction.go).
**Do NOT apply when:**
  - Adapter structs storing *entdb.Tx as a field instead of rebinding via TransactingRepo — the raw tx falls off the ctx transaction (ctx-001 enforcement; pkg/framework/entutils/transaction.go).
  - Adapter helpers calling a.db.Foo() directly without TransactingRepoWithNoValue — silently degrades to Self() and produces partial writes in multi-step flows (openmeter/billing/charges/adapter/search.go).
  - Hand-writing a migration in tools/migrate/migrations/ without `atlas migrate --env local diff` — corrupts atlas.sum (gen-006 enforcement).
  - Editing openmeter/ent/db/ directly — fully generated by make generate (gen-001 enforcement).
- Every new entity needs IDMixin + NamespaceMixin + TimeMixin or multi-tenancy and soft-delete break (BalanceSnapshot intentionally omits IDMixin — it has no surrogate PK).
- ent.View schemas generate query code but are not picked up by Atlas diff; view DDL may need an explicit SQL migration (see AGENTS.md Ent view caveat).
- After a schema change run make generate then atlas migrate diff, and commit schema + generated code + migration + atlas.sum together.
- When adding a column to chargemeta.Mixin, also extend chargesSearchV1Columns or the ChargesSearchV1 union view breaks; run make generate-view-sql.

### Publish and consume async domain events across binaries via Kafka + Watermill [state_management]
**Scope:** `openmeter/watermill`, `openmeter/watermill/eventbus`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification/consumer`, `openmeter/sink`, `openmeter/ingest`
Libraries: `confluent-kafka-go v2.14.1 (librdkafka)`, `Watermill v1.5.2 + watermill-kafka/v3 v3.1.2`, `IBM/sarama v1.49.0`, `OpenTelemetry v1.43.0`
Pattern: openmeter/watermill/eventbus wraps Watermill's cqrs.EventBus; GeneratePublishTopic routes by EventName() prefix to IngestEventsTopic, SystemEventsTopic, or BalanceWorkerEventsTopic. Producers call eventbus.Publisher.Publish or WithContext(ctx).PublishIfNoError. Consumers build routers via openmeter/watermill/router.NewDefaultRouter (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout, HandlerMetrics) and dispatch via grouphandler.NoPublishingHandler keyed on the CloudEvents ce_type; unknown ce_types are silently dropped (ACK) to allow rolling deploys.
Key files: `openmeter/watermill/eventbus/eventbus.go`, `openmeter/watermill/router/router.go`, `openmeter/watermill/grouphandler/grouphandler.go`, `openmeter/billing/charges/events.go`
Example: `// Publishing a domain event from a service:
if err := publisher.Publish(ctx, &billingevents.InvoiceCreated{InvoiceID: inv.ID}); err != nil {
    return fmt.Errorf("publish invoice created: %w", err)
}

// Consumer side (Watermill router):
noPubHandler := grouphandler.NewNoPublishingHandler(
    marshaler,
    grouphandler.NewGroupEventHandler(func(ctx context.Context, ev *billingevents.InvoiceCreated) error {
        return svc.OnInvoiceCreated(ctx, ev)
    }),
)
router.AddNoPublisherHandler("invoice-created", topics.System, subscriber, noPubHandler)`
**Applicable when:** Cross-binary event delivery via eventbus.Publisher where topic routing is determined by EventName() prefix and must match a registered EventVersionSubsystem constant — the default switch case routes any unrecognized prefix to SystemEventsTopic (openmeter/watermill/eventbus/eventbus.go:141-142).
**Do NOT apply when:**
  - Publishing directly to a Kafka topic by string literal or via kafka.NewProducer/sarama.SendMessage instead of eventbus.Publisher (wm-001 enforcement).
  - An ingest/balance-worker event family whose EventName() lacks the matching EventVersionSubsystem prefix — it silently routes to SystemEventsTopic, bypassing topic isolation (openmeter/watermill/eventbus/eventbus.go:141).
  - Substituting context.Background() inside a Watermill handler instead of msg.Context() — severs OTel spans and drops the Ent transaction (wm-002 enforcement).
  - Returning an error for an unknown ce_type in a NoPublishingHandler — poisons the DLQ during rolling deploys (openmeter/watermill/grouphandler/grouphandler.go:54).
- Always build and test with -tags=dynamic so confluent-kafka-go links against librdkafka.
- Build consumer routers only via router.NewDefaultRouter to inherit the fixed middleware stack.
- MaxRetries=0 means zero retries then DLQ — it is off-by-one from the intuitive name.

### Drive billing and charge lifecycle via tagged-union models, state machines, and the LineEngine registry [payments]
**Scope:** `openmeter/billing`, `openmeter/billing/service`, `openmeter/billing/charges`, `openmeter/billing/worker`, `openmeter/billing/adapter`
Libraries: `qmuntal/stateless v1.8.0`, `Goverter v1.9.3`, `Goderive v0.5.1`, `alpacadecimal v0.0.9`, `GOBL v0.403.0`
Pattern: billing.Service is a composite interface implemented in openmeter/billing/service, driving invoice lifecycle through a stateless.StateMachine pooled in sync.Pool (stdinvoicestate.go) bound to Invoice.Status. openmeter/billing/charges owns the Charge/ChargeIntent tagged-union (private meta.ChargeType discriminator) constructed only via NewCharge[T]/NewChargeIntent[T] and accessed via AsFlatFeeCharge/AsUsageBasedCharge/AsCreditPurchaseCharge. Each charge type plugs into a generic Machine[CHARGE,BASE,STATUS] and registers a LineEngine with billing.Service.RegisterLineEngine in app/common/charges.go. Customer-mutating operations acquire a per-customer lock via billing.Service.WithLock inside an active transaction.
Key files: `openmeter/billing/service.go`, `openmeter/billing/service/stdinvoicestate.go`, `openmeter/billing/charges/charge.go`, `openmeter/billing/invoiceline.go`, `app/common/charges.go`, `openmeter/billing/charges/lock.go`
Example: `// Construct a charge intent only via the constructor - never a struct literal:
intent := charges.NewChargeIntent(flatfee.Intent{ /* ... */ })
created, err := chargeService.Create(ctx, charges.CreateInput{
    Namespace: ns,
    Intents:   charges.ChargeIntents{intent},
})
if err != nil {
    return err
}

// Advance asynchronously by publishing the event (routed to the system topic):
return publisher.Publish(ctx, charges.AdvanceChargesEvent{Namespace: ns, CustomerID: cid})`
**Applicable when:** Charge / ChargeIntent / InvoiceLine carry a private discriminator set only by the constructor — a struct literal leaves it zero-valued and all typed accessors error (openmeter/billing/charges/charge.go, openmeter/billing/invoiceline.go).
**Do NOT apply when:**
  - Constructing charges.Charge{}, charges.ChargeIntent{}, or billing.InvoiceLine{} via struct literal — leaves the discriminator zero-valued (billing-003 / billing-005 / billing-008 enforcement).
  - Mutating Invoice.Status directly instead of going through the stateless state machine's FireAndActivate (billing-002 enforcement; openmeter/billing/service/stdinvoicestate.go).
  - Implementing ChargeLike WithStatus/WithBase as pointer receivers — they must return value copies because Machine.Charge is updated by assignment (charge-001 enforcement).
  - Registering a LineEngine from a domain package or cmd/* instead of app/common/charges.go (dep-006 enforcement).
- Register charge LineEngines only in app/common/charges.go, never from domain packages or cmd/*.
- Use charges.NewLockKeyForCharge(chargeID) for per-charge advisory locks; never hand-construct lockr.Key strings.
- Use MockStreamingConnector with explicit StoredAt to exercise stored-at cutoff logic in charge finalization tests.

### Compose each binary with Google Wire provider sets and register cross-domain hooks/validators as side-effects [state_management]
**Scope:** `app/common`, `cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`
Libraries: `Google Wire v0.7.0`
Pattern: Each cmd/<binary>/wire.go declares a wire.Build over composite provider sets defined in app/common/ (per-domain files plus openmeter_<binary>.go per-binary sets). Domain packages expose plain constructors and never import app/common. Cross-domain ServiceHooks and customer RequestValidators are registered inside app/common provider functions as construction side-effects to avoid circular imports. Optional features (credits.enabled=false, Svix unconfigured) are gated by returning noop implementations rather than nil.
Key files: `app/common/billing.go`, `app/common/customer.go`, `app/common/ledger.go`, `app/common/charges.go`, `cmd/billing-worker/wire.go`, `pkg/models/servicehook.go`
Example: `// app/common/customer.go - hook registration as a provider side-effect:
func NewCustomerLedgerServiceHook(
    creditsConfig config.CreditsConfiguration,
    accountResolver customerLedgerProvisioner,
    customerService customer.Service,
) (CustomerLedgerHook, error) {
    if !creditsConfig.Enabled {
        return ledgerresolvers.NoopCustomerLedgerHook{}, nil
    }
    h, err := ledgerresolvers.NewCustomerLedgerHook( /* ... */ )
    if err != nil {
        return nil, err
    }
    customerService.RegisterHooks(h) // side-effect: invisible to Wire's type graph
    return h, nil
}`
**Applicable when:** ServiceHookRegistry re-entrancy guard derives its loop key from the registry's own pointer (fmt.Sprintf('...%p', r)) — correct only while the registry is shared by pointer; copying the value defeats loop prevention (pkg/models/servicehook.go:42).
**Do NOT apply when:**
  - Registering a hook inside a domain package's own constructor instead of an app/common provider — omitting the provider from a binary's wire.Build silently drops the hook with no compile error (pf-006 enforcement; servicehook-001).
  - Provider functions containing business logic beyond construction and hook/validator registration — wire-002 enforcement blocks panic/log.Fatal/os.Exit in app/common.
  - A domain package under openmeter/ importing app/common — the import direction is one-way outward and reversing creates cycles (dep-001 enforcement).
  - Returning nil instead of a noop struct for a disabled optional feature — callers receive the interface and panic on nil (di-001 enforcement).
- Audit each binary's wire.go to confirm every required hook provider is included — an omitted hook provider compiles cleanly but silently drops the hook.
- Guard credits.enabled at all four wiring layers (ledger services, customer hooks, ChargesRegistry, v3 credit handlers).
- Group cohesive services into typed registries (BillingRegistry, AppRegistry) rather than adding individual fields to router.Config; access charges via BillingRegistry.ChargesServiceOrNil().

### Serialize per-customer billing mutations via pg locks inside an active Ent transaction [persistence]
**Scope:** `openmeter/billing`, `openmeter/billing/service`, `openmeter/billing/adapter`, `openmeter/billing/charges`, `openmeter/entitlement`
Libraries: `pkg/framework/lockr`, `PostgreSQL advisory locks + SELECT FOR UPDATE`, `pgx v5.9.2`
Pattern: billing.Service.WithLock wraps the operation in a transaction that UpsertCustomerLock (idempotent insert, OnConflict DoNothing) then LockCustomerForUpdate issues SELECT ... FOR UPDATE on the single BillingCustomerLock row keyed (namespace, customer_id). The generic equivalent, pkg/framework/lockr.LockForTX, calls pg_advisory_xact_lock with a CRC64 hash of the lock key; getTxClient verifies a real Postgres transaction is in ctx (transaction_timestamp() != statement_timestamp()) and errors otherwise. Locks release automatically on commit/rollback.
Key files: `openmeter/billing/service/lock.go`, `openmeter/billing/adapter/lock.go`, `pkg/framework/lockr/locker.go`, `openmeter/billing/charges/lock.go`, `openmeter/ent/schema/billing.go`
Example: `// LockForTX must run inside an active Ent transaction:
return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
    key, err := charges.NewLockKeyForCharge(chargeID)
    if err != nil {
        return err
    }
    if err := locker.LockForTX(ctx, key); err != nil {
        return fmt.Errorf("acquire charge lock: %w", err)
    }
    // ... perform mutations under the lock ...
    return nil
})`
**Applicable when:** The locked entity has a UNIQUE index on the lock-key columns so the (namespace, key) tuple maps to at most one row — BillingCustomerLock declares UNIQUE (namespace, customer_id) (openmeter/ent/schema/billing.go:1356), so the SELECT FOR UPDATE serializes exactly that customer.
**Do NOT apply when:**
  - Lock-key columns lacking a UNIQUE index — lockr hashes scopes to a uint64, so keying on a non-unique column (status/type) would silently serialize unrelated rows under one hash (pkg/framework/lockr/key.go).
  - Calling LockForTX outside an active Postgres transaction — locker.go:134 returns 'lockr only works in a postgres transaction' when statement_timestamp()==transaction_timestamp() (lock-002 enforcement).
  - Wrapping acquisition in context.WithTimeout — pgx cancels the connection on ctx cancel, corrupting the tx; use pgdriver.WithLockTimeout instead (pkg/framework/lockr/locker.go:91-92; lock-004 enforcement).
- Per-charge lock keys come from charges.NewLockKeyForCharge; per-customer from billing.Service.WithLock.
- SessionLocker holds a dedicated connection and is not goroutine-safe under contention — always Close() it.
- Entitlement operations modifying multiple rows for the same customer must also acquire a per-customer lock before mutating.

### Batch usage-event ingestion from Kafka into ClickHouse with exactly-once three-phase flush [persistence]
**Scope:** `openmeter/sink`, `cmd/sink-worker`, `openmeter/streaming`, `openmeter/dedupe`, `openmeter/ingest`
Libraries: `confluent-kafka-go v2.14.1`, `clickhouse-go/v2 v2.46.0`, `go-redis/v9 v9.19.0`, `huandu/go-sqlbuilder v1.41.0`
Pattern: openmeter/sink/sink.go consumes Kafka partitions via confluent-kafka-go, buffers in SinkBuffer, and flush() runs strictly: (1) persistToStorage -> Storage.BatchInsert into the shared ClickHouse MergeTree events table, (2) Consumer.StoreMessage per Kafka offset (largest stored last), (3) dedupeSet (Redis SET NX with retry) only when a Deduplicator is configured. After all three phases FlushEventHandler.OnFlushSuccess fires in a goroutine bounded by FlushSuccessTimeout. The MergeTree is not deduplicated by the engine — dedup is entirely upstream in Redis keyed namespace-source-id with TTL.
Key files: `openmeter/sink/sink.go`, `openmeter/sink/storage.go`, `openmeter/sink/flushhandler/handler.go`, `openmeter/dedupe/redisdedupe/redisdedupe.go`, `openmeter/streaming/clickhouse/event_query.go`
Example: `// Three-phase flush order is load-bearing for exactly-once:
if err := s.persistToStorage(ctx, messages); err != nil {  // 1. ClickHouse BatchInsert
    return err
}
for _, m := range sortedByOffset(messages) {
    if err := s.consumer.StoreMessage(m.KafkaMessage); err != nil {  // 2. Kafka offset commit
        return err
    }
}
if s.deduplicator != nil {
    if err := s.dedupeSet(ctx, messages); err != nil {  // 3. Redis dedupe (last)
        return err
    }
}
go s.flushHandler.OnFlushSuccess(messages)  // post-flush, never blocks the loop`
**Applicable when:** Exactly-once ingestion holds only while ClickHouse is written before the Kafka offset commits — on consumer restart an uncommitted offset re-delivers messages not yet in ClickHouse (openmeter/sink/sink.go:327-372).
**Do NOT apply when:**
  - Setting Redis dedupe before the Kafka offset commit — a crash after dedupe but before commit marks events processed while ClickHouse re-reads the uncommitted offset, dropping them (openmeter/sink/sink.go:350-372; dedupe-001 enforcement).
  - Committing the Kafka offset before the ClickHouse BatchInsert — loses events on crash between commit and insert (sink-001 enforcement).
  - Calling FlushEventHandler.OnFlushSuccess synchronously inside flush() — blocks the main sink loop and causes Kafka partition backpressure (openmeter/sink/sink.go:391-399; sink-002 enforcement).
  - Writing ingest events directly to ClickHouse from domain code instead of through ingest.Collector — skips deduplication and double-counts on retry (ingest-001 enforcement).
- The MergeTree has no engine-level dedup; correctness depends on Redis SET NX being phase 3.
- Adding a column to RawEvent requires updating the createEventsTable DDL and the INSERT column list in the same order, plus an explicit migration on existing deployments (CREATE TABLE IF NOT EXISTS won't alter).
- Use meter.ParseEvent for value/group-by extraction from CloudEvent JSON; do not re-implement JSONPath inline.

### Deliver outbound webhooks via Svix with a reconciliation loop [notifications]
**Scope:** `openmeter/notification`, `openmeter/notification/consumer`, `cmd/notification-service`
Libraries: `svix-webhooks Go SDK v1.94.0`, `Watermill v1.5.2`, `openmeter/watermill/eventbus`
Pattern: openmeter/notification manages channels, rules, events, and delivery status. notification.EventHandler runs Dispatch + Reconcile loops inside cmd/server's run.Group or independently in cmd/notification-service. The Watermill consumer in openmeter/notification/consumer subscribes to the system events topic, builds the payload, and sends through the webhook.Handler interface — concrete impl in openmeter/notification/webhook/svix and a noop fallback when Svix is unconfigured. The NullChannel sentinel prevents unfiltered delivery; payload version is pinned per event family.
Key files: `openmeter/notification/service.go`, `openmeter/notification/consumer`, `openmeter/notification/webhook/svix`, `cmd/notification-service/main.go`
Example: `// Consumer dispatches an invoice event to the webhook.Handler via NullChannel guard:
func (c *Consumer) onInvoiceCreated(ctx context.Context, ev billingevents.InvoiceCreated) error {
    return c.dispatcher.Dispatch(ctx, notification.Event{
        Type:    notification.TypeInvoiceCreated,
        Payload: notification.InvoicePayloadV1{ /* version-pinned constant */ },
    })
}`
**Applicable when:** Notification handlers that must provision the default namespace are registered before initNamespace at startup (cmd/server/main.go).
**Do NOT apply when:**
  - Dispatching directly to the Svix client (MessageCreate) bypassing the NullChannel guard in notification.Service.Dispatch (notification-002 enforcement).
  - Adding ad-hoc retry inside the notification consumer — the Reconcile loop owns retry (notification-001).
  - Shipping a new event family without pinning a payload version constant (notification-001 enforcement).
- Pin payload version constants per event family and treat them as API contracts.
- The Reconcile loop owns retry of failed deliveries — do not duplicate retry logic inline.
- When Svix is unconfigured the noop webhook.Handler runs — verify tests exercise that branch.