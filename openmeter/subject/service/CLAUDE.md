# service

<!-- archie:ai-start -->

> Concrete implementation of subject.Service that orchestrates CRUD lifecycle for Subject entities via subject.Adapter, wrapping all mutating operations in transactions and fanning out lifecycle events through a models.ServiceHookRegistry. Reads bypass transactions; writes always use transaction.Run or transaction.RunWithNoValue so hooks and adapter writes are atomic.

## Patterns

**Interface compliance guard** — var _ subject.Service = (*Service)(nil) at package scope ensures compile-time satisfaction of the interface. (`var _ subject.Service = (*Service)(nil)`)
**Input validation before adapter** — Every mutating method calls input.Validate() and wraps errors with models.NewGenericValidationError before any adapter call. (`if err := input.Validate(); err != nil { return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err)) }`)
**transaction.Run wraps writes and hooks** — Create, Update, and Delete use transaction.Run / transaction.RunWithNoValue so adapter writes and hook fan-outs share the same DB transaction and roll back together on failure. (`return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) { sub, err := s.subjectAdapter.Create(ctx, input); s.hooks.PostCreate(ctx, &sub); return sub, nil })`)
**ServiceHookRegistry fan-out in lifecycle order** — Hooks fire in correct order: PostCreate after insert, PreUpdate before update + PostUpdate after, PreDelete before delete + PostDelete after. All hook calls propagate the transaction-carrying ctx. (`s.hooks.PreUpdate(ctx, &sub); sub, err = s.subjectAdapter.Update(ctx, input); s.hooks.PostUpdate(ctx, &sub)`)
**Read-only methods bypass transaction** — GetById, GetByKey, GetByIdOrKey, and List delegate directly to s.subjectAdapter without transaction.Run to avoid unnecessary savepoint overhead. (`func (s *Service) GetById(ctx context.Context, id models.NamespacedID) (subject.Subject, error) { return s.subjectAdapter.GetById(ctx, id) }`)
**Constructor nil-guard on Adapter** — New() returns an error if subjectAdapter is nil, preventing silent runtime panics from wiring mistakes. (`if subjectAdapter == nil { return nil, fmt.Errorf("subject adapter is required") }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Single-file service implementation holding all subject.Service method bodies, the ServiceHookRegistry field, and the RegisterHooks forwarder. | Adding adapter calls outside transaction.Run/RunWithNoValue in Create, Update, or Delete; omitting pre/post hook calls or firing them in the wrong order; using context.Background() instead of propagating the incoming ctx. |
| `service_test.go` | Integration test for the full CRUD surface driven via subjecttestutils.NewTestEnv; exercises hooks indirectly through customer and entitlement interactions. | Must use t.Context() throughout — never context.Background(); env is built from concrete constructors (adapter + service), not app/common, to avoid import cycles. |

## Anti-Patterns

- Calling adapter methods outside transaction.Run / transaction.RunWithNoValue in Create, Update, or Delete — breaks atomic rollback on hook failure.
- Firing hooks outside the transaction boundary (after transaction.Run returns) — hook side-effects become non-atomic with the DB write.
- Importing app/common in tests — use subjecttestutils.NewTestEnv which wires from concrete constructors to preserve test isolation.
- Using context.Background() or context.TODO() instead of propagating the caller's ctx — drops Ent transaction driver and OTel spans.
- Adding business logic directly in hook callback methods — delegate to a provisioner struct (see hooks/ sub-package) to maintain single responsibility.

## Decisions

- **Transactions wrap both adapter writes and hook execution** — Hooks may write to other tables (e.g. provisioning customer-subject rows); rolling back the transaction on hook failure reverts all writes atomically.
- **ServiceHookRegistry is embedded as a value in Service, not an interface parameter** — Hooks are registered at startup via RegisterHooks after Wire injects the service; embedding the registry avoids an extra constructor parameter and eliminates a nil-interface panic.
- **Read methods bypass transactions** — Reads have no side-effects and trigger no hooks; wrapping them in transaction.Run would add unnecessary savepoint overhead on every List/Get call.

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
        sub, err := s.subjectAdapter.GetById(ctx, models.NamespacedID{Namespace: input.Namespace, ID: input.ID})
// ...
```

<!-- archie:ai-end -->
