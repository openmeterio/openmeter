## Pitfalls

- **Adapter helpers that operate on the raw *entdb.Client instead of going through entutils.TransactingRepo silently fall off any ctx-bound Ent transaction, producing partial writes under concurrency with no error surface.**
  - *Evidence:* pkg/framework/entutils/transaction.go:199-221 TransactingRepo reads *TxDriver from ctx and degrades to repo.Self() (non-tx client) when none is present, returning no error.; openmeter/billing/charges/adapter/CLAUDE.md: 'Never pass a.db directly to a helper - always go through a tx *adapter inside TransactingRepo.'; openmeter/billing/charges/service.go Create / AdvanceCharges / ApplyPatches drive multi-step writes mixing reads, realization runs, and lockr advisory locks inside one ctx-carried transaction.; … (4 total)
  - *Root cause:* The 'entutils.TransactingRepo context-propagated Ent transactions' pattern propagates Ent transactions implicitly through ctx; TransactingRepo's graceful fallback to Self() means a missing wrapper is undetectable at compile time. Multi-step charge advancement orchestrates many helpers, so one un-wrapped helper produces partially-applied state.
  - *Fix direction:*
    - Wrap every adapter helper body in entutils.TransactingRepo or TransactingRepoWithNoValue, including helpers that accept *entdb.Client.
    - Add a golangci-lint analyzer or CI grep that flags a.db. usage in adapter method bodies without TransactingRepo on the call stack.
    - Add an integration test per public charges.Service method asserting atomic commit/rollback.
    - Codify the rule in the /charges skill and PR review template.
- **Cross-cutting feature flags (credits.enabled) enforced by fan-out noop guards across multiple independent Wire layers can be silently re-enabled by any new provider that forgets one guard.**
  - *Evidence:* app/common/ledger.go, app/common/customer.go, app/common/billing.go each independently check creditsConfig.Enabled and return noop implementations.; api/v3/server credit handlers must additionally skip registration when credits are disabled.; AGENTS.md: 'credits.enabled needs explicit guarding at multiple layers ... wired separately.'; … (4 total)
  - *Root cause:* The 'Google Wire DI with noop-for-disabled-features' pattern handles a feature that cross-cuts HTTP handlers, customer hooks, namespace provisioning, and charge creation. The architecture chose fan-out noop guards over a single runtime check, so the guard set is only as complete as the most recently added provider's author remembered to make it; Wire verifies types, not the policy that every ledger-writer provider has a creditsConfig.Enabled branch.
  - *Fix direction:*
    - Enumerate every Wire provider injecting a ledger writer and verify each has a creditsConfig.Enabled branch.
    - Add a credits-disabled integration test asserting zero rows in ledger_accounts and ledger_customer_accounts after representative flows.
    - Add a Wire provider unit test asserting every noop type is wired when credits are disabled.
    - Make the credits guard a P0 review item in the PR template for any ledger-touching provider.
- **Sequential timestamped migration filenames plus a linear atlas.sum hash chain guarantee unmergeable conflicts between any two branches that both add migrations.**
  - *Evidence:* tools/migrate/migrations/ uses timestamped .up.sql/.down.sql files plus an atlas.sum hash chain.; atlas.hcl pins the migrations directory and the ent schema source.; AGENTS.md references a /rebase skill specifically for 'sequential migrations, atlas.sum conflicts, Ent regeneration'.; … (4 total)
  - *Root cause:* The Ent+Atlas migration decision picks deterministic linear migration ordering with cryptographic chain integrity for reviewability and safety, at the cost of guaranteed merge friction proportional to branch lifetime.
  - *Fix direction:*
    - On rebase, delete the branch's migration files and atlas.sum lines, re-run make generate, then atlas migrate --env local diff <name>.
    - Rebase long-lived schema-changing branches daily.
    - Adopt a 'regenerate the migration last, just before merge' policy for schema branches.
    - Follow the /rebase skill checklist.
- **EventName() string-prefix routing in eventbus.GeneratePublishTopic defaults unrecognized prefixes to SystemEventsTopic, so a misnamed or new event family silently misroutes instead of failing fast.**
  - *Evidence:* openmeter/watermill/eventbus/eventbus.go:141-142 default case returns SystemEventsTopic for any unrecognized EventName prefix.; openmeter/watermill/eventbus/CLAUDE.md and openmeter/watermill/CLAUDE.md both warn that events without a recognised EventVersionSubsystem prefix silently route to SystemEventsTopic.; Topic isolation between ingest, system, and balance-worker is the basis of the multi-binary scaling decision.
  - *Root cause:* The 'Kafka + Watermill pub/sub with three prefix-routed topics' pattern chose a catch-all default so genuine system events need no explicit declaration, but it makes a typo or missing EventVersionSubsystem constant indistinguishable from an intentional system event, and misrouted events bypass topic isolation.
  - *Fix direction:*
    - Replace the default-case fallback with an explicit registry of recognised prefixes; unknown prefix returns an error.
    - Add a unit test scanning all marshaler.Event implementations and asserting each EventName starts with a registered prefix.
    - Promote SystemEvents to an explicit subsystem constant so the default case requires an explicit 'system' declaration.
    - Publish the recognised subsystem prefixes as public constants in openmeter/watermill/eventbus.
- **Cross-domain hook and validator registration implemented as side-effects inside Wire provider functions is invisible to Wire's compile-time graph; omitting a hook provider from a binary's wire.Build silently drops the hook with no compile error.**
  - *Evidence:* app/common/customer.go:73 customerService.RegisterHooks(h) runs inside the NewCustomerLedgerServiceHook provider; same shape in NewCustomerSubjectServiceHook and NewCustomerEntitlementValidatorServiceHook.; app/common/billing.go NewBillingRegistry calls customerService.RegisterRequestValidator and subscriptionServices.Service.RegisterHook as construction side-effects.; Wire models types, not side-effects, so a binary that builds customer.Service without the hook providers compiles cleanly and ships with missing hooks.; … (4 total)
  - *Root cause:* The 'ServiceHook Registry for cross-domain lifecycle callbacks' pattern was moved into app/common provider side-effects to break circular imports between billing, customer, subscription, and ledger. Wire's compile-time guarantee covers type satisfaction only, so the side-effect-on-construction pattern is opaque to both the compiler and reviewers, and a provider runs only if some dependency transitively needs its output type.
  - *Fix direction:*
    - Audit each binary's wire.go for the expected hook/validator providers and document the expected list in cmd/<binary>/CLAUDE.md.
    - Promote hook bundles into named registry types (CustomerHookRegistry) so a missing registry is a compile error.
    - Add a per-binary integration smoke test asserting the registered hook count matches the expected set.
    - Consider migrating to explicit registration where main.go calls RegisterHooks(app.CustomerHooks...).
- **Namespace handlers (Ledger, KafkaIngest) are registered only by cmd/server, while worker binaries perform namespace-scoped provisioning inline assuming the default namespace already exists, creating an unenforced cross-binary boot-order contract.**
  - *Evidence:* cmd/server/main.go registers app.LedgerNamespaceHandler and app.KafkaIngestNamespaceHandler before initNamespace; cmd/server/CLAUDE.md: 'cmd/server is the only binary that registers namespace handlers.'; cmd/billing-worker/main.go:86 calls EnsureBusinessAccounts(ctx, GetDefaultNamespace()) and SandboxProvisioner without ever calling NamespaceManager.RegisterHandler.; openmeter/namespace/namespace.go:105-118 createNamespace fans out to handlers and joins with errors.Join (no short-circuit), and requires RegisterHandler before CreateDefaultNamespace - enforced nowhere.; … (4 total)
  - *Root cause:* The 'Namespace Manager fan-out (multi-tenancy provisioning)' pattern plus the multi-binary decision that each binary self-provisions only what it owns designates cmd/server the sole namespace-handler registrant, but workers still run namespace-scoped provisioning inline. The fan-out via errors.Join means partial provisioning does not block startup, and nothing in code or the Helm chart enforces cmd/server-before-workers ordering, so a worker booting first against a fresh database operates on an unprovisioned namespace.
  - *Fix direction:*
    - Encode cmd/server-before-workers ordering in deploy/charts/openmeter via init containers or Helm hook weights.
    - Add a fail-fast precondition check in each worker that the default namespace and its subsystems exist before accepting work.
    - Extract post-Migrate provisioning into a shared idempotent function any role-matching binary can run.
    - Add a boot-order integration test booting workers against an empty database.
- **Watermill consumers silently drop unknown CloudEvents ce_types, which is correct for forward-compatible rolling deploys but indistinguishable from a known event whose payload version the consumer cannot decode.**
  - *Evidence:* openmeter/watermill/grouphandler/grouphandler.go:48-54 returns nil for any ce_type not in typeHandlerMap.; openmeter/watermill/eventbus/eventbus.go:135-143 routes only by EventVersionSubsystem prefix; payload version is not part of routing.; Notification payload version constants live in producer packages, not derived from the TypeSpec contract.; … (4 total)
  - *Root cause:* The 'NoPublishingHandler silent-drop dispatch by CloudEvents ce_type' pattern serves rolling-deploy compatibility, but the codebase has no separate channel for 'known event, unknown version' - a version mismatch surfaces only as an absence of events, not as an error or DLQ entry, because version is not part of routing or dispatch.
  - *Fix direction:*
    - Distinguish unknown-event-type (silent drop) from known-event-type/unknown-version (DLQ) inside grouphandler.
    - Pin webhook payload versions to TypeSpec models and add a per-event-family contract test.
    - Catalog every payload version constant across notification producers.
    - Document the EventVersionSubsystem prefixes and payload-version policy in openmeter/watermill/eventbus/CLAUDE.md.
- **Helper and callback functions whose signatures omit context.Context force callers to substitute context.Background(), severing cancellation, deadlines, OTel spans, and the ctx-bound Ent transaction driver from operations that are logically request- or message-scoped.**
  - *Evidence:* openmeter/server/server.go:213 substitutes context.Background() in the v1 OapiRequestValidatorWithOptions ErrorHandler whose signature lacks *http.Request.; openmeter/app/stripe/client/appclient.go:240 calls UpdateAppStatus with context.Background() from providerError, which takes no ctx parameter (appclient.go:227).; AGENTS.md: 'Do not introduce context.Background() or context.TODO() to sidestep missing context propagation in application code.'; … (4 total)
  - *Root cause:* When an internal helper or a third-party-library callback signature does not carry context.Context (the dual-API kin-openapi ErrorHandler shape; stripeAppClient.providerError), the path of least resistance is context.Background() rather than refactoring the signature. This drops tracing, cancellation, and - for any path that reaches an Ent adapter via the TransactingRepo pattern - the implicit transaction, making writes silently non-transactional and untraced.
  - *Fix direction:*
    - Add context.Context parameters to internal helpers (e.g. stripeAppClient.providerError) and thread the caller's ctx through.
    - For third-party callback signatures missing ctx (kin-openapi ErrorHandler), capture the request in the enclosing closure and use r.Context().
    - Add a golangci-lint rule flagging new context.Background()/context.TODO() outside main() and Shutdown handlers.
    - Document the two legitimate exceptions (root context in main(), post-cancel graceful shutdown) in AGENTS.md.
- **v3 AIP list endpoints expose case-insensitive contains/ocontains filters that compile to leading-wildcard ILIKE, but the Ent schemas back the filtered columns with plain btree (no pg_trgm GIN) indexes, so every such filtered list request degrades to a full sequential scan plus a COUNT(*) scan.**
  - *Evidence:* openmeter/ent/schema/customer.go:42-55 TODO(DoS hardening) documents that name/primary_email/key contains filters compile to ILIKE '%value%' which cannot use the btree indexes and run full seq scans plus a COUNT(*) scan from query.Paginate.; openmeter/ent/schema/customer.go:64-65 declares only plain btree index.Fields("name") and index.Fields("primary_email"); no pg_trgm GIN extension or index exists.; pkg/filter/filter.go:63 ContainsPattern builds the leading-wildcard pattern, filter.go:241 maps $contains to sql.FieldContainsFold (ILIKE), and customer/adapter/customer.go:59 applies it to the indexed column.; … (4 total)
  - *Root cause:* The 'TypeSpec single-source API generation (v1 + v3 + three SDKs)' decision lets the v3 AIP contract expose powerful contains/ocontains operators, and the shared pkg/filter compiles them to leading-wildcard ILIKE (ContainsPattern). Postgres cannot serve a leading wildcard from a btree index, but Ent schemas default to btree and pg_trgm GIN indexes are not generated by Atlas - so any filterable text column is exposed at the API layer faster than it is indexed at the persistence layer, with no single owner reconciling the two.
  - *Fix direction:*
    - For every v3 list field exposing contains/ocontains, add a custom SQL migration creating pg_trgm and a GIN index on lower(col) gin_trgm_ops WHERE deleted_at IS NULL.
    - Add a CI/query-plan check asserting filtered list queries use an index rather than a seq scan.
    - Establish a rule that any new filterable text column must ship its trigram index in the same PR as its filter exposure.
    - Keep parser-side caps (maxCommaSeparatedItems, maxFilterValueLength) and rate limits until indexes land.
- **Cross-aggregate references in the ledger (LedgerCustomerAccount.account_id/customer_id) and ClickHouse usage tables are deliberately FK-less and migration-less, so referential integrity and column/struct alignment are enforced only by application code with no database-level guard.**
  - *Evidence:* openmeter/ent/schema/ledger_customer_account.go:42 func Edges() returns nil; account_id and customer_id are field.String(...).Immutable() with no FK to LedgerAccount or Customer (intentional, to avoid import cycles).; openmeter/streaming/clickhouse/connector.go:78 calls createTable only when !SkipCreateTables, and event_query.go:25 uses sb.IfNotExists() - the events table is create-if-absent with no diff/migration path.; openmeter/streaming/connector.go:24-29 RawEvent struct columns must be kept in sync with the ClickHouse DDL and INSERT column list by hand.; … (4 total)
  - *Root cause:* Two decisions converge: the layered service/adapter import-cycle-avoidance keeps the ledger link table FK-less (Edges() returns nil), and the multi-store decision (Postgres via Ent+Atlas for domain, ClickHouse via create-if-not-exists for usage) leaves ClickHouse outside the migration pipeline. Neither path has a database-level integrity or schema-diff guard, so a deleted account leaving dangling LedgerCustomerAccount rows, or a RawEvent struct change not mirrored in the DDL/INSERT list, drifts undetected until a runtime read fails.
  - *Fix direction:*
    - Add application-level integrity checks/tests asserting every LedgerCustomerAccount.account_id and customer_id resolves to a live row, and cover account/customer deletion paths.
    - Introduce a ClickHouse migration mechanism (or an ALTER-on-startup reconcile) so RawEvent column changes propagate to already-provisioned tables.
    - Add a contract test asserting the RawEvent struct field set equals the createEventsTable DDL column set and the INSERT column list.
    - Document the FK-less and migration-less invariants in the relevant CLAUDE.md so future schema edits know the integrity burden is on application code.
- **An in-progress billing-invoice schema migration leaves deprecated columns and temporary versioning artifacts live in the database, creating coupled cleanup debt across the Ent schema and the billing adapter that must be removed in lockstep once migration completes.**
  - *Evidence:* openmeter/ent/schema/billing.go:416 field.String("line_ids").Deprecated("invoice discounts are deprecated, use line_discounts instead"); billing.go:631 BillingInvoiceSplitLineGroup tax_config/tax_code_id/tax_behavior Deprecated; billing.go:794 BillingInvoiceLineDiscount type/quantity/pre_line_period_quantity Deprecated.; openmeter/ent/schema/billing.go:1170-1171 field.Int("schema_level").Default(1) // schema level for writing invoice data (until invoice migrations are complete).; openmeter/ent/schema/billing.go:1360-1374 BillingInvoiceWriteSchemaLevel is a temporary single-row table tracking the schema level for billing invoices.; … (4 total)
  - *Root cause:* Stems from the Ent-schema-as-source-of-truth persistence decision under an incremental invoice-line migration: rather than a single breaking cutover, the schema carries deprecated columns plus a schema_level discriminator (column + BillingInvoiceWriteSchemaLevel table) so the adapter can write old and new shapes during the transition. The same versioning decision is encoded in two places (per-row schema_level and a standalone table), so the cleanup is a multi-artifact, schema-plus-adapter operation that can be left half-done if the migration stalls.
  - *Fix direction:*
    - Track the invoice-line migration to completion with an explicit checklist of deprecated columns and the schema_level artifacts to drop.
    - When schema_level reaches a single value everywhere, remove the field, the BillingInvoiceWriteSchemaLevel table, and the adapter dual-write branches together in one migration.
    - Generate the Atlas down-migration and verify it before dropping columns, given the atlas.sum linear-chain constraint.
    - Add a test asserting no live billing row references a deprecated column after cutover.

## Error Mapping

| Error | Status Code |
|-------|------------|
| `models.GenericConflictError` | 409 |
| `models.GenericForbiddenError` | 403 |
| `models.GenericNotImplementedError` | 501 |
| `models.GenericValidationError` | 400 |
| `models.GenericNotFoundError` | 404 |
| `models.GenericUnauthorizedError` | 401 |
| `models.GenericPreConditionFailedError` | 412 |
| `ValidationIssue with WithHTTPStatusCodeAttribute` | -1 |
| `context canceled (substring match)` | 408 |
| `unrecognized error (no encoder matches)` | 500 |