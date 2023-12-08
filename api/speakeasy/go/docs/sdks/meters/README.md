# Meters
(*Meters*)

## Overview

Endpoints related to meters

### Available Operations

* [ListMeters](#listmeters) - List meters
* [CreateMeter](#createmeter) - Create meter
* [DeleteMeter](#deletemeter) - Delete meter by slug
* [GetMeter](#getmeter) - Get meter by slugs
* [QueryMeter](#querymeter) - Query meter
* [ListMeterSubjects](#listmetersubjects) - List meter subjects

## ListMeters

List meters

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(""),
    )

    ctx := context.Background()
    res, err := s.Meters.ListMeters(ctx)
    if err != nil {
        log.Fatal(err)
    }

    if res.Classes != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                             | Type                                                  | Required                                              | Description                                           |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `ctx`                                                 | [context.Context](https://pkg.go.dev/context#Context) | :heavy_check_mark:                                    | The context to use for the request.                   |


### Response

**[*operations.ListMetersResponse](../../models/operations/listmetersresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 400-600            | */*                |

## CreateMeter

Create meter

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(""),
    )

    ctx := context.Background()
    res, err := s.Meters.CreateMeter(ctx, components.MeterInput{
        Slug: "my_meter",
        Description: openapi.String("My Meter Description"),
        Aggregation: components.MeterAggregationMin,
        WindowSize: components.WindowSizeDay,
        EventType: "api_request",
        ValueProperty: openapi.String("$.duration_ms"),
        GroupBy: map[string]string{
            "method": "$.method",
            "path": "$.path",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    if res.Meter != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                      | Type                                                           | Required                                                       | Description                                                    |
| -------------------------------------------------------------- | -------------------------------------------------------------- | -------------------------------------------------------------- | -------------------------------------------------------------- |
| `ctx`                                                          | [context.Context](https://pkg.go.dev/context#Context)          | :heavy_check_mark:                                             | The context to use for the request.                            |
| `request`                                                      | [components.MeterInput](../../models/components/meterinput.md) | :heavy_check_mark:                                             | The request object to use for the request.                     |


### Response

**[*operations.CreateMeterResponse](../../models/operations/createmeterresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 400,501                  | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |

## DeleteMeter

Delete meter by slug

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
	"net/http"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(""),
    )


    var meterIDOrSlug string = "string"

    ctx := context.Background()
    res, err := s.Meters.DeleteMeter(ctx, meterIDOrSlug)
    if err != nil {
        log.Fatal(err)
    }

    if res.StatusCode == http.StatusOK {
        // handle response
    }
}
```

### Parameters

| Parameter                                             | Type                                                  | Required                                              | Description                                           |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `ctx`                                                 | [context.Context](https://pkg.go.dev/context#Context) | :heavy_check_mark:                                    | The context to use for the request.                   |
| `meterIDOrSlug`                                       | *string*                                              | :heavy_check_mark:                                    | A unique identifier for the meter.                    |


### Response

**[*operations.DeleteMeterResponse](../../models/operations/deletemeterresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 404,501                  | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |

## GetMeter

Get meter by slugs

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(""),
    )


    var meterIDOrSlug string = "string"

    ctx := context.Background()
    res, err := s.Meters.GetMeter(ctx, meterIDOrSlug)
    if err != nil {
        log.Fatal(err)
    }

    if res.Meter != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                             | Type                                                  | Required                                              | Description                                           |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `ctx`                                                 | [context.Context](https://pkg.go.dev/context#Context) | :heavy_check_mark:                                    | The context to use for the request.                   |
| `meterIDOrSlug`                                       | *string*                                              | :heavy_check_mark:                                    | A unique identifier for the meter.                    |


### Response

**[*operations.GetMeterResponse](../../models/operations/getmeterresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 404                      | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |

## QueryMeter

Query meter

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"openapi/models/operations"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(""),
    )

    ctx := context.Background()
    res, err := s.Meters.QueryMeter(ctx, operations.QueryMeterRequest{
        MeterIDOrSlug: "string",
        WindowTimeZone: openapi.String("America/New_York"),
        Subject: []string{
            "string",
        },
        GroupBy: []string{
            "string",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    if res.MeterQueryResult != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                    | Type                                                                         | Required                                                                     | Description                                                                  |
| ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| `ctx`                                                                        | [context.Context](https://pkg.go.dev/context#Context)                        | :heavy_check_mark:                                                           | The context to use for the request.                                          |
| `request`                                                                    | [operations.QueryMeterRequest](../../models/operations/querymeterrequest.md) | :heavy_check_mark:                                                           | The request object to use for the request.                                   |


### Response

**[*operations.QueryMeterResponse](../../models/operations/querymeterresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 400                      | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |

## ListMeterSubjects

List meter subjects

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(""),
    )


    var meterIDOrSlug string = "string"

    ctx := context.Background()
    res, err := s.Meters.ListMeterSubjects(ctx, meterIDOrSlug)
    if err != nil {
        log.Fatal(err)
    }

    if res.Strings != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                             | Type                                                  | Required                                              | Description                                           |
| ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- | ----------------------------------------------------- |
| `ctx`                                                 | [context.Context](https://pkg.go.dev/context#Context) | :heavy_check_mark:                                    | The context to use for the request.                   |
| `meterIDOrSlug`                                       | *string*                                              | :heavy_check_mark:                                    | A unique identifier for the meter.                    |


### Response

**[*operations.ListMeterSubjectsResponse](../../models/operations/listmetersubjectsresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 400                      | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |
