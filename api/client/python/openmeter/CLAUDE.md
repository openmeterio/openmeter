# openmeter

<!-- archie:ai-start -->

> Public import surface for the OpenMeter Python SDK — a thin shim layer that re-exports generated client, models, and operations from _generated/ while providing a stable, version-pinned API. The primary constraint is that no business logic lives here; only wiring generated internals to public-facing imports.

## Patterns

**Thin shim over _generated** — All operation and model implementations live in _generated/. Top-level packages (aio/, models/, operations/) only re-export via wildcard imports. No new methods or classes are defined here. (`from ._generated._client import OpenMeterClient; class Client(OpenMeterClient): ...`)
**Client subclasses generated OpenMeterClient** — _client.py defines Client(OpenMeterClient) with token injection via ServiceKeyCredential and ServiceKeyCredentialPolicy. All operation routing is inherited, not re-implemented. (`class Client(OpenMeterClient):
    def __init__(self, endpoint, token=None, **kwargs):
        if token: kwargs['authentication_policy'] = policies.ServiceKeyCredentialPolicy(...)
        super().__init__(endpoint=endpoint, **kwargs)`)
**_patch_sdk() call at module end** — Every __init__.py must call _patch_sdk() after all imports so runtime patches from _generated/_patch.py are applied. Omitting silently breaks customizations. (`from ._generated._patch import patch_sdk as _patch_sdk
_patch_sdk()`)
**Graceful _patch ImportError guard** — All imports of _patch symbols are wrapped in try/except ImportError, setting _patch_all = [] as fallback. This allows the package to work when no _patch.py exists. (`try:
    from ._generated._patch import __all__ as _patch_all
    from ._generated._patch import *
except ImportError:
    _patch_all = []`)
**__all__ extension pattern** — __all__ is declared with explicit public names, then extended with _patch_all entries not already present, preserving forward compatibility. (`__all__ = ['Client']
__all__.extend([p for p in _patch_all if p not in __all__])`)
**TYPE_CHECKING-only patch import** — _patch symbols are imported under TYPE_CHECKING guard for static analysis, with the actual runtime import inside the try/except block. (`if TYPE_CHECKING:
    from ._generated._patch import *`)
**Union type aliases in _types.py** — _types.py defines discriminated Union aliases (e.g. App, RateCard, Entitlement) using string-quoted forward references to _models to avoid circular imports at runtime. (`App = Union['_models.StripeApp', '_models.SandboxApp', '_models.CustomInvoicingApp']`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `_client.py` | Defines the public Client class — subclasses generated OpenMeterClient and injects Bearer token authentication policy. Only customization allowed: constructor defaults and auth wiring. | Never add operation methods here — they must come from OpenMeterClient. Ensure __enter__/__exit__ delegates to super(). |
| `__init__.py` | Package entry point: exports Client, VERSION, COMMIT, and any _patch symbols. Must call _patch_sdk() last. | Missing _patch_sdk() call silently disables all runtime patches. __all__ must extend from _patch_all. |
| `_types.py` | Defines Union type aliases for polymorphic API types (App, RateCard, Entitlement, etc.) using forward-reference strings to avoid circular imports. | Duplicate App alias defined twice (lines 161 and 190) — likely a generation artifact. All type aliases use TYPE_CHECKING-safe string references. |
| `_version.py` | Single VERSION constant consumed by __init__.py. Overwritten by make gen-api. | Never hand-edit — always regenerated. |
| `_commit.py` | Single COMMIT constant stamped at generation time. | Do not hand-edit; regenerated artifact. |
| `py.typed` | PEP 561 marker — signals to type checkers that this package ships type stubs. | Must not be deleted; required for mypy/pyright to respect inline types. |

## Anti-Patterns

- Defining new operation classes or methods directly in __init__.py or _client.py — all operations must come from _generated
- Removing or relocating the _patch_sdk() call — runtime patches will silently not apply
- Manually editing files under _generated/ — they are overwritten by make gen-api
- Adding hard runtime imports of _patch symbols outside the try/except ImportError guard — causes ImportError when _patch.py is missing
- Manually maintaining __all__ without extending from _patch_all — breaks forward compatibility when new symbols are patched in

## Decisions

- **Client subclasses generated OpenMeterClient rather than wrapping it** — Inheritance gives automatic access to all generated operation attributes without proxying. Constructor-level auth injection (ServiceKeyCredential) is the only customization needed.
- **Optional _patch.py overlay with graceful ImportError fallback** — _patch.py is the sole file preserved across regenerations as a customization boundary. The fallback ensures the package loads without it in minimal deployments.
- **Union type aliases isolated in _types.py with string forward references** — Discriminated unions for polymorphic API types (App, RateCard, etc.) need to reference _models without triggering circular imports at module load time; string quotes defer resolution to type checkers.

## Example: Defining the public Client with token auth injection

```
from typing import Any, Optional
from typing_extensions import Self
from corehttp.credentials import ServiceKeyCredential
from corehttp.runtime import policies
from ._generated._client import OpenMeterClient

class Client(OpenMeterClient):
    def __init__(self, endpoint: str = 'https://openmeter.cloud', token: Optional[str] = None, **kwargs: Any) -> None:
        if token and not kwargs.get('authentication_policy'):
            credential = ServiceKeyCredential(token)
            kwargs['authentication_policy'] = policies.ServiceKeyCredentialPolicy(credential, 'Authorization', prefix='Bearer')
        super().__init__(endpoint=endpoint, **kwargs)

    def __enter__(self) -> Self:
        return super().__enter__()
// ...
```

<!-- archie:ai-end -->
