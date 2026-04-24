# python

<!-- archie:ai-start -->

> Public Python SDK for OpenMeter — a thin shim layer re-exporting from _generated/ sub-packages with a stable versioned surface. The primary constraint is that all business logic and operations must come from _generated/ only; this folder just wires them together.

## Patterns

**Thin shim over _generated** — openmeter/__init__.py and _client.py re-export from _generated/; no new operation logic is added here. (`from openmeter._generated import OpenMeterClient; class Client(OpenMeterClient): ...`)
**_patch_sdk() at module end** — _patch_sdk() must be called at the end of __init__.py to apply runtime patches from _patch.py; removing or moving it silently disables patches. (`_patch_sdk()`)
**Graceful _patch ImportError guard** — Imports of _patch symbols must be wrapped in try/except ImportError so the SDK still works when _patch.py is absent. (`try:
    from ._patch import *  # type: ignore
except ImportError:
    pass`)
**Poetry for build and publish** — scripts/release.sh computes PEP-440 version from live PyPI, stamps _version.py/_commit.py, then runs poetry publish --build in a single step. (`poetry publish --build`)
**Environment-variable client configuration** — Examples must read endpoint from OPENMETER_ENDPOINT and token from OPENMETER_TOKEN; no hardcoded values. (`endpoint=os.environ['OPENMETER_ENDPOINT'], token=os.environ['OPENMETER_TOKEN']`)
**Sync/async example symmetry** — examples/sync/ and examples/async/ are structurally identical; sync uses openmeter.Client, async uses openmeter.aio.Client exclusively. (`# async/ingest.py
from openmeter.aio import Client`)
**__all__ extended from _patch_all** — __all__ in __init__.py must be extended from _patch_all to stay forward-compatible when new symbols are patched in. (`__all__ = [*_patch_all, 'Client', 'models']`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/_client.py` | Defines the public Client class subclassing the generated OpenMeterClient; only extension points, no new operations. | Adding new operation groups directly here instead of in _generated/. |
| `openmeter/__init__.py` | Public package surface; re-exports Client, models, and operations. Must call _patch_sdk() at end. | Missing _patch_sdk() call; extending __all__ without including _patch_all. |
| `openmeter/_types.py` | Union type aliases using string forward references; keeps heavy type imports out of __init__.py. | Using runtime (non-TYPE_CHECKING) imports of heavy generated types. |
| `openmeter/_version.py` | Written transiently by scripts/release.sh; not committed. | Committing this file — it is regenerated on every publish run. |
| `openmeter/_commit.py` | Written transiently by scripts/release.sh with the git SHA; not committed. | Committing this file. |
| `scripts/release.sh` | Sole publish script; queries PyPI for next alpha version, stamps version files, then calls poetry publish --build. | PY_SDK_RELEASE_TAG values other than 'alpha' hard-error; PY_SDK_RELEASE_VERSION with leading 'v' may produce invalid PEP-440 strings. |
| `pyproject.toml` | Poetry project definition; Python >=3.9, corehttp[requests,aiohttp] >=1.0.0b7, cloudevents ^1.12.1. | Changing version manually here — release.sh overwrites it at publish time. |
| `openmeter/py.typed` | PEP-561 marker file enabling type checking for downstream consumers; must be present in the installed package. | Removing from MANIFEST.in or package data — breaks Pyright/mypy for SDK users. |

## Anti-Patterns

- Defining new operation classes or methods in __init__.py or _client.py — all operations must come from _generated/
- Manually editing files under openmeter/_generated/ — overwritten by make gen-api
- Importing openmeter.aio in sync/ examples or openmeter (sync) in async/ examples
- Committing openmeter/_version.py or openmeter/_commit.py — written transiently by release.sh
- Running pyright type checks with mypy — mypy is incompatible with the generated SDK's overloaded constructors

## Decisions

- **Client subclasses generated OpenMeterClient rather than wrapping it** — Subclassing preserves all generated type signatures and avoids proxy boilerplate; wrapping would require re-exporting every method.
- **Version computed from live PyPI state rather than git tags in release.sh** — Allows independent alpha pre-release numbering without coupling to the monorepo tag cadence.
- **Pyright only (no mypy) for type checking** — Generated SDK uses overloaded constructors incompatible with mypy's strict mode; Pyright handles the generated patterns correctly.

## Example: Ingesting an event using the async client

```
import asyncio, datetime, uuid
from openmeter.aio import Client
from openmeter.models import Event

async def main():
    async with Client(
        endpoint=os.environ['OPENMETER_ENDPOINT'],
        token=os.environ['OPENMETER_TOKEN'],
    ) as client:
        event = Event(
            id=str(uuid.uuid4()),
            source='my-app',
            specversion='1.0',
            type='prompt',
            subject='customer-1',
// ...
```

<!-- archie:ai-end -->
