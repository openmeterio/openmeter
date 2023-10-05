import { describe, it, expect, vi } from 'vitest'
import { createOpenAIStreamCallback } from '../dist/index.js'
import type { OpenAIStreamCallbacks } from 'ai'

describe('next', () => {
  const model = 'gpt-4'
  const prompts = ['Say Hello World!']

  describe('createOpenAIStreamCallback', () => {
    it('should return usage', async () => {
      const onUsage = vi.fn()
      const streamCallbacks = await createOpenAIStreamCallback(
        {
          model,
          prompts,
        },
        {
          onUsage,
        }
      )

      await streamCallbacks.onStart!()
      await streamCallbacks.onToken!('Hello ')
      await streamCallbacks.onToken!(' World!')
      await streamCallbacks.onFinal!('Hello World!')
      expect(onUsage).toHaveBeenCalledTimes(1)
      expect(onUsage).toHaveBeenCalledWith({
        total_tokens: 8,
        prompt_tokens: 4,
        completion_tokens: 4,
      })
    })

    it('should call methods', async () => {
      const onStartSpy = vi.fn()
      const onTokenSpy = vi.fn()
      const onCompletionSpy = vi.fn()
      const onFinalSpy = vi.fn()

      const callbacks: OpenAIStreamCallbacks = {
        onStart: onStartSpy,
        onToken: onTokenSpy,
        onCompletion: onCompletionSpy,
        onFinal: onFinalSpy,
      }

      const streamCallbacks = await createOpenAIStreamCallback(
        {
          model,
          prompts,
        },
        callbacks
      )

      await streamCallbacks.onStart!()
      await streamCallbacks.onToken!('Hello ')
      await streamCallbacks.onToken!(' World!')
      await streamCallbacks.onCompletion!('Hello World!')
      await streamCallbacks.onFinal!('Hello World!')

      expect(onStartSpy).toHaveBeenCalledTimes(1)
      expect(onTokenSpy).toHaveBeenCalledTimes(2)
      expect(onCompletionSpy).toHaveBeenCalledTimes(1)
      expect(onFinalSpy).toHaveBeenCalledTimes(1)
    })
  })
})
