# [Experimental] OpenMeter Credits

OpenMeter can handle complex scenarios related to credit management, usage, and rate limits.
You can grant credits via API for specific subjects with priorities, effective dates, and expiration dates.
OpenMeter then burns down credits in order based on these parameters and the ingested usage, ensuring optimal performance.

OpenMeter Credits can help to implement:

- Monthly usage-limit enforcement
- Implementing prepaid credits
- Combining recurring credits with top-ups
- Rolling over remaining credits between billing periods

## Quickstart

Run docker compose with `--profile postgres` and enable credits in `config.yaml` via `credits: true`
