# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

.PHONY: autorest-python
autorest-python: ## Generate autorest-python
	$(call print-target)
	docker build -t ghcr.io/openmeterio/autorest-python:latest .
	docker push ghcr.io/openmeterio/autorest-python:latest

.PHONY: generate
generate: ## Generate code
	$(call print-target)
	cp ../../openapi.yaml .
	docker run \
		--rm \
		-v "$$(pwd):/workdir" \
		-w /workdir \
		-t ghcr.io/openmeterio/autorest-python:latest config.yaml
	rm openapi.yaml

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
