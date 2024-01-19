import { components } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'

export type PortalToken = components['schemas']['PortalToken']

export class PortalClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * Create portal token
   * Useful for creating a token sharable with your customer to query their own usage
   */
  public async createToken(
    token: {
      subject: string
      expiresAt?: Date
      allowedMeterSlugs?: string[]
    },
    options?: RequestOptions
  ): Promise<PortalToken> {
    return await this.request({
      path: '/api/v1/portal/tokens',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(token),
      options,
    })
  }

  /**
   * Invalidate portal token
   * @note OpenMeter Cloud only feature
   */
  public async invalidateTokens(
    invalidate: { subject?: string } = {},
    options?: RequestOptions
  ): Promise<void> {
    return await this.request({
      path: '/api/v1/portal/tokens/invalidate',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(invalidate),
      options,
    })
  }
}
