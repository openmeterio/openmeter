# OpenAI Example

In this example we will track our customer's OpenAI token consumption.
This is useful if you are building top of the Open AI API like Chat GPT and want to meter your customers usage for reporting or billing purposes.

Language: Node.js, TypeScript

## Example

The OpenAI response contains token usage so all we need to do is to report it to OpenMeter with the corresponding user.
For idempotency we use the OpenAI API response `id` and for time we use response's `created` property.

Check out the [quickstart guide](/quickstart) to see how to run OpenMeter.

Open AI response:

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

Report usage to OpenMeter:

```javascript
await openmeter.ingestEvents({
	specversion: '1.0',
	// We use Open AI response ID as idempotent key
	id: data.id,
	source: 'my-app',
	type: 'openai',
	subject: 'my-awesome-user-id',
	// We use Open AI response date as event date
	time: new Date(data.created * 1000).toISOString(),
	data: {
		total_tokens: data.usage.total_tokens,
		prompt_tokens: data.usage.prompt_tokens,
		completion_tokens: data.usage.completion_tokens,
		model: data.model,
	},
})
```

Note how we also collect the Open AI `model` version.
This is useful as Open AI charges differently for varios models so you may want to group by them in OpenMeter.

Check out the full source code in the `app.ts`.
You can also run it as:

```sh
npm install
OPENAI_ORG=org-.. OPENAI_API_KEY=sk-... npm start
```
