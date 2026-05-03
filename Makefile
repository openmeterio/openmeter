# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

# Docker-local svix secret, only used for testing
SVIX_JWT_SECRET = DUMMY_JWT_SECRET

# dynamic forces confluent-kafka-go to build against local librdkafka
GO_BUILD_FLAGS = -tags=dynamic
GO_TEST_PACKAGE_PARALLELISM ?= 128
GO_TEST_FLAGS = -p ${GO_TEST_PACKAGE_PARALLELISM} -parallel 16 ${GO_BUILD_FLAGS}
GO_LINT_PATH ?= ./...

.PHONY: up
up: ## Start the dependencies via docker compose. `export COMPOSE_PROFILES=dev,redis,...`
	$(call print-target)
	docker compose up -d

.PHONY: down
down: ## Stop the dependencies via docker compose
	$(call print-target)
	docker compose down --remove-orphans --volumes

.PHONY: up-replicated
up-replicated: ## Start a 2-node replicated ClickHouse cluster (docker-compose.replicated.yaml) for testing the ReplicatedMergeTree code path
	$(call print-target)
	docker compose -f docker-compose.replicated.yaml up -d

.PHONY: down-replicated
down-replicated: ## Stop the replicated ClickHouse cluster
	$(call print-target)
	docker compose -f docker-compose.replicated.yaml down --remove-orphans --volumes

.PHONY: patch-oapi-templates
patch-oapi-templates: ## Patch oapi-codegen chi-middleware template with custom filter parsing
	$(call print-target)
	@go mod download github.com/oapi-codegen/oapi-codegen/v2
	@OAPI_MOD_DIR=$$(go list -m -f '{{.Dir}}' github.com/oapi-codegen/oapi-codegen/v2) && \
		if [ -z "$$OAPI_MOD_DIR" ]; then echo "error: could not locate oapi-codegen/v2 module dir"; exit 1; fi && \
		cp "$$OAPI_MOD_DIR/pkg/codegen/templates/chi/chi-middleware.tmpl" api/v3/templates/chi-middleware.tmpl && \
		chmod u+w api/v3/templates/chi-middleware.tmpl && \
		patch -p1 -d api/v3/templates < api/v3/templates/chi-middleware.tmpl.patch

.PHONY: update-openapi
update-openapi: patch-oapi-templates ## Update OpenAPI spec
	$(call print-target)
	$(MAKE) -C api/spec generate
	go generate ./api/...

.PHONY: generate-javascript-sdk
generate-javascript-sdk: ## Generate JavaScript SDK
	$(call print-target)
	$(MAKE) -C api/client/javascript generate

.PHONY: gen-api
gen-api: update-openapi generate-javascript-sdk ## Generate API and SDKs
	$(call print-target)

.PHONY: generate-all
generate-all: update-openapi generate-javascript-sdk ## Execute all code generators
	$(call print-target)
	go generate ./...

.PHONY: migrate-check
migrate-check: migrate-check-schema migrate-check-diff migrate-check-lint migrate-check-validate ## Validate migrations

.PHONY: migrate-check-schema
migrate-check-schema: ## Ensure ent schema is in sync with generated code
	$(call print-target)
	go generate -x ./openmeter/ent/...
	@if ! git diff --quiet -- openmeter/ent || [ -n "$$(git ls-files --others --exclude-standard -- openmeter/ent)" ]; then \
		git --no-pager diff -- openmeter/ent; \
		git ls-files --others --exclude-standard -- openmeter/ent; \
		echo "!!! schema is not in sync with generated code — run 'go generate ./openmeter/ent/...' and commit the changes !!!"; \
		exit 1; \
	fi

.PHONY: migrate-check-diff
migrate-check-diff: ## Ensure migrations are in sync with schema (runs atlas migrate diff against a clean target)
	$(call print-target)
	atlas migrate --env local diff migrate-check >/dev/null
	@if ! git diff --quiet -- tools/migrate/migrations || [ -n "$$(git ls-files --others --exclude-standard -- tools/migrate/migrations)" ]; then \
		git --no-pager diff -- tools/migrate/migrations; \
		git ls-files --others --exclude-standard -- tools/migrate/migrations; \
		echo "!!! migrations are not in sync with schema — run 'atlas migrate --env local diff <name>' and commit the generated files !!!"; \
		exit 1; \
	fi

.PHONY: migrate-check-lint
migrate-check-lint: ## Lint the last 10 migrations
	$(call print-target)
	atlas migrate --env local lint --latest 10

.PHONY: migrate-check-validate
migrate-check-validate: ## Validate migration checksums
	$(call print-target)
	atlas migrate --env local validate

.PHONY: generate-sqlc-testdata
generate-sqlc-testdata: ## Generate SQLC testdata for a specific version (make generate-sqlc-testdata VERSION=20240826120919)
	$(call print-target)
	@if [ -z "$(VERSION)" ]; then echo "Usage: make generate-sqlc-testdata VERSION=<migration_version>"; exit 1; fi
	VERSION=$(VERSION) ./tools/migrate/generate-sqlc-testdata.sh

.PHONY: generate
generate: patch-oapi-templates ## Generate code
	$(call print-target)
	go generate ./...

.PHONY: generate-view-sql
generate-view-sql: ## Generate SQL for ent.View schemas
	$(call print-target)
	go run ./tools/migrate/cmd/viewgen

.PHONY: build-dir
build-dir:
	@mkdir -p build

.PHONY: build
build: build-server build-sink-worker build-benthos-collector build-balance-worker build-billing-worker build-notification-service build-jobs ## Build all binaries

# Cross-compile the benthos-collector binary for release archives.
# Usage: make build-benthos-collector-release GOOS=linux GOARCH=amd64 VERSION=v1.2.3
#   Produces build/release/benthos-collector_<GOOS>_<GOARCH>/benthos (+ README.md, LICENSE)
.PHONY: build-benthos-collector-release
build-benthos-collector-release: ## Cross-compile benthos-collector for release (set GOOS/GOARCH/VERSION)
	$(call print-target)
	@if [ -z "$(GOOS)" ] || [ -z "$(GOARCH)" ]; then echo "ERROR: GOOS and GOARCH are required"; exit 1; fi
	@version="$${VERSION:-unknown}" && \
		outdir="build/release/benthos-collector_$(GOOS)_$(GOARCH)" && \
		rm -rf "$$outdir" && mkdir -p "$$outdir" && \
		CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) \
			go build -trimpath \
			-ldflags "-s -w -X main.version=$${version}" \
			-o "$$outdir/benthos" ./cmd/benthos-collector && \
		cp README.md LICENSE "$$outdir/"

# Produces build/release/benthos-collector_<GOOS>_<GOARCH>.tar.gz from the directory above.
.PHONY: archive-benthos-collector-release
archive-benthos-collector-release: ## Archive the cross-compiled benthos-collector (set GOOS/GOARCH)
	$(call print-target)
	@if [ -z "$(GOOS)" ] || [ -z "$(GOARCH)" ]; then echo "ERROR: GOOS and GOARCH are required"; exit 1; fi
	@name="benthos-collector_$(GOOS)_$(GOARCH)" && \
		tar -C build/release -czf "build/release/$${name}.tar.gz" "$$name"

.PHONY: build-server
build-server: | build-dir ## Build server binary
	$(call print-target)
	go build -o build/server ${GO_BUILD_FLAGS} ./cmd/server

.PHONY: build-sink-worker
build-sink-worker: | build-dir ## Build sink-worker binary
	$(call print-target)
	go build -o build/sink-worker ${GO_BUILD_FLAGS} ./cmd/sink-worker

.PHONY: build-benthos-collector
build-benthos-collector: | build-dir ## Build benthos collector binary
	$(call print-target)
	go build -o build/benthos-collector ${GO_BUILD_FLAGS} ./cmd/benthos-collector

.PHONY: build-balance-worker
build-balance-worker: | build-dir ## Build balance-worker binary
	$(call print-target)
	go build -o build/balance-worker ${GO_BUILD_FLAGS} ./cmd/balance-worker

.PHONY: build-billing-worker
build-billing-worker: | build-dir ## Build billing-worker binary
	$(call print-target)
	go build -o build/billing-worker ${GO_BUILD_FLAGS} ./cmd/billing-worker

.PHONY: build-notification-service
build-notification-service: | build-dir ## Build notification-service binary
	$(call print-target)
	go build -o build/notification-service ${GO_BUILD_FLAGS} ./cmd/notification-service

.PHONY: build-jobs
build-jobs: | build-dir ## Build jobs binary
	$(call print-target)
	go build -o build/jobs ${GO_BUILD_FLAGS} ./cmd/jobs

config.yaml:
	cp config.example.yaml config.yaml

.PHONY: server
server: ## Run sink-worker
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air -c ./cmd/server/.air.toml

.PHONY: sink-worker
sink-worker: ## Run sink-worker
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air -c ./cmd/sink-worker/.air.toml

.PHONY: balance-worker
balance-worker: ## Run balance-worker
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air -c ./cmd/balance-worker/.air.toml

.PHONY: billing-worker
billing-worker: ## Run billing-worker
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air -c ./cmd/billing-worker/.air.toml

.PHONY: notification-service
notification-service: ## Run notification-service
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air -c ./cmd/notification-service/.air.toml

.PHONY: llm-cost-sync
llm-cost-sync: ## Sync LLM cost prices from external sources
	$(call print-target)
	go run ./cmd/jobs llm-cost sync

.PHONY: etoe
etoe: ## Run e2e tests
	$(call print-target)
	$(MAKE) -C e2e test-local

.PHONY: etoe-slow
etoe-slow: ## Run e2e tests with slow tests enabled
	$(call print-target)
	export RUN_SLOW_TESTS=1
	$(MAKE) -C e2e test-local


.PHONY: test
test: ## Run tests
	$(call print-target)
	PGPASSWORD=postgres psql -h 127.0.0.1 -U postgres postgres -c "SELECT version();" || (echo "!!! Postgres is not running. Please start it with 'docker compose up -d postgres' !!!" && false)
	POSTGRES_HOST=127.0.0.1 go test ${GO_TEST_FLAGS} ./...

.PHONY: test-nocache
test-nocache: ## Run tests without cache
	$(call print-target)
	PGPASSWORD=postgres psql -h 127.0.0.1 -U postgres postgres -c "SELECT version();" || (echo "!!! Postgres is not running. Please start it with 'docker compose up -d postgres' !!!" && false)
	POSTGRES_HOST=127.0.0.1 go test ${GO_TEST_FLAGS} -count=1 ./...

.PHONY: test-all
test-all: ## Run tests with svix dependencies, bypassing the test cache
	$(call print-target)
	docker compose up -d postgres svix redis
	./tools/wait-for-compose.sh postgres svix redis
	SVIX_HOST="localhost" SVIX_JWT_SECRET="$(SVIX_JWT_SECRET)" go test ${GO_TEST_FLAGS} -count=1 ./...

.PHONY: lint
lint: lint-go lint-api-spec lint-openapi lint-helm ## Run linters
	$(call print-target)

.PHONY: lint-api-spec
lint-api-spec: ## Lint OpenAPI spec
	$(call print-target)
	$(MAKE) -C api/spec lint

.PHONY: lint-openapi
lint-openapi: ## Lint OpenAPI spec
	$(call print-target)
	spectral lint api/openapi.yaml api/openapi.cloud.yaml api/v3/openapi.yaml

.PHONY: lint-helm
lint-helm: ## Lint Helm charts
	$(call print-target)
	helm lint deploy/charts/openmeter
	helm lint deploy/charts/benthos-collector

# Package a helm chart for release.
# Usage: make package-helm-chart CHART=openmeter VERSION=v1.2.3
#   Produces build/helm/<CHART>-<version-without-v>.tgz
.PHONY: package-helm-chart
package-helm-chart: ## Package a helm chart for release (set CHART and VERSION)
	$(call print-target)
	@if [ -z "$(CHART)" ] || [ -z "$(VERSION)" ]; then echo "ERROR: CHART and VERSION are required"; exit 1; fi
	@chart_dir="deploy/charts/$(CHART)" && \
		version_no_v="$(VERSION:v%=%)" && \
		mkdir -p build/helm && \
		helm-docs --log-level info -s file -c "$$chart_dir" \
			-t "deploy/charts/template.md" -t "$$chart_dir/README.tmpl.md" && \
		helm dependency update "$$chart_dir" && \
		helm package "$$chart_dir" \
			--version "$$version_no_v" \
			--app-version "$(VERSION)" \
			--destination build/helm

.PHONY: lint-go
lint-go: ## Lint Go code
	$(call print-target)
	golangci-lint run -v $(GO_LINT_PATH)

.PHONY: lint-go-fast
lint-go-fast: ## Lint Go bug-finding checks (set GO_LINT_PATH=./openmeter/ledger/...)
	$(call print-target)
	golangci-lint run -v --config .golangci-fast.yaml $(GO_LINT_PATH)

.PHONY: lint-go-style
lint-go-style: ## Lint Go formatting and import order
	$(call print-target)
	golangci-lint fmt -v -d $(GO_LINT_PATH)

.PHONY: lint-go-head
lint-go-head: ## Lint Go code since last commit
	$(call print-target)
	golangci-lint run --new-from-rev=HEAD~1

.PHONY: ci
ci: ## Run CI checks
	$(call print-target)
	$(MAKE) generate-all
	$(MAKE) -j 10 lint test etoe

.PHONY: fmt
fmt: ## Format code
	$(call print-target)
	golangci-lint run --fix

.PHONY: mod
mod: ## go mod tidy
	$(call print-target)
	go mod tidy

.PHONY: seed
seed: ## Seed OpenMeter with test data
	$(call print-target)
	benthos -c etc/seed/seed.yaml

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Variable outputting/exporting rules
var-%: ; @echo $($*)
varexport-%: ; @echo $*=$($*)

define print-target
    @printf "Executing target: \033[36m$@\033[0m\n"
endef
