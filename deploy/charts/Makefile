# Compatibility shim. Canonical task definitions live in ../../Taskfile.yml.

TASK ?= task
TASK_ROOT := ../..

.PHONY: docs

docs:
	@$(TASK) --dir "$(TASK_ROOT)" charts-docs
