import { paths, components } from '../schemas/openapi.js'
import { BaseClient, OpenMeterConfig, RequestOptions } from './client.js'

export enum WindowSize {
  MINUTE = 'MINUTE',
  HOUR = 'HOUR',
  DAY = 'DAY',
}

export enum MeterAggregation {
  SUM = 'SUM',
  COUNT = 'COUNT',
  AVG = 'AVG',
  MIN = 'MIN',
  MAX = 'MAX',
}

export type MeterQueryParams = {
  /**
   * @description Subject(s) to filter by.
   * @example ["customer-1", "customer-2"]
   */
  subject?: string[]
  /**
   * @description Start date.
   * Must be aligned with the window size.
   * Inclusive.
   */
  from?: Date
  /**
   * @description End date.
   * Must be aligned with the window size.
   * Inclusive.
   */
  to?: Date
  /**
   * @description Window Size
   * If not specified, a single usage aggregate will be returned for the entirety of
   * the specified period for each subject and group.
   */
  windowSize?: WindowSizeType
  /**
   * @description The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones).
   * If not specified, the UTC timezone will be used.
   */
  windowTimeZone?: string
  /**
   * @description Group By
   * If not specified a single aggregate will be returned for each subject and time window.
   */
  groupBy?: string[]
}

export type MeterQueryResponse =
  paths['/api/v1/meters/{meterIdOrSlug}/query']['get']['responses']['200']['content']['application/json']

export type Meter = components['schemas']['Meter']
export type WindowSizeType = components['schemas']['WindowSize']

export class MetersClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * Get one meter by slug
   */
  public async get(slug: string, options?: RequestOptions): Promise<Meter> {
    return this.request<Meter>({
      method: 'GET',
      path: `/api/v1/meters/${slug}`,
      options,
    })
  }

  /**
   * List meters
   */
  public async list(options?: RequestOptions): Promise<Meter[]> {
    return this.request<Meter[]>({
      method: 'GET',
      path: `/api/v1/meters`,
      options,
    })
  }

  /**
   * Query a meter
   */
  public async query(
    slug: string,
    params?: MeterQueryParams,
    options?: RequestOptions
  ): Promise<MeterQueryResponse> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return this.request<MeterQueryResponse>({
      method: 'GET',
      path: `/api/v1/meters/${slug}/query`,
      searchParams,
      options,
    })
  }

  /**
   * List subjects of a meter
   */
  public async subjects(
    slug: string,
    options?: RequestOptions
  ): Promise<string[]> {
    return this.request<string[]>({
      method: 'GET',
      path: `/api/v1/meters/${slug}/subjects`,
      options,
    })
  }
}
