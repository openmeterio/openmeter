# aio

<!-- archie:ai-start -->

> Async (asyncio) entry point for the OpenMeter Python SDK — exposes OpenMeterClient with AsyncPipelineClient and one attribute per resource group (portal, apps, customers, meters, etc.). All async I/O is handled here; request construction is delegated to the sync operations layer.

## Patterns

**AsyncPipelineClient injection** — OpenMeterClient.__init__ builds an AsyncPipelineClient from an ordered policy list and injects it, plus shared Serializer/Deserializer, into every Operations class via positional args: (self._client, self._config, self._serialize, self._deserialize). (`self.customers = CustomersOperations(self._client, self._config, self._serialize, self._deserialize)`)
**Policy chain construction** — Policies are assembled in __init__ in a fixed order: headers → user_agent → proxy → ContentDecodePolicy → retry → authentication → logging. Callers may pass a custom policies list via kwargs to override the entire chain. (`_policies = kwargs.pop('policies', None) or [self._config.headers_policy, ..., self._config.logging_policy]`)
**Async context manager protocol** — OpenMeterClient implements __aenter__ / __aexit__ and close() so it can be used as `async with OpenMeterClient(...) as client`. (`async def __aenter__(self) -> Self: await self._client.__aenter__(); return self`)
**_patch.py customization hook** — All hand-written customizations belong in _patch.py. __init__.py imports _patch_all and calls patch_sdk() after importing the generated client — never edit _client.py directly. (`from ._patch import __all__ as _patch_all; from ._patch import *; _patch_sdk()`)
**send_request URL formatting** — send_request deep-copies the HttpRequest and formats the {endpoint} path placeholder via self._serialize.url() before delegating to self._client.send_request — never pass a raw URL directly. (`request_copy.url = self._client.format_url(request_copy.url, endpoint=self._serialize.url(...))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_client.py` | Defines OpenMeterClient: builds AsyncPipelineClient, owns the policy chain, and instantiates every Operations attribute. | This file is generated — adding new resource groups requires regeneration, not hand-editing. Duplicate attribute assignments (e.g. self.subjects assigned twice) are a codegen artifact; do not 'fix' them manually. |
| `_configuration.py` | Holds OpenMeterClientConfiguration: stores endpoint, SDK moniker, polling_interval, and all policy instances used by _client.py. | authentication_policy defaults to None (no auth); callers must supply it via kwargs. RetryPolicy here is AsyncRetryPolicy — do not substitute the sync RetryPolicy. |
| `_patch.py` | Empty hook file for safe hand-written customizations; patch_sdk() is called at import time. | Any logic added here must be idempotent — it runs at every import. Keep __all__ updated if adding public symbols. |
| `__init__.py` | Package entry point: re-exports OpenMeterClient and merges _patch_all into __all__. | The TYPE_CHECKING guard around `from ._patch import *` is intentional for linters — do not remove it. |

## Anti-Patterns

- Hand-editing _client.py — it is generated and will be overwritten on the next SDK regeneration cycle
- Instantiating Operations classes directly rather than accessing them as client attributes (bypasses shared serializer/deserializer)
- Passing a bare URL string to send_request without formatting the {endpoint} placeholder first
- Raising or re-raising exceptions on streaming error responses before calling `await response.read()` (leaks the connection)
- Adding business logic or retry logic inside _patch.py instead of the Operations layer

## Decisions

- **All request-building logic lives in the sync operations/_operations.py build_* functions; async operations only handle execution and response deserialization.** — Avoids duplicating URL/param construction and keeps the sync and async layers in sync with a single generated source.
- **OpenMeterClientConfiguration is a separate class injected into every Operations constructor rather than passed as raw kwargs.** — Centralises policy instantiation (retry, auth, logging) and makes the policy chain replaceable at construction time without subclassing the client.

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
        # Use a named resource group
        meters = await client.meters.list()

        # Raw request — must format {endpoint} placeholder
        req = HttpRequest("GET", "{endpoint}/api/v1/meters")
        resp = await client.send_request(req)
// ...
```

<!-- archie:ai-end -->
