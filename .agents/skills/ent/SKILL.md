---
name: ent
description: Work with Ent ORM schemas and generated code. Use when modifying ent schemas, debugging ent query issues, or dealing with Postgres type mappings.
user-invocable: false
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Ent ORM

Guidance for working with Ent in the OpenMeter codebase.

## Schema Location

- **Schema definitions:** `openmeter/ent/schema/*.go` (source of truth)
- **Generated code:** `openmeter/ent/db/` (DO NOT edit manually)

After any schema change, regenerate with `make generate` before running tests.

## Postgres Array Columns (`text[]`)

- **Do NOT** use `field.Strings(...).SchemaType(map[string]string{dialect.Postgres: "text[]"})` for native array behavior. `SchemaType` changes DDL but does not change the runtime encode/decode path for `field.Strings`.
- **Prefer** `field.Other(..., pgtype.TextArray{})` for native Postgres array encode/decode with Ent methods (create/query/update).

## Custom Selects & Joins

- For custom Ent selects/joins, prefer the generated selected-value parsing helpers from the `entselectedparse` extension (`openmeter/ent/db/selectedparse.go`) instead of hand-written scanners.
- Use `db.Parse<Entity>FromSelectedValues(prefix, row.Value)` for aliased joined columns.
  - Example: `db.ParseLedgerDimensionFromSelectedValues("dimension_", row.Value)`

## Common Patterns

- **Soft-delete unique indexes** include `deleted_at` in the unique constraint (e.g., `index.Fields("namespace", "key", "deleted_at").Unique()`) — always filter with `Where(<entity>db.DeletedAtIsNil())` in queries.
- **Foreign keys** use `char(26)` schema type to match ULID IDs.
- **Cascade deletes** use `entsql.OnDelete(entsql.Cascade)` on the parent edge.
- **JSONB fields** use `entutils.JSONStringValueScanner` — see `openmeter/ent/schema/llmcostprice.go`.

## Regeneration

```bash
# Regenerate all ent code
make generate

# Or just ent specifically
go generate ./openmeter/ent
```
