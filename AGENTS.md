# AGENTS.md

OpenMeter is a billing and metering platform providing usage-based billing, real-time insights, and usage limit enforcement.

## Commands

```bash
make up              # Start dependencies (postgres, kafka, clickhouse)
make server          # Run API server with hot-reload
make test            # Run tests
make test-nocache    # Run tests bypassing cache
make lint            # Run linters
make fmt             # Auto-fix lint issues
make generate        # Regenerate ent schemas + wire DI
make gen-api         # Regenerate TypeSpec -> OpenAPI -> clients
```

Running tests directly:
```bash
TZ=UTC POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -run TestName ./path/to/package
```

## Generated Files - Do Not Edit

- `api/openapi.yaml`, `api/v3/openapi.yaml` - from TypeSpec (`api/spec/src/`)
- `api/api.gen.go`, `api/v3/api.gen.go` - from OpenAPI via oapi-codegen
- `api/client/` - SDK clients
- `openmeter/ent/db/` - from ent schemas (`openmeter/ent/schema/`)
- `cmd/*/wire_gen.go` - from Wire (`wire.go` files)

## Architecture

**Services** (`cmd/`): server, sink-worker, balance-worker, billing-worker, notification-service, benthos-collector, jobs

**Core packages** (`openmeter/`): billing, customer, entitlement, productcatalog, subscription, meter, streaming, ingest, notification, app

**API layers**: TypeSpec (`api/spec/src/`) → OpenAPI → Go handlers (`api/`, `api/v3/handlers/`)

**Storage**: PostgreSQL (ent ORM), ClickHouse (time-series), Kafka (events), Redis (cache)

**DI**: Google Wire - `wire.go` (providers) → `wire_gen.go` (generated)

## Go Build Tags

Required: `-tags=dynamic,wireinject`

## Database Migrations

```bash
atlas migrate --env local diff <migration-name>
```

Migrations: `tools/migrate/migrations/`