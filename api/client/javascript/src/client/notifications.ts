import type { Client } from 'openapi-fetch'
import type { RequestOptions } from './common.js'
import type {
  NotificationChannel,
  NotificationRuleCreateRequest,
  operations,
  paths,
} from './schemas.js'
import { transformResponse } from './utils.js'

/**
 * Notifications
 * @description Notifications provide automated triggers when specific entitlement balances and usage thresholds are reached, ensuring that your customers and sales teams are always informed. Notify customers and internal teams when specific conditions are met, like reaching 75%, 100%, and 150% of their monthly usage allowance.
 */
export class Notifications {
  public channels: NotificationChannels
  public rules: NotificationRules
  public events: NotificationEvents

  constructor(private client: Client<paths, `${string}/${string}`>) {
    this.channels = new NotificationChannels(this.client)
    this.rules = new NotificationRules(this.client)
    this.events = new NotificationEvents(this.client)
  }
}

/**
 * Notification Channels
 * @description Notification channels are the destinations for notifications.
 */
export class NotificationChannels {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a notification channel
   * @param notification - The notification to create
   * @param signal - An optional abort signal
   * @returns The created notification
   */
  public async create(
    notification: NotificationChannel,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/api/v1/notification/channels', {
      body: notification,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a notification channel by ID
   * @param id - The ID of the notification channel
   * @param signal - An optional abort signal
   * @returns The notification channel
   */
  public async get(
    id: operations['getNotificationChannel']['parameters']['path']['channelId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/notification/channels/{channelId}',
      {
        params: {
          path: {
            channelId: id,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * Update a notification channel
   * @param id - The ID of the notification channel
   * @param notification - The notification to update
   * @param signal - An optional abort signal
   * @returns The updated notification
   */
  public async update(
    id: operations['updateNotificationChannel']['parameters']['path']['channelId'],
    notification: NotificationChannel,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT(
      '/api/v1/notification/channels/{channelId}',
      {
        body: notification,
        params: {
          path: {
            channelId: id,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List notification channels
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The list of notification channels
   */
  public async list(
    query?: operations['listNotificationChannels']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/notification/channels', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a notification channel
   * @param id - The ID of the notification channel
   * @param signal - An optional abort signal
   * @returns The deleted notification
   */
  public async delete(
    id: operations['deleteNotificationChannel']['parameters']['path']['channelId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/notification/channels/{channelId}',
      {
        params: {
          path: {
            channelId: id,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}

/**
 * Notification Rules
 * @description Notification rules are the conditions that trigger notifications.
 */
export class NotificationRules {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Create a notification rule
   * @param rule - The rule to create
   * @param signal - An optional abort signal
   * @returns The created rule
   */
  public async create(
    rule: NotificationRuleCreateRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST('/api/v1/notification/rules', {
      body: rule,
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Get a notification rule by ID
   * @param id - The ID of the notification rule
   * @param signal - An optional abort signal
   * @returns The notification rule
   */
  public async get(
    id: operations['getNotificationRule']['parameters']['path']['ruleId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/notification/rules/{ruleId}', {
      params: {
        path: {
          ruleId: id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Update a notification rule
   * @param id - The ID of the notification rule
   * @param rule - The rule to update
   * @param signal - An optional abort signal
   * @returns The updated rule
   */
  public async update(
    id: operations['updateNotificationRule']['parameters']['path']['ruleId'],
    rule: NotificationRuleCreateRequest,
    options?: RequestOptions,
  ) {
    const resp = await this.client.PUT('/api/v1/notification/rules/{ruleId}', {
      body: rule,
      params: {
        path: {
          ruleId: id,
        },
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * List notification rules
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The list of notification rules
   */
  public async list(
    query?: operations['listNotificationRules']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/notification/rules', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Delete a notification rule
   * @param id - The ID of the notification rule
   * @param signal - An optional abort signal
   * @returns The deleted notification
   */
  public async delete(
    id: operations['deleteNotificationRule']['parameters']['path']['ruleId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.DELETE(
      '/api/v1/notification/rules/{ruleId}',
      {
        params: {
          path: {
            ruleId: id,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}

/**
 * Notification Events
 * @description Notification events are the events that trigger notifications.
 */
export class NotificationEvents {
  constructor(private client: Client<paths, `${string}/${string}`>) {}

  /**
   * Get a notification event by ID
   * @param id - The ID of the notification event
   * @param signal - An optional abort signal
   * @returns The notification event
   */
  public async get(
    id: operations['getNotificationEvent']['parameters']['path']['eventId'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET(
      '/api/v1/notification/events/{eventId}',
      {
        params: {
          path: {
            eventId: id,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }

  /**
   * List notification events
   * @param query - The query parameters
   * @param signal - An optional abort signal
   * @returns The list of notification events
   */
  public async list(
    query?: operations['listNotificationEvents']['parameters']['query'],
    options?: RequestOptions,
  ) {
    const resp = await this.client.GET('/api/v1/notification/events', {
      params: {
        query,
      },
      ...options,
    })

    return transformResponse(resp)
  }

  /**
   * Resend a notification event
   * @description Resend a notification event that has already been sent.
   * @param id - The ID of the notification event
   * @param channels - The channels to resend the notification event to, if not provided it will resend to all channels
   * @param signal - An optional abort signal
   * @returns The resent notification event
   */
  public async resend(
    id: operations['resendNotificationEvent']['parameters']['path']['eventId'],
    body: operations['resendNotificationEvent']['requestBody']['content']['application/json'] = {},
    options?: RequestOptions,
  ) {
    const resp = await this.client.POST(
      '/api/v1/notification/events/{eventId}/resend',
      {
        body,
        params: {
          path: {
            eventId: id,
          },
        },
        ...options,
      },
    )

    return transformResponse(resp)
  }
}
