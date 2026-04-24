# aio

<!-- archie:ai-start -->

> Public async namespace for the OpenMeter Python SDK — exposes the async Client class and re-exports async operation symbols from the auto-generated Azure SDK layer (_generated/aio). Acts as a thin public shim: no business logic, only wiring the generated internals to a stable import surface.

## Patterns

**Thin shim over _generated** — This package contains only __init__.py and _client.py. All async operation types come from _generated/aio; this folder's job is to re-export them under the public openmeter.aio namespace. (`from .._generated.aio._client import OpenMeterClient  # in _client.py`)
**Client subclasses OpenMeterClient** — Client in _client.py extends the generated OpenMeterClient, injecting Bearer-token auth via ServiceKeyCredential + ServiceKeyCredentialPolicy when a token kwarg is supplied, and forwarding all other kwargs to super().__init__. (`class Client(OpenMeterClient):
    def __init__(self, endpoint='https://openmeter.cloud', token=None, **kwargs):
        if token and not kwargs.get('authentication_policy'):
            credential = ServiceKeyCredential(token)
            kwargs['authentication_policy'] = policies.ServiceKeyCredentialPolicy(credential, 'Authorization', prefix='Bearer')
        super().__init__(endpoint=endpoint, **kwargs)`)
**_patch_sdk() call at module end** — __init__.py must call _patch_sdk() as the last statement so runtime monkey-patches from _generated/aio/_patch.py are applied after all imports. (`_patch_sdk()  # last line of __init__.py`)
**Graceful _patch ImportError guard** — _patch symbols are imported inside a try/except ImportError block so the package remains importable even if _patch.py is absent (e.g. before code-gen runs). (`try:
    from .._generated.aio._patch import __all__ as _patch_all
    from .._generated.aio._patch import *
except ImportError:
    _patch_all = []`)
**__all__ extension pattern** — __all__ is seeded with public names defined here (Client), then extended with any names from _patch_all that are not already listed, preserving patch-introduced symbols. (`__all__.extend([p for p in _patch_all if p not in __all__])`)
**TYPE_CHECKING-only patch import** — The _patch wildcard import for static type checkers is guarded under TYPE_CHECKING so it never executes at runtime, avoiding double-import side effects. (`if TYPE_CHECKING:
    from .._generated.aio._patch import *`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_client.py` | Defines the public Client class — the sole hand-authored async client. Adds Bearer token auth injection on top of the generated OpenMeterClient base. | Do not add operation methods here; they come from the generated base. If authentication_policy is already set in kwargs, the token is intentionally ignored — do not remove that guard. |
| `__init__.py` | Assembles the public openmeter.aio namespace: imports Client, applies _patch overlay, extends __all__. Must not contain business logic. | _patch_sdk() must be the last line; reordering breaks runtime patches. Wildcard import order matters: base symbols first, patch symbols second so patches override. |

## Anti-Patterns

- Defining new operation classes or methods directly in __init__.py or _client.py — all operations come from _generated
- Removing or relocating the _patch_sdk() call — runtime patches will silently not apply
- Manually editing files under _generated/aio/ — they are auto-generated and will be overwritten on next code-gen run
- Importing specific names from _generated instead of wildcard — breaks forward compatibility when _generated adds new operations
- Adding hard runtime imports of _patch symbols outside the try/except guard — causes ImportError when _patch.py is missing

## Decisions

- **Client subclasses generated OpenMeterClient rather than wrapping it** — Inheritance lets all generated async operations be available on Client without explicit delegation; only auth injection and context-manager protocol need to be overridden.
- **Optional _patch.py overlay with graceful ImportError fallback** — Allows monkey-patching generated behavior without requiring _patch.py to exist at all times, decoupling hand-authored patches from the code-gen cadence.

## Example: Instantiate the async client with a bearer token

```
from openmeter.aio import Client

async with Client(endpoint='https://openmeter.cloud', token='my-api-token') as client:
    result = await client.meters.list()
```

<!-- archie:ai-end -->
