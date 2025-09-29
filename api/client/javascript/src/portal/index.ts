import type { Client, ClientOptions } from 'openapi-fetch'
import createClient, { createQuerySerializer } from 'openapi-fetch'
import type { RequestOptions } from '../client/common.js'
import type { operations, paths } from '../client/schemas.js'
import { encodeDates, transformResponse } from '../client/utils.js'

/**
 * Portal Config
 */
export type Config = Pick<
  ClientOptions,
  'baseUrl' | 'headers' | 'fetch' | 'Request' | 'requestInitExt'
> & {
  portalToken: string
}

/**
 * OpenMeter Portal Client
 * Access to the customer portal.
 */
export class OpenMeter {
  private client: Client<paths, `${string}/${string}`>

  constructor(config: Config) {
    this.client = createClient<paths>({
      ...config,
      headers: {
        ...config.headers,
        Authorization: `Bearer ${config.portalToken}`,
      },
      querySerializer: (q) =>
        createQuerySerializer({
          array: {
            explode: true,
            style: 'form',
          },
          object: {
            explode: true,
            style: 'deepObject',
          },
        })(encodeDates(q)),
    })
  }

  /**
   * Query usage data for a meter by slug for customer portal.
   * This endpoint is publicly exposable to consumers.
   * @param meterSlug - The slug of the meter
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The meter data
   */
  public async query(
    meterSlug: string,
    query?: {
      /** @description Start date-time in RFC 3339 format. Inclusive. */
      from?: string | Date
      /** @description End date-time in RFC 3339 format. Inclusive. */
      to?: string | Date
      /** @description If not specified, a single usage aggregate will be returned for the entirety of the specified period for each subject and group. */
      windowSize?: 'MINUTE' | 'HOUR' | 'DAY'
      /** @description The value is the name of the time zone as defined in the IANA Time Zone Database (http://www.iana.org/time-zones). If not specified, the UTC timezone will be used. */
      windowTimeZone?: string
      /** @description Simple filter for group bys with exact match. */
      filterGroupBy?: Record<string, string>
      /** @description If not specified a single aggregate will be returned for each subject and time window. `subject` is a reserved group by value. */
      groupBy?: string[]
    },
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/portal/meters/{meterSlug}/query',
      {
        headers: {
          Accept: 'application/json',
        },
        params: {
          path: {
            meterSlug,
          },
          query,
        },
        ...options,
      },
    )

    return transformResponse(
      resp,
    ) as operations['queryPortalMeter']['responses']['200']['content']['application/json']
  }
}
