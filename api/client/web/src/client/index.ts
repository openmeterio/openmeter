import { Fetcher } from 'openapi-typescript-fetch'
import { paths, components } from './openapi.js'

export type WindowSize = components['schemas']['WindowSize']
export type MeterQueryRow = components['schemas']['MeterQueryRow']
export type Problem = components['schemas']['UnexpectedProblemResponse']

export class OpenMeterClient {
  private readonly fetcher: ReturnType<typeof Fetcher.for<paths>>

  constructor(baseUrl: string, token: string) {
    this.fetcher = Fetcher.for<paths>()
    this.fetcher.configure({
      baseUrl,
      init: {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      },
    })
  }

  /** @description Query meters with portal */
  public get queryPortalMeter() {
    return this.fetcher
      .path('/api/v1/portal/meters/{meterSlug}/query')
      .method('get')
      .create()
  }
}
