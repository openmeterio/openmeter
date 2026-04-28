## Tech Stack

- **API Spec:** TypeSpec 1.9.0, Prettier 3.8.2
- **Analytics Database:** ClickHouse 25.12.3-alpine
- **Auth:** golang-jwt v5.3.1
- **Authorization:** AuthZed / SpiceDB (authzed-go) v1.4.1
- **Backend Framework:** Chi v5.2.5, oapi-codegen v2.6.1 (pinned fork), kin-openapi v0.135.0
- **Cache:** Redis 7.4.7, go-redis v9.18.0
- **Code Generation:** Goverter v1.9.3, Goderive v0.5.1, oapi-codegen v2.6.1 (pinned fork), Ent codegen v0.14.6, Wire codegen v0.7.0
- **Collector:** Benthos (Redpanda Connect) v4.55.0 (benthos) + v4.61.0 (connect free)
- **Configuration:** Viper v1.21.0, Cobra v1.10.2
- **DI:** Google Wire v0.7.0
- **Database:** PostgreSQL 14.20-alpine3.23
- **Database Driver:** pgx v5.9.2
- **Database Migration:** Atlas CLI 0.36.0, golang-migrate v4.19.1
- **Database ORM:** Ent v0.14.6
- **Invoicing:** GOBL v0.400.1
- **Kubernetes:** controller-runtime v0.23.3
- **Linting:** golangci-lint config version: 2, Spectral CLI 6.13.1, Prettier 3.8.2
- **Messaging:** Confluent Kafka (confluent-kafka-go) v2.14.1, IBM Sarama v1.47.0, Watermill v1.5.1, watermill-kafka v3.1.2
- **Observability:** OpenTelemetry v1.43.0 (otel), v1.43.0 (metric/trace), Prometheus client v1.23.2
- **Payment:** Stripe v80.2.1
- **Runtime:** Go 1.25.5, Node.js v24.12.0, Python ^3.9, pnpm 10.33.0
- **Scheduling:** gocron v2.21.0
- **State Machine:** stateless v1.8.0
- **Testing:** testify v1.11.1, pgtestdb v0.1.1, gofakeit v6.28.0
- **Utilities:** samber/lo v1.53.0, CloudEvents SDK v2.16.2, oklog/ulid v2.1.1
- **Webhooks:** Svix server v1.84.1, Svix Go SDK v1.90.0

## Project Structure

```
openmeter/
├── cmd/                          # Service entrypoints
│   ├── server/                   # Main API server
│   ├── sink-worker/              # Kafka->ClickHouse sink
│   ├── balance-worker/           # Entitlement balance worker
│   ├── billing-worker/           # Billing lifecycle worker
│   ├── notification-service/     # Webhook/notification service
│   ├── jobs/                     # One-off job runner (admin CLI)
│   └── benthos-collector/        # Benthos pipeline collector
├── openmeter/                    # Core business logic
│   ├── billing/                  # Billing domain (charges, invoices, apps)
│   ├── customer/                 # Customer management
│   ├── entitlement/              # Access control & entitlements
│   ├── subscription/             # Subscription lifecycle
│   ├── credit/                   # Credit ledger
│   ├── ledger/                   # Ledger accounts
│   ├── notification/             # Event notifications
│   ├── meter/                    # Meter definitions
│   ├── ingest/                   # Event ingestion pipeline
│   ├── sink/                     # ClickHouse sink logic
│   ├── streaming/                # Streaming connector abstraction
│   ├── ent/                      # Ent ORM schema + generated DB code
│   │   ├── schema/               # Source-of-truth entity definitions
│   │   └── db/                   # Generated ent code (DO NOT EDIT)
│   ├── watermill/                # Watermill pub-sub wiring
│   ├── productcatalog/           # Product/feature catalog
│   ├── namespace/                # Multi-tenancy namespace management
│   ├── app/                      # App integrations (Stripe, sandbox, custominvoicing)
│   ├── llmcost/                  # LLM model cost prices
│   ├── portal/                   # Portal token issuance
│   ├── subject/                  # Subject management
│   ├── taxcode/                  # Tax code management
│   ├── secret/                   # Encrypted secrets store
│   ├── currencies/               # Custom currencies
│   ├── cost/                     # Feature cost computation
│   └── testutils/                # Shared test helpers
├── api/                          # API layer
│   ├── spec/                     # TypeSpec source (source of truth)
│   ├── openapi.yaml              # Generated OpenAPI v1 spec
│   ├── openapi.cloud.yaml        # Generated cloud OpenAPI v1 spec
│   ├── api.gen.go                # Generated Go v1 server stubs
│   ├── v3/                       # AIP-style v3 API (spec + generated + handlers)
│   └── client/
│       ├── go/                   # Generated Go SDK
│       ├── javascript/           # Generated JS SDK (@openmeter/sdk)
│       └── python/               # Generated Python SDK
├── app/                          # Application wiring
│   ├── common/                   # DI wiring (wire.go / wire_gen.go per binary)
│   └── config/                   # Viper config structs
├── pkg/                          # Shared utility packages
│   ├── framework/                # HTTP/Ent/Postgres framework helpers
│   ├── models/                   # Shared domain model types
│   ├── pagination/               # Cursor/offset pagination helpers
│   ├── kafka/                    # Kafka helpers
│   └── ...                       # clock, contextx, errorsx, otelx, etc.
├── tools/
│   └── migrate/                  # Migration tooling
│       ├── migrations/           # SQL migration files (golang-migrate format)
│       └── cmd/viewgen/          # SQL view generator tool
├── deploy/
│   └── charts/
│       ├── openmeter/            # Main Helm chart
│       └── benthos-collector/    # Collector Helm chart
├── e2e/                          # End-to-end test suite
├── collector/                    # Benthos collector config + quickstart
├── quickstart/                   # Quickstart full-stack docker-compose
├── docs/                         # Documentation and ADRs
├── etc/                          # Seed data, misc configs
├── Dockerfile                    # Main multi-stage Docker build (all binaries)
├── benthos-collector.Dockerfile  # Collector Docker build (CGO_ENABLED=0)
├── docker-compose.yaml           # Local dev dependencies
├── docker-compose.base.yaml      # Base service definitions
├── atlas.hcl                     # Atlas migration config
├── config.example.yaml           # Reference config (Viper)
├── go.mod / go.sum               # Go module definition
├── Makefile                      # All development commands
└── flake.nix                     # Nix reproducible environment
```

## Run Commands

```bash
# up
docker compose up -d
# down
docker compose down --remove-orphans --volumes
# server
air -c ./cmd/server/.air.toml
# sink-worker
air -c ./cmd/sink-worker/.air.toml
# balance-worker
air -c ./cmd/balance-worker/.air.toml
# billing-worker
air -c ./cmd/billing-worker/.air.toml
# notification-service
air -c ./cmd/notification-service/.air.toml
# build
go build -o build/ -tags=dynamic ./cmd/...
# build-server
go build -o build/server -tags=dynamic ./cmd/server
# build-sink-worker
go build -o build/sink-worker -tags=dynamic ./cmd/sink-worker
# build-balance-worker
go build -o build/balance-worker -tags=dynamic ./cmd/balance-worker
# build-billing-worker
go build -o build/billing-worker -tags=dynamic ./cmd/billing-worker
# build-notification-service
go build -o build/notification-service -tags=dynamic ./cmd/notification-service
# build-jobs
go build -o build/jobs -tags=dynamic ./cmd/jobs
# build-benthos-collector
go build -o build/benthos-collector -tags=dynamic ./cmd/benthos-collector
# test
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic ./...
# test-nocache
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# test-all
docker compose up -d postgres svix redis && SVIX_HOST=localhost go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# etoe
make -C e2e test-local
# etoe-slow
RUN_SLOW_TESTS=1 make -C e2e test-local
# lint
make lint-go lint-api-spec lint-openapi lint-helm
# lint-go
golangci-lint run -v ./...
# lint-go-fast
golangci-lint run -v --config .golangci-fast.yaml ./...
# lint-go-style
golangci-lint fmt -v -d ./...
# lint-api-spec
make -C api/spec lint
# lint-openapi
spectral lint api/openapi.yaml api/openapi.cloud.yaml api/v3/openapi.yaml
# lint-helm
helm lint deploy/charts/openmeter && helm lint deploy/charts/benthos-collector
# fmt
golangci-lint run --fix
# mod
go mod tidy
# generate
go generate ./...
# generate-all
make update-openapi generate-javascript-sdk && go generate ./...
# gen-api
make update-openapi generate-javascript-sdk
# update-openapi
make -C api/spec generate && go generate ./api/...
# generate-view-sql
go run ./tools/migrate/cmd/viewgen
# migrate-check
make migrate-check-schema migrate-check-diff migrate-check-lint migrate-check-validate
# migrate-diff
atlas migrate --env local diff <migration-name>
# migrate-check-validate
atlas migrate --env local validate
# migrate-check-lint
atlas migrate --env local lint --latest 10
# seed
benthos -c etc/seed/seed.yaml
# llm-cost-sync
go run ./cmd/jobs llm-cost sync
# package-helm-chart
helm package deploy/charts/<CHART> --version <VERSION> --destination build/helm
```

## Code Templates

### domain_service: New domain service package under openmeter/<domain>/ with Service interface, Adapter interface, input types, and constructor

File: `openmeter/{domain}/service.go`

```
type Service interface { Create(ctx context.Context, input CreateInput) (*Entity, error) }
type service struct { adapter Adapter }
func New(adapter Adapter) Service { return &service{adapter: adapter} }
```

### cmd_worker: New cmd/ worker binary with Cobra entrypoint, Viper config loading, and Wire DI wiring

File: `cmd/{worker}/main.go`

```
func main() { root := &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error { return run(cmd.Context(), cfgFile) }}; root.ExecuteContext(context.Background()) }
func run(ctx context.Context, cfgFile string) error { v := viper.New(); v.SetConfigFile(cfgFile); v.ReadInConfig(); /* app, cleanup, err := InitializeWorker(ctx, cfg) */ return nil }
```

### typespec_route: New TypeSpec route file for a REST resource in api/spec/packages/aip/ (v3) or api/spec/packages/legacy/ (v1)

File: `api/spec/packages/{package}/routes/{resource}.tsp`

```
import "@typespec/http"; import "@typespec/rest";
using TypeSpec.Http; using TypeSpec.Rest;
@route("/api/v1/{resources}") interface {Resource}Routes { @get list(@query namespace: string): {Resource}[] | OpenMeterError; }
```

### wire_provider_set: Wire DI provider set for a new domain in app/common/<domain>.go

File: `app/common/{domain}.go`

```
var {Domain} = wire.NewSet(New{Domain}Adapter, New{Domain}Service)
func New{Domain}Adapter(db *entdb.Client) {domain}.Adapter { return adapter.New(db) }
func New{Domain}Service(a {domain}.Adapter) {domain}.Service { return {domain}.New(a) }
```

## Testing

- **golangci-lint config version: 2** — Go linting suite; config in .golangci.yaml / .golangci-fast.yaml
- **Spectral CLI 6.13.1** — OpenAPI spec linting for api/openapi.yaml and api/v3/openapi.yaml (via npx in Nix shell)
- **Prettier 3.8.2** — TypeSpec and JSON/YAML formatting in api/spec/
- **testify v1.11.1** — Assertion and mock library for Go tests
- **pgtestdb v0.1.1** — Fast PostgreSQL test database provisioning
- **gofakeit v6.28.0** — Fake data generation for tests

```bash
# test
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic ./...
# test-nocache
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# test-all
docker compose up -d postgres svix redis && SVIX_HOST=localhost go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# lint
make lint-go lint-api-spec lint-openapi lint-helm
# lint-go
golangci-lint run -v ./...
# lint-go-fast
golangci-lint run -v --config .golangci-fast.yaml ./...
# lint-go-style
golangci-lint fmt -v -d ./...
# lint-api-spec
make -C api/spec lint
# lint-openapi
spectral lint api/openapi.yaml api/openapi.cloud.yaml api/v3/openapi.yaml
# lint-helm
helm lint deploy/charts/openmeter && helm lint deploy/charts/benthos-collector
# migrate-check
make migrate-check-schema migrate-check-diff migrate-check-lint migrate-check-validate
# migrate-check-validate
atlas migrate --env local validate
# migrate-check-lint
atlas migrate --env local lint --latest 10
```