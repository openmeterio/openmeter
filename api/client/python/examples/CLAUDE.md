# examples

<!-- archie:ai-start -->

> Runnable Python example scripts (sync/ and async/) demonstrating openmeter SDK usage against a live endpoint; each script is self-contained, exercises one domain (ingest, query, entitlement, subscription, customer), and doubles as documentation and a Pyright-checked regression surface for the generated SDK. The folder root holds only Poetry/Pyright config and a README.

## Patterns

**Environment-variable configuration with defaults** — Endpoint and token are always read from OPENMETER_ENDPOINT / OPENMETER_TOKEN via os.environ.get() with sensible defaults; never hardcoded. (`endpoint = os.environ.get('OPENMETER_ENDPOINT', 'https://openmeter.cloud')`)
**HttpResponseError catch-all** — Every SDK call is wrapped in try/except corehttp.exceptions.HttpResponseError to surface API errors consistently. (`except HttpResponseError as e: print(e)`)
**Dict-access for discriminated-union list items** — Polymorphic list responses (e.g. entitlements) are plain dicts — use .get(), never attribute access. (`item.get('type') == 'metered'`)
**One file per domain, self-contained** — Each domain lives in its own file under sync/ or async/; no cross-domain imports or shared helpers between example files. (`sync/ingest.py, async/entitlement.py — each self-contained`)
**Client import split: async.aio vs sync top-level** — Files under async/ import from openmeter.aio and use async with + asyncio.run(); sync/ files import from openmeter and call main() directly at module level. (`from openmeter.aio import Client  # only in async/`)
**Client lifetime: module-level (sync) vs context manager (async)** — Sync scripts construct the blocking Client once at module level; async scripts open/close the HTTP session via async with client: inside the coroutine, never manually. (`client = Client(endpoint=..., credential=...)  # top of sync/ingest.py ; async with client:  # inside async def main()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `pyproject.toml` | Poetry project referencing the parent SDK via a path dependency (develop=true); pins python ^3.9 to match the SDK floor. | Do not add a version constraint on the openmeter path dependency — it must track the local checkout; do not raise the python floor beyond ^3.9 without checking SDK compatibility. |
| `pyrightconfig.json` | Pyright config covering both async/ and sync/ at basic mode. | Mypy is unsupported (overloaded SDK constructors) — never add mypy config. |
| `README.md` | Setup + run instructions (poetry install, env-var invocation, poetry run pyright). | States Pyright-only type checking — keep examples Pyright-clean. |
| `async/` | Child: async example scripts using openmeter.aio.Client with async with + asyncio.run(). | asyncio.run() must wrap the top-level coroutine; calling inside a running loop raises RuntimeError. |
| `sync/` | Child: structural mirror of async/ using the blocking openmeter.Client at module level. | Must not use async with or import from openmeter.aio. |

## Anti-Patterns

- Importing from openmeter.aio in sync/ examples or from openmeter (sync) in async/ examples
- Hardcoding endpoint or token instead of reading OPENMETER_ENDPOINT / OPENMETER_TOKEN
- Using attribute access on dict-typed discriminated-union responses instead of .get()
- Adding cross-domain logic or shared helpers across example files — each must stay self-contained
- Running type checks with mypy — incompatible with the SDK's overloaded constructors; use pyright only

## Decisions

- **async/ and sync/ are structurally identical folders with separate client imports** — Keeps both execution models discoverable and ensures every domain is covered in both without a shared abstraction obscuring per-client differences.
- **Pyright only, no mypy** — The generated SDK uses overloaded constructors mypy cannot handle; Pyright is the only supported static type checker.
- **One file per domain, not a monolithic demo** — Each domain script runs and reads independently, and new domain examples are added without touching existing files.

## Example: Async ingest entry point — canonical async/ pattern

```
import asyncio, os
from corehttp.exceptions import HttpResponseError
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
