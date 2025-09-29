import createClient, {
  type Client,
  type ClientOptions,
  createQuerySerializer,
} from 'openapi-fetch'
import { Addons } from './addons.js'
import { Apps } from './apps.js'
import { Billing } from './billing.js'
import { Customers } from './customers.js'
import { Debug } from './debug.js'
import { Entitlements, EntitlementsV2 } from './entitlements.js'
import { Events } from './events.js'
import { Features } from './features.js'
import { Info } from './info.js'
import { Meters } from './meters.js'
import { Notifications } from './notifications.js'
import { Plans } from './plans.js'
import { Portal } from './portal.js'
import type { paths } from './schemas.js'
import { Subjects } from './subjects.js'
import { SubscriptionAddons } from './subscription-addons.js'
import { Subscriptions } from './subscriptions.js'
import { encodeDates } from './utils.js'

export * from './common.js'
export * from './schemas.js'

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

  public addons: Addons
  public apps: Apps
  public billing: Billing
  public customers: Customers
  public debug: Debug
  public entitlementsV1: Entitlements
  public entitlements: EntitlementsV2
  public events: Events
  public features: Features
  public info: Info
  public meters: Meters
  public notifications: Notifications
  public plans: Plans
  public portal: Portal
  public subjects: Subjects
  public subscriptionAddons: SubscriptionAddons
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

    this.addons = new Addons(this.client)
    this.apps = new Apps(this.client)
    this.billing = new Billing(this.client)
    this.customers = new Customers(this.client)
    this.debug = new Debug(this.client)
    this.entitlementsV1 = new Entitlements(this.client)
    this.entitlements = new EntitlementsV2(this.client)
    this.events = new Events(this.client)
    this.features = new Features(this.client)
    this.info = new Info(this.client)
    this.meters = new Meters(this.client)
    this.notifications = new Notifications(this.client)
    this.plans = new Plans(this.client)
    this.portal = new Portal(this.client)
    this.subjects = new Subjects(this.client)
    this.subscriptionAddons = new SubscriptionAddons(this.client)
    this.subscriptions = new Subscriptions(this.client)
  }
}
