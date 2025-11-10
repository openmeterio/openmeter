#!/usr/bin/env sh

set -euo pipefail

# Determine PY_SDK_RELEASE_VERSION if not provided
if [ -z "${PY_SDK_RELEASE_VERSION:-}" ]; then
  # Validate PY_SDK_RELEASE_TAG
  if [ -z "${PY_SDK_RELEASE_TAG:-}" ]; then
    echo "ERROR: PY_SDK_RELEASE_VERSION or PY_SDK_RELEASE_TAG is required"
    exit 1
  fi

  if [ "$PY_SDK_RELEASE_TAG" != "alpha" ]; then
    echo "ERROR: PY_SDK_RELEASE_TAG must be 'alpha'"
    exit 1
  fi

	LATEST_VERSION=$(curl -s https://pypi.org/pypi/openmeter/json | jq -r '.releases | keys[] | select(test("a[0-9]+"))' | sort -V | tail -1)
	if [ -z "$LATEST_VERSION" ]; then
		PY_SDK_RELEASE_VERSION="1.0.0a0"
	else
		BASE_VERSION=$(echo "$LATEST_VERSION" | grep -o '^[0-9]*\.[0-9]*\.[0-9]*')
		PRE_NUM=$(echo "$LATEST_VERSION" | grep -o 'a[0-9]*' | grep -o '[0-9]*' || echo "-1")
		NEXT_NUM=$((PRE_NUM + 1))
		PY_SDK_RELEASE_VERSION="${BASE_VERSION}a${NEXT_NUM}"
	fi
	export PY_SDK_RELEASE_VERSION
fi

# Set COMMIT_SHORT_SHA if not provided
if [ -z "${COMMIT_SHORT_SHA:-}" ]; then
	COMMIT_SHORT_SHA=$(git rev-parse --short=12 HEAD)
fi

# Convert PY_SDK_RELEASE_VERSION to a valid Python version
export PY_SDK_RELEASE_VERSION=$(echo "$PY_SDK_RELEASE_VERSION" | sed -E 's/^v//' | sed -E 's/-alpha\.?/a/; s/-beta\.?/b/;')

# Update poetry version
poetry version "$PY_SDK_RELEASE_VERSION"

# Write version and commit files
printf "VERSION = \"%s\"" "$PY_SDK_RELEASE_VERSION" > openmeter/_version.py || true
printf "COMMIT = \"%s\"" "$COMMIT_SHORT_SHA" > openmeter/_commit.py || true

# Clean dist directory to avoid prompts about existing files
rm -rf dist

# Publish with poetry
poetry publish --build --no-interaction

echo "Published Python SDK version $PY_SDK_RELEASE_VERSION"

