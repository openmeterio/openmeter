# operations

<!-- archie:ai-start -->

> Async (asyncio) mirror of the sync operations layer — each class wraps a resource group (AppsOperations, CustomersOperations, InvoicesOperations, etc.) and exposes `async def` methods that execute HTTP requests via AsyncPipelineClient. All request-building logic is delegated to the sync `operations/_operations.py` build_* functions; only the execution and response-handling is async here.

## Patterns

**Reuse sync request builders** — Every method imports and calls the corresponding `build_<resource>_<action>_request` from `...operations._operations` — never duplicates URL/param construction. (`_request = build_apps_list_request(page=page, page_size=page_size, headers=_headers, params=_params)`)
**Positional constructor injection** — Each Operations class __init__ pops (client, config, serializer, deserializer) from positional args first, kwargs second — never accept them as named-only parameters. (`self._client: AsyncPipelineClient = input_args.pop(0) if input_args else kwargs.pop('client')`)
**Per-method error_map with status-specific model deserialization** — Every method builds a MutableMapping error_map seeded with 404/409/304 defaults, then manually deserializes typed Problem models for 400/401/403/500/503/412 before raising HttpResponseError. (`error = _failsafe_deserialize(_models.BadRequestProblemResponse, response); raise HttpResponseError(response=response, model=error)`)
**Stream-safe response body cleanup** — On non-2xx responses with stream=True, call `await response.read()` inside a try/except for StreamConsumedError and StreamClosedError before calling map_error. (`if _stream: try: await response.read() except (StreamConsumedError, StreamClosedError): pass`)
**_patch.py customization hook** — `_patch.py` is the only sanctioned place to add customizations; `__init__.py` imports `from ._patch import *` and calls `_patch_sdk()` at module load. Keep `_patch.py` minimal — it is the escape hatch, not the norm. (`from ._patch import __all__ as _patch_all; _patch_sdk()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_operations.py` | Contains every async Operations class and their async methods. This file is generated — do not edit directly; changes belong in _patch.py or upstream generators. | DO NOT instantiate Operations classes directly; they are accessed through OpenMeterClient attributes. Adding new methods here without a corresponding sync build_* function in the parent operations package will break the reuse pattern. |
| `__init__.py` | Re-exports all Operations classes and applies _patch.py overrides. Mirrors the sync operations __init__.py exactly. | Any new Operations class added to _operations.py must be added to both __all__ and the import list here, and in the sync __init__.py. |
| `_patch.py` | Empty stub for SDK customizations; imported and executed by __init__.py via patch_sdk(). | Never add business logic here. This file is preserved across regenerations as the intentional customization seam. |

## Anti-Patterns

- Duplicating URL/query-param construction inside async methods instead of calling the shared sync build_* functions
- Instantiating Operations classes directly rather than accessing them via OpenMeterClient attributes
- Adding error handling or deserialization logic outside the per-method error_map block
- Editing _operations.py as if it were hand-written code — it is generated and will be overwritten
- Raising exceptions before calling `await response.read()` on a streaming error response (leaks the socket)

## Decisions

- **Async operations delegate all request construction to sync build_* functions in the parent package** — Avoids duplicating URL and parameter logic across sync/async variants; a single change to the request builder propagates to both.
- **Per-method typed error deserialization instead of a shared error handler** — Different endpoints return different Problem subtypes; inline deserialization keeps error context close to the operation contract without a shared dispatcher that must know all error shapes.

## Example: Adding a new async operation method following the existing pattern

```
from ...operations._operations import build_my_resource_list_request
from ..._utils.model_base import _deserialize, _failsafe_deserialize
from ... import models as _models

async def list(self, *, page: Optional[int] = None, **kwargs: Any) -> _models.MyResourcePaginatedResponse:
    error_map: MutableMapping = {404: ResourceNotFoundError, 409: ResourceExistsError, 304: ResourceNotModifiedError}
    error_map.update(kwargs.pop('error_map', {}) or {})
    _headers = kwargs.pop('headers', {}) or {}
    _params = kwargs.pop('params', {}) or {}
    cls: ClsType[_models.MyResourcePaginatedResponse] = kwargs.pop('cls', None)
    _request = build_my_resource_list_request(page=page, headers=_headers, params=_params)
    path_format_arguments = {'endpoint': self._serialize.url('self._config.endpoint', self._config.endpoint, 'str', skip_quote=True)}
    _request.url = self._client.format_url(_request.url, **path_format_arguments)
    _decompress = kwargs.pop('decompress', True)
    _stream = kwargs.pop('stream', False)
// ...
```

<!-- archie:ai-end -->
