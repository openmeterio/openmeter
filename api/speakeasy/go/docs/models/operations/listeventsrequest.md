# ListEventsRequest


## Fields

| Field                                           | Type                                            | Required                                        | Description                                     |
| ----------------------------------------------- | ----------------------------------------------- | ----------------------------------------------- | ----------------------------------------------- |
| `From`                                          | [*time.Time](https://pkg.go.dev/time#Time)      | :heavy_minus_sign:                              | Start date-time in RFC 3339 format.<br/>Inclusive.<br/> |
| `To`                                            | [*time.Time](https://pkg.go.dev/time#Time)      | :heavy_minus_sign:                              | End date-time in RFC 3339 format.<br/>Inclusive.<br/> |
| `Limit`                                         | **int64*                                        | :heavy_minus_sign:                              | Number of events to return.                     |