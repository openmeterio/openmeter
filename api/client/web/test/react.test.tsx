import { renderHook } from '@testing-library/react'
import React from 'react'
import { Interceptable, MockAgent, setGlobalDispatcher } from 'undici'
import {
  afterAll,
  afterEach,
  beforeAll,
  beforeEach,
  describe,
  expect,
  it,
  vi,
} from 'vitest'
import { OpenMeterClient } from '../dist'
import { useOpenMeter, OpenMeterProvider } from '../dist/react'

type TestContext = {
  openmeter: OpenMeterClient
  fetchMock: Interceptable
}

describe('react', () => {
  beforeAll(() => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
  })

  afterAll(() => {
    vi.resetAllMocks()
  })

  beforeEach<TestContext>((ctx) => {
    const url = 'http://127.0.0.1:8888'
    ctx.openmeter = new OpenMeterClient(url, 'token')
    const mockAgent = new MockAgent()
    mockAgent.disableNetConnect()
    setGlobalDispatcher(mockAgent)

    ctx.fetchMock = mockAgent.get(url)
  })

  afterEach<TestContext>((ctx) => {
    ctx.fetchMock.destroy()
  })

  describe('useOpenMeter', () => {
    it('should throw an error when not wrapped in a provider', () => {
      expect(() => {
        renderHook(() => useOpenMeter())
      }).toThrow('useOpenMeter must be used within an OpenMeterProvider')
    })

    it('should throw an error when no url is provided', () => {
      const wrapper = ({ children }: { children?: React.ReactNode }) => (
        <OpenMeterProvider url="">{children}</OpenMeterProvider>
      )
      expect(() => {
        renderHook(() => useOpenMeter(), { wrapper })
      }).toThrow('OpenMeterProvider must be initialized with a url')
    })

    it('should return null when no token is provided', () => {
      const wrapper = ({ children }: { children?: React.ReactNode }) => (
        <OpenMeterProvider url="foo">{children}</OpenMeterProvider>
      )
      const { result } = renderHook(() => useOpenMeter(), { wrapper })
      expect(result.current).toBeNull()
    })

    it('should return a client', () => {
      const wrapper = ({ children }: { children?: React.ReactNode }) => (
        <OpenMeterProvider url="foo" token="token">
          {children}
        </OpenMeterProvider>
      )
      const { result } = renderHook(() => useOpenMeter(), { wrapper })
      expect(result.current).toBeInstanceOf(OpenMeterClient)
    })

    it<TestContext>('should initialize the client with the provided url and token', async (ctx) => {
      const wrapper = ({ children }: { children?: React.ReactNode }) => (
        <OpenMeterProvider url="http://127.0.0.1:8888" token="token">
          {children}
        </OpenMeterProvider>
      )
      const { result } = renderHook(() => useOpenMeter(), { wrapper })
      ctx.fetchMock
        .intercept({
          path: '/api/v1/portal/meters/m1/query',
          headers: {
            Accept: 'application/json',
            Authorization: 'Bearer token',
          },
        })
        .reply(200, { data: [] })
      result.current?.queryPortalMeter({ meterSlug: 'm1' })
    })
  })
})
