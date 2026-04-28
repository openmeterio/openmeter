# Insomnia E2E Tests

Covers API lifecycle and validation scenarios using Insomnia and the `inso` CLI.

**Collections:**
- `openmeter-v3-product-catalog.json` — plans, addons, plan-addons (mirrors `addons_v3_test.go`, `plans_v3_test.go`, `planaddons_v3_test.go`)
- `openmeter-v3-currencies.json` — custom currencies and cost bases
- `openmeter-v3-meters.json` — meter CRUD lifecycle and validation

## Prerequisites

```bash
# Install inso CLI (requires Node.js 18+)
npm install -g @insomnia/inso

# Or via Homebrew
brew install inso

# Verify
inso --version
```

The server must be running before any test suite executes. Start local dependencies:

```bash
make up          # Postgres, Kafka, ClickHouse
make server      # OpenMeter API server (hot-reload)
```

## Configuration

Test scripts read configuration from environment variables with built-in defaults:

| Variable       | Default                   | Purpose             |
|----------------|---------------------------|---------------------|
| `OM_BASE_URL`  | `http://localhost:8888`   | API server base URL |
| `OM_API_KEY`   | *(empty)*                 | Bearer token        |

The default values work for a locally running server with no auth.

## Running Tests

The export file is passed to the **global** `-w/--workingDir` flag. In inso v12+, an environment
must be selected; pass `--env "Local Dev"` (the Insomnia environment name in the export).
Configuration comes from `OM_BASE_URL` / `OM_API_KEY` process environment variables, not from
the Insomnia environment data.

### Single suite

```bash
# Product catalog
inso --ci -w e2e/insomnia/openmeter-v3-product-catalog.json run test "Addon Lifecycle" --env "Local Dev"
inso --ci -w e2e/insomnia/openmeter-v3-product-catalog.json run test "Plan Lifecycle" --env "Local Dev"
inso --ci -w e2e/insomnia/openmeter-v3-product-catalog.json run test "Plan-Addon Lifecycle" --env "Local Dev"
inso --ci -w e2e/insomnia/openmeter-v3-product-catalog.json run test "Validation Error Cases" --env "Local Dev"

# Currencies
inso --ci -w e2e/insomnia/openmeter-v3-currencies.json run test "Currency Lifecycle" --env "Local Dev"
inso --ci -w e2e/insomnia/openmeter-v3-currencies.json run test "Currency Validation Errors" --env "Local Dev"

# Meters
inso --ci -w e2e/insomnia/openmeter-v3-meters.json run test "Meter Lifecycle" --env "Local Dev"
inso --ci -w e2e/insomnia/openmeter-v3-meters.json run test "Meter Validation" --env "Local Dev"
```

### Against a remote server

```bash
OM_BASE_URL=https://your-server OM_API_KEY=your-token \
  inso --ci -w e2e/insomnia/openmeter-v3-product-catalog.json run test --env "Local Dev"
```

### Shorthand with .insorc

To avoid repeating `-w` on every invocation, add an `.insorc` file in the same directory:

```yaml
workingDir: openmeter-v3-product-catalog.json
```

Then run from `e2e/insomnia/`:

```bash
cd e2e/insomnia
inso --ci run test --env "Local Dev"
inso --ci run test "Addon Lifecycle" --env "Local Dev"
```

## Test Suites

### Addon Lifecycle

| Test | Scenarios covered |
|------|-------------------|
| Full lifecycle | create → get → list → update → publish → archive → delete, with delete-while-active guard |
| Versioning and auto-archive | publish v2 archives v1; `effective_to` equals v2 `effective_from` |
| Mixed rate cards round-trip | flat + unit (with percentage discount) + graduated tiers survive publish + GET |

### Plan Lifecycle

| Test | Scenarios covered |
|------|-------------------|
| Full lifecycle | create → get → list → update (rename phase, add rate card) → publish → update-after-publish rejected → archive → delete |
| Versioning and auto-archive | same as addon versioning |
| Invalid draft | create accepted, `validation_errors` on GET, publish rejected, fix via PUT, republish succeeds |

### Plan-Addon Lifecycle

| Test | Scenarios covered |
|------|-------------------|
| Attach lifecycle | create plan (2 phases) + publish addon → attach → get → list → update `from_plan_phase` → detach → verify list empty |
| Publish plan with attached addon | junction survives plan publish; detach from active plan rejected |
| Duplicate attachment | second attach with same addon returns 409 |

### Validation Error Cases

| Test | Scenarios covered |
|------|-------------------|
| Plan invalid currency | `ZZZ` → 4xx error (server currently returns 500; should be 400 `currency_invalid`) |
| Plan zero phases | empty phases array → 400 schema `min_items` |
| Plan duplicate phase key | same key on two phases → 400 `plan_phase_duplicated_key` |
| Plan publish-time validations | non-last phase missing duration, last phase with duration, phase with zero rate cards — all accepted as draft, rejected at publish |
| Addon fake feature id | non-existent feature ID → 400 with feature ID in detail message |
| Plan-addon attach status matrix | draft+active → 201; active plan → 400; draft addon → 400 |

---

## openmeter-v3-currencies.json

### Currency Lifecycle

| Test | Scenarios covered |
|------|-------------------|
| Custom currency: create → list filtered → verify fiat currencies present | create custom currency → list with `filter[type]=custom` (only custom) → list with `filter[type]=fiat` (USD present, no custom) → list unfiltered (both) |
| Cost basis: create → list → filter by fiat_code | create currency → create USD cost basis → create EUR cost basis → list all → filter by `filter[fiat_code]=USD` → create with explicit future `effective_from` |

### Currency Validation Errors

| Test | Scenarios covered |
|------|-------------------|
| CreateCurrency: missing required fields → 400; duplicate code → 409 | missing name → 400; missing code → 400; missing symbol → 400; duplicate code → 409 |
| CreateCostBasis: invalid inputs → 400; non-existent currency → 409; past effective_from → 400 | zero rate → 400; negative rate → 400; missing fiat_code → 400; past `effective_from` → 400; non-existent currency ID → 409 |

---

## openmeter-v3-meters.json

### Meter Lifecycle

| Test | Scenarios covered |
|------|-------------------|
| Meter lifecycle | create (sum + value_property + dimension) → get → list → filter by key → update name/description/dimensions → delete → 404 |

### Meter Validation

| Test | Scenarios covered |
|------|-------------------|
| Aggregation rules | count without value_property → 201; count with value_property → 400; sum without value_property → 400 |
| Reserved dimensions | dimension key `subject` → 400; dimension key `customer_id` → 400 |

---

## Requests (Manual Exploration)

The `openmeter-v3-product-catalog.json` collection also contains individual requests grouped by
domain for manual use in the Insomnia desktop app:

- **Addons**: Create, Get, List, Update, Publish, Archive, Delete
- **Plans**: Create, Get, List, Update, Publish, Archive, Delete
- **Plan Addons**: Attach, Get, List, Update, Detach

Set `addon_id`, `plan_id`, `plan_addon_id`, and `phase_key` in the Insomnia environment when
sending individual requests.

## How Test Scripts Work

Each unit test is a self-contained JavaScript function that:

1. Reads `OM_BASE_URL` and `OM_API_KEY` from `process.env` (with defaults for local dev).
2. Generates a unique suffix (`timestamp_rand`) so tests never collide on a shared database.
3. Issues HTTP calls using the global `fetch` API.
4. Uses standard chai `expect(value, 'label').to.equal(x)` assertions.

All lifecycle operations run inside a single mocha `it()` callback so state (IDs, keys) flows
naturally between steps.

## Importing into Insomnia Desktop

1. Open Insomnia.
2. **File → Import** (or drag-and-drop) the desired collection JSON file.
3. Select the **Local Dev** environment.
4. Open the **Tests** tab to run individual suites interactively.
