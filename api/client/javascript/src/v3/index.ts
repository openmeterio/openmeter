import createClient, { type Client, createQuerySerializer } from 'openapi-fetch'
import type { Config } from '../client/index.js'
import { encodeDates } from '../client/utils.js'
import { Addons } from './addons.js'
import { Apps } from './apps.js'
import { BillingProfiles } from './billing-profiles.js'
import { Currencies } from './currencies.js'
import { Customers } from './customers.js'
import { Defaults } from './defaults.js'
import { Events } from './events.js'
import { Features } from './features.js'
import { Governance } from './governance.js'
import { LlmCost } from './llm-cost.js'
import { Meters } from './meters.js'
import { Plans } from './plans.js'
import type { paths } from './schemas.js'
import { Subscriptions } from './subscriptions.js'
import { TaxCodes } from './tax-codes.js'

/**
 * v3 compatibility shim.
 *
 * Wraps a dedicated `openapi-fetch` client pointed at the v3 API surface and
 * exposes the v3 resource classes. Reached via `OpenMeter.v3`. This is an
 * interim fallback to the generated v3 SDK in `api/spec/`; it surfaces the
 * snake_case wire shapes verbatim (no field renaming).
 *
 * Base URL: the v3 servers are `<host>/api/v3` and paths are `/openmeter/...`,
 * whereas the v1 `Config.baseUrl` is the bare host. So the v3 client targets
 * `${config.baseUrl}/api/v3`. Auth header and query serialization are reused
 * from the v1 client (deepObject objects cover the `page[...]` pagination
 * params for free).
 */
export class V3 {
  public client: Client<paths, `${string}/${string}`>

  public addons: Addons
  public plans: Plans
  public features: Features
  public customers: Customers
  public subscriptions: Subscriptions
  public meters: Meters
  public events: Events
  public billingProfiles: BillingProfiles
  public currencies: Currencies
  public taxCodes: TaxCodes
  public llmCost: LlmCost
  public apps: Apps
  public governance: Governance
  public defaults: Defaults

  constructor(config: Config) {
    const baseUrl = `${(config.baseUrl ?? '').replace(/\/$/, '')}/api/v3`

    this.client = createClient<paths>({
      ...config,
      baseUrl,
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
    this.plans = new Plans(this.client)
    this.features = new Features(this.client)
    this.customers = new Customers(this.client)
    this.subscriptions = new Subscriptions(this.client)
    this.meters = new Meters(this.client)
    this.events = new Events(this.client)
    this.billingProfiles = new BillingProfiles(this.client)
    this.currencies = new Currencies(this.client)
    this.taxCodes = new TaxCodes(this.client)
    this.llmCost = new LlmCost(this.client)
    this.apps = new Apps(this.client)
    this.governance = new Governance(this.client)
    this.defaults = new Defaults(this.client)
  }
}
