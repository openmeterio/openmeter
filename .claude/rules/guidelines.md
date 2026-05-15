## Implementation Guidelines

### Author HTTP API endpoints (v1 + v3) from a single TypeSpec source compiled to Go server stubs and three SDKs [networking]
**Scope:** `openmeter/server/router`, `api/v3/server`, `api/v3/handlers`, `openmeter/billing/httpdriver`, `openmeter/customer/httpdriver`, `openmeter/meter/httphandler`
Libraries: `TypeSpec @typespec/compiler 1.11.0`, `oapi-codegen v2.6.1 (pinned fork)`, `Chi v5.2.5`, `kin-openapi v0.137.0`, `oasmiddleware v1.1.2`
Pattern: Author endpoints in TypeSpec under api/spec/packages/legacy (v1) or api/spec/packages/aip (v3) with route/tag bindings only in the root openmeter.tsp. Run `make gen-api` to regenerate api/openapi.yaml + api/v3/openapi.yaml + api/api.gen.go + api/v3/api.gen.go + Go/JS/Python SDKs, then `make generate` for downstream Wire/Ent/Goverter/Goderive. Implement the generated ServerInterface in openmeter/<domain>/httpdriver (v1) or api/v3/handlers/<resource> (v3) using pkg/framework/transport/httptransport.NewHandler, which separates decode -> operate -> encode and appends commonhttp.GenericErrorEncoder to map models.Generic* sentinels to RFC 7807 problem+json.
Key files: `api/spec/packages/aip`, `api/spec/packages/legacy`, `api/api.gen.go`, `api/v3/api.gen.go`, `openmeter/server/router`, `api/v3/server`, `pkg/framework/transport/httptransport/handler.go`, `pkg/framework/commonhttp`
Example: `// 1. Edit TypeSpec under api/spec/packages/aip to add the operation
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
**Applicable when:** networking: TypeSpec source compiles to dual API artifacts and three SDKs; both `make gen-api` and `make generate` are required to keep server stubs and SDKs in sync (api/spec/packages/aip/src/openmeter.tsp:1).
**Do NOT apply when:**
  - networking: hand-editing api/openapi.yaml or any *.gen.go file - these are regenerated and edits are silently overwritten
  - networking: adding a v1 endpoint into api/spec/packages/aip/ instead of api/spec/packages/legacy/
  - networking: writing a handler that implements ServeHTTP directly, bypassing httptransport.NewHandler and its GenericErrorEncoder + OTel chain
- Keep v1 changes in api/spec/packages/legacy and v3 changes in api/spec/packages/aip; never mix them in one package.
- Return models.Generic* sentinels from the service layer; GenericErrorEncoder maps them to the correct HTTP status.
- TypeSpec files using @query/@route must import @typespec/http and add `using TypeSpec.Http;`.

### Persist domain data via Ent schema, Atlas migrations, and context-propagated transactions [persistence]
**Scope:** `openmeter/billing/adapter`, `openmeter/billing/charges/adapter`, `openmeter/customer/adapter`, `openmeter/notification/adapter`, `openmeter/ledger`, `openmeter/entitlement`, `openmeter/subscription`
Libraries: `Ent v0.14.6`, `Atlas CLI 0.36.0`, `golang-migrate v4.19.1`, `pgx v5.9.2`
Pattern: Define Ent entity schemas under openmeter/ent/schema (each with entutils.IDMixin + NamespaceMixin + TimeMixin). Run `make generate` to regenerate openmeter/ent/db/. Generate migrations with `atlas migrate --env local diff <name>`, committing the .up.sql/.down.sql pair plus the updated atlas.sum together. Each domain adapter under openmeter/<domain>/adapter implements the TxCreator + TxUser triad (Tx via HijackTx + NewTxDriver, WithTx via NewTxClientFromRawConfig, Self) and wraps every method body in entutils.TransactingRepo or TransactingRepoWithNoValue so it rebinds to any ctx-bound transaction or runs on Self().
Key files: `openmeter/ent/schema`, `openmeter/ent/db`, `tools/migrate/migrations`, `tools/migrate/migrations/atlas.sum`, `atlas.hcl`, `pkg/framework/entutils/transaction.go`, `openmeter/billing/charges/adapter/adapter.go`
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
**Applicable when:** persistence: adapters implementing the TxCreator + TxUser triad require entutils.TransactingRepo on every method body so the ctx-bound Ent transaction is honored (pkg/framework/entutils/transaction.go:199).
**Do NOT apply when:**
  - persistence: an adapter struct storing *entdb.Tx as a field instead of rebinding via TransactingRepo
  - persistence: a helper that accepts *entdb.Client and calls a.db.Foo() directly without TransactingRepoWithNoValue
  - persistence: hand-writing a migration in tools/migrate/migrations/ without going through `atlas migrate --env local diff`
- Never edit openmeter/ent/db/ - it is fully generated by `make generate`.
- After a schema change run `make generate` then `atlas migrate --env local diff <name>`, and commit schema + generated code + migration + atlas.sum together.
- Every new entity needs IDMixin + NamespaceMixin + TimeMixin or multi-tenancy and soft-delete break.

### Publish and consume async domain events across binaries via Kafka + Watermill [state_management]
**Scope:** `openmeter/watermill`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification/consumer`, `openmeter/sink`, `openmeter/ingest`
Libraries: `confluent-kafka-go v2.14.1 (librdkafka)`, `Watermill v1.5.1 + watermill-kafka/v3 v3.1.2`, `OpenTelemetry v1.43.0`
Pattern: openmeter/watermill/eventbus wraps Watermill's cqrs.EventBus with a TopicMapping of IngestEventsTopic, SystemEventsTopic, BalanceWorkerEventsTopic; GeneratePublishTopic routes by EventName() prefix. Producers call eventbus.Publisher.Publish or WithContext(ctx).PublishIfNoError. Consumers build routers via openmeter/watermill/router.NewDefaultRouter (PoisonQueue, DLQ, CorrelationID, Recoverer, Retry, ProcessingTimeout, HandlerMetrics) and dispatch via grouphandler.NoPublishingHandler keyed on CloudEvents ce_type; unknown ce_types are silently dropped to allow rolling deploys.
Key files: `openmeter/watermill/eventbus/eventbus.go`, `openmeter/watermill/router/router.go`, `openmeter/watermill/grouphandler/grouphandler.go`, `openmeter/watermill/marshaler`, `openmeter/billing/worker`, `openmeter/entitlement/balanceworker`, `openmeter/notification/consumer`
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
**Applicable when:** state_management: cross-binary event delivery uses eventbus.Publisher; topic routing is determined by EventName() prefix and must match a recognised EventVersionSubsystem constant (openmeter/watermill/eventbus/eventbus.go:135).
**Do NOT apply when:**
  - state_management: publishing directly to a Kafka topic by string literal instead of through eventbus.Publisher
  - state_management: an EventName() lacking a registered EventVersionSubsystem prefix - it silently routes to SystemEventsTopic
  - state_management: substituting context.Background() inside a Watermill handler instead of msg.Context()
  - state_management: returning an error for an unknown ce_type in a NoPublishingHandler - this poisons the DLQ during rolling deploys
- Always build and test with -tags=dynamic so confluent-kafka-go links against librdkafka.
- Use msg.Context() inside handlers; never substitute context.Background() - it severs OTel spans and drops the Ent transaction.
- Build consumer routers only via router.NewDefaultRouter to inherit the fixed middleware stack.

### Drive billing and charge lifecycle via tagged-union domain models, state machines, and the LineEngine registry [payments]
**Scope:** `openmeter/billing`, `openmeter/billing/service`, `openmeter/billing/charges`, `openmeter/billing/worker`, `openmeter/billing/adapter`
Libraries: `qmuntal/stateless v1.8.0`, `Goverter v1.9.3`, `Goderive v0.5.1`, `alpacadecimal v0.0.9`, `GOBL v0.401.0`
Pattern: billing.Service is a composite interface implemented in openmeter/billing/service, driving the invoice lifecycle through a stateless.StateMachine pooled in sync.Pool (stdinvoicestate.go) bound to Invoice.Status. openmeter/billing/charges owns the Charge / ChargeIntent tagged-union (private meta.ChargeType discriminator) constructed only via NewCharge[T] / NewChargeIntent[T] and accessed via AsFlatFeeCharge / AsUsageBasedCharge / AsCreditPurchaseCharge. Each charge type plugs into a generic Machine[CHARGE,BASE,STATUS] and registers a LineEngine with billing.Service.RegisterLineEngine in app/common/charges.go. Customer-mutating operations acquire pg_advisory_xact_lock per customer via lockr inside an active transaction.
Key files: `openmeter/billing/service.go`, `openmeter/billing/service/service.go`, `openmeter/billing/charges/service.go`, `openmeter/billing/charges/adapter/adapter.go`, `app/common/charges.go`, `pkg/framework/lockr/locker.go`
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
**Applicable when:** payments: Charge / ChargeIntent / InvoiceLine tagged-union construction must go through NewCharge[T] / NewChargeIntent[T] / NewStandardInvoiceLine; a struct literal leaves the private discriminator zero-valued and accessors error (openmeter/billing/charges/service.go:1).
**Do NOT apply when:**
  - payments: constructing charges.Charge{}, charges.ChargeIntent{}, or billing.InvoiceLine{} via struct literal
  - payments: mutating Invoice.Status directly instead of going through the stateless state machine's FireAndActivate
  - payments: calling charges adapter methods directly from tests instead of charges.Service.Create / AdvanceCharges / ApplyPatches
  - payments: calling lockr.LockForTX outside an active Ent transaction
- Register charge LineEngines only in app/common/charges.go, never from domain packages or cmd/*.
- Use charges.NewLockKeyForCharge(chargeID) for per-charge advisory locks; never hand-construct lockr.Key strings.
- Use MockStreamingConnector with explicit StoredAt to exercise stored-at cutoff logic in charge finalization tests.

### Deliver outbound webhooks via Svix with a reconciliation loop [notifications]
**Scope:** `openmeter/notification`, `openmeter/notification/consumer`, `cmd/notification-service`
Libraries: `svix-webhooks Go SDK v1.90.0`, `Watermill v1.5.1`, `openmeter/watermill/eventbus`
Pattern: openmeter/notification manages channels, rules, events, and delivery status. notification.EventHandler runs Dispatch + Reconcile loops inside cmd/server's run.Group or independently in cmd/notification-service. The Watermill consumer in openmeter/notification/consumer subscribes to the system events topic, builds the payload, and sends through the webhook.Handler interface - concrete impl in openmeter/notification/webhook/svix and a noop fallback when Svix is unconfigured. The NullChannel sentinel prevents unfiltered delivery; payload version is pinned per event family.
Key files: `openmeter/notification/service.go`, `openmeter/notification/eventhandler/handler.go`, `openmeter/notification/eventhandler/dispatch.go`, `openmeter/notification/consumer`, `openmeter/notification/webhook/svix/svix.go`, `cmd/notification-service/main.go`
Example: `// Consumer dispatches an invoice event to the webhook.Handler:
func (c *Consumer) onInvoiceCreated(ctx context.Context, ev billingevents.InvoiceCreated) error {
    return c.dispatcher.Dispatch(ctx, notification.Event{
        Type:    notification.TypeInvoiceCreated,
        Payload: notification.InvoicePayloadV1{ /* version-pinned constant */ },
    })
}`
**Applicable when:** notifications: notification handlers must be registered before initNamespace when the default namespace needs them (cmd/server/main.go:1).
**Do NOT apply when:**
  - notifications: adding ad-hoc retry inside the notification consumer - the Reconcile loop owns retry
  - notifications: skipping payload-version pinning for a new event family
  - notifications: dispatching directly to the Svix client, bypassing the NullChannel guard in notification.Service.Dispatch
- Pin payload version constants per event family and treat them as API contracts.
- The Reconcile loop owns retry of failed deliveries - do not duplicate retry logic inline.
- When Svix is unconfigured the noop webhook.Handler runs - verify tests exercise that branch.

### Compose each binary with Google Wire provider sets and register cross-domain hooks as side-effects [state_management]
**Scope:** `app/common`, `cmd/server`, `cmd/billing-worker`, `cmd/balance-worker`, `cmd/sink-worker`, `cmd/notification-service`, `cmd/jobs`
Libraries: `Google Wire v0.7.0`
Pattern: Each cmd/<binary>/wire.go declares a wire.Build over composite provider sets defined in app/common/ (per-domain files plus openmeter_<binary>.go per-binary sets). Domain packages expose plain constructors and never import app/common. Cross-domain ServiceHooks and RequestValidators are registered inside app/common provider functions as construction side-effects to avoid circular imports. Optional features (credits.enabled=false, Svix unconfigured) are gated by returning noop implementations from the provider rather than nil.
Key files: `app/common/billing.go`, `app/common/customer.go`, `app/common/ledger.go`, `app/common/charges.go`, `app/common/openmeter_server.go`, `app/common/openmeter_billingworker.go`, `cmd/billing-worker/wire.go`
Example: `// app/common/customer.go - hook registration as a provider side-effect:
func NewCustomerLedgerServiceHook(
    creditsConfig config.CreditsConfiguration,
    accountResolver customerLedgerProvisioner,
    customerService customer.Service,
) (CustomerLedgerHook, error) {
    if !creditsConfig.Enabled {
        return ledgerresolvers.NoopCustomerLedgerHook{}, nil
    }
    h, err := ledgerresolvers.NewCustomerLedgerHook(/* ... */)
    if err != nil {
        return nil, err
    }
    customerService.RegisterHooks(h) // side-effect: invisible to Wire's type graph
    return h, nil
}`
**Applicable when:** state_management: binary entrypoints composing ~40 domain services compile-time safely - Wire provider sets in app/common are verified at wire.Build compile time (cmd/billing-worker/wire.go:1).
**Do NOT apply when:**
  - state_management: a domain package under openmeter/ importing app/common - the import direction is one-way outward and reversing creates cycles
  - state_management: provider functions containing business logic beyond construction and hook/validator registration
  - state_management: calling domain constructors directly from wire.Build in cmd/* instead of using app/common provider sets
- Audit each binary's wire.go to confirm every required hook provider is included - an omitted hook provider compiles cleanly but silently drops the hook.
- Guard credits.enabled at all four wiring layers (ledger services, customer hooks, ChargesRegistry, v3 credit handlers).
- Return a noop struct, never nil, for disabled optional features - callers receive the interface and would panic on nil.

### Acquire per-customer distributed locks via pg_advisory_xact_lock [state_management]
**Scope:** `openmeter/billing`, `openmeter/billing/charges`, `openmeter/entitlement`
Libraries: `pkg/framework/lockr`, `PostgreSQL advisory locks`
Pattern: pkg/framework/lockr/locker.go calls pg_advisory_xact_lock($1) with a CRC64 hash of the lock key inside the active Ent transaction. getTxClient verifies a real Postgres transaction is in ctx by querying `SELECT transaction_timestamp() != statement_timestamp()` and errors if not. billing.Service.WithLock acquires the lock per CustomerID before any invoice or charge mutation; charges use charges.NewLockKeyForCharge(chargeID) for per-charge locks. Locks release automatically on transaction commit/rollback.
Key files: `pkg/framework/lockr/locker.go`, `openmeter/billing/charges/lock.go`, `openmeter/billing/service/service.go`
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
**Applicable when:** state_management: lockr.LockForTX must run inside an active Postgres transaction; getTxClient returns 'lockr only works in a postgres transaction' when statement_timestamp() == transaction_timestamp() (pkg/framework/lockr/locker.go:135).
**Do NOT apply when:**
  - state_management: calling LockForTX outside an active Ent transaction
  - state_management: wrapping LockForTX in context.WithTimeout - pgx cancels the connection on ctx cancel, see the comment at locker.go:91-93; use pg-side lock timeout instead
  - state_management: hand-constructing lockr.Key strings instead of using charges.NewLockKeyForCharge / billing.WithLock
- Always call LockForTX inside an entutils.TransactingRepo-established transaction.
- Per-charge lock keys come from charges.NewLockKeyForCharge; per-customer from billing.Service.WithLock.
- Do not mix context.WithTimeout with advisory locks - rely on the Postgres lock timeout.