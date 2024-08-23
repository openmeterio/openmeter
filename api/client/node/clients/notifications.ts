import { components, operations } from '../schemas/openapi.js'
import { RequestOptions, BaseClient, OpenMeterConfig } from './client.js'

export type NotificationChannel = components['schemas']['NotificationChannel']
export type NotificationChannelCreateRequest =
  components['schemas']['NotificationChannelCreateRequest']
export type ListNotificationChannelsQueryParams =
  operations['listNotificationChannels']['parameters']['query']
export type NotificationChannelsResponse =
  components['schemas']['NotificationChannelsResponse']

export type NotificationRule = components['schemas']['NotificationRule']
export type NotificationRuleCreateRequest =
  components['schemas']['NotificationRuleCreateRequest']
export type ListNotificationRulesQueryParams =
  operations['listNotificationRules']['parameters']['query']
export type NotificationRulesResponse =
  components['schemas']['NotificationRulesResponse']

export type NotificationEvent = components['schemas']['NotificationEvent']
export type ListNotificationEventsQueryParams =
  operations['listNotificationEvents']['parameters']['query']
export type NotificationEventsResponse =
  components['schemas']['NotificationEventsResponse']

export class NotificationClient extends BaseClient {
  public channels: NotificationChannelsClient
  public rules: NotificationRulesClient
  public events: NotificationEventsClient

  constructor(config: OpenMeterConfig) {
    super(config)

    this.channels = new NotificationChannelsClient(config)
    this.rules = new NotificationRulesClient(config)
    this.events = new NotificationEventsClient(config)
  }
}

class NotificationChannelsClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * List notification channels.
   * @example
   * const channels = await openmeter.notification.channels.list()
   */
  public async list(
    params?: ListNotificationChannelsQueryParams,
    options?: RequestOptions
  ): Promise<NotificationChannelsResponse> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: '/api/v1/notification/channels',
      method: 'GET',
      searchParams,
      options,
    })
  }

  /**
   * Get notification channel.
   * @example
   * const channel = await openmeter.notification.channels.get('01J5Z602369ZDS9J60N3DV7SGE')
   */
  public async get(
    id: string,
    options?: RequestOptions
  ): Promise<NotificationChannel> {
    return await this.request({
      path: `/api/v1/notification/channels/${id}`,
      method: 'GET',
      options,
    })
  }

  /**
   * Delete notification channel.
   * @example
   * await openmeter.notification.channels.delete('01J5Z602369ZDS9J60N3DV7SGE')
   */
  public async delete(id: string, options?: RequestOptions): Promise<void> {
    await this.request({
      path: `/api/v1/notification/channels/${id}`,
      method: 'DELETE',
      options,
    })
  }

  /**
   * Create notification channel.
   * @example
   * const channel = await openmeter.notification.channels.create({
   *   type: 'WEBHOOK',
   *   name: 'My webhook channel',
   *   url: 'https://example.com/webhook',
   *   customHeaders: {
   *    'User-Agent': 'OpenMeter'
   *   },
   * })
   */
  public async create(
    body: NotificationChannelCreateRequest,
    options?: RequestOptions
  ): Promise<NotificationChannel> {
    return await this.request({
      path: '/api/v1/notification/channels',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
      options,
    })
  }
}

class NotificationRulesClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * List notification rules.
   * @example
   * const rules = await openmeter.notification.rules.list()
   */
  public async list(
    params?: ListNotificationRulesQueryParams,
    options?: RequestOptions
  ): Promise<NotificationRulesResponse> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: '/api/v1/notification/rules',
      method: 'GET',
      searchParams,
      options,
    })
  }

  /**
   * Get notification rule.
   * @example
   * const rule = await openmeter.notification.rules.get('01J5Z602369ZDS9J60N3DV7SGE')
   */
  public async get(
    id: string,
    options?: RequestOptions
  ): Promise<NotificationChannel> {
    return await this.request({
      path: `/api/v1/notification/rules/${id}`,
      method: 'GET',
      options,
    })
  }

  /**
   * Delete notification rule.
   * @example
   * await openmeter.notification.rules.delete('01J5Z602369ZDS9J60N3DV7SGE')
   */
  public async delete(id: string, options?: RequestOptions): Promise<void> {
    await this.request({
      path: `/api/v1/notification/rules/${id}`,
      method: 'DELETE',
      options,
    })
  }

  /**
   * Create notification rule.
   * @example
   * const rule = await openmeter.notification.rules.create({
   *   type: 'entitlements.balance.threshold',
   *   name: 'My rule',
   *   channels: ['01J5Z602369ZDS9J60N3DV7SGE'],
   *   thresholds: [{ value: 90, type: 'PERCENT' }],
   * })
   */
  public async create(
    body: NotificationRuleCreateRequest,
    options?: RequestOptions
  ): Promise<NotificationChannel> {
    return await this.request({
      path: '/api/v1/notification/rules',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
      options,
    })
  }
}

class NotificationEventsClient extends BaseClient {
  constructor(config: OpenMeterConfig) {
    super(config)
  }

  /**
   * List notification events.
   * @example
   * const events = await openmeter.notification.events.list()
   */
  public async list(
    params?: ListNotificationEventsQueryParams,
    options?: RequestOptions
  ): Promise<NotificationEventsResponse> {
    const searchParams = params
      ? BaseClient.toURLSearchParams(params)
      : undefined
    return await this.request({
      path: '/api/v1/notification/events',
      method: 'GET',
      searchParams,
      options,
    })
  }

  /**
   * Get notification event.
   * @example
   * const event = await openmeter.notification.events.get('01J5Z602369ZDS9J60N3DV7SGE')
   */
  public async get(
    id: string,
    options?: RequestOptions
  ): Promise<NotificationEvent> {
    return await this.request({
      path: `/api/v1/notification/events/${id}`,
      method: 'GET',
      options,
    })
  }
}
