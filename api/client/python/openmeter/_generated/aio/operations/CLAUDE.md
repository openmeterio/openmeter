# operations

<!-- archie:ai-start -->

> Async (asyncio) mirror of the sync operations layer. Each Operations class wraps a resource group and exposes `async def` methods that execute HTTP via AsyncPipelineClient, delegating all request-building to the sync build_* functions in the parent operations/_operations.py.

## Patterns

**Delegate request construction to sync build_* functions** — Every async method imports build_<resource>_<action>_request from ...operations._operations and never duplicates URL/query-param construction inline. (`_request = build_apps_list_request(page=page, page_size=page_size, headers=_headers, params=_params)`)
**Positional-first constructor injection** — Each Operations __init__ pops (client, config, serializer, deserializer) from positional args first, then kwargs — never named-only parameters. (`self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop('client')`)
**Per-method error_map with typed Problem deserialization** — Each method seeds a MutableMapping error_map with 404/409/304 defaults, then deserializes typed Problem models (BadRequestProblemResponse, etc.) per status code before raising HttpResponseError. (`error = _failsafe_deserialize(_models.BadRequestProblemResponse, response); raise HttpResponseError(response=response, model=error)`)
**Stream-safe response cleanup on error** — On non-2xx when _stream=True, call `await response.read()` inside a try/except for StreamConsumedError/StreamClosedError before map_error to prevent socket leaks. (`if _stream:
    try: await response.read()
    except (StreamConsumedError, StreamClosedError): pass`)
**_patch.py customization hook** — _patch.py is the only sanctioned place to add customizations; __init__.py imports from ._patch and calls patch_sdk() at load. Keep it minimal. (`from ._patch import __all__ as _patch_all; _patch_sdk()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_operations.py` | Every async Operations class and its async methods. Generated — accessed through OpenMeterClient attributes, not instantiated directly. | Do NOT instantiate Operations classes directly. Adding methods here without a corresponding sync build_* function in the parent operations package breaks the reuse pattern. |
| `__init__.py` | Re-exports all Operations classes and applies _patch overrides via _patch_sdk(). Mirrors the sync operations __init__.py. | Any new Operations class in _operations.py must be added to both __all__ and the import list here, and mirrored in the sync __init__.py. |
| `_patch.py` | Empty stub for SDK customizations; imported/executed by __init__.py. Stable seam preserved across regenerations. | Never add business logic; its sole purpose is to survive code regeneration. |

## Anti-Patterns

- Duplicating URL/query-param construction inside async methods instead of calling shared sync build_* functions
- Instantiating Operations classes directly rather than via OpenMeterClient attributes
- Adding error handling or deserialization outside the per-method error_map block
- Editing _operations.py as if hand-written — it is generated and overwritten on next SDK generation
- Raising before `await response.read()` on a streaming error response (leaks the socket)

## Decisions

- **Async operations delegate all request construction to sync build_* functions** — Avoids duplicating URL/parameter logic across sync/async; one change to a request builder propagates to both execution paths.
- **Per-method typed error deserialization instead of a shared handler** — Different endpoints return different Problem subtypes; inline deserialization keeps error context close to the operation contract.

## Example: Adding a new async operation method following the existing pattern

```
from ...operations._operations import build_my_resource_list_request
from ..._utils.model_base import _deserialize, _failsafe_deserialize
from ... import models as _models

async def list(self, *, page: Optional[int] = None, **kwargs: Any) -> _models.MyResourcePaginatedResponse:
    error_map: MutableMapping = {404: ResourceNotFoundError, 409: ResourceExistsError, 304: ResourceNotModifiedError}
    error_map.update(kwargs.pop('error_map', {}) or {})
    _request = build_my_resource_list_request(page=page, headers=kwargs.pop('headers', {}) or {}, params=kwargs.pop('params', {}) or {})
    _request.url = self._client.format_url(_request.url, **path_format_arguments)
```

<!-- archie:ai-end -->
