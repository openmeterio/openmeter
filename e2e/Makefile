# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

# Address of a running OpenMeter to benchmark against. Defaults to the local stack
# (env-local-up). Override on the CLI: make bench-governance OPENMETER_ADDRESS=...
OPENMETER_ADDRESS ?= http://localhost:38888
# Iterations per sub-benchmark. Each governance query is sequential; "20x" fixes the
# count so runs are comparable. Bump for tighter means, or use COUNT for variance.
BENCHTIME ?= 20x
# Number of times to repeat the whole benchmark (feed >1 into benchstat for variance).
COUNT ?= 1

.PHONY: bench-governance
bench-governance: ## Benchmark governance query latency (1x1 baseline + 10/50/100 diagonal)
	$(call print-target)
	TZ=UTC OPENMETER_ADDRESS=$(OPENMETER_ADDRESS) \
		go test -run='^$$' -bench='^BenchmarkGovernanceQuery$$' -benchmem -benchtime=$(BENCHTIME) -count=$(COUNT) ./...

.PHONY: bench-governance-matrix
bench-governance-matrix: ## Benchmark governance query latency (full 3x3 customers x features matrix)
	$(call print-target)
	TZ=UTC OPENMETER_ADDRESS=$(OPENMETER_ADDRESS) GOV_BENCH_FULL_MATRIX=1 \
		go test -run='^$$' -bench='^BenchmarkGovernanceQuery$$' -benchmem -benchtime=$(BENCHTIME) -count=$(COUNT) ./...

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
		-f docker-compose.debug-ports.yaml \
		-f docker-compose.openmeter.yaml \
		-f docker-compose.openmeter-local.yaml \
		down

.PHONY: env-local-up
env-local-up:
	$(call print-target)
	docker compose \
		-f docker-compose.infra.yaml \
		-f docker-compose.debug-ports.yaml \
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
