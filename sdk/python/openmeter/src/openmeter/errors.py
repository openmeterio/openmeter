"""Exceptions raised by the OpenMeter v3 SDK."""

from __future__ import annotations

from typing import Optional

from .models import ProblemDetails


class OpenMeterError(Exception):
    """Base class for SDK-specific errors."""


class InvalidIDError(OpenMeterError, ValueError):
    """Raised before a request when a resource ID is not a valid ULID."""


class TransportError(OpenMeterError):
    """Raised when no HTTP response could be obtained."""


class ResponseTooLargeError(OpenMeterError):
    """Raised when a buffered success response exceeds the safety limit."""

    def __init__(self, limit: int) -> None:
        self.limit = limit
        super().__init__(f"response body exceeds the {limit}-byte buffer limit")


class PaginationError(OpenMeterError):
    """Raised when automatic pagination exceeds its loop-safety limit."""


class APIError(OpenMeterError):
    """Non-successful HTTP response returned by the OpenMeter API.

    Attributes expose the stable RFC 7807 fields while ``raw_body`` preserves
    the response bytes for diagnostics. Error bodies are capped before they are
    stored, and ``raw_body_truncated`` reports when the cap was reached.
    """

    def __init__(
        self,
        status_code: int,
        problem: ProblemDetails,
        raw_body: bytes,
        *,
        raw_body_truncated: bool = False,
    ) -> None:
        self.status_code = status_code
        self.problem = problem
        self.raw_body = raw_body
        self.raw_body_truncated = raw_body_truncated
        super().__init__(self._message())

    @property
    def title(self) -> Optional[str]:
        """Short, stable description of the problem category."""

        return self.problem.title

    @property
    def detail(self) -> Optional[str]:
        """Human-readable detail for this occurrence."""

        return self.problem.detail

    @property
    def type(self) -> Optional[str]:
        """Problem type identifier supplied by the API."""

        return self.problem.type

    @property
    def instance(self) -> Optional[str]:
        """Correlation identifier supplied by the API."""

        return self.problem.instance

    def _message(self) -> str:
        summary = self.detail or self.title
        if summary:
            return f"OpenMeter API returned HTTP {self.status_code}: {summary}"

        raw = self.raw_body[:512].decode("utf-8", errors="replace").strip()
        if raw:
            return f"OpenMeter API returned HTTP {self.status_code}: {raw}"

        return f"OpenMeter API returned HTTP {self.status_code}"
