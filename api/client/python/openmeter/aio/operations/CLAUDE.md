# operations

<!-- archie:ai-start -->

> Public async operations namespace shim that re-exports all async operation classes from the auto-generated Azure SDK layer (_generated/aio/operations) into the stable openmeter.aio.operations namespace, applying optional monkey-patch overrides via _patch.py. No business logic lives here — this is a thin re-export boundary.

## Patterns

**Wildcard re-export from generated layer** — All operations are imported from ..._generated.aio.operations._operations via wildcard import. Never define new operation classes directly in __init__.py. (`from ..._generated.aio.operations._operations import *  # noqa: F401, F403`)
**Patch overlay applied after base import** — _patch.py overrides are imported after _operations so patch symbols take precedence. patch_sdk() must be called at module end to activate runtime patches. (`from ..._generated.aio.operations._patch import *; _patch_sdk()`)
**__all__ extension pattern** — __all__ is set to the generated layer's __all__, then extended with patch symbols not already present. Order matters: generated names first, patch additions appended. (`__all__ = __all__
__all__.extend([p for p in _patch_all if p not in __all__])`)
**TYPE_CHECKING-only patch import guard** — The _patch wildcard import is guarded by TYPE_CHECKING for static analysis, then repeated unconditionally for runtime. This double-import pattern is intentional Azure SDK convention. (`if TYPE_CHECKING:
    from ..._generated.aio.operations._patch import *`)
**Graceful ImportError fallback for patch** — The _patch import is wrapped in try/except ImportError so the SDK works normally when no patch file exists. (`try:
    from ..._generated.aio.operations._patch import __all__ as _patch_all
except ImportError:
    _patch_all = []`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `__init__.py` | Sole file in this package; acts as the public re-export shim. Importing from openmeter.aio.operations resolves here. | Do not add new symbols or logic here. Any new async operation must be added in _generated/aio/operations/_operations.py (generated) or _patch.py (override). Changing import order breaks patch precedence. Removing _patch_sdk() call silently disables all runtime patches. |

## Anti-Patterns

- Defining new operation classes or methods directly in __init__.py — all operations come from _generated
- Removing the _patch_sdk() call at module end — runtime patches will silently not apply
- Manually editing _generated/aio/operations/_operations.py — it is auto-generated and will be overwritten on next SDK generation
- Importing specific names instead of wildcard — breaks compatibility when _generated adds new operations
- Reordering base import and patch import — patch symbols must override base, so patch import must come after _operations import

## Decisions

- **Wildcard re-export shim instead of direct _generated imports** — Gives consumers a stable openmeter.aio.operations namespace that survives generator churn; the _generated path is an internal implementation detail consumers should not depend on.
- **Optional _patch.py layer with graceful ImportError handling** — Allows SDK customisations (auth, retry, helpers) to be injected without forking the generator; if no patch file exists the SDK still works normally.

## Example: Correct structure of __init__.py — re-export with patch overlay

```
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ..._generated.aio.operations._patch import *  # pylint: disable=unused-wildcard-import

from ..._generated.aio.operations._operations import *  # noqa: F401, F403

try:
    from ..._generated.aio.operations._patch import __all__ as _patch_all
    from ..._generated.aio.operations._patch import *  # noqa: F401, F403
except ImportError:
    _patch_all = []
from ..._generated.aio.operations._patch import patch_sdk as _patch_sdk

from ..._generated.aio.operations import __all__
// ...
```

<!-- archie:ai-end -->
