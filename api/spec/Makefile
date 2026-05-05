# Compatibility shim. Canonical task definitions live in ../../Taskfile.yml.

TASK ?= task
TASK_ROOT := ../..

.PHONY: help generate lint format
.DEFAULT_GOAL := help

help:
	@if command -v "$(TASK)" >/dev/null 2>&1; then "$(TASK)" --dir "$(TASK_ROOT)" --list; else echo "task is required for task help."; fi

generate:
	@$(TASK) --dir "$(TASK_ROOT)" api-spec-generate

lint:
	@$(TASK) --dir "$(TASK_ROOT)" api-spec-lint

format:
	@$(TASK) --dir "$(TASK_ROOT)" api-spec-format
