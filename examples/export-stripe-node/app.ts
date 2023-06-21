import assert from 'assert'

import 'dotenv/config'
import moment from 'moment'
import Stripe from 'stripe'
import { OpenMeter, WindowSize } from '@openmeter/sdk'

// Environment variables
assert.ok(
    process.env.STRIPE_KEY,
    'STRIPE_KEY environment variables is required'
)

// Special flag to trigger this example to report all usage occured in this month
const reportAll = process.env.REPORT_ALL === 'true'

const stripe = new Stripe(process.env.STRIPE_KEY, { apiVersion: '2022-11-15' })
const openmeter = new OpenMeter({ baseUrl: 'http://localhost:8888' })

// In a real app you will probably report hourly or daily and run this script at the same frequency via cron or workflow management
const reportingFrequency = 'hour'

async function main() {
    // We round down period to closest windows as OpenMeter aggregates usage in windows.
    // Usage occuring between rounded down date and now will be attributed to the next billing period.
    let to = moment().startOf(reportingFrequency).toDate()
    let from = moment(to).subtract(1, reportingFrequency).toDate()

    // With special flag we report entire day, useful for testing
    if (reportAll) {
        from = moment(to).startOf('day').toDate()
        to = moment(from).add(1, 'day').toDate()
    }

    // List all stripe active subscriptions and expand customer object
    const { data: subscriptions } = await stripe.subscriptions.list({
        status: 'active',
        expand: ['data.customer'],
    })

    // Report usage for all active subscriptions
    for (const subscription of subscriptions) {
        // Skip subscriptions that started after `to`.
        if (moment(subscription.current_period_start).isAfter(to)) {
            continue
        }
        // Type checking for TypeScript
        if (!isCustomer(subscription.customer)) {
            throw new TypeError('Must be customer with expand option')
        }

        await reportUsage(subscription.customer, subscription, from, to)
    }
}

main()
    .then(() => console.info('done'))
    .catch((err) => console.error('failed', err))

/**
 * Reports usage to Stripe
 */
async function reportUsage(
    customer: Stripe.Customer,
    subscription: Stripe.Subscription,
    from: Date,
    to: Date
) {
    // Skip customer item if doesn't have corresponding key
    const subject = customer.metadata['external_key']
    if (!subject) {
        return
    }

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
        const resp = await openmeter.getValuesByMeterId(meterId, subject, from.toISOString(), to.toISOString(), WindowSize.HOUR)

        // Sum usage windows
        // TODO (pmarton): switch to OpenMeter aggregate API
        const total = resp.data.reduce(
            (total, { value }) => total + (value || 0),
            0
        )
        if (total === undefined) {
            continue
        }

        // Report usage to Stripe
        let reportingTimestamp: number | 'now' = moment(to).unix()
        if (reportAll) {
            reportingTimestamp = moment().unix()
        }
        await stripe.subscriptionItems.createUsageRecord(
            item.id,
            {
                quantity: total,
                timestamp: reportingTimestamp,
                action: 'set',
            },
            {
                // Ensures we only report once even if scripts runs multiple times.
                idempotencyKey: `${item.id}-${reportingTimestamp}`,
            }
        )

        // Debug log
        console.debug(
            `stripe_customer: ${customer.id}, stripe_price: ${item.price.id}, subject: ${subject}, meter: ${meterId}, total_usage: ${total}, from: ${from}, to: ${to}`
        )
    }
}

// Typeguard that returns true if customer is a `Stripe.Customer`
function isCustomer(
    customer: string | Stripe.Customer | Stripe.DeletedCustomer
): customer is Stripe.Customer {
    if (typeof customer === 'object' && !customer.deleted) {
        return true
    }
    return false
}

function isError(err: any): err is Error {
    return err instanceof Error
}
