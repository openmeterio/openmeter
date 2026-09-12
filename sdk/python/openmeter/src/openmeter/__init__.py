"""OpenMeter Python SDK for the v3 API baseline."""

from typing import TYPE_CHECKING, Any

from ._version import __version__
from .client import Client, Meters, PlanAddons
from .errors import (
    APIError,
    InvalidIDError,
    OpenMeterError,
    PaginationError,
    ResponseTooLargeError,
    TransportError,
)
from .models import (
    AddonReference,
    CreateMeterRequest,
    CreatePlanAddonRequest,
    InvalidParameter,
    Meter,
    MeterAggregation,
    MeterFilter,
    MeterListParams,
    MeterPage,
    MeterQueryFilters,
    MeterQueryGranularity,
    MeterQueryRequest,
    MeterQueryResult,
    MeterQueryRow,
    PageMeta,
    PageParams,
    PaginatedMeta,
    PlanAddon,
    PlanAddonListParams,
    PlanAddonPage,
    ProblemDetails,
    ProductCatalogValidationError,
    QueryFilterString,
    QueryFilterStringMapItem,
    StringFilter,
    UpdateMeterRequest,
    UpsertPlanAddonRequest,
)

if TYPE_CHECKING:
    from .aio import AsyncByteStream, AsyncClient

_ASYNC_EXPORTS = frozenset({"AsyncByteStream", "AsyncClient"})


def __getattr__(name: str) -> Any:
    if name not in _ASYNC_EXPORTS:
        raise AttributeError(f"module {__name__!r} has no attribute {name!r}")

    try:
        from .aio import AsyncByteStream, AsyncClient
    except ModuleNotFoundError as error:
        if error.name != "httpx":
            raise
        raise ImportError(
            "async support requires the 'async' extra: pip install 'openmeter[async]'"
        ) from error

    exports = {
        "AsyncByteStream": AsyncByteStream,
        "AsyncClient": AsyncClient,
    }
    globals().update(exports)
    return exports[name]


__all__ = [
    "APIError",
    "AddonReference",
    "AsyncByteStream",
    "AsyncClient",
    "Client",
    "CreateMeterRequest",
    "CreatePlanAddonRequest",
    "InvalidIDError",
    "InvalidParameter",
    "Meter",
    "MeterAggregation",
    "MeterFilter",
    "MeterListParams",
    "MeterPage",
    "MeterQueryFilters",
    "MeterQueryGranularity",
    "MeterQueryRequest",
    "MeterQueryResult",
    "MeterQueryRow",
    "Meters",
    "OpenMeterError",
    "PageMeta",
    "PageParams",
    "PaginatedMeta",
    "PaginationError",
    "PlanAddon",
    "PlanAddonListParams",
    "PlanAddonPage",
    "PlanAddons",
    "ProblemDetails",
    "ProductCatalogValidationError",
    "QueryFilterString",
    "QueryFilterStringMapItem",
    "ResponseTooLargeError",
    "StringFilter",
    "TransportError",
    "UpdateMeterRequest",
    "UpsertPlanAddonRequest",
    "__version__",
]
