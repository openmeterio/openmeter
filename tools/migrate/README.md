## View SQL Helper

Generate SQL definitions for `ent.View` schemas:

```bash
make generate-view-sql
```

This writes `tools/migrate/views.sql` by loading `openmeter/ent/schema` via Ent's schema loader and emitting Postgres `CREATE VIEW` statements from `EntSQL` view annotations.

## pg-schema-diff proof of concept

Generate a review-only SQL plan from the current migration history to the
desired Ent schema, generated namespace foreign keys, and Ent-managed views:

```bash
make pgschema-diff-poc
```

The command creates and drops disposable databases on `PG_SCHEMA_DIFF_DEV_DSN`.
It does not mutate an existing application database, but the configured role
must be allowed to create databases. Set `PG_SCHEMA_DIFF_OUTPUT` to write the
plan to a file instead of stdout. The migration state table (`schema_om`) is
removed from both disposable databases before comparison, so it cannot become
part of the generated application migration. The rendered SQL also removes the
`public` schema qualification to match the existing migration style.

This is intentionally a proof of concept. The output is not added to the
migration directory, no down migration is generated, and `atlas.sum` is not
updated. The generated plan is replay-validated against another disposable
database by default; pass `-skip-plan-validation` directly to the command only
when investigating pg-schema-diff behavior. Review the emitted statements and
hazard comments before using them in a migration.
