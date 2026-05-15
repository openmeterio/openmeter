# examples

<!-- archie:ai-start -->

> Runnable Python example scripts (sync and async) demonstrating openmeter SDK usage patterns against a live endpoint; each script is self-contained, exercises one domain area, and serves as both documentation and a Pyright-checked regression surface for the generated SDK.

## Patterns

**Environment-variable configuration** — Endpoint and token are always read from OPENMETER_ENDPOINT and OPENMETER_TOKEN env vars with os.environ.get() and sensible defaults; never hardcoded. (`endpoint = os.environ.get('OPENMETER_ENDPOINT', 'https://openmeter.cloud')`)
**HttpResponseError catch-all** — All SDK calls are wrapped in try/except corehttp.exceptions.HttpResponseError to surface API errors consistently. (`except HttpResponseError as e: print(e)`)
**Dict-access for discriminated-union items** — SDK responses for polymorphic list items (e.g. entitlements) are plain dicts; use .get() not attribute access. (`item.get('type') == 'metered'`)
**One-file-per-domain organisation** — Each domain (ingest, query, entitlement, subscription, customer) lives in its own file under sync/ or async/; no cross-domain imports between example files. (`sync/ingest.py, async/entitlement.py — each self-contained`)
**Async examples use openmeter.aio.Client exclusively** — Files under async/ import from openmeter.aio and use async with + asyncio.run(); sync/ files import from openmeter and call main() directly at module level. (`from openmeter.aio import Client  # only in async/`)
**Module-level client construction for sync examples** — Sync scripts instantiate the blocking openmeter.Client once at module level, not inside main() on every call. (`client = Client(endpoint=..., credential=...)  # top of sync/ingest.py`)
**Async client lifetime via context manager** — Async examples open/close the HTTP session with async with client: inside the coroutine; never manage the session manually. (`async with client:  # inside async def main()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pyproject.toml` | Poetry project referencing the parent SDK via path dependency (develop=true); pins python ^3.9 to match SDK floor. | Do not add a version constraint on the openmeter path dependency — it must track the local checkout. Do not upgrade python floor beyond ^3.9 without checking SDK compatibility. |
| `pyrightconfig.json` | Pyright type-checking config covering both async/ and sync/ directories at basic mode. | Mypy is explicitly unsupported due to overloaded constructors in the generated SDK; never add mypy config here. |
| `async/ingest.py` | Reference implementation for async CloudEvents ingest; shows async with client, event construction, and error handling. | asyncio.run() must wrap the top-level coroutine; calling it inside an already-running loop raises RuntimeError. |
| `sync/ingest.py` | Structural mirror of async/ingest.py using blocking Client; module-level client construction followed by direct main() call. | Must not wrap code in async with or import from openmeter.aio. |

## Anti-Patterns

- Importing from openmeter.aio in sync/ examples or from openmeter (sync) in async/ examples
- Hardcoding endpoint or token instead of reading OPENMETER_ENDPOINT / OPENMETER_TOKEN from environment variables
- Using attribute access on dict-typed discriminated-union SDK responses instead of .get()
- Adding cross-domain logic or shared helpers across example files — each must remain self-contained
- Running type checks with mypy — mypy is incompatible with the generated SDK's overloaded constructors; use pyright only

## Decisions

- **Async and sync examples are structurally identical folders with separate client imports** — Keeps both execution models discoverable and ensures every domain is covered in both without a shared abstraction that could obscure per-client differences.
- **Pyright only, no mypy** — The generated SDK uses overloaded constructors that mypy cannot handle; Pyright is the only supported static type checker for this project.
- **One file per domain, not a monolithic demo** — Each domain script can be run and understood independently, and new domain examples can be added without touching existing files.

## Example: Async ingest example entry point — canonical pattern for all async/ scripts

```
import asyncio, os
from azure.core.exceptions import HttpResponseError
from openmeter.aio import Client

client = Client(
    endpoint=os.environ.get('OPENMETER_ENDPOINT', 'https://openmeter.cloud'),
    credential=os.environ.get('OPENMETER_TOKEN', ''),
)

async def main():
    async with client:
        try:
            await client.ingest_events(...)
        except HttpResponseError as e:
            print(e)
// ...
```

<!-- archie:ai-end -->
