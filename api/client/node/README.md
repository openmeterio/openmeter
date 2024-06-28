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

#### meter subjects

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

### Subject

Subject mappings. Like display name and metadata.

#### upsert

Upsert subjects.

```ts
const subjects = await openmeter.subjects.upsert([
  {
    key: 'customer-1',
    displayName: 'ACME',
  },
])
```

#### list

List subjects.

```ts
const subjects = await openmeter.subjects.list()
```

#### get

Get subject by key.

```ts
const subjects = await openmeter.subjects.get('customer-1')
```

#### delete

Delete subject by key.
It doesn't delete corresponding usage.

```ts
await openmeter.subjects.delete('customer-1')
```

#### createEntitlement

Create entitlement for a subject.
Entitlements allow you to manage subject feature access, balances, and usage limits.

```ts
// Issue 10,000,000 tokens every month
const entitlement = await openmeter.subjects.createEntitlement('customer-1', {
  type: 'metered',
  featureKey: 'ai_tokens',
  usagePeriod: {
    interval: 'MONTH',
  },
  issueAfterReset: 10000000,
})
```

#### listEntitlements

List subject entitlements.

```ts
const entitlement = await openmeter.subjects.listEntitlements('customer-1')
```

#### getEntitlement

Get a subject entitlement by ID by Feature ID or by Feature Key.

```ts
const entitlement = await openmeter.subjects.getEntitlement(
  'customer-1',
  'ai_tokens'
)
```

#### deleteEntitlement

Delete a subject entitlement by ID by Feature ID or by Feature Key.

```ts
await openmeter.subjects.deleteEntitlement('customer-1', 'ai_tokens')
```

#### getEntitlementValue

Get entitlement value by ID by Feature ID or by Feature Key.

```ts
const value = await openmeter.subjects.getEntitlementValue(
  'customer-1',
  'ai_tokens'
)
```

#### getEntitlementHistory

Get entitlement history by ID by Feature ID or by Feature Key

```ts
const entitlement = await openmeter.subjects.getEntitlementHistory(
  'customer-1',
  'ai_tokens'
)
```

#### resetEntitlementUsage

Reset the entitlement usage and start a new period. Eligible grants will be rolled over.

```ts
const entitlement = await openmeter.subjects.resetEntitlementUsage(
  'customer-1',
  {
    retainAnchor: true,
  }
)
```

## Features

Features are the building blocks of your entitlements, part of your product offering.

#### create

Upsert subjects.

```ts
const feature = await openmeter.features.create({
  key: 'ai_tokens',
  name: 'AI Tokens',
  // optional
  meterSlug: 'tokens_total',
})
```

#### list

List features.

```ts
const features = await openmeter.features.list()
```

#### get

Get feature by key.

```ts
const feature = await openmeter.features.get('ai_tokens')
```

#### delete

Delete feature by key.

```ts
await openmeter.features.delete('ai_tokens')
```
