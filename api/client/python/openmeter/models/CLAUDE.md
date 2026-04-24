# models

<!-- archie:ai-start -->

> Public re-export facade for all generated Python SDK model types (dataclasses and enums). Delegates entirely to _generated/models — no hand-written model code lives here.

## Patterns

**Wildcard re-export from generated layer** — Re-export `_models` and `_enums` via `from .._generated.models._models import *` and `from .._generated.models._enums import *`. Never define or duplicate types here. (`from .._generated.models._models import *  # noqa: F401, F403`)
**Optional _patch overlay** — Try-import `_generated.models._patch` and merge its `__all__` into the module's `__all__`. Patch symbols extend (not replace) the generated set. (`try:
    from .._generated.models._patch import __all__ as _patch_all
except ImportError:
    _patch_all = []`)
**TYPE_CHECKING guard for patch stubs** — Import `_patch` under `if TYPE_CHECKING:` for IDE type-narrowing without runtime cost. (`if TYPE_CHECKING:
    from .._generated.models._patch import *`)
**Call patch_sdk() at module load** — Always call `_patch_sdk()` after all re-exports to apply any monkey-patches from the generated layer. (`_patch_sdk()`)
**__all__ propagation** — Set `__all__ = __all__` from the generated module, then extend with patch symbols so consumers get a complete, non-redundant export list. (`__all__ = __all__
__all__.extend([p for p in _patch_all if p not in __all__])`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `__init__.py` | Sole file in the package; thin facade that merges generated models + patch overrides into one importable namespace. | Do not add hand-written model classes here; they will be clobbered on the next `make gen-api` run. |

## Anti-Patterns

- Defining or modifying model classes directly in this package instead of using the _patch mechanism
- Importing from _generated internals deeper than _models/_enums/_patch
- Removing the _patch_sdk() call — patch hooks will silently not apply
- Manually maintaining __all__ instead of deriving it from the generated module

## Decisions

- **Thin facade over generated code rather than duplicating types** — All model types are generated from TypeSpec via `make gen-api`; any hand-written copy would drift immediately. The facade gives a stable import path (openmeter.models.*) regardless of generator internals.

<!-- archie:ai-end -->
