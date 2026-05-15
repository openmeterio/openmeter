# Enforcement: layering (7 rules)

Topic file. Loaded on demand when an agent works on something in the `layering` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `layer-007-adapter-triad` — Every new domain adapter must implement the TxCreator + TxUser triad: Tx(ctx) using HijackTx + NewTxDriver, WithTx(ctx, tx) using NewTxClientFromRawConfig, and Self(). Omitting any method prevents TransactingRepo from rebinding to caller-supplied transactions.

*source: `deep_scan`*

**Why:** The persistence implementation guideline states: 'Adapters under openmeter/<domain>/adapter/ implement TxCreator (Tx via *entdb.Client.HijackTx + entutils.NewTxDriver) and TxUser[T] (WithTx via entdb.NewTxClientFromRawConfig, Self) triad. Every method body wraps with entutils.TransactingRepo or TransactingRepoWithNoValue so it rebinds to the ctx-bound transaction.' A new adapter missing Self() panics when TransactingRepo falls back to it; missing WithTx() prevents participation in caller transactions.

**Example:**

```
type adapter struct{ db *entdb.Client }

// All three methods are required:
func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
    txCtx, cfg, drv, err := a.db.HijackTx(ctx, &sql.TxOptions{})
    return txCtx, entutils.NewTxDriver(drv, cfg), err
}
func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
    return &adapter{db: entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig()).Client()}
}
func (a *adapter) Self() *adapter { return a }
```

**Path glob:** `openmeter/**/adapter/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "entutils\\.TransactingRepo"
    ],
    "must_not_match": [
      "func.*Self\\(\\)"
    ]
  }
]
```

</details>

### `layer-004` — The Service interface for each domain must be defined at the package root (service.go or <domain>.go). Never define the service interface inside the service/ or adapter/ sub-packages.

*source: `deep_scan`*

**Why:** Each domain exposes a Go interface (e.g. billing.Service, customer.Service) defined in <domain>/service.go. A concrete service struct in <domain>/service/ holds business logic and calls an Adapter interface for all DB access. The Adapter interface is defined alongside the Service interface and implemented by Ent-backed structs in <domain>/adapter/ sub-packages.

**Example:**

```
// Correct structure:
// openmeter/billing/service.go         — interface definition
// openmeter/billing/adapter.go         — adapter interface
// openmeter/billing/service/service.go — concrete implementation
// openmeter/billing/adapter/adapter.go — Ent-backed implementation
```

### `layer-006` — v1 API endpoints must only be added in api/spec/packages/legacy/; v3 API endpoints only in api/spec/packages/aip/. Never mix v1 and v3 TypeSpec definitions in the same package.

*source: `deep_scan`*

**Why:** Route and tag bindings are declared only in root openmeter.tsp files, not in domain sub-folder operation files. api/spec/packages/aip/src/openmeter.tsp is the v3 AIP TypeSpec entry point; api/spec/packages/legacy/src/main.tsp is the v1 TypeSpec entry point. Mixing them prevents independent versioning.

## Mechanical Violations (block)

### `layer-001` — Domain packages under openmeter/<domain>/ must expose a Service interface at the package root (service.go or <domain>.go). Ent/PostgreSQL access must live in the adapter/ sub-package. HTTP translation must live in httpdriver/ or httphandler/.

*source: `deep_scan`* · *scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** The layered Service/Adapter/HTTP pattern is enforced throughout the codebase to keep persistence concerns from leaking into business logic and HTTP concerns from leaking into adapters. Violating this makes the domain package difficult to test in isolation and creates transitive import cycles.

### `layer-002` — Google Wire provider sets must live exclusively in app/common/. cmd/* binaries must only call the Wire-generated initializeApplication. Business logic must not be placed in cmd/*/main.go.

*source: `deep_scan`* · *scope: `cmd/`* · *check: `architectural_constraint`*

**Why:** Wire provider sets in app/common are the single compile-time-verified wiring graph. Placing providers or constructor logic in cmd/* creates duplicate, unverifiable graphs. Business logic in cmd/*/main.go cannot be tested without running the full binary and is invisible to Wire's dependency analysis.

## Tradeoff Signals (warn)

### `layer-003` — Service input types must use the <Verb><Noun>Input naming pattern (e.g. CreateCustomerInput, ListInvoicesInput) and implement models.Validator.

*source: `deep_scan`* · *scope: `openmeter/`* · *check: `architectural_constraint`*

**Why:** Consistent input type naming enables code search, review, and tooling to locate all inputs for a domain. Implementing models.Validator ensures validation is collocated with the type and runs before service logic, preventing invalid inputs from reaching the adapter or database.

## Pattern Divergence (inform)

### `layer-005` — v3 API handlers must be organized under api/v3/handlers/<resource>/ sub-packages and implement the generated ServerInterface methods using httptransport.Handler[Request,Response]. Do not place v3 handlers in openmeter/*/httpdriver/.

*source: `deep_scan`*

**Why:** v3 API handler packages are organized per resource group under api/v3/handlers/ (meters, customers, customers/billing, customers/charges, billingprofiles, plans, subscriptions, etc.). Each sub-package implements relevant ServerInterface methods using the httptransport.Handler[Request,Response] pipeline and delegates to domain services. v1 handlers live in openmeter/*/httpdriver/ or openmeter/*/httphandler/.

**Example:**

```
// v3 handler in api/v3/handlers/customers/handler.go
func (h *Handler) ListCustomers(ctx context.Context, req api.ListCustomersRequestObject) (api.ListCustomersResponseObject, error) {
    items, err := h.svc.ListCustomers(ctx, customer.ListCustomersInput{Namespace: req.Params.Namespace})
    if err != nil { return nil, err }
    return api.ListCustomers200JSONResponse{Items: toAPI(items)}, nil
}
```
