# OpenMeter SDK

TypeScript client for the OpenMeter API — usage metering and billing for
AI and DevTool companies. This package is generated from the OpenMeter
TypeSpec definitions and ships fully-typed request and response models.

> [!IMPORTANT]
> This SDK is a work in progress.
>
> This SDK targets the [OpenMeter API v3](https://openmeter.io/docs/api/v3),
> a rewrite of the OpenMeter API following AIP (API Improvement Proposal)
> standardization.

## Table of Contents

- [Installation](#installation)
- [Initialization](#initialization)
- [Usage](#usage)
- [Available Resources and Operations](#available-resources-and-operations)
  - [Events](#events)
  - [Meters](#meters)
  - [Customers](#customers)
  - [Entitlements](#entitlements)
  - [Subscriptions](#subscriptions)
  - [Apps](#apps)
  - [Billing](#billing)
  - [Invoices](#invoices)
  - [Tax](#tax)
  - [Currencies](#currencies)
  - [Features](#features)
  - [LLMCost](#llmcost)
  - [Plans](#plans)
  - [Addons](#addons)
  - [PlanAddons](#planaddons)
  - [Defaults](#defaults)
  - [Governance](#governance)
- [Error Handling](#error-handling)
- [Standalone Functions](#standalone-functions)

## Installation

```bash
npm install @openmeter/client
```

Or with your package manager of choice:

```bash
pnpm add @openmeter/client
yarn add @openmeter/client
```

## Initialization

Create a client with a base URL and an API key. The API key is sent as a
`Bearer` token on every request.

```typescript
import { OpenMeter } from '@openmeter/client'

const client = new OpenMeter({
  baseUrl: 'https://openmeter.cloud/api/v3',
  apiKey: process.env.OPENMETER_API_KEY,
})
```

Konnect regions are addressed with a server template and a `region`
variable:

```typescript
import { OpenMeter, ServerList } from '@openmeter/client'

const client = new OpenMeter({
  baseUrl: ServerList[0],
  serverVariables: { region: 'eu' },
  apiKey: process.env.OPENMETER_API_KEY,
})
```

The `apiKey` may also be a function returning a `string` or
`Promise<string>`, so tokens can be refreshed per request.

## Usage

Every operation is reachable through a fluent, namespaced client and
returns a typed response (or throws an `HTTPError` on a non-2xx status).

```typescript
import { OpenMeter } from '@openmeter/client'

const client = new OpenMeter({
  baseUrl: 'https://openmeter.cloud/api/v3',
  apiKey: process.env.OPENMETER_API_KEY,
})

const meter = await client.meters.create({
  name: 'Tokens',
  key: 'tokens',
  aggregation: 'sum',
  event_type: 'request',
  value_property: '$.tokens',
})

const meters = await client.meters.list()
```

Each method takes the request object as its first argument and an optional
per-request options object (`RequestOptions`) as its second.

## Available Resources and Operations

Operations are grouped by resource and exposed as methods on the client.
The full call path, HTTP route, and a short description are listed below.

### Events

| Method                 | HTTP                     | Description                                                                  |
| ---------------------- | ------------------------ | ---------------------------------------------------------------------------- |
| `client.events.list`   | `GET /openmeter/events`  | List ingested events.                                                        |
| `client.events.ingest` | `POST /openmeter/events` | Ingests an event or batch of events following the CloudEvents specification. |

### Meters

| Method                   | HTTP                                     | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| ------------------------ | ---------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `client.meters.create`   | `POST /openmeter/meters`                 | Create a meter.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.meters.get`      | `GET /openmeter/meters/{meterId}`        | Get a meter by ID.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                             |
| `client.meters.list`     | `GET /openmeter/meters`                  | List meters.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| `client.meters.update`   | `PUT /openmeter/meters/{meterId}`        | Update a meter.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.meters.delete`   | `DELETE /openmeter/meters/{meterId}`     | Delete a meter.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.meters.query`    | `POST /openmeter/meters/{meterId}/query` | Query a meter for usage. Set `Accept: application/json` (the default) to get a structured JSON response. Set `Accept: text/csv` to download the same data as a CSV file suitable for spreadsheets. The CSV columns, in order, are: `from, to, [subject,] [customer_id, customer_key, customer_name,] <dimensions...>, value` The `subject` column is emitted only when `subject` is in the query's `group_by_dimensions`. The three `customer_*` columns are emitted together only when `customer_id` is in the query's `group_by_dimensions`. |
| `client.meters.queryCsv` | `POST /openmeter/meters/{meterId}/query` |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |

### Customers

| Method                                                     | HTTP                                                                                        | Description                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                    |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `client.customers.create`                                  | `POST /openmeter/customers`                                                                 |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.customers.get`                                     | `GET /openmeter/customers/{customerId}`                                                     |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.customers.list`                                    | `GET /openmeter/customers`                                                                  |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.customers.upsert`                                  | `PUT /openmeter/customers/{customerId}`                                                     |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.customers.delete`                                  | `DELETE /openmeter/customers/{customerId}`                                                  |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.customers.billing.get`                             | `GET /openmeter/customers/{customerId}/billing`                                             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.customers.billing.update`                          | `PUT /openmeter/customers/{customerId}/billing`                                             |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.customers.billing.updateAppData`                   | `PUT /openmeter/customers/{customerId}/billing/app-data`                                    |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| `client.customers.billing.createStripeCheckoutSession`     | `POST /openmeter/customers/{customerId}/billing/stripe/checkout-sessions`                   | Create a [Stripe Checkout Session](https://docs.stripe.com/payments/checkout) for the customer. Creates a Checkout Session for collecting payment method information from customers. The session operates in "setup" mode, which collects payment details without charging the customer immediately. The collected payment method can be used for future subscription billing. For hosted checkout sessions, redirect customers to the returned URL. For embedded sessions, use the client_secret to initialize Stripe.js in your application. |
| `client.customers.billing.createStripePortalSession`       | `POST /openmeter/customers/{customerId}/billing/stripe/portal-sessions`                     | Create Stripe Customer Portal Session. Useful to redirect the customer to the Stripe Customer Portal to manage their payment methods, change their billing address and access their invoice history. Only returns URL if the customer billing profile is linked to a stripe app and customer.                                                                                                                                                                                                                                                  |
| `client.customers.credits.grants.create`                   | `POST /openmeter/customers/{customerId}/credits/grants`                                     | Create a new credit grant. A credit grant represents an allocation of prepaid credits to a customer.                                                                                                                                                                                                                                                                                                                                                                                                                                           |
| `client.customers.credits.grants.get`                      | `GET /openmeter/customers/{customerId}/credits/grants/{creditGrantId}`                      | Get a credit grant.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `client.customers.credits.grants.list`                     | `GET /openmeter/customers/{customerId}/credits/grants`                                      | List credit grants.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            |
| `client.customers.credits.balance.get`                     | `GET /openmeter/customers/{customerId}/credits/balance`                                     | Get a credit balance.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `client.customers.credits.adjustments.create`              | `POST /openmeter/customers/{customerId}/credits/adjustments`                                | A credit adjustment can be used to make manual adjustments to a customer's credit balance. Supported use-cases: - Usage correction                                                                                                                                                                                                                                                                                                                                                                                                             |
| `client.customers.credits.grants.updateExternalSettlement` | `POST /openmeter/customers/{customerId}/credits/grants/{creditGrantId}/settlement/external` | Update the payment settlement status of an externally funded credit grant. Use this endpoint to synchronize the payment state of an external payment with the system so that revenue recognition and credit availability work as expected.                                                                                                                                                                                                                                                                                                     |
| `client.customers.credits.transactions.list`               | `GET /openmeter/customers/{customerId}/credits/transactions`                                | List credit transactions for a customer. Returns an immutable, chronological record of credit movements: funded credits and consumed credits. Transactions are returned in reverse chronological order by default.                                                                                                                                                                                                                                                                                                                             |
| `client.customers.charges.list`                            | `GET /openmeter/customers/{customerId}/charges`                                             | List customer charges. Returns the customer's charges that are represented as either flat fee or usage-based charges.                                                                                                                                                                                                                                                                                                                                                                                                                          |
| `client.customers.charges.create`                          | `POST /openmeter/customers/{customerId}/charges`                                            | Create customer charge.                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |

### Entitlements

| Method                                   | HTTP                                                       | Description |
| ---------------------------------------- | ---------------------------------------------------------- | ----------- |
| `client.entitlements.listCustomerAccess` | `GET /openmeter/customers/{customerId}/entitlement-access` |             |

### Subscriptions

| Method                                       | HTTP                                                                         | Description                                                                                                                                    |
| -------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `client.subscriptions.create`                | `POST /openmeter/subscriptions`                                              |                                                                                                                                                |
| `client.subscriptions.list`                  | `GET /openmeter/subscriptions`                                               |                                                                                                                                                |
| `client.subscriptions.get`                   | `GET /openmeter/subscriptions/{subscriptionId}`                              |                                                                                                                                                |
| `client.subscriptions.cancel`                | `POST /openmeter/subscriptions/{subscriptionId}/cancel`                      | Cancels the subscription. Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancelation time. |
| `client.subscriptions.unscheduleCancelation` | `POST /openmeter/subscriptions/{subscriptionId}/unschedule-cancelation`      | Unschedules the subscription cancelation.                                                                                                      |
| `client.subscriptions.change`                | `POST /openmeter/subscriptions/{subscriptionId}/change`                      | Closes a running subscription and starts a new one according to the specification. Can be used for upgrades, downgrades, and plan changes.     |
| `client.subscriptions.listAddons`            | `GET /openmeter/subscriptions/{subscriptionId}/addons`                       | List the addons of a subscription.                                                                                                             |
| `client.subscriptions.getAddon`              | `GET /openmeter/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}` | Get an add-on association for a subscription.                                                                                                  |

### Apps

| Method             | HTTP                          | Description           |
| ------------------ | ----------------------------- | --------------------- |
| `client.apps.list` | `GET /openmeter/apps`         | List installed apps.  |
| `client.apps.get`  | `GET /openmeter/apps/{appId}` | Get an installed app. |

### Billing

| Method                         | HTTP                              | Description                                                                                                                                                                                                                                                                                                              |
| ------------------------------ | --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `client.billing.listProfiles`  | `GET /openmeter/profiles`         | List billing profiles.                                                                                                                                                                                                                                                                                                   |
| `client.billing.createProfile` | `POST /openmeter/profiles`        | Create a new billing profile. Billing profiles contain the settings for billing and controls invoice generation. An organization can have multiple billing profiles defined. A billing profile is linked to a specific app. This association is established during the billing profile's creation and remains immutable. |
| `client.billing.getProfile`    | `GET /openmeter/profiles/{id}`    | Get a billing profile.                                                                                                                                                                                                                                                                                                   |
| `client.billing.updateProfile` | `PUT /openmeter/profiles/{id}`    | Update a billing profile.                                                                                                                                                                                                                                                                                                |
| `client.billing.deleteProfile` | `DELETE /openmeter/profiles/{id}` | Delete a billing profile. Only such billing profiles can be deleted that are: - not the default profile - not pinned to any customer using customer overrides - only have finalized invoices                                                                                                                             |

### Invoices

| Method                | HTTP                                          | Description                                                                                                                                       |
| --------------------- | --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `client.invoices.get` | `GET /openmeter/billing/invoices/{invoiceId}` | Get a billing invoice by ID. Returns the full invoice resource including line items, status details, totals, and workflow configuration snapshot. |

### Tax

| Method                  | HTTP                                      | Description |
| ----------------------- | ----------------------------------------- | ----------- |
| `client.tax.createCode` | `POST /openmeter/tax-codes`               |             |
| `client.tax.getCode`    | `GET /openmeter/tax-codes/{taxCodeId}`    |             |
| `client.tax.listCodes`  | `GET /openmeter/tax-codes`                |             |
| `client.tax.upsertCode` | `PUT /openmeter/tax-codes/{taxCodeId}`    |             |
| `client.tax.deleteCode` | `DELETE /openmeter/tax-codes/{taxCodeId}` |             |

### Currencies

| Method                                   | HTTP                                                        | Description                                                                                                                    |
| ---------------------------------------- | ----------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `client.currencies.list`                 | `GET /openmeter/currencies`                                 | List currencies supported by the billing system.                                                                               |
| `client.currencies.createCustomCurrency` | `POST /openmeter/currencies/custom`                         | Create a custom currency. This operation allows defining your own custom currency for billing purposes.                        |
| `client.currencies.listCostBases`        | `GET /openmeter/currencies/custom/{currencyId}/cost-bases`  | List cost bases for a currency. For custom currencies, there can be multiple cost bases with different `effective_from` dates. |
| `client.currencies.createCostBasis`      | `POST /openmeter/currencies/custom/{currencyId}/cost-bases` | Create a cost basis for a currency.                                                                                            |

### Features

| Method                      | HTTP                                              | Description                                                                |
| --------------------------- | ------------------------------------------------- | -------------------------------------------------------------------------- |
| `client.features.list`      | `GET /openmeter/features`                         | List all features.                                                         |
| `client.features.create`    | `POST /openmeter/features`                        | Create a feature.                                                          |
| `client.features.get`       | `GET /openmeter/features/{featureId}`             | Get a feature by id.                                                       |
| `client.features.update`    | `PATCH /openmeter/features/{featureId}`           | Update a feature by id. Currently only the unit_cost field can be updated. |
| `client.features.delete`    | `DELETE /openmeter/features/{featureId}`          | Delete a feature by id.                                                    |
| `client.features.queryCost` | `POST /openmeter/features/{featureId}/cost/query` | Query the cost of a feature.                                               |

### LLMCost

| Method                          | HTTP                                             | Description                                                                           |
| ------------------------------- | ------------------------------------------------ | ------------------------------------------------------------------------------------- |
| `client.llmCost.listPrices`     | `GET /openmeter/llm-cost/prices`                 | List global LLM cost prices. Returns prices with overrides applied if any.            |
| `client.llmCost.getPrice`       | `GET /openmeter/llm-cost/prices/{priceId}`       | Get a specific LLM cost price by ID. Returns the price with overrides applied if any. |
| `client.llmCost.listOverrides`  | `GET /openmeter/llm-cost/overrides`              | List per-namespace price overrides.                                                   |
| `client.llmCost.createOverride` | `POST /openmeter/llm-cost/overrides`             | Create a per-namespace price override.                                                |
| `client.llmCost.deleteOverride` | `DELETE /openmeter/llm-cost/overrides/{priceId}` | Delete a per-namespace price override.                                                |

### Plans

| Method                 | HTTP                                     | Description             |
| ---------------------- | ---------------------------------------- | ----------------------- |
| `client.plans.list`    | `GET /openmeter/plans`                   | List all plans.         |
| `client.plans.create`  | `POST /openmeter/plans`                  | Create a new plan.      |
| `client.plans.update`  | `PUT /openmeter/plans/{planId}`          | Update a plan by id.    |
| `client.plans.get`     | `GET /openmeter/plans/{planId}`          | Get a plan by id.       |
| `client.plans.delete`  | `DELETE /openmeter/plans/{planId}`       | Delete a plan by id.    |
| `client.plans.archive` | `POST /openmeter/plans/{planId}/archive` | Archive a plan version. |
| `client.plans.publish` | `POST /openmeter/plans/{planId}/publish` | Publish a plan version. |

### Addons

| Method                  | HTTP                                       | Description                |
| ----------------------- | ------------------------------------------ | -------------------------- |
| `client.addons.list`    | `GET /openmeter/addons`                    | List all add-ons.          |
| `client.addons.create`  | `POST /openmeter/addons`                   | Create a new add-on.       |
| `client.addons.update`  | `PUT /openmeter/addons/{addonId}`          | Update an add-on by id.    |
| `client.addons.get`     | `GET /openmeter/addons/{addonId}`          | Get add-on by id.          |
| `client.addons.delete`  | `DELETE /openmeter/addons/{addonId}`       | Soft delete add-on by id.  |
| `client.addons.archive` | `POST /openmeter/addons/{addonId}/archive` | Archive an add-on version. |
| `client.addons.publish` | `POST /openmeter/addons/{addonId}/publish` | Publish an add-on version. |

### PlanAddons

| Method                     | HTTP                                                    | Description                              |
| -------------------------- | ------------------------------------------------------- | ---------------------------------------- |
| `client.planAddons.list`   | `GET /openmeter/plans/{planId}/addons`                  | List add-ons associated with a plan.     |
| `client.planAddons.create` | `POST /openmeter/plans/{planId}/addons`                 | Add an add-on to a plan.                 |
| `client.planAddons.get`    | `GET /openmeter/plans/{planId}/addons/{planAddonId}`    | Get an add-on association for a plan.    |
| `client.planAddons.update` | `PUT /openmeter/plans/{planId}/addons/{planAddonId}`    | Update an add-on association for a plan. |
| `client.planAddons.delete` | `DELETE /openmeter/plans/{planId}/addons/{planAddonId}` | Remove an add-on from a plan.            |

### Defaults

| Method                                       | HTTP                                | Description |
| -------------------------------------------- | ----------------------------------- | ----------- |
| `client.defaults.getOrganizationTaxCodes`    | `GET /openmeter/defaults/tax-codes` |             |
| `client.defaults.updateOrganizationTaxCodes` | `PUT /openmeter/defaults/tax-codes` |             |

### Governance

| Method                          | HTTP                               | Description                                                                                                                                                                                                                                                                                                          |
| ------------------------------- | ---------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `client.governance.queryAccess` | `POST /openmeter/governance/query` | Query feature access for a list of customers. The endpoint resolves each provided identifier to a customer and returns the access status for the requested features, plus optional credit balance availability. _Designed to be called on a fixed refresh interval and the query response is intended to be cached._ |

## Error Handling

A non-2xx response rejects with an `HTTPError` carrying the problem-details
fields (`status`, `type`, `title`, `url`) from the response.

```typescript
import { OpenMeter, HTTPError } from '@openmeter/client'

const client = new OpenMeter({
  baseUrl: 'https://openmeter.cloud/api/v3',
  apiKey: process.env.OPENMETER_API_KEY,
})

try {
  await client.meters.get({ meterId: 'unknown' })
} catch (error) {
  if (error instanceof HTTPError) {
    console.error(error.status, error.title, error.type)
  }
}
```

## Standalone Functions

Every method is also available as a standalone, tree-shakeable function
that takes a `Client` and returns a `Result` instead of throwing.

```typescript
import { Client, funcs } from '@openmeter/client'

const client = new Client({
  baseUrl: 'https://openmeter.cloud/api/v3',
  apiKey: process.env.OPENMETER_API_KEY,
})

const result = await funcs.listMeters(client)
if (result.ok) {
  console.log(result.value)
} else {
  console.error(result.error)
}
```
