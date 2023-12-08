# MeterQueryRow


## Fields

| Field                                     | Type                                      | Required                                  | Description                               |
| ----------------------------------------- | ----------------------------------------- | ----------------------------------------- | ----------------------------------------- |
| `Value`                                   | *float64*                                 | :heavy_check_mark:                        | N/A                                       |
| `WindowStart`                             | [time.Time](https://pkg.go.dev/time#Time) | :heavy_check_mark:                        | N/A                                       |
| `WindowEnd`                               | [time.Time](https://pkg.go.dev/time#Time) | :heavy_check_mark:                        | N/A                                       |
| `Subject`                                 | **string*                                 | :heavy_minus_sign:                        | The subject of the meter value.           |
| `GroupBy`                                 | map[string]*string*                       | :heavy_minus_sign:                        | N/A                                       |