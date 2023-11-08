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

## Helpers

### Vercel AI SDK / Next.js

The OpenAI streaming API used by the Vercel AI SDK doesn't return token usage metadata by default.
The OpenMeter `createOpenAIStreamCallback` helper function decorates the callback with a `onUsage`
callback which you can use to report usage to OpenMeter.

```ts
import OpenAI from 'openai'
import { OpenAIStream, StreamingTextResponse } from 'ai'
import { createOpenAIStreamCallback } from '@openmeter/sdk'

export async function POST(req: Request) {
  const { messages } = await req.json()
  const model = 'gpt-3.5-turbo'

  const response = await openai.chat.completions.create({
    model,
    messages,
    stream: true,
  })

  const streamCallbacks = await createOpenAIStreamCallback(
    {
      model,
      prompts: messages.map(({ content }) => content),
    },
    {
      // onToken() => {...}
      // onFinal() => {...}
      async onUsage(usage) {
        try {
          await openmeter.events.ingest({
            source: 'my-app',
            type: 'my-event-type',
            subject: 'my-customer-id',
            data: {
              // Usage is { total_tokens, prompt_tokens, completion_tokens }
              ...usage,
              model,
            },
          })
        } catch (err) {
          console.error('failed to ingest usage', err)
        }
      },
    }
  )

  const stream = OpenAIStream(response, streamCallbacks)
  return new StreamingTextResponse(stream)
}
```
