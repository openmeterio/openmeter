# operations

<!-- archie:ai-start -->

> Re-exports async operation classes from the auto-generated Azure SDK layer (_generated/aio/operations) into the public openmeter.aio.operations namespace, applying optional monkey-patch overrides via _patch.py. This is a thin public-API shim — no business logic lives here.

## Patterns

**Wildcard re-export from generated layer** — All operations are imported from ..._generated.aio.operations._operations via 'from ... import *'. Never define new operation classes directly in this __init__.py. (`from ..._generated.aio.operations._operations import *  # noqa: F401, F403`)
**Patch overlay applied after base import** — _patch.py overrides are imported after _operations and merged into __all__ so patch symbols take precedence. patch_sdk() must be called at module end to activate runtime patches. (`from ..._generated.aio.operations._patch import *; _patch_sdk()`)
**__all__ extension pattern** — __all__ is set to the generated layer's __all__, then extended with patch symbols not already present. Order matters: generated names first, patch additions appended. (`__all__ = __all__; __all__.extend([p for p in _patch_all if p not in __all__])`)
**TYPE_CHECKING-only patch import guard** — The _patch wildcard import is guarded by TYPE_CHECKING for static analysis only, then repeated unconditionally for runtime. This double-import pattern is intentional Azure SDK convention. (`if TYPE_CHECKING:
    from ..._generated.aio.operations._patch import *`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `__init__.py` | Sole file in this package; acts as the public re-export shim. Importing from openmeter.aio.operations resolves here. | Do not add new symbols or logic here; any new async operation must be added in _generated/aio/operations/_operations.py (generated) or _patch.py (override). Changing import order breaks patch precedence. |

## Anti-Patterns

- Defining new operation classes or methods directly in __init__.py — all operations come from _generated
- Removing the _patch_sdk() call at module end — runtime patches will silently not apply
- Manually editing _generated/aio/operations/_operations.py — it is auto-generated and will be overwritten
- Importing specific names instead of wildcard — breaks compatibility when _generated adds new operations
- Reordering base import and patch import — patch symbols must override base, so patch import must come last

## Decisions

- **Wildcard re-export shim instead of direct _generated imports** — Gives consumers a stable openmeter.aio.operations namespace that survives generator churn; the generated path (_generated) is an internal implementation detail.
- **Optional _patch.py layer with graceful ImportError handling** — Allows SDK customisations (auth, retry, helpers) to be injected without forking the generator; if no patch file exists the SDK still works normally.

<!-- archie:ai-end -->
