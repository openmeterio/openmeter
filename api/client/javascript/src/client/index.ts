import createClient, {
  createQuerySerializer,
  type Client,
  type ClientOptions,
} from 'openapi-fetch'
import { Apps } from './apps.js'
import { Billing } from './billing.js'
import { Customers } from './customers.js'
import { Entitlements } from './entitlements.js'
import { Events } from './events.js'
import { Features } from './features.js'
import { Meters } from './meters.js'
import { Notifications } from './notifications.js'
import { Plans } from './plans.js'
import { Portal } from './portal.js'
import { Subjects } from './subjects.js'
import { Subscriptions } from './subscriptions.js'
import { encodeDates } from './utils.js'
import type { paths } from './schemas.js'

export type * from './schemas.js'

/**
 * OpenMeter Config
 */
export type Config = Pick<
  ClientOptions,
  'baseUrl' | 'headers' | 'fetch' | 'Request' | 'requestInitExt'
> &
  (
    | {
        apiKey?: string
      }
    | {
        baseUrl: 'https://openmeter.cloud'
        apiKey: string
      }
  )

/**
 * OpenMeter Client
 */
export class OpenMeter {
  public client: Client<paths, `${string}/${string}`>

  public apps: Apps
  public billing: Billing
  public customers: Customers
  public entitlements: Entitlements
  public events: Events
  public features: Features
  public meters: Meters
  public notifications: Notifications
  public plans: Plans
  public portal: Portal
  public subjects: Subjects
  public subscriptions: Subscriptions

  constructor(public config: Config) {
    this.client = createClient<paths>({
      ...config,
      headers: {
        ...config.headers,
        Authorization: config.apiKey ? `Bearer ${config.apiKey}` : undefined,
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

    this.apps = new Apps(this.client)
    this.billing = new Billing(this.client)
    this.customers = new Customers(this.client)
    this.entitlements = new Entitlements(this.client)
    this.events = new Events(this.client)
    this.features = new Features(this.client)
    this.meters = new Meters(this.client)
    this.notifications = new Notifications(this.client)
    this.plans = new Plans(this.client)
    this.portal = new Portal(this.client)
    this.subjects = new Subjects(this.client)
    this.subscriptions = new Subscriptions(this.client)
  }
}
