# async

<!-- archie:ai-start -->

> Runnable async Python examples demonstrating openmeter SDK usage patterns against a live endpoint; each file is a self-contained script that exercises one domain (customers, entitlements, ingest, meter queries, subscriptions) using the async client from openmeter.aio.

## Patterns

**Async context manager client lifetime** — Always instantiate the client as `async with Client(endpoint=..., token=...) as client:` so the underlying HTTP session is properly closed. (`async with Client(endpoint=ENDPOINT, token=token) as client: ...`)
**Environment-variable configuration with defaults** — All connection and domain parameters are read from os.environ with a fallback literal; never hardcode credentials. (`ENDPOINT: str = environ.get('OPENMETER_ENDPOINT') or 'https://openmeter.cloud'`)
**HttpResponseError catch-all** — Every main() wraps all client calls in a single try/except HttpResponseError block imported from corehttp.exceptions. (`except HttpResponseError as e: print(f'Error: {e}')`)
**asyncio.run entry point** — Scripts call asyncio.run(main()) at module level; no if __name__ == '__main__' guard is used. (`asyncio.run(main())`)
**Dict-access for discriminated-union items** — When SDK deserialises discriminated unions (e.g. entitlements list) items may come back as plain dicts; use .get() not attribute access. (`entitlement_type = entitlement.get('type')`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ingest.py` | Shows CloudEvents ingestion: construct Event with uuid id, UTC time, specversion='1.0', then await client.events.ingest_event(event). | time must be timezone-aware (datetime.timezone.utc); omitting tzinfo silently produces a naive datetime. |
| `entitlement.py` | Demonstrates three entitlement sub-types (metered/static/boolean) via both get_customer_entitlement_value and customer_entitlements_v2.list. | items_property list returns dicts, not typed objects — attribute access will raise AttributeError. |
| `query.py` | Meter query examples: plain total, group_by list, and advanced_meter_group_by_filters with FilterString(eq=...). | r.data may be empty; always guard len(r.data) > 0 before indexing r.data[0]. |
| `subscription.py` | Full subscription lifecycle: create via PlanSubscriptionCreate, get_expanded, list_customer_subscriptions with status filter. | list_customer_subscriptions result uses items_property not items. |

## Anti-Patterns

- Using `from openmeter import Client` (sync) instead of `from openmeter.aio import Client` in async examples
- Calling asyncio.run() inside an already-running event loop
- Accessing entitlement list items with attribute syntax instead of .get() when the SDK returns dicts
- Hardcoding endpoint or token instead of reading from environment variables

## Decisions

- **One file per domain, not a single monolithic demo** — Keeps each example runnable in isolation; developers can copy a single file without unrelated dependencies.
- **async with for client lifetime instead of explicit close()** — Guarantees HTTP session teardown even when exceptions are raised inside main().

## Example: Ingest a CloudEvent asynchronously

```
import asyncio, datetime, uuid
from openmeter.aio import Client
from openmeter.models import Event
from corehttp.exceptions import HttpResponseError

async def main() -> None:
    async with Client(endpoint=ENDPOINT, token=token) as client:
        try:
            event = Event(
                id=str(uuid.uuid4()),
                source="my-app",
                specversion="1.0",
                type="prompt",
                subject="customer-1",
                time=datetime.datetime.now(datetime.timezone.utc),
// ...
```

<!-- archie:ai-end -->
