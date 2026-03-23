## View SQL Helper

Generate SQL definitions for `ent.View` schemas:

```bash
make generate-view-sql
```

This writes `tools/migrate/views.sql` by loading `openmeter/ent/schema` via Ent's schema loader and emitting Postgres `CREATE VIEW` statements from `EntSQL` view annotations.
