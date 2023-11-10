# OpenMeter Go SDK

## Install

```sh
go get github.com/openmeterio/openmeter/api/client/go
```

## Usage

Initialize client.

```go
import (
  cloudevents "github.com/cloudevents/sdk-go/v2/event"
  om "github.com/openmeterio/openmeter/api/client/go"
)

func main() {
  // Initialize OpenMeter client
  om, err := openmeter.NewClient("http://localhost:8888")
  if err != nil {
      panic(err.Error())
  }

  // Use OpenMeter client
  // ...
}
```

### Ingest Event

Report usage to OpenMeter.

```go
e := cloudevents.New()
e.SetID("00001")
e.SetSource("my-app")
e.SetType("tokens")
e.SetSubject("user-id")
e.SetTime(time.Now())
e.SetData("application/json", map[string]string{
  "tokens": "15",
  "model": "gpt-4",
})

_, err := client.IngestEvent(ctx, e)
```

### Query Meter

Retreive usage from OpenMeter.

```go
slug := "token-usage"
subject := []string{"user-id"}
from, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
to, _ := time.Parse(time.RFC3339, "2023-01-02T00:00:00Z")
resp, _ := client.QueryMeter(ctx, slug, &om.QueryMeterParams{
    Subject: &subject,
    From:    &from,
    To:      &to,
})
payload, _ := om.ParseQueryMeterResponse(resp)
```
