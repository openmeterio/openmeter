## Implementation Guidelines

### HTTP API authoring (v1 + v3) via TypeSpec + oapi-codegen + httptransport.Handler [networking]
Libraries: `TypeSpec @typespec/compiler 1.9.0`, `oapi-codegen v2.6.1 (pinned fork)`, `Chi v5.2.5`, `kin-openapi (request validation middleware)`
Pattern: Author endpoints in TypeSpec under api/spec/packages/{aip,legacy}. Run `make gen-api` to regenerate api/openapi.yaml + api/v3/openapi.yaml + api/api.gen.go + api/v3/api.gen.go + Go/JS/Python SDKs, then `make generate` for downstream Go code (Wire, Ent, Goverter, Goderive). Handler packages (openmeter/<domain>/httpdriver for v1; api/v3/handlers/<resource>/ for v3) implement the generated server interface using pkg/framework/transport/httptransport.NewHandler[Req,Resp] which separates decode → operate → encode and applies an ErrorEncoder chain (commonhttp.GenericErrorEncoder maps models.Generic* sentinel errors and ValidationIssue with WithHTTPStatusCodeAttribute to RFC 7807 problem+json).
Key files: `api/spec/packages/aip`, `api/spec/packages/legacy`, `api/openapi.yaml`, `api/v3/openapi.yaml`, `api/api.gen.go`, `api/v3/api.gen.go`, `openmeter/server/router/router.go`, `api/v3/server/server.go`, `pkg/framework/transport/httptransport/handler.go`, `pkg/framework/commonhttp`
Example: `// 1. Edit TypeSpec (api/spec/packages/aip/...) to add a new operation
// 2. make gen-api    # regenerates api/v3/openapi.yaml + Go server stubs + SDKs
// 3. make generate   # regenerates api/v3/api.gen.go + Wire/Ent/Goverter/Goderive
// 4. Implement the generated interface in api/v3/handlers/foo/handler.go:

func (h *Handler) ListFoos() http.Handler {
    return httptransport.NewHandler(
        func(ctx context.Context, r *http.Request) (ListFoosInput, error) {
            return ListFoosInput{Namespace: chi.URLParam(r, "namespace")}, nil
        },
        func(ctx context.Context, in ListFoosInput) ([]Foo, error) {
            return h.svc.List(ctx, in)
        },
        func(ctx context.Context, w http.ResponseWriter, items []Foo) error {
            return commonhttp.JSONResponseEncoder(ctx, w, items)
        },
        httptransport.AppendOptions(commonhttp.GenericErrorEncoder()),
    )
}`
- Never hand-edit *.gen.go — always regenerate via make gen-api + make generate.
- If TypeSpec adds @query and the file lacks HTTP decorators, import @typespec/http and `using TypeSpec.Http;`.
- Keep v1 changes in api/spec/packages/legacy/, v3 changes in api/spec/packages/aip/; do not mix.
- Use models.Generic* sentinels (NotFound/Validation/Conflict) in service layer; the GenericErrorEncoder maps them to correct status codes — never write status codes directly in handlers.

### Persistence layer: Ent schema + Atlas migrations + transaction-aware adapters [persistence]
Libraries: `Ent v0.14.6`, `Atlas (atlas cli)`, `golang-migrate v4.19.1`, `pgx v5.9.2`, `pkg/framework/entutils`
Pattern: Define Ent entity schemas under openmeter/ent/schema/ as the source of truth. Run `make generate` to regenerate openmeter/ent/db/. Generate migrations with `atlas migrate --env local diff <name>` which writes timestamped .up.sql/.down.sql plus atlas.sum hash chain in tools/migrate/migrations/. Adapters under openmeter/<domain>/adapter/ implement the TxCreator (Tx via *entdb.Client.HijackTx + entutils.NewTxDriver) and TxUser[T] (WithTx via entdb.NewTxClientFromRawConfig, Self) triad. Every method body wraps with entutils.TransactingRepo or TransactingRepoWithNoValue so it rebinds to the ctx-bound transaction or runs on Self() if none.
Key files: `openmeter/ent/schema/`, `openmeter/ent/db/`, `tools/migrate/migrations/`, `tools/migrate/migrations/atlas.sum`, `atlas.hcl`, `pkg/framework/entutils/transaction.go`, `pkg/framework/entutils/mixins.go`, `openmeter/billing/charges/adapter/adapter.go`
Example: `// openmeter/<domain>/adapter/adapter.go
import (
    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
    "github.com/openmeterio/openmeter/pkg/framework/entutils"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

type adapter struct{ db *entdb.Client }

func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
    txCtx, cfg, drv, err := a.db.HijackTx(ctx, &sql.TxOptions{ReadOnly: false})
    if err != nil { return nil, nil, fmt.Errorf("hijack tx: %w", err) }
    return txCtx, entutils.NewTxDriver(drv, cfg), nil
}
func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
    txDb := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())
    return &adapter{db: txDb.Client()}
}
func (a *adapter) Self() *adapter { return a }

func (a *adapter) Create(ctx context.Context, in CreateInput) (*Entity, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) {
        row, err := tx.db.Entity.Create().SetNamespace(in.Namespace).Save(ctx)
        if err != nil { return nil, err }
        return toDomain(row), nil
    })
}`
- Never edit openmeter/ent/db/ — it is fully generated.
- After schema changes: make generate, then atlas migrate --env local diff <name>.
- Helpers that accept *entdb.Client must still wrap with TransactingRepo to honor ctx tx.
- Ent views may not appear in migrate/schema.go; add explicit SQL migration if atlas reports no changes.
- Adapter struct must NEVER store *entdb.Tx — always rebind via TransactingRepo.

### Async event bus across binaries via Kafka + Watermill [state_management]
Libraries: `confluent-kafka-go v2.14.1 (librdkafka)`, `Watermill v1.5.1 + watermill-kafka/v3 v3.1.2`, `OpenTelemetry`
Pattern: openmeter/watermill/eventbus wraps Watermill's cqrs.EventBus with TopicMapping (IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic). GeneratePublishTopic routes by EventName() prefix (ingestevents.EventVersionSubsystem+'.', balanceworkerevents.EventVersionSubsystem+'.', else SystemEventsTopic). Producers call publisher.Publish or the WithContext(ctx).PublishIfNoError shortcut. Consumers (billing-worker, balance-worker, notification-service, sink-worker) build routers via openmeter/watermill/router.NewDefaultRouter (fixed middleware: PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout+RestoreContext, HandlerMetrics) and dispatch via grouphandler.NoPublishingHandler keyed on CloudEvents ce_type — unknown types silently dropped to enable rolling deploys.
Key files: `openmeter/watermill/eventbus/eventbus.go`, `openmeter/watermill/router`, `openmeter/watermill/grouphandler`, `openmeter/watermill/marshaler`, `openmeter/watermill/driver/kafka`, `openmeter/sink/sink.go`, `openmeter/entitlement/balanceworker`, `openmeter/billing/worker`, `openmeter/notification/consumer`
Example: `// Publishing a domain event from a service:
if err := publisher.Publish(ctx, &billingevents.InvoiceCreated{InvoiceID: inv.ID}); err != nil {
    return fmt.Errorf("publish invoice created: %w", err)
}

// Inline pattern using PublishIfNoError:
return publisher.WithContext(ctx).PublishIfNoError(handler.handleEvent(ctx, ev))

// Consumer side (Watermill router):
noPubHandler := grouphandler.NewNoPublishingHandler(
    marshaler,
    grouphandler.NewGroupEventHandler(func(ctx context.Context, ev *billingevents.InvoiceCreated) error {
        return svc.OnInvoiceCreated(ctx, ev)
    }),
)
router.AddNoPublisherHandler("invoice-created", topics.System, subscriber, noPubHandler)`
- Always build with -tags=dynamic so librdkafka links.
- Use eventbus.Publisher; never write to a raw Kafka topic string.
- EventName() must start with a recognised EventVersionSubsystem prefix; otherwise it silently routes to SystemEventsTopic.
- Use msg.Context() inside handlers; do not substitute context.Background().
- MaxRetries=0 in router.Options is off-by-one — use 1 for single-attempt.

### Billing/charges lifecycle (charges.Service + invoice state machine + line engines) [payments]
Libraries: `Ent + entutils.TransactingRepo`, `qmuntal/stateless v1.8.0 (state machine)`, `Goverter v1.9.3 (type conversion)`, `Goderive v0.5.1 (derived equality/diff for billing types)`, `GOBL v0.400.1 (currency + invoice format)`
Pattern: billing.Service is a composite interface (Profile, Invoice, Line, Sequence, App, Lock, LineEngine, etc.) implemented in openmeter/billing/service. Customer-mutating operations call transactionForInvoiceManipulation which UpsertCustomerLock outside the tx then LockCustomerForUpdate inside. Invoice lifecycle is driven by stateless.StateMachine (openmeter/billing/service/stdinvoicestate.go) pooled in sync.Pool with external storage bound to Invoice.Status. AdvancementStrategy switches between inline and queued (publishes AdvanceStandardInvoiceEvent for billing-worker). openmeter/billing/charges owns the Charge tagged-union (NewCharge[T] discriminator) with Service.Create, AdvanceCharges, ApplyPatches; charge engines register with billing.Service.RegisterLineEngine. Tests use BaseSuite + SubscriptionMixin + MockStreamingConnector with explicit StoredAt to model late-arriving usage.
Key files: `openmeter/billing/service.go`, `openmeter/billing/service/service.go`, `openmeter/billing/service/stdinvoicestate.go`, `openmeter/billing/adapter/adapter.go`, `openmeter/billing/charges/service.go`, `openmeter/billing/charges/adapter/adapter.go`, `openmeter/billing/worker/advance/advance.go`, `openmeter/billing/worker/subscriptionsync/service.go`, `openmeter/billing/rating/service.go`
Example: `// Drive a charge lifecycle through the service facade, never the adapter directly:
intent := charges.NewChargeIntent(flatfee.Intent{ /* ... */ })
created, err := chargeService.Create(ctx, charges.CreateInput{
    Namespace: ns,
    Intents:   charges.ChargeIntents{intent},
})
if err != nil { return err }

// Advance asynchronously by publishing the event:
evt := charges.AdvanceChargesEvent{Namespace: ns, CustomerID: cid}
return publisher.Publish(ctx, evt) // routed to system topic for billing-worker

// Locking around invoice mutation in service code:
return transactionForInvoiceManipulation(ctx, s, in.Customer, func(ctx context.Context) (T, error) {
    return s.executeTriggerOnInvoice(ctx, invoice, billing.TriggerNext)
})`
- Test charge lifecycle through charges.Service.Create / AdvanceCharges / ApplyPatches, not via low-level adapters.
- Use MockStreamingConnector with explicit StoredAt to model late-arriving usage and exercise stored-at cutoff logic.
- Use NewCharge[T] / NewChargeIntent[T] — struct-literal Charge{} leaves discriminator empty and accessors error.
- All adapter helpers in openmeter/billing/charges/.../adapter must wrap bodies in entutils.TransactingRepo even if accepting *entdb.Client.
- Per-charge advisory lock keys: charges.NewLockKeyForCharge(chargeID); never hand-construct lockr.Key strings.

### Webhook delivery via Svix with reconciliation loop [notifications]
Libraries: `svix-webhooks Go SDK v1.90.0`, `Watermill (consumer)`, `openmeter/watermill/eventbus`
Pattern: openmeter/notification manages channels, rules, events, and delivery status. notification.EventHandler runs Dispatch + Reconcile loops in cmd/server's run.Group (or independently in cmd/notification-service). The Watermill consumer in openmeter/notification/consumer subscribes to the system events topic, builds the payload, and sends through the webhook.Handler interface — concrete impls in openmeter/notification/webhook/svix/svix.go (Svix client) and a noop fallback when Svix is unconfigured. NullChannel sentinel prevents unfiltered delivery. Payload version is pinned per event family; svix application registration includes the registered event types.
Key files: `openmeter/notification/service.go`, `openmeter/notification/eventhandler.go`, `openmeter/notification/consumer/consumer.go`, `openmeter/notification/webhook/handler.go`, `openmeter/notification/webhook/svix/svix.go`, `cmd/notification-service/main.go`
Example: `// Consumer dispatches an invoice event to Svix:
func (c *Consumer) onInvoiceCreated(ctx context.Context, ev billingevents.InvoiceCreated) error {
    payload := notification.InvoicePayloadV1{ /* version pinned constant */ }
    return c.dispatcher.Dispatch(ctx, notification.Event{
        Type:    notification.TypeInvoiceCreated,
        Payload: payload,
    })
}`
- Pin payload version constants per event family and treat them as API contracts.
- Reconcile loop owns retry of failed deliveries — do not duplicate retry logic inline.
- Boot order: register notification handlers before initNamespace when the default namespace needs them.
- When SVIX is unconfigured the noop handler runs — verify in tests that this branch is exercised.

### credits.enabled feature gating across four wiring layers [state_management]
Libraries: `Google Wire v0.7.0`, `openmeter/ledger/noop`, `openmeter/ledger/resolvers`
Pattern: credits.enabled must be honored at four independent layers: (1) app/common/ledger.go wires ledger services to ledgernoop.* implementations when disabled; (2) app/common/customer.go NewCustomerLedgerServiceHook returns ledgerresolvers.NoopCustomerLedgerHook{}; (3) app/common/billing.go NewBillingRegistry skips newChargesRegistry entirely so BillingRegistry.Charges stays nil and ChargesServiceOrNil() returns nil to callers; (4) v3 server credit handlers must skip registration. Additionally, NewLedgerNamespaceHandler type-asserts against ledgernoop.AccountResolver to skip namespace handler registration. A single guard is insufficient.
Key files: `app/config/config.go`, `app/common/ledger.go`, `app/common/customer.go`, `app/common/billing.go`, `openmeter/ledger/noop`, `openmeter/ledger/resolvers`, `openmeter/ledger/account/service.go`, `openmeter/namespace/namespace.go`
Example: `// app/common/customer.go style:
func NewCustomerLedgerServiceHook(
    creditsConfig config.CreditsConfiguration,
    tracer trace.Tracer,
    accountResolver customerLedgerProvisioner,
    customerService customer.Service,
) (CustomerLedgerHook, error) {
    if !creditsConfig.Enabled {
        return ledgerresolvers.NoopCustomerLedgerHook{}, nil
    }
    h, err := ledgerresolvers.NewCustomerLedgerHook(ledgerresolvers.CustomerLedgerHookConfig{
        Service: accountResolver,
        Tracer:  tracer,
    })
    if err != nil { return nil, fmt.Errorf("create customer ledger hook: %w", err) }
    customerService.RegisterHooks(h)
    return h, nil
}`
- Any new code that touches ledger_accounts or ledger_customer_accounts must have a credits.enabled=false path that does nothing.
- When writing a backfill that genuinely needs ledger writes, build the concrete adapters directly — DI defaults are noops when credits disabled.
- Add a credits-disabled integration test that asserts no ledger table rows are produced under representative flows.
- Always use BillingRegistry.ChargesServiceOrNil() — never depend on BillingRegistry.Charges directly.

### Distributed locking via pg_advisory_xact_lock (lockr) [state_management]
Libraries: `pkg/framework/lockr`, `PostgreSQL advisory locks`
Pattern: pkg/framework/lockr/locker.go calls pg_advisory_xact_lock($1) inside the active Ent transaction. Requires an active Postgres transaction in ctx (panics/errors otherwise). billing.Service.WithLock acquires the lock per CustomerID before any invoice or charge mutation; charges use charges.NewLockKeyForCharge(chargeID) for per-charge advisory locking. Locks release automatically on transaction commit/rollback. SessionLocker (pkg/framework/lockr/session.go) is the connection-scoped variant for admin flows that need locks to outlive transactions.
Key files: `pkg/framework/lockr/locker.go`, `pkg/framework/lockr/session.go`, `openmeter/billing/charges/lock.go`, `openmeter/billing/service/service.go`
Example: `// Acquiring per-charge lock inside a transaction:
key, err := charges.NewLockKeyForCharge(chargeID)
if err != nil { return err }
return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
    if err := locker.LockForTX(ctx, key); err != nil {
        return fmt.Errorf("acquire charge lock: %w", err)
    }
    // ... perform mutations under the lock ...
    return nil
})`
- Locker.LockForTX MUST run inside an active Ent transaction; calling outside returns an error.
- Use charges.NewLockKeyForCharge / billing.WithLock helpers — do not construct lockr.Key strings inline.
- Don't mix context.WithTimeout with advisory locks — use pgdriver.WithLockTimeout instead.
- SessionLocker is not goroutine-safe under high contention; always Close() to release the dedicated connection.

### Observability: OpenTelemetry tracing, metrics, and ctx propagation [analytics]
Libraries: `OpenTelemetry otel v1.43.0`, `Prometheus client v1.23.2`, `pkg/framework/tracex`
Pattern: Every entry point (HTTP handler, Kafka consumer, Ent adapter) is instrumented. trace.Tracer is injected via Wire into service constructors. ingest.ingestadapter.WithTelemetry wraps openmeter/ingest.Collector. openmeter/watermill/router.NewDefaultRouter installs OTel middleware. tracex.Start/Wrap is preferred over tracer.Start because it records errors, sets span status, and recovers panics. ctx must be threaded from HTTP handler / Kafka consumer all the way through service+adapter; introducing context.Background() or context.TODO() to bridge missing plumbing is a project-rule violation. Tests use t.Context() rather than context.Background().
Key files: `app/common/telemetry.go`, `openmeter/ingest/ingestadapter`, `openmeter/watermill/router`, `openmeter/watermill/grouphandler`, `pkg/framework/tracex/tracex.go`
Example: `// Always thread ctx through:
func (s *svc) DoWork(ctx context.Context, id string) error {
    return tracex.Start(ctx, s.tracer, "svc.DoWork", func(span *tracex.Span[any]) (any, error) {
        return nil, span.Wrap(s.adapter.Write(span.Ctx(), id))
    }).Err()
}

// In Kafka consumer handler — use msg.Context(), never context.Background():
func (h *handler) onEvent(msg *message.Message) error {
    ctx := msg.Context()
    return svc.OnInvoiceCreated(ctx, ev)
}`
- Never substitute context.Background() to work around a missing ctx plumbing issue; fix the caller.
- In tests, use t.Context() instead of context.Background() when *testing.T is available.
- Two legitimate exceptions for context.Background(): root context at program start in main(), and post-cancel graceful shutdown (e.g. apiServer.Shutdown(context.Background()) in cmd/server/main.go after parent ctx is cancelled).
- Prefer tracex.Start/Wrap over tracer.Start to centralise error recording and panic recovery.