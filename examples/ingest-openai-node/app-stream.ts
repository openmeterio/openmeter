import assert from 'assert'

import 'dotenv/config'
import { OpenMeter } from '@openmeter/sdk'
import OpenAI from 'openai'
import * as tiktoken from "js-tiktoken";
import { ChatCompletionMessageParam } from 'openai/resources/chat';

const apiKey = process.env.OPENAI_API_KEY

// Environment variables
assert.ok(apiKey, 'OPENAI_API_KEY environment variables is required')

const openai = new OpenAI({ apiKey })
const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })

async function main() {
  const messages: Array<ChatCompletionMessageParam> = [{ role: 'user', content: 'Say Hello OpenMeter!' }]
  const model = 'gpt-3.5-turbo'

  // Get the tokenization encoding for the model
  const enc = tiktoken.encodingForModel(model);

  // Prompt tokens (input) and completion tokens (output) are priced differently with OpenAI
  const promptTokens = messages.reduce((acc, m) => acc + enc.encode(m.content ?? '').length, 0);
  let completionTokens = 0;

  // Create a stream of chat completions
  const stream = await openai.chat.completions.create({
    model,
    messages,
    stream: true,
  })

  // Consume stream and tokenize messages
  for await (const chunk of stream) {
    const content = chunk.choices[0]?.delta?.content || ""
    const tokens = enc.encode(content);
    completionTokens += tokens.length;

    console.log(`Tokens: ${tokens.length}, Message: "${content}"`);
  }

  // Report usage to OpenMeter
  await openmeter.events.ingest({
    specversion: '1.0',
    source: 'my-app',
    type: 'openai',
    subject: 'my-awesome-user-id',
    data: {
      total_tokens: promptTokens + completionTokens,
      prompt_tokens: promptTokens,
      completion_tokens: completionTokens,
      model,
    }
  })
}

main()
  .then(() => console.log('done'))
  .catch((err) => console.error(err))
