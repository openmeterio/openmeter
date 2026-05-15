# scripts

<!-- archie:ai-start -->

> Contains the single release script (release.sh) that computes the next PEP-440 pre-release version by querying live PyPI state, stamps it into pyproject.toml and openmeter/_version.py/_commit.py, then invokes `poetry publish --build`. Its primary constraint: only 'alpha' is a valid PY_SDK_RELEASE_TAG value; any other tag hard-errors.

## Patterns

**PEP-440 alpha versioning via live PyPI query** — When PY_SDK_RELEASE_VERSION is absent, the script queries PyPI for the latest alpha release, extracts the pre-release number, increments it, and constructs the next version (e.g. 1.0.0a3). Any new release automation must follow this monotonic increment pattern. (`LATEST_VERSION=$(curl -s https://pypi.org/pypi/openmeter/json | jq -r '.releases | keys[] | select(test("a[0-9]+"))' | sort -V | tail -1)`)
**Version normalisation before poetry invocation** — Before calling `poetry version`, strip a leading 'v' and convert semver pre-release suffixes (-alpha, -beta) to PEP-440 equivalents (a, b) via sed. Any injected PY_SDK_RELEASE_VERSION must survive this sed pipeline without unintended mutation. (`export PY_SDK_RELEASE_VERSION=$(echo "$PY_SDK_RELEASE_VERSION" | sed -E 's/^v//' | sed -E 's/-alpha\.?/a/; s/-beta\.?/b/;')`)
**Unconditional dist clean before publish** — `rm -rf dist` runs before `poetry publish --build` to prevent poetry's interactive prompt about overwriting existing artifacts. Any pre-publish build step added must not recreate dist before this clean. (`rm -rf dist
poetry publish --build --no-interaction`)
**Version/commit stamped into package at build time** — The script writes openmeter/_version.py and openmeter/_commit.py as transient side-effects before the build. These files must not be committed to source control. The `|| true` silently ignores write failures if the openmeter/ directory is missing. (`printf "VERSION = \"%s\"" "$PY_SDK_RELEASE_VERSION" > openmeter/_version.py || true
printf "COMMIT = \"%s\"" "$COMMIT_SHORT_SHA" > openmeter/_commit.py || true`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `release.sh` | Single-entry-point release script invoked by CI (sdk-python-dev-release.yaml) with PY_SDK_RELEASE_TAG=alpha or a pre-set PY_SDK_RELEASE_VERSION. Computes version, stamps files, runs `poetry publish --build`. | The `|| true` on printf lines silently ignores write failures for _version.py and _commit.py — a missing openmeter/ directory will not abort the script. Only 'alpha' is accepted as PY_SDK_RELEASE_TAG; any other value exits with error. COMMIT_SHORT_SHA falls back to `git rev-parse --short=12 HEAD` if not set by CI. |

## Anti-Patterns

- Committing openmeter/_version.py or openmeter/_commit.py — they are written transiently by release.sh and must not be in source control
- Setting PY_SDK_RELEASE_TAG to anything other than 'alpha' — the script hard-errors on unknown tags
- Calling `poetry build` separately before this script — dist artifacts left behind will trigger an interactive poetry prompt that hangs CI
- Injecting a PY_SDK_RELEASE_VERSION with unusual formats (e.g. 'v1.0.0-alpha.1') without verifying the sed pipeline produces a valid PEP-440 string
- Adding a pre-publish build step that recreates the dist/ directory after the `rm -rf dist` clean

## Decisions

- **Version computed from live PyPI state rather than git tags** — Alpha releases are continuous and not tied to formal git tags; querying PyPI ensures the next published version is always monotonically incremented even if the CI job is re-run.
- **poetry used for both build and publish in a single command** — Keeps the release pipeline to one tool invocation, avoiding a separate `poetry build` step that would require managing dist artifact cleanup independently.

<!-- archie:ai-end -->
