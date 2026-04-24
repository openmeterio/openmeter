## Implementation Guidelines

### HTTP API (v1 + v3) [api]
Libraries: `TypeSpec`, `oapi-codegen`, `chi`, `kin-openapi`
Pattern: Author endpoints in TypeSpec under api/spec/packages/{aip,legacy}. Run `make gen-api` to regenerate OpenAPI YAML + Go server stubs + Go/JS/Python SDKs, then `make generate` for downstream Go code. Handler packages under openmeter/*/httpdriver (v1) or api/v3/handlers (v3) implement the generated server interface using the generic pkg/framework/transport/httptransport.Handler[Req,Resp] adapter and delegate to domain services.
Key files: `api/spec/packages/`, `api/openapi.yaml`, `api/v3/openapi.yaml`, `api/api.gen.go`, `api/v3/api.gen.go`, `openmeter/server/router/router.go`, `api/v3/server/server.go`, `pkg/framework/transport/httptransport`
Example: `// In api/spec/packages/aip/... add the TypeSpec op, then:
//   make gen-api   # regenerates api/v3/openapi.yaml + SDKs
//   make generate  # regenerates api/v3/api.gen.go + Go types
// In api/v3/handlers/foo/handler.go implement the generated interface:
func (h *Handler) ListFoos(ctx context.Context, req api.ListFoosRequestObject) (api.ListFoosResponseObject, error) {
    items, err := h.svc.List(ctx, req.Params.Namespace)
    if err != nil { return nil, err }
    return api.ListFoos200JSONResponse{Items: toAPI(items)}, nil
}`
- Never hand-edit *.gen.go; always regenerate.
- If TypeSpec adds @query and the file lacks HTTP decorators, import @typespec/http and `using TypeSpec.Http;`.
- Keep v1 changes in legacy/, v3 in aip/; do not mix.

### Persistence (schema + migrations) [database]
Libraries: `Ent`, `Atlas`, `golang-migrate`, `pgx`
Pattern: Ent schemas under openmeter/ent/schema are the source of truth. `make generate` regenerates openmeter/ent/db/. Migrations are produced by `atlas migrate --env local diff <name>` into tools/migrate/migrations/ as timestamped .up.sql/.down.sql plus atlas.sum. Adapters under openmeter/<domain>/adapter implement domain interfaces with Ent queries and must be transaction-aware via entutils.TransactingRepo.
Key files: `openmeter/ent/schema/`, `openmeter/ent/db/`, `tools/migrate/migrations/`, `tools/migrate/migrations/atlas.sum`, `atlas.hcl`, `pkg/framework/entutils`
Example: `// openmeter/<domain>/adapter/adapter.go
func (a *adapter) Create(ctx context.Context, in domain.CreateInput) (*domain.Entity, error) {
    return entutils.TransactingRepo(ctx, a.client, func(tx *entdb.Tx) (*domain.Entity, error) {
        row, err := tx.Entity.Create().SetNamespace(in.Namespace).SetName(in.Name).Save(ctx)
        if err != nil { return nil, err }
        return toDomain(row), nil
    })
}`
- Never edit openmeter/ent/db/; it is generated.
- After schema changes: make generate, then atlas migrate --env local diff <name>.
- Helpers that take *entdb.Client must still wrap with entutils.TransactingRepo to honor ctx tx.
- Ent views may not appear in migrate/schema.go; add explicit SQL migration if atlas reports no changes.

### Event ingestion and async workers [messaging]
Libraries: `confluent-kafka-go`, `Watermill`, `OpenTelemetry`
Pattern: openmeter/watermill wraps confluent-kafka-go. Publishers route to three topics (ingest, system, balance). Consumers (sink, balance, billing, notification) use watermill/router with OTel tracing. Sink flushes ingest events to ClickHouse, then publishes ingest notifications to trigger balance recalculation.
Key files: `openmeter/watermill/eventbus/eventbus.go`, `openmeter/watermill/router`, `openmeter/ingest/kafkaingest/collector.go`, `openmeter/sink/sink.go`, `openmeter/entitlement/balanceworker/worker.go`, `openmeter/notification/consumer/consumer.go`
Example: `// Publishing a domain event from a service:
if err := h.eventbus.Publish(ctx, eventbus.SystemTopic, &billingevents.InvoiceCreated{InvoiceID: inv.ID}); err != nil {
    return fmt.Errorf("publish invoice created: %w", err)
}
// Consumer side (Watermill router):
router.AddNoPublisherHandler("invoice-created", topics.System, subscriber, func(msg *message.Message) error {
    var ev billingevents.InvoiceCreated
    if err := json.Unmarshal(msg.Payload, &ev); err != nil { return err }
    return svc.OnInvoiceCreated(msg.Context(), ev)
})`
- Always build with -tags=dynamic so librdkafka links.
- Use the named topic constant from openmeter/watermill, not string literals.
- Carry ctx through the consumer via msg.Context() - do not substitute context.Background().

### Billing and invoicing [domain]
Libraries: `GOBL`, `Ent`, `Goverter`, `Goderive`
Pattern: billing.Service is a composite interface (Profile, Invoice, Line, Sequence, etc.) implemented in openmeter/billing/service. openmeter/billing/charges owns charge lifecycle (usage-based, flat-fee, credit-purchase). openmeter/billing/worker runs auto-advance and collect loops; openmeter/billing/worker/subscriptionsync reconciles subscription views into invoice lines. Invoice format is GOBL.
Key files: `openmeter/billing/service.go`, `openmeter/billing/service/service.go`, `openmeter/billing/adapter/adapter.go`, `openmeter/billing/charges/service.go`, `openmeter/billing/worker/advance/advance.go`, `openmeter/billing/worker/subscriptionsync/service.go`, `openmeter/billing/rating/service.go`
Example: `// Drive a charge lifecycle through the service facade, not the adapter:
charge, err := charges.Create(ctx, charges.CreateInput{CustomerID: cid, Kind: charges.KindUsageBased})
if err != nil { return err }
if _, err := charges.AdvanceCharges(ctx, charges.AdvanceInput{CustomerID: cid, AsOf: now}); err != nil {
    return err
}`
- Test charge lifecycle through charges.Service.Create / AdvanceCharges / ApplyPatches, not via low-level adapters.
- Use MockStreamingConnector with explicit StoredAt to model late-arriving usage and exercise stored-at cutoff logic.
- For integration tests, use billing.BaseSuite + SubscriptionMixin.

### Credits feature flag [feature-flag]
Libraries: `Google Wire`
Pattern: credits.enabled must be honored at four independent layers: (1) app/common wires ledger services to noop when disabled; (2) api/v3/server credit handlers must skip registration; (3) customer ledger hooks must be unregistered; (4) namespace default-account provisioning must skip ledger account creation. A single guard is insufficient.
Key files: `app/config/config.go`, `app/common/customer.go`, `app/common/app.go`, `api/v3/server/server.go`, `openmeter/ledger/account/service.go`, `openmeter/namespace/namespace.go`
Example: `// In app/common/customer.go (conceptual):
func ProvideCustomerHooks(cfg config.Configuration, ledgerSvc ledger.Ledger) []customer.ServiceHook {
    hooks := []customer.ServiceHook{subjectHook}
    if cfg.Credits.Enabled {
        hooks = append(hooks, ledgerHook(ledgerSvc))
    }
    return hooks
}`
- Any new code that touches ledger_accounts or ledger_customer_accounts must have a credits.enabled=false path that does nothing.
- When writing a backfill that genuinely needs ledger writes, build the concrete adapters directly instead of relying on DI defaults (they are noops).
- Add a credits-disabled integration test that asserts no ledger table rows are produced.

### Webhook delivery (Svix) [webhook]
Libraries: `Svix Go SDK`
Pattern: openmeter/notification manages channels, rules, events, and delivery status. notification.EventHandler runs Dispatch + Reconcile loops. The Watermill consumer in openmeter/notification/consumer subscribes to the system events topic, builds the payload, and sends it through Svix. Payload version is pinned per event family.
Key files: `openmeter/notification/service.go`, `openmeter/notification/eventhandler.go`, `openmeter/notification/consumer/consumer.go`, `openmeter/notification/webhook/handler.go`, `cmd/notification-service/main.go`
Example: `// Consumer dispatches to Svix:
func (c *Consumer) onInvoiceCreated(ctx context.Context, ev billingevents.InvoiceCreated) error {
    payload := notification.InvoicePayloadV1{ /* ... */ }
    return c.dispatcher.Dispatch(ctx, notification.Event{Type: notification.TypeInvoiceCreated, Payload: payload})
}`
- Pin payload version constants per event family and treat them as API contracts.
- Reconcile loop is responsible for retrying failed deliveries; don't duplicate retry logic inline.
- Boot order: register notification handlers before initNamespace when the default namespace needs them.

### Observability (tracing + metrics) [observability]
Libraries: `OpenTelemetry`
Pattern: Every entry point (HTTP handlers, Kafka consumers, Ent adapters) is instrumented. ingestadapter wraps openmeter/ingest.Collector with OTel. watermill router construction adds OTel tracing. All domain calls must propagate ctx (no context.Background() / TODO()).
Key files: `openmeter/ingest/ingestadapter`, `openmeter/watermill/router`, `app/common/telemetry.go`
Example: `// Ensure ctx is threaded through from HTTP handler or Kafka consumer:
func (s *svc) DoWork(ctx context.Context, id string) error {
    ctx, span := s.tracer.Start(ctx, "svc.DoWork")
    defer span.End()
    return s.adapter.Write(ctx, id)
}`
- Never substitute context.Background() to work around a missing ctx plumbing issue; fix the caller.
- In tests, use t.Context() instead of context.Background() when testing.T is available.