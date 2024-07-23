# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

.PHONY: up
up: ## Start the dependencies via docker compose
	$(call print-target)
	docker compose up -d

.PHONY: down
down: ## Stop the dependencies via docker compose
	$(call print-target)
	docker compose down --remove-orphans --volumes

.PHONY: gen-api
gen-api: ## Generate API and SDKs
	$(call print-target)
	go generate ./api/...
	dagger call --source .:default generate node-sdk -o api/client/node
	dagger call --source .:default generate web-sdk -o api/client/web
	dagger call --source .:default generate python-sdk -o api/client/python

.PHONY: generate
generate: ## Generate code
	$(call print-target)
	go generate ./...

.PHONY: build-server
build-server: ## Build server binary
	$(call print-target)
	go build -o build/server ./cmd/server

.PHONY: build-sink-worker
build-sink-worker: ## Build sink-worker binary
	$(call print-target)
	go build -o build/sink-worker ./cmd/sink-worker

.PHONY: build-benthos-collector
build-benthos-collector: ## Build benthos collector binary
	$(call print-target)
	go build -o build/benthos-collector ./cmd/benthos-collector

.PHONY: build-balance-worker
build-balance-worker: ## Build balance-worker binary
	$(call print-target)
	go build -o build/balance-worker ./cmd/balance-worker

.PHONY: build-notification-service
build-notification-service: ## Build notification-service binary
	$(call print-target)
	go build -o build/notification-service ./cmd/notification-service

config.yaml:
	cp config.example.yaml config.yaml

.PHONY: server
server: ## Run sink-worker
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air -c ./cmd/server/.air.toml

.PHONY: sink-worker
sink-worker: ## Run sink-worker
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air -c ./cmd/sink-worker/.air.toml

.PHONY: balance-worker
balance-worker: ## Run balance-worker
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air -c ./cmd/balance-worker/.air.toml

.PHONY: notification-service
notification-service: ## Run notification-service
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air -c ./cmd/notification-service/.air.toml

.PHONY: etoe
etoe: ## Run e2e tests
	$(call print-target)
	dagger call --source .:default etoe

.PHONY: test
test: ## Run tests
	$(call print-target)
	dagger call --source .:default test

.PHONY: lint
lint: ## Run linters
	$(call print-target)
	dagger call --source .:default lint all

.PHONY: fmt
fmt: ## Format code
	$(call print-target)
	golangci-lint run --fix

.PHONY: mod
mod: ## go mod tidy
	$(call print-target)
	go mod tidy

.PHONY: seed
seed: ## Seed OpenMeter with test data
	$(call print-target)
	benthos -c etc/seed/seed.yaml

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
