"""Behavioral tests for sync and async v3 SDK clients."""

from __future__ import annotations

import asyncio
import io
import json
import os
import subprocess
import sys
import threading
import unittest
from datetime import datetime, timezone
from decimal import Decimal
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import Any, Optional
from unittest.mock import Mock, patch
from urllib.error import HTTPError, URLError
from urllib.parse import parse_qs, urlsplit

import httpx
from pydantic import ValidationError

from openmeter import (
    AddonReference,
    APIError,
    AsyncClient,
    Client,
    CreateMeterRequest,
    CreatePlanAddonRequest,
    InvalidIDError,
    Meter,
    MeterFilter,
    MeterListParams,
    MeterQueryFilters,
    MeterQueryRequest,
    PageParams,
    PlanAddonListParams,
    ProblemDetails,
    QueryFilterString,
    QueryFilterStringMapItem,
    ResponseTooLargeError,
    StringFilter,
    TransportError,
    UpdateMeterRequest,
    UpsertPlanAddonRequest,
)

METER_ID = "01G65Z755AFWAKHE12NY0CQ9FH"
SECOND_METER_ID = "01G65Z755AFWAKHE12NY0CQ9FJ"
PLAN_ID = "01G65Z755AFWAKHE12NY0CQ9FK"
PLAN_ADDON_ID = "01G65Z755AFWAKHE12NY0CQ9FM"
ADDON_ID = "01G65Z755AFWAKHE12NY0CQ9FN"
ERROR_ID = "01G65Z755AFWAKHE12NY0CQ9FP"
REDIRECT_ID = "01G65Z755AFWAKHE12NY0CQ9FQ"
NOW = "2026-07-14T10:00:00Z"
BUFFER_LIMIT = 10 * 1024 * 1024
ERROR_LIMIT = 1024 * 1024


def _meter(meter_id: str = METER_ID, key: str = "tokens_total") -> dict[str, Any]:
    return {
        "id": meter_id,
        "key": key,
        "name": "Tokens Total",
        "aggregation": "sum",
        "event_type": "prompt",
        "created_at": NOW,
        "updated_at": NOW,
        "unknown_future_field": "accepted",
    }


def _plan_addon() -> dict[str, Any]:
    return {
        "id": PLAN_ADDON_ID,
        "name": "Pro add-on",
        "addon": {"id": ADDON_ID},
        "from_plan_phase": "default",
        "created_at": NOW,
        "updated_at": NOW,
        "validation_errors": [],
    }


class _Handler(BaseHTTPRequestHandler):
    requests: list[dict[str, Any]] = []
    # Maps a path to the number of remaining transient 503s to return before
    # falling through to its normal response; used to test retry behavior.
    fail_counts: dict[str, int] = {}
    response_gate: Optional[threading.Event] = None

    def _maybe_fail(self, path: str) -> bool:
        remaining = self.fail_counts.get(path, 0)
        if remaining <= 0:
            return False

        self.fail_counts[path] = remaining - 1
        self._json({"status": 503, "title": "Service Unavailable"}, status=503)
        return True

    def do_GET(self) -> None:  # noqa: N802
        path, query, _ = self._capture()
        if self.response_gate is not None and not self.response_gate.wait(timeout=1):
            self._json({"status": 500, "title": "Event loop blocked"}, status=500)
            return
        if self._maybe_fail(path):
            return
        if path == "/openmeter/meters":
            page = int(query.get("page[number]", ["1"])[0])
            data = [_meter()] if page == 1 else [_meter(SECOND_METER_ID, "requests_total")]
            self._json(
                {
                    "data": data,
                    "meta": {"page": {"number": page, "size": 1, "total": 2}},
                }
            )
            return
        if path.endswith(REDIRECT_ID):
            self.send_response(302)
            self.send_header("Location", f"/api/v3/openmeter/meters/{METER_ID}")
            self.end_headers()
            return
        if path.endswith(ERROR_ID):
            self._json(
                {
                    "status": 404,
                    "title": "Not Found",
                    "type": "https://httpstatuses.com/404",
                    "instance": "kong:trace:test",
                    "detail": "Meter was not found",
                },
                status=404,
                content_type="application/problem+json",
            )
            return
        if path.endswith("/addons"):
            self._json(
                {
                    "data": [_plan_addon()],
                    "meta": {"page": {"number": 1, "size": 100, "total": 1}},
                }
            )
            return
        if "/addons/" in path:
            self._json(_plan_addon())
            return

        self._json(_meter())

    def do_POST(self) -> None:  # noqa: N802
        path, _, body = self._capture()
        if self._maybe_fail(path):
            return
        if path.endswith("/query"):
            if self.headers.get("Accept") == "text/csv":
                self._bytes(
                    b"from,to,value\n2026-07-14T09:00:00Z,2026-07-14T10:00:00Z,12\n", "text/csv"
                )
            else:
                self._json(
                    {
                        "from": "2026-07-14T09:00:00Z",
                        "to": NOW,
                        "data": [
                            {
                                "value": "12.50",
                                "from": "2026-07-14T09:00:00Z",
                                "to": NOW,
                                "dimensions": {"model": "gpt-4.1"},
                            }
                        ],
                    }
                )
            return
        if path.endswith("/addons"):
            response = _plan_addon()
            response.update({key: value for key, value in body.items() if key != "addon"})
            self._json(response, status=201)
            return

        response = _meter(key=body["key"])
        response.update(
            {key: value for key, value in body.items() if key not in {"key", "aggregation"}}
        )
        self._json(response, status=201)

    def do_PUT(self) -> None:  # noqa: N802
        path, _, body = self._capture()
        response = _plan_addon() if "/addons/" in path else _meter()
        response.update(body)
        self._json(response)

    def do_DELETE(self) -> None:  # noqa: N802
        self._capture()
        self.send_response(204)
        self.end_headers()

    def _capture(self) -> tuple[str, dict[str, list[str]], dict[str, Any]]:
        parsed = urlsplit(self.path)
        path = parsed.path.removeprefix("/api/v3")
        length = int(self.headers.get("Content-Length", "0"))
        raw_body = self.rfile.read(length) if length else b""
        body = json.loads(raw_body) if raw_body else {}
        query = parse_qs(parsed.query)
        self.__class__.requests.append(
            {
                "method": self.command,
                "path": path,
                "raw_path": self.path,
                "query": query,
                "headers": dict(self.headers),
                "body": body,
            }
        )
        return path, query, body

    def _json(
        self,
        value: dict[str, Any],
        *,
        status: int = 200,
        content_type: str = "application/json",
    ) -> None:
        self._bytes(json.dumps(value).encode(), content_type, status=status)

    def _bytes(self, value: bytes, content_type: str, *, status: int = 200) -> None:
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(value)))
        self.end_headers()
        self.wfile.write(value)

    def log_message(self, *_: object) -> None:
        return


_server: Optional[ThreadingHTTPServer] = None
_server_thread: Optional[threading.Thread] = None
BASE_URL = ""


def setUpModule() -> None:  # noqa: N802
    global _server, _server_thread, BASE_URL
    _server = ThreadingHTTPServer(("127.0.0.1", 0), _Handler)
    _server_thread = threading.Thread(target=_server.serve_forever, daemon=True)
    _server_thread.start()
    BASE_URL = f"http://127.0.0.1:{_server.server_port}/api/v3"


def tearDownModule() -> None:  # noqa: N802
    if _server is not None:
        _server.shutdown()
        _server.server_close()
    if _server_thread is not None:
        _server_thread.join(timeout=5)


class ModelTests(unittest.TestCase):
    def test_request_models_validate_openapi_constraints(self) -> None:
        with self.assertRaises(ValidationError):
            CreateMeterRequest(name="", key="Tokens", aggregation="median", event_type="")

        with self.assertRaises(ValidationError):
            CreatePlanAddonRequest(
                name="Pro",
                addon=AddonReference(id=ADDON_ID),
                from_plan_phase="default",
                max_quantity=0,
            )

        with self.assertRaises(ValidationError):
            MeterQueryRequest(from_=datetime(2026, 7, 14, 9))

    def test_query_filter_requires_exactly_one_operator(self) -> None:
        self.assertEqual(QueryFilterStringMapItem(eq="acme").eq, "acme")
        nested = QueryFilterStringMapItem(and_=[QueryFilterString(eq="acme")])
        self.assertEqual(nested.and_[0].eq, "acme")

        with self.assertRaises(ValidationError):
            QueryFilterStringMapItem(eq="acme", contains="ac")

    def test_response_models_accept_future_fields_and_enum_values(self) -> None:
        value = _meter()
        value["aggregation"] = "future_aggregation"
        meter = Meter.model_validate(value)
        self.assertEqual(meter.aggregation, "future_aggregation")

    def test_api_error_exposes_invalid_parameters(self) -> None:
        problem = ProblemDetails.model_validate(
            {"invalid_parameters": [{"field": "name", "reason": "must not be empty"}]}
        )
        error = APIError(400, problem, b"")

        self.assertEqual(error.invalid_parameters[0].field, "name")

    def test_request_labels_validate_keys(self) -> None:
        request = CreateMeterRequest(
            name="Tokens Total",
            key="tokens_total",
            aggregation="sum",
            event_type="prompt",
            labels={"openmeter_internal": "platform"},
        )
        self.assertEqual(request.labels, {"openmeter_internal": "platform"})

        invalid_keys = [
            "",
            "bad_",
            "a" * 64,
            "_internal",
            "kong_internal",
            "konnect_internal",
            "mesh_internal",
            "kic_internal",
        ]
        for label_key in invalid_keys:
            with self.subTest(label_key=label_key):
                with self.assertRaises(ValidationError):
                    CreateMeterRequest(
                        name="Tokens Total",
                        key="tokens_total",
                        aggregation="sum",
                        event_type="prompt",
                        labels={label_key: "value"},
                    )

    def test_sync_import_does_not_require_async_dependency(self) -> None:
        script = """
import builtins

real_import = builtins.__import__


def import_without_httpx(name, *args, **kwargs):
    if name == "httpx" or name.startswith("httpx."):
        raise ModuleNotFoundError("No module named 'httpx'", name="httpx")
    return real_import(name, *args, **kwargs)


builtins.__import__ = import_without_httpx

import openmeter

assert openmeter.Client
try:
    openmeter.AsyncClient
except ImportError as error:
    assert "openmeter[async]" in str(error)
else:
    raise AssertionError("AsyncClient must require the async extra")
"""
        subprocess.run([sys.executable, "-c", script], check=True)


class SyncClientTests(unittest.TestCase):
    def setUp(self) -> None:
        _Handler.requests.clear()
        _Handler.fail_counts.clear()
        self.client = Client(BASE_URL, token="om_test")

    def test_meter_crud_list_filters_and_pagination(self) -> None:
        request = CreateMeterRequest(
            name="Tokens Total",
            key="tokens_total",
            aggregation="sum",
            event_type="prompt",
        )
        self.assertEqual(self.client.meters.create(request).key, "tokens_total")
        self.assertEqual(self.client.meters.get(METER_ID).id, METER_ID)
        self.assertEqual(
            self.client.meters.update(METER_ID, UpdateMeterRequest(name="Renamed")).name,
            "Renamed",
        )

        params = MeterListParams(
            page=PageParams(number=1, size=1),
            sort=["name", "created_at desc"],
            filter=MeterFilter(
                key=StringFilter(oeq=["tokens_total", "requests_total"]),
                name=StringFilter(exists=True),
            ),
        )
        page = self.client.meters.list(params)
        self.assertEqual(page.meta.page.total, 2)
        self.assertEqual(
            [meter.id for meter in self.client.meters.list_all(params)], [METER_ID, SECOND_METER_ID]
        )
        self.client.meters.delete(METER_ID)

        list_request = next(request for request in _Handler.requests if request["query"])
        self.assertEqual(list_request["query"]["page[size]"], ["1"])
        self.assertEqual(list_request["query"]["sort"], ["name,created_at desc"])
        self.assertEqual(
            list_request["query"]["filter[key][oeq]"],
            ["tokens_total,requests_total"],
        )
        self.assertEqual(list_request["query"]["filter[name][$exists]"], ["true"])
        self.assertEqual(_Handler.requests[0]["headers"]["Authorization"], "Bearer om_test")
        self.assertTrue(
            _Handler.requests[0]["headers"]["User-Agent"].startswith("openmeter-python/")
        )

    def test_meter_query_json_csv_and_stream(self) -> None:
        request = MeterQueryRequest(
            from_=datetime(2026, 7, 14, 9, tzinfo=timezone.utc),
            group_by_dimensions=["model"],
            filters=MeterQueryFilters(dimensions={"model": QueryFilterStringMapItem(eq="gpt-4.1")}),
        )
        result = self.client.meters.query(METER_ID, request)
        self.assertEqual(result.data[0].value, Decimal("12.50"))
        self.assertEqual(
            self.client.meters.query_csv(METER_ID, request).splitlines()[0], b"from,to,value"
        )

        with self.client.meters.query_csv_stream(METER_ID, request) as stream:
            self.assertEqual(stream.readline(), b"from,to,value\n")

        query_body = _Handler.requests[0]["body"]
        self.assertIn("from", query_body)
        self.assertNotIn("from_", query_body)
        self.assertNotIn("time_zone", query_body)

    def test_plan_addon_crud_list_and_pagination(self) -> None:
        create = CreatePlanAddonRequest(
            name="Pro add-on",
            addon=AddonReference(id=ADDON_ID),
            from_plan_phase="default",
        )
        self.assertEqual(self.client.plan_addons.create(PLAN_ID, create).addon.id, ADDON_ID)
        self.assertEqual(self.client.plan_addons.get(PLAN_ID, PLAN_ADDON_ID).id, PLAN_ADDON_ID)
        update = UpsertPlanAddonRequest(name="Renamed", from_plan_phase="default")
        self.assertEqual(
            self.client.plan_addons.update(PLAN_ID, PLAN_ADDON_ID, update).name,
            "Renamed",
        )
        params = PlanAddonListParams(page=PageParams(number=1, size=100))
        self.assertEqual(len(list(self.client.plan_addons.list_all(PLAN_ID, params))), 1)
        self.client.plan_addons.delete(PLAN_ID, PLAN_ADDON_ID)

    def test_api_error_and_invalid_ids(self) -> None:
        with self.assertRaises(APIError) as raised:
            self.client.meters.get(ERROR_ID)
        self.assertEqual(raised.exception.status_code, 404)
        self.assertEqual(raised.exception.instance, "kong:trace:test")

        for resource_id in ("", "invalid/id", "not-a-ulid"):
            with self.subTest(resource_id=resource_id):
                with self.assertRaises(InvalidIDError):
                    self.client.meters.get(resource_id)

    def test_redirect_response_raises_api_error_without_following(self) -> None:
        with self.assertRaises(APIError) as raised:
            self.client.meters.get(REDIRECT_ID)

        self.assertEqual(raised.exception.status_code, 302)
        self.assertEqual(len(_Handler.requests), 1)

    def test_token_falls_back_to_environment(self) -> None:
        with patch.dict(os.environ, {"OPENMETER_TOKEN": "om_env"}):
            client = Client(BASE_URL)
            client.meters.get(METER_ID)

        self.assertEqual(_Handler.requests[-1]["headers"]["Authorization"], "Bearer om_env")

    def test_buffered_success_response_is_bounded(self) -> None:
        opener = Mock()
        opener.open.return_value = io.BytesIO(b"x" * (BUFFER_LIMIT + 1))
        client = Client(BASE_URL, opener=opener)

        with self.assertRaises(ResponseTooLargeError) as raised:
            client.meters.query_csv(METER_ID, MeterQueryRequest())

        self.assertEqual(raised.exception.limit, BUFFER_LIMIT)

    def test_malformed_error_response_is_bounded(self) -> None:
        error_body = b"x" * (ERROR_LIMIT + 1)
        opener = Mock()
        opener.open.side_effect = HTTPError(BASE_URL, 500, "error", {}, io.BytesIO(error_body))
        client = Client(BASE_URL, opener=opener)

        with self.assertRaises(APIError) as raised:
            client.meters.get(METER_ID)

        self.assertEqual(raised.exception.status_code, 500)
        self.assertEqual(len(raised.exception.raw_body), ERROR_LIMIT)
        self.assertTrue(raised.exception.raw_body_truncated)


class AsyncClientTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        _Handler.requests.clear()
        _Handler.fail_counts.clear()

    async def test_async_meter_and_plan_addon_operations(self) -> None:
        async with AsyncClient(BASE_URL, token="om_test", trust_env=False) as client:
            create_meter = CreateMeterRequest(
                name="Tokens Total",
                key="tokens_total",
                aggregation="sum",
                event_type="prompt",
            )
            self.assertEqual((await client.meters.create(create_meter)).key, "tokens_total")
            meter = await client.meters.get(METER_ID)
            self.assertEqual(meter.id, METER_ID)
            self.assertEqual(
                (await client.meters.update(METER_ID, UpdateMeterRequest(name="Renamed"))).name,
                "Renamed",
            )

            meter_ids = [
                meter.id
                async for meter in client.meters.list_all(MeterListParams(page=PageParams(size=1)))
            ]
            self.assertEqual(meter_ids, [METER_ID, SECOND_METER_ID])

            query = MeterQueryRequest()
            self.assertEqual(
                (await client.meters.query(METER_ID, query)).data[0].value, Decimal("12.50")
            )
            self.assertTrue((await client.meters.query_csv(METER_ID, query)).startswith(b"from,to"))
            async with client.meters.query_csv_stream(METER_ID, query) as stream:
                self.assertEqual(
                    await stream.read(),
                    b"from,to,value\n2026-07-14T09:00:00Z,2026-07-14T10:00:00Z,12\n",
                )

            async with client.meters.query_csv_stream(METER_ID, query) as stream:
                self.assertEqual(await stream.read(5), b"from,")
                self.assertEqual(
                    await stream.read(),
                    b"to,value\n2026-07-14T09:00:00Z,2026-07-14T10:00:00Z,12\n",
                )

            async with client.meters.query_csv_stream(METER_ID, query) as stream:
                lines = [line async for line in stream]
                self.assertEqual(
                    lines,
                    [
                        b"from,to,value\n",
                        b"2026-07-14T09:00:00Z,2026-07-14T10:00:00Z,12\n",
                    ],
                )

            async with client.meters.query_csv_stream(METER_ID, query) as stream:
                self.assertEqual(await stream.read(5), b"from,")
                response = stream._response
                self.assertFalse(response.is_closed)
            self.assertTrue(response.is_closed)

            plan_addon = await client.plan_addons.get(PLAN_ID, PLAN_ADDON_ID)
            self.assertEqual(plan_addon.id, PLAN_ADDON_ID)
            create_plan_addon = CreatePlanAddonRequest(
                name="Pro add-on",
                addon=AddonReference(id=ADDON_ID),
                from_plan_phase="default",
            )
            self.assertEqual(
                (await client.plan_addons.create(PLAN_ID, create_plan_addon)).addon.id,
                ADDON_ID,
            )
            update_plan_addon = UpsertPlanAddonRequest(
                name="Renamed",
                from_plan_phase="default",
            )
            self.assertEqual(
                (
                    await client.plan_addons.update(
                        PLAN_ID,
                        PLAN_ADDON_ID,
                        update_plan_addon,
                    )
                ).name,
                "Renamed",
            )
            plan_addons = [item.id async for item in client.plan_addons.list_all(PLAN_ID)]
            self.assertEqual(plan_addons, [PLAN_ADDON_ID])
            await client.plan_addons.delete(PLAN_ID, PLAN_ADDON_ID)
            await client.meters.delete(METER_ID)

    def test_trust_env_uses_http_client_default_and_allows_override(self) -> None:
        with patch("openmeter._transport_async.httpx.AsyncClient") as constructor:
            AsyncClient(BASE_URL)
            constructor.assert_called_once_with(timeout=30.0, trust_env=True)

        with patch("openmeter._transport_async.httpx.AsyncClient") as constructor:
            AsyncClient(BASE_URL, trust_env=False)
            constructor.assert_called_once_with(timeout=30.0, trust_env=False)

    async def test_async_calls_do_not_block_other_tasks(self) -> None:
        response_gate = threading.Event()
        marker: list[str] = []

        async def mark() -> None:
            await asyncio.sleep(0)
            response_gate.set()
            marker.append("ran")

        _Handler.response_gate = response_gate
        try:
            with patch("asyncio.to_thread", side_effect=AssertionError("thread offload used")):
                async with AsyncClient(BASE_URL, trust_env=False) as client:
                    await asyncio.gather(client.meters.get(METER_ID), mark())
        finally:
            _Handler.response_gate = None

        self.assertEqual(marker, ["ran"])

    async def test_injected_http_client_remains_caller_owned(self) -> None:
        def respond(_: httpx.Request) -> httpx.Response:
            return httpx.Response(200, json=_meter())

        http_client = httpx.AsyncClient(transport=httpx.MockTransport(respond))
        try:
            async with AsyncClient(BASE_URL, client=http_client) as client:
                self.assertEqual((await client.meters.get(METER_ID)).id, METER_ID)

            self.assertFalse(http_client.is_closed)
        finally:
            await http_client.aclose()

    async def test_redirect_response_raises_api_error(self) -> None:
        requests: list[httpx.Request] = []

        def redirect(request: httpx.Request) -> httpx.Response:
            requests.append(request)
            if request.url.path.endswith(METER_ID):
                return httpx.Response(
                    302,
                    headers={"location": f"{BASE_URL}/openmeter/meters/{SECOND_METER_ID}"},
                )

            return httpx.Response(200, json=_meter(SECOND_METER_ID))

        http_client = httpx.AsyncClient(
            transport=httpx.MockTransport(redirect),
            follow_redirects=True,
        )
        try:
            async with AsyncClient(BASE_URL, client=http_client) as client:
                with self.assertRaises(APIError) as raised:
                    await client.meters.get(METER_ID)

            self.assertEqual(raised.exception.status_code, 302)
            self.assertEqual(len(requests), 1)
        finally:
            await http_client.aclose()

    async def test_token_falls_back_to_environment(self) -> None:
        with patch.dict(os.environ, {"OPENMETER_TOKEN": "om_env"}):
            async with AsyncClient(BASE_URL, trust_env=False) as client:
                await client.meters.get(METER_ID)

        self.assertEqual(_Handler.requests[-1]["headers"]["Authorization"], "Bearer om_env")

    async def test_buffered_success_response_is_bounded(self) -> None:
        def respond(request: httpx.Request) -> httpx.Response:
            return httpx.Response(200, content=b"x" * (BUFFER_LIMIT + 1), request=request)

        http_client = httpx.AsyncClient(transport=httpx.MockTransport(respond))
        try:
            async with AsyncClient(BASE_URL, client=http_client) as client:
                with self.assertRaises(ResponseTooLargeError) as raised:
                    await client.meters.query_csv(METER_ID, MeterQueryRequest())

            self.assertEqual(raised.exception.limit, BUFFER_LIMIT)
        finally:
            await http_client.aclose()

    async def test_malformed_error_response_is_bounded(self) -> None:
        def respond(request: httpx.Request) -> httpx.Response:
            return httpx.Response(500, content=b"x" * (ERROR_LIMIT + 1), request=request)

        http_client = httpx.AsyncClient(transport=httpx.MockTransport(respond))
        try:
            async with AsyncClient(BASE_URL, client=http_client) as client:
                with self.assertRaises(APIError) as raised:
                    await client.meters.get(METER_ID)

            self.assertEqual(raised.exception.status_code, 500)
            self.assertEqual(len(raised.exception.raw_body), ERROR_LIMIT)
            self.assertTrue(raised.exception.raw_body_truncated)
        finally:
            await http_client.aclose()


class RetryTests(unittest.TestCase):
    def setUp(self) -> None:
        _Handler.requests.clear()
        _Handler.fail_counts.clear()
        self.client = Client(BASE_URL, token="om_test")
        self.meter_path = f"/openmeter/meters/{METER_ID}"

    def test_idempotent_get_retries_transient_5xx_then_succeeds(self) -> None:
        _Handler.fail_counts[self.meter_path] = 2
        with patch("openmeter._transport._RETRY_BACKOFF_SECONDS", 0.001):
            meter = self.client.meters.get(METER_ID)

        self.assertEqual(meter.id, METER_ID)
        self.assertEqual(len(_Handler.requests), 3)

    def test_idempotent_get_gives_up_after_max_retries(self) -> None:
        _Handler.fail_counts[self.meter_path] = 10
        with patch("openmeter._transport._RETRY_BACKOFF_SECONDS", 0.001):
            with self.assertRaises(APIError) as raised:
                self.client.meters.get(METER_ID)

        self.assertEqual(raised.exception.status_code, 503)
        self.assertEqual(len(_Handler.requests), 3)

    def test_non_idempotent_post_is_not_retried_on_5xx(self) -> None:
        _Handler.fail_counts["/openmeter/meters"] = 5
        request = CreateMeterRequest(
            name="Tokens Total", key="tokens_total", aggregation="sum", event_type="prompt"
        )
        with patch("openmeter._transport._RETRY_BACKOFF_SECONDS", 0.001):
            with self.assertRaises(APIError) as raised:
                self.client.meters.create(request)

        self.assertEqual(raised.exception.status_code, 503)
        self.assertEqual(len(_Handler.requests), 1)

    def test_read_only_post_query_retries_transient_5xx_then_succeeds(self) -> None:
        _Handler.fail_counts[f"{self.meter_path}/query"] = 2
        with patch("openmeter._transport._RETRY_BACKOFF_SECONDS", 0.001):
            result = self.client.meters.query(METER_ID, MeterQueryRequest())

        self.assertEqual(result.data[0].value, Decimal("12.50"))
        self.assertEqual(len(_Handler.requests), 3)

    def test_idempotent_get_retries_connection_failures(self) -> None:
        opener = Mock()
        opener.open.side_effect = [
            URLError("temporarily unavailable"),
            URLError("temporarily unavailable"),
            io.BytesIO(json.dumps(_meter()).encode()),
        ]
        client = Client(BASE_URL, opener=opener)

        with patch("openmeter._transport._RETRY_BACKOFF_SECONDS", 0.001):
            meter = client.meters.get(METER_ID)

        self.assertEqual(meter.id, METER_ID)
        self.assertEqual(opener.open.call_count, 3)

    def test_non_idempotent_post_does_not_retry_connection_failure(self) -> None:
        opener = Mock()
        opener.open.side_effect = URLError("temporarily unavailable")
        client = Client(BASE_URL, opener=opener)
        request = CreateMeterRequest(
            name="Tokens Total", key="tokens_total", aggregation="sum", event_type="prompt"
        )

        with self.assertRaises(TransportError):
            client.meters.create(request)

        self.assertEqual(opener.open.call_count, 1)


class AsyncRetryTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self) -> None:
        _Handler.requests.clear()
        _Handler.fail_counts.clear()
        self.meter_path = f"/openmeter/meters/{METER_ID}"

    async def test_idempotent_get_retries_transient_5xx_then_succeeds(self) -> None:
        _Handler.fail_counts[self.meter_path] = 2
        with patch("openmeter._transport_async._RETRY_BACKOFF_SECONDS", 0.001):
            async with AsyncClient(BASE_URL, token="om_test", trust_env=False) as client:
                meter = await client.meters.get(METER_ID)

        self.assertEqual(meter.id, METER_ID)
        self.assertEqual(len(_Handler.requests), 3)

    async def test_idempotent_get_gives_up_after_max_retries(self) -> None:
        _Handler.fail_counts[self.meter_path] = 10
        with patch("openmeter._transport_async._RETRY_BACKOFF_SECONDS", 0.001):
            async with AsyncClient(BASE_URL, token="om_test", trust_env=False) as client:
                with self.assertRaises(APIError) as raised:
                    await client.meters.get(METER_ID)

        self.assertEqual(raised.exception.status_code, 503)
        self.assertEqual(len(_Handler.requests), 3)

    async def test_non_idempotent_post_is_not_retried_on_5xx(self) -> None:
        _Handler.fail_counts["/openmeter/meters"] = 5
        request = CreateMeterRequest(
            name="Tokens Total", key="tokens_total", aggregation="sum", event_type="prompt"
        )
        with patch("openmeter._transport_async._RETRY_BACKOFF_SECONDS", 0.001):
            async with AsyncClient(BASE_URL, token="om_test", trust_env=False) as client:
                with self.assertRaises(APIError) as raised:
                    await client.meters.create(request)

        self.assertEqual(raised.exception.status_code, 503)
        self.assertEqual(len(_Handler.requests), 1)

    async def test_read_only_post_query_retries_transient_5xx_then_succeeds(self) -> None:
        _Handler.fail_counts[f"{self.meter_path}/query"] = 2
        with patch("openmeter._transport_async._RETRY_BACKOFF_SECONDS", 0.001):
            async with AsyncClient(BASE_URL, token="om_test", trust_env=False) as client:
                result = await client.meters.query(METER_ID, MeterQueryRequest())

        self.assertEqual(result.data[0].value, Decimal("12.50"))
        self.assertEqual(len(_Handler.requests), 3)

    async def test_idempotent_get_retries_connection_failures(self) -> None:
        attempts = 0

        def respond(request: httpx.Request) -> httpx.Response:
            nonlocal attempts
            attempts += 1
            if attempts < 3:
                raise httpx.ConnectError("temporarily unavailable", request=request)

            return httpx.Response(200, json=_meter(), request=request)

        http_client = httpx.AsyncClient(transport=httpx.MockTransport(respond))
        try:
            with patch("openmeter._transport_async._RETRY_BACKOFF_SECONDS", 0.001):
                async with AsyncClient(BASE_URL, client=http_client) as client:
                    meter = await client.meters.get(METER_ID)

            self.assertEqual(meter.id, METER_ID)
            self.assertEqual(attempts, 3)
        finally:
            await http_client.aclose()

    async def test_non_idempotent_post_does_not_retry_connection_failure(self) -> None:
        attempts = 0

        def fail(request: httpx.Request) -> httpx.Response:
            nonlocal attempts
            attempts += 1
            raise httpx.ConnectError("temporarily unavailable", request=request)

        http_client = httpx.AsyncClient(transport=httpx.MockTransport(fail))
        request = CreateMeterRequest(
            name="Tokens Total", key="tokens_total", aggregation="sum", event_type="prompt"
        )
        try:
            async with AsyncClient(BASE_URL, client=http_client) as client:
                with self.assertRaises(TransportError):
                    await client.meters.create(request)

            self.assertEqual(attempts, 1)
        finally:
            await http_client.aclose()


if __name__ == "__main__":
    unittest.main()
