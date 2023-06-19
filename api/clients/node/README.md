# @openmeter/sdk

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
 time: new Date().toISOString(),
 data: {
  api_calls: 1,
 },
}
await openmeter.ingestEvents(event)

// Fetching a meter
const meter = await openmeter.getMetersById('m1')
```

## API

The OpenMeter SDK uses [openapi-typescript-codegen](https://www.npmjs.com/package/openapi-typescript-codegen) under the hood to generate the HTTP client.

### ingestEvents

```js
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
await openmeter.ingestEvents(event)
```

### getMeters

```js
const meters = await openmeter.getMeters()
```

### getMetersById

```js
const meter = await openmeter.getMetersById('m1')
```

### getValuesByMeterId

```js
const meterId = 'm2'
const subject = 'user-1'
const from = new Date('2021-01-01').toISOString()
const to = new Date('2021-01-02').toISOString()
const windowSize = WindowSize.HOUR
const values = await openmeter.getValuesByMeterId(meterId, subject, from, to, windowSize)
```
