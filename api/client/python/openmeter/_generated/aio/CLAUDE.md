# aio

<!-- archie:ai-start -->

> Async (asyncio) entry point of the generated OpenMeter Python SDK — exposes OpenMeterClient built on AsyncPipelineClient with one attribute per resource group. All async I/O lives here; request construction is delegated entirely to the sync operations layer in the parent package. Generated code — customize only via _patch.py.

## Patterns

**AsyncPipelineClient with fixed policy chain + positional injection** — OpenMeterClient.__init__ builds AsyncPipelineClient from an ordered policy list (headers → user_agent → proxy → ContentDecodePolicy → retry → authentication → logging) and injects (self._client, self._config, self._serialize, self._deserialize) as positional args into every Operations class. (`self.customers = CustomersOperations(self._client, self._config, self._serialize, self._deserialize)`)
**Async context manager protocol** — OpenMeterClient implements __aenter__/__aexit__ and close() for `async with OpenMeterClient(...) as client`; __aenter__ delegates to self._client.__aenter__() and returns self. (`async def __aenter__(self) -> Self: await self._client.__aenter__(); return self`)
**send_request formats {endpoint} via deepcopy** — send_request deep-copies the HttpRequest and formats the {endpoint} placeholder through self._client.format_url() before delegating — never pass a raw URL. (`request_copy.url = self._client.format_url(request_copy.url, **path_format_arguments)`)
**AsyncRetryPolicy in configuration** — _configuration.py uses policies.AsyncRetryPolicy (not the sync RetryPolicy); authentication_policy defaults to None and must be supplied by callers via kwargs. (`self.retry_policy = kwargs.get('retry_policy') or policies.AsyncRetryPolicy(**kwargs)`)
**_patch.py customization hook** — All hand-written customizations belong in _patch.py; __init__.py imports _patch_all and calls patch_sdk() after importing the generated client. Never edit _client.py directly. (`from ._patch import __all__ as _patch_all; from ._patch import *; _patch_sdk()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_client.py` | Defines OpenMeterClient: builds AsyncPipelineClient, owns the policy chain, instantiates every Operations attribute. | Generated — do not hand-edit. Duplicate attribute assignments (e.g. self.subjects twice) are a codegen artifact; new resource groups require SDK regeneration. |
| `_configuration.py` | OpenMeterClientConfiguration: stores endpoint, sdk_moniker, polling_interval, and all policy instances consumed by _client.py. | authentication_policy defaults to None (no auth) — callers must pass it via kwargs; use AsyncRetryPolicy here, not sync RetryPolicy. |
| `_patch.py` | Empty hook for safe hand-written customizations; patch_sdk() runs at every import. | Any logic added must be idempotent — it runs at every import; keep __all__ updated when adding public symbols. |
| `__init__.py` | Package entry point: re-exports OpenMeterClient and merges _patch_all into __all__. | The TYPE_CHECKING guard around `from ._patch import *` is intentional for linters — do not remove it. |
| `operations/_operations.py` | Async Operations classes; async def methods delegate request building to the sync build_* functions in the parent operations layer. | Generated and overwritten on regeneration; per-method error_map handles typed deserialization — do not hand-edit. |

## Anti-Patterns

- Hand-editing _client.py or operations/_operations.py — generated and overwritten on the next SDK regeneration.
- Instantiating Operations classes directly rather than via OpenMeterClient attributes — bypasses shared serializer/deserializer injection.
- Passing a bare URL to send_request without formatting the {endpoint} placeholder via self._client.format_url().
- Raising on a streaming error response before `await response.read()` — leaks the socket/connection.
- Adding business or retry logic inside _patch.py instead of the Operations layer.

## Decisions

- **Async operations only handle execution and deserialization; all request-building lives in the sync operations/_operations.py build_* functions.** — Avoids duplicating URL/param construction and keeps sync and async layers consistent from a single generated source.
- **OpenMeterClientConfiguration is a separate class injected into every Operations constructor rather than raw kwargs.** — Centralises policy instantiation (retry, auth, logging) and makes the whole policy chain replaceable at construction without subclassing the client.

## Example: Async context manager usage with custom auth and a raw send_request call

```
from corehttp.runtime.pipeline import BearerTokenCredentialPolicy
from openmeter._generated.aio import OpenMeterClient
from corehttp.rest import HttpRequest

async def main():
    async with OpenMeterClient(
        endpoint="https://openmeter.cloud",
        authentication_policy=BearerTokenCredentialPolicy(credential, "api"),
    ) as client:
        meters = await client.meters.list()
        req = HttpRequest("GET", "{endpoint}/api/v1/meters")
        resp = await client.send_request(req)
```

<!-- archie:ai-end -->
