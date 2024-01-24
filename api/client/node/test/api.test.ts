import crypto from 'crypto'
import { setGlobalDispatcher } from 'undici'
import { vi, describe, it, expect, beforeEach } from 'vitest'
// test built version
import { CloudEvents } from '../clients/event.js'
import {
  OpenMeter,
  type Event,
  IngestedEvent,
  WindowSize,
} from '../dist/index.js'
import { mockAgent } from './agent.js'
import { mockEvent, mockMeter, mockMeterValue, mockSubject } from './mocks.js'

declare module 'vitest' {
  export interface TestContext {
    openmeter: OpenMeter
  }
}

setGlobalDispatcher(mockAgent)

describe('sdk', () => {
  beforeEach((ctx) => {
    ctx.openmeter = new OpenMeter({
      baseUrl: 'http://127.0.0.1:8888',
    })
  })

  describe('events', () => {
    describe('ingest', () => {
      it('should ingest event', async ({ openmeter }) => {
        const data = await openmeter.events.ingest(mockEvent)
        expect(data).toBeUndefined()
      })

      it('should ingest event with defaults', async ({ openmeter }) => {
        vi.spyOn(crypto, 'randomUUID').mockReturnValue(
          'aaf17be7-860c-4519-91d3-00d97da3cc65'
        )

        const event: Event = {
          type: 'my-type',
          subject: 'my-awesome-user-id',
          data: {
            api_calls: 1,
          },
        }

        const data = await openmeter.events.ingest(event)
        expect(data).toBeUndefined()
      })

      it('should batch ingest event', async ({ openmeter }) => {
        const data = await openmeter.events.ingest([mockEvent])
        expect(data).toBeUndefined()
      })
    })

    describe('list', () => {
      it('should list events', async ({ openmeter }) => {
        const events = await openmeter.events.list()
        const event = mockEvent as CloudEvents
        const expected: IngestedEvent = {
          event: {
            ...event,
            time: mockEvent.time?.toISOString(),
          },
        }
        expect(events).toEqual([expected])
      })
    })
  })

  describe('meters', () => {
    describe('list', () => {
      it('should list meters', async ({ openmeter }) => {
        const meters = await openmeter.meters.list()
        expect(meters).toEqual([mockMeter])
      })
    })

    describe('get', () => {
      it('should get meter', async ({ openmeter }) => {
        const meters = await openmeter.meters.get(mockMeter.slug)
        expect(meters).toEqual(mockMeter)
      })
    })

    describe('query', () => {
      it('should query meter', async ({ openmeter }) => {
        const { windowSize, data, from, to } = await openmeter.meters.query(
          mockMeter.slug
        )
        expect(from).toBe(mockMeterValue.windowStart)
        expect(to).toBe(mockMeterValue.windowEnd)
        expect(windowSize).toBe(WindowSize.HOUR)
        expect(data).toEqual([mockMeterValue])
      })

      it('should query meter (with params)', async ({ openmeter }) => {
        const subject = ['user-1']
        const groupBy = ['a', 'b']
        const from = new Date('2021-01-01')
        const to = new Date('2021-01-02')
        const windowSize = WindowSize.HOUR

        const data = await openmeter.meters.query(mockMeter.slug, {
          subject,
          groupBy,
          from,
          to,
          windowSize,
        })

        expect(data.from).toBe(mockMeterValue.windowStart)
        expect(data.to).toBe(mockMeterValue.windowEnd)
        expect(data.windowSize).toBe(WindowSize.HOUR)
        expect(data.data).toEqual([mockMeterValue])
      })
    })

    describe('meter subjects', () => {
      it('should get meter subjects', async ({ openmeter }) => {
        const subjects = await openmeter.meters.subjects(mockMeter.slug)
        expect(subjects).toEqual([mockMeterValue.subject])
      })
    })

    describe('portal', () => {
      it('should create token', async ({ openmeter }) => {
        const token = await openmeter.portal.createToken({
          subject: 'customer-1',
        })
        expect(token).toEqual({
          subject: 'customer-1',
          expiresAt: new Date('2023-01-01').toISOString(),
        })
      })
    })

    describe('subjects', () => {
      it('should upsert subjects', async ({ openmeter }) => {
        const subject = await openmeter.subject.upsert({
          key: mockSubject.key,
          displayName: mockSubject.displayName,
          metadata: mockSubject.metadata,
        })
        expect(subject).toEqual(mockSubject)
      })

      it('should list subjects', async ({ openmeter }) => {
        const subjects = await openmeter.subject.list()
        expect(subjects).toEqual([mockSubject])
      })

      it('should get subject', async ({ openmeter }) => {
        const subjects = await openmeter.subject.get(mockSubject.key)
        expect(subjects).toEqual(mockSubject)
      })

      it('should delete subject', async ({ openmeter }) => {
        const resp = await openmeter.subject.delete(mockSubject.key)
        expect(resp).toBeUndefined()
      })
    })
  })
})
