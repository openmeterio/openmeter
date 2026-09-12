"""httpx-based transport giving ``AsyncClient`` true async I/O."""

from __future__ import annotations

import asyncio
from typing import Optional, Type, TypeVar
from urllib.parse import urlsplit

import httpx
from pydantic import BaseModel, ValidationError

from ._version import __version__
from .errors import APIError, ResponseTooLargeError, TransportError
from .models import ProblemDetails

_BUFFER_LIMIT = 10 * 1024 * 1024
_ERROR_LIMIT = 1024 * 1024
_USER_AGENT = f"openmeter-python/{__version__}"

# GET/PUT/DELETE and read-only meter queries are safe to retry. Other POST
# requests create resources and could double-create when retried.
_RETRYABLE_METHODS = frozenset({"GET", "PUT", "DELETE"})
_RETRYABLE_POST_PATH_SUFFIX = "/query"
_RETRYABLE_STATUS_CODES = frozenset({502, 503, 504})
_MAX_RETRIES = 2
_RETRY_BACKOFF_SECONDS = 0.5

ModelT = TypeVar("ModelT", bound=BaseModel)


class _AsyncTransport:
    def __init__(
        self,
        base_url: str,
        *,
        token: Optional[str],
        timeout: Optional[float],
        trust_env: bool,
        client: Optional[httpx.AsyncClient],
    ) -> None:
        parsed = urlsplit(base_url)
        if parsed.scheme not in {"http", "https"} or not parsed.netloc:
            raise ValueError("base_url must be an absolute HTTP or HTTPS URL")
        if parsed.query or parsed.fragment:
            raise ValueError("base_url must not contain a query string or fragment")
        if timeout is not None and timeout <= 0:
            raise ValueError("timeout must be positive or None")

        self.base_url = base_url.rstrip("/")
        self.token = token
        self._owns_client = client is None
        self._client = (
            client
            if client is not None
            else httpx.AsyncClient(timeout=timeout, trust_env=trust_env)
        )

    async def request_json(
        self,
        method: str,
        path: str,
        response_model: Type[ModelT],
        *,
        query: Optional[list[tuple[str, str]]] = None,
        body: Optional[BaseModel] = None,
    ) -> ModelT:
        response = await self._open(method, path, query=query, body=body, accept="application/json")
        try:
            data = await self._read_success(response)
        finally:
            await response.aclose()

        return response_model.model_validate_json(data)

    async def request_bytes(
        self,
        method: str,
        path: str,
        *,
        query: Optional[list[tuple[str, str]]] = None,
        body: Optional[BaseModel] = None,
        accept: str,
    ) -> bytes:
        response = await self._open(method, path, query=query, body=body, accept=accept)
        try:
            return await self._read_success(response)
        finally:
            await response.aclose()

    async def request_stream(
        self,
        method: str,
        path: str,
        *,
        body: BaseModel,
        accept: str,
    ) -> httpx.Response:
        return await self._open(method, path, body=body, accept=accept)

    async def request_empty(self, method: str, path: str) -> None:
        response = await self._open(method, path, accept="application/json")
        try:
            await self._read_success(response)
        finally:
            await response.aclose()

    async def aclose(self) -> None:
        """Close the HTTP client when it was created by this transport."""

        if self._owns_client:
            await self._client.aclose()

    async def _open(
        self,
        method: str,
        path: str,
        *,
        query: Optional[list[tuple[str, str]]] = None,
        body: Optional[BaseModel] = None,
        accept: str,
    ) -> httpx.Response:
        url = self.base_url + path
        headers = {
            "Accept": accept,
            "User-Agent": _USER_AGENT,
        }
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"

        content = None
        if body is not None:
            content = body.model_dump_json(
                by_alias=True,
                exclude_none=True,
                exclude_unset=True,
            ).encode("utf-8")
            headers["Content-Type"] = "application/json"

        params = httpx.QueryParams(tuple(query)) if query else None
        retryable = method in _RETRYABLE_METHODS or (
            method == "POST"
            and path.startswith("/openmeter/meters/")
            and path.endswith(_RETRYABLE_POST_PATH_SUFFIX)
        )
        retries_left = _MAX_RETRIES if retryable else 0
        attempt = 0
        while True:
            request = self._client.build_request(
                method, url, params=params, content=content, headers=headers
            )
            try:
                response = await self._client.send(request, stream=True, follow_redirects=False)
            except httpx.TransportError as error:
                if retries_left > 0:
                    retries_left -= 1
                    await asyncio.sleep(_RETRY_BACKOFF_SECONDS * (2**attempt))
                    attempt += 1
                    continue
                raise TransportError(f"OpenMeter request failed: {error}") from error

            if 200 <= response.status_code < 300:
                return response

            raw_body, truncated = await self._read_bounded(response, _ERROR_LIMIT)
            await response.aclose()
            if retries_left > 0 and response.status_code in _RETRYABLE_STATUS_CODES:
                retries_left -= 1
                await asyncio.sleep(_RETRY_BACKOFF_SECONDS * (2**attempt))
                attempt += 1
                continue

            try:
                problem = ProblemDetails.model_validate_json(raw_body)
            except (ValidationError, ValueError):
                problem = ProblemDetails(status=response.status_code)
            raise APIError(
                response.status_code,
                problem,
                raw_body,
                raw_body_truncated=truncated,
            ) from None

    @staticmethod
    async def _read_success(response: httpx.Response) -> bytes:
        data, truncated = await _AsyncTransport._read_bounded(response, _BUFFER_LIMIT)
        if truncated:
            raise ResponseTooLargeError(_BUFFER_LIMIT)

        return data

    @staticmethod
    async def _read_bounded(response: httpx.Response, limit: int) -> tuple[bytes, bool]:
        chunks: list[bytes] = []
        total = 0
        async for chunk in response.aiter_bytes():
            chunks.append(chunk)
            total += len(chunk)
            if total > limit:
                break

        data = b"".join(chunks)
        if len(data) > limit:
            return data[:limit], True

        return data, False
