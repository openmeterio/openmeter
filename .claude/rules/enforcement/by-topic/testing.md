# Enforcement: testing (6 rules)

Topic file. Loaded on demand when an agent works on something in the `testing` area. The pre-edit hook reads `.archie/rules.json` directly — this file is for browsing/context only.

## Pitfalls (block)

### `pf-test-cycle-001` — Keep test/billing non-test files free of imports into charges/service and subscriptionsync/service

*source: `deep_scan`*

**Why:** Pitfall pf_0014: internal test packages (package service) in openmeter/billing/charges/service and openmeter/billing/worker/subscriptionsync/service import the shared test/billing fixtures, which in turn imports those same service packages from its own _test files — a test-only import cycle. Go tolerates it only because the back-edges live in _test files. Moving a back-edge import into a non-test file of test/billing (e.g. a fixture helper constructing a real subscriptionsync service) would promote it into a hard production import cycle that breaks the build.

## Pattern Divergence (inform)

### `place-test-001` — Co-locate unit tests as <name>_test.go; put cross-domain integration harnesses under top-level test/<domain>/ and e2e under e2e/

*source: `deep_scan`*

**Why:** Most tests are co-located *_test.go files. Heavier cross-cutting integration suites and shared harnesses live under a top-level test/ tree organized by domain (test/billing, test/customer, test/subscription, ...); e.g. test/billing is imported by openmeter/billing/charges/service/*_test.go. End-to-end HTTP-over-the-wire tests live separately in e2e/ with docker-compose files.

### `prac-test-decimal-001` — Assert decimal equality in tests via require.Equal on InexactFloat64(), not boolean Equal()

*source: `deep_scan`*

**Why:** When asserting alpacadecimal.Decimal equality in tests, prefer require.Equal(t, expectedFloat64, actual.InexactFloat64()) over boolean assertions like require.True(t, expected.Equal(actual)) when precision allows. Prefer simple float64(5)-style literals over verbose decimal construction for expected values.

### `prac-test-ctx-001` — In tests use t.Context() and pair clock.FreezeTime with defer clock.UnFreeze in the same scope

*source: `deep_scan`*

**Why:** In tests, prefer t.Context() over context.Background() so cancellation and lifecycle tie to the test harness. When using clock.FreezeTime(...), immediately pair it with defer clock.UnFreeze() in the same scope so later assertions or subtests do not inherit frozen time accidentally.

**Example:**

```
clock.FreezeTime(now); defer clock.UnFreeze()
```

### `prac-test-helpers-001` — Build domain test deps from underlying constructors under testutils, independent of app/common

*source: `deep_scan`*

**Why:** Keep domain test helpers under openmeter/.../testutils independent of app/common; build test dependencies from underlying constructors (repos, adapters, services, lockr), not the wiring layer, or unrelated wiring additions can create test-only import cycles. Prefer driving UBP billing lifecycle tests through charges.Service.Create/AdvanceCharges/ApplyPatches rather than lower-level charge adapters.

### `infra-pgtest-001` — Set POSTGRES_HOST=127.0.0.1 and -tags=dynamic for DB-touching Go tests, or suites silently skip

*source: `deep_scan`*

**Why:** Always set POSTGRES_HOST=127.0.0.1 for DB-touching Go tests (and ensure Postgres is up via docker compose up -d postgres), or suites silently skip during setup. Run tests with -tags=dynamic (required for confluent-kafka-go) and the Make parallelism flags -p 128 -parallel 16.

**Example:**

```
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic -v ./openmeter/billing/...
```
