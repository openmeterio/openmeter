# @openmeter/sdk

## Install

```sh
npm install --save @openmeter/sdk
```

## Example

```js
import openmeter from '@openmeter/sdk'

const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })

// Ingesting an event
await openmeter.ingestEvents({
	specversion: '1.0',
	id: 'id-1',
	source: 'my-app',
	type: 'my-type',
	subject: 'my-awesome-user-id',
	time: new Date().toISOString(),
	data: {
		api_calls: 1,
	},
})

// Fetching a meter
const response = await openmeter.getMetersById({ meterId: 'm1' })
const meter = await response.json()
```

## API

The OpenMeter SDK uses the `fetch` API.
You can pass a custom `fetch` implementation to the constructor and extend the request params per method call, for example:

```js
import nodeFetch from 'node-fetch'
import openmeter from '@openmeter/sdk'

const openmeter = new OpenMeter({
	baseUrl: 'http://localhost:8888',
	fetch: nodeFetch,
})

await openmeter.getMetersById(
	{ meterId: 'm1' },
	{ headers: { 'x-foo': 'bar' } }
)
```

### ingestEvents

```js
const { error } = await openmeter.ingestEvents({
	specversion: '1.0',
	id: 'id-1',
	source: 'my-app',
	type: 'my-type',
	subject: 'my-awesome-user-id',
	time: new Date().toISOString(),
	data: {
		api_calls: 1,
	},
})
```

### getMeters

```js
const { data, error } = await openmeter.getMeters()
```

### getMetersById

```js
const { data, error } = await openmeter.getMetersById({ meterId: 'm1' })
```

### getValuesByMeterId

```js
const { data, error } = await openmeter.getValuesByMeterId(
	{ meterId: 'm1' },
	{
		subject: 'my-ubject',
		windowSize: 'HOUR',
		from: new Date('2023-01-01'),
		to: new Date('2023-02-01'),
	}
)
```
