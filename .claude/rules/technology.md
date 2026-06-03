## Tech Stack

- **AI/ML:** LLM cost sync (no inference SDK) internal (openmeter/llmcost)
- **Analytics Store:** ClickHouse clickhouse-go/v2 v2.46.0; ch-go v0.72.0
- **Auth:** golang-jwt v5.3.1 (golang-jwt/jwt/v5)
- **Backend Framework:** Chi router v5.2.5, oapi-codegen v2.6.1-0.20260403235458 (pinned pseudo-version, go.mod tool), kin-openapi v0.139.0, oapi-codegen/nethttp-middleware (oasmiddleware) v1.1.2, Cobra v1.10.2, Viper v1.21.0, oklog/run openmeterio fork pseudo-version (replace directive in go.mod), qmuntal/stateless v1.8.0, gocron v2.21.2 (go-co-op/gocron/v2), Redpanda Benthos/Connect benthos (collector go.mod, separate module)
- **Cache:** Redis go-redis/v9 v9.19.0 (+ redisotel v9.19.0)
- **Codegen:** Goverter v1.9.3 (go.mod tool: jmattheis/goverter/cmd/goverter), Goderive v0.5.1 (go.mod tool: awalterschulze/goderive), TypeSpec @typespec/prettier-plugin-typespec 1.11.0 (api/spec/package.json)
- **Database:** PostgreSQL 15 (atlas.hcl dev db docker://postgres/15; docker-compose base), Ent ORM v0.14.6 (entgo.io/ent), pgx driver v5.9.2 (jackc/pgx/v5), Atlas atlasx 0.36.0 (flake.nix); go module ariga.io/atlas v0.36.2 pseudo-version, golang-migrate v4.19.1, huandu/go-sqlbuilder v1.41.0
- **Events:** CloudEvents SDK v2.16.2 (cloudevents/sdk-go/v2)
- **Frontend Framework:** React 19.2.5 (devDep) / peer >=18.0.0 (api/client/javascript/package.json)
- **Infrastructure:** Nix (devenv/flake-parts) nixpkgs-unstable, Air from Nix shell, Helm kubernetes-helm + helm-docs from Nix, Kubernetes / controller-runtime k8s.io/* v0.35.3 (indirect)
- **Linting/Formatting:** golangci-lint from Nix shell; config .golangci.yaml (v2) + .golangci-fast.yaml, Spectral 6.16.0 (via pnpx in flake.nix), Prettier 3.8.3 (api/spec), 3.8.2 (js client), Biome 2.4.11 (api/client/javascript), commitizen / prek from Nix (git-hooks)
- **Monitoring:** OpenTelemetry otel v1.43.0 (traces/metrics/logs SDK + OTLP exporters), Prometheus client v1.23.2 (prometheus/client_golang) + otel prometheus exporter v0.65.0, slog + tint/devslog/otelslog tint v1.1.3, devslog v0.0.15, go-slog/otelslog v0.3.0, slog-multi v1.8.0
- **Payments:** Stripe Go SDK v80.2.1 (stripe-go/v80), GOBL v0.403.0 (invopop/gobl), alpacadecimal v0.0.9 (alpacahq/alpacadecimal)
- **Queue:** Kafka (confluent-kafka-go) v2.14.1 (links librdkafka v2.14.1 pinned in flake.nix), Watermill + watermill-kafka watermill v1.5.2 + watermill-kafka/v3 v3.1.2, IBM/sarama v1.49.0
- **Runtime:** Go 1.25.6 (go.mod); toolchain golang:1.26.3-alpine in Dockerfile, go_1_26 in flake.nix, Node.js nodejs-slim_24 (flake.nix); SDK engines >=22.0.0, Python python314 (flake.nix); SDK targets ^3.9 (pyproject.toml)
- **State Management / DI:** Google Wire v0.7.0 (go.mod tool: google/wire/cmd/wire)
- **Testing:** testify v1.11.1, pgtestdb v0.1.1 (peterldowns/pgtestdb), gofakeit v6.28.0 (brianvoe/gofakeit/v6), gotestsum v1.13.0 (go.mod tool)
- **Utilities:** samber/lo v1.53.0, oklog/ulid v2.1.1 (oklog/ulid/v2), clock (clock.FreezeTime) internal (pkg)
- **Validation:** models.Validator + ValidationIssue internal (pkg/models)
- **Webhooks:** Svix v1.94.0 (svix/svix-webhooks)

## Project Structure

```
openmeter/
|-- cmd/                         # Six Go binaries (each main.go is a wire+startup entrypoint, no business logic)
|   |-- server/                  # Main HTTP API (wire.go, wire_gen.go, main.go, .air.toml)
|   |-- sink-worker/             # Kafka -> ClickHouse high-throughput sink
|   |-- balance-worker/          # Entitlement balance recalculation
|   |-- billing-worker/          # Billing/charge lifecycle worker
|   |-- notification-service/    # Svix webhook dispatcher
|   `-- jobs/                    # Cobra admin CLI (billing advance, llm-cost sync, migrate)
|-- collector/                   # Separate Go module: benthos collector binary (cmd/main.go) + plugins
|-- openmeter/                   # Core domain packages (layered service/adapter/httpdriver)
|   |-- billing/                 # billing.Service, charges, adapter, service, worker, rating
|   |-- customer/ entitlement/ subscription/ credit/ ledger/ notification/
|   |-- meter/ ingest/ sink/ streaming/ namespace/ productcatalog/ app/ llmcost/ portal/
|   |-- watermill/               # eventbus, router, grouphandler, marshaler
|   |-- ent/schema/              # Ent entity schemas (DB source of truth)
|   |-- ent/db/                  # Generated Ent code (DO NOT EDIT)
|   `-- testutils/               # Shared test helpers (must not import app/common)
|-- api/
|   |-- spec/                    # TypeSpec source (packages/aip v3, packages/legacy v1)
|   |-- openapi.yaml, openapi.cloud.yaml  # Generated v1 specs
|   |-- api.gen.go               # Generated v1 server stubs (DO NOT EDIT)
|   |-- v3/                      # v3 server, handlers/, openapi.yaml, api.gen.go, codegen.yaml, templates/
|   `-- client/                  # go/ (client.gen.go), javascript/ (@openmeter/sdk), python/ (Poetry)
|-- app/
|   |-- common/                  # Google Wire provider sets (one file per domain + openmeter_<binary>.go)
|   `-- config/                  # Viper config structs + SetViperDefaults
|-- pkg/                         # Shared infra: framework/ (httptransport, entutils, lockr, commonhttp, tracex), models/, pagination/, kafka/, clock/
|-- tools/migrate/               # migrations/ (Atlas golang-migrate SQL + atlas.sum), cmd/viewgen/
|-- deploy/charts/               # Helm charts: openmeter/, benthos-collector/
|-- e2e/                         # End-to-end test suite (own docker-compose files)
|-- collector/quickstart/, quickstart/  # Quickstart docker-compose setups
|-- Dockerfile                   # Multi-stage build producing 6 binaries (static musl, CGO_ENABLED=1)
|-- benthos-collector.Dockerfile # Separate collector image
|-- docker-compose.yaml / .base.yaml  # Local dev dependencies (kafka, clickhouse, redis, postgres, svix)
|-- atlas.hcl                    # Atlas config (ent schema source, golang-migrate format)
|-- flake.nix                    # Nix reproducible toolchain (.#ci shell)
|-- config.example.yaml          # Reference Viper config (copied to config.yaml)
|-- .golangci.yaml / .golangci-fast.yaml
`-- Makefile                     # Canonical task runner
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
make build (go build -o build/ -tags=dynamic across cmd/server, sink-worker, balance-worker, billing-worker, notification-service, jobs, plus benthos-collector)
# build-server
go build -o build/server -tags=dynamic ./cmd/server
# test
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic ./... (make test; checks Postgres is running first)
# test-nocache
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# test-all
docker compose up -d postgres svix redis && SVIX_HOST=localhost SVIX_JWT_SECRET=DUMMY_JWT_SECRET go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
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
# lint-openapi
spectral lint api/openapi.yaml api/openapi.cloud.yaml api/v3/openapi.yaml
# lint-helm
helm lint deploy/charts/openmeter && helm lint deploy/charts/benthos-collector
# fmt
golangci-lint run --fix
# mod
go mod tidy && go mod tidy -C collector
# generate
make generate (patch-oapi-templates then go generate ./...)
# generate-all
make update-openapi generate-javascript-sdk && go generate ./...
# gen-api
make update-openapi generate-javascript-sdk
# update-openapi
make patch-oapi-templates && make -C api/spec generate && go generate ./api/...
# generate-view-sql
go run ./tools/migrate/cmd/viewgen
# migrate-check
make migrate-check-schema migrate-check-diff migrate-check-lint migrate-check-validate
# migrate-diff
atlas migrate --env local diff <migration-name>
# migrate-check-lint
atlas migrate --env local lint --latest 10
# migrate-check-validate
atlas migrate --env local validate
# seed
benthos -c etc/seed/seed.yaml
# llm-cost-sync
go run ./cmd/jobs llm-cost sync
# ci
make generate-all && make -j 10 lint test etoe
# package-helm-chart
make package-helm-chart CHART=<name> VERSION=<v>
# nix-ci-shell
nix develop --impure .#ci -c <command>
```

## Code Templates

### domain_service: New domain Service interface defined at package root (service.go or <domain>.go), implemented in <domain>/service/, calling an Adapter interface for all DB access

File: `openmeter/{domain}/service.go`

```
type Service interface { Create(ctx context.Context, in CreateInput) (*Entity, error) }
type service struct { adapter Adapter }
func New(adapter Adapter) Service { return &service{adapter: adapter} }
```

### domain_adapter: Ent/PostgreSQL adapter implementing the TxCreator+TxUser triad (Tx/WithTx/Self); every method body wraps with entutils.TransactingRepo so it rebinds to the ctx-bound transaction

File: `openmeter/{domain}/adapter/adapter.go`

```
type adapter struct{ db *entdb.Client }
func (a *adapter) Self() *adapter { return a }
func (a *adapter) Create(ctx context.Context, in CreateInput) (*Entity, error) { return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { return toDomain(tx.db.Entity.Create().Save(ctx)) }) }
```

### v3_handler: v3 API handler per resource group implementing generated ServerInterface methods, returning api.<Op>200JSONResponse and models.Generic* errors

File: `api/v3/handlers/{resource}/handler.go`

```
func (h *Handler) ListItems(ctx context.Context, req api.ListItemsRequestObject) (api.ListItemsResponseObject, error) {
    items, err := h.svc.List(ctx, domain.ListInput{Namespace: req.Params.Namespace})
    if err != nil { return nil, err }
    return api.ListItems200JSONResponse{Items: toAPI(items)}, nil
}
```

### wire_provider_set: Wire provider set for a new domain, placed in app/common (never in domain packages); cross-domain hooks registered here as side-effects

File: `app/common/{domain}.go`

```
var Domain = wire.NewSet(NewDomainAdapter, NewDomainService)
func NewDomainAdapter(db *entdb.Client) domain.Adapter { return adapter.New(db) }
func NewDomainService(a domain.Adapter) domain.Service { return domain.New(a) }
```

### ent_schema: New Ent entity schema; Mixin() must return IDMixin, NamespaceMixin, TimeMixin as the first three mixins (multi-tenancy + soft-delete)

File: `openmeter/ent/schema/{entity}.go`

```
type Entity struct { ent.Schema }
func (Entity) Mixin() []ent.Mixin { return []ent.Mixin{entutils.IDMixin{}, entutils.NamespaceMixin{}, entutils.TimeMixin{}} }
func (Entity) Fields() []ent.Field { return []ent.Field{field.String("name")} }
```

### cmd_worker: New worker binary: Cobra/main.go performs only wire + run.Group lifecycle, with a matching app/common/openmeter_<binary>.go provider set

File: `cmd/{worker}/main.go`

```
func main() { root := &cobra.Command{RunE: func(cmd *cobra.Command, args []string) error { return run(cmd.Context(), cfgFile) }}; root.ExecuteContext(context.Background()) }
func run(ctx context.Context, cfgFile string) error { app, cleanup, err := initializeApplication(ctx, cfg); defer cleanup(); return app.Run() }
```

## Testing

- **testify v1.11.1** — Assertions and suites for Go unit/integration tests
- **pgtestdb v0.1.1 (peterldowns/pgtestdb)** — Fast per-test PostgreSQL database provisioning
- **gofakeit v6.28.0 (brianvoe/gofakeit/v6)** — Fake test data generation
- **gotestsum v1.13.0 (go.mod tool)** — Test runner with improved output

```bash
# test
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic ./... (make test; checks Postgres is running first)
# test-nocache
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# test-all
docker compose up -d postgres svix redis && SVIX_HOST=localhost SVIX_JWT_SECRET=DUMMY_JWT_SECRET go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
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
# lint-openapi
spectral lint api/openapi.yaml api/openapi.cloud.yaml api/v3/openapi.yaml
# lint-helm
helm lint deploy/charts/openmeter && helm lint deploy/charts/benthos-collector
# migrate-check
make migrate-check-schema migrate-check-diff migrate-check-lint migrate-check-validate
# migrate-check-lint
atlas migrate --env local lint --latest 10
# migrate-check-validate
atlas migrate --env local validate
```