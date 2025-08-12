# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

.PHONY: generate
generate: ## Generate JavaScript SDK
	$(call print-target)
	pnpm --frozen-lockfile install
	pnpm run generate
	pnpm build
	pnpm test

# Warning: keep in sync with .dagger/release.go for now
.PHONY: publish-javascript-sdk
publish-javascript-sdk: ## Publish JavaScript SDK
	$(call print-target)
	@if [ -z "$$JS_SDK_RELEASE_VERSION" ]; then \
		echo "ERROR: JS_SDK_RELEASE_VERSION is required"; \
		echo "Usage: JS_SDK_RELEASE_VERSION=1.2.3 NPM_TOKEN=xxxx make publish-js-sdk [JS_SDK_RELEASE_TAG=beta]"; \
		exit 1; \
	fi

	@if [ -z "$$JS_SDK_RELEASE_TAG" ]; then \
		echo "ERROR: JS_SDK_RELEASE_TAG is required"; \
		echo "Usage: JS_SDK_RELEASE_VERSION=1.2.3 NPM_TOKEN=xxxx make publish-js-sdk [JS_SDK_RELEASE_TAG=beta]"; \
		exit 1; \
	fi

	pnpm --frozen-lockfile install
	pnpm version $${JS_SDK_RELEASE_VERSION} --no-git-tag-version
	CACHE_BUSTER=$$(date --rfc-3339=seconds) pnpm publish --no-git-checks --tag $${JS_SDK_RELEASE_TAG}
	@echo "✅ Published $${JS_SDK_RELEASE_TAG} JavaScript SDK version $${JS_SDK_RELEASE_VERSION} with tag $${JS_SDK_RELEASE_TAG}"

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
