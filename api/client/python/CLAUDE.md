# python

<!-- archie:ai-start -->

> Public Python SDK root for OpenMeter — a thin shim layer over _generated/ sub-packages that re-exports the generated client, models, and operations under a stable versioned API surface. No business logic or operation definitions live here; everything routes through _generated/.

## Patterns

**Client subclasses generated OpenMeterClient** — openmeter/_client.py defines Client by subclassing the generated OpenMeterClient — never wrapping it. This preserves all generated type signatures without proxy boilerplate. (`from openmeter._generated import OpenMeterClient
class Client(OpenMeterClient): ...`)
**_patch_sdk() called at module end** — _patch_sdk() must be the last call in openmeter/__init__.py. Removing or relocating it silently disables all runtime patches from _patch.py. (`_patch_sdk()`)
**Graceful _patch ImportError guard** — All imports from _patch must be wrapped in try/except ImportError so the SDK works when _patch.py is absent. (`try:
    from ._patch import *  # type: ignore
except ImportError:
    pass`)
**__all__ extended from _patch_all** — __all__ in __init__.py must extend from _patch_all to remain forward-compatible when new symbols are patched in. (`__all__ = [*_patch_all, 'Client', 'models']`)
**Union type aliases in _types.py with string forward references** — _types.py isolates heavy Union type aliases using string forward references and TYPE_CHECKING-only imports, keeping __init__.py lightweight. (`from typing import TYPE_CHECKING
if TYPE_CHECKING:
    from ._generated.models import SomeModel`)
**Environment-variable client configuration in examples** — All examples read endpoint from OPENMETER_ENDPOINT and token from OPENMETER_TOKEN; no hardcoded values allowed. (`client = Client(
    endpoint=os.environ['OPENMETER_ENDPOINT'],
    token=os.environ['OPENMETER_TOKEN'],
)`)
**Sync/async example symmetry** — examples/sync/ and examples/async/ are structurally identical folders; sync uses openmeter.Client, async uses openmeter.aio.Client exclusively — never mixed. (`# async/ingest.py
from openmeter.aio import Client`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/_client.py` | Public Client class subclassing generated OpenMeterClient; only extension points, no new operations. | Adding new operation groups directly here instead of in _generated/. |
| `openmeter/__init__.py` | Public package surface; re-exports Client, models, operations. Must call _patch_sdk() at the very end. | Missing _patch_sdk() call; extending __all__ without including _patch_all. |
| `openmeter/_types.py` | Union type aliases with string forward references; keeps heavy type imports out of __init__.py. | Using runtime (non-TYPE_CHECKING) imports of heavy generated types. |
| `openmeter/_version.py` | Transient file written by scripts/release.sh; never committed. | Committing this file — it is regenerated on every publish run. |
| `openmeter/_commit.py` | Transient file written by scripts/release.sh with the git SHA; never committed. | Committing this file. |
| `scripts/release.sh` | Sole publish script: queries PyPI for next alpha version, stamps _version.py/_commit.py, runs poetry publish --build. | PY_SDK_RELEASE_TAG values other than 'alpha' hard-error; PY_SDK_RELEASE_VERSION with leading 'v' may produce invalid PEP-440 strings. |
| `pyproject.toml` | Poetry project definition; Python >=3.9, corehttp[requests,aiohttp] >=1.0.0b7, cloudevents ^1.12.1. | Manually bumping version here — release.sh overwrites it at publish time. |
| `openmeter/py.typed` | PEP-561 marker enabling type checking for downstream consumers; must be present in the installed package. | Removing from MANIFEST.in or package data — breaks Pyright for SDK users. |

## Anti-Patterns

- Defining new operation classes or methods in __init__.py or _client.py — all operations must come from _generated/
- Manually editing files under openmeter/_generated/ — they are overwritten by make gen-api
- Importing openmeter.aio in sync/ examples or openmeter (sync) in async/ examples
- Committing openmeter/_version.py or openmeter/_commit.py — written transiently by release.sh
- Running type checks with mypy — mypy is incompatible with the generated SDK's overloaded constructors; use pyright only

## Decisions

- **Client subclasses generated OpenMeterClient rather than wrapping it** — Subclassing preserves all generated type signatures and avoids proxy boilerplate; wrapping would require re-exporting every method.
- **Version computed from live PyPI state in release.sh rather than from git tags** — Allows independent alpha pre-release numbering without coupling to the monorepo tag cadence.
- **Pyright only (no mypy) for type checking** — Generated SDK uses overloaded constructors incompatible with mypy's strict mode; Pyright handles the generated patterns correctly.

## Example: Ingesting an event using the async client (canonical example pattern)

```
import asyncio, datetime, os, uuid
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
