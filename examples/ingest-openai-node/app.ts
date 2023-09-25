import assert from 'assert'

import 'dotenv/config'
import { OpenMeter, WindowSize } from '@openmeter/sdk'
import OpenAI from 'openai'
import dayjs from 'dayjs'

const apiKey = process.env.OPENAI_API_KEY

// Environment variables
assert.ok(apiKey, 'OPENAI_API_KEY environment variables is required')

const openai = new OpenAI({ apiKey })
const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })

async function main() {
  const model = 'gpt-3.5-turbo'
  const completion = await await openai.chat.completions.create({
    model,
    messages: [{ role: 'user', content: 'Say Hello OpenMeter!' }],
  })

  if (completion.usage) {
    await openmeter.events.ingest({
      specversion: '1.0',
      // We use Open AI response ID as idempotent key
      id: completion.id,
      source: 'my-app',
      type: 'openai',
      subject: 'my-awesome-user-id',
      // We get date from Open AI response
      time: new Date(completion.created * 1000),
      // We report usage with model
      data: {
        total_tokens: completion.usage.total_tokens,
        model,
      },
    })

    // Debug logs
    console.debug(
      `input: Hello world, output: ${completion.choices
        .map(({ message }) => message?.content)
        .join(' ')}`
    )
    console.debug(
      `total_tokens: ${completion.usage.total_tokens}, subject: my-awesome-user-id`
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
    groupBy: ['model'],
    from,
    to,
    windowSize: WindowSize.DAY,
  })

  console.log('Query meter:')
  console.log(JSON.stringify(values, null, 2))
  // Will print similar to:
  // [
  //     {
  //         subject: 'my-awesome-user-id',
  //         windowStart: '2023-07-25T00:00:00Z',
  //         windowEnd: '2023-07-26T00:00:00Z',
  //         value: 71,
  //         groupBy: { 'model': 'gpt-3.5-turbo' }
  //     }
  // ]

  // Calling with multiple models will lead to:
  // [
  //     {
  //       subject: 'my-awesome-user-id',
  //       windowStart: '2023-07-25T00:00:00Z',
  //       windowEnd: '2023-07-26T00:00:00Z',
  //       value: 19,
  //       groupBy: { 'model': 'gpt-3.5-turbo' }
  //     },
  //     {
  //       subject: 'my-awesome-user-id',
  //       windowStart: '2023-07-25T00:00:00Z',
  //       windowEnd: '2023-07-26T00:00:00Z',
  //       value: 89,
  //       groupBy: { 'model': 'gpt-4' }
  //     }
  // ]

  // If you need the total across all models, you can use the following:
  const total = values.reduce(
    (total: number, { value }) => total + (value || 0),
    0
  )
  console.log(`Total token usage across all models: ${total}`)
}

main()
  .then(() => query())
  .then(() => console.log('done'))
  .catch((err) => console.error(err))
