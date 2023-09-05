import assert from 'assert'

import 'dotenv/config'
import Stripe from 'stripe'

// Environment variables
assert.ok(
  process.env.STRIPE_KEY,
  'STRIPE_KEY environment variables is required'
)

const stripe = new Stripe(process.env.STRIPE_KEY, { apiVersion: '2022-11-15' })

// Meter in your config, we use it to map our price to this meter
var meterSlug = 'm2'

async function setup() {
  // Create a Stripe Product
  const product = await stripe.products.create({
    name: 'AI Tokens',
  })
  console.log(
    `Stripe product created: https://dashboard.stripe.com/test/products/${product.id}`
  )

  // Create a metered Stripe Price
  const price = await stripe.prices.create({
    product: product.id,
    // The meter ID this price belongs to
    currency: 'usd',
    recurring: {
      interval: 'month',
      usage_type: 'metered',
    },
    billing_scheme: 'per_unit',
    unit_amount: 1, // cent
  })
  console.log(
    `Stripe price created: https://dashboard.stripe.com/test/prices/${price.id}`
  )

  // Create a Stripe customer
  const customer = await stripe.customers.create({
    name: 'My Awesome Customer',
    metadata: {
      // Useful to map to internal ID
      external_key: 'my-awesome-user-id',
    },
  })
  console.log(
    `Stripe customer created: https://dashboard.stripe.com/test/customers/${customer.id}`
  )

  // Start a new Stripe subscription for customer with the price created above
  const subscription = await stripe.subscriptions.create({
    customer: customer.id,
    items: [
      {
        price: price.id,
        metadata: {
          om_meter_id: meterSlug,
        },
      },
    ],
  })
  console.log(
    `Stripe subscription created: https://dashboard.stripe.com/test/subscriptions/${subscription.id}`
  )
}

setup()
  .then(() => console.info('setup done'))
  .catch((err) => console.error('setup failed', err))
