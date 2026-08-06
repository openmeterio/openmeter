"""Pydantic models for the OpenMeter v3 SDK baseline."""

from __future__ import annotations

from decimal import Decimal
from typing import Annotated, Any, Literal, Optional, Union

from pydantic import (
    AfterValidator,
    AwareDatetime,
    BaseModel,
    ConfigDict,
    Field,
    StringConstraints,
    model_validator,
)

ULID = Annotated[
    str,
    StringConstraints(pattern=r"^[0-7][0-9A-HJKMNP-TV-Z]{25}$"),
]
ResourceKey = Annotated[
    str,
    StringConstraints(
        min_length=1,
        max_length=64,
        pattern=r"^[a-z0-9]+(?:_[a-z0-9]+)*$",
    ),
]
ResourceName = Annotated[str, StringConstraints(min_length=1, max_length=256)]
ResourceDescription = Annotated[str, StringConstraints(max_length=1024)]
LabelValue = Annotated[
    str,
    StringConstraints(
        min_length=1,
        max_length=63,
        pattern=r"^[a-z0-9A-Z]{1}([a-z0-9A-Z-._]*[a-z0-9A-Z]+)?$",
    ),
]

_RESERVED_LABEL_PREFIXES = (
    "_",
    "kong",
    "konnect",
    "mesh",
    "kic",
)


def _validate_label_key(value: str) -> str:
    if value.startswith(_RESERVED_LABEL_PREFIXES):
        raise ValueError("label key must not use a reserved prefix")

    return value


LabelKey = Annotated[LabelValue, AfterValidator(_validate_label_key)]
Labels = Annotated[dict[LabelKey, LabelValue], Field(max_length=50)]

MeterAggregation = Literal[
    "sum",
    "count",
    "unique_count",
    "avg",
    "min",
    "max",
    "latest",
]
MeterQueryGranularity = Literal["PT1M", "PT1H", "P1D", "P1M"]


class _RequestModel(BaseModel):
    model_config = ConfigDict(extra="forbid", populate_by_name=True)


class _ResponseModel(BaseModel):
    model_config = ConfigDict(extra="ignore", populate_by_name=True)


class PageParams(_RequestModel):
    """Page number and size for page-based v3 collection endpoints."""

    size: Optional[int] = Field(default=None, gt=0)
    number: Optional[int] = Field(default=None, gt=0)


class PageMeta(_ResponseModel):
    """Pagination values returned by the API."""

    number: int
    size: int
    total: int


class PaginatedMeta(_ResponseModel):
    """Pagination metadata envelope returned by list operations."""

    page: PageMeta


class StringFilter(_RequestModel):
    """One string comparison used by meter list filters.

    Exactly one operator must be provided. ``oeq`` and ``ocontains`` accept
    Python lists and are encoded as the API's comma-delimited query value.
    """

    eq: Optional[str] = None
    neq: Optional[str] = None
    contains: Optional[str] = None
    ocontains: Optional[list[str]] = Field(default=None, min_length=1)
    oeq: Optional[list[str]] = Field(default=None, min_length=1)
    gt: Optional[str] = None
    gte: Optional[str] = None
    lt: Optional[str] = None
    lte: Optional[str] = None
    exists: Optional[bool] = None

    @model_validator(mode="after")
    def validate_operator(self) -> StringFilter:
        """Require one representable filter operator."""

        values = [getattr(self, name) for name in self.__class__.model_fields]
        if sum(value is not None for value in values) != 1:
            raise ValueError("exactly one string filter operator must be set")

        for values in (self.oeq, self.ocontains):
            if values is not None and any("," in value for value in values):
                raise ValueError("oeq and ocontains values must not contain commas")

        return self


class MeterFilter(_RequestModel):
    """Filters supported by the v3 list-meters endpoint."""

    key: Optional[Union[str, StringFilter]] = None
    name: Optional[Union[str, StringFilter]] = None


class MeterListParams(_RequestModel):
    """Optional pagination, sorting, and filtering for listing meters."""

    page: Optional[PageParams] = None
    sort: list[str] = Field(default_factory=list)
    filter: Optional[MeterFilter] = None


class PlanAddonListParams(_RequestModel):
    """Optional pagination for listing a plan's add-on associations."""

    page: Optional[PageParams] = None


class _StringQueryFilter(_RequestModel):
    eq: Optional[str] = None
    neq: Optional[str] = None
    in_: Optional[list[str]] = Field(default=None, alias="in", min_length=1, max_length=100)
    nin: Optional[list[str]] = Field(default=None, min_length=1, max_length=100)
    contains: Optional[str] = None
    ncontains: Optional[str] = None
    and_: Optional[list[QueryFilterString]] = Field(
        default=None,
        alias="and",
        min_length=1,
        max_length=10,
    )
    or_: Optional[list[QueryFilterString]] = Field(
        default=None,
        alias="or",
        min_length=1,
        max_length=10,
    )

    @model_validator(mode="after")
    def validate_operator(self) -> _StringQueryFilter:
        """Require the mutually exclusive query-filter operator."""

        values = [getattr(self, name) for name in self.__class__.model_fields]
        if sum(value is not None for value in values) != 1:
            raise ValueError("exactly one query filter operator must be set")

        return self


class QueryFilterString(_StringQueryFilter):
    """Recursive string filter used inside logical query expressions."""


class QueryFilterStringMapItem(_StringQueryFilter):
    """String filter used for one meter-query dimension."""

    exists: Optional[bool] = None


class MeterQueryFilters(_RequestModel):
    """Dimension filters for a meter usage query."""

    dimensions: dict[str, QueryFilterStringMapItem] = Field(default_factory=dict, max_length=10)


class CreateMeterRequest(_RequestModel):
    """Request body for creating a meter."""

    name: ResourceName
    key: ResourceKey
    aggregation: MeterAggregation
    event_type: Annotated[str, StringConstraints(min_length=1)]
    description: Optional[ResourceDescription] = None
    labels: Optional[Labels] = None
    events_from: Optional[AwareDatetime] = None
    value_property: Optional[Annotated[str, StringConstraints(min_length=1)]] = None
    dimensions: Optional[dict[str, str]] = None


class UpdateMeterRequest(_RequestModel):
    """Request body for updating a meter's mutable fields.

    The request schema makes every field optional, but the server currently
    rejects updates that omit ``name``. ``description`` is preserved when
    omitted, while omitting ``labels`` or ``dimensions`` clears that field.
    """

    name: Optional[ResourceName] = None
    description: Optional[ResourceDescription] = None
    labels: Optional[Labels] = None
    dimensions: Optional[dict[str, str]] = None


class Meter(_ResponseModel):
    """Meter configuration returned by the v3 API.

    ``aggregation`` intentionally remains a plain string so newly introduced
    server values remain readable by older SDK versions.
    """

    id: ULID
    key: ResourceKey
    name: ResourceName
    aggregation: str
    event_type: Annotated[str, StringConstraints(min_length=1)]
    created_at: AwareDatetime
    updated_at: AwareDatetime
    description: Optional[ResourceDescription] = None
    labels: dict[str, LabelValue] = Field(default_factory=dict, max_length=50)
    deleted_at: Optional[AwareDatetime] = None
    events_from: Optional[AwareDatetime] = None
    value_property: Optional[Annotated[str, StringConstraints(min_length=1)]] = None
    dimensions: dict[str, str] = Field(default_factory=dict)


class MeterPage(_ResponseModel):
    """One page of meters and its pagination metadata."""

    data: list[Meter]
    meta: PaginatedMeta


class MeterQueryRequest(_RequestModel):
    """Request body for querying aggregated meter usage."""

    from_: Optional[AwareDatetime] = Field(default=None, alias="from")
    to: Optional[AwareDatetime] = None
    granularity: Optional[MeterQueryGranularity] = None
    time_zone: str = "UTC"
    group_by_dimensions: list[str] = Field(default_factory=list, max_length=100)
    filters: Optional[MeterQueryFilters] = None


class MeterQueryRow(_ResponseModel):
    """One aggregated bucket in a meter query result."""

    value: Decimal = Field(allow_inf_nan=False)
    from_: AwareDatetime = Field(alias="from")
    to: AwareDatetime
    dimensions: dict[str, str]


class MeterQueryResult(_ResponseModel):
    """Structured JSON result returned by a meter usage query."""

    data: list[MeterQueryRow]
    from_: Optional[AwareDatetime] = Field(default=None, alias="from")
    to: Optional[AwareDatetime] = None


class AddonReference(_RequestModel):
    """Reference to an add-on by ULID."""

    id: ULID


class ProductCatalogValidationError(_ResponseModel):
    """Server-reported validation issue on a plan-add-on association."""

    code: str
    message: str
    field: str
    attributes: dict[str, Any] = Field(default_factory=dict)


class CreatePlanAddonRequest(_RequestModel):
    """Request body for associating an add-on with a plan."""

    name: ResourceName
    addon: AddonReference
    from_plan_phase: ResourceKey
    description: Optional[ResourceDescription] = None
    labels: Optional[Labels] = None
    max_quantity: Optional[int] = Field(default=None, ge=1)


class UpsertPlanAddonRequest(_RequestModel):
    """Request body for updating a plan-add-on association.

    ``name`` and ``from_plan_phase`` are required by the request schema.
    The server currently ignores ``name`` and ``description``, preserves
    ``labels`` when omitted, and resets ``max_quantity`` to unlimited when
    omitted.
    """

    name: ResourceName
    from_plan_phase: ResourceKey
    description: Optional[ResourceDescription] = None
    labels: Optional[Labels] = None
    max_quantity: Optional[int] = Field(default=None, ge=1)


class PlanAddon(_ResponseModel):
    """Association controlling which add-on is available for a plan."""

    id: ULID
    name: ResourceName
    addon: AddonReference
    from_plan_phase: ResourceKey
    created_at: AwareDatetime
    updated_at: AwareDatetime
    description: Optional[ResourceDescription] = None
    labels: dict[str, LabelValue] = Field(default_factory=dict, max_length=50)
    max_quantity: Optional[int] = Field(default=None, ge=1)
    validation_errors: list[ProductCatalogValidationError] = Field(default_factory=list)
    deleted_at: Optional[AwareDatetime] = None


class PlanAddonPage(_ResponseModel):
    """One page of plan-add-on associations and pagination metadata."""

    data: list[PlanAddon]
    meta: PaginatedMeta


class InvalidParameter(_ResponseModel):
    """Machine-readable context for one invalid request parameter."""

    field: str
    reason: str
    rule: Optional[str] = None
    source: Optional[str] = None
    minimum: Optional[int] = None
    maximum: Optional[int] = None
    choices: Optional[list[Any]] = None
    dependents: Optional[list[Any]] = None


class ProblemDetails(BaseModel):
    """RFC 7807-style error document returned by OpenMeter v3."""

    model_config = ConfigDict(extra="allow")

    status: Optional[int] = None
    title: Optional[str] = None
    type: Optional[str] = None
    instance: Optional[str] = None
    detail: Optional[str] = None
    invalid_parameters: list[InvalidParameter] = Field(default_factory=list)


QueryFilterString.model_rebuild()
QueryFilterStringMapItem.model_rebuild()
