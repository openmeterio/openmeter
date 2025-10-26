# coding=utf-8
# pylint: disable=wrong-import-position

"""
Re-exports all operations from the generated code for a cleaner API surface.
"""

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ..._generated.aio.operations._patch import *  # pylint: disable=unused-wildcard-import

# Re-export all operations from _generated.aio.operations
from ..._generated.aio.operations._operations import *  # noqa: F401, F403

try:
    from ..._generated.aio.operations._patch import __all__ as _patch_all
    from ..._generated.aio.operations._patch import *  # noqa: F401, F403
except ImportError:
    _patch_all = []
from ..._generated.aio.operations._patch import patch_sdk as _patch_sdk

# Import and re-export the __all__ list
from ..._generated.aio.operations import __all__

__all__ = __all__
__all__.extend([p for p in _patch_all if p not in __all__])  # pyright: ignore

_patch_sdk()
