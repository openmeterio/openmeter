# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

.PHONY: generate
generate: ## Generate OpenAPI spec
	$(call print-target)
	pnpm --frozen-lockfile install
	pnpm compile
	cp output/openapi.OpenMeter.yaml ../openapi.yaml
	cp output/openapi.OpenMeterCloud.yaml ../openapi.cloud.yaml
	$(MAKE) hack

### Remove the generated fake path '/prune-this-in-script:' without touching YAML formatting.
# In plain English:
# - Awk reads the file line by line
# - We keep a simple `skip` flag to decide whether to print a line.
# - Once we hit a line that looks like the path key (can be indented), we flip `skip` on and start ignoring lines.
# - We keep skipping until we reach the next `components:` line
# - Whenever `skip` is off, we print the line.
.PHONY: hack
hack: ## Hack the OpenAPI spec
	awk 'BEGIN{skip=0} /^[[:space:]]*\/prune-this-in-script:/ {skip=1; next} skip && /^components:/ {skip=0} !skip' ../openapi.yaml > ../openapi.yaml.tmp && mv ../openapi.yaml.tmp ../openapi.yaml
	awk 'BEGIN{skip=0} /^[[:space:]]*\/prune-this-in-script:/ {skip=1; next} skip && /^components:/ {skip=0} !skip' ../openapi.cloud.yaml > ../openapi.cloud.yaml.tmp && mv ../openapi.cloud.yaml.tmp ../openapi.cloud.yaml

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
