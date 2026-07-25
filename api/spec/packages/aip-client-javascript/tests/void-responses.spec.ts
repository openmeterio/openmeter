import fetchMock from '@fetch-mock/vitest'
import { beforeEach, describe, expect, it } from 'vitest'
import { Client, HTTPError, funcs } from '../src/index.js'

beforeEach(() => {
  fetchMock.mockReset()
})

function client() {
  return new Client({
    baseUrl: 'https://eu.api.konghq.com/v3',
    apiKey: 'k',
    fetch: fetchMock.fetchHandler,
  })
}

describe('void operations do not parse the response body', () => {
  it('resolves ingest of a single event on an empty 202 Accepted body', async () => {
    fetchMock.route('*', { status: 202, body: '' })
    const result = await funcs.ingestMeteringEvents(client(), {
      specversion: '1.0',
      id: 'evt-1',
      source: 'test',
      type: 'request',
      subject: 'customer-1',
    })
    expect(result.ok).toBe(true)
    expect(result.value).toBeUndefined()
  })

  it('resolves ingest of a batch of events (array body)', async () => {
    fetchMock.route('*', { status: 202, body: '' })
    const result = await funcs.ingestMeteringEvents(client(), [
      {
        specversion: '1.0',
        id: 'evt-1',
        source: 'test',
        type: 'request',
        subject: 'customer-1',
      },
      {
        specversion: '1.0',
        id: 'evt-2',
        source: 'test',
        type: 'request',
        subject: 'customer-2',
      },
    ])
    expect(result.ok).toBe(true)
    expect(result.value).toBeUndefined()
    const sent = JSON.parse(
      fetchMock.callHistory.lastCall()!.options.body as string,
    )
    expect(Array.isArray(sent)).toBe(true)
    expect(sent).toHaveLength(2)
  })

  it('resolves delete on a 204 No Content response', async () => {
    fetchMock.route('*', { status: 204 })
    const result = await funcs.deleteMeter(client(), { meterId: 'm' })
    expect(result.ok).toBe(true)
    expect(result.value).toBeUndefined()
  })

  it('still rejects a void operation on a non-2xx status', async () => {
    fetchMock.route('*', {
      status: 500,
      body: 'oops',
      headers: { 'Content-Type': 'text/plain' },
    })
    const result = await funcs.deleteMeter(client(), { meterId: 'm' })
    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(HTTPError)
    expect((result.error as HTTPError).status).toBe(500)
  })

  it('preserves problem+json error detail on a void operation', async () => {
    fetchMock.route('*', {
      status: 404,
      body: {
        type: 't',
        title: 'Not Found',
        status: 404,
        detail: 'nope',
        instance: '/x',
      },
      headers: {
        'Content-Type': 'application/problem+json; charset=utf-8',
      },
    })
    const result = await funcs.deleteMeter(client(), { meterId: 'm' })
    expect(result.ok).toBe(false)
    const error = result.error as HTTPError
    expect(error.status).toBe(404)
    expect(error.title).toBe('Not Found')
    expect(error.message).toBe('nope')
  })
})
