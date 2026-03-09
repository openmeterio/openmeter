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

Edit or create files in `openmeter/ent/schema/`. Look at existing schema files for conventions:

- Mixins for common fields (namespace, timestamps, metadata)
- Index definitions
- Edge (relationship) definitions
- Field validations and defaults

If the user described what change they want ($ARGUMENTS), implement it. Otherwise, ask what schema changes are needed.

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
