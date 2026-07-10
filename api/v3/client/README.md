# OpenMeter Go SDK

Go client for the OpenMeter API — usage metering and billing for
AI and DevTool companies. This package is generated from the OpenMeter
TypeSpec definitions and ships typed request and response models.

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
- [Pagination and Streaming](#pagination-and-streaming)

## Installation

```bash
go get github.com/openmeterio/openmeter/api/v3/client
```

## Initialization

Create a client with a base URL and an API key. The API key is sent as a
`Bearer` token on every request.

```go
package main

import (
	"log"
	"os"

	"github.com/openmeterio/openmeter/api/v3/client"
)

func main() {
	om, err := openmeter.New(
		"https://openmeter.cloud/api/v3",
		openmeter.WithToken(os.Getenv("OPENMETER_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	_ = om
}
```

For region-specific deployments, pass the concrete API base URL for that
region to `New`.

## Usage

Every operation is reachable through a namespaced service on the client and
returns a typed response plus an `error`.

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/openmeterio/openmeter/api/v3/client"
)

func main() {
	om, err := openmeter.New(
		"https://openmeter.cloud/api/v3",
		openmeter.WithToken(os.Getenv("OPENMETER_API_KEY")),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	meter, err := om.Meters.Create(ctx, openmeter.CreateMeterRequest{
		Name:          "Tokens",
		Key:           "tokens",
		Aggregation:   openmeter.MeterAggregationSum,
		EventType:     "request",
		ValueProperty: openmeter.String("$.tokens"),
	})
	if err != nil {
		log.Fatal(err)
	}

	meters, err := om.Meters.List(ctx, openmeter.MeterListParams{})
	if err != nil {
		log.Fatal(err)
	}

	_, _ = meter, meters
}
```

Operation arguments follow the generated method signature: path parameters
come first, then a typed request body when present, then typed query params
when present.

## Available Resources and Operations

Operations are grouped by resource and exposed as services on the client.
The full call path, HTTP route, and a short description are listed below.

### Events

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Events.List` | `GET /openmeter/events` | List ingested events. |
| `om.Events.IngestEvent` | `POST /openmeter/events` | Ingests an event or batch of events following the CloudEvents specification. |
| `om.Events.IngestEvents` | `POST /openmeter/events` |  |
| `om.Events.IngestEventsJSON` | `POST /openmeter/events` |  |

### Meters

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Meters.Create` | `POST /openmeter/meters` | Create a meter. |
| `om.Meters.Get` | `GET /openmeter/meters/{meterId}` | Get a meter by ID. |
| `om.Meters.List` | `GET /openmeter/meters` | List meters. |
| `om.Meters.Update` | `PUT /openmeter/meters/{meterId}` | Update a meter. |
| `om.Meters.Delete` | `DELETE /openmeter/meters/{meterId}` | Delete a meter. |
| `om.Meters.Query` | `POST /openmeter/meters/{meterId}/query` | Query a meter for usage. Set `Accept: application/json` (the default) to get a structured JSON response. Set `Accept: text/csv` to download the same data as a CSV file suitable for spreadsheets. The CSV columns, in order, are: `from, to, [subject,] [customer_id, customer_key, customer_name,] <dimensions...>, value` The `subject` column is emitted only when `subject` is in the query's `group_by_dimensions`. The three `customer_*` columns are emitted together only when `customer_id` is in the query's `group_by_dimensions`. |
| `om.Meters.QueryCSV` | `POST /openmeter/meters/{meterId}/query` |  |
| `om.Meters.QueryCSVStream` | `POST /openmeter/meters/{meterId}/query` | Streaming variant of `QueryCSV` returning an `io.ReadCloser`. |

### Customers

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Customers.Create` | `POST /openmeter/customers` |  |
| `om.Customers.Get` | `GET /openmeter/customers/{customerId}` |  |
| `om.Customers.List` | `GET /openmeter/customers` |  |
| `om.Customers.Upsert` | `PUT /openmeter/customers/{customerId}` |  |
| `om.Customers.Delete` | `DELETE /openmeter/customers/{customerId}` |  |
| `om.Customers.Billing.Get` | `GET /openmeter/customers/{customerId}/billing` |  |
| `om.Customers.Billing.Update` | `PUT /openmeter/customers/{customerId}/billing` |  |
| `om.Customers.Billing.UpdateAppData` | `PUT /openmeter/customers/{customerId}/billing/app-data` |  |
| `om.Customers.Billing.CreateStripeCheckoutSession` | `POST /openmeter/customers/{customerId}/billing/stripe/checkout-sessions` | Create a [Stripe Checkout Session](https://docs.stripe.com/payments/checkout) for the customer. Creates a Checkout Session for collecting payment method information from customers. The session operates in "setup" mode, which collects payment details without charging the customer immediately. The collected payment method can be used for future subscription billing. For hosted checkout sessions, redirect customers to the returned URL. For embedded sessions, use the client_secret to initialize Stripe.js in your application. |
| `om.Customers.Billing.CreateStripePortalSession` | `POST /openmeter/customers/{customerId}/billing/stripe/portal-sessions` | Create Stripe Customer Portal Session. Useful to redirect the customer to the Stripe Customer Portal to manage their payment methods, change their billing address and access their invoice history. Only returns URL if the customer billing profile is linked to a stripe app and customer. |
| `om.Customers.Credits.Grants.Create` | `POST /openmeter/customers/{customerId}/credits/grants` | Create a new credit grant. A credit grant represents an allocation of prepaid credits to a customer. |
| `om.Customers.Credits.Grants.Get` | `GET /openmeter/customers/{customerId}/credits/grants/{creditGrantId}` | Get a credit grant. |
| `om.Customers.Credits.Grants.List` | `GET /openmeter/customers/{customerId}/credits/grants` | List credit grants. |
| `om.Customers.Credits.Grants.UpdateExternalSettlement` | `POST /openmeter/customers/{customerId}/credits/grants/{creditGrantId}/settlement/external` | Update the payment settlement status of an externally funded credit grant. Use this endpoint to synchronize the payment state of an external payment with the system so that revenue recognition and credit availability work as expected. |
| `om.Customers.Credits.Balance.Get` | `GET /openmeter/customers/{customerId}/credits/balance` | Get a credit balance. |
| `om.Customers.Credits.Adjustments.Create` | `POST /openmeter/customers/{customerId}/credits/adjustments` | A credit adjustment can be used to make manual adjustments to a customer's credit balance. Supported use-cases: - Usage correction |
| `om.Customers.Credits.Transactions.List` | `GET /openmeter/customers/{customerId}/credits/transactions` | List credit transactions for a customer. Returns an immutable, chronological record of credit movements: funded credits and consumed credits. Transactions are returned in reverse chronological order by default. |
| `om.Customers.Charges.List` | `GET /openmeter/customers/{customerId}/charges` | List customer charges. Returns the customer's charges that are represented as either flat fee or usage-based charges. |
| `om.Customers.Charges.Create` | `POST /openmeter/customers/{customerId}/charges` | Create customer charge. |

### Entitlements

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Entitlements.ListCustomerAccess` | `GET /openmeter/customers/{customerId}/entitlement-access` |  |

### Subscriptions

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Subscriptions.Create` | `POST /openmeter/subscriptions` |  |
| `om.Subscriptions.List` | `GET /openmeter/subscriptions` |  |
| `om.Subscriptions.Get` | `GET /openmeter/subscriptions/{subscriptionId}` |  |
| `om.Subscriptions.Cancel` | `POST /openmeter/subscriptions/{subscriptionId}/cancel` | Cancels the subscription. Will result in a scheduling conflict if there are other subscriptions scheduled to start after the cancelation time. |
| `om.Subscriptions.UnscheduleCancelation` | `POST /openmeter/subscriptions/{subscriptionId}/unschedule-cancelation` | Unschedules the subscription cancelation. |
| `om.Subscriptions.Change` | `POST /openmeter/subscriptions/{subscriptionId}/change` | Closes a running subscription and starts a new one according to the specification. Can be used for upgrades, downgrades, and plan changes. |
| `om.Subscriptions.CreateAddon` | `POST /openmeter/subscriptions/{subscriptionId}/addons` | Add add-on to a subscription. |
| `om.Subscriptions.ListAddons` | `GET /openmeter/subscriptions/{subscriptionId}/addons` | List the add-ons of a subscription. |
| `om.Subscriptions.GetAddon` | `GET /openmeter/subscriptions/{subscriptionId}/addons/{subscriptionAddonId}` | Get an add-on association for a subscription. |

### Apps

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Apps.List` | `GET /openmeter/apps` | List installed apps. |
| `om.Apps.Get` | `GET /openmeter/apps/{appId}` | Get an installed app. |

### Billing

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Billing.ListProfiles` | `GET /openmeter/profiles` | List billing profiles. |
| `om.Billing.CreateProfile` | `POST /openmeter/profiles` | Create a new billing profile. Billing profiles contain the settings for billing and controls invoice generation. An organization can have multiple billing profiles defined. A billing profile is linked to a specific app. This association is established during the billing profile's creation and remains immutable. |
| `om.Billing.GetProfile` | `GET /openmeter/profiles/{id}` | Get a billing profile. |
| `om.Billing.UpdateProfile` | `PUT /openmeter/profiles/{id}` | Update a billing profile. |
| `om.Billing.DeleteProfile` | `DELETE /openmeter/profiles/{id}` | Delete a billing profile. Only such billing profiles can be deleted that are: - not the default profile - not pinned to any customer using customer overrides - only have finalized invoices |

### Invoices

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Invoices.List` | `GET /openmeter/billing/invoices` | List billing invoices. Returns a page of invoices. Gathering invoices are never included. Use `filter` to narrow by status, customer, dates, or service period start. Use `sort` to control ordering. |
| `om.Invoices.Get` | `GET /openmeter/billing/invoices/{invoiceId}` | Get a billing invoice by ID. Returns the full invoice resource including line items, status details, totals, and workflow configuration snapshot. |
| `om.Invoices.Update` | `PUT /openmeter/billing/invoices/{invoiceId}` | Update a billing invoice. Only the mutable fields of the invoice can be edited: description, labels, supplier, customer, workflow settings, and top-level lines. Top-level lines are matched by `id`; lines without an `id` are created, and existing lines omitted from `lines` are deleted. Detailed (child) lines are always computed and cannot be edited directly. Only invoices in draft status can be updated. |
| `om.Invoices.Delete` | `DELETE /openmeter/billing/invoices/{invoiceId}` | Delete a billing invoice. Only standard invoices in draft status can be deleted. Deleting an invoice will also delete all associated line items and workflow configuration. |

### Tax

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Tax.CreateCode` | `POST /openmeter/tax-codes` |  |
| `om.Tax.GetCode` | `GET /openmeter/tax-codes/{taxCodeId}` |  |
| `om.Tax.ListCodes` | `GET /openmeter/tax-codes` |  |
| `om.Tax.UpsertCode` | `PUT /openmeter/tax-codes/{taxCodeId}` |  |
| `om.Tax.DeleteCode` | `DELETE /openmeter/tax-codes/{taxCodeId}` |  |

### Currencies

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Currencies.List` | `GET /openmeter/currencies` | List currencies supported by the billing system. |
| `om.Currencies.CreateCustomCurrency` | `POST /openmeter/currencies/custom` | Create a custom currency. This operation allows defining your own custom currency for billing purposes. |
| `om.Currencies.ListCostBases` | `GET /openmeter/currencies/custom/{currencyId}/cost-bases` | List cost bases for a currency. For custom currencies, there can be multiple cost bases with different `effective_from` dates. |
| `om.Currencies.CreateCostBasis` | `POST /openmeter/currencies/custom/{currencyId}/cost-bases` | Create a cost basis for a currency. |

### Features

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Features.List` | `GET /openmeter/features` | List all features. |
| `om.Features.Create` | `POST /openmeter/features` | Create a feature. |
| `om.Features.Get` | `GET /openmeter/features/{featureId}` | Get a feature by id. |
| `om.Features.Update` | `PATCH /openmeter/features/{featureId}` | Update a feature by id. Currently only the unit_cost field can be updated. |
| `om.Features.Delete` | `DELETE /openmeter/features/{featureId}` | Delete a feature by id. |
| `om.Features.QueryCost` | `POST /openmeter/features/{featureId}/cost/query` | Query the cost of a feature. |

### LLMCost

| Method | HTTP | Description |
| --- | --- | --- |
| `om.LLMCost.ListPrices` | `GET /openmeter/llm-cost/prices` | List global LLM cost prices. Returns prices with overrides applied if any. |
| `om.LLMCost.GetPrice` | `GET /openmeter/llm-cost/prices/{priceId}` | Get a specific LLM cost price by ID. Returns the price with overrides applied if any. |
| `om.LLMCost.ListOverrides` | `GET /openmeter/llm-cost/overrides` | List per-namespace price overrides. |
| `om.LLMCost.CreateOverride` | `POST /openmeter/llm-cost/overrides` | Create a per-namespace price override. |
| `om.LLMCost.DeleteOverride` | `DELETE /openmeter/llm-cost/overrides/{priceId}` | Delete a per-namespace price override. |

### Plans

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Plans.List` | `GET /openmeter/plans` | List all plans. |
| `om.Plans.Create` | `POST /openmeter/plans` | Create a new plan. |
| `om.Plans.Update` | `PUT /openmeter/plans/{planId}` | Update a plan by id. |
| `om.Plans.Get` | `GET /openmeter/plans/{planId}` | Get a plan by id. |
| `om.Plans.Delete` | `DELETE /openmeter/plans/{planId}` | Delete a plan by id. |
| `om.Plans.Archive` | `POST /openmeter/plans/{planId}/archive` | Archive a plan version. |
| `om.Plans.Publish` | `POST /openmeter/plans/{planId}/publish` | Publish a plan version. |

### Addons

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Addons.List` | `GET /openmeter/addons` | List all add-ons. |
| `om.Addons.Create` | `POST /openmeter/addons` | Create a new add-on. |
| `om.Addons.Update` | `PUT /openmeter/addons/{addonId}` | Update an add-on by id. |
| `om.Addons.Get` | `GET /openmeter/addons/{addonId}` | Get add-on by id. |
| `om.Addons.Delete` | `DELETE /openmeter/addons/{addonId}` | Soft delete add-on by id. |
| `om.Addons.Archive` | `POST /openmeter/addons/{addonId}/archive` | Archive an add-on version. |
| `om.Addons.Publish` | `POST /openmeter/addons/{addonId}/publish` | Publish an add-on version. |

### PlanAddons

| Method | HTTP | Description |
| --- | --- | --- |
| `om.PlanAddons.List` | `GET /openmeter/plans/{planId}/addons` | List add-ons associated with a plan. |
| `om.PlanAddons.Create` | `POST /openmeter/plans/{planId}/addons` | Add an add-on to a plan. |
| `om.PlanAddons.Get` | `GET /openmeter/plans/{planId}/addons/{planAddonId}` | Get an add-on association for a plan. |
| `om.PlanAddons.Update` | `PUT /openmeter/plans/{planId}/addons/{planAddonId}` | Update an add-on association for a plan. |
| `om.PlanAddons.Delete` | `DELETE /openmeter/plans/{planId}/addons/{planAddonId}` | Remove an add-on from a plan. |

### Defaults

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Defaults.GetOrganizationTaxCodes` | `GET /openmeter/defaults/tax-codes` |  |
| `om.Defaults.UpdateOrganizationTaxCodes` | `PUT /openmeter/defaults/tax-codes` |  |

### Governance

| Method | HTTP | Description |
| --- | --- | --- |
| `om.Governance.QueryAccess` | `POST /openmeter/governance/query` | Query feature access for a list of customers. The endpoint resolves each provided identifier to a customer and returns the access status for the requested features, plus optional credit balance availability. _Designed to be called on a fixed refresh interval and the query response is intended to be cached._ |

## Error Handling

A non-2xx response returns an `*APIError` carrying the problem-details
fields (`StatusCode`, `Status`, `Type`, `Title`, `Detail`, `Instance`) from
the response where available. Client-side validation errors such as an empty
path ID are returned before any HTTP request is made.

```go
package main

import (
	"context"
	"errors"
	"log"

	"github.com/openmeterio/openmeter/api/v3/client"
)

func main() {
	om, err := openmeter.New("https://openmeter.cloud/api/v3", openmeter.WithToken("om_..."))
	if err != nil {
		log.Fatal(err)
	}

	_, err = om.Meters.Get(context.Background(), "unknown")
	if err != nil {
		var apiErr *openmeter.APIError
		if errors.As(err, &apiErr) {
			log.Printf("%d %s %s", apiErr.StatusCode, apiErr.Title, apiErr.Type)
			return
		}
		log.Fatal(err)
	}
}
```

## Pagination and Streaming

Paginated list operations also emit `ListAll` helpers that return
`iter.Seq2[T, error]`. Text responses such as meter CSV export emit a byte
returning method and a `Stream` variant for callers that want an
`io.ReadCloser`.

Cursor-paginated responses report their position as `Next` and `Previous`
on `CursorMetaPage`. Both are opaque cursor tokens: pass them back verbatim
as the `page[after]` / `page[before]` query parameters
(`CursorPageParams.After` / `CursorPageParams.Before`); do not parse or
construct them.

Iterating with `Before` set walks pages backward while the items within
each page stay in forward order, so the resulting stream is not globally
sorted.

```go
for meter, err := range om.Meters.ListAll(ctx, openmeter.MeterListParams{}) {
	if err != nil {
		log.Fatal(err)
	}
	log.Println(meter.Key)
}

stream, err := om.Meters.QueryCSVStream(ctx, "meter-id", openmeter.MeterQueryRequest{})
if err != nil {
	log.Fatal(err)
}
defer stream.Close()
```
