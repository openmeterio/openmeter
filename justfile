[private]
default:
    @just --list

# Start dependencies via docker compose. Set COMPOSE_PROFILES=dev,redis,... as needed.
up:
    @printf 'Executing target: \033[36m%s\033[0m\n' up
    docker compose up -d

# Stop dependencies via docker compose.
down:
    @printf 'Executing target: \033[36m%s\033[0m\n' down
    docker compose down --remove-orphans --volumes

# Patch oapi-codegen chi-middleware template with custom filter parsing.
patch-oapi-templates:
    @printf 'Executing target: \033[36m%s\033[0m\n' patch-oapi-templates
    ./tools/tasks/patch-oapi-templates.sh

# Generate OpenAPI specs from TypeSpec and Go API code from OpenAPI.
update-openapi: patch-oapi-templates
    @printf 'Executing target: \033[36m%s\033[0m\n' update-openapi
    ./tools/tasks/api-spec-generate.sh
    go generate ./api/...

# Generate OpenAPI specs only.
api-spec-generate:
    @printf 'Executing target: \033[36m%s\033[0m\n' api-spec-generate
    ./tools/tasks/api-spec-generate.sh

# Format TypeSpec API specs.
api-spec-format:
    @printf 'Executing target: \033[36m%s\033[0m\n' api-spec-format
    cd api/spec && pnpm --frozen-lockfile install
    cd api/spec && pnpm format

# Lint TypeSpec API specs.
api-spec-lint:
    @printf 'Executing target: \033[36m%s\033[0m\n' api-spec-lint
    cd api/spec && pnpm --frozen-lockfile install
    cd api/spec && pnpm lint

# Generate JavaScript SDK.
generate-javascript-sdk:
    @printf 'Executing target: \033[36m%s\033[0m\n' generate-javascript-sdk
    cd api/client/javascript && pnpm --frozen-lockfile install
    cd api/client/javascript && pnpm run generate
    cd api/client/javascript && pnpm build
    cd api/client/javascript && pnpm test

# Generate API and SDKs.
gen-api: update-openapi generate-javascript-sdk
    @printf 'Executing target: \033[36m%s\033[0m\n' gen-api

# Execute all code generators.
generate-all: update-openapi generate-javascript-sdk
    @printf 'Executing target: \033[36m%s\033[0m\n' generate-all
    go generate ./...

# Generate Go code.
generate: patch-oapi-templates
    @printf 'Executing target: \033[36m%s\033[0m\n' generate
    go generate ./...

# Generate SQL for ent.View schemas.
generate-view-sql:
    @printf 'Executing target: \033[36m%s\033[0m\n' generate-view-sql
    go run ./tools/migrate/cmd/viewgen

# Generate SQLC testdata for a specific version.
generate-sqlc-testdata version='':
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' generate-sqlc-testdata
    version="{{ version }}"
    version="${version:-${VERSION:-}}"
    if [ -z "$version" ]; then
      echo "Usage: just generate-sqlc-testdata <migration_version>"
      echo "   or: VERSION=<migration_version> just generate-sqlc-testdata"
      exit 1
    fi
    VERSION="$version" ./tools/migrate/generate-sqlc-testdata.sh

# Validate migrations.
migrate-check: migrate-check-schema migrate-check-diff migrate-check-lint migrate-check-validate
    @printf 'Executing target: \033[36m%s\033[0m\n' migrate-check

# Ensure ent schema is in sync with generated code.
migrate-check-schema:
    @printf 'Executing target: \033[36m%s\033[0m\n' migrate-check-schema
    go generate -x ./openmeter/ent/...
    ./tools/tasks/check-clean-paths.sh "!!! schema is not in sync with generated code - run 'go generate ./openmeter/ent/...' and commit the changes !!!" openmeter/ent

# Ensure migrations are in sync with schema.
migrate-check-diff:
    @printf 'Executing target: \033[36m%s\033[0m\n' migrate-check-diff
    atlas migrate --env local diff migrate-check >/dev/null
    ./tools/tasks/check-clean-paths.sh "!!! migrations are not in sync with schema - run 'atlas migrate --env local diff <name>' and commit the generated files !!!" tools/migrate/migrations

# Lint the last 10 migrations.
migrate-check-lint:
    @printf 'Executing target: \033[36m%s\033[0m\n' migrate-check-lint
    atlas migrate --env local lint --latest 10

# Validate migration checksums.
migrate-check-validate:
    @printf 'Executing target: \033[36m%s\033[0m\n' migrate-check-validate
    atlas migrate --env local validate

# Ensure build output directory exists.
build-dir:
    @mkdir -p build

# Build all binaries.
build:
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' build
    GO_BUILD_FLAGS="${GO_BUILD_FLAGS--tags=dynamic}"
    mkdir -p build

    pids=()
    build_one() {
      local name="$1"
      local out="$2"
      local pkg="$3"
      (
        printf 'Executing target: \033[36m%s\033[0m\n' "$name"
        # GO_BUILD_FLAGS intentionally follows the old Makefile word-splitting behavior.
        # shellcheck disable=SC2086
        go build -o "$out" ${GO_BUILD_FLAGS} "$pkg"
      ) &
      pids+=("$!")
    }

    build_one build-server build/server ./cmd/server
    build_one build-sink-worker build/sink-worker ./cmd/sink-worker
    build_one build-benthos-collector build/benthos-collector ./cmd/benthos-collector
    build_one build-balance-worker build/balance-worker ./cmd/balance-worker
    build_one build-billing-worker build/billing-worker ./cmd/billing-worker
    build_one build-notification-service build/notification-service ./cmd/notification-service
    build_one build-jobs build/jobs ./cmd/jobs

    failed=0
    for pid in "${pids[@]}"; do
      wait "$pid" || failed=1
    done
    exit "$failed"

# Build server binary.
build-server:
    @printf 'Executing target: \033[36m%s\033[0m\n' build-server
    mkdir -p build
    go build -o build/server ${GO_BUILD_FLAGS--tags=dynamic} ./cmd/server

# Build sink-worker binary.
build-sink-worker:
    @printf 'Executing target: \033[36m%s\033[0m\n' build-sink-worker
    mkdir -p build
    go build -o build/sink-worker ${GO_BUILD_FLAGS--tags=dynamic} ./cmd/sink-worker

# Build benthos collector binary.
build-benthos-collector:
    @printf 'Executing target: \033[36m%s\033[0m\n' build-benthos-collector
    mkdir -p build
    go build -o build/benthos-collector ${GO_BUILD_FLAGS--tags=dynamic} ./cmd/benthos-collector

# Build balance-worker binary.
build-balance-worker:
    @printf 'Executing target: \033[36m%s\033[0m\n' build-balance-worker
    mkdir -p build
    go build -o build/balance-worker ${GO_BUILD_FLAGS--tags=dynamic} ./cmd/balance-worker

# Build billing-worker binary.
build-billing-worker:
    @printf 'Executing target: \033[36m%s\033[0m\n' build-billing-worker
    mkdir -p build
    go build -o build/billing-worker ${GO_BUILD_FLAGS--tags=dynamic} ./cmd/billing-worker

# Build notification-service binary.
build-notification-service:
    @printf 'Executing target: \033[36m%s\033[0m\n' build-notification-service
    mkdir -p build
    go build -o build/notification-service ${GO_BUILD_FLAGS--tags=dynamic} ./cmd/notification-service

# Build jobs binary.
build-jobs:
    @printf 'Executing target: \033[36m%s\033[0m\n' build-jobs
    mkdir -p build
    go build -o build/jobs ${GO_BUILD_FLAGS--tags=dynamic} ./cmd/jobs

# Cross-compile benthos-collector for release. Set GOOS/GOARCH/VERSION or pass args.
build-benthos-collector-release goos='' goarch='' version='':
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' build-benthos-collector-release
    GOOS="${GOOS:-{{ goos }}}" \
      GOARCH="${GOARCH:-{{ goarch }}}" \
      VERSION="${VERSION:-{{ version }}}" \
      ./tools/tasks/build-benthos-collector-release.sh

# Archive the cross-compiled benthos-collector release. Set GOOS/GOARCH or pass args.
archive-benthos-collector-release goos='' goarch='':
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' archive-benthos-collector-release
    GOOS="${GOOS:-{{ goos }}}" \
      GOARCH="${GOARCH:-{{ goarch }}}" \
      ./tools/tasks/archive-benthos-collector-release.sh

# Copy config.example.yaml to config.yaml.
config:
    @printf 'Executing target: \033[36m%s\033[0m\n' config
    cp config.example.yaml config.yaml

# Run API server with hot reload.
server:
    @printf 'Executing target: \033[36m%s\033[0m\n' server
    ./tools/tasks/check-config-fresh.sh
    air -c ./cmd/server/.air.toml

# Run sink-worker with hot reload.
sink-worker:
    @printf 'Executing target: \033[36m%s\033[0m\n' sink-worker
    ./tools/tasks/check-config-fresh.sh
    air -c ./cmd/sink-worker/.air.toml

# Run balance-worker with hot reload.
balance-worker:
    @printf 'Executing target: \033[36m%s\033[0m\n' balance-worker
    ./tools/tasks/check-config-fresh.sh
    air -c ./cmd/balance-worker/.air.toml

# Run billing-worker with hot reload.
billing-worker:
    @printf 'Executing target: \033[36m%s\033[0m\n' billing-worker
    ./tools/tasks/check-config-fresh.sh
    air -c ./cmd/billing-worker/.air.toml

# Run notification-service with hot reload.
notification-service:
    @printf 'Executing target: \033[36m%s\033[0m\n' notification-service
    ./tools/tasks/check-config-fresh.sh
    air -c ./cmd/notification-service/.air.toml

# Sync LLM cost prices from external sources.
llm-cost-sync:
    @printf 'Executing target: \033[36m%s\033[0m\n' llm-cost-sync
    go run ./cmd/jobs llm-cost sync

# Run e2e tests against local OpenMeter.
e2e-test-local:
    @printf 'Executing target: \033[36m%s\033[0m\n' e2e-test-local
    ./tools/tasks/e2e-local.sh

# Stop local e2e docker compose stack.
e2e-env-local-down:
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' e2e-env-local-down
    cd e2e
    docker compose \
      -f docker-compose.infra.yaml \
      -f docker-compose.debug-ports.yaml \
      -f docker-compose.openmeter.yaml \
      -f docker-compose.openmeter-local.yaml \
      down

# Start local e2e docker compose stack.
e2e-env-local-up:
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' e2e-env-local-up
    cd e2e
    docker compose \
      -f docker-compose.infra.yaml \
      -f docker-compose.debug-ports.yaml \
      -f docker-compose.openmeter.yaml \
      -f docker-compose.openmeter-local.yaml \
      up -d --build --force-recreate

# Run e2e tests.
e2e: e2e-test-local

# Compatibility alias for the old typo target.
etoe: e2e

# Run e2e tests with slow tests enabled.
e2e-slow:
    @printf 'Executing target: \033[36m%s\033[0m\n' e2e-slow
    RUN_SLOW_TESTS=1 ./tools/tasks/e2e-local.sh

# Compatibility alias for the old typo target.
etoe-slow: e2e-slow

# Run quickstart tests against local stack with debug ports exposed.
quickstart-test-local:
    @printf 'Executing target: \033[36m%s\033[0m\n' quickstart-test-local
    ./tools/tasks/quickstart-local.sh

# Stop local quickstart docker compose stack.
quickstart-env-local-down:
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' quickstart-env-local-down
    cd quickstart
    docker compose \
      -f docker-compose.yaml \
      -f docker-compose.debug-ports.yaml \
      down

# Start local quickstart docker compose stack.
quickstart-env-local-up:
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' quickstart-env-local-up
    cd quickstart
    docker compose \
      -f docker-compose.yaml \
      -f docker-compose.debug-ports.yaml \
      up -d --force-recreate

[private]
check-postgres:
    @./tools/tasks/check-postgres.sh

# Run tests.
test: check-postgres
    @printf 'Executing target: \033[36m%s\033[0m\n' test
    ./tools/tasks/go-test.sh

# Run tests without cache.
test-nocache: check-postgres
    @printf 'Executing target: \033[36m%s\033[0m\n' test-nocache
    ./tools/tasks/go-test.sh --nocache

# Alias for the less weird spelling.
test-no-cache: test-nocache

# Run tests with svix dependencies, bypassing the test cache.
test-all:
    @printf 'Executing target: \033[36m%s\033[0m\n' test-all
    ./tools/tasks/go-test-all.sh

# Run all linters.
lint: lint-go lint-api-spec lint-openapi lint-helm
    @printf 'Executing target: \033[36m%s\033[0m\n' lint

# Lint Go code.
lint-go:
    @printf 'Executing target: \033[36m%s\033[0m\n' lint-go
    golangci-lint run -v ${GO_LINT_PATH-./...}

# Lint Go bug-finding checks. Set GO_LINT_PATH=./openmeter/ledger/... as needed.
lint-go-fast:
    @printf 'Executing target: \033[36m%s\033[0m\n' lint-go-fast
    golangci-lint run -v --config .golangci-fast.yaml ${GO_LINT_PATH-./...}

# Lint Go formatting and import order.
lint-go-style:
    @printf 'Executing target: \033[36m%s\033[0m\n' lint-go-style
    golangci-lint fmt -v -d ${GO_LINT_PATH-./...}

# Lint Go code since last commit.
lint-go-head:
    @printf 'Executing target: \033[36m%s\033[0m\n' lint-go-head
    golangci-lint run --new-from-rev=HEAD~1

# Lint API spec.
lint-api-spec:
    @printf 'Executing target: \033[36m%s\033[0m\n' lint-api-spec
    cd api/spec && pnpm --frozen-lockfile install
    cd api/spec && pnpm lint

# Lint OpenAPI specs.
lint-openapi:
    @printf 'Executing target: \033[36m%s\033[0m\n' lint-openapi
    spectral lint api/openapi.yaml api/openapi.cloud.yaml api/v3/openapi.yaml

# Lint Helm charts.
lint-helm:
    @printf 'Executing target: \033[36m%s\033[0m\n' lint-helm
    helm lint deploy/charts/openmeter
    helm lint deploy/charts/benthos-collector

# Run CI checks.
ci:
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' ci
    just generate-all

    pids=()
    just lint &
    pids+=("$!")
    just test &
    pids+=("$!")
    just e2e &
    pids+=("$!")

    failed=0
    for pid in "${pids[@]}"; do
      wait "$pid" || failed=1
    done
    exit "$failed"

# Format code.
fmt:
    @printf 'Executing target: \033[36m%s\033[0m\n' fmt
    golangci-lint run --fix

# Run go mod tidy.
mod:
    @printf 'Executing target: \033[36m%s\033[0m\n' mod
    go mod tidy

# Seed OpenMeter with test data.
seed:
    @printf 'Executing target: \033[36m%s\033[0m\n' seed
    benthos -c etc/seed/seed.yaml

# Package a helm chart for release. Set CHART/VERSION or pass args.
package-helm-chart chart='' version='':
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' package-helm-chart
    CHART="${CHART:-{{ chart }}}" \
      VERSION="${VERSION:-{{ version }}}" \
      ./tools/tasks/package-helm-chart.sh

# Generate chart docs from deploy/charts.
charts-docs:
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' charts-docs
    cd deploy/charts
    helm-docs --log-level trace -s file -c . -t "$PWD/template.md" -t README.tmpl.md

# Publish JavaScript SDK.
publish-javascript-sdk:
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' publish-javascript-sdk
    if [ -z "${JS_SDK_RELEASE_VERSION:-}" ]; then
      echo "ERROR: JS_SDK_RELEASE_VERSION is required"
      echo "Usage: JS_SDK_RELEASE_VERSION=1.2.3 JS_SDK_RELEASE_TAG=beta just publish-javascript-sdk"
      exit 1
    fi
    if [ -z "${JS_SDK_RELEASE_TAG:-}" ]; then
      echo "ERROR: JS_SDK_RELEASE_TAG is required"
      echo "Usage: JS_SDK_RELEASE_VERSION=1.2.3 JS_SDK_RELEASE_TAG=beta just publish-javascript-sdk"
      exit 1
    fi
    cd api/client/javascript
    pnpm --frozen-lockfile install
    pnpm version "${JS_SDK_RELEASE_VERSION}" --no-git-tag-version
    CACHE_BUSTER="$(date +%s)" pnpm publish --no-git-checks --tag "${JS_SDK_RELEASE_TAG}"
    echo "Published JavaScript SDK version ${JS_SDK_RELEASE_VERSION} with tag ${JS_SDK_RELEASE_TAG}"

# Publish Python SDK.
publish-python-sdk:
    @printf 'Executing target: \033[36m%s\033[0m\n' publish-python-sdk
    cd api/client/python && ./scripts/release.sh

# Echo a task variable.
var name:
    #!/usr/bin/env bash
    set -euo pipefail
    case "{{ name }}" in
      SVIX_JWT_SECRET) printf '%s\n' "${SVIX_JWT_SECRET-DUMMY_JWT_SECRET}" ;;
      GO_BUILD_FLAGS) printf '%s\n' "${GO_BUILD_FLAGS--tags=dynamic}" ;;
      GO_TEST_PACKAGE_PARALLELISM) printf '%s\n' "${GO_TEST_PACKAGE_PARALLELISM-128}" ;;
      GO_TEST_FLAGS)
        go_build_flags="${GO_BUILD_FLAGS--tags=dynamic}"
        go_test_package_parallelism="${GO_TEST_PACKAGE_PARALLELISM-128}"
        printf '%s\n' "${GO_TEST_FLAGS--p ${go_test_package_parallelism} -parallel 16 ${go_build_flags}}"
        ;;
      GO_LINT_PATH) printf '%s\n' "${GO_LINT_PATH-./...}" ;;
      *) printenv "{{ name }}" 2>/dev/null || true ;;
    esac

# Echo a task variable as name=value.
varexport name:
    #!/usr/bin/env bash
    set -euo pipefail
    value="$(just var "{{ name }}")"
    printf '%s=%s\n' "{{ name }}" "$value"

# Tag and push a beta prerelease.
release:
    #!/usr/bin/env bash
    set -euo pipefail
    printf 'Executing target: \033[36m%s\033[0m\n' release

    git checkout main > /dev/null 2>&1
    git diff-index --quiet HEAD || (echo "Git directory is dirty" && exit 1)

    version=v$(semver bump prerelease beta.. $(git describe --abbrev=0))

    echo "Detected version: ${version}"
    read -r -n 1 -p "Is that correct (y/N)? " answer
    echo

    case ${answer:0:1} in
        y|Y )
            echo "Tagging release with version ${version}"
        ;;
        * )
            echo "Aborting"
            exit 1
        ;;
    esac

    git tag -m "Release ${version}" "$version"
    git push origin "$version"
