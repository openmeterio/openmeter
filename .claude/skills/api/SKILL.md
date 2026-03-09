---
name: api
description: Add or modify API endpoints using TypeSpec. Use when adding new API routes, modifying request/response types, or changing the OpenAPI spec.
user-invocable: true
argument-hint: "[description of API change]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# API Development

You are helping the user add or modify API endpoints in OpenMeter.

## Context

- **API spec source:** `api/spec/src/` — TypeSpec definitions
- **New APIs go in v3:** `api/spec/src/v3/` — all new endpoints must be added here
- **Generated outputs (DO NOT edit manually):**
  - `api/openapi.yaml`, `api/openapi.cloud.yaml` — OpenAPI specs
  - `api/client/javascript/`, `api/client/go/` — SDK clients
  - `api/api.gen.go`, `api/v3/api.gen.go` — Go server code (oapi-codegen)

## V3 API Structure

```text
api/spec/src/v3/
├── main.tsp              # Top-level imports
├── openmeter.tsp         # Service definition, routes, and interface wiring
├── konnect.tsp           # Konnect-specific service definition
├── common/               # Shared types: errors, pagination, parameters
├── shared/               # Shared resources: ULID, request/response wrappers, tags
├── meters/               # Domain: models + operations
├── customers/            # Domain: models + operations
├── subscriptions/        # ...
├── billing/
├── apps/
├── currencies/
├── llmcost/
└── ...
```

Each domain typically has:

- `index.tsp` — imports for the domain
- `<resource>.tsp` — model/type definitions
- `operations.tsp` — interface with CRUD operations

Routes are wired in `openmeter.tsp` via interface declarations with `@route` and `@tag` decorators.

## Workflow

Follow these steps in order:

### Step 1: Edit the TypeSpec API spec

For a new domain/resource:

1. Create a new directory under `api/spec/src/v3/<domain>/`
2. Add `index.tsp`, model file(s), and `operations.tsp`
3. Import the domain in `api/spec/src/v3/openmeter.tsp`
4. Wire up the route interface in `openmeter.tsp`

For modifying an existing endpoint:

1. Find the relevant files under `api/spec/src/v3/<domain>/`
2. Edit the model or operations as needed

Look at existing domains (e.g., `meters/`, `customers/`) for conventions:

- Use `Shared.CreateRequest<T>`, `Shared.GetResponse<T>`, `Shared.PagePaginatedResponse<T>` wrappers
- Use `Common.ErrorResponses`, `Common.NotFound` for error types
- Use `Common.PagePaginationQuery` for list operations
- Use `@operationId`, `@summary`, `@tag` decorators on operations
- Use `Shared.ULID` for resource IDs in path parameters
- Routes follow the pattern `/openmeter/<resource>`

### Step 2: Generate API code

Run:

```bash
make gen-api
```

This generates the OpenAPI spec, SDK clients, and Go server stubs. Check that it completes without errors.

Then run:

```bash
make generate
```

This regenerates Go server code from the updated OpenAPI spec (oapi-codegen).

### Step 3: Implement the handler

After generating, you'll need to implement the new handler methods in the Go server code. The generated interfaces will show what methods need to be implemented.

The handlers are located at `api/v3/handlers`, you also need to connect handlers at `api/v3/server/routes.go`.

### Step 4: Review

- Check the generated `api/openapi.yaml` or `api/v3/api.gen.go` to verify the endpoints look correct
- Present a summary of the API changes to the user

## Important Reminders

- All new APIs go in v3 (`api/spec/src/v3/`)
- Never edit generated files manually (`api/openapi.yaml`, `api/client/`, `api/*.gen.go`)
- Run `make gen-api` to generate Go types
- Follow existing TypeSpec patterns and conventions from other v3 domains
