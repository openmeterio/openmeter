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
const { data } = await openmeter.getMetersById({ meterId: 'm1' })
```

## API

The OpenMeter SDK uses [openapi-fetch](https://github.com/drwpow/openapi-typescript/tree/main/packages/openapi-fetch).
You can pass a custom `fetch` implementation to the constructor and extend the request params per method call, for example:

```js
import nodeFetch from 'node-fetch'
import openmeter from '@openmeter/sdk'

const openmeter = new OpenMeter({
	baseUrl: 'http://localhost:8888',
	fetch: nodeFetch,
	// ...fetch options see: https://developer.mozilla.org/en-US/docs/Web/API/fetch#options
})

const { data } = await openmeter.getMetersById(
	{ meterId: 'm1' },
	{ headers: { 'x-foo': 'bar' } }
)
```

### Response

All methods return an object with **data**, **error**, and **response**.

- **data** will contain that endpoint’s `2xx` response if the server returned `2xx`;
- **response** has response info like `status`, `headers`, etc. It is not typechecked.

For non `2xx` a `HttpError` will be thrown.

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
const { data } = await openmeter.getMeters()
```

### getMetersById

```js
const { data } = await openmeter.getMetersById({ meterId: 'm1' })
```

### getValuesByMeterId

```js
const { data } = await openmeter.getValuesByMeterId(
	{ meterId: 'm1' },
	{
		subject: 'my-ubject',
		windowSize: 'HOUR',
		from: new Date('2023-01-01'),
		to: new Date('2023-02-01'),
	}
)
```
