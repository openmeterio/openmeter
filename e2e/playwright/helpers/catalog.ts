import type { APIRequestContext } from '@playwright/test'
import { expect } from '@playwright/test'
import { faker } from '@faker-js/faker'

export const BASE = '/api/v3/openmeter'
export const V1_BASE = '/api/v1'

export type CreateMeterOverrides = {
  key?: string
  name?: string
  aggregation?: 'sum' | 'count' | 'unique_count' | 'avg' | 'min' | 'max' | 'latest'
  event_type?: string
  value_property?: string
  description?: string
}

export type Meter = {
  id: string
  key: string
}

export async function createMeter(
  request: APIRequestContext,
  overrides: CreateMeterOverrides = {},
): Promise<Meter> {
  const body = {
    key: overrides.key ?? `meter_${faker.string.alphanumeric({ length: 16, casing: 'lower' })}`,
    name: overrides.name ?? 'Test Meter',
    aggregation: overrides.aggregation ?? 'sum',
    event_type: overrides.event_type ?? 'request',
    value_property: overrides.value_property ?? '$.duration_ms',
    ...(overrides.description ? { description: overrides.description } : {}),
  }

  const res = await request.post(`${BASE}/meters`, { data: body })
  expect(res.status(), `meter create failed: ${await res.text()}`).toBe(201)
  const meter = await res.json()
  return { id: meter.id, key: meter.key }
}

export type CreateFeatureOverrides = {
  key?: string
  name?: string
  meterId?: string
}

export type Feature = {
  id: string
  key: string
}

export async function createFeature(
  request: APIRequestContext,
  overrides: CreateFeatureOverrides = {},
): Promise<Feature> {
  const body: Record<string, unknown> = {
    key: overrides.key ?? `feature_${faker.string.alphanumeric({ length: 16, casing: 'lower' })}`,
    name: overrides.name ?? 'Test Feature',
  }
  if (overrides.meterId) {
    body.meter = { id: overrides.meterId }
  }

  const res = await request.post(`${BASE}/features`, { data: body })
  expect(res.status(), `feature create failed: ${await res.text()}`).toBe(201)
  const feature = await res.json()
  return { id: feature.id, key: feature.key }
}

export type CloudEvent = {
  id: string
  source: string
  type: string
  subject: string
  time?: string
  data?: Record<string, unknown>
}

export async function ingestEvent(
  request: APIRequestContext,
  event: CloudEvent,
): Promise<void> {
  const body = {
    specversion: '1.0',
    id: event.id,
    source: event.source,
    type: event.type,
    subject: event.subject,
    ...(event.time ? { time: event.time } : {}),
    ...(event.data ? { data: event.data } : {}),
  }

  const res = await request.post(`${V1_BASE}/events`, {
    headers: { 'Content-Type': 'application/cloudevents+json' },
    data: body,
  })
  expect(res.status(), `ingest failed: ${await res.text()}`).toBe(204)
}
