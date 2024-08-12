# Data Migrations in OpenMeter

OpenMeter uses [ent](https://entgo.io) for its data storage and schema management. Database state is synced from the ent schema definitions under `internal/ent/schema` via either `ent` schema upsertions or migrations.

## AutoMigrate

OpenMeter can automatically sync the database schema even in multi-instance deployments. This behavior can be configured via `postgres.autoMigrate` in the configuration.
- Choosing the calue of `ent` will internally call `ent.Schema.Create` which runs a schema upsertion. This is the default behavior, intended for development and testing.
- Choosing the value of `migration` will automatically execute the scripts in the `migrations` directory.
- Choosing the value of `false` will disable the automatic schema sync and lets the user manage the schema manually, for example `migrate -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable&x-migrations-table=schema_om" -path ./services/common/migrations/om up`

## Generating Migrations

OpenMeter uses [atlas](https://atlasgo.io/) to generate versioned migrations from changes in the ent schema.

After changing the schema and running `go generate` you can create a new migration diff via `atlas migrate --env local diff <migration-name>`, the generated migration files will be placed in the `migrations` directory.

## Running Migrations

The recommended way to run migrations is to use the `golang-migrate` CLI tool. You can run the migrations via
```bash
migrate -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable&x-migrations-table=schema_om" -path ./services/common/migrations/om up
```

OpenMeter `autoMigrate` uses the `schema_om` lock table.
