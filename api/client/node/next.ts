import type { OpenAIStreamCallbacks } from 'ai'
import type { TiktokenModel } from 'js-tiktoken'

let encodingForModel: (model: TiktokenModel) => any | undefined

type OpenAIUsage = {
  total_tokens: number
  prompt_tokens: number
  completion_tokens: number
}

type OpenAIStreamCallbacksWithUsage = OpenAIStreamCallbacks & {
  onUsage?: (usage: OpenAIUsage) => void
}

export function createOpenAIStreamCallback(
  {
    model,
    prompts,
  }: {
    model: TiktokenModel
    prompts: string[]
  },
  openAIStreamCallbacks: OpenAIStreamCallbacksWithUsage
) {
  // Tiktoken is an optional dependency, so we require it conditionally
  if (!encodingForModel) {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const { encodingForModel: encodingForModel_ } = require('js-tiktoken')
    encodingForModel = encodingForModel_
  }

  const enc = encodingForModel(model)
  let promptTokens = 0
  let completionTokens = 0

  const streamCallbacks: OpenAIStreamCallbacks = {
    ...openAIStreamCallbacks,

    onStart() {
      for (const content of prompts) {
        const tokens = enc.encode(content)
        promptTokens += tokens.length
      }

      if (typeof openAIStreamCallbacks?.onStart === 'function') {
        return openAIStreamCallbacks.onStart()
      }
    },
    onToken(content) {
      // To test tokenizaton see: https://platform.openai.com/tokenizer
      const tokens = enc.encode(content)
      completionTokens += tokens.length

      if (typeof openAIStreamCallbacks?.onToken === 'function') {
        return openAIStreamCallbacks.onToken(content)
      }
    },
    onFinal(completion: string) {
      // Mimicking OpenAI usage metadata API
      const usage: OpenAIUsage = {
        total_tokens: promptTokens + completionTokens,
        prompt_tokens: promptTokens,
        completion_tokens: completionTokens,
      }

      if (typeof openAIStreamCallbacks?.onUsage === 'function') {
        openAIStreamCallbacks.onUsage(usage)
      }

      if (typeof openAIStreamCallbacks?.onFinal === 'function') {
        return openAIStreamCallbacks.onFinal(completion)
      }
    },
  }
  return streamCallbacks
}
