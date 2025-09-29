import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type { Event, operations, paths } from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Events are used to track usage of your product or service.
 * Events are processed asynchronously by the meters, so they may not be immediately available for querying.
 */
export class Events {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Ingests an event or batch of events
   * @param events - The events to ingest
   * @param signal - An optional abort signal
   * @returns The ingested events
   */
  public async ingest(events: Event | Event[], options?: RequestOptions) {
    const body = await Promise.all(
      (Array.isArray(events) ? events : [events]).map(setDefaultsForEvent),
    )

    const resp = await this.client.POST('/api/v1/events', {
      body,
      headers: {
        'Content-Type': 'application/cloudevents-batch+json',
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List events
   * @param params - The query parameters
   * @param options - Optional request options
   * @returns The events
   */
  public async list(
    params?: operations['listEvents']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/events', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List events (V2)
   * @description List ingested events with advanced filtering and cursor pagination.
   * @param params - The query parameters
   * @param options - Optional request options
   * @returns The events
   */
  public async listV2(
    params?: operations['listEventsV2']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v2/events', {
      params: { query: params },
      ...options,
    })

    return transformResponse(resp)
  }
}

/**
 * Sets the defaults for an event
 * @param ev - The event to set the defaults for
 * @returns The event with the defaults set
 */
export async function setDefaultsForEvent(ev: Event): Promise<Event> {
  return {
    ...ev,
    id: ev.id ?? (await generateId()),
    source: ev.source ?? '@openmeter/sdk',
    specversion: ev.specversion ?? '1.0',
    time: ev.time ?? new Date(),
  }
}

let _randomUUID: (() => string) | undefined

// One-off attempt to load node:crypto and capture randomUUID (if present)
async function loadUUIDProvider() {
  if (_randomUUID !== undefined) {
    // already tried
    return _randomUUID
  }

  try {
    const c = await import('node:crypto')
    if (typeof c.randomUUID === 'function') {
      // available
      _randomUUID = c.randomUUID.bind(c)
    }
  } catch {
    // not-available
  }

  return _randomUUID
}

/**
 * Generates a random ID
 * @returns A random ID
 */
async function generateId() {
  const randomUUID = await loadUUIDProvider()
  if (randomUUID) {
    return randomUUID()
  }

  // Fallback to semi-random ID
  const bytes = new Uint8Array(16)
  for (let i = 0; i < 16; i++) {
    bytes[i] = (Math.random() * 256) | 0
  }

  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80

  const hex = [...bytes].map((b) => b.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}
