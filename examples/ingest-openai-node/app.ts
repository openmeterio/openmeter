import assert from 'assert'

import 'dotenv/config'
import { OpenMeter, WindowSize } from '@openmeter/sdk'
import { Configuration, OpenAIApi } from 'openai'
import dayjs from 'dayjs'

// Environment variables
assert.ok(
  process.env.OPENAI_ORG,
  'OPENAI_ORG environment variables is required'
)
assert.ok(
  process.env.OPENAI_API_KEY,
  'OPENAI_API_KEY environment variables is required'
)

const configuration = new Configuration({
  organization: process.env.OPENAI_ORG,
  apiKey: process.env.OPENAI_API_KEY,
})
const openai = new OpenAIApi(configuration)
const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })

async function main() {
  const { data } = await openai.createChatCompletion({
    model: 'gpt-3.5-turbo',
    messages: [{ role: 'user', content: 'Hello world' }],
  })

  if (data.usage) {
    await openmeter.events.ingest({
      specversion: '1.0',
      // We use Open AI response ID as idempotent key
      id: data.id,
      source: 'my-app',
      type: 'openai',
      subject: 'my-awesome-user-id',
      // We get date from Open AI response
      time: new Date(data.created * 1000),
      // We report usage with model
      data: {
        total_tokens: data.usage.total_tokens,
        prompt_tokens: data.usage.prompt_tokens,
        completion_tokens: data.usage.completion_tokens,
        model: data.model,
      },
    })

    // Debug logs
    console.debug(
      `input: Hello world, output: ${data.choices
        .map(({ message }) => message?.content)
        .join(' ')}`
    )
    console.debug(
      `total_tokens: ${data.usage.total_tokens}, subject: my-awesome-user-id`
    )
  }
}

// Query Example
// we can query usage by meter id, subject, time range on varios window sizes
async function query() {
  const from = dayjs().startOf('day').toDate()
  const to = dayjs(from).add(1, 'day').toDate()
  const { data: values } = await openmeter.meters.query('m2', {
    subject: ['my-awesome-user-id'],
    from,
    to,
  })

  console.log('Collected meter values:')
  console.log(JSON.stringify(values, null, 2))
  // Will print similar to:
  // [
  //     {
  //         subject: 'my-awesome-user-id',
  //         windowStart: '2023-07-25T00:00:00Z',
  //         windowEnd: '2023-07-26T00:00:00Z',
  //         value: 71,
  //         groupBy: { 'model': 'gpt-3.5-turbo-0613' }
  //     }
  // ]

  // Calling with multiple models will lead to:
  // [
  //     {
  //       subject: 'my-awesome-user-id',
  //       windowStart: '2023-07-25T00:00:00Z',
  //       windowEnd: '2023-07-26T00:00:00Z',
  //       value: 19,
  //       groupBy: { 'model': 'gpt-3.5-turbo-0301' }
  //     },
  //     {
  //       subject: 'my-awesome-user-id',
  //       windowStart: '2023-07-25T00:00:00Z',
  //       windowEnd: '2023-07-26T00:00:00Z',
  //       value: 89,
  //       groupBy: { 'model': 'gpt-3.5-turbo-0613' }
  //     }
  // ]
}

main()
  .then(() => query())
  .then(() => console.log('done'))
  .catch((err) => console.error(err))
