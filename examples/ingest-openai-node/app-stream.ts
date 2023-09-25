import assert from 'assert'

import 'dotenv/config'
import { OpenMeter } from '@openmeter/sdk'
import OpenAI from 'openai'
import * as tiktoken from "js-tiktoken";

const apiKey = process.env.OPENAI_API_KEY

// Environment variables
assert.ok(apiKey, 'OPENAI_API_KEY environment variables is required')

const openai = new OpenAI({ apiKey })
const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })

async function main() {
  const model = 'gpt-3.5-turbo'
  const enc = tiktoken.encodingForModel(model);
  let totalTokens = 0;

  const stream = await await openai.chat.completions.create({
    model: 'gpt-3.5-turbo',
    messages: [{ role: 'user', content: 'Say Hello OpenMeter!' }],
    stream: true,
  })

  // Consume stream and tokenize messages
  for await (const chunk of stream) {
    const content = chunk.choices[0]?.delta?.content || ""
    const tokens = enc.encode(content);
    totalTokens += tokens.length;

    console.log(`Tokens: ${tokens.length}, Message: "${content}"`);
  }

  // Report usage to OpenMeter
  await openmeter.events.ingest({
    specversion: '1.0',
    source: 'my-app',
    type: 'openai',
    subject: 'my-awesome-user-id',
    data: {
      total_tokens: totalTokens,
      model,
    }
  })
}

main()
  .then(() => console.log('done'))
  .catch((err) => console.error(err))
