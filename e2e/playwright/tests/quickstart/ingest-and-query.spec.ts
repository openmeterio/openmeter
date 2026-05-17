/**
 * Quickstart > ingest and query
 *
 * Mirrors the OpenMeter quickstart flow: create a SUM meter, ingest three
 * CloudEvents with distinct ids and timestamps, then poll the meter query
 * endpoint until the aggregated value reflects all events.
 *
 * Endpoints exercised:
 *   POST /api/v3/openmeter/meters
 *   POST /api/v3/openmeter/events (CloudEvents JSON, 202 accepted)
 *   POST /api/v3/openmeter/meters/{meterId}/query
 *
 * Requires the sink-worker to be running alongside the API server; the
 * v3 query stays at 0 until events are processed off Kafka into ClickHouse.
 */
import { test, expect } from '@playwright/test'
import { faker } from '@faker-js/faker'
import { BASE, createMeter, ingestEvent } from '../../helpers/catalog'

test.describe.configure({ mode: 'serial' })

test.describe('Quickstart > ingest and query', () => {
  let meterId: string
  let eventType: string
  let subject: string

  // Three events whose duration_ms values sum to a known total. Different ids
  // and timestamps to mirror the quickstart docs.
  const events = [
    { id: faker.string.uuid(), time: '2023-01-01T00:00:00Z', durationMs: 10 },
    { id: faker.string.uuid(), time: '2023-01-01T01:00:00Z', durationMs: 20 },
    { id: faker.string.uuid(), time: '2023-01-02T00:00:00Z', durationMs: 30 },
  ]
  const expectedSum = events.reduce((acc, e) => acc + e.durationMs, 0)

  test('creates a meter', async ({ request }) => {
    eventType = `request_${faker.string.alphanumeric({ length: 8, casing: 'lower' })}`
    const meter = await createMeter(request, {
      aggregation: 'sum',
      event_type: eventType,
      value_property: '$.duration_ms',
    })
    meterId = meter.id
    expect(meterId).toBeTruthy()
  })

  test('ingests three CloudEvents and queries the meter', async ({ request }) => {
    expect(meterId).toBeTruthy()
    subject = `customer_${faker.string.alphanumeric({ length: 12, casing: 'lower' })}`

    for (const ev of events) {
      await ingestEvent(request, {
        id: ev.id,
        source: 'playwright-smoke',
        type: eventType,
        subject,
        time: ev.time,
        data: { duration_ms: String(ev.durationMs) },
      })
    }

    // Sink processing is async (Kafka -> ClickHouse). Poll the query endpoint
    // until the aggregated value reflects all three events.
    await expect.poll(
      async () => {
        const res = await request.post(`${BASE}/meters/${meterId}/query`, {
          data: {
            from: '2023-01-01T00:00:00Z',
            to: '2023-01-03T00:00:00Z',
            filters: { dimensions: { subject: { eq: subject } } },
          },
        })
        if (res.status() !== 200) return -1
        const body = await res.json()
        if (!Array.isArray(body.data) || body.data.length === 0) return 0
        return Number(body.data[0].value)
      },
      {
        timeout: 60_000,
        intervals: [500, 1000, 2000, 5000],
        message: `meter ${meterId} did not converge to ${expectedSum} for subject ${subject}`,
      },
    ).toBe(expectedSum)
  })
})
