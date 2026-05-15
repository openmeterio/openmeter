# models

<!-- archie:ai-start -->

> Public re-export facade for all generated Python SDK model types (dataclasses and enums). Delegates entirely to _generated/models — no hand-written model code lives here; stable import path for consumers regardless of generator internals.

## Patterns

**Wildcard re-export from generated layer** — Re-export _models and _enums via wildcard imports from _generated.models. Never define or duplicate types in this package. (`from .._generated.models._models import *  # noqa: F401, F403
from .._generated.models._enums import *  # noqa: F401, F403`)
**Optional _patch overlay** — Try-import _generated.models._patch and merge its __all__ into the module __all__. Patch symbols extend (never replace) the generated set. (`try:
    from .._generated.models._patch import __all__ as _patch_all
except ImportError:
    _patch_all = []`)
**TYPE_CHECKING guard for patch stubs** — Import _patch under `if TYPE_CHECKING:` for IDE type-narrowing without runtime cost. (`if TYPE_CHECKING:
    from .._generated.models._patch import *`)
**Call _patch_sdk() at module load** — Always call _patch_sdk() after all re-exports to apply monkey-patches from the generated layer. (`_patch_sdk()`)
**__all__ propagation** — Derive __all__ from the generated module, then extend with patch-only symbols to provide a complete, non-redundant export list. (`__all__ = __all__
__all__.extend([p for p in _patch_all if p not in __all__])`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `__init__.py` | Sole file in the package; thin facade that merges generated models + patch overrides into one importable namespace (openmeter.models.*). | Do not add hand-written model classes here — they will be clobbered on the next `make gen-api` run. Do not import deeper than _models/_enums/_patch from _generated internals. |

## Anti-Patterns

- Defining or modifying model classes directly in this package instead of using the _patch mechanism
- Importing from _generated internals deeper than _models/_enums/_patch
- Removing the _patch_sdk() call — patch hooks will silently not apply
- Manually maintaining __all__ instead of deriving it from the generated module
- Duplicating generated types here — they drift immediately on next codegen run

## Decisions

- **Thin facade over generated code rather than duplicating types** — All model types are generated from TypeSpec via `make gen-api`; any hand-written copy would drift immediately. The facade gives a stable import path (openmeter.models.*) regardless of generator internals.

## Example: Full __init__.py structure — the only valid pattern for this package

```
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from .._generated.models._patch import *

from .._generated.models._models import *  # noqa: F401, F403
from .._generated.models._enums import *  # noqa: F401, F403

try:
    from .._generated.models._patch import __all__ as _patch_all
    from .._generated.models._patch import *  # noqa: F401, F403
except ImportError:
    _patch_all = []
from .._generated.models._patch import patch_sdk as _patch_sdk

// ...
```

<!-- archie:ai-end -->
