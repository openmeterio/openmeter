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

Run docker compose with `--profile postgres` and enable entitlements in `config.yaml` via
```
entitlements:
    enabled: true
```
## Test

Credit tests require a Postgres database. The recommended way to run tests is to use `make test`, which runs the necessary dependencies via Dagger.
If you need to iterate on credit tests quickly, you can run your own Postgres instance (or use the one run by docker-compose)
and run  tests manually as: `POSTGRES_HOST=localhost go test ./internal/credit/...`
