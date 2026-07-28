# Package Refactoring

Use this workflow when refactoring an existing `openmeter/` package toward the
standard service/adapter pattern.

## Target Pattern

See [Service Package Development](service-patterns.md) for the full target
pattern. In summary, a conventional feature package has:

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
- **Check consumers**: Before moving types, use CodeGraph to inspect callers and
  impact. Fall back to source search when the graph is inconclusive.
- **Wire regeneration**: After changing constructors or interfaces, always run `make generate` to update `wire_gen.go` files.
