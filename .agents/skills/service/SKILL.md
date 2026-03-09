---
name: service
description: Create or modify a service package following OpenMeter conventions. Use when building new domain packages or modifying existing service/adapter layers.
user-invocable: true
argument-hint: "[description of service to create or modify]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Service Package Development

You are helping the user create or modify a service package in OpenMeter following established conventions.

## Package Structure

Each domain package lives under `openmeter/<domain>/` and follows this structure:

```text
openmeter/<domain>/
├── service.go          # Service interface definition
├── adapter.go          # Adapter interface definition
├── <domain>.go         # Domain types and models
├── errors.go           # Custom errors (optional, only when needed)
├── event.go            # Domain events (optional, for packages that modify DB entities)
├── adapter/            # Adapter layer implementation (business logic)
│   ├── adapter.go      # Config, New(), transaction boilerplate
│   ├── <operation>.go  # One file per operation (list.go, get.go, create.go, etc.)
│   └── mapping.go      # Entity ↔ domain type mapping functions
├── service/            # Service layer implementation (thin orchestration layer)
│   └── service.go
├── driver/             # v1 API, do not implement for new services (also called: httpdriver, driver)
│   └── <operation>.go
api/v3/handlers/<domain>/
└── <api_operation>/    # The API operation defined in API spec
```

## Interfaces: service.go and adapter.go

### service.go — Service Interface

Defines the public API of the domain. This is what other packages depend on.

See `openmeter/customer/service.go` and `openmeter/llmcost/service.go` for examples.

```go
package <domain>

type Service interface {
    List<Resource>s(ctx context.Context, input List<Resource>sInput) (pagination.Result[<Resource>], error)
    Create<Resource>(ctx context.Context, input Create<Resource>Input) (*<Resource>, error)
    Get<Resource>(ctx context.Context, input Get<Resource>Input) (*<Resource>, error)
    Update<Resource>(ctx context.Context, input Update<Resource>Input) (*<Resource>, error)
    Delete<Resource>(ctx context.Context, input Delete<Resource>Input) error
}
```

### adapter.go — Adapter Interface

Defines the persistence layer contract. Implements DB access using ent ORM.

See `openmeter/customer/adapter.go` and `openmeter/llmcost/adapter.go` for examples.

```go
package <domain>

type Adapter interface {
    entutils.TxCreator
    // Same methods as Service, plus any internal-only persistence methods
}
```

The adapter interface typically mirrors the service interface but may include additional internal methods (e.g., `UpsertGlobalPrice`).

## Input Types and Validation

All input structs MUST have a `Validate()` method. Follow these patterns:

- Use `models.NewNillableGenericValidationError(errors.Join(errs...))` to return validation errors
- Implement `models.Validator` interface (compile-time check with `var _ models.Validator = (*MyInput)(nil)`)
- Validate all required fields and return collected errors

See `openmeter/llmcost/service.go` for comprehensive validation examples.

```go
var _ models.Validator = (*Create<Resource>Input)(nil)

type Create<Resource>Input struct {
    Namespace string
    Name      string
}

func (i Create<Resource>Input) Validate() error {
    var errs []error

    if i.Namespace == "" {
        errs = append(errs, fmt.Errorf("namespace is required"))
    }

    if i.Name == "" {
        errs = append(errs, fmt.Errorf("name is required"))
    }

    return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

## Service Layer Implementation (`service/`)

The service layer is a **thin orchestration layer**. It:

- Runs input validation via request validators when applicable
- Wraps adapter calls in transactions
- Publishes domain events after mutations
- Calls service hooks (PostCreate, PreDelete, PostDelete, PreUpdate, PostUpdate)
- Does NOT contain business logic — that belongs in the adapter

See `openmeter/customer/service/customer.go` for a full example with hooks and events.
See `openmeter/llmcost/service/service.go` for a simpler passthrough example.

Constructor patterns:

- Simple: `func New(adapter <domain>.Adapter, logger *slog.Logger) <domain>.Service`
- With config: `func New(config Config) (*Service, error)` where `Config` has a `Validate()` method

### Transaction Patterns in Service Layer

Use `transaction.Run()` for methods returning a value, `transaction.RunWithNoValue()` for void methods:

```go
func (s *service) Create<Resource>(ctx context.Context, input <domain>.Create<Resource>Input) (*<domain>.<Resource>, error) {
    return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*<domain>.<Resource>, error) {
        result, err := s.adapter.Create<Resource>(ctx, input)
        if err != nil {
            return nil, err
        }

        // Publish event, call hooks, etc.
        return result, nil
    })
}

func (s *service) Delete<Resource>(ctx context.Context, input <domain>.Delete<Resource>Input) error {
    return transaction.RunWithNoValue(ctx, s.adapter, func(ctx context.Context) error {
        return s.adapter.Delete<Resource>(ctx, input)
    })
}
```

Reference: `openmeter/llmcost/service/service.go`, `openmeter/customer/service/customer.go`

### Service Hooks Pattern

For services that need lifecycle hooks (e.g., other services reacting to creates/deletes), use `models.ServiceHookRegistry`:

```go
type Service struct {
    adapter   <domain>.Adapter
    publisher eventbus.Publisher
    hooks     models.ServiceHookRegistry[<domain>.<Resource>]
}

func (s *Service) RegisterHooks(hooks ...models.ServiceHook[<domain>.<Resource>]) {
    s.hooks.RegisterHooks(hooks...)
}
```

Available hook points: `PostCreate`, `PreDelete`, `PostDelete`, `PreUpdate`, `PostUpdate`. Call them inside the transaction:

```go
// In CreateCustomer:
if err = s.hooks.PostCreate(ctx, created); err != nil {
    return nil, err
}

// In DeleteCustomer:
if err = s.hooks.PreDelete(ctx, existing); err != nil {
    return err
}
// ... perform delete ...
if err = s.hooks.PostDelete(ctx, deleted); err != nil {
    return err
}
```

Reference: `openmeter/customer/service/service.go`, `openmeter/customer/service/customer.go`

## Adapter Layer Implementation (`adapter/`)

The adapter implements business logic and database access using the ent ORM. It:

- Implements the `Adapter` interface
- Contains transaction boilerplate (`Tx`, `WithTx`, `Self`)
- Wraps each method in `entutils.TransactingRepo()` for transaction support
- Maps between ent DB entities and domain types (in `mapping.go`)
- MUST call `input.Validate()` when the service layer is a passthrough (no additional validation)

See `openmeter/customer/adapter/` and `openmeter/llmcost/adapter/` for examples.

### Adapter Transaction Boilerplate

Every adapter MUST implement these three methods. Copy from `openmeter/llmcost/adapter/adapter.go:61-83`:

```go
func (a *adapter) Tx(ctx context.Context) (context.Context, transaction.Driver, error) {
    ctx, rawConfig, eDriver, err := a.db.HijackTx(ctx, &sql.TxOptions{
        ReadOnly: false,
    })
    if err != nil {
        return nil, nil, fmt.Errorf("failed to hijack transaction: %w", err)
    }

    return ctx, entutils.NewTxDriver(eDriver, rawConfig), nil
}

func (a *adapter) WithTx(ctx context.Context, tx *entutils.TxDriver) *adapter {
    txClient := entdb.NewTxClientFromRawConfig(ctx, *tx.GetConfig())

    return &adapter{
        db:     txClient.Client(),
        logger: a.logger,
    }
}

func (a *adapter) Self() *adapter {
    return a
}
```

### Adapter Method Pattern with TransactingRepo

Each adapter method wraps its logic in `entutils.TransactingRepo()` (or `TransactingRepoWithNoValue()` for void):

```go
func (a *adapter) List<Resource>s(ctx context.Context, input <domain>.List<Resource>sInput) (pagination.Result[<domain>.<Resource>], error) {
    return entutils.TransactingRepo(ctx, a, func(ctx context.Context, a *adapter) (pagination.Result[<domain>.<Resource>], error) {
        if err := input.Validate(); err != nil {
            return pagination.Result[<domain>.<Resource>]{}, err
        }

        query := a.db.<Entity>.Query().
            Where(<entity>db.DeletedAtIsNil())  // Always filter soft-deleted

        // Apply ordering
        order := entutils.GetOrdering(sortx.OrderDefault)
        if !input.Order.IsDefaultValue() {
            order = entutils.GetOrdering(input.Order)
        }
        switch input.OrderBy {
        case "id":
            query = query.Order(<entity>db.ByID(order...))
        default:
            query = query.Order(<entity>db.ByID())
        }

        // Paginate
        entities, err := query.Paginate(ctx, input.Page)
        if err != nil {
            return pagination.Result[<domain>.<Resource>]{}, fmt.Errorf("failed to list: %w", err)
        }

        return pagination.MapResultErr(entities, map<Resource>FromEntity)
    })
}
```

For void operations, use `entutils.TransactingRepoWithNoValue()`:

```go
func (a *adapter) Delete<Resource>(ctx context.Context, input <domain>.Delete<Resource>Input) error {
    return entutils.TransactingRepoWithNoValue(ctx, a, func(ctx context.Context, a *adapter) error {
        // ...
    })
}
```

Reference: `openmeter/llmcost/adapter/price.go`

### Entity Mapping (`mapping.go`)

Create `mapping.go` with functions that convert ent entities to domain types:

```go
func map<Resource>FromEntity(entity *db.<Entity>) (<domain>.<Resource>, error) {
    if entity == nil {
        return <domain>.<Resource>{}, errors.New("entity is required")
    }

    return <domain>.<Resource>{
        ManagedModel: models.ManagedModel{
            CreatedAt: entity.CreatedAt,
            UpdatedAt: entity.UpdatedAt,
            DeletedAt: entity.DeletedAt,
        },
        ID:   entity.ID,
        Name: entity.Name,
        // ... map all fields
    }, nil
}
```

For paginated results, use `pagination.MapResultErr(entities, mapFn)`.

Reference: `openmeter/llmcost/adapter/mapping.go`

## Custom Errors (`errors.go`)

Only create custom errors when they bring real value and visibility. All custom errors MUST inherit from generic errors in `pkg/models/errors.go`.

Available generic error types:

- `models.NewGenericNotFoundError(err)` — resource not found
- `models.NewGenericConflictError(err)` — conflict (duplicate key, etc.)
- `models.NewGenericValidationError(err)` — input validation failure
- `models.NewGenericForbiddenError(err)` — authorization failure
- `models.NewGenericPreConditionFailedError(err)` — precondition not met
- `models.NewGenericUnauthorizedError(err)` — authentication failure
- `models.NewGenericNotImplementedError(err)` — not implemented
- `models.NewGenericStatusFailedDependencyError(err)` — dependency failure

See `openmeter/customer/errors.go` for the error pattern:

```go
type MyCustomError struct {
    err error
}

func (e MyCustomError) Error() string { return e.err.Error() }
func (e MyCustomError) Unwrap() error { return e.err }
```

Each custom error should:

- Wrap a generic error from `pkg/models/errors.go`
- Implement `models.GenericError` interface
- Have a constructor function (`NewMyCustomError(...)`)
- Have an `Is` check function (`IsMyCustomError(err error) bool`) if needed

## Domain Events (`event.go`)

Packages that modify database entities should emit domain events. See `openmeter/customer/event.go` for the full pattern.

Events follow this structure:

- Define event name constants using `metadata.EventSubsystem` and `metadata.EventName`
- Implement `EventName() string` and `EventMetadata() metadata.EventMetadata`
- Include a `Validate()` method
- Include a constructor that captures session context: `NewCustomerCreateEvent(ctx, customer)`
- Publish events in the service layer after successful mutations

## Database Schema

When the service requires database tables, use the `/db-migration` skill for creating ent schemas and generating migrations.

## API Handlers

When implementing API handlers for the service, use the `/api` skill for handler implementation patterns, wiring into the server, and type conversion.

## Dependency Injection Wiring

Services are wired together using [Wire](https://github.com/google/wire) for dependency injection.

### Wire provider in `app/common/`

Create a file `app/common/<domain>.go` that defines a Wire provider set and a constructor function. This is where the adapter and service are instantiated and connected.

See `app/common/llmcost.go` for a simple example and `app/common/customer.go` for a more complex one with hooks.

```go
package common

import (
    "fmt"
    "log/slog"

    "github.com/google/wire"

    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
    "<domain>"
    <domain>adapter "<domain>/adapter"
    <domain>service "<domain>/service"
)

var <Domain> = wire.NewSet(
    New<Domain>Service,
)

func New<Domain>Service(logger *slog.Logger, db *entdb.Client) (<domain>.Service, error) {
    adapter, err := <domain>adapter.New(<domain>adapter.Config{
        Client: db,
        Logger: logger.With("subsystem", "<domain>"),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to initialize <domain> adapter: %w", err)
    }

    return <domain>service.New(adapter, logger.With("subsystem", "<domain>")), nil
}
```

Key patterns:

- The constructor takes dependencies as parameters (logger, db client, event publisher, etc.)
- It creates the adapter first, then passes it to the service constructor
- Use `logger.With("subsystem", "<domain>")` for structured logging
- If the service publishes events, also inject `eventbus.Publisher`

### Register in `cmd/<micro_service>/wire.go`

Add the service to the `Application` struct and include the Wire provider set in `wire.Build()`:

1. Add the service field to the `Application` struct:

```go
type Application struct {
    // ...
    <Domain>Service <domain>.Service
}
```

1. Add the provider set to `wire.Build()`:

```go
func initializeApplication(ctx context.Context, conf config.Configuration) (Application, func(), error) {
    wire.Build(
        // ...
        common.<Domain>,
        // ...
    )
}
```

1. Run `make generate` to regenerate `wire_gen.go`

### Multiple entry points

If the service is needed in other entry points (e.g., `cmd/billing-worker`, `cmd/balance-worker`), add it to their `wire.go` files as well. Check which `cmd/*/wire.go` files need the service based on its consumers.

## Workflow

### Creating a new service package

1. Create the package directory: `openmeter/<domain>/`
2. Define domain types in `<domain>.go`
3. Define the `Service` interface in `service.go` with input types and their `Validate()` methods
4. Define the `Adapter` interface in `adapter.go`
5. Implement the service layer in `service/service.go`
6. Create the ent schema if needed (use `/db-migration` skill)
7. Implement the adapter layer in `adapter/adapter.go`, `adapter/<operation>.go`, `adapter/mapping.go`
8. Add `errors.go` only if custom errors are needed
9. Add `event.go` if the service modifies entities
10. Wire it up: create `app/common/<domain>.go` and register in `cmd/<micro_service>/wire.go`
11. Run `make generate` to regenerate Wire bindings
12. Implement API handlers (use `/api` skill)

### Modifying an existing service

1. Read existing interfaces and understand the current patterns
2. Add new methods to both `Service` and `Adapter` interfaces
3. Add input types with `Validate()` methods
4. Implement in both `service/` and `adapter/` layers
