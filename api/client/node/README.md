# OpenMeter Node SDK

## Install

```sh
npm install --save @openmeter/sdk@alpha
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
 time: new Date().toISOString(),
 data: {
 api_calls: 1,
 },
}
await openmeter.events.ingestEvents(event)

// Fetching a meter
const meter = await openmeter.meters.getMeter('m1')
```

## API

The OpenMeter SDK uses [openapi-typescript-codegen](https://www.npmjs.com/package/openapi-typescript-codegen) under the hood to generate the HTTP client.

### Events

#### ingestEvents

```ts
import { type Event } from '@openmeter/sdk'

const event: Event = {
 specversion: '1.0',
 id: 'id-1',
 source: 'my-app',
 type: 'my-type',
 subject: 'my-awesome-user-id',
 time: new Date().toISOString(),
 data: {
  api_calls: 1,
 },
}
await openmeter.events.ingestEvents(event)
```

### Meters

#### listMeters

```ts
const meters = await openmeter.meters.listMeters()
```

#### getMeter

```ts
const meter = await openmeter.meters.getMeter('m1')
```

#### getMeterValues

```ts
import { type WindowSize } from '@openmeter/sdk'

const meterSlug = 'm2'
const namespace = undefined
const subject = 'user-1'
const from = new Date('2021-01-01').toISOString()
const to = new Date('2021-01-02').toISOString()
const windowSize = WindowSize.HOUR
const values = await openmeter.meters.getMeterValues(
 meterSlug,
 namespace,
 subject,
 from,
 to,
 windowSize
)
```
