# OpenMeter JavaScript SDK

## Install

```sh
npm install --save @openmeter/sdk
```

## Configuration for accessing the OpenMeter API

To use the OpenMeter SDK on your backend, you need to configure `baseUrl` and `apiKey` for OpenMeter Cloud:

```ts
import { OpenMeter } from '@openmeter/sdk'

const openmeter = new OpenMeter({
  baseUrl: 'https://openmeter.cloud',
  apiKey: 'om_...',
})
```

## Configuration for accessing the OpenMeter Portal API

To use the OpenMeter Portal SDK on your frontend, you need to configure it use a portal token in your configuration:

```ts
import { OpenMeter } from '@openmeter/sdk/portal'

const openmeter = new OpenMeter({
  baseUrl: 'https://openmeter.cloud',
  portalToken: 'om_portal_...',
})
```

## Configuration for accessing the OpenMeter React SDK

To use the OpenMeter React SDK for the portal API, you need to configure a Portal Client and a React Context:

```ts
import {
  OpenMeter,
  OpenMeterProvider,
  useOpenMeter,
} from '@openmeter/sdk/react'

function App() {
  // get portal token from your backend
  const openmeter = new OpenMeter({
    baseUrl: 'https://openmeter.cloud',
    portalToken,
  })

  return (
    <OpenMeterProvider value={openmeter}>
      <UsageComponent />
      {/* ... */}
    </OpenMeterProvider>
  )
}

function UsageComponent() {
  // get openmeter client from context
  const openmeter = useOpenMeter()

  // ...
}
```
