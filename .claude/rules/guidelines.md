## Implementation Guidelines

### Author HTTP API endpoints (v1 + v3) via TypeSpec + oapi-codegen + httptransport.Handler [networking]
Libraries: `TypeSpec @typespec/compiler 1.9.0`, `oapi-codegen v2.6.1 (pinned fork)`, `Chi v5.2.5`, `kin-openapi v0.135.0`
Pattern: Author endpoints in TypeSpec under api/spec/packages/{aip,legacy}. Run `make gen-api` to regenerate api/openapi.yaml + api/v3/openapi.yaml + api/api.gen.go + api/v3/api.gen.go + Go/JS/Python SDKs, then `make generate` for downstream Go code (Wire, Ent, Goverter, Goderive). Handler packages (openmeter/<domain>/httpdriver for v1; api/v3/handlers/<resource>/ for v3) implement the generated server interface using pkg/framework/transport/httptransport.NewHandler[Req,Resp] which separates decode -> operate -> encode and applies an ErrorEncoder chain (commonhttp.GenericErrorEncoder maps models.Generic* sentinels and ValidationIssue with WithHTTPStatusCodeAttribute to RFC 7807 problem+json).
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
**Applicable when:** TypeSpec source compiles to dual API artifacts; both `make gen-api` and `make generate` are required to keep server stubs and SDKs in sync (api/spec/packages/aip/src/openmeter.tsp:1).
**Do NOT apply when:**
  - Hand-editing api/openapi.yaml or any *.gen.go file (these are regenerated; edits will be overwritten)
  - Adding a v1 endpoint - that goes in api/spec/packages/legacy/, not aip/
  - Writing handlers that bypass httptransport.NewHandler and call ServeHTTP directly
- If TypeSpec adds @query and the file lacks HTTP decorators, import @typespec/http and add `using TypeSpec.Http;`.
- Use models.Generic* sentinels (NotFound/Validation/Conflict) in service layer; the GenericErrorEncoder maps them to correct status codes.
- Keep v1 changes in api/spec/packages/legacy/, v3 changes in api/spec/packages/aip/; do not mix.

### Persist domain data via Ent schema + Atlas migrations + transaction-aware adapters [persistence]
**Scope:** `openmeter/billing/adapter`, `openmeter/billing/charges`, `openmeter/customer`, `openmeter/entitlement`, `openmeter/subscription`, `openmeter/ledger`, `openmeter/credit`, `openmeter/notification`, `openmeter/productcatalog`, `openmeter/app`, `openmeter/ent/schema`, `tools/migrate`
Libraries: `Ent v0.14.6`, `Atlas CLI 0.36.0`, `golang-migrate v4.19.1`, `pgx v5.9.2`
Pattern: Define Ent entity schemas under openmeter/ent/schema/ as source of truth. Run `make generate` to regenerate openmeter/ent/db/. Generate migrations with `atlas migrate --env local diff <name>` which writes timestamped .up.sql/.down.sql plus atlas.sum hash chain in tools/migrate/migrations/. Adapters under openmeter/<domain>/adapter/ implement TxCreator (Tx via *entdb.Client.HijackTx + entutils.NewTxDriver) and TxUser[T] (WithTx via entdb.NewTxClientFromRawConfig, Self) triad. Every method body wraps with entutils.TransactingRepo or TransactingRepoWithNoValue so it rebinds to the ctx-bound transaction or runs on Self() if none.
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
    if err != nil {
        return nil, nil, fmt.Errorf("hijack tx: %w", err)
    }
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
        if err != nil {
            return nil, err
        }
        return toDomain(row), nil
    })
}`
**Applicable when:** Adapters implementing TxCreator + TxUser[T] triad require entutils.TransactingRepo on every method body so the ctx-bound Ent transaction is honored (openmeter/billing/charges/adapter/adapter.go:1, pkg/framework/entutils/transaction.go:1).
**Do NOT apply when:**
  - Adapter struct stores *entdb.Tx as a field (defeats ctx-propagated transaction reuse)
  - Helper accepts *entdb.Client and skips TransactingRepoWithNoValue
  - Migration is hand-written in tools/migrate/migrations/ without going through `atlas migrate --env local diff`
- Never edit openmeter/ent/db/ - it is fully generated.
- After schema changes: make generate, then atlas migrate --env local diff <name>.
- Helpers that accept *entdb.Client must still wrap with TransactingRepo to honor ctx tx.
- Ent views may not appear in migrate/schema.go; add explicit SQL migration if atlas reports no changes.

### Publish and consume async domain events across binaries via Kafka + Watermill [state_management]
**Scope:** `openmeter/watermill`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification`, `openmeter/sink`, `openmeter/ingest`
Libraries: `confluent-kafka-go v2.14.1 (librdkafka)`, `Watermill v1.5.1 + watermill-kafka/v3 v3.1.2`, `OpenTelemetry v1.43.0`
Pattern: openmeter/watermill/eventbus wraps Watermill's cqrs.EventBus with TopicMapping (IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic). GeneratePublishTopic routes by EventName() prefix. Producers call publisher.Publish or the WithContext(ctx).PublishIfNoError shortcut. Consumers (billing-worker, balance-worker, notification-service, sink-worker) build routers via openmeter/watermill/router.NewDefaultRouter (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout+RestoreContext, HandlerMetrics) and dispatch via grouphandler.NoPublishingHandler keyed on CloudEvents ce_type. Unknown types silently dropped to enable rolling deploys.
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
**Applicable when:** Cross-binary event delivery uses the eventbus.Publisher; topic routing is determined by EventName() prefix and must match a recognised EventVersionSubsystem (openmeter/watermill/eventbus/eventbus.go:1).
**Do NOT apply when:**
  - Publishing directly to a Kafka topic by string literal
  - EventName lacking a registered EventVersionSubsystem prefix (silently routes to SystemEventsTopic)
  - Substituting context.Background() inside a Watermill handler instead of msg.Context()
- Always build with -tags=dynamic so librdkafka links.
- Use eventbus.Publisher; never write to a raw Kafka topic string.
- EventName() must start with a recognised EventVersionSubsystem prefix; otherwise it silently routes to SystemEventsTopic.
- Use msg.Context() inside handlers; do not substitute context.Background().

### Drive billing/charges lifecycle via charges.Service + invoice state machine + line engines [payments]
**Scope:** `openmeter/billing`, `openmeter/billing/service`, `openmeter/billing/adapter`, `openmeter/billing/charges`, `openmeter/billing/worker`
Libraries: `Ent + entutils.TransactingRepo`, `qmuntal/stateless v1.8.0`, `Goverter v1.9.3`, `Goderive v0.5.1`, `GOBL v0.400.1`
Pattern: billing.Service is a composite interface (Profile, Invoice, Line, Sequence, App, Lock, LineEngine) implemented in openmeter/billing/service. Customer-mutating operations call transactionForInvoiceManipulation which UpsertCustomerLock outside the tx then LockCustomerForUpdate inside. Invoice lifecycle is driven by stateless.StateMachine (openmeter/billing/service/stdinvoicestate.go) pooled in sync.Pool with external storage bound to Invoice.Status. AdvancementStrategy switches between inline and queued. openmeter/billing/charges owns the Charge tagged-union (NewCharge[T] discriminator) with Service.Create, AdvanceCharges, ApplyPatches; charge engines register with billing.Service.RegisterLineEngine.
Key files: `openmeter/billing/service.go`, `openmeter/billing/service/service.go`, `openmeter/billing/service/stdinvoicestate.go`, `openmeter/billing/adapter/adapter.go`, `openmeter/billing/charges/service.go`, `openmeter/billing/charges/adapter/adapter.go`, `openmeter/billing/worker/advance/advance.go`, `openmeter/billing/worker/subscriptionsync/service.go`, `openmeter/billing/rating/service.go`
Example: `// Drive a charge lifecycle through the service facade, never the adapter directly:
intent := charges.NewChargeIntent(flatfee.Intent{ /* ... */ })
created, err := chargeService.Create(ctx, charges.CreateInput{
    Namespace: ns,
    Intents:   charges.ChargeIntents{intent},
})
if err != nil {
    return err
}

// Advance asynchronously by publishing the event:
evt := charges.AdvanceChargesEvent{Namespace: ns, CustomerID: cid}
return publisher.Publish(ctx, evt) // routed to system topic for billing-worker

// Locking around invoice mutation in service code:
return transactionForInvoiceManipulation(ctx, s, in.Customer, func(ctx context.Context) (T, error) {
    return s.executeTriggerOnInvoice(ctx, invoice, billing.TriggerNext)
})`
**Applicable when:** Charge tagged-union construction must go through NewCharge[T] / NewChargeIntent[T]; struct-literal Charge{} leaves the discriminator empty (openmeter/billing/charges/service.go:1).
**Do NOT apply when:**
  - Writing tests that call charges adapter methods directly instead of charges.Service.Create / AdvanceCharges / ApplyPatches
  - Constructing charges.Charge{} via struct literal (discriminator empty, accessors will error)
  - Calling lockr advisory locks outside an active Ent transaction
- Test charge lifecycle through charges.Service.Create / AdvanceCharges / ApplyPatches, not via low-level adapters.
- Use MockStreamingConnector with explicit StoredAt to model late-arriving usage.
- Use NewCharge[T] / NewChargeIntent[T] - struct-literal Charge{} leaves discriminator empty.
- Per-charge advisory lock keys: charges.NewLockKeyForCharge(chargeID); never hand-construct lockr.Key strings.

### Deliver outbound webhooks via Svix with reconciliation loop [notifications]
**Scope:** `openmeter/notification`, `cmd/notification-service`
Libraries: `svix-webhooks Go SDK v1.90.0`, `Watermill v1.5.1`, `openmeter/watermill/eventbus`
Pattern: openmeter/notification manages channels, rules, events, and delivery status. notification.EventHandler runs Dispatch + Reconcile loops in cmd/server's run.Group (or independently in cmd/notification-service). The Watermill consumer in openmeter/notification/consumer subscribes to the system events topic, builds the payload, and sends through the webhook.Handler interface - concrete impls in openmeter/notification/webhook/svix/svix.go (Svix client) and a noop fallback when Svix is unconfigured. NullChannel sentinel prevents unfiltered delivery. Payload version is pinned per event family.
Key files: `openmeter/notification/service.go`, `openmeter/notification/eventhandler.go`, `openmeter/notification/consumer/consumer.go`, `openmeter/notification/webhook/handler.go`, `openmeter/notification/webhook/svix/svix.go`, `cmd/notification-service/main.go`
Example: `// Consumer dispatches an invoice event to Svix:
func (c *Consumer) onInvoiceCreated(ctx context.Context, ev billingevents.InvoiceCreated) error {
    payload := notification.InvoicePayloadV1{
        // version pinned constant
    }
    return c.dispatcher.Dispatch(ctx, notification.Event{
        Type:    notification.TypeInvoiceCreated,
        Payload: payload,
    })
}`
**Applicable when:** Notification handlers must register before initNamespace when the default namespace needs them (cmd/server/main.go:1).
**Do NOT apply when:**
  - Adding ad-hoc retry inside notification consumer (Reconcile loop owns retry)
  - Skipping payload version pinning for a new event family
  - Boot-order shifts that register notification handlers after initNamespace
- Pin payload version constants per event family and treat them as API contracts.
- Reconcile loop owns retry of failed deliveries - do not duplicate retry logic inline.
- When SVIX is unconfigured the noop handler runs - verify in tests that this branch is exercised.

### Gate the credits.enabled feature flag at four wiring layers [state_management]
**Scope:** `app/common`, `openmeter/ledger`, `openmeter/customer`, `openmeter/namespace`, `api/v3/server`
Libraries: `Google Wire v0.7.0`, `openmeter/ledger/noop`, `openmeter/ledger/resolvers`
Pattern: credits.enabled must be honored at four independent layers: (1) app/common/ledger.go wires ledger services to ledgernoop.* implementations when disabled; (2) app/common/customer.go NewCustomerLedgerServiceHook returns ledgerresolvers.NoopCustomerLedgerHook{}; (3) app/common/billing.go NewBillingRegistry skips newChargesRegistry entirely; (4) v3 server credit handlers must skip registration. Additionally, NewLedgerNamespaceHandler type-asserts against ledgernoop.AccountResolver to skip namespace handler registration. A single guard is insufficient.
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
    if err != nil {
        return nil, fmt.Errorf("create customer ledger hook: %w", err)
    }
    customerService.RegisterHooks(h)
    return h, nil
}`
**Applicable when:** Any new code that touches ledger_accounts or ledger_customer_accounts must have a credits.enabled=false path that does nothing (app/common/ledger.go:1, app/common/customer.go:1).
**Do NOT apply when:**
  - Single global runtime check inside ledger.Ledger (other call graphs still attempt writes)
  - Adding an HTTP middleware at the v3 boundary as the only guard
  - Depending on BillingRegistry.Charges directly without ChargesServiceOrNil()
- When writing a backfill that genuinely needs ledger writes, build the concrete adapters directly - DI defaults are noops when credits disabled.
- Add a credits-disabled integration test that asserts no ledger table rows are produced under representative flows.
- Always use BillingRegistry.ChargesServiceOrNil() - never depend on BillingRegistry.Charges directly.

### Acquire distributed locks via pg_advisory_xact_lock (lockr) [state_management]
**Scope:** `openmeter/billing`, `openmeter/billing/charges`, `openmeter/customer`, `openmeter/entitlement`, `openmeter/subscription`
Libraries: `pkg/framework/lockr`, `PostgreSQL advisory locks`
Pattern: pkg/framework/lockr/locker.go calls pg_advisory_xact_lock($1) inside the active Ent transaction. Requires an active Postgres transaction in ctx (panics/errors otherwise). billing.Service.WithLock acquires the lock per CustomerID before any invoice or charge mutation; charges use charges.NewLockKeyForCharge(chargeID) for per-charge advisory locking. Locks release automatically on transaction commit/rollback. SessionLocker (pkg/framework/lockr/session.go) is the connection-scoped variant for admin flows that need locks to outlive transactions.
Key files: `pkg/framework/lockr/locker.go`, `pkg/framework/lockr/session.go`, `openmeter/billing/charges/lock.go`, `openmeter/billing/service/service.go`
Example: `// Acquiring per-charge lock inside a transaction:
key, err := charges.NewLockKeyForCharge(chargeID)
if err != nil {
    return err
}
return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, tx *adapter) error {
    if err := locker.LockForTX(ctx, key); err != nil {
        return fmt.Errorf("acquire charge lock: %w", err)
    }
    // ... perform mutations under the lock ...
    return nil
})`
**Applicable when:** Locker.LockForTX must run inside an active Ent transaction; calling outside returns an error (pkg/framework/lockr/locker.go:1).
**Do NOT apply when:**
  - Hand-constructing lockr.Key strings instead of using charges.NewLockKeyForCharge / billing.WithLock
  - Mixing context.WithTimeout with advisory locks - use pgdriver.WithLockTimeout instead
  - Sharing SessionLocker across goroutines under high contention without Close()
- Use charges.NewLockKeyForCharge / billing.WithLock helpers - do not construct lockr.Key strings inline.
- Don't mix context.WithTimeout with advisory locks - use pgdriver.WithLockTimeout instead.
- SessionLocker is not goroutine-safe under high contention; always Close() to release the dedicated connection.

### Instrument with OpenTelemetry tracing, metrics, and ctx propagation [analytics]
Libraries: `OpenTelemetry otel v1.43.0`, `Prometheus client v1.23.2`, `pkg/framework/tracex`
Pattern: Every entry point (HTTP handler, Kafka consumer, Ent adapter) is instrumented. trace.Tracer is injected via Wire into service constructors. ingest.ingestadapter.WithTelemetry wraps openmeter/ingest.Collector. openmeter/watermill/router.NewDefaultRouter installs OTel middleware. tracex.Start/Wrap is preferred over tracer.Start because it records errors, sets span status, and recovers panics. ctx must be threaded from HTTP handler / Kafka consumer all the way through service+adapter; introducing context.Background() or context.TODO() to bridge missing plumbing is a project-rule violation. Tests use t.Context() rather than context.Background().
Key files: `app/common/telemetry.go`, `openmeter/ingest/ingestadapter`, `openmeter/watermill/router`, `openmeter/watermill/grouphandler`, `pkg/framework/tracex/tracex.go`
Example: `// Always thread ctx through:
func (s *svc) DoWork(ctx context.Context, id string) error {
    return tracex.Start(ctx, s.tracer, "svc.DoWork", func(span *tracex.Span[any]) (any, error) {
        return nil, span.Wrap(s.adapter.Write(span.Ctx(), id))
    }).Err()
}

// In Kafka consumer handler - use msg.Context(), never context.Background():
func (h *handler) onEvent(msg *message.Message) error {
    ctx := msg.Context()
    return svc.OnInvoiceCreated(ctx, ev)
}`
**Do NOT apply when:**
  - Substituting context.Background() to work around a missing ctx plumbing issue
  - Using context.Background() in tests when *testing.T is available (use t.Context())
  - Calling tracer.Start directly instead of tracex.Start/Wrap (loses error recording and panic recovery)
- Two legitimate exceptions for context.Background(): root context at program start in main(), and post-cancel graceful shutdown.
- Prefer tracex.Start/Wrap over tracer.Start to centralise error recording and panic recovery.
- In tests, use t.Context() instead of context.Background() when *testing.T is available.