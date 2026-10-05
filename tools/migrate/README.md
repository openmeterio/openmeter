## Schema source

Atlas loads Ent's schema through `entschema`, including deferred FK definitions
from `deferred_constraints.sql` that Ent cannot express.

## View SQL Helper

Generate SQL definitions for `ent.View` schemas:

```bash
make generate-view-sql
```

This writes `tools/migrate/views.sql` by loading `openmeter/ent/schema` via Ent's schema loader and emitting Postgres `CREATE VIEW` statements from `EntSQL` view annotations.
