import createClient, {
  createQuerySerializer,
  type Client,
  type ClientOptions,
} from 'openapi-fetch'
import { Apps } from './src/apps.js'
import { Billing } from './src/billing.js'
import { Entitlements } from './src/entitlements.js'
import { Events } from './src/events.js'
import { Features } from './src/features.js'
import { Meters } from './src/meters.js'
import { Notifications } from './src/notifications.js'
import { Plans } from './src/plans.js'
import { Subjects } from './src/subjects.js'
import { encodeDates } from './src/utils.js'
import type { paths } from './src/schemas.js'

export type * from './src/schemas.js'

export interface Config
  extends Pick<
    ClientOptions,
    'baseUrl' | 'headers' | 'fetch' | 'Request' | 'requestInitExt'
  > {
  apiKey?: string
}

export class OpenMeter {
  config: Config
  public client: Client<paths, `${string}/${string}`>

  public apps: Apps
  public billing: Billing
  public entitlements: Entitlements
  public events: Events
  public features: Features
  public meters: Meters
  public notifications: Notifications
  public plans: Plans
  public subjects: Subjects

  constructor(config: Config) {
    this.config = config
    this.client = createClient<paths>({
      ...config,
      headers: {
        ...config.headers,
        Authorization: config.apiKey ? `Bearer ${config.apiKey}` : undefined,
      },
      querySerializer: (q) =>
        createQuerySerializer({
          array: {
            style: 'form',
            explode: true,
          },
          object: {
            style: 'deepObject',
            explode: true,
          },
        })(encodeDates(q)),
    })

    this.apps = new Apps(this.client)
    this.billing = new Billing(this.client)
    this.entitlements = new Entitlements(this.client)
    this.events = new Events(this.client)
    this.features = new Features(this.client)
    this.meters = new Meters(this.client)
    this.notifications = new Notifications(this.client)
    this.plans = new Plans(this.client)
    this.subjects = new Subjects(this.client)
  }
}
