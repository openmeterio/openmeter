# OpenMeter OSS quickstart

Run a complete local OpenMeter stack, send a usage event, and query its metered
value.

## Prerequisites

- [Docker with Compose](https://docs.docker.com/compose/)
- [Git](https://git-scm.com/)
- one of:
  - Bash and `curl`
  - Node.js 22+ and npm for TypeScript
  - Python 3.9+ and pip for Python

## 1. Start OpenMeter

```sh
git clone https://github.com/openmeterio/openmeter.git
cd openmeter/quickstart
docker compose up -d --wait
```

The API is available at `http://localhost:48888`. The Compose stack also starts
OpenMeter's workers and local Kafka, ClickHouse, PostgreSQL, Redis, and Svix
dependencies.

> [!NOTE]
> This setup uses `latest` OpenMeter images and development-grade dependencies.
> It is intended for local evaluation, not production.

## 2. Meter an event

The included configuration defines an `api_requests_total` meter that counts
events with type `request`. Choose a client; each example sends one event and
polls until its asynchronous processing completes.

<details open>
<summary><strong>Bash (curl)</strong></summary>

```sh
(
  status=$(curl -sS -o /dev/null -w '%{http_code}' \
    -X POST http://localhost:48888/api/v1/events \
    -H 'Content-Type: application/cloudevents+json' \
    --data-raw '{
      "specversion": "1.0",
      "type": "request",
      "id": "quickstart-curl-1",
      "source": "quickstart",
      "subject": "quickstart-curl",
      "data": { "method": "GET", "route": "/hello" }
    }') || exit 1

  if [ "$status" != 204 ]; then
    printf 'event ingestion failed (HTTP %s)\n' "$status" >&2
    exit 1
  fi
  printf '%s\n' "$status"

  for attempt in 1 2 3 4 5 6 7 8 9 10; do
    response=$(curl -fsS 'http://localhost:48888/api/v1/meters/api_requests_total/query?subject=quickstart-curl') || exit 1
    if printf '%s' "$response" | grep -Eq '"value"[[:space:]]*:[[:space:]]*1[[:space:]]*[,}]'; then
      printf '%s\n' "$response"
      exit 0
    fi
    sleep 1
  done

  printf '%s\n' 'usage was not processed in time' >&2
  exit 1
)
```

The ingest prints `204`; the query response contains
`"subject":"quickstart-curl"` and `"value":1`.

</details>

<details>
<summary><strong>TypeScript SDK</strong></summary>

Install the [OpenMeter TypeScript SDK](https://www.npmjs.com/package/@openmeter/sdk):

```sh
npm install @openmeter/sdk tsx
```

Save as `quickstart.ts`, then run `npx tsx quickstart.ts`:

```ts
import { OpenMeter } from '@openmeter/sdk'

const openmeter = new OpenMeter({ baseUrl: 'http://localhost:48888' })

const queryUsage = () =>
  openmeter.meters.query('api_requests_total', {
    subject: ['quickstart-typescript'],
  })

async function main() {
  await openmeter.events.ingest({
    type: 'request',
    id: 'quickstart-typescript-1',
    source: 'quickstart',
    subject: 'quickstart-typescript',
    data: { method: 'GET', route: '/hello' },
  })

  let usage = await queryUsage()
  for (let attempt = 1; usage.data[0]?.value !== 1 && attempt < 10; attempt++) {
    await new Promise((resolve) => setTimeout(resolve, 1000))
    usage = await queryUsage()
  }

  if (usage.data[0]?.value !== 1) {
    throw new Error('usage was not processed in time')
  }
  console.log(usage.data[0].value)
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
```

The script prints `1`.

</details>

<details>
<summary><strong>Python SDK</strong></summary>

The Python SDK is in preview, so install it with pre-releases enabled:

```sh
python -m pip install --pre openmeter
```

Save as `quickstart.py`, then run `python quickstart.py`:

```python
import time

from openmeter import Client
from openmeter.models import Event

with Client(endpoint="http://localhost:48888") as openmeter:
    openmeter.events.ingest_event(
        Event(
            id="quickstart-python-1",
            source="quickstart",
            specversion="1.0",
            type="request",
            subject="quickstart-python",
            data={"method": "GET", "route": "/hello"},
        )
    )

    for _ in range(10):
        usage = openmeter.meters.query_json(
            "api_requests_total",
            subject=["quickstart-python"],
        )
        if usage.data and usage.data[0].value == 1:
            break
        time.sleep(1)
    else:
        raise RuntimeError("usage was not processed in time")

    print(usage.data[0].value)
```

The script prints `1.0`.

</details>

## 3. Explore

Group usage by hour, method, and route:

```sh
curl 'http://localhost:48888/api/v1/meters/api_requests_total/query?windowSize=HOUR&groupBy=method&groupBy=route'
```

Then continue with:

- how [meters and customer attribution](https://openmeter.io/docs/metering/overview) work
- common [AI, API, and compute metering examples](https://openmeter.io/docs/metering/guides/common-examples)
- [entitlements and usage limits](https://openmeter.io/docs/billing/entitlements/quickstart)
- [plans, pricing, and subscriptions](https://openmeter.io/docs/product-catalog/overview)
- the [OSS API reference](https://openmeter.io/docs/api/open-source)
- the [Kubernetes deployment guide](https://openmeter.io/docs/open-source/kubernetes)

## Cleanup

Remove the quickstart containers and their data volumes:

```sh
docker compose down -v
```
