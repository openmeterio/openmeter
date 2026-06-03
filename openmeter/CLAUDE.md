# openmeter

<!-- archie:ai-start -->

> The entire domain layer of OpenMeter: an organisational root holding every business-logic package (billing, customer, entitlement, subscription, credit, ledger, productcatalog, meter, ingest, sink, streaming, watermill, notification, app, namespace, and supporting domains) under a uniform service/adapter/httpdriver layering. No source lives directly here; its overriding constraint is that domain packages stay leaf nodes — never importing app/common or cmd/* — so they remain independently testable and wirable.

## Patterns

**Layered service/adapter/httpdriver split per domain** — Each domain exposes a Service interface and Adapter interface at the package root (service.go / adapter.go contract-only), a concrete service in service/, an Ent/PostgreSQL adapter in adapter/, and HTTP handlers in httpdriver/ or httphandler/. Business logic in service/, persistence in adapter/, HTTP at the edge. (`openmeter/billing/service.go (interface) -> service/service.go (impl) + adapter/adapter.go (Ent)`)
**TransactingRepo wraps every Ent adapter write** — Adapter method bodies wrap DB access with entutils.TransactingRepo / TransactingRepoWithNoValue so the ctx-bound transaction is honored; multi-step writes go through transaction.Run, and per-customer mutations take a pg_advisory_lock inside that transaction (billing, subscription, entitlement, credit, ledger). (`return entutils.TransactingRepo(ctx, a, func(ctx, tx *adapter) (*Entity, error) { return toDomain(tx.db.Entity.Create().Save(ctx)) })`)
**Input.Validate() at every service boundary; models.Generic*/ValidationIssue errors** — Every cross-boundary input implements Validate() (often with a compile-time models.Validator assertion) called before business logic; domain errors are models.Generic* sentinels or ValidationIssue so the HTTP layer maps them to correct RFC 7807 status codes rather than 500. (`if err := input.Validate(); err != nil { return nil, err }  // returns models.NewGenericValidationError(...)`)
**Tagged unions constructed only via constructors** — Polymorphic billing/catalog types (InvoiceLine, charges.Charge/ChargeIntent, RateCard, Price, EntitlementType) carry private discriminators and must be built with their constructors; struct literals leave the discriminator zero and accessors error. Adding a variant means updating MarshalJSON/UnmarshalJSON/Equal/Validate and every downstream type-switch. (`billing.NewStandardInvoiceLine(...) not billing.InvoiceLine{}`)
**Cross-domain callbacks via ServiceHooks / RequestValidatorRegistry registered in app/common** — Lifecycle reactions and pre-mutation guards (billing/ledger/subscription/entitlement reacting to customer or subject changes) register through models.ServiceHooks[T] or a RequestValidatorRegistry inside app/common provider side-effects — never by a domain importing another domain's service — which is what keeps the import graph acyclic. (`customerService.RegisterHooks(billingCustomerHook) // inside an app/common provider, with a re-entrancy skip context`)
**Async cross-binary events only via watermill eventbus; never raw Kafka** — Producers publish via eventbus.Publisher (routed to ingest/system/balance-worker topics by EventVersionSubsystem prefix), events implement marshaler.Event in CloudEvents 1.0, consumers build routers via router.NewDefaultRouter and drop unknown ce_types. The sink worker keeps strict three-phase flush ordering (ClickHouse -> Kafka offset commit -> Redis dedupe) for exactly-once. (`publisher.Publish(ctx, &billingevents.InvoiceCreated{...}) // never a raw topic string`)
**clock.Now() and caller ctx everywhere; credits.enabled noop-guarded** — Production code uses clock.Now() (not time.Now()) for deterministic billing tests and threads the caller ctx (never context.Background()). When credits.enabled=false, ledger services, customer ledger hooks, ChargesRegistry, and v3 credit handlers each independently wire noop implementations rather than nil. (`now := clock.Now(); if !creditsConfig.Enabled { return ledgernoop.AccountService{}, nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/` | Largest domain and the billing contract layer: composite billing.Service (12 sub-interfaces), billing.Adapter, InvoiceLine tagged-union, InvoicingApp plugin, with service/ (invoice state machine), adapter/, charges/, rating/, worker/ | Root is contract-only; construct InvoiceLine/Charge via constructors; mutate invoices only through transactionForInvoiceManipulation and the per-customer lock |
| `openmeter/productcatalog/` | Shared type contract between billing, entitlement, and subscription: polymorphic RateCard/Price/Discount/EntitlementTemplate/Phase value types plus feature/plan/addon entity aggregates | Validation and event publishing belong in the service/connector layer; EffectivePeriod-derived status changes only via Publish/Archive; bridge packages break import cycles |
| `openmeter/subscription/ + openmeter/entitlement/ + openmeter/credit/` | Subscription lifecycle (SubscriptionSpec mutated only via the AppliesToSpec patch system), three-sub-type entitlement orchestration, and credit grant/burn-down with the balance worker | Per-customer advisory lock inside transaction.Run on composite writes; truncate credit effective times to Granularity (time.Minute); go through the top-level Service, not sub-connectors |
| `openmeter/watermill/eventbus/eventbus.go` | Single Publisher facade routing events to ingest/system/balance-worker topics by EventVersionSubsystem prefix | Always publish via eventbus.Publisher; an EventName without a recognised prefix silently routes to SystemEventsTopic |
| `openmeter/sink/sink.go` | Kafka->ClickHouse sink worker with exactly-once three-phase flush and post-flush callbacks to the balance worker | Flush order must be ClickHouse BatchInsert -> Kafka offset commit -> Redis dedupe; FlushEventHandler runs in a goroutine, never synchronously |
| `openmeter/ledger/ (+ noop/)` | Double-entry ledger; TransactionInput built only via transactions.ResolveTransactions templates; noop/ supplies zero-value impls when credits.enabled=false | Never construct EntryInput/TransactionInput outside transactions/; wire ledgernoop.* (not nil, not real DB paths) when credits disabled |
| `openmeter/ent/entc.go` | Single codegen driver for all Ent-generated code in openmeter/ent/db/ and the view SQL side-output; schema is the single source of database truth | Never edit openmeter/ent/db/ or tools/migrate/views.sql; edit openmeter/ent/schema then run make generate |
| `openmeter/server/ + openmeter/streaming/connector.go` | Chi server mounting v1+v3 behind shared middleware and the streaming.Connector ClickHouse abstraction every usage-querying domain depends on | No business logic in server.go (delegate to httpdriver/router); use query-struct toSQL() with sqlbuilder.Var(), never raw SQL/fmt.Sprintf in Connector methods |

## Anti-Patterns

- Importing app/common or cmd/* from any openmeter/<domain>/ or testutils package — creates import cycles Wire cannot resolve
- Using time.Now() instead of clock.Now(), or context.Background()/TODO() instead of the caller ctx — breaks time freezing and drops the Ent tx and OTel spans
- Adding business logic or DB queries to the root service.go/adapter.go contract files instead of the service/ and adapter/ sub-packages
- Calling sub-type connectors, adapters, or a.db directly (bypassing the top-level Service, TransactingRepo, advisory lock, or eventbus) — yields partial writes, races, and misrouted events
- Registering ledger namespace handlers or customer ledger hooks, or returning raw fmt.Errorf for domain conditions, when noop/credits guards and models.Generic* errors are required

## Decisions

- **Domain packages are leaf nodes with no dependency on cmd/* or app/common** — Keeps each domain independently testable and wirable and prevents DI-graph complexity from leaking into business logic; cross-domain hooks are registered as app/common provider side-effects to avoid circular imports
- **ServiceHook / RequestValidatorRegistry pattern for all cross-domain reactions** — Billing, ledger, subscription, and entitlement must react to customer/subject lifecycle without importing each other; registries fan out at wire time and break the import cycles a direct dependency would create
- **credits.enabled guarded at four independent wiring layers rather than one central check** — Credit writes fan out from HTTP handlers, customer hooks, namespace provisioning, and charge creation across unrelated call graphs, so no single choke point can gate every write path — each layer wires a noop independently

## Example: New domain adapter method with full TransactingRepo transaction awareness

```
// openmeter/<domain>/adapter/adapter.go
import (
    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
    "github.com/openmeterio/openmeter/pkg/framework/entutils"
)

func (a *adapter) Create(ctx context.Context, in domain.CreateInput) (*domain.Entity, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*domain.Entity, error) {
        row, err := tx.db.Entity.Create().
            SetNamespace(in.Namespace).
            SetName(in.Name).
            Save(ctx)
        if err != nil { return nil, err }
        return toDomain(row), nil
    })
// ...
```

<!-- archie:ai-end -->
