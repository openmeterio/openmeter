# Compatibility shim. Canonical task definitions live in ../Taskfile.yml.

TASK ?= task
TASK_ROOT := ..

.PHONY: help test-local env-local-down env-local-up
.DEFAULT_GOAL := help

help:
	@if command -v "$(TASK)" >/dev/null 2>&1; then "$(TASK)" --dir "$(TASK_ROOT)" --list; else echo "task is required for task help."; fi

test-local:
	@$(TASK) --dir "$(TASK_ROOT)" e2e-test-local

env-local-down:
	@$(TASK) --dir "$(TASK_ROOT)" e2e-env-local-down

env-local-up:
	@$(TASK) --dir "$(TASK_ROOT)" e2e-env-local-up
