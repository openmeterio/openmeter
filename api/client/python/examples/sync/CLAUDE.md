# sync

<!-- archie:ai-start -->

> Runnable synchronous Python examples mirroring the async/ folder; each script exercises one domain using the blocking `openmeter.Client` and is structurally identical to its async counterpart except for the absence of async/await.

## Patterns

**Module-level client instantiation** — The sync Client is instantiated at module level (outside any function), not inside a context manager, because there is no async session to manage. (`client = Client(endpoint=ENDPOINT, token=token)`)
**Environment-variable configuration with defaults** — All connection and domain parameters are read from os.environ with a fallback literal. (`ENDPOINT: str = environ.get('OPENMETER_ENDPOINT') or 'https://openmeter.cloud'`)
**HttpResponseError catch-all** — Every main() wraps all client calls in a single try/except HttpResponseError imported from corehttp.exceptions. (`except HttpResponseError as e: print(f'Error: {e}')`)
**Direct main() call at module level** — Scripts call main() at module level with no asyncio.run() and no if __name__ == '__main__' guard. (`main()`)
**Dict-access for discriminated-union items** — Entitlement list items come back as plain dicts; use .get() not attribute access. (`entitlement_type = entitlement.get('type')`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ingest.py` | Blocking CloudEvents ingestion: client.events.ingest_event(event) with timezone-aware datetime. | time must be timezone-aware; client is module-level, not a context manager. |
| `entitlement.py` | Same three-type entitlement demo as async counterpart using client.customer_entitlements_v2.list and client.customer_entitlement.get_customer_entitlement_value. | items_property returns dicts; attribute access on list items raises AttributeError. |
| `query.py` | Meter queries including FilterString(eq=...) for advanced_meter_group_by_filters. | r.data may be empty; guard before indexing. |
| `subscription.py` | Subscription lifecycle using client.subscriptions and client.customers.list_customer_subscriptions. | list result pagination uses items_property not items. |

## Anti-Patterns

- Using `from openmeter.aio import Client` (async) in sync examples
- Wrapping the sync client in `async with` or calling asyncio.run()
- Accessing entitlement list items with attribute syntax instead of .get()
- Hardcoding endpoint or token instead of reading from environment variables

## Decisions

- **Module-level client construction instead of per-call instantiation** — Connection setup is done once; repeated calls reuse the same underlying HTTP connection pool.
- **Structural parity with async/ folder** — Developers can compare sync and async examples side-by-side to understand the only difference: import path and absence of await/asyncio.run.

## Example: Ingest a CloudEvent synchronously

```
import datetime, uuid
from openmeter import Client
from openmeter.models import Event
from corehttp.exceptions import HttpResponseError

client = Client(endpoint=ENDPOINT, token=token)

def main() -> None:
    try:
        event = Event(
            id=str(uuid.uuid4()),
            source="my-app",
            specversion="1.0",
            type="prompt",
            subject="customer-1",
// ...
```

<!-- archie:ai-end -->
