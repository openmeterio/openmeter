# aipclientgo

Type-safe Go SDK for the OpenMeter API.

## Summary

OpenMeter is a cloud native usage metering and billing service. The OpenMeter
API allows you to ingest events, query meter usage, and manage resources.

## Table of Contents

* [Installation](#installation)
* [Example](#example)
* [Services](#services)
* [Server selection](#server-selection)
* [Custom HTTP client](#custom-http-client)
* [Error handling](#error-handling)

## Installation

```bash
go get github.com/openmeterio/openmeter/api/spec/packages/aip-client-go
```

## Example

```go
package main

import (
	"context"
	"log"

	aipclientgo "github.com/openmeterio/openmeter/api/spec/packages/aip-client-go"
)

func main() {
	ctx := context.Background()
	s := aipclientgo.New()

	res, err := s.OpenMeterEvents.ListMeteringEvents(ctx)
	if err != nil {
		log.Fatal(err)
	}
	_ = res
}
```

## Services

* `OpenMeterEvents` — 2 operations
* `OpenMeterMeters` — 5 operations
* `OpenMeterMetersQuery` — 1 operation
* `OpenMeterCustomers` — 5 operations
* `OpenMeterCustomerBilling` — 5 operations
* `OpenMeterCustomerEntitlements` — 1 operation
* `OpenMeterCustomerCreditGrants` — 3 operations
* `OpenMeterCustomerCreditBalance` — 1 operation
* `OpenMeterCustomerCreditAdjustments` — 1 operation
* `OpenMeterCustomerCreditGrant` — 1 operation
* `OpenMeterCustomerCreditTransaction` — 1 operation
* `OpenMeterCustomerCharges` — 1 operation
* `OpenMeterSubscriptions` — 6 operations
* `OpenMeterSubscriptionAddon` — 1 operation
* `OpenMeterApps` — 2 operations
* `OpenMeterBillingProfiles` — 5 operations
* `OpenMeterTaxCodes` — 5 operations
* `OpenMeterCurrencies` — 1 operation
* `OpenMeterCurrenciesCustom` — 1 operation
* `OpenMeterCurrenciesCustomCostBases` — 2 operations
* `OpenMeterFeatures` — 5 operations
* `OpenMeterFeatureCost` — 1 operation
* `OpenMeterLlmCostPrices` — 2 operations
* `OpenMeterLlmCostOverrides` — 3 operations
* `OpenMeterPlans` — 7 operations
* `OpenMeterAddons` — 7 operations
* `OpenMeterPlanAddon` — 5 operations
* `OpenMeterOrganizationDefaultTaxCodes` — 2 operations
* `OpenMeterGovernance` — 1 operation
* `OpenMeterFieldFilters` — 1 operation

## Server selection

The SDK ships with the following server URLs (selected via `WithServerIndex`):

0. `http://localhost:{port}/api/v3` — Local
1. `https://openmeter.cloud/api/v3` — Cloud
2. `https://global.api.konghq.com/v3` — Global Production region
3. `https://in.api.konghq.com/v3` — India Production region
4. `https://me.api.konghq.com/v3` — Middle-East Production region
5. `https://au.api.konghq.com/v3` — Australia Production region
6. `https://eu.api.konghq.com/v3` — Europe Production region
7. `https://us.api.konghq.com/v3` — United-States Production region

You can also pass a custom URL with `WithServerURL`:

```go
s := aipclientgo.New(aipclientgo.WithServerURL("https://api.example.com"))
```

## Custom HTTP client

Provide any value implementing the `HTTPClient` interface (`Do(*http.Request) (*http.Response, error)`):

```go
httpClient := &http.Client{Timeout: 30 * time.Second}
s := aipclientgo.New(aipclientgo.WithClient(httpClient))
```

## Error handling

All operations return `(*operations.XxxResponse, error)`. Status-coded errors are typed:

```go
res, err := s.SomeService.SomeOperation(ctx, req)
if err != nil {
    var notFound *apierrors.NotFoundError
    if errors.As(err, &notFound) {
        // resource missing
    }
    return err
}
```

---

_This SDK was generated from a TypeSpec definition. Do not edit generated files directly._
