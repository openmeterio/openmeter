# aio

<!-- archie:ai-start -->

> Public async namespace shim for the OpenMeter Python SDK: exposes the hand-authored async Client (subclass of the generated OpenMeterClient with Bearer-token auth injection) and re-exports async operation symbols from the auto-generated Azure SDK layer (_generated/aio). No business logic — only wires generated internals to a stable openmeter.aio import surface; the operations/ child mirrors this for the operations namespace.

## Patterns

**Thin shim over _generated** — All async operation types come from _generated/aio and are re-exported under openmeter.aio; never define operation classes here. (`from .._generated.aio._client import OpenMeterClient  # in _client.py`)
**Client subclasses generated OpenMeterClient** — Client in _client.py extends OpenMeterClient and injects ServiceKeyCredential + ServiceKeyCredentialPolicy (Bearer prefix) when a token kwarg is supplied, forwarding all other kwargs to super().__init__. (`if token and not kwargs.get('authentication_policy'): kwargs['authentication_policy'] = policies.ServiceKeyCredentialPolicy(ServiceKeyCredential(token), 'Authorization', prefix='Bearer')`)
**_patch_sdk() as the last line of __init__.py** — __init__.py must call _patch_sdk() last so runtime monkey-patches from _generated/aio/_patch.py apply after all imports. (`_patch_sdk()  # last line of __init__.py`)
**Graceful _patch ImportError guard** — _patch symbols are imported inside try/except ImportError so the package stays importable even if _patch.py is absent (e.g. before code-gen runs). (`try: from .._generated.aio._patch import __all__ as _patch_all; from .._generated.aio._patch import * except ImportError: _patch_all = []`)
**__all__ extension pattern** — __all__ is seeded with names defined here (Client) and extended with any _patch_all names not already present, preserving patch-introduced symbols. (`__all__.extend([p for p in _patch_all if p not in __all__])`)
**TYPE_CHECKING-only patch import** — The _patch wildcard import for static type checkers is guarded under TYPE_CHECKING so it never runs at runtime. (`if TYPE_CHECKING: from .._generated.aio._patch import *`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_client.py` | Defines the public async Client class — the sole hand-authored async client; adds Bearer auth injection over the generated OpenMeterClient base and the context-manager protocol. | Do not add operation methods here. If authentication_policy is already in kwargs the token is intentionally ignored — keep that guard. |
| `__init__.py` | Assembles the public openmeter.aio namespace: imports Client, applies the _patch overlay, extends __all__, calls _patch_sdk(). | _patch_sdk() must be the last line; base import must precede patch import so patches override base symbols. |

## Anti-Patterns

- Defining new operation classes or methods in __init__.py or _client.py — all operations come from _generated.
- Removing or relocating the _patch_sdk() call — runtime patches will silently not apply.
- Manually editing files under _generated/aio/ — auto-generated and overwritten on next code-gen.
- Importing specific names from _generated instead of wildcard — breaks forward compatibility when new operations are added.
- Adding hard runtime imports of _patch symbols outside the try/except guard — causes ImportError when _patch.py is missing.

## Decisions

- **Client subclasses the generated OpenMeterClient rather than wrapping it.** — Inheritance exposes all generated async operations without explicit delegation; only auth injection and the context-manager protocol need overriding.
- **Optional _patch.py overlay with graceful ImportError fallback.** — Allows monkey-patching generated behavior without requiring _patch.py to always exist, decoupling hand-authored patches from the code-gen cadence.

## Example: Instantiate the async client with a bearer token

```
from openmeter.aio import Client

async with Client(endpoint='https://openmeter.cloud', token=my_api_token) as client:
    result = await client.meters.list()
```

<!-- archie:ai-end -->
