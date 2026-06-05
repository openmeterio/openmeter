# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

.PHONY: generate
generate: format ## Generate OpenAPI spec
	$(call print-target)
	pnpm --frozen-lockfile install
	pnpm generate
	# Replace inline filter definitions with $ref to common/definitions/aip_filters.yaml
	@AIP_REF="../../../../common/definitions/aip_filters.yaml#/components/schemas"; \
	FILE="packages/aip/output/definitions/metering-and-billing/v3/openapi.MeteringAndBilling.yaml"; \
	for schema in SortQuery BooleanFieldFilter NumericFieldFilter StringFieldFilter StringFieldFilterExact DateTimeFieldFilter LabelsFieldFilter; do \
		if yq -e ".components.schemas | has(\"$$schema\")" "$$FILE" > /dev/null 2>&1; then \
			REF_VAL="$$AIP_REF/$$schema" SCHEMA="$$schema" yq -i '.components.schemas[strenv(SCHEMA)] = {"$$ref": strenv(REF_VAL)}' "$$FILE"; \
		fi; \
	done
	pnpm --filter @openmeter/api-spec-aip exec openapi bundle output/definitions/metering-and-billing/v3/openapi.OpenMeter.yaml -o ../../../v3/openapi.yaml
	cp packages/legacy/output/openapi.OpenMeter.yaml ../openapi.yaml
	cp packages/legacy/output/openapi.OpenMeterCloud.yaml ../openapi.cloud.yaml
	cp packages/aip/output/definitions/metering-and-billing/v3/openapi.Test.yaml ../v3/test/openapi.test.yaml

.PHONY: publish-aip-sdk
publish-aip-sdk: ## Publish the AIP TypeScript SDK (@openmeter/client) to npm
	$(call print-target)
	@if [ -z "$$AIP_SDK_RELEASE_VERSION" ]; then \
		echo "ERROR: AIP_SDK_RELEASE_VERSION is required"; \
		echo "Usage: AIP_SDK_RELEASE_VERSION=1.2.3 make publish-aip-sdk [AIP_SDK_RELEASE_TAG=beta]"; \
		exit 1; \
	fi

	@if [ -z "$$AIP_SDK_RELEASE_TAG" ]; then \
		echo "ERROR: AIP_SDK_RELEASE_TAG is required"; \
		echo "Usage: AIP_SDK_RELEASE_VERSION=1.2.3 make publish-aip-sdk [AIP_SDK_RELEASE_TAG=beta]"; \
		exit 1; \
	fi

	pnpm --frozen-lockfile install

	cd packages/aip-client-javascript && \
		pnpm version "$${AIP_SDK_RELEASE_VERSION}" --no-git-tag-version && \
		CACHE_BUSTER="$$(date --rfc-3339=seconds)" pnpm publish --no-git-checks --access public --tag "$${AIP_SDK_RELEASE_TAG}"
	@echo "✅ Published $${AIP_SDK_RELEASE_TAG} AIP TypeScript SDK (@openmeter/client) version $${AIP_SDK_RELEASE_VERSION} with tag $${AIP_SDK_RELEASE_TAG}"

.PHONY: lint
lint: ## Lint OpenAPI spec
	$(call print-target)
	pnpm --frozen-lockfile install
	pnpm lint

.PHONY: format
format: ## Format OpenAPI spec
	$(call print-target)
	pnpm --frozen-lockfile install
	pnpm format

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
