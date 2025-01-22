# OpenMeter Node SDK

## Install

```sh
npm install --save @openmeter/sdk
```

## Configuration

To use the OpenMeter SDK, you need to configure it the `baseUrl` and `apiKey` for OpenMeter Cloud:

```ts
import { OpenMeter } from '@openmeter/sdk'

const openmeter = new OpenMeter({
  baseUrl: 'https://openmeter.cloud',
  apiKey: 'your-api-key',
})
```
