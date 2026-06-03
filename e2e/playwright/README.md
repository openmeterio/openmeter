# Playwright e2e — v3 shim smoke

Live-server smoke test for the `@openmeter/sdk` **v3 compatibility shim**
(`om.v3.*`, under `api/client/javascript/src/v3/`). It is the TypeScript parity
of `e2e/productcatalog_smoke_v3_test.go`: the same plan + addon authoring flow,
but driven through the shim instead of raw Go HTTP.

This uses [Playwright's API-testing mode](https://playwright.dev/docs/api-testing)
— Playwright is only the runner (`test`/`expect`, no browser, no `webServer`).
The client under test is the shim itself, over Node's `fetch`. Dependencies are
deliberately minimal: Playwright plus the local `@openmeter/sdk`.

## Run it

1. Start a local OpenMeter (the same server the Go e2e suite uses):

   ```bash
   make -C .. env-local-up   # from e2e/: brings up docker-compose, server on :38888
   ```

2. Build the SDK (the shim is consumed from its built `dist/`) and install:

   ```bash
   pnpm install              # links @openmeter/sdk via file:
   pnpm run build:sdk        # builds api/client/javascript -> dist/
   ```

3. Run the smoke:

   ```bash
   pnpm test                 # OPENMETER_ADDRESS defaults to http://localhost:38888
   ```

   `pnpm run test:full` does build:sdk + install + test in one shot.

Point at a different server with `OPENMETER_ADDRESS=https://host pnpm test`.
There is no auth locally; set an API key by constructing the client with
`{ baseUrl, apiKey }` if you target an authenticated environment.

## Notes

- The shim surfaces the v3 wire shape **verbatim** (snake_case, "Option A"), so
  responses read with snake_case fields (`validation_errors`, `effective_from`,
  `from_plan_phase`, …) — assertions match the Go test's wire assertions.
- Not wired into CI yet. To run in CI, build the SDK and run this against the
  same docker-compose server the `e2e` job already stands up.
