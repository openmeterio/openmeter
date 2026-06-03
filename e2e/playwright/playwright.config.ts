import { defineConfig } from '@playwright/test'

// API-testing config for the @openmeter/sdk v3 shim smoke test. There is no
// browser project and no webServer: the tests drive the shim (`om.v3.*`) over
// Node's fetch against an already-running OpenMeter server (the same one the Go
// e2e suite targets — `make -C e2e env-local-up`, OPENMETER_ADDRESS).
export default defineConfig({
  testDir: './tests',
  // The smoke flow is inherently ordered (describe.serial) and shares state, so
  // a single worker keeps runs deterministic and avoids key collisions.
  workers: 1,
  fullyParallel: false,
  // No retries — a flaky smoke should fail loudly, not be papered over.
  retries: 0,
  reporter: 'list',
  // Bound a hung server the way the Go suite's v3RequestTimeout does.
  timeout: 60_000,
  expect: { timeout: 30_000 },
})
