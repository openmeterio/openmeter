# OpenMeter

OpenMeter is a metering and billing platform with usage based pricing and access control.

## Tips for working with the codebase

If during your work anything confuses you or something isn't trivial for you, please augment AGENTS.md with your findings so next time it will be easier for you. AGENTS.md files are for you to edit and update as you go so you can interact with the codebase the most effectively.

Development commands are run via `Makefile`, it contains all commonly used commands during development. `Dagger` and `justfile` are also present but seldom used. Use the Makefile commands for common tasks like running tests, generating code, linting, etc.

## AGENTS.md maintenance

- Treat this file as long-lived project guidance for all agents and contributors.
- Prefer durable wording over time-based wording (avoid labels like "recent", "latest", "today").
- Keep entries actionable and specific (what to do, where, and why), not conversational history.
- When adding new guidance, fold it into the most relevant section and remove/merge stale or duplicate notes.

## Testing

To run all tests, invoke `make test` or `make test-nocache` if you want to bypass the test cache.

When running tests for a single file or testcase (invoking directly and not with Make), make sure the environment is set correctly. Examples of a correct setup can be found in the `Makefile`'s `test` command, or in `.vscode/settings.json` `go.testEnvVars`. Example command would be:

E2E tests are run via `make etoe`, they are API tests that need to start dependencies via docker compose, always invoke them via Make.

## Code Generation

Some directories are generated from code, never edit them manually. A non-exhaustive list of them is:
- `make gen-api`: generates from TypeSpec
  - the clients in `api/client`
  - the OAPI spec in `api/openapi.yaml`
- `make generate`: runs go codegen steps
  - database access in `**/ent/db` from the ent schema in `**/ent/schema`
  - dependency injection with wire in `**/wire_gen.go` from `**/wire.go`
- `atlas migrate --env local diff <migration-name>`: generates a migration diff from changes in the generated ent schema (in `tools/migrate/migrations`)

## Ledger gotchas

- For Postgres-backed tests run directly with `go test` (not via Make), ensure the PG env is set (e.g. `POSTGRES_HOST=localhost`). If tests rely on DB constraints/triggers, run real migrations (not just `Schema.Create`).
- For `CreateEntries`, prefer Ent `CreateBulk` path (not driver-level `ExecContext`) and map DB trigger violations to validation errors (`ledger_entries_dimension_ids_fk` / SQLSTATE 23503 path).
- When adding/adjusting historical adapter tests, follow existing repo conventions (`NewTestEnv`, `DBSchemaMigrate`, `t.Cleanup(env.Close)`), and use migration-backed schema so trigger-based constraints are actually exercised.

## Ent gotchas

- For Postgres `text[]` columns in Ent, avoid `field.Strings(...).SchemaType(map[string]string{dialect.Postgres: "text[]"})` in this codebase/version when you need native array behavior. `SchemaType` changes DDL, but does not change the runtime encode/decode path for `field.Strings`.
- Prefer `field.Other(..., pgtype.TextArray{})` for native Postgres array encode/decode with Ent methods (create/query/update) when working with `text[]`.
- If you change Ent schema types, regenerate Ent code (`go generate ./openmeter/ent` or `make generate`) before running tests.
- For custom Ent selects/joins, prefer the generated selected-value parsing helpers from the `entselectedparse` extension (`openmeter/ent/db/selectedparse.go`) instead of hand-written scanners.
- Use `db.Parse<Entity>FromSelectedValues(prefix, row.Value)` for aliased joined columns (example: `db.ParseLedgerDimensionFromSelectedValues("dimension_", row.Value)`).

