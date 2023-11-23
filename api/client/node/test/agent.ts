import { MockAgent } from 'undici'
import { mockEvent, mockMeter, mockMeterValue } from './mocks.js'

export const mockAgent = new MockAgent()
mockAgent.disableNetConnect()

const client = mockAgent.get('http://127.0.0.1:8888')

/** Event */
client
  .intercept({
    path: '/api/v1/events',
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/cloudevents+json',
    },
    body: JSON.stringify({
      specversion: '1.0',
      id: 'id-1',
      source: 'my-app',
      type: 'my-type',
      subject: 'my-awesome-user-id',
      time: new Date('2023-01-01'),
      data: {
        api_calls: 1,
      },
    }),
  })
  .reply(204)

client
  .intercept({
    path: `/api/v1/events`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(
    200,
    [
      {
        event: mockEvent,
      },
    ],
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  )

client
  .intercept({
    path: '/api/v1/events',
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/cloudevents+json',
    },
    body: JSON.stringify({
      specversion: '1.0',
      id: 'aaf17be7-860c-4519-91d3-00d97da3cc65',
      source: '@openmeter/sdk',
      type: 'my-type',
      subject: 'my-awesome-user-id',
      data: {
        api_calls: 1,
      },
    }),
  })
  .reply(204)

/** Portal */
client
  .intercept({
    path: '/api/v1/meters',
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, [mockMeter], {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/meters/${mockMeter.slug}`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, mockMeter, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

/** Meter Query */
client
  .intercept({
    path: `/api/v1/meters/${mockMeter.slug}/query`,
    query: {},
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(
    200,
    {
      from: mockMeterValue.windowStart,
      to: mockMeterValue.windowEnd,
      windowSize: 'HOUR',
      data: [mockMeterValue],
    },
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  )

client
  .intercept({
    path: `/api/v1/meters/${mockMeter.slug}/query`,
    query: {
      subject: 'user-1',
      groupBy: 'a,b',
      from: new Date('2021-01-01').toISOString(),
      to: new Date('2021-01-02').toISOString(),
      windowSize: 'HOUR',
    },
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(
    200,
    {
      from: mockMeterValue.windowStart,
      to: mockMeterValue.windowEnd,
      windowSize: 'HOUR',
      data: [mockMeterValue],
    },
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  )

/** Meter Subjects */
client
  .intercept({
    path: `/api/v1/meters/${mockMeter.slug}/subjects`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, [mockMeterValue.subject], {
    headers: {
      'Content-Type': 'application/json',
    },
  })

/** Portal */
client
  .intercept({
    path: '/api/v1/portal/tokens',
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      subject: 'customer-1',
    }),
  })
  .reply(
    201,
    {
      subject: 'customer-1',
      expiresAt: new Date('2023-01-01'),
    },
    {
      headers: {
        'Content-Type': 'application/json',
      },
    }
  )

client
  .intercept({
    path: '/api/v1/portal/tokens/invalidate',
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({}),
  })
  .reply(204)
