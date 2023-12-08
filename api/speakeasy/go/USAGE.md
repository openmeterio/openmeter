<!-- Start SDK Example Usage [usage] -->
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