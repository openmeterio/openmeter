# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

# Docker-local svix secret, only used for testing
SVIX_JWT_SECRET = DUMMY_JWT_SECRET

# dynamic forces confluent-kafka-go to build against local librdkafka
GO_BUILD_FLAGS = -tags=dynamic
GO_TEST_FLAGS = -p 128 -parallel 16 ${GO_BUILD_FLAGS}

.PHONY: up
up: ## Start the dependencies via docker compose. `export COMPOSE_PROFILES=dev,redis,...`
	$(call print-target)
	docker compose up -d

.PHONY: down
down: ## Stop the dependencies via docker compose
	$(call print-target)
	docker compose down --remove-orphans --volumes

.PHONY: update-openapi
update-openapi: ## Update OpenAPI spec
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
migrate-check: ## Validate migrations
	$(call print-target)
	dagger call migrate check

.PHONY: generate-sqlc-testdata
generate-sqlc-testdata: ## Generate SQLC testdata for a specific version (make generate-sqlc-testdata VERSION=20240826120919)
	$(call print-target)
	@if [ -z "$(VERSION)" ]; then echo "Usage: make generate-sqlc-testdata VERSION=<migration_version>"; exit 1; fi
	dagger call migrate generate-sqlc-testdata --version=$(VERSION) export --path=tools/migrate/testdata/sqlcgen/$(VERSION)

.PHONY: generate
generate: ## Generate code
	$(call print-target)
	go generate ./...

.PHONY: build-dir
build-dir:
	@mkdir -p build

.PHONY: build
build: build-server build-sink-worker build-benthos-collector build-balance-worker build-billing-worker build-notification-service build-jobs ## Build all binaries

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

.PHONY: lint-go
lint-go: ## Lint Go code
	$(call print-target)
	golangci-lint run -v

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
