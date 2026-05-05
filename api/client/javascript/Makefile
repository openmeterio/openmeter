# Compatibility shim. Canonical task definitions live in ../../../Taskfile.yml.

TASK ?= task
TASK_ROOT := ../../..

.PHONY: help generate publish-javascript-sdk
.DEFAULT_GOAL := help

help:
	@if command -v "$(TASK)" >/dev/null 2>&1; then "$(TASK)" --dir "$(TASK_ROOT)" --list; else echo "task is required for task help."; fi

generate:
	@if command -v "$(TASK)" >/dev/null 2>&1; then \
	  "$(TASK)" --dir "$(TASK_ROOT)" generate-javascript-sdk; \
	else \
	  pnpm --frozen-lockfile install && \
	  pnpm run generate && \
	  pnpm build && \
	  pnpm test; \
	fi

publish-javascript-sdk:
	@if command -v "$(TASK)" >/dev/null 2>&1; then \
	  "$(TASK)" --dir "$(TASK_ROOT)" publish-javascript-sdk; \
	else \
	  if [ -z "$$JS_SDK_RELEASE_VERSION" ]; then \
	    echo "ERROR: JS_SDK_RELEASE_VERSION is required"; \
	    echo "Usage: JS_SDK_RELEASE_VERSION=1.2.3 make publish-javascript-sdk [JS_SDK_RELEASE_TAG=beta]"; \
	    exit 1; \
	  fi; \
	  if [ -z "$$JS_SDK_RELEASE_TAG" ]; then \
	    echo "ERROR: JS_SDK_RELEASE_TAG is required"; \
	    echo "Usage: JS_SDK_RELEASE_VERSION=1.2.3 make publish-javascript-sdk [JS_SDK_RELEASE_TAG=beta]"; \
	    exit 1; \
	  fi; \
	  pnpm --frozen-lockfile install && \
	  pnpm version "$${JS_SDK_RELEASE_VERSION}" --no-git-tag-version && \
	  CACHE_BUSTER="$$(date +%s)" pnpm publish --no-git-checks --tag "$${JS_SDK_RELEASE_TAG}" && \
	  echo "Published JavaScript SDK version $${JS_SDK_RELEASE_VERSION} with tag $${JS_SDK_RELEASE_TAG}"; \
	fi
