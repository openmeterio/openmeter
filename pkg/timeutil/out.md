# intro

Dump of test case outputs.

We are generating period starting from an anchor.

Example output:
```
iteration[0]:
	exp: [2024-12-31T00:00:00Z..2025-01-31T00:00:00Z]
	got: [2024-12-31T00:00:00Z..2025-01-31T00:00:00Z]
	v2:  [2024-12-31T00:00:00Z..2025-01-31T00:00:00Z]
```

iteration: which period's index
exp: expected output
got: current implementation
v2: proposed new implementation


# testcases

## Should return periods for monthly recurrence

Anchor: `2024-12-31T00:00:00Z`
Recurrence: 1month

What it validates: months have different lengths, but we want to align to the end of the month and not overflow to the next month.

iteration[0]:
	exp: [2024-12-31T00:00:00Z..2025-01-31T00:00:00Z]
	got: [2024-12-31T00:00:00Z..2025-01-31T00:00:00Z]
	v2:  [2024-12-31T00:00:00Z..2025-01-31T00:00:00Z]
iteration[1]:
	exp: [2025-01-31T00:00:00Z..2025-02-28T00:00:00Z]
	got: [2025-01-31T00:00:00Z..2025-03-03T00:00:00Z]
	v2:  [2025-01-31T00:00:00Z..2025-02-28T00:00:00Z]
iteration[2]:
	exp: [2025-02-28T00:00:00Z..2025-03-31T00:00:00Z]
	got: [2025-03-03T00:00:00Z..2025-04-03T00:00:00Z]
	v2:  [2025-02-28T00:00:00Z..2025-03-31T00:00:00Z]
iteration[3]:
	exp: [2025-03-31T00:00:00Z..2025-04-30T00:00:00Z]
	got: [2025-04-03T00:00:00Z..2025-05-03T00:00:00Z]
	v2:  [2025-03-31T00:00:00Z..2025-04-30T00:00:00Z]


## Leap year handling

Anchor: 2024-02-29T00:00:00Z
Period: 1 Year

Expectation: 2025 is not a leap year, so next period ends on 28th instead of 29th.

iteration[0]:
	exp: [2024-02-29T00:00:00Z..2025-02-28T00:00:00Z]
	got: [2024-02-29T00:00:00Z..2025-03-01T00:00:00Z]
	v2:  [2024-02-29T00:00:00Z..2025-02-28T00:00:00Z]
iteration[1]:
	exp: [2025-02-28T00:00:00Z..2026-02-28T00:00:00Z]
	got: [2025-03-01T00:00:00Z..2026-03-01T00:00:00Z]
	v2:  [2025-02-28T00:00:00Z..2026-02-28T00:00:00Z]


## Daylight savings changes - anchor has timezone information

Anchor: `2025-02-01T13:00:00+01:00` (BP Time)
Period: 1 month

DST change preserves the time of day information

iteration[0]:
	exp: [2025-02-01T13:00:00+01:00..2025-03-01T13:00:00+01:00]
	got: [2025-02-01T13:00:00+01:00..2025-03-01T13:00:00+01:00]
	v2:  [2025-02-01T13:00:00+01:00..2025-03-01T13:00:00+01:00]
iteration[1]:
	exp: [2025-03-01T13:00:00+01:00..2025-04-01T13:00:00+02:00]
	got: [2025-03-01T13:00:00+01:00..2025-04-01T13:00:00+02:00]
	v2:  [2025-03-01T13:00:00+01:00..2025-04-01T13:00:00+02:00]
iteration[2]:
	exp: [2025-04-01T13:00:00+02:00..2025-05-01T13:00:00+02:00]
	got: [2025-04-01T13:00:00+02:00..2025-05-01T13:00:00+02:00]
	v2:  [2025-04-01T13:00:00+02:00..2025-05-01T13:00:00+02:00]


## Daylight savings changes - anchor in UTC

Anchor: `2025-02-01T12:00:00Z` (UTC)
Period: 1 month
Output timezone: DST

Q: Are we fine with this?
By omitting the anchor's TZ information (e.g. moving to UTC) we will loose the possiblity to keep the time part the same in the customer's TZ.

iteration[0]:
	exp: [2025-02-01T13:00:00+01:00..2025-03-01T13:00:00+01:00]
	got: [2025-02-01T13:00:00+01:00..2025-03-01T13:00:00+01:00]
	v2:  [2025-02-01T13:00:00+01:00..2025-03-01T13:00:00+01:00]
iteration[1]:
	exp: [2025-03-01T13:00:00+01:00..2025-04-01T14:00:00+02:00]
	got: [2025-03-01T13:00:00+01:00..2025-04-01T14:00:00+02:00]
	v2:  [2025-03-01T13:00:00+01:00..2025-04-01T14:00:00+02:00]
iteration[2]:
	exp: [2025-04-01T14:00:00+02:00..2025-05-01T14:00:00+02:00]
	got: [2025-04-01T14:00:00+02:00..2025-05-01T14:00:00+02:00]
	v2:  [2025-04-01T14:00:00+02:00..2025-05-01T14:00:00+02:00]


## Leap second handling

Works.

iteration[0]:
	exp: [2016-11-30T00:00:00Z..2016-12-30T00:00:00Z]
	got: [2016-11-30T00:00:00Z..2016-12-30T00:00:00Z]
	v2:  [2016-11-30T00:00:00Z..2016-12-30T00:00:00Z]
iteration[1]:
	exp: [2016-12-30T00:00:00Z..2017-01-30T00:00:00Z]
	got: [2016-12-30T00:00:00Z..2017-01-30T00:00:00Z]
	v2:  [2016-12-30T00:00:00Z..2017-01-30T00:00:00Z]
iteration[2]:
	exp: [2017-01-30T00:00:00Z..2017-02-28T00:00:00Z]
	got: [2017-01-30T00:00:00Z..2017-03-02T00:00:00Z]
	v2:  [2017-01-30T00:00:00Z..2017-02-28T00:00:00Z]
iteration[3]:
	exp: [2017-02-28T00:00:00Z..2017-03-30T00:00:00Z]
	got: [2017-03-02T00:00:00Z..2017-04-02T00:00:00Z]
	v2:  [2017-02-28T00:00:00Z..2017-03-30T00:00:00Z]
