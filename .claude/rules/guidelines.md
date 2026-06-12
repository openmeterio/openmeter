## Implementation Guidelines

### Transaction-aware Ent adapter participating in a caller's transaction [persistence]
Libraries: `entgo.io/ent v0.14.6`, `github.com/jackc/pgx/v5 v5.9.2`
Pattern: Adapter structs hold a *entdb.Client and implement entutils.TxCreator (Tx hijacks an Ent tx onto ctx) and TxUser[T] (WithTx rebinds the adapter to the tx client; Self returns the non-tx instance). Each adapter method wraps its body in entutils.TransactingRepo(ctx, a, func(ctx, rep){...}) so it transparently joins a caller-supplied tx (rebinding on the *TxDriver in ctx) or runs standalone via Self(). Services compose other services' adapters inside one transaction.Run.
Key files: `pkg/framework/entutils/transaction.go`, `pkg/framework/transaction/transaction.go`, `openmeter/customer/adapter/adapter.go`, `openmeter/billing/adapter/adapter.go`
Example: `func (a *adapter) GetCustomer(ctx context.Context, id models.NamespacedID) (*customer.Customer, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, rep *adapter) (*customer.Customer, error) {
        row, err := rep.db.Customer.Query().
            Where(customerdb.Namespace(id.Namespace), customerdb.ID(id.ID)).
            Only(ctx)
        if err != nil {
            return nil, err
        }
        return mapCustomerFromDB(row), nil
    })
}`
**Applicable when:** Adapter whose backing struct holds an *entdb.Client and implements both TxUser[T] and TxCreator — TransactingRepo only rebinds when a *TxDriver is on ctx, else falls back to repo.Self() (pkg/framework/entutils/transaction.go:199).
**Do NOT apply when:**
  - Pure in-memory adapter with no Ent client (registry/map-backed components) — there is no transaction to join (openmeter/app/adapter/marketplace.go:120).
  - Helper that accepts a raw *entdb.Client argument instead of reading the tx from ctx — per AGENTS.md the charges adapter convention still requires wrapping the body with entutils.TransactingRepo / TransactingRepoWithNoValue so it rebinds to the ctx tx; passing a non-tx client silently writes outside the caller's transaction.
- Never introduce context.Background()/context.TODO() to sidestep tx propagation (AGENTS.md); thread the caller's ctx through the full path.
- Self() must return the non-tx instance so standalone calls do not accidentally reuse a stale tx client.
- Use transaction.Run(...) at the service layer to open the tx that adapters then join.

### Per-customer serialization with Postgres advisory locks (lockr) [persistence]
**Scope:** `subscription`, `billing`, `ledger`, `entitlement`
Libraries: `entgo.io/ent v0.14.6 (pg_advisory_xact_lock via raw SQL)`, `github.com/zeebo/xxh3 (xxh3 hash)`
Pattern: lockr.NewKey(scopes...) joins scopes with ':' and xxh3-hashes to a 64-bit int; Locker.LockForTX runs SELECT pg_advisory_xact_lock($1) on the current Ent tx, auto-released on commit/rollback. Domains expose typed key builders (subscription.GetCustomerLock(customerID)). getTxClient asserts the call is inside a real Postgres tx and errors otherwise.
Key files: `pkg/framework/lockr/locker.go`, `pkg/framework/lockr/key.go`, `openmeter/subscription/locks.go`, `openmeter/subscription/service/service.go`
Example: `err := transaction.Run(ctx, a.driver, func(ctx context.Context) error {
    key := subscription.GetCustomerLock(customerID)
    if err := a.locker.LockForTX(ctx, key); err != nil {
        return err
    }
    // all mutations below are serialized per-customer and
    // the lock releases automatically when this tx commits
    return a.applySubscriptionEdits(ctx, edits)
})`
**Applicable when:** The lock-key identity component is a globally-unique row id — subscription.GetCustomerLock keys on customer id, and customer ids are PK-unique via IDMixin field.String("id").Unique() + index.Fields("namespace","id").Unique() (openmeter/subscription/locks.go:6).
**Do NOT apply when:**
  - Caller keys the lock on a namespace-non-unique column — customer key is only unique under namespace + deleted_at IS NULL (openmeter/ent/schema/customer.go:58-62), so a key-based lock could serialize unrelated namespaces or collide a live row with a soft-deleted one.
  - Caller is not inside a Postgres transaction — lockr.getTxClient (pkg/framework/lockr/locker.go:134) hard-errors when transaction_timestamp()==statement_timestamp(), so LockForTX outside transaction.Run always fails.
  - Operation mutates exactly one row with no cross-row invariant — Ent SELECT ... FOR UPDATE row locking suffices and avoids two scope strings colliding into the same 64-bit advisory slot.
- Two distinct scope strings can xxh3-collide into the same advisory slot; keep scope vocabularies disjoint per domain.
- Billing also serializes per-customer via the BillingCustomerLock row (SELECT ... FOR UPDATE) — pick the mechanism the surrounding code already uses for that domain.
- The lock is transaction-scoped: do work and commit; do not hold across multiple transactions.

### Guarded lifecycle state machine with external (aggregate-stored) state [state_management]
**Scope:** `billing`, `charges`
Libraries: `github.com/qmuntal/stateless v1.8.0`
Pattern: stateless.NewStateMachineWithExternalStorage keeps the current state on the domain aggregate (invoice.status / charge.status). Each state is Configure'd with Permit/PermitDynamic(guard) edges and OnActive side-effect callbacks. The charge machine is generic over STATUS (~string + Validate()) and a CHARGE implementing ChargeLike (GetStatus/WithStatus/GetBase/WithBase).
Key files: `openmeter/billing/service/stdinvoicestate.go`, `openmeter/billing/charges/statemachine/machine.go`
Example: `sm := stateless.NewStateMachineWithExternalStorage(
    func(ctx context.Context) (any, error) { return inv.Status, nil },
    func(ctx context.Context, s any) error { inv.Status = s.(billing.InvoiceStatus); return nil },
    stateless.FiringImmediate,
)
sm.Configure(billing.InvoiceStatusDraft).
    Permit(triggerFinalize, billing.InvoiceStatusIssued).
    OnActive(func(ctx context.Context, _ ...any) error {
        return svc.finalizeInvoice(ctx, inv)
    })
if err := sm.Fire(triggerFinalize); err != nil {
    return err
}`
**Applicable when:** Charge aggregate that carries its status as a string enum and implements ChargeLike — machine.go:39 requires GetStatus/WithStatus/GetBase/WithBase and machine.go:16 constrains STATUS to ~string + Validate().
**Do NOT apply when:**
  - State is not externally owned by the aggregate — these machines use NewStateMachineWithExternalStorage (stdinvoicestate.go:48) and read/write status on the persisted row; an in-memory-state FSM would not persist transitions across requests.
  - A transition has no guard and no side effect and the type has only two states — a plain bool/enum field is simpler than configuring a stateless machine.
- Declare every legal transition as a Permit edge; a status reachable only by direct mutation bypasses guards and side effects.
- OnActive side effects run inside the surrounding transaction — keep them idempotent for retry safety.
- Reuse the generic charge machine across flat-fee/usage-based/credit-purchase by satisfying ChargeLike rather than writing a new machine per subtype.

### Compile-time dependency injection with feature-gated provider swaps (Wire) [state_management]
Libraries: `github.com/google/wire v0.7.0`
Pattern: Constructors are grouped into wire.NewSet provider sets per concern in app/common/*.go; each binary's build-tagged wire.go (//go:build wireinject) lists the sets it needs and make generate emits wire_gen.go. Feature flags choose concrete vs noop providers at wiring time (app/common/ledger.go on credits.enabled; NewCustomerLedgerService returns a Noop hook when disabled).
Key files: `cmd/server/wire.go`, `app/common/customer.go`, `app/common/billing.go`, `app/common/ledger.go`, `app/common/app.go`
Example: `// app/common/customer.go
var Customer = wire.NewSet(
    NewCustomerService,
    NewCustomerAdapter,
)

func NewCustomerLedgerService(conf config.Configuration) customer.ServiceHook {
    if !conf.Credits.Enabled {
        return ledgernoop.NewCustomerHook() // feature disabled -> noop
    }
    return ledger.NewCustomerHook(/* concrete deps */)
}`
**Applicable when:** Subsystem gated by a config flag that must disable cleanly at every layer — credits.enabled wires ledger account services/resolvers to noop implementations across api/v3 handlers, customer ledger hooks, and namespace provisioning (app/common/ledger.go; AGENTS.md).
**Do NOT apply when:**
  - Checking the feature flag inside service business logic instead of swapping the provider — a disabled subsystem must be a noop at the wiring seam, not a runtime conditional scattered through methods (AGENTS.md credits.enabled guidance).
  - Any ledger-account backfill that must write real ledger_accounts / ledger_customer_accounts rows — when credits.enabled is false the default DI outputs are noop, so the backfill must construct concrete ledger account + resolver adapters directly (AGENTS.md).
- Never edit wire_gen.go by hand; change the provider set and run make generate.
- Each binary links only the provider sets in its wire.go — adding a service to one binary does not add it to the others.
- Register namespace handlers before initNamespace(...) in cmd/server/main.go if they must provision the default namespace at startup (AGENTS.md).

### Domain events over Kafka via Watermill CQRS and type-routed consumers [notifications]
**Scope:** `notification`, `billing`, `entitlement`, `ingest`
Libraries: `github.com/ThreeDotsLabs/watermill v1.5.2`, `github.com/ThreeDotsLabs/watermill-kafka/v3 v3.1.2`, `github.com/cloudevents/sdk-go/v2 v2.16.2`, `github.com/confluentinc/confluent-kafka-go/v2 v2.14.1`
Pattern: Outbound: eventbus.New wraps a watermill cqrs.EventBus; Publish routes each marshaler.Event to one of three Kafka topics by event-name prefix (ingest/balance-worker/system). Inbound: grouphandler.NewNoPublishingHandler builds map[eventName][]GroupEventHandler, derives the CloudEvent type per message, unmarshals once into handler[0].NewEvent(), runs all matching handlers joining errors, and counts unknown types as ignored. A notification DLQ topic (om_sys.notification_service_dlq) catches poison messages.
Key files: `openmeter/watermill/eventbus/eventbus.go`, `openmeter/watermill/grouphandler/grouphandler.go`, `openmeter/notification/consumer/consumer.go`
Example: `handler := grouphandler.NewNoPublishingHandler(
    logger,
    eventMarshaler,
    grouphandler.NewGroupEventHandler(func(ctx context.Context, e *MyEvent) error {
        return svc.Handle(ctx, e)
    }),
)
// publishing from a producer:
if err := eventBus.Publish(ctx, myEvent); err != nil {
    return err
}`
**Applicable when:** Asynchronous cross-binary work where eventual consistency is acceptable — publish routes by event-name prefix to IngestEventsTopic/BalanceWorkerEventsTopic/SystemEventsTopic (openmeter/watermill/eventbus/eventbus.go).
**Do NOT apply when:**
  - Cross-domain effect that must commit in the same transaction as the triggering write (e.g. provisioning a ledger account on customer create) — use the in-process Service hook registry (openmeter/customer/service/service.go), not Kafka.
  - Producing/consuming Kafka directly with confluent-kafka-go instead of the eventbus/grouphandler abstractions — bypasses topic routing, CloudEvents marshaling, and unknown-type accounting.
- Event-name prefix determines the topic; name new events consistently so they route correctly.
- All handlers for a type run and their errors are joined — make each handler independently retry-safe.
- Unknown event types are ignored (counted), not errored — a missing handler registration fails silently.

### Pluggable third-party app integration via the marketplace registry [payments]
**Scope:** `app`
Libraries: `github.com/stripe/stripe-go/v80 v80.2.1`, `github.com/invopop/gobl v0.403.0`
Pattern: The app adapter holds registry map[AppType]RegistryItem. Each integration's service constructor calls AppService.RegisterMarketplaceListing(RegistryItem{Listing, Factory}) once at wiring time; RegisterMarketplaceListing rejects duplicate AppTypes and validates the listing. Install operations look up the factory by AppType; capability support (e.g. billing) is discovered by type-asserting the constructed instance against capability interfaces.
Key files: `openmeter/app/adapter/marketplace.go`, `openmeter/app/registry.go`, `openmeter/app/marketplace.go`, `openmeter/app/stripe/service/service.go`
Example: `func NewStripeService(appSvc app.Service, deps Deps) (Service, error) {
    s := &service{deps: deps}
    err := appSvc.RegisterMarketplaceListing(app.RegistryItem{
        Listing: stripeListing,
        Factory: s, // builds an App instance for an install
    })
    if err != nil {
        return nil, err
    }
    return s, nil
}`
**Applicable when:** Integration factory that registers exactly once per process at DI wiring time — RegisterMarketplaceListing rejects a second registration of the same AppType (openmeter/app/adapter/marketplace.go:121).
**Do NOT apply when:**
  - Registration happens after the listing surface is already serving requests — the map has no locking around late registration, so listings must be registered during DI wiring before the HTTP/worker surface is live (openmeter/app/adapter/marketplace.go:121).
- Discover an installed app's billing/invoicing capability by type assertion against the capability interface, not by an AppType switch.
- Use GOBL for invoice document formatting behind the billing service rather than emitting payment-provider-specific invoice shapes.
- Stripe API access uses stripe-go/v80; keep the pinned major in sync with the SDK version in go.mod.

### Persisted entity definition via Ent schema mixins + Atlas migrations [persistence]
Libraries: `entgo.io/ent v0.14.6`, `ariga.io/atlas v0.36.x`, `github.com/golang-migrate/migrate/v4 v4.19.1`, `github.com/oklog/ulid/v2 v2.1.1`
Pattern: A schema struct composes entutils mixins in Mixin() (ResourceMixin → IDMixin ULID char(26) PK + NamespaceMixin + MetadataMixin jsonb + TimeMixin created/updated/deleted_at + unique (namespace,id); UniqueResourceMixin adds a (namespace,key,deleted_at) unique index). Edit the schema, run make generate to regenerate openmeter/ent/db, then atlas migrate --env local diff <name> to emit .up.sql/.down.sql under tools/migrate/migrations and update atlas.sum.
Key files: `pkg/framework/entutils/mixins.go`, `openmeter/ent/schema/customer.go`, `openmeter/ent/schema/billing.go`, `tools/migrate/migrations`
Example: `func (Customer) Mixin() []ent.Mixin {
    return []ent.Mixin{
        entutils.ResourceMixin{},   // id+namespace+metadata+timestamps+soft-delete
        BillingAddressMixin{},
    }
}

func (Customer) Fields() []ent.Field {
    return []ent.Field{
        field.String("key").Optional(),
        field.String("primary_email").Optional(),
    }
}`
**Applicable when:** Entity relying on key-uniqueness within a namespace — UniqueResourceMixin's (namespace,key,deleted_at) index only approximates partial uniqueness (pkg/framework/entutils/mixins.go:47).
**Do NOT apply when:**
  - Entity needs true partial-unique (namespace,key) WHERE deleted_at IS NULL — add a custom SQL migration with IndexWhere as Customer does (openmeter/ent/schema/customer.go:58-62), rather than relying on the mixin's deleted_at-in-key approximation.
  - Ent ent.View schemas — they generate query code but do NOT appear in migrate.Tables, so the view DDL needs an explicit SQL migration (ChargesSearchV1; AGENTS.md ent-view caveat).
- Never edit openmeter/ent/db (generated); change the schema and run make generate.
- Drop incidental go.sum entries (e.g. tablewriter) that make generate / atlas diff add unless the task requires a dependency change (AGENTS.md).
- pr-checks enforce atlas.sum append-only; never rewrite existing migration files.

### Typed v1 HTTP handler with domain-error-to-status mapping [networking]
Libraries: `github.com/go-chi/chi/v5 v5.2.5`, `github.com/getkin/kin-openapi v0.139.0`, `github.com/go-chi/render v1.0.3`
Pattern: Per-domain httpdriver packages build httptransport.NewHandler[Request,Response](decode, service-op, encode, ...opts): a decoder maps the HTTP request to a typed Request, the service op runs, an encoder writes the Response, and an errorEncoder chains commonhttp.HandleErrorIfTypeMatches[T] in order, short-circuiting on errors.As against concrete domain error types into RFC7807 problem documents. The v1 router validates requests against the embedded OpenAPI3 spec first.
Key files: `pkg/framework/transport/httptransport/handler.go`, `openmeter/notification/httpdriver/handler.go`, `pkg/framework/commonhttp/errors.go`, `openmeter/notification/httpdriver/errors.go`
Example: `func (h handler) GetChannel() httptransport.Handler[GetChannelRequest, api.Channel] {
    return httptransport.NewHandler(
        func(ctx context.Context, r *http.Request) (GetChannelRequest, error) {
            return GetChannelRequest{ID: chi.URLParam(r, "id")}, nil
        },
        func(ctx context.Context, req GetChannelRequest) (api.Channel, error) {
            return h.service.GetChannel(ctx, req.ID)
        },
        commonhttp.JSONResponseEncoder[api.Channel],
        httptransport.WithErrorEncoder(h.resolveErrorEncoder()),
    )
}`
**Applicable when:** Implementing a v1 endpoint where domain errors must map to HTTP status — errorEncoder uses ordered errors.As short-circuits (notification.NotFoundError→404, GenericValidationError→400, UpdateAfterDeleteError→409) (openmeter/notification/httpdriver/errors.go).
**Do NOT apply when:**
  - Implementing a v3 AIP endpoint — v3 uses oapi-codegen ServerInterface delegators in api/v3/server/routes.go with shared rendering in api/v3/apierrors, not httptransport.Handler.
  - A handler that maps a domain error inline to a status instead of through the typed errorEncoder chain — breaks the uniform RFC7807 mapping (pkg/framework/commonhttp/errors.go).
- Order the errorEncoder chain most-specific-first; errors.As stops at the first match.
- Return domain error types (NotFoundError, GenericValidationError) from services so the encoder can map them — do not pre-format HTTP errors in services.
- GenericValidationError from accumulating Validate() maps to a single 400 ValidationIssue.

### In-process cross-domain reactions via Service hook registry [state_management]
**Scope:** `customer`, `subscription`, `ledger`
Libraries: `github.com/openmeterio/openmeter/pkg/models (ServiceHookRegistry)`
Pattern: A service embeds models.ServiceHookRegistry[T] and exposes RegisterHooks(...ServiceHook[T]); on a lifecycle event it invokes registered hooks synchronously inside the same transaction. Wire providers in app/common build the hook and call targetService.RegisterHooks(h) at startup, registering a Noop hook when the feature is disabled.
Key files: `openmeter/customer/service/service.go`, `app/common/customer.go`, `pkg/models`
Example: `// at wiring time in app/common/customer.go
ledgerHook := NewCustomerLedgerService(conf) // concrete or Noop
customerService.RegisterHooks(ledgerHook)

// the hook reacts to customer creation in-process / in-tx
func (h *ledgerHook) PostCreate(ctx context.Context, c *customer.Customer) error {
    return h.ledger.ProvisionAccount(ctx, c)
}`
**Applicable when:** Cross-domain effect that must be synchronous and transactional with the triggering write and feature-gatable — customer-create → ledger account provisioning is registered as a hook and swapped to Noop when credits.enabled is false (app/common/customer.go).
**Do NOT apply when:**
  - Effect tolerates eventual consistency and should not block the triggering transaction — publish a domain event over the Kafka eventbus instead (openmeter/watermill/eventbus/eventbus.go).
  - Triggering service would gain a static import dependency on the dependent domain — the hook registry exists precisely to avoid that compile-time edge; do not hard-code the dependent call in the service body.
- Hooks run inside the triggering transaction; a hook error should roll the whole operation back when the effect is mandatory.
- Provide a Noop hook implementation so a disabled feature registers a no-op rather than nothing.
- Keep hook signatures generic via ServiceHook[T] so multiple reactors can register against one service.

### Code-generated type conversion between domain, API, and DB representations [persistence]
Libraries: `github.com/jmattheis/goverter v1.9.3`, `github.com/awalterschulze/goderive v0.5.1`
Pattern: convert.gen.go files are generated from goverter converter interfaces declared in convert.go; billing/derived.gen.go is generated from goderive annotations (equality/clone helpers). Hand-written conversion functions follow the FromAPI.../ToAPI.../FromDB.../ToDB... naming convention (the go-types-conversion skill); make generate regenerates them.
Key files: `api/convert.go`, `api/convert.gen.go`, `openmeter/billing/derived.gen.go`, `openmeter/customer/adapter/entitymapping.go`
Example: `// goverter interface in convert.go (source of generation)
// goverter:converter
type Converter interface {
    ToAPICustomer(c customer.Customer) api.Customer
    FromAPICustomerCreate(in api.CustomerCreate) customer.CustomerMutate
}
// run `make generate` -> convert.gen.go implements Converter`
**Applicable when:** Translating between domain/API/DB types where the mapping is structural — hand-written mappers must use FromAPI/ToAPI/FromDB/ToDB names (openmeter/customer/adapter/entitymapping.go).
**Do NOT apply when:**
  - Editing a *.gen.go converter by hand — they carry a DO NOT EDIT header; change convert.go (goverter) or the goderive annotation and regenerate.
  - Using project/projected terminology for domain mapping — AGENTS.md mandates map/mapped terminology and the FromAPI/ToAPI/FromDB/ToDB function names.
- For equality/clone of billing aggregates use the goderive-generated helpers in derived.gen.go rather than hand-writing deep-equals.
- Keep one converter interface per boundary (API↔domain, DB↔domain) so goverter generation stays scoped.

### Svix-backed webhook delivery behind a Handler interface [notifications]
**Scope:** `notification`
Libraries: `github.com/svix/svix-webhooks v1.95.1`, `go.opentelemetry.io/otel v1.44.0`
Pattern: notification/webhook defines a Handler interface (CreateWebhook/UpdateWebhook/endpoint secret + header management) with a Svix implementation (webhook/svix) and a noop implementation (webhook/noop) for when webhooks are disabled. Svix calls are wrapped in OpenTelemetry tracex spans. A reconciliation loop in eventhandler/reconcile.go re-drives delivery for events whose webhook send did not confirm.
Key files: `openmeter/notification/webhook/svix/webhook.go`, `openmeter/notification/webhook/noop/noop.go`, `openmeter/notification/eventhandler/reconcile.go`
Example: `func NewWebhookHandler(conf config.NotificationConfiguration) (webhook.Handler, error) {
    if !conf.Webhook.Enabled {
        return noop.NewHandler(), nil
    }
    return svix.NewHandler(svix.Config{
        ServerURL: conf.Webhook.SvixServerURL,
        APIKey:    conf.Webhook.SvixAPIKey,
    })
}`
**Applicable when:** Delivering notification events to customer-configured endpoints where delivery is feature-gated — the Handler interface has both a Svix and a noop implementation selected at wiring time (openmeter/notification/webhook/noop/noop.go).
**Do NOT apply when:**
  - Webhooks disabled in config — wire the noop Handler, not the Svix one, so no Svix API key is required (openmeter/notification/webhook/noop/noop.go).
  - Treating a Svix send as the durable source of truth — the NotificationEventDeliveryStatus row + reconcile loop (openmeter/notification/eventhandler/reconcile.go) are authoritative; an un-reconciled send can be re-driven.
- Persist NotificationEventDeliveryStatus before/around the Svix call so the reconcile loop can recover unconfirmed deliveries.
- Wrap Svix calls in tracex spans for observability (the svix Handler already does).
- Payload versioning lives in eventpayload.go — bump the payload version when changing the wire shape (see /notification skill).