---
name: test
description: Write tests for OpenMeter services following project conventions. Use when creating unit tests, integration tests, or service tests.
user-invocable: true
argument-hint: "[description of what to test]"
allowed-tools: Read, Edit, Write, Bash, Grep, Glob, Agent
---

# Testing

You are helping the user write tests for OpenMeter following established conventions.

## Test Types & File Locations

| Type | Purpose | Location | DB Required |
|------|---------|----------|-------------|
| Unit tests | Validation, pure functions | `openmeter/<domain>/<domain>_test.go` | No |
| Integration tests | Adapter against real Postgres | `openmeter/<domain>/adapter/*_test.go` | Yes |
| Service tests | Full stack via TestEnv | `openmeter/<domain>/service/*_test.go` | Yes |

## Running Tests

```bash
make test                 # All tests (sets POSTGRES_HOST=127.0.0.1)
make test-nocache         # Tests bypassing cache
make etoe                 # End-to-end tests (requires docker compose)
```

Before running: `docker compose up -d postgres`

Build tag: all Go test commands use `-tags=dynamic`.

For running a specific test directly:

```bash
POSTGRES_HOST=127.0.0.1 go test -tags=dynamic ./openmeter/<domain>/...
```

## Key Test Utilities

From `openmeter/testutils/`:

| Utility | Usage |
|---------|-------|
| `testutils.InitPostgresDB(t)` | Provisions fresh Postgres DB per test; skips if `POSTGRES_HOST` not set |
| `testutils.NewDiscardLogger(t)` | Silent logger for tests |
| `testutils.NewLogger(t)` | Default slog logger for tests |

From `openmeter/watermill/eventbus/`:

| Utility | Usage |
|---------|-------|
| `eventbus.NewMock(t)` | Mock event publisher for tests |

## TestEnv Pattern

For service/integration tests, create a `testutils/` package with a `TestEnv` that wires up the full stack.

Reference: `openmeter/customer/testutils/env.go`

```go
package testutils

import (
    "sync"
    "testing"

    "github.com/stretchr/testify/require"

    "<domain>"
    <domain>adapter "<domain>/adapter"
    <domain>service "<domain>/service"
    entdb "github.com/openmeterio/openmeter/openmeter/ent/db"
    "github.com/openmeterio/openmeter/openmeter/testutils"
    "github.com/openmeterio/openmeter/openmeter/watermill/eventbus"
)

type TestEnv struct {
    Logger  *slog.Logger
    Service <domain>.Service
    Client  *entdb.Client
    db      *testutils.TestDB
    close   sync.Once
}

func (e *TestEnv) DBSchemaMigrate(t *testing.T) {
    t.Helper()
    require.NotNilf(t, e.db, "database must be initialized")
    err := e.db.EntDriver.Client().Schema.Create(t.Context())
    require.NoErrorf(t, err, "schema migration must not fail")
}

func (e *TestEnv) Close(t *testing.T) {
    t.Helper()
    e.close.Do(func() {
        if e.db != nil {
            if err := e.db.EntDriver.Close(); err != nil {
                t.Errorf("failed to close ent driver: %v", err)
            }
            if err := e.db.PGDriver.Close(); err != nil {
                t.Errorf("failed to close postgres driver: %v", err)
            }
        }
        if e.Client != nil {
            if err := e.Client.Close(); err != nil {
                t.Errorf("failed to close ent client: %v", err)
            }
        }
    })
}

func NewTestEnv(t *testing.T) *TestEnv {
    t.Helper()

    logger := testutils.NewDiscardLogger(t)

    // Init database
    db := testutils.InitPostgresDB(t)
    client := db.EntDriver.Client()

    // Init event publisher
    publisher := eventbus.NewMock(t)

    // Init adapter
    adapter, err := <domain>adapter.New(<domain>adapter.Config{
        Client: client,
        Logger: logger,
    })
    require.NoErrorf(t, err, "initializing adapter must not fail")

    // Init service
    service, err := <domain>service.New(<domain>service.Config{
        Adapter:   adapter,
        Publisher: publisher,
    })
    require.NoErrorf(t, err, "initializing service must not fail")

    return &TestEnv{
        Logger:  logger,
        Service: service,
        Client:  client,
        db:      db,
        close:   sync.Once{},
    }
}
```

### Using TestEnv in Tests

```go
func TestCreate<Resource>(t *testing.T) {
    env := testutils.NewTestEnv(t)
    t.Cleanup(func() { env.Close(t) })
    env.DBSchemaMigrate(t)

    ns := testutils.NewTestNamespace(t) // generates a random ULID namespace

    t.Run("success", func(t *testing.T) {
        result, err := env.Service.Create<Resource>(t.Context(), <domain>.Create<Resource>Input{
            Namespace: ns,
            Name:      "test",
        })
        require.NoError(t, err)
        assert.Equal(t, "test", result.Name)
    })

    t.Run("validation error", func(t *testing.T) {
        _, err := env.Service.Create<Resource>(t.Context(), <domain>.Create<Resource>Input{})
        require.Error(t, err)
        assert.True(t, models.IsGenericValidationError(err))
    })
}
```

## Unit Test Pattern

For testing validation, pure functions, and domain logic without DB:

```go
func TestCreate<Resource>Input_Validate(t *testing.T) {
    tests := []struct {
        name    string
        input   <domain>.Create<Resource>Input
        wantErr bool
    }{
        {
            name: "valid",
            input: <domain>.Create<Resource>Input{
                Namespace: "test-ns",
                Name:      "test",
            },
            wantErr: false,
        },
        {
            name:    "missing namespace",
            input:   <domain>.Create<Resource>Input{Name: "test"},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.input.Validate()
            if tt.wantErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

## Testing Conventions

- Use `require` for fatal assertions (test cannot continue), `assert` for soft assertions
- Use `t.Helper()` in all helper functions
- Use `t.Context()` instead of `context.Background()`
- Use table-driven tests with `t.Run()` for multiple cases
- Use `testutils.NewTestULID(t)` or `testutils.NewTestNamespace(t)` for random test identifiers
- Each test gets its own namespace to avoid cross-test interference
- Integration tests are automatically skipped when `POSTGRES_HOST` is not set (via `testutils.InitPostgresDB`)
