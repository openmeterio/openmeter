# OpenAI Example

In this example, we will track our customers' OpenAI token consumption.
This is useful if you are building on top of the OpenAI API, like ChatGPT,
and want to meter your customers' usage for reporting or billing purposes.

Language: `Node.js`, `TypeScript`

## Example

The OpenAI response contains token usage so all we need to do is to report it to OpenMeter with the corresponding user.
For idempotency we use the OpenAI API response `id` and for time we use response's `created` property.

Check out the [quickstart guide](/quickstart) to see how to run OpenMeter.

The Open AI response:

```json
{
   "id":"chatcmpl-abc123",
   "created":1677858242,
   "model":"gpt-3.5-turbo-0301",
   "usage":{
      "prompt_tokens":13,
      "completion_tokens":7,
      "total_tokens":20
   },
   ...
}
```

You can report usage to OpenMeter as:

```javascript
await openmeter.ingestEvents({
  specversion: '1.0',
  // We use Open AI response ID as idempotent key
  id: completion.id,
  source: 'my-app',
  type: 'openai',
  subject: 'my-awesome-user-id',
  // We use Open AI response date as event date
  time: new Date(completion.created * 1000).toISOString(),
  data: {
    total_tokens: completion.usage.total_tokens,
    prompt_tokens: completion.usage.prompt_tokens,
    completion_tokens: completion.usage.completion_tokens,
    model: completion.model,
  },
})
```

Note how we report the Open AI `model` version to OpenMeter.
This is useful as Open AI charges differently for varios models so you may want to group by them in OpenMeter.

Check out the full source code in the [app.ts](./app.ts).
You can run this example as:

```sh
npm install
OPENAI_API_KEY=sk-... npm start
```

## Stream Example

Modern applications thrive on being responsive and fluid. With Large Language Models (LLMs) like ChatGPT,
generating extensive outputs can take a while. Stream APIs allow for processing responses as soon as they become available.
OpenAI's data-only stream API makes this possible, but it doesn’t return token usage metadata,
which is by default included in OpenAI’s blocking API call response.

To fill the gap and enable accurate usage metering with stream APIs,
we implemented an example in `app-stream.ts` that tokenizes messages as they become available.

Check out the full source code in the [app-stream.ts](./app-stream.ts).
