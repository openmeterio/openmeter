# openapi

<div align="left">
    <a href="https://speakeasyapi.dev/"><img src="https://custom-icon-badges.demolab.com/badge/-Built%20By%20Speakeasy-212015?style=for-the-badge&logoColor=FBE331&logo=speakeasy&labelColor=545454" /></a>
    <a href="https://opensource.org/licenses/MIT">
        <img src="https://img.shields.io/badge/License-MIT-blue.svg" style="width: 100px; height: 28px;" />
    </a>
</div>


## 🏗 **Welcome to your new SDK!** 🏗

It has been generated successfully based on your OpenAPI spec. However, it is not yet ready for production use. Here are some next steps:
- [ ] 🛠 Make your SDK feel handcrafted by [customizing it](https://www.speakeasyapi.dev/docs/customize-sdks)
- [ ] ♻️ Refine your SDK quickly by iterating locally with the [Speakeasy CLI](https://github.com/speakeasy-api/speakeasy)
- [ ] 🎁 Publish your SDK to package managers by [configuring automatic publishing](https://www.speakeasyapi.dev/docs/productionize-sdks/publish-sdks)
- [ ] ✨ When ready to productionize, delete this section from the README

<!-- Start SDK Installation [installation] -->
## SDK Installation

```bash
go get openapi
```
<!-- End SDK Installation [installation] -->

<!-- Start SDK Example Usage [usage] -->
## SDK Example Usage

### Example

```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
	"openapi/types"
	"time"
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
<!-- End SDK Example Usage [usage] -->

<!-- Start Available Resources and Operations [operations] -->
## Available Resources and Operations

### [Events](docs/sdks/events/README.md)

* [ListEvents](docs/sdks/events/README.md#listevents) - Retrieve latest raw events.
* [IngestEvents](docs/sdks/events/README.md#ingestevents) - Ingest events

### [Meters](docs/sdks/meters/README.md)

* [ListMeters](docs/sdks/meters/README.md#listmeters) - List meters
* [CreateMeter](docs/sdks/meters/README.md#createmeter) - Create meter
* [DeleteMeter](docs/sdks/meters/README.md#deletemeter) - Delete meter by slug
* [GetMeter](docs/sdks/meters/README.md#getmeter) - Get meter by slugs
* [QueryMeter](docs/sdks/meters/README.md#querymeter) - Query meter
* [ListMeterSubjects](docs/sdks/meters/README.md#listmetersubjects) - List meter subjects

### [Portal](docs/sdks/portal/README.md)

* [CreatePortalToken](docs/sdks/portal/README.md#createportaltoken)
* [InvalidatePortalTokens](docs/sdks/portal/README.md#invalidateportaltokens)
* [QueryPortalMeter](docs/sdks/portal/README.md#queryportalmeter)
<!-- End Available Resources and Operations [operations] -->

<!-- Start Error Handling [errors] -->
## Error Handling

Handling errors in this SDK should largely match your expectations.  All operations return a response object or an error, they will never return both.  When specified by the OpenAPI spec document, the SDK will return the appropriate subclass.

| Error Object             | Status Code              | Content Type             |
| ------------------------ | ------------------------ | ------------------------ |
| sdkerrors.Problem        | 400                      | application/problem+json |
| sdkerrors.SDKError       | 400-600                  | */*                      |

### Example

```go
package main

import (
	"context"
	"errors"
	"log"
	"openapi"
	"openapi/models/components"
	"openapi/models/sdkerrors"
	"openapi/types"
	"time"
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

		var e *sdkerrors.Problem
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}

		var e *sdkerrors.SDKError
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}
	}
}

```
<!-- End Error Handling [errors] -->

<!-- Start Server Selection [server] -->
## Server Selection

### Select Server by Index

You can override the default server globally using the `WithServerIndex` option when initializing the SDK client instance. The selected server will then be used as the default on the operations that use it. This table lists the indexes associated with the available servers:

| # | Server | Variables |
| - | ------ | --------- |
| 0 | `http://localhost:8080` | None |
| 1 | `https://openmeter.cloud` | None |

#### Example

```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
	"openapi/types"
	"time"
)

func main() {
	s := openapi.New(
		openapi.WithServerIndex(1),
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


### Override Server URL Per-Client

The default server can also be overridden globally using the `WithServerURL` option when initializing the SDK client instance. For example:
```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
	"openapi/types"
	"time"
)

func main() {
	s := openapi.New(
		openapi.WithServerURL("http://localhost:8080"),
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
<!-- End Server Selection [server] -->

<!-- Start Custom HTTP Client [http-client] -->
## Custom HTTP Client

The Go SDK makes API calls that wrap an internal HTTP client. The requirements for the HTTP client are very simple. It must match this interface:

```go
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
```

The built-in `net/http` client satisfies this interface and a default client based on the built-in is provided by default. To replace this default with a client of your own, you can implement this interface yourself or provide your own client configured as desired. Here's a simple example, which adds a client with a 30 second timeout.

```go
import (
	"net/http"
	"time"
	"github.com/myorg/your-go-sdk"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	sdkClient  = sdk.New(sdk.WithClient(httpClient))
)
```

This can be a convenient way to configure timeouts, cookies, proxies, custom headers, and other low-level configuration.
<!-- End Custom HTTP Client [http-client] -->

<!-- Start Authentication [security] -->
## Authentication

### Per-Client Security Schemes

This SDK supports the following security scheme globally:

| Name          | Type          | Scheme        |
| ------------- | ------------- | ------------- |
| `PortalToken` | http          | HTTP Bearer   |

You can configure it using the `WithSecurity` option when initializing the SDK client instance. For example:
```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/types"
	"time"
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
<!-- End Authentication [security] -->

<!-- Start Special Types [types] -->
## Special Types


<!-- End Special Types [types] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This SDK is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

While we value open-source contributions to this SDK, this library is generated programmatically.
Feel free to open a PR or a Github issue as a proof of concept and we'll do our best to include it in a future release!

### SDK Created by [Speakeasy](https://docs.speakeasyapi.dev/docs/using-speakeasy/client-sdks)
