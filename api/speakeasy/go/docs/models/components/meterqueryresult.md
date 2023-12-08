# MeterQueryResult


## Fields

| Field                                                                  | Type                                                                   | Required                                                               | Description                                                            |
| ---------------------------------------------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------- | ---------------------------------------------------------------------- |
| `From`                                                                 | [*time.Time](https://pkg.go.dev/time#Time)                             | :heavy_minus_sign:                                                     | N/A                                                                    |
| `To`                                                                   | [*time.Time](https://pkg.go.dev/time#Time)                             | :heavy_minus_sign:                                                     | N/A                                                                    |
| `WindowSize`                                                           | [*components.WindowSize](../../models/components/windowsize.md)        | :heavy_minus_sign:                                                     | Aggregation window size.                                               |
| `Data`                                                                 | [][components.MeterQueryRow](../../models/components/meterqueryrow.md) | :heavy_check_mark:                                                     | N/A                                                                    |