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
Run the sample code to create Stripe entities and report usage on active subscriptions:

```sh
STRIPE_KEY=sk_test_... go run . setup
```

This will print the entities created:

```text
Stripe product created: https://dashboard.stripe.com/test/products/prod_xxx
Stripe price created: https://dashboard.stripe.com/test/prices/price_xxx
Stripe customer created: https://dashboard.stripe.com/test/customers/cus_xxx
Stripe subscription created: https://dashboard.stripe.com/test/subscriptions/sub_xxx
```

## 2. Report Usage

Ensure your OpenMeter runs. Check out the [quickstart guide](/quickstart) to see how to run OpenMeter.
Run the app to report usage:

```sh
STRIPE_KEY=sk_test_... go run . report
```

You should see the usage reported to Stripe:

```text
stripe_customer: cus_.., stripe_price: price_.., meter: m1, total_usage: 10.000000, from: 2023-06-07 14:00:00 -0700 PDT, to: 2023-07-07 14:00:00 -0700 PDT
```

If you visit the subscription on the Stripe dashboard, you should see usage reported on it.

## 3. Check out the sample code

Check out the sample code's `app.ts` file in this repo to see how to report usage to Stripe.

> **Note** OpenMeter collects usage in windows. The default window duration is hourly. In this example we round start and end dates to the closest OpenMeter windows.
> For example if a subscription's billing period ends at 1:45 PM, we will only report at 1 PM so usage occurring after 1PM will slip into the next billing cycle.
> It depends on your use-case what window size makes sense for your application. In OpenMeter you can configure window sizes per meter.
> **Note** In the sample code, we call Stripe's report API with `action=set` We do this to ensure idempotency so that no double reporting can happen.
