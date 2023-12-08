# Portal
(*Portal*)

## Overview

Endpoints related to the portal

### Available Operations

* [CreatePortalToken](#createportaltoken)
* [InvalidatePortalTokens](#invalidateportaltokens)
* [QueryPortalMeter](#queryportalmeter)

## CreatePortalToken

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
    res, err := s.Portal.CreatePortalToken(ctx, &components.PortalTokenInput{
        Subject: "string",
        AllowedMeterSlugs: []string{
            "string",
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    if res.PortalToken != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                  | Type                                                                       | Required                                                                   | Description                                                                |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| `ctx`                                                                      | [context.Context](https://pkg.go.dev/context#Context)                      | :heavy_check_mark:                                                         | The context to use for the request.                                        |
| `request`                                                                  | [components.PortalTokenInput](../../models/components/portaltokeninput.md) | :heavy_check_mark:                                                         | The request object to use for the request.                                 |


### Response

**[*operations.CreatePortalTokenResponse](../../models/operations/createportaltokenresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 400                      | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |

## InvalidatePortalTokens

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"openapi/models/operations"
	"log"
	"net/http"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(""),
    )

    ctx := context.Background()
    res, err := s.Portal.InvalidatePortalTokens(ctx, &operations.InvalidatePortalTokensRequestBody{})
    if err != nil {
        log.Fatal(err)
    }

    if res.StatusCode == http.StatusOK {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                    | Type                                                                                                         | Required                                                                                                     | Description                                                                                                  |
| ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                        | [context.Context](https://pkg.go.dev/context#Context)                                                        | :heavy_check_mark:                                                                                           | The context to use for the request.                                                                          |
| `request`                                                                                                    | [operations.InvalidatePortalTokensRequestBody](../../models/operations/invalidateportaltokensrequestbody.md) | :heavy_check_mark:                                                                                           | The request object to use for the request.                                                                   |


### Response

**[*operations.InvalidatePortalTokensResponse](../../models/operations/invalidateportaltokensresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 400                      | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |

## QueryPortalMeter

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
    res, err := s.Portal.QueryPortalMeter(ctx, operations.QueryPortalMeterRequest{
        MeterSlug: "string",
        WindowTimeZone: openapi.String("America/New_York"),
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

| Parameter                                                                                | Type                                                                                     | Required                                                                                 | Description                                                                              |
| ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `ctx`                                                                                    | [context.Context](https://pkg.go.dev/context#Context)                                    | :heavy_check_mark:                                                                       | The context to use for the request.                                                      |
| `request`                                                                                | [operations.QueryPortalMeterRequest](../../models/operations/queryportalmeterrequest.md) | :heavy_check_mark:                                                                       | The request object to use for the request.                                               |


### Response

**[*operations.QueryPortalMeterResponse](../../models/operations/queryportalmeterresponse.md), error**
| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 400,401                  | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |
