# AIP-142 — Time & duration

Reference: https://kong-aip.netlify.app/aip/142/

## Timestamps (AIP-compliant)

- **Field name pattern**: `<past_participle>_at` for fields representing a point in time — e.g., `created_at`, `updated_at`, `deleted_at`, `expires_at`
- **Wire format**: RFC-3339 string in UTC with `Z` suffix, e.g., `"2023-02-27T02:15:00Z"`
- `T` separates date and time; fractional seconds are optional

## Durations (OpenMeter deviates from AIP-142)

AIP-142 specifies durations as **integer values with a unit suffix in the field name**: `ttl_ms`, `flight_duration_mins`, `lifespan_yrs`. Permitted suffixes: `ns`, `ms`, `secs`, `mins`, `hrs`, `days`, `yrs`. Values must be non-negative integers within `0 <= N < 2^53`.

**OpenMeter does not follow AIP-142 for durations.** Instead, OpenMeter uses the **ISO-8601 duration format** as an opaque string:

- `PT1M` — one minute
- `PT1H` — one hour
- `P1D` — one day
- `P1M` — one month

Components: years (`Y`), months (`M`), weeks (`W`), days (`D`), time separator `T`, hours (`H`), minutes (`M` after `T`), seconds (`S`). When authoring a new duration field in OpenMeter, use the ISO-8601 form and do **not** append a unit suffix to the field name. This deviation is intentional and documented here so reviewers know to allow it.
