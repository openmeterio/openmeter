# OpenMeter Performance Testing

## Setup

```bash
pnpm install
pnpm build
```

## Running the tests

Before running tests, make sure that

- the dependencies are running (`make up`)
- the server and the worker are running (`make server` and `make sink-worker`)
- the server is seeded with data (`make seed`)

```bash
# Run all tests
./run-all.sh

# Run a single test
k6 run dist/$TEST_NAME.js

# To create timestamped json reports under ./reports
k6 run --env CREATE_JSON_REPORT=true dist/$TEST_NAME.js
```

Configurable values are in `src/shared/config.ts`.

## How to Contribute

Test scenarios are in `src/tests/` ending in `.test.ts`. They get compiled and bundled and then are subsequently run with `k6`. Currently tooling is not present for measuring performance degradation, the current tests are A/B tests comparing alternative implementations of the same functionality.
