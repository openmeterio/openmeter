import { Interceptable, MockAgent, setGlobalDispatcher } from 'undici'
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { MeterQueryRow, OpenMeterClient } from '../dist'

const mockMeterQueryRow: MeterQueryRow = {
  subject: 'user-1',
  windowStart: '2023-01-01T00:00:00Z',
  windowEnd: '2023-01-02T00:00:00Z',
  value: 1,
  groupBy: {},
}

declare module 'vitest' {
  export interface TestContext {
    openmeter: OpenMeterClient
    fetchMock: Interceptable
  }
}

describe('web', () => {
  beforeEach((ctx) => {
    const url = 'http://127.0.0.1:8888'
    ctx.openmeter = new OpenMeterClient(url, 'token')
    const mockAgent = new MockAgent()
    mockAgent.disableNetConnect()
    setGlobalDispatcher(mockAgent)

    ctx.fetchMock = mockAgent.get(url)
  })

  afterEach((ctx) => {
    ctx.fetchMock.destroy()
  })

  describe('portal', () => {
    describe('query', () => {
      beforeEach((ctx) => {
        ctx.fetchMock
          .intercept({
            path: `/api/v1/portal/meters/m1/query`,
            query: {
              from: '2023-01-01T00:00:00Z',
              to: '2023-01-02T00:00:00Z',
            },
            method: 'GET',
            headers: {
              Accept: 'application/json',
            },
          })
          .reply(
            200,
            {
              from: mockMeterQueryRow.windowStart,
              to: mockMeterQueryRow.windowEnd,
              data: [mockMeterQueryRow],
            },
            {
              headers: {
                'Content-Type': 'application/json',
              },
            }
          )
      })

      it('should return meter query rows', async ({ openmeter }) => {
        const from = '2023-01-01T00:00:00Z'
        const to = '2023-01-02T00:00:00Z'
        const { data } = await openmeter.queryPortalMeter({
          meterSlug: 'm1',
          from,
          to,
        })
        expect(data).toEqual({
          from,
          to,
          data: [mockMeterQueryRow],
        })
      })
    })
  })
})
