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

- **Do NOT** use `field.Strings(...)` for Postgres array columns. Without an explicit schema type it creates `jsonb`, and with `SchemaType(map[string]string{dialect.Postgres: "text[]"})` it changes DDL without changing the runtime encode/decode path for `field.Strings`.
- **Prefer** `field.Other(..., pq.StringArray{}).SchemaType(map[string]string{dialect.Postgres: "text[]"})` for native Postgres `text[]` encode/decode with Ent methods (create/query/update). Import `github.com/lib/pq` in the schema file.

## Custom Selects & Joins

- For custom Ent selects/joins, prefer the generated selected-value parsing helpers from the `entselectedparse` extension (`openmeter/ent/db/selectedparse.go`) instead of hand-written scanners.
- Use `db.Parse<Entity>FromSelectedValues(prefix, row.Value)` for aliased joined columns.
  - Example: `db.ParseLedgerDimensionFromSelectedValues("dimension_", row.Value)`

## Common Patterns

- **Soft-delete unique indexes** include `deleted_at` in the unique constraint (e.g., `index.Fields("namespace", "key", "deleted_at").Unique()`) — always filter with `Where(<entity>db.DeletedAtIsNil())` in queries.
- **Foreign keys** use `char(26)` schema type to match ULID IDs.
- **Cascade deletes** use `entsql.OnDelete(entsql.Cascade)` on the parent edge.
- **PostgreSQL identifier length** is 63 bytes by default (PostgreSQL docs, “Lexical Structure” / `NAMEDATALEN`). Long Ent-generated table, index, and FK names can truncate and collide even when their full names differ. When a schema/entity/edge name is verbose, proactively shorten generated FK symbols with `StorageKey(edge.Symbol("..."))` and shorten index names with `StorageKey("...")` before generating migrations.
- **JSONB fields** use `entutils.JSONStringValueScanner` — see `openmeter/ent/schema/llmcostprice.go`.
- **Non-empty strings at the DB layer**: `field.String(...).NotEmpty()` enforces Ent-side validation, but Atlas may still diff only `SET NOT NULL` for existing tables. If the database must reject empty strings too, add an explicit `entsql.Checks(...)` annotation in the schema or mixin alongside `NotEmpty()`.
- **Upserts with nullable/optional fields**: `UpdateNewValues()` only updates fields that were set on the create mutation. If an upsert must clear a previously set nullable/optional column, explicitly chain the generated `Update<Field>()` method for that field after `UpdateNewValues()` (for example `UpdateDescription()` or `UpdateDeletedAt()`). This lets Ent use the excluded insert value, including `NULL`, instead of leaving the old value untouched. See billing adapter upserts such as `openmeter/billing/adapter/stdinvoicelines.go`.
- **Generated `SetOrClear<Field>` helpers**: prefer these helpers for nullable/optional update fields when they have a straightforward signature. For awkward generated signatures such as double-pointer JSON fields, prefer the explicit pattern used for fields like normalized metadata: `if value != nil { update = update.Set<Field>(value) } else { update = update.Clear<Field>() }`. This avoids passing the address of a nil pointer, which can make Ent treat the field as set with a nil value and then panic or fail in generated validators.

## Regeneration

Depending on the change, the generators need to be re-run. For schema changes, edit files under `openmeter/ent/schema/` and run the repo generation target:

```bash
# Regenerate all generated Go code, including Ent
make generate
```

During local iteration, you can use the narrower Ent-only command when you intentionally want to avoid a full generation run:

```bash
# Regenerate Ent only after schema changes
go generate ./openmeter/ent/...
```
