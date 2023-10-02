import { describe, it, expect, vi } from 'vitest'
import { createOpenAIStreamCallback } from '../dist/index.js'
import type { OpenAIStreamCallbacks } from 'ai'

describe('next', () => {
  const model = 'gpt-4'
  const prompts = ['Say Hello World!']

  describe('createOpenAIStreamCallback', () => {
    it('should return usage', () =>
      new Promise<void>((done) => {
        const streamCallbacks = createOpenAIStreamCallback(
          {
            model,
            prompts,
          },
          {
            onUsage: (usage) => {
              expect(usage).toEqual({
                total_tokens: 8,
                prompt_tokens: 4,
                completion_tokens: 4,
              })

              done()
            },
          }
        )

        streamCallbacks.onStart!()
        streamCallbacks.onToken!('Hello ')
        streamCallbacks.onToken!(' World!')
        streamCallbacks.onFinal!('Hello World!')
      }))

    it('should call methods', () =>
      new Promise<void>((done) => {
        const onStartSpy = vi.fn()
        const onTokenSpy = vi.fn()
        const onCompletionSpy = vi.fn()

        const callbacks: OpenAIStreamCallbacks = {
          onStart: onStartSpy,
          onToken: onTokenSpy,
          onCompletion: onCompletionSpy,
          onFinal() {
            expect(onStartSpy).toHaveBeenCalledTimes(1)
            expect(onTokenSpy).toHaveBeenCalledTimes(2)
            expect(onCompletionSpy).toHaveBeenCalledTimes(1)

            done()
          },
        }

        const streamCallbacks = createOpenAIStreamCallback(
          {
            model,
            prompts,
          },
          callbacks
        )

        streamCallbacks.onStart!()
        streamCallbacks.onToken!('Hello ')
        streamCallbacks.onToken!(' World!')
        streamCallbacks.onCompletion!('Hello World!')
        streamCallbacks.onFinal!('Hello World!')
      }))
  })
})
