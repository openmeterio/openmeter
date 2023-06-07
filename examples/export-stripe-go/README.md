# OpenMeter Stripe Example

This OpenMeter example will show how you can report metered usage to Stripe; this is needed if you use Usage-Based Pricing (UBP), where customers pay after their reported consumption.
As Stripe's usage report API has rate limits you can't use it directly for metering; This is where OpenMeter comes to the rescue to report aggregated usage to Stripe.

You can read more about Stripe's Usage-Based Pricing features [here](https://stripe.com/docs/products-prices/pricing-models#usage-based-pricing).

## Our Example

In this example, we integrate usage reporting with Stripe for an imaginary serverless product called Epsilon which charges their customer after execution duration.
To report usage to Stripe, we have two options:

1. Periodic usage reporting (hourly, daily, etc.)
1. Report usage before invoice creation via webhook

In your app, you can combine both approaches to have an up-to-date view of customer usage in Stripe.

In our example, we will report usage via [webhook](https://stripe.com/docs/billing/subscriptions/webhooks) at subscription updates just before invoice creation happens. We will use Stripe's `customer.subscription.updated` event, which fires when a subscription resets its billing cycle. When we receive this subscription update event, we will query the customer's usage from OpenMeter for the subscription period and report it to Stripe. Stripe will then create an invoice with the reported usage and bill the customer accordingly.

## 1. Setting Up Stripe

In this example we will create a metered product and price with a monthly recurring billing period, priced at $0.1 per unit.
Login to stripe and start forwarding webhook to your local machine ([read more](https://stripe.com/docs/webhooks/test)):

```sh
stripe login
stripe listen --forward-to localhost:4242/webhook
```

Grab yout the webhook secret (`whsec...`) printed out and your Stripe test key [here](https://dashboard.stripe.com/test/developers):
Run the sample code this repo in an another terminal with your keys as:

```sh
STRIPE_KEY=sk_test_... STRIPE_WEBHOOK_SECRET=whsec_... go run .
```

This will create product, price, customer and subscription entities in your Stripe test account and print out the links to the them:

```text
Stripe product created: https://dashboard.stripe.com/test/products/prod_xxx
Stripe price created: https://dashboard.stripe.com/test/prices/price_xxx
Stripe customer created: https://dashboard.stripe.com/test/customers/cus_xxx
Stripe subscription created: https://dashboard.stripe.com/test/subscriptions/sub_xxx
```

## 2. Report Usage

Ensure your OpenMeter runs. Check out the [quickstart guide](/quickstart) to see how to run OpenMeter.
When our sample server receives the `customer.subscription.updated` event, we will validate the signature if this event comes from Stripe, then we will report usage for the subscription period.

To trigger invoice generation visit the link printed in your terminal, should look like:
<https://dashboard.stripe.com/test/subscriptions/sub_xxx>

1. Open subscription on Stripe UI.
1. Click `Actions` then `Update Subscription` edit on the UI.
1. Check-in `Reset billing cycle`
1. Press `Update Subscription`.

Go back to your terminal, you should see that the server handled the webhook and printed out the result.

```text
stripe_customer: cus_.., stripe_price: price_.., meter: m1, total_usage: 10.000000, from: 2023-06-07 14:00:00 -0700 PDT, to: 2023-07-07 14:00:00 -0700 PDT
```

If you visit the subscription on the Stripe dashboard again, you will see an invoice geenrated with the reported usage.

## 3. Check out the sample code

Check out the sample code's `server.go` file in this repo to see how to handle webhook and report OpenMeter usage for `customer.subscription.updated` events.
The sample code does the following:

1. Validates webhook signature
1. Get's usage from OpenMeter
1. Reports usage to Stripe

> **Note** OpenMeter collects usage in windows. The default window duration is hourly. In this example we round down billign period start and end to the closest OpenMeter windows.
For example if a subscription's billing period ends at 1:45 PM, we will round it down to 1 PM. Usage occuring after 1PM will slip into the next billing cycle.
It depends on your use-case what window size makes sense for your application. In OpenMeter you can configure window sizes per meter.

> **Note** In the sample code, we call Stripe's report API with `action=set,` and use the subscriptions' current period start as reporting timestamp. We do this to ensure idempotency so that no double reporting can happen. We also add one to the timestamp because Stripe doesn't allow us to report usage on the exact period start and end time. In your app, you may choose a more sophisticated logic to report daily usage. Remember that Stripe subscriptions can end before the current billing period ends, so your last day can be partial.
