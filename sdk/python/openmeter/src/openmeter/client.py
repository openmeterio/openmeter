"""Synchronous OpenMeter v3 client and resource groups."""

from __future__ import annotations

import os
from typing import BinaryIO, Iterator, Optional, Union
from urllib.request import OpenerDirector

from pydantic import TypeAdapter, ValidationError

from ._transport import _Transport
from .errors import InvalidIDError, PaginationError
from .models import (
    ULID,
    CreateMeterRequest,
    CreatePlanAddonRequest,
    Meter,
    MeterListParams,
    MeterPage,
    MeterQueryRequest,
    MeterQueryResult,
    PageParams,
    PlanAddon,
    PlanAddonListParams,
    PlanAddonPage,
    StringFilter,
    UpdateMeterRequest,
    UpsertPlanAddonRequest,
)

_METERS_PATH = "/openmeter/meters"
_PLANS_PATH = "/openmeter/plans"
_MAX_PAGES = 10_000
_ULID_ADAPTER = TypeAdapter(ULID)


class Client:
    """Synchronous client for the OpenMeter v3 API baseline.

    ``base_url`` must include the deployment's v3 prefix, for example
    ``https://openmeter.cloud/api/v3``. The client uses only the Python standard
    library for HTTP and accepts an optional ``urllib`` opener for custom proxy,
    TLS, authentication, or test behavior.

    ``token`` falls back to the ``OPENMETER_TOKEN`` environment variable when
    not passed explicitly.
    """

    def __init__(
        self,
        base_url: str = "https://openmeter.cloud/api/v3",
        *,
        token: Optional[str] = None,
        timeout: Optional[float] = 30.0,
        opener: Optional[OpenerDirector] = None,
    ) -> None:
        transport = _Transport(
            base_url,
            token=token if token is not None else os.getenv("OPENMETER_TOKEN"),
            timeout=timeout,
            opener=opener,
        )
        self._transport = transport
        self.meters = Meters(transport)
        self.plan_addons = PlanAddons(transport)

    def close(self) -> None:
        """Close the client.

        The standard-library opener owns no persistent user-managed resource,
        so this method is currently a no-op and exists for context-manager use.
        """

    def __repr__(self) -> str:
        return f"Client(base_url={self._transport.base_url!r})"

    def __enter__(self) -> Client:
        return self

    def __exit__(self, *_: object) -> None:
        self.close()


class Meters:
    """Synchronous operations for v3 meter resources."""

    def __init__(self, transport: _Transport) -> None:
        self._transport = transport

    def create(self, request: CreateMeterRequest) -> Meter:
        """Create a meter and return the created resource."""

        return self._transport.request_json(
            "POST",
            _METERS_PATH,
            Meter,
            body=request,
        )

    def get(self, meter_id: str) -> Meter:
        """Retrieve a meter by ULID."""

        return self._transport.request_json("GET", _resource_path(_METERS_PATH, meter_id), Meter)

    def update(self, meter_id: str, request: UpdateMeterRequest) -> Meter:
        """Update a meter's mutable fields and return the updated resource.

        Omitting ``labels`` or ``dimensions`` clears them — see
        ``UpdateMeterRequest`` for per-field merge-vs-replace behavior.
        """

        return self._transport.request_json(
            "PUT",
            _resource_path(_METERS_PATH, meter_id),
            Meter,
            body=request,
        )

    def delete(self, meter_id: str) -> None:
        """Delete a meter by ULID."""

        self._transport.request_empty("DELETE", _resource_path(_METERS_PATH, meter_id))

    def list(self, params: Optional[MeterListParams] = None) -> MeterPage:
        """Return one page of meters."""

        params = params or MeterListParams()
        return self._transport.request_json(
            "GET",
            _METERS_PATH,
            MeterPage,
            query=_meter_list_query(params),
        )

    def list_all(self, params: Optional[MeterListParams] = None) -> Iterator[Meter]:
        """Yield all meters while fetching successive pages lazily."""

        params = params or MeterListParams()
        page_number, page_size = _page_start(params.page)
        for _ in range(_MAX_PAGES):
            page_params = params.model_copy(
                update={"page": PageParams(number=page_number, size=page_size)}
            )
            response = self.list(page_params)
            if not response.data:
                return

            yield from response.data
            if _last_page(
                response.meta.page.number, response.meta.page.size, response.meta.page.total
            ):
                return
            page_number += 1

        raise PaginationError(f"automatic pagination exceeded {_MAX_PAGES} pages")

    def query(self, meter_id: str, request: MeterQueryRequest) -> MeterQueryResult:
        """Query aggregated meter usage and return structured JSON."""

        return self._transport.request_json(
            "POST",
            _resource_path(_METERS_PATH, meter_id) + "/query",
            MeterQueryResult,
            body=request,
        )

    def query_csv(self, meter_id: str, request: MeterQueryRequest) -> bytes:
        """Query aggregated usage and return a bounded CSV response."""

        return self._transport.request_bytes(
            "POST",
            _resource_path(_METERS_PATH, meter_id) + "/query",
            body=request,
            accept="text/csv",
        )

    def query_csv_stream(self, meter_id: str, request: MeterQueryRequest) -> BinaryIO:
        """Query usage and return an unbuffered CSV body that the caller closes."""

        return self._transport.request_stream(
            "POST",
            _resource_path(_METERS_PATH, meter_id) + "/query",
            body=request,
            accept="text/csv",
        )


class PlanAddons:
    """Synchronous operations for add-ons nested under a v3 plan."""

    def __init__(self, transport: _Transport) -> None:
        self._transport = transport

    def create(self, plan_id: str, request: CreatePlanAddonRequest) -> PlanAddon:
        """Associate an add-on with a plan."""

        return self._transport.request_json(
            "POST",
            _plan_addons_path(plan_id),
            PlanAddon,
            body=request,
        )

    def get(self, plan_id: str, plan_addon_id: str) -> PlanAddon:
        """Retrieve one plan-add-on association."""

        return self._transport.request_json(
            "GET",
            _plan_addon_path(plan_id, plan_addon_id),
            PlanAddon,
        )

    def update(
        self,
        plan_id: str,
        plan_addon_id: str,
        request: UpsertPlanAddonRequest,
    ) -> PlanAddon:
        """Update a plan-add-on association.

        See ``UpsertPlanAddonRequest`` for per-field merge-vs-replace
        behavior, including which fields the server currently ignores.
        """

        return self._transport.request_json(
            "PUT",
            _plan_addon_path(plan_id, plan_addon_id),
            PlanAddon,
            body=request,
        )

    def delete(self, plan_id: str, plan_addon_id: str) -> None:
        """Remove an add-on association from a plan."""

        self._transport.request_empty("DELETE", _plan_addon_path(plan_id, plan_addon_id))

    def list(
        self,
        plan_id: str,
        params: Optional[PlanAddonListParams] = None,
    ) -> PlanAddonPage:
        """Return one page of add-ons associated with a plan."""

        params = params or PlanAddonListParams()
        return self._transport.request_json(
            "GET",
            _plan_addons_path(plan_id),
            PlanAddonPage,
            query=_page_query(params.page),
        )

    def list_all(
        self,
        plan_id: str,
        params: Optional[PlanAddonListParams] = None,
    ) -> Iterator[PlanAddon]:
        """Yield all add-on associations for a plan lazily."""

        params = params or PlanAddonListParams()
        page_number, page_size = _page_start(params.page)
        for _ in range(_MAX_PAGES):
            page_params = params.model_copy(
                update={"page": PageParams(number=page_number, size=page_size)}
            )
            response = self.list(plan_id, page_params)
            if not response.data:
                return

            yield from response.data
            if _last_page(
                response.meta.page.number, response.meta.page.size, response.meta.page.total
            ):
                return
            page_number += 1

        raise PaginationError(f"automatic pagination exceeded {_MAX_PAGES} pages")


def _resource_path(base: str, resource_id: str) -> str:
    try:
        resource_id = _ULID_ADAPTER.validate_python(resource_id)
    except ValidationError:
        raise InvalidIDError("resource ID must be a valid ULID") from None

    return base + "/" + resource_id


def _plan_addons_path(plan_id: str) -> str:
    return _resource_path(_PLANS_PATH, plan_id) + "/addons"


def _plan_addon_path(plan_id: str, plan_addon_id: str) -> str:
    return _resource_path(_plan_addons_path(plan_id), plan_addon_id)


def _page_query(page: Optional[PageParams]) -> list[tuple[str, str]]:
    query: list[tuple[str, str]] = []
    if page is not None:
        if page.size is not None:
            query.append(("page[size]", str(page.size)))
        if page.number is not None:
            query.append(("page[number]", str(page.number)))

    return query


def _meter_list_query(params: MeterListParams) -> list[tuple[str, str]]:
    query = _page_query(params.page)
    if params.sort:
        query.append(("sort", ",".join(params.sort)))
    if params.filter is not None:
        _add_string_filter(query, "filter[key]", params.filter.key)
        _add_string_filter(query, "filter[name]", params.filter.name)

    return query


def _add_string_filter(
    query: list[tuple[str, str]],
    name: str,
    value: Optional[Union[str, StringFilter]],
) -> None:
    if value is None:
        return
    if isinstance(value, str):
        query.append((name, value))
        return

    for operator, operand in value.model_dump(exclude_none=True).items():
        encoded_operator = "$exists" if operator == "exists" else operator
        if isinstance(operand, list):
            encoded_operand = ",".join(operand)
        elif isinstance(operand, bool):
            encoded_operand = str(operand).lower()
        else:
            encoded_operand = str(operand)
        query.append((f"{name}[{encoded_operator}]", encoded_operand))


def _page_start(page: Optional[PageParams]) -> tuple[int, int]:
    if page is None:
        return 1, 100

    return page.number or 1, page.size or 100


def _last_page(number: int, size: int, total: int) -> bool:
    return total > 0 and number * size >= total
