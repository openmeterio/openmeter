# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

.PHONY: generate
generate: ## Generate OpenAPI spec
	$(call print-target)
	pnpm --frozen-lockfile install
	pnpm compile
	pnpm run bundle:v3 -o ../v3/openapi.yaml
	cp output/openapi.OpenMeter.yaml ../openapi.yaml
	cp output/openapi.OpenMeterCloud.yaml ../openapi.cloud.yaml

.PHONY: lint
lint: ## Lint OpenAPI spec
	$(call print-target)
	pnpm --frozen-lockfile install
	pnpm lint

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
