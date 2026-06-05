# Enforcement: layering (11 rules)

Topic file. Loaded on demand when an agent works on something in the `layering` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dec-api-surface-001` — v3 handlers must be thin delegators to shared domain services, never carry their own domain logic

*source: `deep_scan`*

**Why:** v3 is the forward AIP-style surface; v1 cannot be dropped without breaking clients, so both surfaces front the identical domain services and share error rendering (commonhttp.HandleErrorIfTypeMatches, api/v3/apierrors). A v3 handler with its own domain logic, or a service call duplicated across both surfaces with diverging behavior, makes the two transport layers drift. Both v1 httpdriver packages and v3 handlers must delegate to the same openmeter/<domain> service.

### `dec-style-001` — Keep usage metering in ClickHouse/Kafka and control-plane state in Postgres; do not collapse them

*source: `deep_scan`*

**Why:** OpenMeter splits high-volume append-only usage metering (ClickHouse MergeTree + Kafka + sink-worker) from ACID control-plane billing (PostgreSQL + Ent + lockr) because read-heavy ingest/aggregation and transaction-heavy billing have opposing scaling and failure profiles. Putting usage events in Postgres, using a single monolith binary, or a microservice-per-domain-with-own-DB layout all break the root constraint of providing event-time metering AND ACID usage-based billing over one shared codebase.

## Pitfalls (block)

### `pf-v3-unimplemented-001` — Do not leave v3 operations advertised as stable while their Server method only delegates to api.Unimplemented

*source: `deep_scan`*

**Why:** Pitfall pf_0015: TypeSpec generates each v3 operation's server-interface method and SDK client method from the contract independently of whether the server implements it, and the v3 Server can satisfy the interface by delegating to api.Unimplemented{} (api.gen.go:7186), which always returns 501. Because Go only checks interface satisfaction, a permanently-stubbed operation compiles identically to a finished one, so the advertised contract and the three SDKs diverge from runtime capability with no compile-time signal.

## Tradeoff Signals (warn)

### `tr-monolith-001` — Keep one Go module and one Ent client; do not split go.mod per binary or give a domain its own database

*source: `deep_scan`*

**Why:** Six binaries over one module: a change to a high-fan-in magnet (pkg/models 229 in-edges, productcatalog 104, customer 103) ripples across every binary, but the payoff is one codebase, one Ent client, one transaction boundary for cross-domain atomicity. Splitting go.mod per binary, duplicating pkg/models types per service, using a separate database per domain, or introducing a distributed transaction/saga between binaries breaks the shared-transaction guarantee that subscription→billing→charges→ledger relies on.

## Pattern Divergence (inform)

### `place-domain-iface-001` — Declare Service/Adapter/Connector interfaces and value models in the domain root package, implementations in nested subpackages

*source: `deep_scan`*

**Why:** Domain root packages (openmeter/<domain>/<domain>.go, service.go, connector.go) declare interfaces (Service/Adapter/Connector/Repository) and value models; implementations live in nested service/ and adapter/ subpackages. openmeter/billing/invoice.go declares billing.Service; openmeter/streaming/connector.go declares streaming.Connector. Confirmed across billing, customer, streaming, notification, entitlement.

### `place-service-001` — Concrete services live in a service/ subpackage with a Service struct, Config, Config.Validate(), and New(Config)

*source: `deep_scan`*

**Why:** Concrete services live in a service/ subpackage, take a Config struct of injected dependencies, validate each is non-nil in Config.Validate(), and assert interface satisfaction via `var _ billing.Service = (*Service)(nil)`. Loggers are injected, never slog.Default(). openmeter/billing/service/service.go and openmeter/notification/service/service.go follow this shape.

**Example:**

```
var _ billing.Service = (*Service)(nil)
func New(c Config) (*Service, error) { if err := c.Validate(); err != nil { return nil, err }; return &Service{...}, nil }
```

### `place-adapter-001` — Ent persistence lives in an adapter/ subpackage holding *entdb.Client with Tx/WithTx/Self

*source: `deep_scan`*

**Why:** Ent persistence is isolated in adapter/ packages that hold *entdb.Client and implement transaction hijacking (Tx returns a transaction.Driver; WithTx rebuilds the adapter from the tx config; Self returns the non-tx instance). Confirmed in billing/adapter/adapter.go and customer/adapter.

### `place-v3-handler-001` — v3 HTTP handlers live under api/v3/handlers/<resource>/ with a Handler interface and per-operation files

*source: `deep_scan`*

**Why:** The v3 API centralizes handlers under api/v3/handlers/, one package per resource (nested for sub-resources e.g. customers/charges, customers/credits). Each exposes a Handler interface returning httptransport handlers via New(resolveNamespace, service, options...); routes in api/v3/server/routes.go delegate via .With(params).ServeHTTP. Per-operation files are create.go/get.go/list.go/delete.go/upsert.go. Confirmed in customers/handler.go + routes.go.

### `place-legacy-httpdriver-001` — Legacy v1 HTTP handlers are co-located per domain as httpdriver/ or httphandler/ packages

*source: `deep_scan`*

**Why:** The legacy v1 API keeps HTTP handlers co-located with each domain as httpdriver/httphandler packages (openmeter/customer/httpdriver, openmeter/meter/httphandler, openmeter/notification/httpdriver), assembled in openmeter/server/router/router.go. Confirmed by router.go imports.

### `name-pkg-alias-001` — Implementation subpackages physically named service/adapter declare a domain-prefixed package name

*source: `deep_scan`*

**Why:** Implementation subpackages are physically named service/adapter but declare a domain-prefixed package name (package billingservice in service/service.go, package billingadapter in adapter/adapter.go) so call sites read unambiguously. Confirmed in billing service.go and adapter.go.

### `name-iface-assert-001` — Assert compile-time interface conformance with var _ <Interface> = (*<Struct>)(nil)

*source: `deep_scan`*

**Why:** Compile-time interface conformance is asserted with the blank-identifier pattern at the top of each implementation file (var _ billing.Service = (*Service)(nil), var _ billing.Adapter = (*adapter)(nil)). Confirmed in billing service.go, adapter.go, notification service.go.

**Example:**

```
var _ notification.Service = (*Service)(nil)
```
