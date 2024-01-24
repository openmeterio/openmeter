# OpenMeter Node SDK

## Install

```sh
npm install --save @openmeter/sdk
```

## Example

```ts
import { OpenMeter, type Event } from '@openmeter/sdk'

const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })

// Ingesting an event
const event: Event = {
  specversion: '1.0',
  id: 'id-1',
  source: 'my-app',
  type: 'my-type',
  subject: 'my-awesome-user-id',
  time: new Date(),
  data: {
    api_calls: 1,
  },
}
await openmeter.events.ingest(event)

// Fetching a meter
const meter = await openmeter.meters.get('m1')
```

## API

### Events

#### ingest

```ts
import { type Event } from '@openmeter/sdk'

const event: Event = {
  specversion: '1.0',
  id: 'id-1',
  source: 'my-app',
  type: 'my-type',
  subject: 'my-awesome-user-id',
  time: new Date(),
  data: {
    api_calls: 1,
  },
}
await openmeter.events.ingest(event)
```

### batch ingest

```ts
await openmeter.events.ingest([event1, event2, event3])
```

#### list

Retrieve latest raw events. Useful for debugging.

```ts
const events = await openmeter.events.list()
```

### Meters

#### list

List meters.

```ts
const meters = await openmeter.meters.list()
```

#### get

Get one meter by slug.

```ts
const meter = await openmeter.meters.get('m1')
```

#### query

Query meter values.

```ts
import { WindowSize } from '@openmeter/sdk'

const values = await openmeter.meters.query('my-meter-slug', {
  subject: ['user-1'],
  groupBy: ['method', 'path'],
  from: new Date('2021-01-01'),
  to: new Date('2021-01-02'),
  windowSize: WindowSize.HOUR,
})
```

#### subjects

List meter subjects.

```ts
const subjects = await openmeter.meters.subjects('my-meter-slug')
```

### Portal

#### createToken

Create subject specific tokens.
Useful to build consumer dashboards.

```ts
const token = await openmeter.portal.createToken({ subject: 'customer-1' })
```

#### invalidateTokens

Invalidate portal tokens for all or specific subjects.

```ts
await openmeter.portal.invalidateTokens()
```
