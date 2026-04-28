## Pitfalls

- **Charges adapter helpers that accept a raw *entdb.Client bypass the ctx-bound Ent transaction and produce partial writes under concurrency.**
  - *Evidence:* AGENTS.md explicit warning about TransactingRepo / TransactingRepoWithNoValue in openmeter/billing/charges/.../adapter.; openmeter/billing/charges/adapter/adapter.go and openmeter/billing/charges/adapter/CLAUDE.md require TransactingRepo on every method body and warn against passing a.db directly to helpers.; openmeter/billing/charges/service.go drives multi-step writes (Create / AdvanceCharges / ApplyPatches) that mix reads, realization runs, lockr advisory locks, and ledger-bound writes — partial writes are correctness-fatal.; … (4 total)
  - *Root cause:* Ent transactions propagate implicitly via ctx; the TransactingRepo wrapper is the only way to rebind the Ent client to the active transaction, and there is no compile-time enforcement that every adapter helper performs the rebind.
  - *Fix direction:*
    - Wrap every helper body in openmeter/billing/charges/**/adapter with entutils.TransactingRepo or TransactingRepoWithNoValue.
    - Codify this as a checklist item in the /charges skill and PR review template.
    - Add a custom golangci-lint analyzer (or grep CI check) that flags adapter methods using a.db directly without TransactingRepo on the call stack.
    - Add an integration test that opens a transaction, calls each public Service method, and asserts atomic commit/rollback.
- **Disabling credits at the config level does not fully stop ledger writes unless every wiring layer (ledger services, customer hooks, charges registry, namespace handler, v3 HTTP handlers) is independently guarded.**
  - *Evidence:* AGENTS.md identifies four distinct layers: api/v3/server credit handlers, customer ledger hooks, namespace/default-account provisioning, and app/common noop fallback.; app/common/ledger.go: NewLedgerAccountService, NewLedgerHistoricalLedger, NewLedgerResolversService, NewLedgerNamespaceHandler each independently check creditsConfig.Enabled.; app/common/customer.go NewCustomerLedgerServiceHook returns NoopCustomerLedgerHook independent of ledger.go's guards.; … (5 total)
  - *Root cause:* Credits feature is cross-cutting across HTTP handlers, customer hooks, charges, namespace provisioning, and ledger services. No single injection point dominates all call graphs that can reach ledger writes; any new ledger-touching provider added without an explicit guard re-introduces the bug.
  - *Fix direction:*
    - Enumerate every ledger write path (grep for ledger_accounts / ledger_customer_accounts touches) and trace each back to a Wire provider.
    - Verify every such provider has a creditsConfig.Enabled branch returning a noop type.
    - Add a smoke integration test that boots with credits.enabled=false and asserts ledger tables stay empty under representative flows.
    - When adding a new provider that writes to ledger, treat the credits guard as a P0 review item — codify in PR template.
- **The multi-generator toolchain (TypeSpec + Ent + Goverter + Wire + Goderive) is easy to leave partially regenerated, producing silent drift between specs, generated server stubs, ent code, and SDKs.**
  - *Evidence:* AGENTS.md generator table lists five independent generators feeding different artifacts: api/*.gen.go (oapi-codegen), openmeter/ent/db/ (ent), **/wire_gen.go (wire), **/convert.gen.go (goverter), billing/derived.gen.go (goderive).; AGENTS.md workflow mandates both `make gen-api` AND `make generate` for API changes — a single-step regen leaves SDKs out of sync.; cmd/*/wire_gen.go files must be regenerated whenever constructors change in app/common; missing this means runtime crashes from missing providers or stale interfaces.
  - *Root cause:* Several generators read different sources and write different outputs; a developer who runs only one of them leaves the repo in an inconsistent state that may compile but behave incorrectly. The two-step gen-api → generate cadence is documented but not enforced by tooling.
  - *Fix direction:*
    - Always run `make generate-all` after touching TypeSpec, Ent schema, Wire provider sets, Goverter interfaces, or Goderive-annotated code.
    - In CI, run all generators and fail if the working tree is dirty (already partially in place).
    - Document the generator graph as a dependency diagram in AGENTS.md and keep it in sync.
    - Add a pre-commit hook that runs `make generate-all` when any of api/spec/, openmeter/ent/schema/, app/common/*.go, or **/convert.go changes.
- **Sequential Atlas migration filenames + atlas.sum chain hashing guarantee merge collisions on long-lived feature branches.**
  - *Evidence:* AGENTS.md migration workflow (atlas migrate --env local diff) and the /rebase skill description explicitly mention 'sequential migrations, atlas.sum conflicts, Ent regeneration'.; tools/migrate/migrations/ holds timestamped .up.sql/.down.sql files plus atlas.sum.; atlas.sum records a linear hash chain over migration files; two branches both appending migrations produce two different hash chains that cannot merge cleanly.
  - *Root cause:* atlas.sum chain hashing enforces deterministic linear migration order; concurrent branch development that adds migrations cannot avoid collisions by design.
  - *Fix direction:*
    - On rebase, delete the branch's migration files and its atlas.sum lines.
    - Re-run `make generate` (Ent) then `atlas migrate --env local diff <name>` to regenerate the migration against the rebased schema.
    - Follow the /rebase skill checklist documented in AGENTS.md.
    - For long-lived branches with schema changes, consider rebasing daily to minimise the cost of regeneration.
- **Tests or build invocations that omit -tags=dynamic fail to link confluent-kafka-go against librdkafka, producing confusing link errors that look unrelated to Kafka.**
  - *Evidence:* Makefile sets GO_BUILD_FLAGS = -tags=dynamic for every build and test target.; AGENTS.md: 'Always build with -tags=dynamic to link confluent-kafka-go against the local librdkafka — omitting this tag causes build failures.'; Codex / agent shells often miss .envrc loading, leading to ad-hoc go test invocations that drop the tag.; … (4 total)
  - *Root cause:* confluent-kafka-go's dynamic linking against librdkafka is a CGO requirement that is enforced only at link time, not by Go modules. Each new test target or shell-based invocation must inherit the tag explicitly.
  - *Fix direction:*
    - Always run tests through `make test` / `make test-nocache` which set -tags=dynamic and POSTGRES_HOST.
    - If running ad-hoc, use `nix develop --impure .#ci -c go test -tags=dynamic ...` to inherit the project's pinned librdkafka.
    - Document the tag in any per-package CLAUDE.md that covers Kafka-touching code.
    - Consider a small wrapper script (tools/run-test.sh) that injects the tag if missing, to reduce footguns in CI matrix expansions.
- **Cross-domain hook/validator registration is implemented as side-effects inside Wire provider functions; omitting the provider in a binary's wire.Build silently drops the hook with no compile error.**
  - *Evidence:* app/common/customer.go NewCustomerLedgerServiceHook, NewCustomerSubjectServiceHook, NewCustomerEntitlementValidatorServiceHook each call customerService.RegisterHooks(h) inside the provider as side-effects.; app/common/billing.go NewBillingRegistry registers customerService.RegisterRequestValidator(validator) and subscriptionServices.Service.RegisterHook(subscriptionValidator) the same way.; Wire sees only types, not side-effects; a binary that does not pull NewCustomerLedgerServiceHook into its wire.Build still builds successfully but loses the ledger hook.; … (4 total)
  - *Root cause:* Hook registration is intentionally moved to app/common to avoid circular imports between domain packages, but the side-effect-on-construction pattern is opaque to Wire's compile-time graph and hence to reviewers.
  - *Fix direction:*
    - Audit each binary's wire.go for the expected hook providers and document the expected list in cmd/<binary>/CLAUDE.md.
    - Promote hook bundles into named registry types (CustomerHookRegistry) so a missing registry is a compile error.
    - Add an integration test per binary that asserts customerService.HookCount() (or similar) matches the expected count for that binary's role.
    - Consider migrating to an explicit-registration pattern where main.go (not wire) calls customerService.RegisterHooks(app.CustomerHooks...).
- **EventName prefix-based topic routing in eventbus.GeneratePublishTopic falls through to SystemEventsTopic by default; an event family that forgets to declare a recognised EventVersionSubsystem silently misroutes instead of failing fast.**
  - *Evidence:* openmeter/watermill/eventbus/eventbus.go GeneratePublishTopic: switch matches ingestVersionSubsystemPrefix and balanceWorkerVersionSubsystemPrefix; default returns SystemEventsTopic.; openmeter/watermill/eventbus/CLAUDE.md and openmeter/watermill/CLAUDE.md both flag this as a documented anti-pattern.; Topic isolation is the basis of the multi-binary scaling decision; misrouting an ingest event to the system topic can starve consumers and bypass DLQ semantics.
  - *Root cause:* Default-case fallback to SystemEventsTopic was a pragmatic choice for genuine system events but means no error fires for misnamed event families; the prefix match is structurally fragile against typos and missing constants.
  - *Fix direction:*
    - Replace the default-case fallback with a registry of recognised EventVersionSubsystem prefixes; unknown prefix returns an error.
    - Add a unit test that scans all marshaler.Event implementations in the codebase and asserts each EventName starts with a registered prefix.
    - Promote the SystemEvents subsystem to an explicit constant so the default-case is only hit when the event explicitly declares 'system' subsystem.
- **TypeSpec edits without a follow-up `make gen-api` + `make generate` produce silent drift between the API contract and Go server stubs / SDKs; the repo compiles but ships an outdated contract or SDK.**
  - *Evidence:* AGENTS.md workflow for API changes mandates: edit api/spec/, run make gen-api, run make generate.; Generator outputs (api/openapi.yaml, api/api.gen.go, api/v3/api.gen.go, api/client/go/client.gen.go, api/client/javascript/, Python SDK) are all .gen.go or generated trees that compile fine when stale.; Once Wave 1 documented `cmd/jobs/internal/wire.go` and the v3 handler pattern, multiple regen steps must run in a specific order or downstream code (handlers, SDK consumers) sees an inconsistent contract.
  - *Root cause:* TypeSpec is the source of truth, but the toolchain has an intermediate (OpenAPI YAML) plus several downstream generators (oapi-codegen for two server versions, JS SDK, Python SDK, Go SDK). Skipping any step leaves drift that compiles fine but misrepresents the contract.
  - *Fix direction:*
    - Always run `make generate-all` (which chains gen-api and generate) after editing api/spec/.
    - Add a CI check that runs `make generate-all` and fails if the working tree is dirty (catches missing regen).
    - Document the canonical workflow in api/spec/CLAUDE.md (edit → gen-api → generate).

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
| `billing.ValidationError / billing.NotFoundError / billing.UpdateAfterDeleteError` | 400 |
| `context canceled (any error with 'context canceled' substring)` | 408 |
| `internal server error (unrecognized by any encoder)` | 500 |