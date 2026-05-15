## Tech Stack

- **API Specification:** TypeSpec 1.11.0 (@typespec/compiler)
- **Auth:** golang-jwt v5.3.1, AuthZed/SpiceDB (authzed-go) v1.4.1
- **Backend Framework:** Chi v5.2.5, kin-openapi v0.137.0, oasmiddleware (oapi-codegen/nethttp-middleware) v1.1.2, oapi-codegen v2.6.1 (pinned fork at oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen), Cobra v1.10.2, Viper v1.21.0, oklog/run (openmeterio fork) v0.0.0-20250217124527-c72029d4b634, Redpanda Benthos/Connect v4.55.0 (benthos) + v4.61.0 (connect free), Goverter v1.9.3, Goderive v0.5.1, qmuntal/stateless v1.8.0, gocron v2.21.1, go-resty v2.17.2
- **Cache:** Redis 7.4.7 (docker image)
- **Database:** PostgreSQL 15 (dev docker), 14.20-alpine3.23 (per docs), ClickHouse 25.12.3-alpine (docker image)
- **Database Driver:** pgx v5.9.2
- **Database Migration:** Atlas ariga.io/atlas v0.36.2 (custom fork), golang-migrate v4.19.1
- **Database ORM:** Ent v0.14.6
- **Infrastructure:** Nix (devenv/flake) nixpkgs-unstable, Helm from Nix shell, Kubernetes/controller-runtime v0.23.3, Air from Nix shell, Depot depot/build-push-action
- **Linting/Formatting:** golangci-lint from Nix shell (.#ci), Spectral CLI 6.13.1 (via npx in Nix shell), Prettier 3.8.3 (api/spec), 3.8.2 (js client), Biome 2.4.11, commitizen from Nix (git-hooks)
- **Monitoring:** OpenTelemetry (otel) v1.43.0, Prometheus client v1.23.2
- **Payments:** Stripe Go SDK v80.2.1, GOBL v0.402.0
- **Queue:** Kafka (confluent-kafka-go) v2.14.1 (librdkafka v2.14.1), Watermill + watermill-kafka v1.5.1 + v3.1.2, IBM Sarama v1.48.0
- **Runtime:** Go 1.25.5, Node.js 24 (nodejs-slim_24 via Nix), Python 3.14 (pkgs.python314 via Nix)
- **State Management:** Google Wire v0.7.0
- **Testing:** testify v1.11.1, pgtestdb v0.1.1, gofakeit v6.28.0, gotestsum v1.13.0
- **Utilities:** samber/lo v1.53.0, CloudEvents SDK v2.16.2, oklog/ulid v2.1.1, alpacadecimal v0.0.9, hashicorp/golang-lru v2.0.7
- **Webhooks:** Svix Go SDK v1.92.2

## Project Structure

```
openmeter/
├── cmd/                          # Seven binary entrypoints
│   ├── server/                   # Main API (wire.go, wire_gen.go, main.go)
│   ├── sink-worker/              # Kafka→ClickHouse
│   ├── balance-worker/           # Entitlement balance recalculation
│   ├── billing-worker/           # Billing lifecycle
│   ├── notification-service/     # Webhook dispatcher
│   ├── jobs/                     # Cobra admin CLI
│   └── benthos-collector/        # Redpanda Benthos pipeline
├── openmeter/                    # Core domain packages
│   ├── billing/                  # Billing domain (charges, invoices, apps, worker)
│   ├── customer/                 # Customer lifecycle
│   ├── entitlement/              # Entitlement and balance worker
│   ├── subscription/             # Subscription lifecycle
│   ├── credit/                   # Credit grants and balance snapshots
│   ├── ledger/                   # Double-entry ledger
│   ├── notification/             # Notification channels, rules, events
│   ├── meter/                    # Meter definitions
│   ├── ingest/                   # CloudEvent ingestion
│   ├── sink/                     # ClickHouse sink
│   ├── streaming/                # Streaming connector abstraction
│   ├── ent/
│   │   ├── schema/               # ~30 Ent entity schemas (source of truth)
│   │   └── db/                   # Generated Ent code (DO NOT EDIT)
│   ├── watermill/                # Watermill pub-sub (eventbus, router, grouphandler)
│   ├── productcatalog/           # Plans, features, rate cards, addons
│   ├── namespace/                # Multi-tenancy management
│   ├── app/                      # App marketplace (Stripe, Sandbox, CustomInvoicing)
│   ├── llmcost/                  # LLM model cost prices
│   ├── portal/                   # Portal JWT token issuance
│   └── testutils/                # Shared test helpers
├── api/
│   ├── spec/                     # TypeSpec source (source of truth)
│   │   └── packages/
│   │       ├── aip/              # v3 AIP-style TypeSpec
│   │       └── legacy/           # v1 legacy TypeSpec
│   ├── openapi.yaml              # Generated v1 OpenAPI spec
│   ├── openapi.cloud.yaml        # Generated cloud v1 OpenAPI spec
│   ├── api.gen.go                # Generated Go v1 server stubs (DO NOT EDIT)
│   ├── v3/
│   │   ├── openapi.yaml          # Generated v3 OpenAPI spec
│   │   ├── api.gen.go            # Generated Go v3 server stubs (DO NOT EDIT)
│   │   ├── server/               # v3 server wiring
│   │   └── handlers/             # v3 handler packages per resource group
│   └── client/
│       ├── go/                   # Generated Go SDK (client.gen.go)
│       ├── javascript/           # @openmeter/sdk (npm)
│       └── python/               # openmeter Python SDK (PyPI)
├── app/
│   ├── common/                   # Google Wire provider sets (one file per domain)
│   └── config/                   # Viper config structs
├── pkg/
│   ├── framework/
│   │   ├── transport/httptransport/ # Generic Handler[Request,Response]
│   │   ├── entutils/             # TransactingRepo, mixins, ULID
│   │   ├── lockr/                # pg_advisory_xact_lock wrapper
│   │   ├── commonhttp/           # RFC 7807 error encoding
│   │   └── tracex/               # OTel span helpers
│   ├── models/                   # Shared domain primitives
│   ├── pagination/               # Cursor/offset pagination
│   ├── kafka/                    # Kafka helpers
│   └── ...                       # clock, contextx, errorsx, filter, etc.
├── tools/
│   └── migrate/
│       ├── migrations/           # Atlas SQL migration files (golang-migrate format)
│       └── cmd/viewgen/          # SQL view generator
├── deploy/
│   └── charts/
│       ├── openmeter/            # Main Helm chart
│       └── benthos-collector/    # Collector Helm chart
├── e2e/                          # End-to-end test suite
├── collector/                    # Benthos collector config + quickstart presets
├── Dockerfile                    # Multi-stage build producing 6 binaries
├── benthos-collector.Dockerfile  # Separate collector Docker build (CGO_ENABLED=0)
├── docker-compose.yaml           # Local dev dependencies
├── docker-compose.base.yaml      # Base service definitions
├── atlas.hcl                     # Atlas migration config
├── config.example.yaml           # Reference config (Viper)
├── flake.nix                     # Nix reproducible environment
├── go.mod / go.sum               # Go module definition
└── Makefile                      # All development commands
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
# lint-go-head
golangci-lint run --new-from-rev=HEAD~1
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

### domain_service: New domain service package under openmeter/<domain>/ with Service interface, Adapter interface, and constructor

File: `openmeter/{domain}/service.go`

```
type Service interface { Create(ctx context.Context, input CreateInput) (*Entity, error) }
type service struct { adapter Adapter }
func New(adapter Adapter) Service { return &service{adapter: adapter} }
```

### domain_adapter: Ent/PostgreSQL adapter implementing TxCreator + TxUser triad so TransactingRepo rebinds to ctx-carried tx

File: `openmeter/{domain}/adapter/adapter.go`

```
type adapter struct{ db *entdb.Client }
func (a *adapter) Create(ctx context.Context, in CreateInput) (*Entity, error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { return toDomain(tx.db.Entity.Create().Save(ctx)) })
}
```

### v3_handler: v3 API handler in api/v3/handlers/<resource>/ using generated ServerInterface methods

File: `api/v3/handlers/{resource}/handler.go`

```
func (h *Handler) ListItems(ctx context.Context, req api.ListItemsRequestObject) (api.ListItemsResponseObject, error) {
    items, err := h.svc.List(ctx, domain.ListInput{Namespace: req.Params.Namespace})
    if err != nil { return nil, err }
    return api.ListItems200JSONResponse{Items: toAPI(items)}, nil
}
```

### wire_provider_set: Wire DI provider set for a new domain in app/common/<domain>.go

File: `app/common/{domain}.go`

```
var Domain = wire.NewSet(NewDomainAdapter, NewDomainService)
func NewDomainAdapter(db *entdb.Client) domain.Adapter { return adapter.New(db) }
func NewDomainService(a domain.Adapter) domain.Service { return domain.New(a) }
```

### cmd_worker: New cmd/ worker binary with Cobra, Viper config, Wire DI, and run.Group lifecycle

File: `cmd/{worker}/main.go`

```
func main() { root := &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error { return run(cmd.Context(), cfgFile) }}; root.ExecuteContext(context.Background()) }
func run(ctx context.Context, cfgFile string) error { app, cleanup, err := initializeApplication(ctx, cfg); defer cleanup(); return app.Run() }
```

### ent_schema: New Ent entity schema in openmeter/ent/schema/ with standard mixins

File: `openmeter/ent/schema/{entity}.go`

```
type Entity struct { ent.Schema }
func (Entity) Mixin() []ent.Mixin { return []ent.Mixin{entutils.IDMixin{}, entutils.NamespaceMixin{}, entutils.TimeMixin{}} }
func (Entity) Fields() []ent.Field { return []ent.Field{field.String("name")} }
```

## Testing

- **testify v1.11.1** — Assertions and mocking for all Go unit and integration tests
- **pgtestdb v0.1.1** — Fast PostgreSQL test database provisioning
- **gofakeit v6.28.0** — Fake data generation for test fixtures
- **gotestsum v1.13.0** — Test runner with improved output formatting

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
# lint-go-head
golangci-lint run --new-from-rev=HEAD~1
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