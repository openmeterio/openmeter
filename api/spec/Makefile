# Compatibility shim. Canonical task definitions live in ../../justfile.

JUST ?= just

.PHONY: help generate lint format
.DEFAULT_GOAL := help

help:
	@if command -v "$(JUST)" >/dev/null 2>&1; then "$(JUST)" --list; else echo "just is required for task help."; fi

generate:
	@$(JUST) api-spec-generate

lint:
	@$(JUST) api-spec-lint

format:
	@$(JUST) api-spec-format
