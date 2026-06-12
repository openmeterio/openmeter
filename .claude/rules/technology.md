## Tech Stack

- **API Definition:** TypeSpec @typespec/* with prettier-plugin 1.11.0, OpenAPI / kin-openapi kin-openapi v0.139.0
- **Analytics Database:** ClickHouse (clickhouse-go/v2 + ch-go) clickhouse-go/v2 v2.46.0, ch-go v0.72.0
- **Auth:** golang-jwt/jwt/v5 v5.3.1, xdg-go/scram v1.2.0
- **Backend Framework:** Chi router (go-chi/chi) v5.2.5, go-chi/cors + go-chi/render cors v1.2.2, render v1.0.3, oapi-codegen v2.6.1-0.20260403235458 (tool); runtime v1.4.1; nethttp-middleware v1.1.2
- **Cache:** hashicorp/golang-lru/v2 v2.0.7
- **Cache / Dedupe:** Redis (go-redis/v9) v9.20.0 + redisotel v9.20.0
- **Codegen:** goverter (jmattheis/goverter) v1.9.3 (tool), goderive (awalterschulze/goderive) v0.5.1 (tool)
- **Collector:** Benthos / Redpanda Connect benthos/v4 v4.73.0, connect free bundle v4.93.0
- **Collector (K8s):** controller-runtime + k8s.io client-go controller-runtime v0.24.1, client-go v0.36.1
- **Commit Tooling:** commitizen / prek cz + prek via Nix git-hooks
- **Config:** Viper + Cobra + pflag viper v1.21.0, cobra v1.10.2, pflag v1.0.10
- **DI / Wiring:** google/wire v0.7.0 (tool: google/wire/cmd/wire)
- **Database:** PostgreSQL server postgres:15 (atlas dev db); pgx/v5 v5.9.2 driver; lib/pq v1.12.3
- **Database / Codegen:** sqlc provided via Nix shell (sqlc); gen target generate-sqlc-testdata
- **Database / Migrations:** Atlas (ariga.io/atlas) + atlasx atlas pinned 0.36.0 in flake atlasx; ariga.io/atlas v0.36.2-0.20250730, golang-migrate v4.19.1
- **Database / ORM:** Ent (entgo.io/ent) v0.14.6
- **Decimal / Money:** alpacahq/alpacadecimal + govalues/decimal + shopspring/decimal alpacadecimal v0.0.9, govalues/decimal v0.1.36, shopspring/decimal v1.4.0
- **Dev Environment:** Nix + devenv + flake-parts + git-hooks nixpkgs-unstable; devenv flake; prek pre-commit, air (hot reload) via Nix shell
- **Distributed Locking:** cirello.io/pglock v1.16.1
- **Events:** CloudEvents SDK (cloudevents/sdk-go/v2) v2.16.2
- **Health:** AppsFlyer/go-sundheit v0.6.0
- **Invoicing:** GOBL (invopop/gobl) v0.403.0
- **Linting (API):** Spectral + Prettier (TypeSpec) spectral-cli 6.16.0 (via pnpx), prettier 3.8.3
- **Linting (Go):** golangci-lint config version 2; provided by Nix shell
- **Linting (Helm):** helm lint + helm-docs kubernetes-helm + helm-docs via Nix
- **Linting / Formatting (JS SDK):** Biome @biomejs/biome 2.4.11
- **Logging:** log/slog + samber/slog-multi + lmittmann/tint + golang-cz/devslog slog-multi v1.8.0, tint v1.1.3, devslog v0.0.15
- **Observability:** OpenTelemetry (go.opentelemetry.io/otel) otel v1.44.0 + otlp grpc/http exporters; otelslog bridge v0.19.0; otelsql v0.42.0; otelhttp v0.69.0
- **Observability / Metrics:** Prometheus client_golang v1.23.2 + otel prometheus exporter v0.66.0
- **Payments:** Stripe (stripe-go/v80) v80.2.1
- **Queue / Messaging:** Watermill + watermill-kafka/v3 watermill v1.5.2, watermill-kafka/v3 v3.1.2
- **Queue / Streaming:** Kafka (confluent-kafka-go/v2) v2.14.1 (librdkafka v2.14.1 pinned in flake), IBM/sarama v1.49.0
- **Resilience:** avast/retry-go/v4 + sony/gobreaker retry-go v4.7.0, gobreaker v1.0.0
- **Runtime:** Go 1.25.6 (module); Docker builder golang:1.26.3-alpine; Nix shell go_1_26, Go (collector module) 1.26.3, Node.js v24.15.0 (.nvmrc); engines node>=22 for JS SDK, Python ^3.9 (SDK package); Nix shell python314
- **SDK Tooling (JS):** openapi-typescript + orval + openapi-fetch + zod openapi-typescript 7.13.0, orval 8.7.0, openapi-fetch 0.17.0, zod 4.3.6
- **Scheduling:** go-co-op/gocron/v2 + robfig/cron/v3 gocron v2.21.2, cron v3.0.1
- **State Machine:** qmuntal/stateless v1.8.0
- **Testing:** stretchr/testify v1.11.1, gotestsum v1.13.0 (tool), peterldowns/pgtestdb v0.1.1, brianvoe/gofakeit/v6 v6.28.0
- **Testing (JS SDK):** Vitest + fetch-mock vitest 4.1.4, @fetch-mock/vitest 0.2.18
- **Utilities:** samber/lo + samber/mo lo v1.53.0, mo v1.16.0, rickb777/period + custom pkg/datetime period v1.0.27, oklog/ulid/v2 + google/uuid ulid v2.1.1, uuid v1.6.0
- **Validation:** models.NewNillableGenericValidationError (custom) + getkin/kin-openapi kin-openapi v0.139.0
- **Webhooks:** Svix (svix/svix-webhooks) v1.95.1

## Project Structure

```
openmeter/
├── api/                      # API layer (generated + source)
│   ├── spec/                 # TypeSpec source of truth (packages/legacy, packages/aip)
│   ├── api.gen.go            # oapi-codegen legacy server/types (generated)
│   ├── v3/                   # v3 (AIP) API: api.gen.go, openapi.yaml, filters, templates
│   ├── openapi.yaml / openapi.cloud.yaml
│   └── client/               # javascript (pnpm/@openmeter/sdk), go, python SDKs
├── app/
│   ├── common/               # Wire DI providers (wire.go -> wire_gen.go)
│   └── config/               # Viper config structs
├── cmd/                      # Service entrypoints (.air.toml each)
│   ├── server/  billing-worker/  balance-worker/  sink-worker/
│   ├── notification-service/  jobs/
├── openmeter/                # Core business logic (layered service/adapter)
│   ├── billing/ (service/, adapter/, rating/, worker/, charges/)
│   ├── subscription/  entitlement/  credit/  ledger/  customer/
│   ├── notification/ (service/, adapter/, consumer/, eventhandler/, webhook/, httpdriver/)
│   ├── meter/ meterevent/ ingest/ sink/ streaming/ productcatalog/
│   ├── app/ secret/ portal/ namespace/ llmcost/ cost/ progressmanager/
│   ├── ent/schema/           # Ent entity definitions (DB source of truth)
│   ├── ent/db/               # Generated ent code (DO NOT EDIT)
│   ├── registry/ server/ watermill/ event/ testutils/
├── pkg/                      # Shared utils (clock, models, filter, framework/entutils, kafka, otelx, pagination, ...)
├── collector/                # Separate Go module: Benthos/Redpanda Connect collector
├── tools/migrate/            # Migration tooling + migrations/ (golang-migrate SQL) + atlas.sum
├── e2e/                      # End-to-end tests (docker-compose driven)
├── deploy/charts/            # Helm charts: openmeter, benthos-collector
├── docs/  etc/  quickstart/  test/
├── Dockerfile  benthos-collector.Dockerfile  docker-compose*.yaml
├── atlas.hcl  flake.nix  Makefile  justfile  .golangci.yaml  config.example.yaml
```

## Run Commands

```bash
# up
docker compose up -d  (start kafka/clickhouse + profiled postgres/redis/svix/dev deps)
# down
docker compose down --remove-orphans --volumes
# server
air -c ./cmd/server/.air.toml  (hot-reload API server; checks config.yaml freshness)
# sink-worker
air -c ./cmd/sink-worker/.air.toml
# balance-worker
air -c ./cmd/balance-worker/.air.toml
# billing-worker
air -c ./cmd/billing-worker/.air.toml
# notification-service
air -c ./cmd/notification-service/.air.toml
# llm-cost-sync
go run ./cmd/jobs llm-cost sync
# test
PGPASSWORD=postgres psql ... && POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic ./...
# test-nocache
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# test-all
docker compose up -d postgres svix redis && SVIX_HOST=localhost SVIX_JWT_SECRET=DUMMY_JWT_SECRET go test -tags=dynamic -count=1 ./...
# etoe
make -C e2e test-local  (TZ=UTC OPENMETER_ADDRESS=http://localhost:38888 go test -count=1 ./...)
# etoe-slow
RUN_SLOW_TESTS=1 make -C e2e test-local
# generate
patch-oapi-templates + go generate ./...
# generate-all
update-openapi + generate-javascript-sdk + go generate ./...
# gen-api
update-openapi (make -C api/spec generate + go generate ./api/...) + generate-javascript-sdk
# update-openapi
patch-oapi-templates + make -C api/spec generate + go generate ./api/...
# generate-javascript-sdk
make -C api/client/javascript generate (pnpm install + generate + build + test)
# patch-oapi-templates
copy oapi-codegen chi-middleware.tmpl and apply api/v3/templates/chi-middleware.tmpl.patch
# generate-view-sql
go run ./tools/migrate/cmd/viewgen  (SQL for ent.View schemas)
# generate-sqlc-testdata
VERSION=<ts> ./tools/migrate/generate-sqlc-testdata.sh
# migrate-check
migrate-check-schema + migrate-check-diff + migrate-check-lint + migrate-check-validate
# migrate-check-diff
atlas migrate --env local diff migrate-check (must produce no changes)
# migrate-check-lint
atlas migrate --env local lint --latest 10
# migrate-check-validate
atlas migrate --env local validate
# atlas-diff
atlas migrate --env local diff <migration-name>  (generate new migration)
# lint
lint-go + lint-api-spec + lint-openapi + lint-helm
# lint-go
golangci-lint run -v ./...
# lint-go-fast
golangci-lint run -v --config .golangci-fast.yaml $(GO_LINT_PATH)
# lint-go-style
golangci-lint fmt -v -d $(GO_LINT_PATH)
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
# build
build-server build-sink-worker build-benthos-collector build-balance-worker build-billing-worker build-notification-service build-jobs
# build-server
go build -o build/server -tags=dynamic ./cmd/server
# build-benthos-collector
go build -C collector -o ../build/benthos-collector -tags=dynamic ./cmd
# build-benthos-collector-release
CGO_ENABLED=0 GOOS=.. GOARCH=.. go build -C collector -trimpath -ldflags '-s -w -X main.version=..' (set GOOS/GOARCH/VERSION)
# package-helm-chart
helm-docs + helm dependency update + helm package (set CHART and VERSION)
# seed
benthos -c etc/seed/seed.yaml
# ci
make generate-all && make -j 10 lint test etoe
# ci-shell
nix develop --impure .#ci -c <command>  (run any toolchain command in the reproducible CI shell)
```

## Code Templates

### domain service: Service layer struct in openmeter/<domain>/service/service.go: takes a Config of injected deps (Repository, sub-services, *slog.Logger), New() validates required deps and asserts the interface. Loggers injected, never slog.Default().

File: `openmeter/<domain>/service/service.go`

```
var _ <domain>.Service = (*Service)(nil)
type Config struct { Adapter <domain>.Repository; Logger *slog.Logger }
func New(c Config) (*Service, error) { if c.Adapter == nil { return nil, errors.New("missing repository") }; return &Service{adapter: c.Adapter, logger: c.Logger}, nil }
```

### domain adapter (Ent repo): Repository/adapter in openmeter/<domain>/adapter/adapter.go: wraps *entdb.Client, Config.Validate() requires client+logger, returns the domain Repository interface; helpers stay transaction-aware via entutils.TransactingRepo.

File: `openmeter/<domain>/adapter/adapter.go`

```
type Config struct { Client *entdb.Client; Logger *slog.Logger }
func (c Config) Validate() error { if c.Client == nil { return errors.New("postgres client is required") }; return nil }
func New(c Config) (<domain>.Repository, error) { if err := c.Validate(); err != nil { return nil, err }; return &adapter{db: c.Client, logger: c.Logger}, nil }
```

### Ent schema entity: Entity definition in openmeter/ent/schema/<entity>.go: embed ent.Schema, declare Mixin() (IDMixin/TimeMixin/MetadataMixin), Fields(), Edges(), Indexes(); run `make generate` then `atlas migrate --env local diff <name>` to produce migrations.

File: `openmeter/ent/schema/<entity>.go`

```
type Feature struct { ent.Schema }
func (Feature) Mixin() []ent.Mixin { return []ent.Mixin{entutils.IDMixin{}, entutils.TimeMixin{}, entutils.MetadataMixin{}} }
func (Feature) Fields() []ent.Field { return []ent.Field{ field.String("namespace").NotEmpty().Immutable() } }
```

### HTTP handler (httpdriver): HTTP handlers live in openmeter/<domain>/httpdriver/ and are wired into the Chi server via generated oapi-codegen stubs; validation surfaces ValidationIssue (see /api skill).

File: `openmeter/<domain>/httpdriver/<resource>.go`

```
func (h *handler) Create() Create<Resource>Handler { return httptransport.NewHandlerWithArgs(resolveRequest, handle, encodeResponse) }
```

### Goverter converter: Define a converter interface with goverter annotations in <pkg>/convert.go; `make generate` emits <pkg>/convert.gen.go. Use FromAPI/ToAPI/FromDB/ToDB naming (/go-types-conversion skill).

File: `openmeter/<domain>/convert.go`

```
// goverter:converter
type Converter interface { ToAPI(domain.X) api.X }
```

### Wire DI provider: Register a new dependency by adding a provider to a wire.go set in app/common/ (or domain registry), then run `make generate` to regenerate wire_gen.go.

File: `app/common/wire.go`

```
var <Domain>Set = wire.NewSet(adapter.New, service.New, wire.Bind(new(domain.Service), new(*service.Service)))
```

### SQL migration: Generated by diffing the Ent schema; never hand-edit timestamped files except ent.View DDL which Atlas does not emit. atlas.sum must stay append-only (enforced in pr-checks.yaml).

File: `tools/migrate/migrations/<timestamp>_<name>.up.sql / .down.sql`

```
-- generated via: atlas migrate --env local diff <name>
```

### service test (TestEnv): Tests build deps from underlying constructors (repos/adapters/services/lockr), not app/common wiring; require POSTGRES_HOST=127.0.0.1; use t.Context(); freeze time with clock.FreezeTime + defer clock.UnFreeze (/test skill, AGENTS.md).

File: `openmeter/<domain>/.../<thing>_test.go`

```
func TestX(t *testing.T) { ctx := t.Context(); /* given/when/then */ require.Equal(t, float64(5), got.InexactFloat64()) }
```

## Testing

- **stretchr/testify v1.11.1** — Assertion/require + mock framework for all Go tests (go.mod, AGENTS.md /test skill)
- **gotestsum v1.13.0 (tool)** — Test runner with richer output (go.mod tool, flake)
- **peterldowns/pgtestdb v0.1.1** — Ephemeral Postgres test databases per test; requires POSTGRES_HOST=127.0.0.1 or suites skip (go.mod, AGENTS.md)
- **brianvoe/gofakeit/v6 v6.28.0** — Fake data generation in tests (go.mod)

```bash
# test
PGPASSWORD=postgres psql ... && POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic ./...
# test-nocache
POSTGRES_HOST=127.0.0.1 go test -p 128 -parallel 16 -tags=dynamic -count=1 ./...
# test-all
docker compose up -d postgres svix redis && SVIX_HOST=localhost SVIX_JWT_SECRET=DUMMY_JWT_SECRET go test -tags=dynamic -count=1 ./...
# generate-sqlc-testdata
VERSION=<ts> ./tools/migrate/generate-sqlc-testdata.sh
# migrate-check
migrate-check-schema + migrate-check-diff + migrate-check-lint + migrate-check-validate
# migrate-check-diff
atlas migrate --env local diff migrate-check (must produce no changes)
# migrate-check-lint
atlas migrate --env local lint --latest 10
# migrate-check-validate
atlas migrate --env local validate
# lint
lint-go + lint-api-spec + lint-openapi + lint-helm
# lint-go
golangci-lint run -v ./...
# lint-go-fast
golangci-lint run -v --config .golangci-fast.yaml $(GO_LINT_PATH)
# lint-go-style
golangci-lint fmt -v -d $(GO_LINT_PATH)
# lint-go-head
golangci-lint run --new-from-rev=HEAD~1
# lint-openapi
spectral lint api/openapi.yaml api/openapi.cloud.yaml api/v3/openapi.yaml
# lint-helm
helm lint deploy/charts/openmeter && helm lint deploy/charts/benthos-collector
```