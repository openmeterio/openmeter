# hooks

<!-- archie:ai-start -->

> Implements cross-domain lifecycle hooks that provision Subject entities in response to Customer create/update events. Acts as the bridge between customer.Service and subject.Service, ensuring that every UsageAttribution.SubjectKey has a corresponding Subject row without creating circular imports.

## Patterns

**Embed NoopServiceHook for partial overrides** — Hook structs embed the noop type alias (NoopCustomerSubjectHook = models.NoopServiceHook[customer.Customer]) and override only the needed lifecycle methods (PostCreate, PostUpdate). All other lifecycle methods are no-ops automatically. (`type customerSubjectHook struct { NoopCustomerSubjectHook; provisioner *SubjectProvisioner; ... }`)
**Context-key skip flag to prevent re-entrant loops** — NewContextWithSkipSubjectCustomer sets a typed context key; EnsureSubjects checks SkipSubjectCustomerFromContext before acting. Subject.Create called inside the hook must carry this context to avoid triggering the reverse customer hook and causing infinite recursion. (`sub, err = p.subject.Create(NewContextWithSkipSubjectCustomer(ctx), subject.CreateInput{...})`)
**Config struct with Validate() for constructor injection** — All dependencies (Subject service, Logger, Tracer) are bundled in SubjectProvisionerConfig which implements models.Validator. Constructor NewCustomerSubjectHook calls config.Validate() before allocation; invalid configs return a wrapped error. (`func NewCustomerSubjectHook(config CustomerSubjectHookConfig) (CustomerSubjectHook, error) { if err := config.Validate(); err != nil { return nil, fmt.Errorf("...") } ... }`)
**OTel span per operation with deferred status** — Every exported method opens a tracer span at entry and defers span.End() together with status setting in a closure that captures the returned error variable. RecordError is called only on non-nil err. (`ctx, span := p.tracer.Start(ctx, "subject_provisioner.ensure_subjects"); defer func() { if err != nil { span.SetStatus(...) }; span.End() }()`)
**Use models.IsGenericNotFoundError for idempotent upsert** — EnsureSubject calls subject.GetByIdOrKey and treats GenericNotFoundError (or a deleted subject) as the signal to create, propagating all other errors. Never catch errors by string matching. (`if models.IsGenericNotFoundError(err) || sub.IsDeleted() { sub, err = p.subject.Create(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customersubject.go` | Sole production file: defines CustomerSubjectHook (models.ServiceHook[customer.Customer] impl), SubjectProvisioner (worker), and their config structs. The type alias CustomerSubjectHookConfig = SubjectProvisionerConfig keeps the public API surface minimal. | Any new lifecycle method (e.g. PostDelete) must use NewContextWithSkipSubjectCustomer when calling back into subject.Service to avoid re-entrant cycles. Never skip the Validate() call in constructors. |
| `customersubject_test.go` | Integration test wiring: builds a real TestEnv from concrete adapters (subjectadapter, customeradapter, customerservice, subjectservice) without importing app/common. Covers WithExistingSubject, WithoutExistingSubject, and mutual Conflict (both hooks registered simultaneously) scenarios. | Tests use t.Context() not context.Background(). testutils.InitPostgresDB + DBSchemaMigrate must be called before any DB interaction. NoopCustomerOverrideService shows how to stub billing deps without importing billing wiring. |

## Anti-Patterns

- Calling subject.Service methods inside hook without NewContextWithSkipSubjectCustomer when a reverse hook from customer/service/hooks is also registered — causes infinite recursion
- Importing app/common in tests — breaks isolation; build deps from concrete constructors (adapter + service) directly
- Using context.Background() instead of propagating the caller ctx through hook methods
- Catching errors from subject.Service by string comparison instead of models.IsGenericNotFoundError
- Adding business logic outside SubjectProvisioner (e.g. directly in customerSubjectHook lifecycle methods) — delegation to provisioner keeps each type single-responsibility

## Decisions

- **Separate SubjectProvisioner from customerSubjectHook** — SubjectProvisioner can be reused independently for non-hook provisioning paths (e.g. batch backfills) without carrying the models.ServiceHook interface overhead.
- **Skip flag in context rather than a boolean field on the hook** — A context key is naturally scoped to one call stack, preventing skip state from leaking across concurrent requests that share the same hook instance.
- **Type alias CustomerSubjectHookConfig = SubjectProvisionerConfig** — Avoids duplicating config fields while letting callers reference the hook-specific name; a single Validate() implementation covers both entry points.

## Example: Adding a new PostDelete lifecycle that removes orphaned subjects while preventing reverse-hook loops

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/customer"
	subjectservicehooks "github.com/openmeterio/openmeter/openmeter/subject/service/hooks"
)

func (s customerSubjectHook) PostDelete(ctx context.Context, cus *customer.Customer) error {
	ctx, span := s.tracer.Start(ctx, "customer_subject_hook.post_delete")
	defer span.End()
	// Pass skip-flag context so subject.Delete does not re-trigger customer hooks
	ctx = subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx)
	return s.provisioner.RemoveSubjects(ctx, cus)
}
```

<!-- archie:ai-end -->
