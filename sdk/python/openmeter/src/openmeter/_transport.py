"""Standard-library HTTP transport used by the synchronous client."""

from __future__ import annotations

import time
from typing import Any, BinaryIO, Optional, Type, TypeVar
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urlsplit
from urllib.request import OpenerDirector, Request, build_opener

from pydantic import BaseModel, ValidationError

from ._version import __version__
from .errors import APIError, ResponseTooLargeError, TransportError
from .models import ProblemDetails

_BUFFER_LIMIT = 10 * 1024 * 1024
_ERROR_LIMIT = 1024 * 1024
_USER_AGENT = f"openmeter-python/{__version__}"

# GET/PUT/DELETE are idempotent, so a failed attempt is safe to retry; POST is
# excluded because it creates resources and a retried POST could double-create.
_RETRYABLE_METHODS = frozenset({"GET", "PUT", "DELETE"})
_RETRYABLE_STATUS_CODES = frozenset({502, 503, 504})
_MAX_RETRIES = 2
_RETRY_BACKOFF_SECONDS = 0.5

ModelT = TypeVar("ModelT", bound=BaseModel)


class _Transport:
    def __init__(
        self,
        base_url: str,
        *,
        token: Optional[str],
        timeout: Optional[float],
        opener: Optional[OpenerDirector],
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
        self.timeout = timeout
        self.opener = opener or build_opener()

    def request_json(
        self,
        method: str,
        path: str,
        response_model: Type[ModelT],
        *,
        query: Optional[list[tuple[str, str]]] = None,
        body: Optional[BaseModel] = None,
    ) -> ModelT:
        response = self._open(method, path, query=query, body=body, accept="application/json")
        with response:
            data = self._read_success(response)

        return response_model.model_validate_json(data)

    def request_bytes(
        self,
        method: str,
        path: str,
        *,
        query: Optional[list[tuple[str, str]]] = None,
        body: Optional[BaseModel] = None,
        accept: str,
    ) -> bytes:
        response = self._open(method, path, query=query, body=body, accept=accept)
        with response:
            return self._read_success(response)

    def request_stream(
        self,
        method: str,
        path: str,
        *,
        body: BaseModel,
        accept: str,
    ) -> BinaryIO:
        return self._open(method, path, body=body, accept=accept)

    def request_empty(self, method: str, path: str) -> None:
        response = self._open(method, path, accept="application/json")
        with response:
            self._read_success(response)

    def _open(
        self,
        method: str,
        path: str,
        *,
        query: Optional[list[tuple[str, str]]] = None,
        body: Optional[BaseModel] = None,
        accept: str,
    ) -> Any:
        url = self.base_url + path
        if query:
            url += "?" + urlencode(query)

        headers = {
            "Accept": accept,
            "User-Agent": _USER_AGENT,
        }
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"

        data = None
        if body is not None:
            data = body.model_dump_json(
                by_alias=True,
                exclude_none=True,
                exclude_unset=True,
            ).encode("utf-8")
            headers["Content-Type"] = "application/json"

        retries_left = _MAX_RETRIES if method in _RETRYABLE_METHODS else 0
        attempt = 0
        while True:
            request = Request(url, data=data, headers=headers, method=method)
            try:
                return self.opener.open(request, timeout=self.timeout)
            except HTTPError as error:
                with error:
                    raw_body, truncated = _read_error(error)
                if retries_left > 0 and error.code in _RETRYABLE_STATUS_CODES:
                    retries_left -= 1
                    time.sleep(_RETRY_BACKOFF_SECONDS * (2**attempt))
                    attempt += 1
                    continue
                try:
                    problem = ProblemDetails.model_validate_json(raw_body)
                except (ValidationError, ValueError):
                    problem = ProblemDetails(status=error.code)
                raise APIError(
                    error.code,
                    problem,
                    raw_body,
                    raw_body_truncated=truncated,
                ) from None
            except (URLError, TimeoutError, OSError) as error:
                if retries_left > 0:
                    retries_left -= 1
                    time.sleep(_RETRY_BACKOFF_SECONDS * (2**attempt))
                    attempt += 1
                    continue
                raise TransportError(f"OpenMeter request failed: {error}") from error

    @staticmethod
    def _read_success(response: Any) -> bytes:
        data = response.read(_BUFFER_LIMIT + 1)
        if len(data) > _BUFFER_LIMIT:
            raise ResponseTooLargeError(_BUFFER_LIMIT)

        return data


def _read_error(response: Any) -> tuple[bytes, bool]:
    data = response.read(_ERROR_LIMIT + 1)
    if len(data) > _ERROR_LIMIT:
        return data[:_ERROR_LIMIT], True

    return data, False
