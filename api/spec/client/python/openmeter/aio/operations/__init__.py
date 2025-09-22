# coding=utf-8
# pylint: disable=wrong-import-position

from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ._patch import *  # pylint: disable=unused-wildcard-import

from ._operations import OpenMeterCloudOperations  # type: ignore
from ._operations import EventsOperations  # type: ignore
from ._operations import EventsV2Operations  # type: ignore
from ._operations import MetersOperations  # type: ignore
from ._operations import DebugOperations  # type: ignore

from ._patch import __all__ as _patch_all
from ._patch import *
from ._patch import patch_sdk as _patch_sdk

__all__ = [
    "OpenMeterCloudOperations",
    "EventsOperations",
    "EventsV2Operations",
    "MetersOperations",
    "DebugOperations",
]
__all__.extend([p for p in _patch_all if p not in __all__])  # pyright: ignore
_patch_sdk()
