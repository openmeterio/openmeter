# scripts

<!-- archie:ai-start -->

> Contains the single release script that versions and publishes the Python SDK to PyPI. Its only job is to compute the next PEP-440 pre-release version, stamp it into pyproject.toml and openmeter/_version.py/_commit.py, then invoke `poetry publish --build`.

## Patterns

**PEP-440 alpha versioning via PyPI query** — When PY_SDK_RELEASE_VERSION is absent and PY_SDK_RELEASE_TAG=alpha, the script queries PyPI for the latest alpha release of the 'openmeter' package, extracts the pre-release number, increments it, and constructs the next version (e.g. 1.0.0a3). Any new release automation must follow this same increment pattern. (`LATEST_VERSION=$(curl -s https://pypi.org/pypi/openmeter/json | jq -r '.releases | keys[] | select(test("a[0-9]+"))' | sort -V | tail -1)`)
**Version normalisation before poetry** — Before calling `poetry version`, the script strips a leading 'v' and converts semver pre-release suffixes (-alpha, -beta) to PEP-440 equivalents (a, b) via sed. Any caller that injects PY_SDK_RELEASE_VERSION must supply a value that survives this sed pipeline without unintended mutation. (`export PY_SDK_RELEASE_VERSION=$(echo "$PY_SDK_RELEASE_VERSION" | sed -E 's/^v//' | sed -E 's/-alpha\.?/a/; s/-beta\.?/b/;')`)
**Explicit dist clean before publish** — `rm -rf dist` is run unconditionally before `poetry publish --build` to prevent poetry's interactive prompt about overwriting existing dist artifacts. Any addition of a pre-publish build step must not recreate dist before this clean. (`rm -rf dist
poetry publish --build --no-interaction`)
**Version/commit stamped into openmeter package at build time** — The script writes openmeter/_version.py and openmeter/_commit.py as side-effects before the build. These files must not be committed to source control; they are ephemeral outputs of the release pipeline. (`printf "VERSION = \"%s\"" "$PY_SDK_RELEASE_VERSION" > openmeter/_version.py || true`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `release.sh` | Single-entry-point release script: computes version, stamps files, runs `poetry publish --build`. Invoked by CI (sdk-python-dev-release.yaml) with PY_SDK_RELEASE_TAG=alpha or a pre-set PY_SDK_RELEASE_VERSION. | The `|| true` on the printf lines silently ignores write failures for _version.py and _commit.py — a missing openmeter/ package directory will not abort the script. Also note only 'alpha' is accepted as PY_SDK_RELEASE_TAG; any other tag value exits with an error. |

## Anti-Patterns

- Committing openmeter/_version.py or openmeter/_commit.py — they are written transiently by this script
- Setting PY_SDK_RELEASE_TAG to anything other than 'alpha' — the script hard-errors on unknown tags
- Calling `poetry build` separately before this script — dist artifacts left behind will trigger an interactive poetry prompt that hangs CI
- Injecting a PY_SDK_RELEASE_VERSION with a leading 'v' and relying on it to be stripped without testing the sed expression — versions like 'v1.0.0-alpha.1' pass through, but unusual formats may silently produce invalid PEP-440 strings

## Decisions

- **Version computed from live PyPI state rather than from git tags** — Alpha releases are continuous and not tied to formal git tags; querying PyPI ensures the next published version is always monotonically incremented even if the CI job is re-run.
- **poetry used for both build and publish in a single command** — Keeps the release pipeline to one tool invocation, avoiding a separate `poetry build` step that would require managing dist artifact cleanup independently.

<!-- archie:ai-end -->
