# Compatibility shim. Canonical task definitions live in ../justfile.

JUST ?= just

.PHONY: help test-local env-local-down env-local-up
.DEFAULT_GOAL := help

help:
	@if command -v "$(JUST)" >/dev/null 2>&1; then "$(JUST)" --list; else echo "just is required for task help."; fi

test-local:
	@$(JUST) quickstart-test-local

env-local-down:
	@$(JUST) quickstart-env-local-down

env-local-up:
	@$(JUST) quickstart-env-local-up
