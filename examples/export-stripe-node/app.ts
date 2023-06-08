import assert from 'assert'
import fs from 'fs'

import yml from 'yaml'
import express from 'express'
import bodyParser from 'body-parser'
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

const app = express()

// Match the raw body to content type application/json
app.post(
  '/webhook',
  bodyParser.json({ type: 'application/json' }),
  async (request, response) => {
    // in a production app you want to check if signature matches
    // see: https://stripe.com/docs/payments/handling-payment-events#signature-checking
    const event = request.body

    // Handle the event
    switch (event.type) {
      case 'customer.subscription.updated':
        // Event type is `subscription`
        // See: https://stripe.com/docs/api/events/types#event_types-customer.subscription.updated
        const subscription = event.data.object as Stripe.Subscription
        const previousAttributes = event.data.previous_attributes

        // Report usage if billing period changed
        if (previousAttributes.current_period_start) {
          const from = previousAttributes.current_period_start
          const to = subscription.current_period_start
          await reportUsage(event.id, subscription, from, to)
        }

        break
    }

    // Return a 200 response to acknowledge receipt of the event
    response.json({ received: true })
  }
)

/**
 * Reports usage to Stripe
 */
async function reportUsage(
  eventId: string,
  subscription: Stripe.Subscription,
  fromUnix: number,
  toUnix: number
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

    // We round down period to closest windows as OpenMeter aggregates usage in windows.
    // Usage occuring between rounded down `current_period_end` and `current_period_end` will be attributed to the next billing period.
    const from = moment.unix(fromUnix).startOf('minute').toISOString()
    const to = moment.unix(toUnix).startOf('minute').toISOString()

    // Query usage from OpenMeter for billing period
    const customerId =
      typeof subscription.customer === 'object'
        ? subscription.customer.id
        : subscription.customer
    const values = await openmeter.getValuesByMeterId({
      meterId,
      subject: customerId,
      from,
      to,
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
    // We use `action=set` so even if this webhook get called multiple time
    // we still end up with only one usage record for the same period.
    await stripe.subscriptionItems.createUsageRecord(
      item.id,
      {
        quantity: total,
        timestamp: toUnix - 1,
        action: 'set',
      },
      {
        idempotencyKey: eventId,
      }
    )

    // Debug log
    console.debug(
      `stripe_customer: ${customerId}, stripe_price: ${item.price.id}, meter: ${meterId}, total_usage: ${total}, from: ${from}, to: ${to}`
    )
  }
}

// Start server
app.listen(4242, () => console.log('Running on port 4242'))
