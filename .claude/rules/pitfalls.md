## Pitfalls

- **Adapter helpers that operate on the raw *entdb.Client instead of going through entutils.TransactingRepo silently fall off any ctx-bound Ent transaction, producing partial writes under concurrency with no error surface.**
  - *Evidence:* pkg/framework/entutils/transaction.go:199-221 TransactingRepo reads *TxDriver from ctx and degrades to repo.Self() (non-tx client) when none is present, returning no error.; openmeter/billing/charges/adapter/CLAUDE.md: 'Never pass a.db directly to a helper - always go through a tx *adapter inside TransactingRepo.'; openmeter/billing/charges/service.go Create / AdvanceCharges / ApplyPatches drive multi-step writes that mix reads, realization runs, and lockr advisory locks inside one ctx-carried transaction.; … (4 total)
  - *Root cause:* Ent transactions propagate implicitly through ctx and TransactingRepo's graceful fallback to Self() means a missing wrapper is undetectable at compile time. Multi-step charge advancement orchestrates many helpers, so one un-wrapped helper produces partially-applied state.
  - *Fix direction:*
    - Wrap every adapter helper body in entutils.TransactingRepo or TransactingRepoWithNoValue, including helpers that accept *entdb.Client.
    - Add a golangci-lint analyzer or CI grep that flags a.db. usage in adapter method bodies without TransactingRepo on the call stack.
    - Add an integration test per public charges.Service method asserting atomic commit/rollback.
    - Codify the rule in the /charges skill and PR review template.
- **Cross-cutting feature flags (credits.enabled) enforced by fan-out noop guards across multiple independent Wire layers can be silently re-enabled by any new provider that forgets one guard.**
  - *Evidence:* app/common/ledger.go, app/common/customer.go, app/common/billing.go each independently check creditsConfig.Enabled and return noop implementations.; api/v3/server credit handlers must additionally skip registration when credits are disabled.; AGENTS.md: 'credits.enabled needs explicit guarding at multiple layers... wired separately.'; … (4 total)
  - *Root cause:* The credits feature cross-cuts HTTP handlers, customer hooks, namespace provisioning, and charge creation. The architecture chose fan-out noop guards over a single runtime check, so the guard set is only as complete as the most recently added provider's author remembered to make it.
  - *Fix direction:*
    - Enumerate every Wire provider injecting a ledger writer and verify each has a creditsConfig.Enabled branch.
    - Add a credits-disabled integration test asserting zero rows in ledger_accounts and ledger_customer_accounts after representative flows.
    - Add a Wire provider unit test asserting every noop type is wired when credits are disabled.
    - Make the credits guard a P0 review item in the PR template for any ledger-touching provider.
- **Sequential timestamped migration filenames plus a linear atlas.sum hash chain guarantee unmergeable conflicts between any two branches that both add migrations.**
  - *Evidence:* tools/migrate/migrations/ uses timestamped .up.sql/.down.sql files plus an atlas.sum hash chain.; atlas.hcl pins the migrations directory and the ent schema source.; AGENTS.md references a /rebase skill specifically for 'sequential migrations, atlas.sum conflicts, Ent regeneration'.; … (4 total)
  - *Root cause:* Deterministic linear migration ordering with cryptographic chain integrity is intentionally chosen for reviewability and safety, at the cost of guaranteed merge friction proportional to branch lifetime.
  - *Fix direction:*
    - On rebase, delete the branch's migration files and atlas.sum lines, re-run make generate, then atlas migrate --env local diff <name>.
    - Rebase long-lived schema-changing branches daily.
    - Adopt a 'regenerate the migration last, just before merge' policy for schema branches.
    - Follow the /rebase skill checklist.
- **EventName() string-prefix routing in eventbus.GeneratePublishTopic defaults unrecognized prefixes to SystemEventsTopic, so a misnamed or new event family silently misroutes instead of failing fast.**
  - *Evidence:* openmeter/watermill/eventbus/eventbus.go:141-142 default case returns SystemEventsTopic for any unrecognized EventName prefix.; openmeter/watermill/eventbus/CLAUDE.md and openmeter/watermill/CLAUDE.md both warn that events without a recognised EventVersionSubsystem prefix silently route to SystemEventsTopic.; Topic isolation between ingest, system, and balance-worker is the basis of the multi-binary scaling decision.
  - *Root cause:* The catch-all default was chosen so genuine system events need no explicit declaration, but it makes a typo or missing EventVersionSubsystem constant indistinguishable from an intentional system event, and misrouted events bypass topic isolation.
  - *Fix direction:*
    - Replace the default-case fallback with an explicit registry of recognised prefixes; unknown prefix returns an error.
    - Add a unit test scanning all marshaler.Event implementations and asserting each EventName starts with a registered prefix.
    - Promote SystemEvents to an explicit subsystem constant so the default case requires an explicit 'system' declaration.
    - Publish the recognised subsystem prefixes as public constants in openmeter/watermill/eventbus.
- **Cross-domain hook and validator registration implemented as side-effects inside Wire provider functions is invisible to Wire's compile-time graph; omitting a hook provider from a binary's wire.Build silently drops the hook with no compile error.**
  - *Evidence:* app/common/customer.go:73 customerService.RegisterHooks(h) runs inside the NewCustomerLedgerServiceHook provider; same shape in NewCustomerSubjectServiceHook and NewCustomerEntitlementValidatorServiceHook.; app/common/billing.go NewBillingRegistry calls customerService.RegisterRequestValidator and subscriptionServices.Service.RegisterHook as construction side-effects.; Wire models types, not side-effects, so a binary that builds customer.Service without the hook providers compiles cleanly and ships with missing hooks.; … (4 total)
  - *Root cause:* Hook registration was moved into app/common provider side-effects to break circular imports between billing, customer, subscription, and ledger. Wire's compile-time guarantee covers type satisfaction only, so the side-effect-on-construction pattern is opaque to both the compiler and reviewers.
  - *Fix direction:*
    - Audit each binary's wire.go for the expected hook/validator providers and document the expected list in cmd/<binary>/CLAUDE.md.
    - Promote hook bundles into named registry types (CustomerHookRegistry) so a missing registry is a compile error.
    - Add a per-binary integration smoke test asserting the registered hook count matches the expected set.
    - Consider migrating to explicit registration where main.go calls RegisterHooks(app.CustomerHooks...).
- **Watermill consumers silently drop unknown CloudEvents ce_types, which is correct for forward-compatible rolling deploys but indistinguishable from a known event whose payload version the consumer cannot decode.**
  - *Evidence:* openmeter/watermill/grouphandler/grouphandler.go:48-54 returns nil for any ce_type not in typeHandlerMap.; openmeter/watermill/eventbus/eventbus.go:135-143 routes only by EventVersionSubsystem prefix; payload version is not part of routing.; Notification payload version constants live in producer packages, not derived from the TypeSpec contract.; … (4 total)
  - *Root cause:* The silent-drop policy serves rolling-deploy compatibility, but the codebase has no separate channel for 'known event, unknown version' — a version mismatch surfaces only as an absence of events, not as an error or DLQ entry.
  - *Fix direction:*
    - Distinguish unknown-event-type (silent drop) from known-event-type/unknown-version (DLQ) inside grouphandler.
    - Pin webhook payload versions to TypeSpec models and add a per-event-family contract test.
    - Catalog every payload version constant across notification producers.
    - Document the EventVersionSubsystem prefixes and payload-version policy in openmeter/watermill/eventbus/CLAUDE.md.
- **Helper and callback functions whose signatures omit context.Context force callers to substitute context.Background(), severing cancellation, deadlines, OTel spans, and the ctx-bound Ent transaction driver from operations that are logically request- or message-scoped.**
  - *Evidence:* openmeter/server/server.go:213 substitutes context.Background() in the v1 OapiRequestValidatorWithOptions ErrorHandler whose signature lacks *http.Request.; openmeter/app/stripe/client/appclient.go:240 calls UpdateAppStatus with context.Background() from providerError, which takes no ctx parameter.; AGENTS.md: 'Do not introduce context.Background() or context.TODO() to sidestep missing context propagation in application code.'; … (4 total)
  - *Root cause:* When an internal helper or a third-party-library callback signature does not carry context.Context, the path of least resistance is context.Background() rather than refactoring the signature. This drops tracing, cancellation, and — for any path that reaches an Ent adapter — the implicit transaction, making writes silently non-transactional and untraced.
  - *Fix direction:*
    - Add context.Context parameters to internal helpers (e.g. stripeAppClient.providerError) and thread the caller's ctx through.
    - For third-party callback signatures missing ctx (kin-openapi ErrorHandler), capture the request in the enclosing closure and use r.Context().
    - Add a golangci-lint rule flagging new context.Background()/context.TODO() outside main() and Shutdown handlers.
    - Document the two legitimate exceptions (root context in main(), post-cancel graceful shutdown) in AGENTS.md.

## Error Mapping

| Error | Status Code |
|-------|------------|
| `models.GenericNotFoundError` | 404 |
| `models.GenericValidationError` | 400 |
| `models.GenericConflictError` | 409 |
| `models.GenericForbiddenError` | 403 |
| `models.GenericUnauthorizedError` | 401 |
| `models.GenericNotImplementedError` | 501 |
| `models.GenericPreConditionFailedError` | 412 |
| `models.GenericStatusFailedDependencyError` | 424 |
| `ValidationIssue with WithHTTPStatusCodeAttribute` | -1 |
| `context canceled (any error with 'context canceled' substring)` | 408 |
| `internal server error (unrecognized by any encoder)` | 500 |