# OpenMeter

OpenMeter is a usage metering and billing platform for AI and DevTool companies, built in Go.

## Quick Reference

| Task                        | Command                                    |
| --------------------------- | ------------------------------------------ |
| Start dependencies          | `make up`                                  |
| Stop dependencies           | `make down`                                |
| Run API server (hot reload) | `make server`                              |
| Run all tests               | `make test`                                |
| Run e2e tests               | `make etoe`                                |
| Generate all code           | `make generate-all`                        |
| Generate Go code only       | `make generate` (runs `go generate ./...`) |
| Generate API + SDKs         | `make gen-api`                             |
| Lint all                    | `make lint`                                |
| Lint Go only                | `make lint-go`                             |
| Format code                 | `make fmt`                                 |
| Tidy modules                | `make mod`                                 |
| Build all binaries          | `make build`                               |

## Architecture

Services live in `cmd/`: `server`, `sink-worker`, `balance-worker`, `billing-worker`, `notification-service`, `jobs`, `benthos-collector`. Each can be run individually via `make <service-name>` (uses `air` for hot reload).

Core business logic is in `openmeter/`, shared utilities in `pkg/`, API layer in `api/`.

**Stack:** Go + PostgreSQL (ent ORM) + Kafka + ClickHouse. API defined in TypeSpec, generated to OpenAPI.

## Code Generation

Some directories are **generated — never edit them manually**:

| Generated artifact                            | Source                                | Regenerate with |
| --------------------------------------------- | ------------------------------------- | --------------- |
| `api/openapi.yaml`, `api/openapi.cloud.yaml`  | TypeSpec in `api/spec/`               | `make gen-api`  |
| `api/client/javascript/`, `api/client/go/`     | OpenAPI spec                          | `make gen-api`  |
| `api/api.gen.go`, `api/v3/api.gen.go`          | OpenAPI spec via oapi-codegen         | `make generate` |
| `**/ent/db/`                                   | Ent schema in `openmeter/ent/schema/` | `make generate` |
| `**/wire_gen.go`                               | Wire providers in `**/wire.go`        | `make generate` |
| `**/generated/` (goverter)                     | Converter interfaces                  | `make generate` |

**Workflow for changing the API:**

1. Edit TypeSpec files in `api/spec/`
2. Run `make gen-api` to regenerate OpenAPI spec and SDKs
3. Run `make generate` to regenerate Go server/client code

**Workflow for changing Go types/DI:**

1. Edit the source files (ent schema, wire.go, converter interfaces)
2. Run `make generate` (or `go generate ./...`)

## Database Migrations

Uses [ent](https://entgo.io) for schema definition and [Atlas](https://atlasgo.io/) for migration generation. Migrations are in `tools/migrate/migrations/` using golang-migrate format.

**Schema files:** `openmeter/ent/schema/*.go`

**Workflow for schema changes:**

1. Edit the ent schema in `openmeter/ent/schema/`
2. Run `make generate` to regenerate ent code in `openmeter/ent/db/`
3. Generate migration: `atlas migrate --env local diff <migration-name>`
   - This creates timestamped `.up.sql` / `.down.sql` files in `tools/migrate/migrations/`
   - Also updates `tools/migrate/migrations/atlas.sum`
4. Migrations run automatically on startup when `postgres.autoMigrate` is set to `ent` (default for dev) or `migration`

**Atlas config:** `atlas.hcl` — schema source is `ent://openmeter/ent/schema`, migrations dir is `file://tools/migrate/migrations`.

**Local Postgres:** `postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable`

## Testing

Tests require PostgreSQL running locally. Start it with `docker compose up -d postgres`.

| Command              | Description                                          |
| -------------------- | ---------------------------------------------------- |
| `make test`          | Run all tests (parallel: `-p 128 -parallel 16`)     |
| `make test-nocache`  | Run tests bypassing cache                            |
| `make test-all`      | Run tests including Svix/Redis dependencies          |
| `make etoe`          | Run e2e tests (requires docker compose dependencies) |

**Build tag:** All Go commands use `-tags=dynamic` for confluent-kafka-go to link against local librdkafka.

**Environment for direct test runs:** Set `POSTGRES_HOST=127.0.0.1`. See `Makefile` test target or `.vscode/settings.json` for full env vars.

## Building

```bash
make build              # All binaries → build/
make build-server       # Just the server
```

All builds use `GO_BUILD_FLAGS=-tags=dynamic`.

## Configuration

- Copy `config.example.yaml` to `config.yaml` (done automatically by Make targets)
- Key settings: `postgres.url`, `postgres.autoMigrate`, `billing`, `notification`, meter definitions
- Make targets for running services will warn if `config.yaml` is outdated vs `config.example.yaml`

## Project Layout

```
cmd/                    # Service entrypoints
openmeter/              # Core business logic (billing, customer, entitlement, meter, etc.)
openmeter/ent/schema/   # Ent entity definitions (source of truth for DB schema)
openmeter/ent/db/       # Generated ent code (DO NOT EDIT)
api/                    # API specs, generated code, SDKs
api/spec/               # TypeSpec API definitions (source of truth for API)
pkg/                    # Shared utility packages
tools/migrate/          # Migration tooling and SQL migration files
e2e/                    # End-to-end tests
deploy/                 # Helm charts
docs/                   # Documentation and ADRs
```
