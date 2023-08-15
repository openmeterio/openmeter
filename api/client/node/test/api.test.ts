import crypto from 'crypto'
import { setGlobalDispatcher } from 'undici'
import { vi, describe, it, expect, beforeEach } from 'vitest'
// test built version
import { OpenMeter, type Event, WindowSize } from '../dist/index.js'
import { mockAgent } from './agent.js'
import { mockMeter, mockMeterValue } from './mocks.js'

declare module 'vitest' {
    export interface TestContext {
        openmeter: OpenMeter
    }
}

setGlobalDispatcher(mockAgent);

describe('sdk', () => {
    beforeEach((ctx) => {
        ctx.openmeter = new OpenMeter({
            baseUrl: 'http://127.0.0.1:8888',
        })
    })

    describe('events', () => {
        describe('ingest', () => {
            it('should ingest event', async ({ openmeter }) => {
                const event: Event = {
                    specversion: '1.0',
                    id: 'id-1',
                    source: 'my-app',
                    type: 'my-type',
                    subject: 'my-awesome-user-id',
                    time: new Date('2023-01-01'),
                    data: {
                        api_calls: 1,
                    },
                }
                const data = await openmeter.events.ingest(event)
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
        })
    })

    describe(('meters'), () => {
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

        describe('values', () => {
            it('should get meter values', async ({ openmeter }) => {
                const { windowSize, data } = await openmeter.meters.values(mockMeter.slug)
                expect(windowSize).toBe(WindowSize.HOUR)
                expect(data).toEqual([mockMeterValue])
            })

            it('should get meter values (with params)', async ({ openmeter }) => {
                const subject = 'user-1'
                const from = new Date('2021-01-01')
                const to = new Date('2021-01-02')
                const windowSize = WindowSize.HOUR

                const data = await openmeter.meters.values(mockMeter.slug, {
                    subject,
                    from,
                    to,
                    windowSize
                })

                expect(data.windowSize).toBe(WindowSize.HOUR)
                expect(data.data).toEqual([mockMeterValue])
            })
        })
    })
})
