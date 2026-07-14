# OpenMeter Python SDK (v3 baseline)

An idiomatic, hand-written Python SDK for the OpenMeter v3 API. This baseline
covers the same endpoint groups as the Go SDK baseline in PR #4644: meters and
plan add-ons. It is intentionally not a complete OpenMeter client.

## Requirements

- Python 3.9 or newer
- Pydantic v2
- httpx (async client only)

The sync `Client` uses only the Python standard library (`urllib`) for HTTP.
The async `AsyncClient` uses `httpx.AsyncClient` for true async I/O — no
thread pool involved. Pydantic and httpx are the only runtime dependencies.

## Install

```bash
python -m pip install ./sdk/python/openmeter
```

## Client setup

The base URL must include the deployment's v3 prefix:

| Deployment | Base URL |
| --- | --- |
| Local | `http://127.0.0.1:8888/api/v3` |
| OpenMeter Cloud | `https://openmeter.cloud/api/v3` |
| Kong Konnect | `https://<region>.api.konghq.com/v3` |

```python
from openmeter import Client

with Client(
    "https://openmeter.cloud/api/v3",
    token="om_...",
) as client:
    page = client.meters.list()
    print(page.data)
```

The async client has the same resource groups and method names:

```python
import asyncio

from openmeter import AsyncClient


async def main() -> None:
    async with AsyncClient(
        "https://openmeter.cloud/api/v3",
        token="om_...",
    ) as client:
        async for meter in client.meters.list_all():
            print(meter.key)


asyncio.run(main())
```

Complete runnable snippets live in [`examples/`](examples/). Public classes and
methods include docstrings, so `help(openmeter.Client)` and editor tooltips also
serve as API documentation.

## Pydantic models

Request models validate v3 constraints before sending a request:

```python
from openmeter import CreateMeterRequest

request = CreateMeterRequest(
    name="Tokens Total",
    key="tokens_total",
    aggregation="sum",
    event_type="prompt",
    value_property="$.tokens",
)
```

Unknown response fields are ignored to preserve additive API compatibility.
Response enum-like fields remain plain strings, allowing a newer server value to
round-trip through an older SDK. Request enum values are validated against the
current v3 contract.

## Implemented endpoints

| Resource method | HTTP endpoint |
| --- | --- |
| `meters.create` / `meters.list` / `meters.list_all` | `POST`, `GET /openmeter/meters` |
| `meters.get` / `meters.update` / `meters.delete` | `GET`, `PUT`, `DELETE /openmeter/meters/{meterId}` |
| `meters.query` / `meters.query_csv` / `meters.query_csv_stream` | `POST /openmeter/meters/{meterId}/query` |
| `plan_addons.create` / `plan_addons.list` / `plan_addons.list_all` | `POST`, `GET /openmeter/plans/{planId}/addons` |
| `plan_addons.get` / `plan_addons.update` / `plan_addons.delete` | `GET`, `PUT`, `DELETE /openmeter/plans/{planId}/addons/{planAddonId}` |

`list_all` is a lazy iterator on the sync client and an async iterator on the
async client. Buffered JSON and CSV bodies are capped at 10 MiB. Use
`query_csv_stream` for large CSV exports and close it with `with` or `async with`.
Plan-add-on operations and meter update/query operations are marked unstable in
the current v3 specification and can evolve before they become stable.

## Errors and transport

Non-2xx responses raise `APIError`, exposing `status_code`, `title`, `detail`,
`type`, `instance`, parsed invalid parameters, and the capped raw response body.
Transport failures raise `TransportError`; client-side model failures use
Pydantic's `ValidationError`.

The sync client accepts an optional `urllib.request.OpenerDirector` to customize
proxy, TLS, and authentication behavior without adding an HTTP dependency. The
async client accepts an optional preconfigured `httpx.AsyncClient` for the same
purpose; by default it sets `trust_env=False`, ignoring system proxy
environment variables (`HTTP_PROXY`, `NO_PROXY`, etc.), since httpx cannot
parse some `NO_PROXY` formats (IPv6 addresses, CIDR ranges) and will crash on
construction rather than ignore them — pass your own `httpx.AsyncClient` to
opt back into proxy env vars. Pass `timeout=None` only when the caller
supplies another transport-level bound.

GET, PUT, and DELETE requests (idempotent) are retried up to twice with
exponential backoff on connection failures or a 502/503/504 response. POST
(create) is never retried, since a retried create could duplicate the
resource or its notification/webhook side effects.

## Development

The package uses [uv](https://docs.astral.sh/uv/) for the contributor environment
and lockfile. From `sdk/python/openmeter`, install the development dependencies
with the supported Python floor:

```bash
uv sync --python 3.9
```

Run linting, formatting, type checking, tests, and the package build through the
locked environment:

```bash
uv run flake8 src tests examples
uv run ruff check .
uv run ruff format --check .
uv run pyright
uv run python -m unittest discover -s tests -v
uv build
```

Tests use `unittest` and a local standard-library HTTP server. Package consumers
do not need uv and can continue installing the SDK with pip or another Python
package manager.
