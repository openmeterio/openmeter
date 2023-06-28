# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

.PHONY: up
up: ## Start the dependencies via docker compose
	$(call print-target)
	docker compose up -d

.PHONY: down
down: ## Stop the dependencies via docker compose
	$(call print-target)
	docker compose down --remove-orphans --volumes

.PHONY: generate
generate: ## Generate code
	$(call print-target)
	go generate ./...

.PHONY: build
build: ## Build binary
	$(call print-target)
	go build -tags dynamic -o build/ .

config.yaml:
	cp config.example.yaml config.yaml

run: config.yaml
run: ## Run OpenMeter
	@ if [ config.yaml -ot config.example.yaml ]; then diff -u config.yaml config.example.yaml || (echo "!!! The configuration example changed. Please update your config.yaml file accordingly (or at least touch it). !!!" && false); fi
	$(call print-target)
	air

.PHONY: test
test: ## Run tests
	$(call print-target)
	dagger run mage -d ci -w . test

.PHONY: lint
lint: ## Run linters
	$(call print-target)
	dagger run mage -d ci -w . lint

.PHONY: fmt
fmt: ## Format code
	$(call print-target)
	golangci-lint run --fix

.PHONY: mod
mod: ## go mod tidy
	$(call print-target)
	go mod tidy

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
