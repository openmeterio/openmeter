## Tech Stack

- **AI/ML:** LLM cost sync (internal) openmeter/llmcost (no inference SDK)
- **Analytics Store:** ClickHouse server 25.12.3-alpine (docker-compose); clickhouse-go/v2 v2.46.0; ch-go v0.72.0
- **Auth:** golang-jwt v5.3.1 (github.com/golang-jwt/jwt/v5)
- **Backend Framework:** Chi router v5.2.5 (github.com/go-chi/chi/v5), oapi-codegen v2.6.1-0.20260403235458-a76544bd16ff (pinned pseudo-version, go.mod tool), kin-openapi v0.139.0 (github.com/getkin/kin-openapi), oapi-codegen nethttp-middleware (oasmiddleware) v1.1.2, Cobra v1.10.2 (github.com/spf13/cobra), Viper v1.21.0 (github.com/spf13/viper), oklog/run v1.1.1-pseudo, replaced with github.com/openmeterio/run fork (go.mod replace directive), qmuntal/stateless v1.8.0, gocron v2.21.2 (github.com/go-co-op/gocron/v2), Redpanda Benthos/Connect benthos v4.73.0 + connect free bundle v4.93.0 (collector/go.mod)
- **Cache:** Redis 7.4.7 (docker-compose); go-redis/v9 v9.19.0 (+ redisotel v9.19.0)
- **Codegen:** Goverter v1.9.3 (go.mod tool: jmattheis/goverter/cmd/goverter), Goderive v0.5.1 (go.mod tool: awalterschulze/goderive), TypeSpec @typespec/prettier-plugin-typespec 1.11.0 (api/spec/package.json)
- **Database:** PostgreSQL 14.20-alpine3.23 (docker-compose.base.yaml); 15 (atlas.hcl dev db docker://postgres/15), Ent ORM v0.14.6 (entgo.io/ent), pgx v5.9.2 (github.com/jackc/pgx/v5), Atlas (atlasx) 0.36.0 (flake.nix atlasx derivation); ariga.io/atlas v0.36.2 pseudo (go.mod indirect), golang-migrate v4.19.1 (github.com/golang-migrate/migrate/v4), huandu/go-sqlbuilder v1.41.0, cirello.io/pglock v1.16.1
- **Events:** CloudEvents SDK v2.16.2 (github.com/cloudevents/sdk-go/v2)
- **Frontend Framework:** React 19.2.5 devDep / >=18.0.0 peer (api/client/javascript/package.json)
- **Infrastructure:** Nix (devenv/flake-parts) nixpkgs-unstable; devenv + git-hooks.nix, Helm + helm-docs kubernetes-helm + helm-docs from Nix, Air from Nix shell, Kubernetes / controller-runtime k8s.io/* v0.35.3 (root indirect); collector sigs.k8s.io/controller-runtime v0.24.1
- **Linting/Formatting:** golangci-lint v2 config (.golangci.yaml); from Nix shell, Spectral 6.16.0 (via pnpx in flake.nix), Prettier 3.8.3 (api/spec), 3.8.2 (js client), Biome 2.4.11 (api/client/javascript), commitizen / prek from Nix git-hooks (flake.nix)
- **Monitoring:** OpenTelemetry otel v1.43.0 (trace/metric/log SDK + OTLP grpc/http exporters), Prometheus client v1.23.2 (prometheus/client_golang) + otel prometheus exporter v0.65.0, slog + tint/devslog/otelslog tint v1.1.3, devslog v0.0.15, go-slog/otelslog v0.3.0, slog-multi v1.8.0
- **Payments:** Stripe Go SDK v80.2.1 (github.com/stripe/stripe-go/v80), GOBL v0.403.0 (github.com/invopop/gobl), alpacadecimal v0.0.9 (github.com/alpacahq/alpacadecimal)
- **Queue:** Kafka (confluent-kafka-go) v2.14.1 links librdkafka v2.14.1 (flake.nix pinned), Watermill + watermill-kafka watermill v1.5.2 + watermill-kafka/v3 v3.1.2, IBM/sarama v1.49.0
- **Runtime:** Go 1.25.6 (go.mod); Docker/flake build golang 1.26.3 / go_1_26, Go (collector module) 1.26.3, Node.js nodejs-slim_24 / corepack_24 (flake.nix); SDK engines >=22.0.0, Python python314 (flake.nix); SDK targets ^3.9 (api/client/python/pyproject.toml)
- **State Management / DI:** Google Wire v0.7.0 (go.mod tool: google/wire/cmd/wire)
- **Testing:** testify v1.11.1 (github.com/stretchr/testify), pgtestdb v0.1.1 (github.com/peterldowns/pgtestdb), gofakeit v6.28.0 (github.com/brianvoe/gofakeit/v6), gotestsum v1.13.0 (go.mod tool: gotest.tools/gotestsum)
- **Utilities:** samber/lo v1.53.0 (also samber/mo v1.16.0), oklog/ulid v2.1.1 (github.com/oklog/ulid/v2), rickb777/period v1.0.27, zeebo/xxh3 / cespare/xxhash xxh3 v1.1.0, xxhash v2.3.0
- **Validation:** models.Validator + ValidationIssue (internal) pkg/models
- **Webhooks:** Svix svix-webhooks Go SDK v1.94.0; svix-server v1.84.1 (docker-compose)

## Project Structure

```
openmeter/
|-- api/                      # API contract surface
|   |-- spec/                 # TypeSpec source (packages/aip v3, packages/legacy v1) -> OpenAPI + SDKs
|   |-- openapi.yaml, openapi.cloud.yaml, api.gen.go   # generated v1
|   |-- v3/                   # v3 server, handlers/, openapi.yaml, api.gen.go, codegen.yaml, templates/
|   |-- client/              # go/ (client.gen.go), javascript/ (@openmeter/sdk), python/ (Poetry); node/ web/ tombstones
|   `-- types/                # hand-authored x-go-type gap-fill types
|-- app/
|   |-- common/               # Google Wire provider sets (per-domain + openmeter_<binary>.go)
|   `-- config/               # Viper config structs + SetViperDefaults
|-- cmd/                      # 6 binary entrypoints: server, sink-worker, balance-worker, billing-worker, notification-service, jobs
|-- collector/                # SEPARATE Go module: benthos collector (cmd/main.go) + plugins, own go.mod/go.sum
|-- openmeter/                # core domain packages (service/adapter/httpdriver layering)
|   |-- billing/ charges/ customer/ entitlement/ subscription/ credit/ ledger/ notification/
|   |-- meter/ ingest/ sink/ streaming/ namespace/ productcatalog/ app/ llmcost/ portal/
|   |-- watermill/            # eventbus, router, grouphandler, marshaler
|   |-- ent/schema/           # Ent entity schemas (DB source of truth)
|   `-- ent/db/               # generated Ent ORM code (DO NOT EDIT)
|-- pkg/                      # shared infra: framework/ (httptransport, entutils, lockr, commonhttp, tracex), models/, pagination/, kafka/, clock/, filter/
|-- tools/migrate/            # migrations/ (Atlas golang-migrate SQL + atlas.sum), cmd/viewgen/
|-- deploy/charts/            # Helm charts: openmeter/, benthos-collector/
|-- e2e/                      # end-to-end test suite (own docker-compose)
|-- test/                     # shared integration test helpers (billing suite etc.)
|-- etc/                      # seed configs (benthos seed.yaml)
|-- docs/ assets/ quickstart/
|-- Dockerfile                # multi-stage, 6 binaries (static musl, CGO_ENABLED=1, -tags musl)
|-- benthos-collector.Dockerfile  # separate collector image (CGO_ENABLED=0)
|-- docker-compose.yaml / .base.yaml   # local dev deps (kafka, clickhouse, redis, postgres, svix)
|-- atlas.hcl                 # Atlas config (ent schema source, golang-migrate format)
|-- flake.nix / flake.lock    # Nix reproducible toolchain (.#ci shell)
|-- config.example.yaml       # reference Viper config (copied to config.yaml)
|-- .golangci.yaml / .golangci-fast.yaml
|-- Makefile / justfile       # task runners
`-- .github/workflows/        # CI, artifacts, release, npm-release, security, pr-checks
```

## Run Commands

```bash
# up
docker compose up -d
# down
docker compose down --remove-orphans --volumes
# server
air -c ./cmd/server/.air.toml (make server)
# sink-worker
air -c ./cmd/sink-worker/.air.toml (make sink-worker)
# balance-worker
air -c ./cmd/balance-worker/.air.toml (make balance-worker)
# billing-worker
air -c ./cmd/billing-worker/.air.toml (make billing-worker)
# notification-service
air -c ./cmd/notification-service/.air.toml (make notification-service)
# build
make build (builds server, sink-worker, benthos-collector, balance-worker, billing-worker, notification-service, jobs with -tags=dynamic)
# build-server
go build -o build/server -tags=dynamic ./cmd/server
# build-benthos-collector
go build -C ./collector -o ../build/benthos-collector -tags=dynamic ./cmd
# build-benthos-collector-release
make build-benthos-collector-release GOOS=<os> GOARCH=<arch> VERSION=<v> (CGO_ENABLED=0, -trimpath)
# test
make test => PGPASSWORD=postgres psql healthcheck then POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic ./...
# test-nocache
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# test-all
docker compose up -d postgres svix redis; SVIX_HOST=localhost SVIX_JWT_SECRET=DUMMY_JWT_SECRET go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# etoe
make -C e2e test-local
# etoe-slow
RUN_SLOW_TESTS=1 make -C e2e test-local
# lint
make lint => lint-go lint-api-spec lint-openapi lint-helm
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
make generate-all (update-openapi + generate-javascript-sdk + go generate ./...)
# gen-api
make gen-api (update-openapi + generate-javascript-sdk)
# update-openapi
make patch-oapi-templates && make -C api/spec generate && go generate ./api/...
# generate-javascript-sdk
make -C api/client/javascript generate
# generate-view-sql
go run ./tools/migrate/cmd/viewgen
# patch-oapi-templates
copy oapi-codegen chi-middleware.tmpl into api/v3/templates and apply chi-middleware.tmpl.patch
# migrate-check
make migrate-check => migrate-check-schema migrate-check-diff migrate-check-lint migrate-check-validate
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

### Ent entity schema: New persisted entity; Mixin() must return IDMixin, NamespaceMixin, TimeMixin first; then make generate + atlas migrate diff

File: `openmeter/ent/schema/{entity}.go`

```
func (Entity) Mixin() []ent.Mixin { return []ent.Mixin{entutils.IDMixin{}, entutils.NamespaceMixin{}, entutils.TimeMixin{}} }
func (Entity) Fields() []ent.Field { return []ent.Field{field.String("name")} }
```

### Domain service interface: Service interface at domain package root; concrete impl in <domain>/service/

File: `openmeter/{domain}/service.go`

```
type Service interface { Create(ctx context.Context, in CreateInput) (*Entity, error) }
type service struct { adapter Adapter }
func New(adapter Adapter) Service { return &service{adapter: adapter} }
```

### Ent adapter: Tx/WithTx/Self triad; every method wrapped in entutils.TransactingRepo to rebind to ctx tx

File: `openmeter/{domain}/adapter/adapter.go`

```
type adapter struct{ db *entdb.Client }
func (a *adapter) Self() *adapter { return a }
func (a *adapter) Create(ctx context.Context, in CreateInput) (*Entity, error) { return entutils.TransactingRepo(ctx, a, func(ctx context.Context, tx *adapter) (*Entity, error) { return toDomain(tx.db.Entity.Create().Save(ctx)) }) }
```

### v3 HTTP handler: Per-resource handler implementing generated ServerInterface methods, returns api.<Op>200JSONResponse / models.Generic* errors

File: `api/v3/handlers/{resource}/handler.go`

```
func (h *Handler) ListItems(ctx context.Context, req api.ListItemsRequestObject) (api.ListItemsResponseObject, error) {
    items, err := h.svc.List(ctx, domain.ListInput{Namespace: req.Params.Namespace})
    if err != nil { return nil, err }
    return api.ListItems200JSONResponse{Items: toAPI(items)}, nil }
```

### Wire provider set: Provider set in app/common (never in domain packages); cross-domain hooks registered here as side-effects

File: `app/common/{domain}.go`

```
var Domain = wire.NewSet(NewDomainAdapter, NewDomainService)
func NewDomainService(a domain.Adapter) domain.Service { return domain.New(a) }
```

### cmd worker binary: main.go does only wire init + run.Group lifecycle; matching app/common/openmeter_<binary>.go set

File: `cmd/{worker}/main.go`

```
func run(ctx context.Context, cfgFile string) error { app, cleanup, err := initializeApplication(ctx, cfg); defer cleanup(); return app.Run() }
```

### TypeSpec v3 endpoint: Author op in packages/aip; route/tag only in root openmeter.tsp; then make gen-api && make generate

File: `api/spec/packages/aip/src/{resource}.tsp`

```
import "@typespec/http";
using TypeSpec.Http;
model MyResource { id: string; name: string; }
```

### SQL migration: Generated only via atlas migrate diff; never hand-edited; commit .up.sql/.down.sql + atlas.sum together

File: `tools/migrate/migrations/{timestamp}_{name}.up.sql`

```
atlas migrate --env local diff <name>  # produces .up.sql/.down.sql + updates atlas.sum
```

### Benthos collector plugin: Registered in init(); activated via blank import from collector/cmd/main.go; sub-package per concern

File: `collector/benthos/{input|bloblang}/{plugin}.go`

```
func init() { service.RegisterInput(...) }  // blank-imported: import _ "github.com/openmeterio/openmeter/collector/benthos/input"
```

## Testing

- **testify v1.11.1 (github.com/stretchr/testify)** — Assertions and test suites
- **pgtestdb v0.1.1 (github.com/peterldowns/pgtestdb)** — Fast per-test Postgres database provisioning
- **gofakeit v6.28.0 (github.com/brianvoe/gofakeit/v6)** — Fake test data generation
- **gotestsum v1.13.0 (go.mod tool: gotest.tools/gotestsum)** — Test runner with improved output

```bash
# test
make test => PGPASSWORD=postgres psql healthcheck then POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic ./...
# test-nocache
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# test-all
docker compose up -d postgres svix redis; SVIX_HOST=localhost SVIX_JWT_SECRET=DUMMY_JWT_SECRET go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# lint
make lint => lint-go lint-api-spec lint-openapi lint-helm
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
make migrate-check => migrate-check-schema migrate-check-diff migrate-check-lint migrate-check-validate
# migrate-check-lint
atlas migrate --env local lint --latest 10
# migrate-check-validate
atlas migrate --env local validate
```