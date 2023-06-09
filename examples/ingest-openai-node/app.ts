import fs from 'fs'
import assert from 'assert'

import 'dotenv/config'
import { Configuration, OpenAIApi } from 'openai'
import { OpenAPIClientAxios } from 'openapi-client-axios'
import yml from 'yaml'
import { Client as OpenMeterClient } from './openapi'

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
const openmeterApi = fs.readFileSync('../../api/openapi.yml', 'utf8')
const openmeter = await new OpenAPIClientAxios({
  definition: yml.parse(openmeterApi),
  withServer: { url: 'http://localhost:8888' },
}).initSync<OpenMeterClient>()

async function main() {
  const { data } = await openai.createChatCompletion({
    model: 'gpt-3.5-turbo',
    messages: [{ role: 'user', content: 'Hello world' }],
  })

  if (data.usage) {
    await openmeter.ingestEvents(
      null,
      {
        specversion: '1.0',
        // We use Open AI response ID as idempotent key
        id: data.id,
        source: 'my-app',
        type: 'openai',
        subject: 'my-awesome-user-id',
        // We get date from Open AI response
        time: new Date(data.created * 1000).toISOString(),
        // We report usage with model
        data: {
          total_tokens: data.usage.total_tokens,
          prompt_tokens: data.usage.prompt_tokens,
          completion_tokens: data.usage.completion_tokens,
          model: data.model,
        },
      },
      {
        headers: {
          'Content-Type': 'application/cloudevents+json',
        },
      }
    )
  }
}

main()
  .then(() => console.log('done'))
  .catch((err) => console.error(err))
