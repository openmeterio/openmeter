# aio

<!-- archie:ai-start -->

> Async (asyncio) entry point for the OpenMeter Python SDK — exposes OpenMeterClient with AsyncPipelineClient and one attribute per resource group. All async I/O is handled here; request construction is entirely delegated to the sync operations layer under the parent package.

## Patterns

**AsyncPipelineClient injection with fixed policy chain** — OpenMeterClient.__init__ builds AsyncPipelineClient from an ordered policy list (headers → user_agent → proxy → ContentDecodePolicy → retry → authentication → logging) and injects (self._client, self._config, self._serialize, self._deserialize) as positional args into every Operations class. (`self.customers = CustomersOperations(self._client, self._config, self._serialize, self._deserialize)`)
**Async context manager protocol** — OpenMeterClient implements __aenter__ / __aexit__ and close() for use as `async with OpenMeterClient(...) as client`. __aenter__ delegates to self._client.__aenter__() and returns self. (`async def __aenter__(self) -> Self: await self._client.__aenter__(); return self`)
**send_request URL formatting via deepcopy** — send_request deep-copies the HttpRequest and formats the {endpoint} path placeholder via self._client.format_url() before delegating to self._client.send_request — never pass a raw URL directly. (`request_copy.url = self._client.format_url(request_copy.url, **path_format_arguments)`)
**_patch.py customization hook** — All hand-written customizations belong in _patch.py. __init__.py imports _patch_all and calls patch_sdk() after importing the generated client. Never edit _client.py directly — it is generated and overwritten on SDK regeneration. (`from ._patch import __all__ as _patch_all; from ._patch import *; _patch_sdk()`)
**AsyncRetryPolicy in configuration** — _configuration.py uses policies.AsyncRetryPolicy (not the sync RetryPolicy) for the retry_policy field. authentication_policy defaults to None — callers must supply it via kwargs. (`self.retry_policy = kwargs.get('retry_policy') or policies.AsyncRetryPolicy(**kwargs)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_client.py` | Defines OpenMeterClient: builds AsyncPipelineClient, owns the policy chain, and instantiates every Operations attribute as a named instance attribute. | Generated file — do not hand-edit. Duplicate attribute assignments (e.g. self.subjects assigned twice) are a codegen artifact; do not fix manually. Adding new resource groups requires SDK regeneration. |
| `_configuration.py` | Holds OpenMeterClientConfiguration: stores endpoint, sdk_moniker, polling_interval, and all policy instances consumed by _client.py. | authentication_policy defaults to None (no auth); callers must pass it via kwargs. Use AsyncRetryPolicy here, not the sync RetryPolicy. |
| `_patch.py` | Empty hook file for safe hand-written customizations; patch_sdk() is called at every import. | Any logic added here must be idempotent — it runs at every import. Keep __all__ updated if adding public symbols. |
| `__init__.py` | Package entry point: re-exports OpenMeterClient and merges _patch_all into __all__. | The TYPE_CHECKING guard around `from ._patch import *` is intentional for linters — do not remove it. |

## Anti-Patterns

- Hand-editing _client.py — it is generated and will be overwritten on the next SDK regeneration cycle
- Instantiating Operations classes directly rather than accessing them as client attributes (bypasses shared serializer/deserializer injection)
- Passing a bare URL string to send_request without formatting the {endpoint} placeholder via self._client.format_url()
- Raising or re-raising exceptions on streaming error responses before calling `await response.read()` — leaks the connection
- Adding business logic or retry logic inside _patch.py instead of in the Operations layer

## Decisions

- **All request-building logic lives in the sync operations/_operations.py build_* functions; async operations only handle execution and response deserialization.** — Avoids duplicating URL/param construction and keeps sync and async layers consistent from a single generated source.
- **OpenMeterClientConfiguration is a separate class injected into every Operations constructor rather than passed as raw kwargs.** — Centralises policy instantiation (retry, auth, logging) and makes the entire policy chain replaceable at construction time without subclassing the client.

## Example: Async context manager usage with custom auth and a raw send_request call

```
from corehttp.runtime.pipeline import BearerTokenCredentialPolicy
from openmeter._generated.aio import OpenMeterClient
from corehttp.rest import HttpRequest

async def main():
    async with OpenMeterClient(
        endpoint="https://openmeter.cloud",
        authentication_policy=BearerTokenCredentialPolicy(credential, "api")
    ) as client:
        meters = await client.meters.list()

        # Raw request — must use send_request to format {endpoint}
        req = HttpRequest("GET", "{endpoint}/api/v1/meters")
        resp = await client.send_request(req)
```

<!-- archie:ai-end -->
