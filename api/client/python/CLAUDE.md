# python

<!-- archie:ai-start -->

> Root of the public OpenMeter Python SDK: a generated client (TypeSpec via @typespec/http-client-python) plus a thin hand-authored shim that re-exports it under stable, version-pinned import paths. Its primary constraint is that everything under openmeter/_generated/ is overwritten by make gen-api, so only the shim layer (Client subclass, _types.py aliases) and tooling (release.sh, examples) may be hand-edited.

## Patterns

**Generated core, hand-authored shim only** — Public surface is a thin re-export shim over openmeter/_generated/; no operations or business logic are defined outside _generated/. New API methods arrive only via make gen-api regeneration. (`from openmeter._generated import OpenMeterClient
class Client(OpenMeterClient): ...`)
**release.sh is the sole publish path** — Makefile target publish-python-sdk delegates entirely to scripts/release.sh, which computes the next PEP-440 alpha version from live PyPI, stamps _version.py/_commit.py, cleans dist/, and runs poetry publish --build. (`make publish-python-sdk  # -> ./scripts/release.sh`)
**Transient build-time version stamping** — openmeter/_version.py and openmeter/_commit.py are written by release.sh at publish time and must never be committed; pyproject.toml version (0.0.0) is also overwritten by the script. (`# _version.py / _commit.py are .gitignore-adjacent build artifacts`)
**Sync/async client import discipline (examples)** — examples/sync/ uses openmeter.Client; examples/async/ uses openmeter.aio.Client. The two trees are structurally identical and must never cross-import. (`from openmeter.aio import Client  # async/ only`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `Makefile` | Single phony target publish-python-sdk wrapping scripts/release.sh; default goal is self-documenting help. | Adding build logic here instead of in release.sh — publish flow must stay in one script. |
| `pyproject.toml` | Poetry project definition: python ^3.9, corehttp[requests,aiohttp] >=1.0.0b7, cloudevents ^1.12.1, version 0.0.0 placeholder. | Manually bumping version — release.sh overwrites it at publish time. |
| `MANIFEST.in` | Packaging manifest; explicitly includes openmeter/py.typed and README/LICENSE. | Removing openmeter/py.typed — breaks PEP-561 type checking for downstream consumers. |
| `.gitignore` | Ignores dist/, build/, egg-info, caches, and venvs for the SDK build. | Leftover dist/ artifacts cause poetry publish --build to hang on an interactive prompt in CI. |
| `scripts/release.sh` | Sole publish script (child scripts/): PyPI-derived alpha versioning, stamps version/commit, cleans dist, poetry publish --build. | PY_SDK_RELEASE_TAG other than 'alpha' hard-errors; a 'v'-prefixed PY_SDK_RELEASE_VERSION can yield invalid PEP-440. |
| `openmeter/__init__.py` | Public package surface (child openmeter/): re-exports Client/models/operations; must call _patch_sdk() last. | Missing or relocated _patch_sdk(); extending __all__ without including _patch_all. |

## Anti-Patterns

- Hand-editing files under openmeter/_generated/ — overwritten by make gen-api
- Committing openmeter/_version.py or openmeter/_commit.py — written transiently by release.sh
- Defining new operation classes/methods in openmeter/__init__.py or _client.py instead of regenerating _generated/
- Running a separate poetry build before release.sh — leftover dist/ artifacts hang CI
- Cross-importing sync (openmeter) into async examples or openmeter.aio into sync examples

## Decisions

- **Public Client subclasses generated OpenMeterClient rather than wrapping it** — Subclassing preserves all generated type signatures and avoids re-exporting every method through proxy boilerplate.
- **Release version computed from live PyPI state in release.sh, not from git tags** — Allows independent alpha pre-release numbering decoupled from the monorepo tag cadence.
- **Pyright-only type checking (no mypy)** — The generated SDK's overloaded constructors are incompatible with mypy's strict mode; Pyright handles them correctly.

## Example: Canonical SDK usage: ingest a CloudEvent with the async client

```
import asyncio, datetime, os, uuid
from openmeter.aio import Client
from openmeter.models import Event

async def main():
    async with Client(
        endpoint=os.environ['OPENMETER_ENDPOINT'],
        token=os.environ['OPENMETER_TOKEN'],
    ) as client:
        await client.events.ingest_event(Event(
            id=str(uuid.uuid4()), source='my-app', specversion='1.0',
            type='prompt', subject='customer-1',
            time=datetime.datetime.now(datetime.timezone.utc),
            data={'tokens': 100, 'model': 'gpt-4o', 'type': 'input'}))

// ...
```

<!-- archie:ai-end -->
