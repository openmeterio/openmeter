# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

# Address of a running OpenMeter to benchmark against. Defaults to the local stack
# (env-local-up). Override on the CLI: make bench-governance OPENMETER_ADDRESS=...
OPENMETER_ADDRESS ?= http://localhost:38888
# Iterations per sub-benchmark. Each governance query is sequential; "20x" fixes the
# count so runs are comparable. Bump for tighter means, or use COUNT for variance.
BENCHTIME ?= 20x
# Number of times to repeat the whole benchmark (feed >1 into benchstat for variance).
COUNT ?= 1
# Entitlement kind the seeded features use: boolean (default, cheap algorithmic baseline),
# metered (forces the ClickHouse balance query in GetAccess — production-representative),
# mixed (~50/50 boolean+metered), or all (runs all three kinds back to back).
GOV_BENCH_KIND ?= boolean
OPENMETER_E2E_EXTRA_COMPOSE_FILES ?=

.PHONY: bench-governance
bench-governance: ## Benchmark governance query latency (1x1 baseline + 10/50/100 diagonal)
	$(call print-target)
	TZ=UTC OPENMETER_ADDRESS=$(OPENMETER_ADDRESS) GOV_BENCH_KIND=$(GOV_BENCH_KIND) \
		go test -run='^$$' -bench='^BenchmarkGovernanceQuery$$' -benchmem -benchtime=$(BENCHTIME) -count=$(COUNT) ./...

.PHONY: bench-governance-matrix
bench-governance-matrix: ## Benchmark governance query latency (full 3x3 customers x features matrix)
	$(call print-target)
	TZ=UTC OPENMETER_ADDRESS=$(OPENMETER_ADDRESS) GOV_BENCH_FULL_MATRIX=1 GOV_BENCH_KIND=$(GOV_BENCH_KIND) \
		go test -run='^$$' -bench='^BenchmarkGovernanceQuery$$' -benchmem -benchtime=$(BENCHTIME) -count=$(COUNT) ./...

.PHONY: bench-governance-metered
bench-governance-metered: ## Benchmark governance query latency, metered entitlements (diagonal)
	$(MAKE) bench-governance GOV_BENCH_KIND=metered

.PHONY: bench-governance-matrix-metered
bench-governance-matrix-metered: ## Benchmark governance query latency, metered entitlements (full 3x3 matrix)
	$(MAKE) bench-governance-matrix GOV_BENCH_KIND=metered

# --- Trace capture ---
# Capturing traces for big metered cells (50x50, 100x100) OOMs the local Tempo and the
# traces are too large to open anyway (>16MiB query cap). This target captures ONE SMALL
# cell, which shows the SAME per-entitlement query structure (query counts are per
# entitlement). Low benchtime keeps the number of huge traces ingested down.
# Override: TRACE_KIND=mixed, BENCH_CELL=customers=1/features=1, TRACE_BENCHTIME=1x.
TRACE_KIND ?= metered
BENCH_CELL ?= customers=10/features=10
TRACE_BENCHTIME ?= 2x

.PHONY: bench-governance-trace
bench-governance-trace: ## Capture traces for ONE small cell (default metered 10x10, 2x) — safe for local Tempo
	$(call print-target)
	TZ=UTC OPENMETER_ADDRESS=$(OPENMETER_ADDRESS) GOV_BENCH_KIND=$(TRACE_KIND) \
		go test -run='^$$' -bench='^BenchmarkGovernanceQuery$$/kind=$(TRACE_KIND)/$(BENCH_CELL)$$' -benchmem -benchtime=$(TRACE_BENCHTIME) -count=1 ./...

# --- Over-compute demonstration ---
# Selective query: seed 50 entitlements/customer but request only 10. Reveals that GetAccess
# computes all 50 balances (50 ClickHouse queries) while the result uses 10 — cost scales with a
# customer's TOTAL entitlements, not what the query asks for. The regular cells hide this (they
# request every seeded feature). Requires GOV_BENCH_OVERCOMPUTE=1 (set by these targets).
.PHONY: bench-governance-overcompute
bench-governance-overcompute: ## Latency: selective query (50 seeded / 10 queried), 1 and 10 customers
	$(call print-target)
	TZ=UTC OPENMETER_ADDRESS=$(OPENMETER_ADDRESS) GOV_BENCH_KIND=metered GOV_BENCH_OVERCOMPUTE=1 \
		go test -run='^$$' -bench='^BenchmarkGovernanceQuery$$/kind=metered/' -benchmem -benchtime=$(BENCHTIME) -count=$(COUNT) ./...

.PHONY: bench-governance-overcompute-trace
bench-governance-overcompute-trace: ## Capture ONE trace of the over-compute cell (1 customer, 50 seeded, 10 queried)
	$(call print-target)
	TZ=UTC OPENMETER_ADDRESS=$(OPENMETER_ADDRESS) GOV_BENCH_KIND=metered GOV_BENCH_OVERCOMPUTE=1 \
		go test -run='^$$' -bench='^BenchmarkGovernanceQuery$$/kind=metered/customers=1$$/features=50$$/query=10$$' -benchmem -benchtime=$(TRACE_BENCHTIME) -count=1 ./...

.PHONY: test-local
test-local: ## Run tests against local openmeter
	$(call print-target)
	$(MAKE) test-local-base
	$(MAKE) test-local-credits-disabled

.PHONY: test-local-credits-disabled
test-local-credits-disabled: ## Run credits-disabled e2e tests against local openmeter
	$(call print-target)
	$(MAKE) env-local-down
	set -e; trap '$(MAKE) env-local-down' EXIT; \
		OPENMETER_E2E_EXTRA_COMPOSE_FILES="-f docker-compose.credits-disabled.yaml" $(MAKE) env-local-up; \
		$(MAKE) wait-local; \
		$(MAKE) test-credits-disabled

.PHONY: test-local-base
test-local-base: ## Run default e2e tests against local openmeter
	$(call print-target)
	$(MAKE) env-local-down
	set -e; trap '$(MAKE) env-local-down' EXIT; \
		$(MAKE) env-local-up; \
		$(MAKE) wait-local; \
		$(MAKE) test-base

.PHONY: test-credits-disabled
test-credits-disabled: ## Run e2e tests that require credits.enabled=false
	$(call print-target)
	TZ=UTC OPENMETER_ADDRESS=$(OPENMETER_ADDRESS) go test -count=1 -v ./creditsdisabled

.PHONY: test-base
test-base: ## Run the default e2e test package
	$(call print-target)
	TZ=UTC OPENMETER_ADDRESS=$(OPENMETER_ADDRESS) go test -count=1 -v ./

.PHONY: wait-local
wait-local:
	$(call print-target)
	curl --fail --retry 10 --retry-max-time 120 --retry-all-errors http://localhost:30000/healthz
	curl --fail --retry 10 --retry-max-time 120 --retry-all-errors http://localhost:30001/healthz

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
		$(OPENMETER_E2E_EXTRA_COMPOSE_FILES) \
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
