# Developing against a stripe test account

## Prerequisites

This guide assumes that you have the secret key available (starts with `sk_test`).

You have the stripe cli installed on your computer.

Before executing the following commands, please make sure that you are specifying the `--stripe-disable-webhook-registration` command line argument to the `Server` component (see (.vscode/launch.json)[../.vscode/launch.json]). This is required as OpenMeter would create a new webhook for your service on the stripe side. We are going to rely on the stripe cli instead, so that we don't clutter the test account's webhook list.

## Setting up stripe

Please start the server using the `--stripe-disable-webhook-registration` arguments.

Register the stripe app (replace the `sk_test_***` with your API key):

```sh
# curl -H "Content-Type: application/json" -X POST -d '{"apiKey": "sk_test_***", "name": "stripe"}' http://127.0.0.1:8000/api/v1/marketplace/listings/stripe/install/apikey
{"app":{"createdAt":"2025-02-03T14:13:23.276159Z","default":true,"description":"Stripe account ***","id":"01JK62H967WX20W9E6RNS05E80","listing":{"capabilities":[{"description":"Process payments","key":"stripe_collect_payment","name":"Payment","type":"collectPayments"},{"description":"Calculate tax for a payment","key":"stripe_calculate_tax","name":"Calculate Tax","type":"calculateTax"},{"description":"Invoice a customer","key":"stripe_invoice_customer","name":"Invoice Customer","type":"invoiceCustomers"}],"description":"Send invoices, calculate tax and collect payments.","name":"Stripe","type":"stripe"},"livemode":false,"maskedAPIKey":"sk_test_***QpT","metadata":null,"name":"stripe","status":"ready","stripeAccountId":"acct_1OSLCsE98Y117at0","type":"stripe","updatedAt":"2025-02-03T14:13:23.276165Z"},"defaultForCapabilityTypes":["calculateTax","invoiceCustomers","collectPayments"]}
```

## Receiving webhooks

> You might need to log in to your stripe test account using `stripe login -i --api-key sk_test_***` before using this command

To start receiving stripe webhooks you will need the stripe cli. Start it in event listening more:

```sh
stripe listen  -l --forward-to http://127.0.0.1:8000/api/v1/apps/01JK62H967WX20W9E6RNS05E80/stripe/webhook
```

The output should look something like this:
```
> Ready! You are using Stripe API Version [2025-01-27.acacia]. Your webhook signing secret is whsec_c67e0e4f97227eb66ca026a4fcd9124923035ff778a1945601d109756ca707f4 (^C to quit)
```

The last thing needs to be done is to update the signing secret for the stripe app:
```sh
psql -h 127.0.0.1 -U postgres postgres
UPDATE app_stripes set webhook_secret = 'whsec_c67e0e4f97227eb66ca026a4fcd9124923035ff778a1945601d109756ca707f4'
```

## Creating a stripe enabled customer

First let's create the customer:

```sh
curl -X POST http://127.0.0.1:8000/api/v1/customers \
  -H 'Content-Type: application/json' \
  --data-raw '
  {
  "name": "test",
  "usageAttribution": {
    "subjectKeys": [
      "test"
    ]
  },
  "currency": "USD",
  "billingAddress": {
    "country": "US"
  }
}'
```

Response:
```json
{"billingAddress":{"country":"US"},"createdAt":"2025-02-03T14:25:03.154256Z","currency":"USD","id":"01JK636MNJ1R9NKFA2VS62HVTF","metadata":null,"name":"test","updatedAt":"2025-02-03T14:25:03.154257Z","usageAttribution":{"subjectKeys":["test"]}}
```

Then let's bind it to a stripe customer:
```sh
curl -X PUT http://127.0.0.1:8000/api/v1/customers/01JK636MNJ1R9NKFA2VS62HVTF/apps \
  -H 'Content-Type: application/json' \
  --data-raw '[
  {
    "type": "stripe",
    "stripeCustomerId": "cus_****"
  }
]
  '
```
