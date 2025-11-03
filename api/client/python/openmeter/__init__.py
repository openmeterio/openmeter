# coding=utf-8
# pylint: disable=wrong-import-position

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ._generated._patch import *  # pylint: disable=unused-wildcard-import

from ._client import Client  # type: ignore
from ._version import VERSION
from ._commit import COMMIT

__version__ = VERSION
__commit__ = COMMIT

try:
    from ._generated._patch import __all__ as _patch_all
    from ._generated._patch import *
except ImportError:
    _patch_all = []
from ._generated._patch import patch_sdk as _patch_sdk

__all__ = [
    "Client",
]
__all__.extend([p for p in _patch_all if p not in __all__])  # pyright: ignore

_patch_sdk()
