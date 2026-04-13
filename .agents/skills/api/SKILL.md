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

- **API spec source:** `api/spec/packages/` — TypeSpec definitions (two packages: `aip` for v3, `legacy` for v1)
- **Local server port:** The server runs on port 8888 by default locally (`http://localhost:8888/api/v3`)
- **New APIs go in AIP package:** `api/spec/packages/aip/src/` — all new endpoints must be added here
- **Generated outputs (DO NOT edit manually):**
  - `api/openapi.yaml`, `api/openapi.cloud.yaml` — OpenAPI specs
  - `api/client/javascript/`, `api/client/go/` — SDK clients
  - `api/api.gen.go`, `api/v3/api.gen.go` — Go server code (oapi-codegen)

## AIP (v3) API Structure

```text
api/spec/packages/aip/src/
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

Routes are wired in `api/spec/packages/aip/src/openmeter.tsp` via interface declarations with `@route` and `@tag` decorators.

## Workflow

Follow these steps in order:

### Step 1: Edit the TypeSpec API spec

For a new domain/resource:

1. Create a new directory under `api/spec/packages/aip/src/<domain>/`
2. Add `index.tsp`, model file(s), and `operations.tsp`
3. Import the domain in `api/spec/packages/aip/src/openmeter.tsp`
4. Wire up the route interface in `openmeter.tsp`

For modifying an existing endpoint:

1. Find the relevant files under `api/spec/packages/aip/src/<domain>/`
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

After generating, implement the handler package and wire it into the server.

#### Handler Package Structure

Each handler domain lives at `api/v3/handlers/<domain>/` and contains:

- `handler.go` — Handler interface + constructor
- `<operation>.go` — One file per operation (create.go, list.go, get.go, delete.go)
- `convert.go` — Domain ↔ API type mapping functions

Reference: `api/v3/handlers/llmcost/`

#### Handler Interface & Constructor (`handler.go`)

```go
package <domain>

type Handler interface {
    List<Resource>s() List<Resource>sHandler
    Create<Resource>() Create<Resource>Handler
    Get<Resource>() Get<Resource>Handler
    Delete<Resource>() Delete<Resource>Handler
}

type handler struct {
    resolveNamespace func(ctx context.Context) (string, error)
    service          <domain>.Service
    options          []httptransport.HandlerOption
}

func New(
    resolveNamespace func(ctx context.Context) (string, error),
    service <domain>.Service,
    options ...httptransport.HandlerOption,
) Handler {
    return &handler{
        resolveNamespace: resolveNamespace,
        service:          service,
        options:          options,
    }
}
```

Reference: `api/v3/handlers/llmcost/handler.go`

#### Handler Operation Pattern (`<operation>.go`)

Each operation file uses `httptransport.NewHandlerWithArgs` with 4 arguments:

1. **Request decoder** — parse HTTP request → domain input, resolve namespace
2. **Operation function** — call service, map result to API response type
3. **Response encoder** — `commonhttp.JSONResponseEncoderWithStatus[T](http.StatusXxx)`
4. **Options** — `httptransport.AppendOptions(h.options, httptransport.WithOperationName("..."), httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))`

> **List endpoints with filtering:** if the operation supports `?filter[...]` query parameters, use the `/api-filters` skill for the decoder and adapter wiring. It covers `api/v3/filters.Parse`, the typed filter structs, `Convert*` helpers, range splitting, and the Ent `.Select(field)` application — everything this skill does not cover.

Type alias convention at top of file:

```go
type (
    List<Resource>sRequest  = <domain>.List<Resource>sInput
    List<Resource>sResponse = response.PagePaginationResponse[api.<Resource>]
    List<Resource>sParams   = api.List<Resource>sParams
    List<Resource>sHandler  = httptransport.HandlerWithArgs[List<Resource>sRequest, List<Resource>sResponse, List<Resource>sParams]
)
```

Full example:

```go
func (h *handler) List<Resource>s() List<Resource>sHandler {
    return httptransport.NewHandlerWithArgs(
        // 1. Request decoder
        func(ctx context.Context, r *http.Request, params List<Resource>sParams) (List<Resource>sRequest, error) {
            ns, err := h.resolveNamespace(ctx)
            if err != nil {
                return List<Resource>sRequest{}, err
            }

            req := List<Resource>sRequest{
                Namespace: ns,
            }

            // Pagination
            req.Page = pagination.NewPage(1, 20)
            if params.Page != nil {
                req.Page = pagination.NewPage(
                    lo.FromPtrOr(params.Page.Number, 1),
                    lo.FromPtrOr(params.Page.Size, 20),
                )
                if err := req.Page.Validate(); err != nil {
                    return req, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
                        {Field: "page", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
                    })
                }
            }

            // Sort
            if params.Sort != nil {
                sort, err := request.ParseSortBy(*params.Sort)
                if err != nil {
                    return req, apierrors.NewBadRequestError(ctx, err, apierrors.InvalidParameters{
                        {Field: "sort", Reason: err.Error(), Source: apierrors.InvalidParamSourceQuery},
                    })
                }
                if !validSortField(sort.Field) {
                    return req, apierrors.NewBadRequestError(ctx, fmt.Errorf("unsupported sort field: %s", sort.Field), apierrors.InvalidParameters{
                        {Field: "sort", Reason: fmt.Sprintf("unsupported sort field %q", sort.Field), Source: apierrors.InvalidParamSourceQuery},
                    })
                }
                req.OrderBy = sort.Field
                req.Order = sort.Order.ToSortxOrder()
            }

            return req, nil
        },
        // 2. Operation function
        func(ctx context.Context, request List<Resource>sRequest) (List<Resource>sResponse, error) {
            result, err := h.service.List<Resource>s(ctx, request)
            if err != nil {
                return List<Resource>sResponse{}, fmt.Errorf("failed to list: %w", err)
            }

            items := lo.Map(result.Items, func(item <domain>.<Resource>, _ int) api.<Resource> {
                return domainToAPI(item)
            })

            return response.NewPagePaginationResponse(items, response.PageMetaPage{
                Size:   request.Page.PageSize,
                Number: request.Page.PageNumber,
                Total:  lo.ToPtr(result.TotalCount),
            }), nil
        },
        // 3. Response encoder
        commonhttp.JSONResponseEncoderWithStatus[List<Resource>sResponse](http.StatusOK),
        // 4. Options
        httptransport.AppendOptions(
            h.options,
            httptransport.WithOperationName("list-<resource>s"),
            httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()),
        )...,
    )
}
```

For handlers without params (e.g., Create), use `httptransport.NewHandler` (3 arguments, no params):

```go
type (
    Create<Resource>Handler = httptransport.Handler[Create<Resource>Request, Create<Resource>Response]
)
```

Reference: `api/v3/handlers/llmcost/list_prices.go`

#### Error Mapping

Domain errors auto-map to HTTP status codes via the error encoder:

- `GenericNotFoundError` → 404
- `GenericValidationError` → 400
- `GenericConflictError` → 409
- `GenericForbiddenError` → 403
- `GenericPreConditionFailedError` → 412

No need to manually handle these — just return them from the service and the error encoder handles it.

#### Structured Validation Errors (ValidationIssue)

In v3 API handlers, use `models.ValidationIssue` for structured validation errors with codes, field paths, and severity levels. This is the **handler-layer** pattern — service/adapter layers continue using `models.NewGenericValidationError()`.

```go
// Define validation issues as package-level variables
var errMissingName = models.NewValidationError("missing_name", "name is required")
var errInvalidCurrency = models.NewValidationWarning("invalid_currency", "currency not recognized")

// Use with field paths
err := errMissingName.WithPathString("body", "name")

// Convert from domain errors to structured issues
issues, err := models.AsValidationIssues(domainErr)
```

Key types from `pkg/models/validationissue.go`:

- `models.NewValidationError(code, message)` — critical severity
- `models.NewValidationWarning(code, message)` — warning severity
- `models.NewValidationIssue(code, message, opts...)` — with options
- `.WithPathString("body", "field")` — attach JSONPath field location
- `.WithComponent(component)` — attach component name
- `models.AsValidationIssues(err)` — convert error tree to structured issues

### Step 4: Wire Handler into Server

Three files to modify:

**1. `api/v3/server/server.go`:**

- Add service to `Config` struct
- Add handler field to `Server` struct
- Instantiate handler in `NewServer()` using `<domain>handler.New(resolveNamespace, config.<Domain>Service, httptransport.WithErrorHandler(config.ErrorHandler))`

**2. `api/v3/server/routes.go`:**

- Add route methods that delegate to handler:

```go
// For operations WITH params (list, get by ID, delete by ID):
func (s *Server) List<Resource>s(w http.ResponseWriter, r *http.Request, params api.List<Resource>sParams) {
    s.<domain>Handler.List<Resource>s().With(params).ServeHTTP(w, r)
}

func (s *Server) Get<Resource>(w http.ResponseWriter, r *http.Request, id api.ULID) {
    s.<domain>Handler.Get<Resource>().With(id).ServeHTTP(w, r)
}

// For operations WITHOUT params (create):
func (s *Server) Create<Resource>(w http.ResponseWriter, r *http.Request) {
    s.<domain>Handler.Create<Resource>().ServeHTTP(w, r)
}
```

**3. Import the handler package in `server.go`.**

Reference: `api/v3/server/server.go:138-218`, `api/v3/server/routes.go`

### Step 5: Review

- Check the generated `api/openapi.yaml` or `api/v3/api.gen.go` to verify the endpoints look correct
- Present a summary of the API changes to the user

## AIP Standards (Kong AIP)

OpenMeter v3 APIs follow [Kong's AIP](https://kong-aip.netlify.app/list/) conventions. Each rule lives in its own file under `rules/` next to this SKILL — open the rule file you need for the task at hand.

### Rule index

| File                              | Covers                                                                           |
| --------------------------------- | -------------------------------------------------------------------------------- |
| `rules/aip-122-naming.md`         | Naming conventions + base resource models (`Shared.Resource`)                    |
| `rules/aip-126-enums.md`          | Enum wire values, `unknown` zero member, prefer-enum-over-bool                   |
| `rules/aip-visibility.md`         | `@visibility` + `Lifecycle.Read/Create/Update`                                   |
| `rules/aip-134-135-crud.md`       | Create/Get/Update/Upsert/Delete templates, PATCH rules, DELETE rules             |
| `rules/aip-132-list.md`           | List endpoints, sort, trailing slash                                             |
| `rules/aip-158-pagination.md`     | Page-based and cursor-based pagination                                           |
| `rules/aip-160-filtering.md`      | Filter query syntax, `Common.*FieldFilter` types, label dot-notation             |
| `rules/aip-129-labels.md`         | Label key constraints, PATCH-with-null semantics                                 |
| `rules/aip-193-errors.md`         | RFC-7807 error responses, `Common.ErrorResponses`, 403-before-404 rule           |
| `rules/aip-composition.md`        | Composition-over-inheritance (spread, `model is`, `@discriminator`)              |
| `rules/aip-docs.md`               | `@doc`/`/** */` requirements, `@operationId`, `@summary`                         |
| `rules/aip-181-stability.md`      | `x-private` / `x-unstable` / `x-internal` stability markers                      |
| `rules/aip-142-time.md`           | RFC-3339 timestamps, ISO-8601 duration deviation                                 |
| `rules/aip-137-content-type.md`   | `Content-Type` validation, 415 on unsupported                                    |
| `rules/aip-235-bulk-delete.md`    | `POST .../bulk-delete` transactional vs 207 partial                              |
| `rules/aip-3101-versioning.md`    | URL-path versioning, per-resource versioning                                     |
| `rules/aip-3106-empty-fields.md`  | Always return all fields, `null` / `[]` / `{}` for empty                         |

For filtering specifically, `rules/aip-160-filtering.md` covers the **TypeSpec side** (which `Common.*FieldFilter` to pick, `Shared.ResourceFilters`, label dot-notation, `deepObject` exposure). The **Go implementation side** — parsing deepObject query params into typed filters, converting to `pkg/filter`, and applying Ent predicates — is in the `/api-filters` skill.

## Important Reminders

- All new APIs go in the AIP package (`api/spec/packages/aip/src/`)
- Legacy/v1 APIs are in `api/spec/packages/legacy/src/` — avoid adding new endpoints there
- Never edit generated files manually (`api/openapi.yaml`, `api/client/`, `api/*.gen.go`)
- Run `make gen-api` to generate Go types
- Follow existing TypeSpec patterns and conventions from other AIP domains
- When implementing the service layer, use the `/service` skill for service/adapter patterns
