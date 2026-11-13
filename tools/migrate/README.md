## Schema source

Atlas runs `cmd/entschema`, which uses the reusable `entschema.GenerateSQL`
exporter with a schema path, PostgreSQL version, and supplemental DDL.
[`supplemental_schema.sql`](supplemental_schema.sql) defines deferred foreign keys
and indexes on implicit join tables that Ent cannot express.

## View SQL Helper

Generate SQL definitions for `ent.View` schemas:

```bash
make generate-view-sql
```

`viewgen.GenerateSQL` returns PostgreSQL `CREATE VIEW` statements from EntSQL
annotations. The command adds the regeneration header and writes
`tools/migrate/views.sql`.
