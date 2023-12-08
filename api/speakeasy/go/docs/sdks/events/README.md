# Events
(*Events*)

## Overview

Endpoints related to ingesting events

### Available Operations

* [ListEvents](#listevents) - Retrieve latest raw events.
* [IngestEvents](#ingestevents) - Ingest events

## ListEvents

Retrieve latest raw events.

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"time"
	"openapi/types"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(""),
    )


    var from *time.Time = types.MustTimeFromString("2022-04-10T19:21:26.844Z")

    var to *time.Time = types.MustTimeFromString("2023-07-10T04:24:38.503Z")

    var limit *int64 = 724187

    ctx := context.Background()
    res, err := s.Events.ListEvents(ctx, from, to, limit)
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
| `from`                                                | [*time.Time](https://pkg.go.dev/time#Time)            | :heavy_minus_sign:                                    | Start date-time in RFC 3339 format.<br/>Inclusive.<br/> |
| `to`                                                  | [*time.Time](https://pkg.go.dev/time#Time)            | :heavy_minus_sign:                                    | End date-time in RFC 3339 format.<br/>Inclusive.<br/> |
| `limit`                                               | **int64*                                              | :heavy_minus_sign:                                    | Number of events to return.                           |


### Response

**[*operations.ListEventsResponse](../../models/operations/listeventsresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 400                      | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |

## IngestEvents

Ingest events

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"openapi/types"
	"log"
	"net/http"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(""),
    )

    ctx := context.Background()
    res, err := s.Events.IngestEvents(ctx, []components.Event{
        components.Event{
            ID: "5c10fade-1c9e-4d6c-8275-c52c36731d3c",
            Source: "services/service-0",
            Specversion: "1.0",
            Type: "api_request",
            Datacontenttype: components.DatacontenttypeApplicationJSON.ToPointer(),
            Subject: "customer_id",
            Time: types.MustTimeFromString("2023-01-01T01:01:01.001Z"),
            Data: map[string]interface{}{
                "duration_ms": "string",
                "path": "string",
            },
        },
    })
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
| `request`                                             | [[]components.Event](../../.md)                       | :heavy_check_mark:                                    | The request object to use for the request.            |


### Response

**[*operations.IngestEventsResponse](../../models/operations/ingesteventsresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 400                      | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |
