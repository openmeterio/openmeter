# operations

<!-- archie:ai-start -->

> Public re-export facade for all generated Python SDK operation classes (API call builders). Mirrors the exact same facade pattern as openmeter/models but for _generated/operations — no operation logic is hand-written here.

## Patterns

**Wildcard re-export from generated operations** — Re-export _operations via wildcard import from _generated.operations. Never implement API call logic in this package. (`from .._generated.operations._operations import *  # noqa: F401, F403`)
**Optional _patch overlay** — Try-import _generated.operations._patch and merge its __all__. Patch symbols extend the generated set non-destructively. (`try:
    from .._generated.operations._patch import __all__ as _patch_all
except ImportError:
    _patch_all = []`)
**TYPE_CHECKING guard for patch stubs** — Import _patch under `if TYPE_CHECKING:` for IDE support without runtime cost. (`if TYPE_CHECKING:
    from .._generated.operations._patch import *`)
**Call _patch_sdk() at module load** — Always call _patch_sdk() after all re-exports to apply operation-level monkey-patches. (`_patch_sdk()`)
**__all__ propagation** — Derive __all__ from the generated operations module and extend with patch-only symbols. (`__all__ = __all__
__all__.extend([p for p in _patch_all if p not in __all__])`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `__init__.py` | Sole file; thin facade merging generated operation classes + patch overrides into one importable namespace (openmeter.operations.*). | Do not implement request/response logic here — operations are generated. Any code added here is overwritten on the next `make gen-api` run. Do not import deeper than _operations/_patch from _generated internals. |

## Anti-Patterns

- Implementing operation or request logic directly in this package — belongs in _generated/operations
- Importing deeper than _operations/_patch from _generated internals
- Removing the _patch_sdk() call — patch hooks silently will not apply
- Manually curating __all__ instead of delegating to the generated module
- Adding hand-written operation classes that duplicate generated ones and drift on codegen

## Decisions

- **Stable public import path via facade over generated internals** — Generated operation classes live under _generated and may be reorganised by the generator; the facade decouples consumer imports from generator layout, providing a stable openmeter.operations.* surface.

## Example: Full __init__.py structure — the only valid pattern for this package

```
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from .._generated.operations._patch import *

from .._generated.operations._operations import *  # noqa: F401, F403

try:
    from .._generated.operations._patch import __all__ as _patch_all
    from .._generated.operations._patch import *  # noqa: F401, F403
except ImportError:
    _patch_all = []
from .._generated.operations._patch import patch_sdk as _patch_sdk

from .._generated.operations import __all__
// ...
```

<!-- archie:ai-end -->
