"""Asynchronous OpenMeter v3 client with true async I/O via httpx."""

from __future__ import annotations

import os
from typing import AsyncIterator, Optional

import httpx

from ._transport_async import _AsyncTransport
from .client import (
    _MAX_PAGES,
    _METERS_PATH,
    _last_page,
    _meter_list_query,
    _page_query,
    _page_start,
    _plan_addon_path,
    _plan_addons_path,
    _resource_path,
)
from .errors import PaginationError
from .models import (
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
    UpdateMeterRequest,
    UpsertPlanAddonRequest,
)


class AsyncClient:
    """Asynchronous client for the OpenMeter v3 API baseline.

    HTTP calls use ``httpx.AsyncClient`` for true async I/O, so the event loop
    is never blocked on network waits and no worker-thread pool is involved.

    ``token`` falls back to the ``OPENMETER_TOKEN`` environment variable when
    not passed explicitly. The internal httpx client ignores system proxy
    environment variables (``HTTP_PROXY``, ``NO_PROXY``, etc.) by default, to
    stay deterministic in environments with proxy configuration httpx cannot
    parse (IPv6 addresses, CIDR ranges in ``NO_PROXY``). Pass ``client`` to
    opt into proxy env vars or otherwise fully customize TLS, HTTP/2, or
    authentication behavior; when supplied, ``timeout`` is ignored and the
    caller owns closing it.
    """

    def __init__(
        self,
        base_url: str = "https://openmeter.cloud/api/v3",
        *,
        token: Optional[str] = None,
        timeout: Optional[float] = 30.0,
        client: Optional[httpx.AsyncClient] = None,
    ) -> None:
        transport = _AsyncTransport(
            base_url,
            token=token if token is not None else os.getenv("OPENMETER_TOKEN"),
            timeout=timeout,
            client=client,
        )
        self._transport = transport
        self.meters = AsyncMeters(transport)
        self.plan_addons = AsyncPlanAddons(transport)

    async def close(self) -> None:
        """Close HTTP resources created and owned by this client."""

        await self._transport.aclose()

    def __repr__(self) -> str:
        return f"AsyncClient(base_url={self._transport.base_url!r})"

    async def __aenter__(self) -> AsyncClient:
        return self

    async def __aexit__(self, *_: object) -> None:
        await self.close()


class AsyncByteStream:
    """Async wrapper around an httpx streaming response body."""

    def __init__(self, response: httpx.Response) -> None:
        self._response = response
        self._iterator = response.aiter_bytes()
        self._buffer = b""
        self._exhausted = False

    async def read(self, size: int = -1) -> bytes:
        """Read up to ``size`` bytes, or the entire remaining body if negative."""

        if size < 0:
            chunks = [self._buffer]
            self._buffer = b""
            async for chunk in self._iterator:
                chunks.append(chunk)
            self._exhausted = True
            return b"".join(chunks)

        while len(self._buffer) < size and not self._exhausted:
            try:
                self._buffer += await self._iterator.__anext__()
            except StopAsyncIteration:
                self._exhausted = True

        data, self._buffer = self._buffer[:size], self._buffer[size:]
        return data

    async def close(self) -> None:
        """Close the underlying response body."""

        await self._response.aclose()

    def __aiter__(self) -> AsyncByteStream:
        return self

    async def __anext__(self) -> bytes:
        while b"\n" not in self._buffer and not self._exhausted:
            try:
                self._buffer += await self._iterator.__anext__()
            except StopAsyncIteration:
                self._exhausted = True

        if not self._buffer:
            raise StopAsyncIteration

        newline_index = self._buffer.find(b"\n")
        if newline_index == -1:
            line, self._buffer = self._buffer, b""
        else:
            split_at = newline_index + 1
            line, self._buffer = self._buffer[:split_at], self._buffer[split_at:]

        return line

    async def __aenter__(self) -> AsyncByteStream:
        return self

    async def __aexit__(self, *_: object) -> None:
        await self.close()


class AsyncMeters:
    """Asynchronous operations for v3 meter resources."""

    def __init__(self, transport: _AsyncTransport) -> None:
        self._transport = transport

    async def create(self, request: CreateMeterRequest) -> Meter:
        """Create a meter and return the created resource."""

        return await self._transport.request_json(
            "POST",
            _METERS_PATH,
            Meter,
            body=request,
        )

    async def get(self, meter_id: str) -> Meter:
        """Retrieve a meter by ULID."""

        return await self._transport.request_json(
            "GET", _resource_path(_METERS_PATH, meter_id), Meter
        )

    async def update(self, meter_id: str, request: UpdateMeterRequest) -> Meter:
        """Update a meter's mutable fields and return the updated resource.

        Omitting ``labels`` or ``dimensions`` clears them — see
        ``UpdateMeterRequest`` for per-field merge-vs-replace behavior.
        """

        return await self._transport.request_json(
            "PUT",
            _resource_path(_METERS_PATH, meter_id),
            Meter,
            body=request,
        )

    async def delete(self, meter_id: str) -> None:
        """Delete a meter by ULID."""

        await self._transport.request_empty("DELETE", _resource_path(_METERS_PATH, meter_id))

    async def list(self, params: Optional[MeterListParams] = None) -> MeterPage:
        """Return one page of meters."""

        params = params or MeterListParams()
        return await self._transport.request_json(
            "GET",
            _METERS_PATH,
            MeterPage,
            query=_meter_list_query(params),
        )

    async def list_all(self, params: Optional[MeterListParams] = None) -> AsyncIterator[Meter]:
        """Yield all meters while fetching successive pages lazily."""

        params = params or MeterListParams()
        page_number, page_size = _page_start(params.page)
        for _ in range(_MAX_PAGES):
            page_params = params.model_copy(
                update={"page": PageParams(number=page_number, size=page_size)}
            )
            response = await self.list(page_params)
            if not response.data:
                return

            for meter in response.data:
                yield meter
            if _last_page(
                response.meta.page.number, response.meta.page.size, response.meta.page.total
            ):
                return
            page_number += 1

        raise PaginationError(f"automatic pagination exceeded {_MAX_PAGES} pages")

    async def query(self, meter_id: str, request: MeterQueryRequest) -> MeterQueryResult:
        """Query aggregated meter usage and return structured JSON."""

        return await self._transport.request_json(
            "POST",
            _resource_path(_METERS_PATH, meter_id) + "/query",
            MeterQueryResult,
            body=request,
        )

    async def query_csv(self, meter_id: str, request: MeterQueryRequest) -> bytes:
        """Query aggregated usage and return a bounded CSV response."""

        return await self._transport.request_bytes(
            "POST",
            _resource_path(_METERS_PATH, meter_id) + "/query",
            body=request,
            accept="text/csv",
        )

    async def query_csv_stream(
        self,
        meter_id: str,
        request: MeterQueryRequest,
    ) -> AsyncByteStream:
        """Query usage and return an asynchronously readable CSV stream."""

        response = await self._transport.request_stream(
            "POST",
            _resource_path(_METERS_PATH, meter_id) + "/query",
            body=request,
            accept="text/csv",
        )
        return AsyncByteStream(response)


class AsyncPlanAddons:
    """Asynchronous operations for add-ons nested under a v3 plan."""

    def __init__(self, transport: _AsyncTransport) -> None:
        self._transport = transport

    async def create(self, plan_id: str, request: CreatePlanAddonRequest) -> PlanAddon:
        """Associate an add-on with a plan."""

        return await self._transport.request_json(
            "POST",
            _plan_addons_path(plan_id),
            PlanAddon,
            body=request,
        )

    async def get(self, plan_id: str, plan_addon_id: str) -> PlanAddon:
        """Retrieve one plan-add-on association."""

        return await self._transport.request_json(
            "GET",
            _plan_addon_path(plan_id, plan_addon_id),
            PlanAddon,
        )

    async def update(
        self,
        plan_id: str,
        plan_addon_id: str,
        request: UpsertPlanAddonRequest,
    ) -> PlanAddon:
        """Update a plan-add-on association.

        See ``UpsertPlanAddonRequest`` for per-field merge-vs-replace
        behavior, including which fields the server currently ignores.
        """

        return await self._transport.request_json(
            "PUT",
            _plan_addon_path(plan_id, plan_addon_id),
            PlanAddon,
            body=request,
        )

    async def delete(self, plan_id: str, plan_addon_id: str) -> None:
        """Remove an add-on association from a plan."""

        await self._transport.request_empty("DELETE", _plan_addon_path(plan_id, plan_addon_id))

    async def list(
        self,
        plan_id: str,
        params: Optional[PlanAddonListParams] = None,
    ) -> PlanAddonPage:
        """Return one page of add-ons associated with a plan."""

        params = params or PlanAddonListParams()
        return await self._transport.request_json(
            "GET",
            _plan_addons_path(plan_id),
            PlanAddonPage,
            query=_page_query(params.page),
        )

    async def list_all(
        self,
        plan_id: str,
        params: Optional[PlanAddonListParams] = None,
    ) -> AsyncIterator[PlanAddon]:
        """Yield all add-on associations for a plan lazily."""

        params = params or PlanAddonListParams()
        page_number, page_size = _page_start(params.page)
        for _ in range(_MAX_PAGES):
            page_params = params.model_copy(
                update={"page": PageParams(number=page_number, size=page_size)}
            )
            response = await self.list(plan_id, page_params)
            if not response.data:
                return

            for plan_addon in response.data:
                yield plan_addon
            if _last_page(
                response.meta.page.number, response.meta.page.size, response.meta.page.total
            ):
                return
            page_number += 1

        raise PaginationError(f"automatic pagination exceeded {_MAX_PAGES} pages")
