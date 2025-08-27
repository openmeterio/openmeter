# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

.PHONY: test-local
test-local: ## Run tests against local openmeter
	$(call print-target)
	$(MAKE) env-local-down
	$(MAKE) env-local-up

	# wait for sink-worker to be ready
	curl --retry 10 --retry-max-time 120 --retry-all-errors http://localhost:30000/healthz

	TZ=UTC OPENMETER_ADDRESS=http://localhost:38888 go test -count=1 -v ./...

	$(MAKE) env-local-down

.PHONY: env-local-down
env-local-down:
	$(call print-target)
	docker compose \
		-f docker-compose.infra.yaml \
		-f docker-compose.openmeter.yaml \
		-f docker-compose.openmeter-local.yaml \
		down

.PHONY: env-local-up
env-local-up:
	$(call print-target)
	docker compose \
		-f docker-compose.infra.yaml \
		-f docker-compose.openmeter.yaml \
		-f docker-compose.openmeter-local.yaml \
		up -d --build --force-recreate

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
