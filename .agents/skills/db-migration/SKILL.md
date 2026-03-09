---
name: db-migration
description: Create or update database schema and generate migrations. Use when modifying ent schema, adding database fields/tables, or generating migration files.
user-invocable: true
argument-hint: "[description of schema change]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Database Schema Change & Migration

You are helping the user modify the OpenMeter database schema and generate a corresponding migration.

## Context

- **Schema files:** `openmeter/ent/schema/*.go` — ent schema definitions (source of truth)
- **Generated ent code:** `openmeter/ent/db/` — DO NOT edit manually
- **Migrations dir:** `tools/migrate/migrations/` — DO NOT edit manually
- **Always use `--env local`** — we do not use Atlas Cloud services

## Workflow

Follow these steps in order:

### Step 1: Modify the ent schema

Edit or create files in `openmeter/ent/schema/`. Look at existing schema files for conventions.

If the user described what change they want ($ARGUMENTS), implement it. Otherwise, ask what schema changes are needed. When creating a new schema always define schema to support soft delete.

Schemas supporting soft delete always have a `deleted_at` field.

### Step 2: Regenerate ent code

Run:

```bash
make generate
```

This runs `go generate ./...` which regenerates the ent client code in `openmeter/ent/db/` from the schema definitions. Check that it completes without errors.

### Step 3: Generate the migration diff

Run:

```bash
atlas migrate --env local diff <migration-name>
```

Where `<migration-name>` is a short descriptive snake_case name for the change (e.g., `add_customer_email`, `create_invoice_table`). Derive the name from the schema change being made.

This creates timestamped `.up.sql` and `.down.sql` files in `tools/migrate/migrations/` and updates `atlas.sum`.

### Step 4: Review the generated migration

Read the generated `.up.sql` file and verify:

- The SQL matches the intended schema change
- No unintended changes are included
- Indexes are created where appropriate

Present a summary of the migration to the user.

## Available Mixins

From `pkg/framework/entutils/mixins.go`:

| Mixin | Fields | Notes |
|-------|--------|-------|
| `entutils.IDMixin{}` | `id` char(26) ULID | Auto-generated, unique, immutable |
| `entutils.NamespaceMixin{}` | `namespace` string | Immutable, indexed |
| `entutils.TimeMixin{}` | `created_at`, `updated_at`, `deleted_at` (nillable) | Provides soft delete support |
| `entutils.MetadataMixin{}` | `metadata` JSONB `map[string]string` | Optional |
| `entutils.ResourceMixin{}` | ID + Namespace + Metadata + Time + `name` + `description` | Composite of above mixins |
| `entutils.UniqueResourceMixin{}` | Resource + `key` | Adds unique index on `(namespace, key, deleted_at)` |
| `entutils.KeyMixin{}` | `key` string | Immutable, not empty |
| `entutils.CadencedMixin{}` | `active_from`, `active_to` (nillable) | For time-bounded entities |

Usage in schema:

```go
func (<Entity>) Mixin() []ent.Mixin {
    return []ent.Mixin{
        entutils.IDMixin{},
        entutils.NamespaceMixin{},
        entutils.TimeMixin{},
    }
}
```

## Common Field Patterns

### JSONB custom type

```go
field.String("config").
    GoType(MyConfig{}).
    ValueScanner(entutils.JSONStringValueScanner[MyConfig]).
    SchemaType(map[string]string{dialect.Postgres: "jsonb"})
```

### Decimal

```go
field.Other("amount", alpacadecimal.Decimal{}).
    SchemaType(map[string]string{dialect.Postgres: "numeric"})
```

### Foreign key

```go
field.String("customer_id").
    SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
    Immutable()
```

### Enum-like string

```go
field.String("status").
    Default("active")
```

### Optional nillable

```go
field.Time("effective_to").
    Optional().
    Nillable()
```

## Edge (Relationship) Patterns

### One-to-many (parent → children)

```go
// In parent schema:
edge.To("children", Child.Type).
    Annotations(entsql.OnDelete(entsql.Cascade))

// In child schema:
edge.From("parent", Parent.Type).
    Ref("children").
    Unique().
    Required().
    Field("parent_id")
```

### Many-to-one foreign key (child → parent, no back-edge)

```go
// Define the FK field:
field.String("parent_id").
    SchemaType(map[string]string{dialect.Postgres: "char(26)"}).
    Immutable()

// Define the edge:
edge.To("parent", Parent.Type).
    Unique().
    Required().
    Field("parent_id")
```

## Index Patterns

### Soft-delete aware unique index

```go
// In Indexes():
index.Fields("namespace", "key", "deleted_at").Unique()
```

This ensures uniqueness among non-deleted records. Always filter with `Where(<entity>db.DeletedAtIsNil())` in queries.

### Composite index

```go
index.Fields("namespace", "provider", "model_id")
```

### GIN index for JSONB

```go
index.Fields("annotations").
    Annotations(
        entsql.IndexTypes(map[string]string{
            dialect.Postgres: "GIN",
        }),
    )
```

## Troubleshooting

### Rehashing migrations

If the `atlas.sum` file gets out of sync (e.g., after manually editing a migration file or resolving conflicts), rehash it:

```bash
atlas migrate --env local hash
```

### Dev database

Atlas uses a Docker-based dev database (`docker://postgres/15/dev`) for diffing. Make sure Docker is running before generating migrations.

### Migration format

Migrations are generated, never edit them manually.
Migrations use golang-migrate format. Each migration has:

- `<timestamp>_<name>.up.sql` — applied when migrating up
- `<timestamp>_<name>.down.sql` — applied when migrating down

## Important Reminders

- Always use `--env local` with atlas commands
- Never edit files in `openmeter/ent/db/` manually
- Never edit migration files in `tools/migrate/migrations` manually
- Run `make generate` before `atlas migrate diff` so the ent code is up to date
- If compilation errors occur after schema changes, fix the schema first, then re-run `make generate`
