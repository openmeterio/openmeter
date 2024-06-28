import { MockAgent } from 'undici'
import {
  mockCreateEntitlementInput,
  mockCreateFeatureInput,
  mockEntitlement,
  mockEntitlementGrant,
  mockEntitlementGrantCreateInput,
  mockEntitlementValue,
  mockEvent,
  mockFeature,
  mockMeter,
  mockMeterValue,
  mockSubject,
  mockWindowedBalanceHistory,
} from './mocks.js'

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

// Batch ingest
client
  .intercept({
    path: '/api/v1/events',
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/cloudevents-batch+json',
    },
    body: JSON.stringify([
      {
        specversion: '1.0',
        id: 'id-1',
        source: 'my-app',
        type: 'my-type',
        subject: 'my-awesome-user-id',
        time: new Date('2023-01-01'),
        data: {
          api_calls: 1,
        },
      },
    ]),
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
      groupBy: ['a', 'b'],
      from: new Date('2021-01-01').toISOString(),
      to: new Date('2021-01-02').toISOString(),
      windowSize: 'HOUR',
      'filterGroupBy[model]': 'gpt-4',
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

/** Subjects */
client
  .intercept({
    path: '/api/v1/subjects',
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify([
      {
        key: mockSubject.key,
        displayName: mockSubject.displayName,
        metadata: mockSubject.metadata,
      },
    ]),
  })
  .reply(200, [mockSubject], {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: '/api/v1/subjects',
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, [mockSubject], {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, mockSubject, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}`,
    method: 'DELETE',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(204)

/** Subject Entitlements */

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}/entitlements`,
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(mockCreateEntitlementInput),
  })
  .reply(201, mockEntitlement, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}/entitlements`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, [mockEntitlement], {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}/entitlements/${mockFeature.key}`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, mockEntitlement, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}/entitlements/${mockFeature.key}`,
    method: 'DELETE',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(204)

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}/entitlements/${mockFeature.key}/value`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, mockEntitlementValue, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}/entitlements/${mockFeature.key}/history`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, mockWindowedBalanceHistory, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}/entitlements/${mockFeature.key}/reset`,
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      retainAnchor: true,
    }),
  })
  .reply(204, mockEntitlement, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

/** Subject Entitlement Grants */

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}/entitlements/${mockFeature.key}/grants`,
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(mockEntitlementGrantCreateInput),
  })
  .reply(201, mockEntitlementGrant, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/subjects/${mockSubject.key}/entitlements/${mockFeature.key}/grants`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, [mockEntitlementGrant], {
    headers: {
      'Content-Type': 'application/json',
    },
  })

/** Features */

client
  .intercept({
    path: '/api/v1/features',
    method: 'POST',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(mockCreateFeatureInput),
  })
  .reply(201, mockFeature, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: '/api/v1/features',
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, [mockFeature], {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/features/${mockFeature.key}`,
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, mockFeature, {
    headers: {
      'Content-Type': 'application/json',
    },
  })

client
  .intercept({
    path: `/api/v1/features/${mockFeature.key}`,
    method: 'DELETE',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(204)

/** Entitlements */

client
  .intercept({
    path: '/api/v1/entitlements',
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, [mockEntitlement], {
    headers: {
      'Content-Type': 'application/json',
    },
  })

/** Grants */

client
  .intercept({
    path: '/api/v1/grants',
    method: 'GET',
    headers: {
      Accept: 'application/json',
    },
  })
  .reply(200, [mockEntitlementGrant], {
    headers: {
      'Content-Type': 'application/json',
    },
  })
