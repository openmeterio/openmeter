# Compatibility shim. Canonical task definitions live in ../../../justfile.

JUST ?= just

.PHONY: help publish-python-sdk
.DEFAULT_GOAL := help

help:
	@if command -v "$(JUST)" >/dev/null 2>&1; then "$(JUST)" --list; else echo "just is required for task help."; fi

publish-python-sdk:
	@$(JUST) publish-python-sdk
