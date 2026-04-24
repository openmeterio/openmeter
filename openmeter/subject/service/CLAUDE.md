# service

<!-- archie:ai-start -->

> Concrete implementation of subject.Service that orchestrates CRUD lifecycle for subjects (usage measurement subjects) via subject.Adapter, wraps mutating operations in transactions, and fans out lifecycle events through a models.ServiceHookRegistry. All writes run inside transaction.Run / transaction.RunWithNoValue so hooks execute within the same DB transaction as the mutation.

## Patterns

**Interface compliance guard** — var _ subject.Service = (*Service)(nil) at package scope ensures the struct satisfies the interface at compile time. (`var _ subject.Service = (*Service)(nil)`)
**Input validation before any adapter call** — Every mutating method calls input.Validate() and wraps errors with models.NewGenericValidationError before touching the adapter. (`if err := input.Validate(); err != nil { return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err)) }`)
**Wrap mutating adapter calls in transaction.Run / transaction.RunWithNoValue** — Create, Update, and Delete use transaction.Run or transaction.RunWithNoValue so hook execution and adapter writes share the same DB transaction. (`return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) { ... })`)
**ServiceHookRegistry fan-out in lifecycle order** — Hooks fire in the correct order: PostCreate after insert, PreUpdate/PostUpdate around update, PreDelete/PostDelete around delete. All hook calls propagate ctx. (`s.hooks.PostCreate(ctx, &sub); s.hooks.PreUpdate(ctx, &sub); s.hooks.PostUpdate(ctx, &sub)`)
**Read-only methods bypass transaction** — GetById, GetByKey, GetByIdOrKey, and List delegate directly to s.subjectAdapter without wrapping in transaction.Run. (`func (s *Service) GetById(ctx context.Context, id models.NamespacedID) (subject.Subject, error) { return s.subjectAdapter.GetById(ctx, id) }`)
**Constructor nil-guard on Adapter** — New() returns an error if subjectAdapter is nil, preventing wiring mistakes from producing a silently broken service. (`if subjectAdapter == nil { return nil, fmt.Errorf("subject adapter is required") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Single-file service implementation. Holds all subject.Service method bodies plus the ServiceHookRegistry field and RegisterHooks forwarder. | Adding adapter calls outside a transaction.Run/RunWithNoValue block for write paths; forgetting to call hooks in the correct pre/post order; introducing context.Background() instead of propagating the incoming ctx. |
| `service_test.go` | Integration test for the full CRUD surface driven via subjecttestutils.NewTestEnv; tests hooks indirectly through customer/entitlement interactions. | Tests use t.Context() throughout — never context.Background(); env is built from concrete constructors (adapter + service), not from app/common wiring. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run / transaction.RunWithNoValue in Create, Update, or Delete.
- Firing hooks outside the transaction boundary (after transaction.Run returns), breaking atomic rollback on hook error.
- Importing app/common in tests — use subjecttestutils.NewTestEnv which wires from concrete constructors.
- Using context.Background() or context.TODO() instead of propagating the caller's ctx.
- Adding business logic directly in hook callback methods — delegate to a provisioner struct to keep single-responsibility.

## Decisions

- **Transactions wrap both adapter writes and hook execution** — Hooks may write to other tables (e.g. provisioning a customer-subject row); rolling back the transaction on hook failure reverts all writes atomically.
- **ServiceHookRegistry is a value type embedded in Service, not an interface parameter** — Hooks are registered at startup via RegisterHooks; embedding the registry avoids an extra constructor parameter and lets Wire inject the service before hooks are registered.
- **Read methods bypass transactions** — Reads have no side-effects and no hooks; wrapping them in a transaction would introduce unnecessary savepoint overhead.

## Example: Implement a new mutating method on Service following the established pattern

```
import (
    "context"
    "fmt"

    "github.com/openmeterio/openmeter/openmeter/subject"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
    "github.com/openmeterio/openmeter/pkg/models"
)

func (s *Service) Rename(ctx context.Context, input subject.RenameInput) (subject.Subject, error) {
    if err := input.Validate(); err != nil {
        return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
    }

    return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) {
// ...
```

<!-- archie:ai-end -->
