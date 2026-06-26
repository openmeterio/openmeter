# testutils

<!-- archie:ai-start -->

> Test harness for the subject domain. Provides a TestEnv that wires a real Postgres-backed subject service together with customer, feature, and entitlement dependencies so subject/service tests run against production constructors.

## Patterns

**TestEnv with NewTestEnv constructor** — NewTestEnv(t) builds a TestEnv from concrete package constructors (subjectadapter.New, subjectservice.New, customeradapter/customerservice, productcatalogadapter, entitlementpgadapter) over a Postgres test DB. (`subjectAdapter, _ := subjectadapter.New(client); subjectService, _ := subjectservice.New(subjectAdapter)`)
**Postgres + mock eventbus init** — DB comes from testutils.InitPostgresDB(t); events use eventbus.NewMock(t); logger is testutils.NewDiscardLogger(t); tracer is noop. (`db := testutils.InitPostgresDB(t); publisher := eventbus.NewMock(t)`)
**ULID namespace helpers** — NewTestULID(t) returns a fresh ULID; NewTestNamespace is aliased to it for per-test namespace isolation. (`var NewTestNamespace = NewTestULID`)
**Explicit schema migrate + idempotent Close** — DBSchemaMigrate(t) runs Schema.Create; Close(t) tears down ent/pg drivers and the client exactly once via sync.Once. (`e.close.Do(func() { e.db.EntDriver.Close(); e.db.PGDriver.Close(); e.Client.Close() })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `env.go` | Defines TestEnv (SubjectService, CustomerService, EntitlementAdapter, FeatureService, Client) and NewTestEnv/DBSchemaMigrate/Close plus ULID helpers. | Meter adapter is a mockadapter (meteradapter.New(nil)); feature/entitlement use real Postgres adapters; build deps from package constructors, never from app/common, to avoid test import cycles. |

## Anti-Patterns

- Importing app/common wiring here — must build deps from underlying constructors per repo testutils rule.
- Skipping DBSchemaMigrate before exercising the service (the table will not exist).
- Calling Close more than the sync.Once guards or leaking ent/pg drivers.

## Decisions

- **Wire real Postgres-backed services rather than mocks for subject/customer/feature/entitlement** — Subject behavior (soft-delete windows, constraints) only surfaces against a real DB; mocks would hide adapter-level logic.

## Example: Constructing the subject test environment

```
func NewTestEnv(t *testing.T) *TestEnv {
  logger := testutils.NewDiscardLogger(t)
  db := testutils.InitPostgresDB(t)
  client := db.EntDriver.Client()
  subjectAdapter, err := subjectadapter.New(client)
  require.NoError(t, err)
  subjectService, err := subjectservice.New(subjectAdapter)
  require.NoError(t, err)
  return &TestEnv{SubjectService: subjectService, Client: client, db: db, close: sync.Once{}}
}
```

<!-- archie:ai-end -->
