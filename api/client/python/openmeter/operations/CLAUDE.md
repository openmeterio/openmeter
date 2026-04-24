# operations

<!-- archie:ai-start -->

> Public re-export facade for all generated Python SDK operation classes (API call builders). Mirrors the same pattern as openmeter/models but for _generated/operations.

## Patterns

**Wildcard re-export from generated operations** — Re-export `_operations` via `from .._generated.operations._operations import *`. Never define operation methods here. (`from .._generated.operations._operations import *  # noqa: F401, F403`)
**Optional _patch overlay** — Try-import `_generated.operations._patch` and merge its `__all__`. Patch symbols extend the generated set non-destructively. (`try:
    from .._generated.operations._patch import __all__ as _patch_all
except ImportError:
    _patch_all = []`)
**TYPE_CHECKING guard for patch stubs** — Import `_patch` under `if TYPE_CHECKING:` for IDE support only. (`if TYPE_CHECKING:
    from .._generated.operations._patch import *`)
**Call patch_sdk() at module load** — Always call `_patch_sdk()` after all re-exports. (`_patch_sdk()`)
**__all__ propagation** — Derive `__all__` from the generated module and extend with patch-only symbols. (`__all__ = __all__
__all__.extend([p for p in _patch_all if p not in __all__])`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `__init__.py` | Sole file; thin facade merging generated operation classes + patch overrides into one importable namespace. | Do not implement API call logic here; operations are generated. Adding code here breaks on next `make gen-api`. |

## Anti-Patterns

- Implementing operation/request logic directly in this package
- Importing deeper than _operations/_patch from _generated internals
- Removing the _patch_sdk() call
- Manually curating __all__ instead of delegating to the generated module

## Decisions

- **Stable public import path via facade over generated internals** — Generated operation classes live under _generated and may be reorganised by the generator; the facade decouples consumer imports from generator layout.

<!-- archie:ai-end -->
