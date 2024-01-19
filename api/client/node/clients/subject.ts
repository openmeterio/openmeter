import { components } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'

export type Subject = components['schemas']['Subject']

export class SubjectClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * Upsert subject
   * Useful to map display name and metadata to subjects
   * @note OpenMeter Cloud only feature
   */
  public async upsert(
    subject: Omit<Subject, 'id'>,
    options?: RequestOptions
  ): Promise<Subject> {
    return await this.request({
      path: '/api/v1/subjects',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(subject),
      options,
    })
  }

  /**
   * List subjects
   * @note OpenMeter Cloud only feature
   */
  public async list(options?: RequestOptions): Promise<void> {
    return await this.request({
      path: '/api/v1/subjects',
      method: 'GET',
      options,
    })
  }
}
