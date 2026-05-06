# Compatibility shim. Canonical task definitions live in ./justfile.

JUST ?= just

# Docker-local svix secret, only used for testing.
SVIX_JWT_SECRET = DUMMY_JWT_SECRET

# dynamic forces confluent-kafka-go to build against local librdkafka.
GO_BUILD_FLAGS = -tags=dynamic
GO_TEST_PACKAGE_PARALLELISM ?= 128
GO_TEST_FLAGS = -p ${GO_TEST_PACKAGE_PARALLELISM} -parallel 16 ${GO_BUILD_FLAGS}
GO_LINT_PATH ?= ./...

JUST_TARGETS := \
	up \
	down \
	patch-oapi-templates \
	update-openapi \
	api-spec-generate \
	api-spec-format \
	api-spec-lint \
	gen-api \
	generate-all \
	generate \
	generate-view-sql \
	migrate-check \
	migrate-check-schema \
	migrate-check-diff \
	migrate-check-lint \
	migrate-check-validate \
	build-dir \
	build \
	build-server \
	build-sink-worker \
	build-benthos-collector \
	build-balance-worker \
	build-billing-worker \
	build-notification-service \
	build-jobs \
	config \
	server \
	sink-worker \
	balance-worker \
	billing-worker \
	notification-service \
	llm-cost-sync \
	e2e \
	etoe \
	e2e-test-local \
	e2e-env-local-down \
	e2e-env-local-up \
	e2e-slow \
	etoe-slow \
	quickstart-test-local \
	quickstart-env-local-down \
	quickstart-env-local-up \
	test \
	test-nocache \
	test-no-cache \
	test-all \
	lint \
	lint-go \
	lint-go-fast \
	lint-go-style \
	lint-go-head \
	lint-api-spec \
	lint-openapi \
	lint-helm \
	ci \
	fmt \
	mod \
	seed \
	charts-docs \
	publish-python-sdk \
	release

.PHONY: help $(JUST_TARGETS) generate-javascript-sdk publish-javascript-sdk generate-sqlc-testdata build-benthos-collector-release archive-benthos-collector-release package-helm-chart
.DEFAULT_GOAL := help

help:
	@if command -v "$(JUST)" >/dev/null 2>&1; then \
	  "$(JUST)" --list; \
	else \
	  echo "just is required for task help. Run via 'nix develop --impure .#ci -c just --list' or install just."; \
	fi

$(JUST_TARGETS):
	@if command -v "$(JUST)" >/dev/null 2>&1; then \
	  "$(JUST)" $@; \
	else \
	  echo "just is required for '$@'. Run via 'nix develop --impure .#ci -c just $@' or install just."; \
	  exit 127; \
	fi

generate-javascript-sdk:
	@if command -v "$(JUST)" >/dev/null 2>&1; then \
	  "$(JUST)" $@; \
	else \
	  cd api/client/javascript && pnpm --frozen-lockfile install && \
	  pnpm run generate && \
	  pnpm build && \
	  pnpm test; \
	fi

publish-javascript-sdk:
	@if command -v "$(JUST)" >/dev/null 2>&1; then \
	  "$(JUST)" $@; \
	else \
	  if [ -z "$$JS_SDK_RELEASE_VERSION" ]; then \
	    echo "ERROR: JS_SDK_RELEASE_VERSION is required"; \
	    echo "Usage: JS_SDK_RELEASE_VERSION=1.2.3 make publish-javascript-sdk [JS_SDK_RELEASE_TAG=beta]"; \
	    exit 1; \
	  fi; \
	  if [ -z "$$JS_SDK_RELEASE_TAG" ]; then \
	    echo "ERROR: JS_SDK_RELEASE_TAG is required"; \
	    echo "Usage: JS_SDK_RELEASE_VERSION=1.2.3 make publish-javascript-sdk [JS_SDK_RELEASE_TAG=beta]"; \
	    exit 1; \
	  fi; \
	  cd api/client/javascript && \
	  pnpm --frozen-lockfile install && \
	  pnpm version "$${JS_SDK_RELEASE_VERSION}" --no-git-tag-version && \
	  CACHE_BUSTER="$$(date +%s)" pnpm publish --no-git-checks --tag "$${JS_SDK_RELEASE_TAG}" && \
	  echo "Published JavaScript SDK version $${JS_SDK_RELEASE_VERSION} with tag $${JS_SDK_RELEASE_TAG}"; \
	fi

generate-sqlc-testdata:
	@VERSION="$(VERSION)" $(JUST) $@

build-benthos-collector-release:
	@GOOS="$(GOOS)" GOARCH="$(GOARCH)" VERSION="$(VERSION)" $(JUST) $@

archive-benthos-collector-release:
	@GOOS="$(GOOS)" GOARCH="$(GOARCH)" $(JUST) $@

package-helm-chart:
	@CHART="$(CHART)" VERSION="$(VERSION)" $(JUST) $@

config.yaml:
	cp config.example.yaml config.yaml

var-%:
	@echo $($*)

varexport-%:
	@echo $*=$($*)
