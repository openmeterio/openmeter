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
