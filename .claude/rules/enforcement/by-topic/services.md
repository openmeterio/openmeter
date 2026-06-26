# Enforcement: services (12 rules)

Topic file. Loaded on demand when an agent works on something in the `services` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Decision Violations (block)

### `dec-wire-001` — Wire new services, hooks, and workers through Google Wire provider sets in app/common, not manual or runtime DI

*source: `deep_scan`*

**Why:** Each binary has a build-tagged wire.go (//go:build wireinject) listing wire.NewSet provider sets; constructors are plain functions grouped per concern in app/common/*.go. make generate runs wire to emit wire_gen.go. Compile-time DI means a missing provider fails the build, and each binary only links the providers it declares. Runtime/reflection DI and manual wiring in main() are rejected because the graph (billing→charges→ledger→customer→productcatalog) is too large to wire by hand without divergence.

**Example:**

```
var Customer = wire.NewSet(NewCustomerService)
```

### `dec-hook-001` — Use the ServiceHookRegistry for synchronous in-process cross-domain reactions, not Kafka or a static cross-domain call

*source: `deep_scan`*

**Why:** Some cross-domain effects must be synchronous and in-process (provisioning a ledger account when a customer is created), unlike the asynchronous Kafka event bus used for billing/notification. A service embeds models.ServiceHookRegistry[T] and exposes RegisterHooks(...ServiceHook[T]); Wire providers construct the hook and call targetService.RegisterHooks(h) at startup, registering a Noop hook when the feature is off. Routing a transactional side effect through Kafka loses same-transaction commit guarantees; hard-coding the dependent call would create a static dependency edge and prevent the noop-when-disabled swap.

### `dec-entitlement-subtype-001` — Add entitlement subtype behavior via a SubTypeConnector and a getTypeConnector switch arm, not inline branching

*source: `deep_scan`*

**Why:** entitlement.SubTypeConnector is implemented three times (metered/static/boolean, each in its own connector.go). The aggregate service holds all three and getTypeConnector(typed) switches over the closed EntitlementType set with a default-error arm (openmeter/entitlement/service/service.go:424). A new EntitlementType value MUST get a SubTypeConnector field plus a switch arm or it falls through to the default error. Inline switching over entitlement type everywhere, or open/registry-based dispatch, breaks the sealed-enum contract.

### `dec-app-registry-001` — Register marketplace app integrations once at wiring time via RegisterMarketplaceListing, not via a central switch or late registration

*source: `deep_scan`*

**Why:** The app adapter holds registry map[AppType]RegistryItem. Each integration's constructor calls AppService.RegisterMarketplaceListing(RegistryItem{Listing, Factory}) exactly once at wiring time; RegisterMarketplaceListing rejects duplicate AppTypes and validates the listing (openmeter/app/adapter/marketplace.go:121). The map has no late-registration locking, so all listings must register during DI before the HTTP/worker surface is live. Capability support is discovered by type-asserting the Factory to the requested install interface.

## Tradeoff Signals (warn)

### `tr-flag-001` — Gate features at the DI wiring seam with concrete-or-noop providers, not with if feature.enabled inside service logic

*source: `deep_scan`*

**Why:** Feature subsystems are wired as concrete-or-noop at the DI seam (credits.enabled, webhooks), so disabling a feature requires every layer's provider to honor the flag (api/v3 handlers, customer ledger hooks, namespace provisioning). Putting `if credits.enabled` inside a service method, constructing ledger adapters when credits are disabled, or building a Svix client when webhooks are off scatters runtime conditionals through service logic and leaves gated subsystems half-disabled.

**Path glob:** `openmeter/**/service/**`, `api/v3/handlers/**`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "if\\s+\\w*\\.?[Cc]redits\\w*\\.Enabled",
      "if\\s+\\w*[Ww]ebhook\\w*\\.Enabled"
    ],
    "must_not_match": [
      "_test\\.go"
    ]
  }
]
```

</details>

## Pattern Divergence (inform)

### `dec-ledger-backfill-001` — Ledger backfills that must write real ledger_accounts rows must construct concrete ledger adapters, not rely on default DI

*source: `deep_scan`*

**Why:** When credits.enabled is false, app/common wires ledger account services/resolvers to noop implementations. Any ledger account backfill that must write real ledger_accounts / ledger_customer_accounts rows needs to construct concrete ledger account + resolver adapters directly instead of relying on the default DI outputs, which would silently no-op the writes.

### `place-wire-001` — Google Wire provider sets live in app/common, one wire.NewSet file per subsystem

*source: `deep_scan`*

**Why:** Google Wire provider sets live in app/common, one file per subsystem (app/common/billing.go declares var Billing = wire.NewSet(...)); per-binary Application structs are assembled in openmeter_<binary>.go files. Wire-generated output is wire_gen.go (DO NOT EDIT). Confirmed by billing.go.

### `name-config-001` — Services/adapters take a Config struct, validate it in Config.Validate(), and construct via New(config)

*source: `deep_scan`*

**Why:** Services/adapters take a Config struct, validate each dependency non-nil in Config.Validate(), and return via New(config) (billingadapter.New(Config), notification service New(Config)). Constructors reject invalid config rather than panicking. Confirmed in billing adapter.go and service.go and notification service.go.

### `prac-slog-001` — Require and inject *slog.Logger via Config; never fall back to slog.Default() in production code

*source: `deep_scan`*

**Why:** In production constructors/initialization, always require and inject a *slog.Logger explicitly via Config; never fall back to slog.Default(). AGENTS.md mandates this; the service/adapter Config.Validate() requires a non-nil logger. slog.Default() hides the dependency and yields uncontrolled logging configuration.

**Path glob:** `openmeter/**`, `app/**`, `pkg/**`

<details><summary>Code-shape trigger</summary>

```json
[
  {
    "kind": "regex_in_content",
    "must_match": [
      "slog\\.Default\\(\\)"
    ],
    "must_not_match": [
      "_test\\.go"
    ]
  }
]
```

</details>

### `prac-lo-001` — Use samber/lo and stdlib helpers instead of local ptr/must wrappers

*source: `deep_scan`*

**Why:** Use samber/lo helpers instead of local wrappers: slices.Clone for defensive copies, lo.ToPtr for pointer literals, lo.Must only for (value,err) panic-on-failure test setup. Never add local ptr/must/loPtr/loMust helpers when github.com/samber/lo already covers the need.

### `prac-helper-001` — Do not extract helpers that only wrap 2-4 trivial lines or pass through to another function

*source: `deep_scan`*

**Why:** Do not extract helpers that only wrap 2-4 trivial lines/guards without adding domain intent, and remove leftover pass-through wrappers that only call another function — call the underlying function directly. Prefer function names that explain the domain reason for the call over names that restate the implementation steps.

### `data-namespace-handler-001` — Register namespace handlers before initNamespace if they must provision the default namespace at startup

*source: `deep_scan`*

**Why:** cmd/server/main.go migrates the DB before creating the default namespace; register namespace handlers (app.LedgerNamespaceHandler, app.KafkaIngestNamespaceHandler) before initNamespace(...) if they must provision the default namespace during startup. Registering after initNamespace means the default namespace is created without that handler's provisioning.
