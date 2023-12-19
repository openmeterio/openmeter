import dotenv from 'dotenv'
import { OpenAI } from 'langchain/llms/openai'
import { ChatPromptTemplate } from 'langchain/prompts'
import { BaseCallbackHandler } from 'langchain/callbacks'
import { LLMChain } from 'langchain/chains'
import { OpenMeter, Event } from '@openmeter/sdk'
import { Serialized } from 'langchain/load/serializable'
import { LLMResult } from 'langchain/schema'

// Load environment variables from .env file
dotenv.config()

// Validate environment variables
if (!process.env.OPENAI_MODEL_NAME) {
  throw new Error('OPENAI_MODEL_NAME is required')
}
if (!process.env.OPENAI_API_KEY) {
  throw new Error('OPENAI_API_KEY is required')
}
if (!process.env.OPENMETER_BASE_URL) {
  throw new Error('OPENMETER_BASE_URL is required')
}

// We can construct an LLMChain from a PromptTemplate and an LLM.
const model = new OpenAI({
  temperature: 0,
  modelName: process.env.OPENAI_MODEL_NAME,
})
const prompt = ChatPromptTemplate.fromMessages([
  ['system', 'You are a cat expert, answer questions about them.'],
  ['user', '{question}'],
])

const om = new OpenMeter({
  baseUrl: process.env.OPENMETER_BASE_URL,
  token: process.env.OPENMETER_TOKEN,
})

export class OpenMeterCallbackHandler extends BaseCallbackHandler {
  name = 'OpenMeterCallbackHandler'

  async handleLLMStart(llm: Serialized, prompts: string[], runId: string) {
    const input = prompts.join('\n')
    console.log('start', runId, input)

    const event: Event = {
      id: `${runId}-input`,
      subject: 'user-1',
      type: 'tokens',
      source: 'langchain',
      data: {
        tokens: await model.getNumTokens(input),
        type: 'input',
        model: model.modelName,
      },
    }

    await om.events.ingest(event)
  }

  async handleLLMEnd(output: LLMResult, runId: string) {
    const event: Event = {
      id: `${runId}-output`,
      subject: 'user-1',
      type: 'tokens',
      source: 'langchain',
      data: {
        tokens: await model.getNumTokens(
          output.generations[0].map(({ text }) => text).join('')
        ),
        type: 'output',
        model: model.modelName,
      },
    }
    await om.events.ingest(event)
  }
}

// Create a chain with the model and prompt
const chain = new LLMChain({
  llm: model,
  prompt,
})

const res = await chain.call(
  {
    question: 'What is the personality of siameses?',
  },
  { callbacks: [new OpenMeterCallbackHandler()] }
)

console.log('Response:', res.text)
