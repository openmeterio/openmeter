# Compatibility shim. Canonical task definitions live in ../../../Taskfile.yml.

TASK ?= task
TASK_ROOT := ../../..

.PHONY: help publish-python-sdk
.DEFAULT_GOAL := help

help:
	@if command -v "$(TASK)" >/dev/null 2>&1; then "$(TASK)" --dir "$(TASK_ROOT)" --list; else echo "task is required for task help."; fi

publish-python-sdk:
	@$(TASK) --dir "$(TASK_ROOT)" publish-python-sdk
