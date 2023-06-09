import assert from 'assert'
import fs from 'fs'

import yml from 'yaml'
import Stripe from 'stripe'
import moment from 'moment'
import { OpenAPIClientAxios } from 'openapi-client-axios'

import { Client as OpenMeterClient } from './openapi'

// Environment variables
assert.ok(
  process.env.STRIPE_KEY,
  'STRIPE_KEY environment variables is required'
)

const stripe = new Stripe(process.env.STRIPE_KEY, { apiVersion: '2022-11-15' })
const openmeterApi = fs.readFileSync('../../api/openapi.yml', 'utf8')
const openmeter = await new OpenAPIClientAxios({
  definition: yml.parse(openmeterApi),
  withServer: { url: 'http://localhost:8888' },
}).initSync<OpenMeterClient>()

async function main() {
  // We round down period to closest windows as OpenMeter aggregates usage in windows.
  // Usage occuring between rounded down date and now will be attributed to the next billing period.
  const to = moment().startOf('hour').toDate()
  const from = moment(to).subtract(1, 'hour').toDate()
  const { data: subscriptions } = await stripe.subscriptions.list({
    status: 'active',
  })

  // Report usage for all active subscriptions
  for (const subscription of subscriptions) {
    // Skip subscriptions that started before `to`.
    if (moment(subscription.current_period_start).isBefore(to)) {
      continue
    }

    await reportUsage(subscription, from, to)
  }
}

main()
  .then(() => console.info('done'))
  .catch((err) => console.error('failed', err))

/**
 * Reports usage to Stripe
 */
async function reportUsage(
  subscription: Stripe.Subscription,
  from: Date,
  to: Date
) {
  for (const item of subscription.items.data) {
    // Skip non metered items
    if (item.price.recurring?.usage_type != 'metered') {
      continue
    }

    // Skip subscription item if doesn't have corresponding OpenMeter meter
    const meterId = item.metadata['om_meter_id']
    if (!meterId) {
      continue
    }

    // Query usage from OpenMeter for billing period
    const customerId =
      typeof subscription.customer === 'object'
        ? subscription.customer.id
        : subscription.customer
    const values = await openmeter.getValuesByMeterId({
      meterId,
      subject: customerId,
      from: from.toISOString(),
      to: to.toISOString(),
    })

    // Sum usage windows
    // TODO (pmarton): switch to OpenMeter aggregate API
    const total = values.data.values?.reduce(
      (total, { value }) => total + (value || 0),
      0
    )
    if (total === undefined) {
      continue
    }

    // Report usage to Stripe
    const timestamp = moment(to).unix()
    await stripe.subscriptionItems.createUsageRecord(
      item.id,
      {
        quantity: total,
        timestamp,
        action: 'set',
      },
      {
        // Ensures we only report once even if scripts runs multiple times.
        idempotencyKey: `${item.id}-${timestamp}`,
      }
    )

    // Debug log
    console.debug(
      `stripe_customer: ${customerId}, stripe_price: ${item.price.id}, meter: ${meterId}, total_usage: ${total}, from: ${from}, to: ${to}`
    )
  }
}
