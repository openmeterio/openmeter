# OpenMeter Stripe Example

This OpenMeter example will show how you can report metered usage to Stripe; this is needed if you use Usage-Based Pricing (UBP), where customers pay after their reported consumption.

You can't use Stripe's usage report API for metering comes with rate limits, so you can't use it directly for metering;  This is where OpenMeter comes to the rescue to report aggregated usage to Stripe.

You can read more about Stripe's Usage-Based Pricing features [here](https://stripe.com/docs/products-prices/pricing-models#usage-based-pricing).

## Our Example

In this example, we integrate usage reporting with Stripe for an imaginary serverless product called Epsilon which charges their customer $0.20 per 1M requests.

To report usage to Stripe, we have two options:

1. Periodic usage reporting (hourly, daily, etc.)
1. Report usage before invoice creation via webhook

In your app, you can combine both approaches to have an up-to-date view of customer usage in Stripe.

In our example, we will report usage via [webhook](https://stripe.com/docs/billing/subscriptions/webhooks) at subscription updates just before invoice creation happens. We will use Stripe's `customer.subscription.updated` event, which fires when a subscription resets its billing cycle. When we receive this subscription update event, we will query the customer's usage from OpenMeter for the subscription period and report it to Stripe. Stripe will then create an invoice with the reported usage and bill the customer accordingly.

## 1. Setting Up Stripe

## 1.1. Create a Product and Price

On the Stripe UI, go to [Create Product](https://dashboard.stripe.com/test/products/create), and for usage-based, choose `Pricing model` to be either `Package,` `Graduated` or `Volume`:

1. In our example above, choose `Package` and define $0.20 per 1,000,000 units.
1. Pick `Recurring` from the `Recurring` and `One-time` options
1. Put a checkmark on `Usage is metered.`
1. The default `Sum of usage values during period` works for our example
1. Press `Save product`
1. Copy out the price's API ID (`price_xxx`) after redirect; it's under the `Pricing` section

In the background, Stripe will create a Product and corresponding Price entity. You can also set up products and prices via the Stripe API.

## 1.2. Setup Webhook

[Create Webhook](https://dashboard.stripe.com/test/webhooks/create) Select the `customer.subscription.updated` event and point to your server route.
You can trigger or forward webhooks locally; see [Test webhooks](https://stripe.com/docs/webhooks/test).

## 2. Report Usage

When our server receives the `customer.subscription.updated` event, we will validate the signature if this event comes from Stripe, then we will report usage for the subscription period.

Check out the [quickstart guide](/quickstart) to see how to run OpenMeter.

Check out the sample code in this repo to see how to handle webhook and report OpenMeter usage for `customer.subscription.updated`.
The sample code does the following:

1. Validates webhook signature
1. Get's usage from OpenMeter
1. Reports usage to Stripe

In the sample code, we call Stripe's report API with `action=set,` and use the subscriptions' current period start as reporting timestamp. We do this to ensure idempotency so that no double reporting can happen. We also add one to the timestamp because Stripe doesn't allow us to report usage on the exact period start and end time. In your app, you may choose a more sophisticated logic to report daily usage. Remember that Stripe subscriptions can end before the current billing period ends, so your last day can be partial.
