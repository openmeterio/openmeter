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

function mockProblem(status: number, body: Record<string, unknown>) {
  fetchMock.route('*', {
    status,
    body,
    headers: { 'Content-Type': 'application/problem+json; charset=utf-8' },
  })
}

describe('error mapping', () => {
  it('maps problem+json to a typed HTTPError with parsed fields', async () => {
    mockProblem(404, {
      type: 't',
      title: 'Not Found',
      status: 404,
      detail: 'nope',
      instance: '/x',
    })
    const result = await funcs.getMeter(client(), { meterId: 'x' })
    expect(result.ok).toBe(false)
    expect(result.error).toBeInstanceOf(HTTPError)
    const httpError = result.error as HTTPError
    expect(httpError.status).toBe(404)
    expect(httpError.title).toBe('Not Found')
    expect(httpError.message).toBe('nope')
  })

  it('exposes invalid_parameters via getField', async () => {
    mockProblem(400, {
      type: 't',
      title: 'Bad Request',
      status: 400,
      detail: 'validation failed',
      instance: '/x',
      invalid_parameters: [{ field: 'name', reason: 'is required' }],
    })
    const result = await funcs.getMeter(client(), { meterId: 'x' })
    const httpError = result.error as HTTPError
    expect(httpError.getField('invalid_parameters')).toEqual([
      { field: 'name', reason: 'is required' },
    ])
  })

  it('falls back to a status-only error for non-problem responses', async () => {
    fetchMock.route('*', {
      status: 500,
      body: 'oops',
      headers: { 'Content-Type': 'text/plain' },
    })
    const result = await funcs.getMeter(client(), { meterId: 'x' })
    expect(result.error).toBeInstanceOf(HTTPError)
    const httpError = result.error as HTTPError
    expect(httpError.status).toBe(500)
  })
})
