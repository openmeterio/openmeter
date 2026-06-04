# Enforcement: dependencies (13 rules)

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

### `deps-001-prefer-stdlib-samberlo` — Prefer stdlib and samber/lo helpers over local ptr/must/clone wrappers

*source: `deep_scan`*

**Why:** The project already depends on samber/lo v1.53.0 and modern stdlib; always prefer slices.Clone for defensive copies, lo.ToPtr for pointer literals, and other samber/lo helpers instead of introducing local ptr/loPtr/must/loMust wrapper functions. lo.Must is acceptable only in test setup, never in production code paths.

**Example:**

```
ptr := lo.ToPtr(value)
clone := slices.Clone(in)
```

**Path glob:** `openmeter/**/*.go`, `pkg/**/*.go`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "func\\s+(ptr|loPtr|toPtr|must|loMust)\\["
    ],
    "must_not_match": [
      "_test\\.go"
    ]
  }
]
```

</details>

### `infra-006-nvmrc-node-sync` — Keep .nvmrc in sync with the Nix .#ci shell node version

*source: `deep_scan`*

**Why:** flake.nix enterShell writes 'node -v > .nvmrc', and CI's 'Validate Node version file' step fails if .nvmrc differs from 'nix develop --impure .#ci -c node -v'. Hand-editing .nvmrc to a node version not pinned by the Nix shell breaks CI and produces non-reproducible TypeSpec/JS SDK builds.

### `infra-007-golangci-config` — Lint Go with the project golangci-lint v2 config, not an ad-hoc linter setup

*source: `deep_scan`*

**Why:** Go linting uses golangci-lint v2 with .golangci.yaml (errcheck/govet/staticcheck/sloglint/bodyclose/misspell/ineffassign/nolintlint/unconvert/whitespace) and formatters gci/gofumpt/goimports with prefix github.com/openmeterio/openmeter; collector/benthos/internal and examples are excluded. Running a different linter or ignoring these formatters produces import-ordering and formatting diffs that fail CI.

### `infra-008-collector-separate-module` — Keep the collector as a separate Go module with its own go.mod

*source: `deep_scan`*

**Why:** The Benthos collector is a separate Go module (collector/go.mod) that uses a replace directive pointing openmeter back at the parent module, and is built into its own benthos-collector Docker image with CGO_ENABLED=0. Adding collector code to the root module or removing the replace directive breaks the independent collector build and 'go mod tidy -C collector'.

**Path glob:** `collector/**/*.go`

### `infra-009-conventional-commits` — Write commitizen-conventional commit messages

*source: `deep_scan`*

**Why:** flake.nix git-hooks run commitizen/commitizen-branch via prek, and CI runs 'prek run -a' and 'prek run --stage manual' to validate commit messages. A commit message that is not commitizen-conventional (e.g. missing a type prefix like feat:/fix:/chore:) fails the local hook and the CI commit-message validation step.

### `infra-010-python-sdk-publish` — Publish the Python SDK via its dedicated make target, not Poetry directly

*source: `deep_scan`*

**Why:** The generated Python SDK under api/client/python is published via 'make -C api/client/python publish-python-sdk'; release.sh overwrites pyproject.toml version at publish time. Invoking poetry publish directly bypasses the version-stamping step and ships a mis-versioned package.
