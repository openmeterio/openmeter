## Pitfalls

- **Charges adapter helpers accepting a raw *entdb.Client bypass the ctx-bound Ent transaction and produce partial writes under concurrency.**
  - *Evidence:* AGENTS.md explicit warning about TransactingRepo / TransactingRepoWithNoValue in openmeter/billing/charges/.../adapter.; openmeter/billing/charges/adapter.go; openmeter/billing/charges/service.go; charges.Service.Create / AdvanceCharges / ApplyPatches drive multi-write flows.
  - *Root cause:* Ent transactions propagate implicitly via ctx; any helper that uses the raw client instead of rebinding falls off the transaction.
  - *Fix direction:*
    - Wrap every such helper body with entutils.TransactingRepo / TransactingRepoWithNoValue.
    - Codify this as a checklist item in the /charges skill and PR review template.
- **Disabling credits at the config level does not fully stop ledger writes unless every wiring layer is guarded independently.**
  - *Evidence:* AGENTS.md calls out four distinct layers: api/v3/server ledger-backed credit handlers, customer ledger hooks, namespace/default-account provisioning, and app/common noop fallback.; app/common/customer.go wires customer service with ledger hooks; openmeter/namespace/namespace.go registers handlers including ledger.; openmeter/ledger/customerbalance and openmeter/ledger/account expose write paths.
  - *Root cause:* The credits feature is cross-cutting; no single injection point dominates all call graphs that can reach ledger writes.
  - *Fix direction:*
    - Enumerate every ledger write path (grep for ledger_accounts / ledger_customer_accounts touches).
    - Verify each path is behind a Wire-provided noop when credits.enabled=false.
    - Add a smoke test that boots with credits.enabled=false and asserts ledger tables stay empty.
- **The multi-generator toolchain (TypeSpec + Ent + Goverter + Wire + Goderive) is easy to leave partially regenerated, producing silent drift between specs, code, and SDKs.**
  - *Evidence:* AGENTS.md generator table lists five independent generators feeding different artifacts (api/*.gen.go, openmeter/ent/db/, **/wire_gen.go, **/convert.gen.go, billing/derived.gen.go).; AGENTS.md workflow mandates both `make gen-api` AND `make generate` for API changes.; cmd/*/wire_gen.go files must be regenerated whenever constructors change.
  - *Root cause:* Several generators read different sources and write different outputs; a developer who runs only one of them leaves the repo in an inconsistent state that may compile but behave incorrectly.
  - *Fix direction:*
    - Always run `make generate-all` after touching TypeSpec, Ent schema, Wire provider sets, Goverter interfaces, or Goderive-annotated code.
    - In CI, run the generators and fail if the working tree is dirty.
    - Document the generator graph in AGENTS.md (already partially present) and keep it in sync.
- **Sequential Atlas migration filenames + atlas.sum chain hashing guarantee merge collisions on long-lived feature branches.**
  - *Evidence:* AGENTS.md migration workflow (atlas migrate --env local diff) and the /rebase skill description explicitly mention 'sequential migrations, atlas.sum conflicts'.; tools/migrate/migrations/ holds timestamped .up.sql/.down.sql files plus atlas.sum.
  - *Root cause:* atlas.sum records a linear hash chain over migration files; two branches both appending migrations produce two different hash chains that cannot merge cleanly.
  - *Fix direction:*
    - On rebase, delete the branch's migration files and its atlas.sum lines.
    - Re-run `make generate` (Ent) then `atlas migrate --env local diff <name>` to regenerate the migration against the rebased schema.
    - Follow the /rebase skill checklist.

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