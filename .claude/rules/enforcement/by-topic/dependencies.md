# Enforcement: dependencies (7 rules)

Topic file. Loaded on demand when an agent works on something in the `dependencies` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dep-001` — Domain packages under openmeter/ must not import app/common. Wire wiring flows outward (app/common imports domain packages); reversing the direction creates import cycles and defeats the Wire compile-time graph.

*source: `deep_scan`* · *scope: `openmeter/`* · *check: `forbidden_import`*

**Why:** Domain packages under openmeter/ have no dependency on cmd/* or app/common (enforced by leaf-node import direction). Each cmd/<binary>/wire.go composes only the provider sets it needs. Cross-domain hook/validator registration done inside app/common avoids circular imports — if a domain package imported app/common the cycle would be unresolvable.

**Path glob:** `openmeter/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "\"github\\.com/openmeterio/openmeter/app/common\""
    ]
  }
]
```

</details>

### `dep-002` — Every new cmd/* worker binary must have a matching app/common/openmeter_<binary>.go Wire provider set file. Adding a binary without this file leaves its dependency graph unverified at compile time.

*source: `deep_scan`* · *scope: `app/common/`* · *check: `file_naming`*

**Why:** Each cmd/<binary> binary needs its own provider graph but every domain service must be wireable identically; Wire makes this compile-time checked. Workers added without matching app/common/openmeter_*worker.go Wire set are a documented violation signal.

**Example:**

```
// app/common/openmeter_billingworker.go
var BillingWorker = wire.NewSet(
    Billing,
    Charges,
    LedgerStack,
    // ...
)
```

### `dep-003` — Cross-domain hooks and request validators must be registered inside app/common provider functions, not inside domain package constructors. Domain packages must expose RegisterHooks/RegisterRequestValidator methods that app/common calls after wiring.

*source: `deep_scan`*

**Why:** Cross-domain hooks (billing → customer, ledger → customer, billing → subscription) would create circular imports if registered inside source packages. app/common/customer.go registers customerService.RegisterRequestValidator(validator) and customerService.RegisterHooks(ledgerHook, subjectHook) as side-effects of Wire provider functions to avoid circular imports.

**Example:**

```
// app/common/customer.go
func NewCustomerService(adapter customer.Adapter, ledgerHook customer.ServiceHook, ...) customer.Service {
    svc := customer.New(adapter)
    svc.RegisterHooks(ledgerHook, subjectHook)
    svc.RegisterRequestValidator(billingValidator)
    return svc
}
```

### `dep-005` — New billing backend integrations must implement billing.InvoicingApp and self-register via app.Service.RegisterMarketplaceListing() in their New() constructor. Never hardcode provider-specific logic inside billing.Service.

*source: `deep_scan`*

**Why:** The App Factory / Registry pattern keeps billing.Service decoupled from specific payment providers. Each app's New() self-registers a factory via app.Service.RegisterMarketplaceListing in service/factory.go. No billing service code changes are needed when adding a new backend.

**Example:**

```
// openmeter/app/stripe/service/factory.go
func New(appSvc app.Service, ...) (*StripeApp, error) {
    appSvc.RegisterMarketplaceListing(stripeMarketplaceListing, stripeFactory)
    return &StripeApp{}, nil
}
```

## Mechanical Violations (block)

### `build-001` — Always include -tags=dynamic in all Go build and test invocations. Omitting this tag causes confluent-kafka-go to fail to link against librdkafka.

*source: `deep_scan`* · *scope: `.`* · *check: `required_pattern`*

**Why:** confluent-kafka-go uses CGo with dynamic librdkafka linking. Without -tags=dynamic the build uses a stub that errors at link time. The Makefile sets GO_BUILD_FLAGS=-tags=dynamic for this reason; manual go build or go test commands must replicate this.

## Tradeoff Signals (warn)

### `dep-004` — Never spawn goroutines outside the oklog/run.Group in worker and server binaries. Goroutines spawned outside run.Group bypass graceful shutdown and can leak resources.

*source: `deep_scan`*

**Why:** Goroutine spawned outside run.Group (bypasses graceful shutdown) is an explicit violation signal for the multi-binary orchestration trade-off. cmd/server/main.go and all worker main.go files orchestrate lifecycle through an oklog/run.Group with explicit Start and Interrupt functions.

**Path glob:** `cmd/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "^\\s*go func\\("
    ],
    "must_not_match": [
      "run\\.Add",
      "run\\.Group"
    ]
  }
]
```

</details>

## Pattern Divergence (inform)

### `dep-006` — New charge type engines must be registered with billing.Service.RegisterLineEngine() in app/common/charges.go. Do not call RegisterLineEngine from domain packages or cmd/* binaries.

*source: `deep_scan`*

**Why:** billingservice.engineRegistry stores a map[LineEngineType]LineEngine under a RWMutex. Each charge type implements its own Engine and registers it at startup via app/common/charges.go. The service.New() constructor also pre-registers the standard invoice line engine. All engine registration is a side-effect of Wire provider functions.

**Example:**

```
// app/common/charges.go
func NewFlatFeeChargesService(billingService billing.Service, ...) flatfee.Service {
    svc := flatfee.New(...)
    billingService.RegisterLineEngine(svc) // side-effect registration
    return svc
}
```
