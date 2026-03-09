---
name: refactoring
description: Refactor existing packages toward the standard service/adapter pattern. Use when restructuring a domain package, splitting a monolithic package, or removing anti-patterns.
user-invocable: true
argument-hint: "[package to refactor or description of refactoring]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Package Refactoring

You are helping the user refactor existing `openmeter/` packages toward the standard service/adapter pattern described in the `/service` skill.

## Target Pattern

See the `/service` skill for the full target pattern. In summary, every feature package should have:

```text
openmeter/<domain>/
├── service.go          # Service interface definition
├── adapter.go          # Adapter interface definition
├── <domain>.go         # Domain types and models
├── errors.go           # Custom errors (optional, only when needed)
├── event.go            # Domain events (optional, for packages that modify DB entities)
├── adapter/            # Adapter layer implementation (data access)
│   ├── adapter.go      # Config, New(), transaction boilerplate
│   ├── <operation>.go  # One file per operation (list.go, get.go, create.go, etc.)
│   └── mapping.go      # Entity ↔ domain type mapping functions
├── service/            # Service layer implementation (business logic + orchestration)
│   └── service.go
├── driver/             # v1 API, do not implement for new services (also called: httpdriver, driver)
│   └── <operation>.go
```

Key rules:

- All types and interfaces in root package
- Service = business rules + orchestration
- Adapter = pure data access
- No deep nesting, no connectors, no global state

## Packages Needing Refactoring

### High Priority

Complex domain packages with non-standard structure:

| Package | Issues |
|---------|--------|
| `subscription` | 30+ files in root, logic spread across root (apply, billing, context, locks, patch), specialized subdirs (addon/, entitlement/, hooks/, patch/) |
| `productcatalog` | 20+ files in root, multiple entity types mixed together (addon, plan, feature, discount, entitlement), inconsistent subdir naming (driver/ vs adapter/) |
| `billing` | 20+ files in root (invoice, customer, discount, app), complex domain mixed into single package |
| `app` | Heavy root (app, appbase, customer, marketplace, webhook, registry, input), multiple impl subdirs (stripe/, sandbox/, custominvoicing/) |
| `credit` | Domain split across balance/, grant/, engine/ subdirs with connector pattern in root |
| `entitlement` | Has adapter/service but also boolean/, metered/, static/, snapshot/, balanceworker/, hooks/ — uses connector pattern |
| `notification` | Has adapter/service but also consumer/, eventhandler/, internal/ — non-standard extensions |

### Medium Priority

Partially compliant or minor structural issues:

| Package | Issues |
|---------|--------|
| `ingest` | Non-standard adapter naming (ingestadapter/), mixed patterns (kafkaingest/, inmemory in root) |
| `streaming` | Uses connector pattern, clickhouse/ impl dir, no service/adapter split |
| `sink` | No service/adapter pattern, utility-focused with flushhandler/, models/ |

### Low Priority / Not Applicable

These are infrastructure, utility, or minimal packages where the pattern may not apply:

`ent`, `watermill`, `dedupe`, `server`, `namespace`, `registry`, `event`, `apiconverter`, `testutils`, `debug`, `session`, `info`

## Refactoring Workflow

When refactoring a package toward the standard pattern:

1. **Analyze current structure**: Read the package to understand all files, types, and dependencies. Map out which code is domain types, which is business logic, and which is data access.

2. **Identify entity boundaries**: If the package mixes multiple independent entities (e.g., `productcatalog` has plan, addon, feature), consider splitting into separate packages first.

3. **Extract root interfaces**: Move all types, interfaces, input DTOs, and errors to the root package. Remove any implementation code from root.

4. **Create adapter/**: Move all database queries, entity mapping, and Ent ORM code into `adapter/`. Ensure it only does data access — no business decisions.

5. **Create service/**: Move all business logic, orchestration, and transaction wrapping into `service/`. This includes validation beyond simple input checks, precondition enforcement, multi-step operations, and event publishing.

6. **Remove anti-patterns**: Eliminate connectors, deep nesting, scattered types. Replace global state with constructor injection.

7. **Update wiring**: Update `app/common/<domain>.go` and `cmd/*/wire.go` to match new constructor signatures. Run `make generate`.

8. **Update imports**: Fix all imports across the codebase that reference moved types or functions.

9. **Run tests**: `make test` to verify nothing is broken.

## Important Considerations

- **Incremental refactoring**: For large packages, consider refactoring in phases rather than all at once. Extract one entity or one layer at a time.
- **Preserve behavior**: Refactoring should not change any behavior. Run tests frequently during the process.
- **Check consumers**: Before moving types, check what other packages import them. Use `grep` to find all import paths.
- **Wire regeneration**: After changing constructors or interfaces, always run `make generate` to update `wire_gen.go` files.
