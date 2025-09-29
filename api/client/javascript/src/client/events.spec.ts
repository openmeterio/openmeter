import fetchMock from '@fetch-mock/vitest'
import { beforeEach, describe, expect, it } from 'vitest'
import type { Event } from './index.js'
import { OpenMeter } from './index.js'

interface Context {
  baseUrl: string
  client: OpenMeter
}

describe('Events', () => {
  beforeEach<Context>((ctx) => {
    fetchMock.mockReset()
    const baseUrl = 'http://openmeter-mock.local'
    const client = new OpenMeter({
      baseUrl,
      fetch: fetchMock.fetchHandler,
    })

    ctx.baseUrl = baseUrl
    ctx.client = client
  })

  it<Context>('ingest (POST /api/v1/events)', async ({
    baseUrl,
    client,
    task,
  }) => {
    const route = `${baseUrl}/api/v1/events`
    const event: Event = {
      data: {
        tokens: 100,
      },
      id: '5c10fade-1c9e-4d6c-8275-c52c36731d3c',
      subject: 'customer_id',
      time: new Date(),
      type: 'prompt',
    }

    fetchMock.route(
      route,
      {
        status: 200,
      },
      {
        body: [
          {
            ...event,
            source: '@openmeter/sdk',
            specversion: '1.0',
            subject: 'customer_id',
            time: event.time?.toISOString(),
            type: 'prompt',
          },
        ],
        headers: {
          'Content-Type': 'application/cloudevents-batch+json',
        },
        method: 'POST',
        name: task.name,
      },
    )
    const resp = await client.events.ingest(event)
    expect(resp).toBeUndefined()
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })

  it<Context>('list (GET /api/v1/events)', async ({
    baseUrl,
    client,
    task,
  }) => {
    const query = {
      from: new Date(),
      hasError: false,
      id: '5c10fade-1c9e-4d6c-8275-c52c36731d3c',
      ingestedAtFrom: new Date(),
      ingestedAtTo: new Date(),
      limit: 10,
      subject: 'customer_id',
      to: new Date(),
    }
    const route = `${baseUrl}/api/v1/events`
    const respBody = []
    fetchMock.route(
      route,
      {
        body: respBody,
        status: 200,
      },
      {
        method: 'GET',
        name: task.name,
        query: {
          from: query.from.toISOString(),
          hasError: query.hasError.toString(),
          id: query.id,
          ingestedAtFrom: query.ingestedAtFrom.toISOString(),
          ingestedAtTo: query.ingestedAtTo.toISOString(),
          limit: query.limit.toString(),
          subject: query.subject,
          to: query.to.toISOString(),
        },
      },
    )
    const resp = await client.events.list(query)
    expect(resp).toEqual(respBody)
    expect(fetchMock.callHistory.done(task.name)).toBeTruthy()
  })
})
