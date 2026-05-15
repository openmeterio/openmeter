# hooks

<!-- archie:ai-start -->

> Implements the CustomerSubjectHook that bridges customer.Service lifecycle events to subject.Service provisioning, ensuring every UsageAttribution.SubjectKey has a corresponding Subject row without creating circular imports between subject and customer domains.

## Patterns

**Embed NoopServiceHook for partial overrides** — Hook structs embed the noop type alias (NoopCustomerSubjectHook = models.NoopServiceHook[customer.Customer]) and override only the needed lifecycle methods (PostCreate, PostUpdate). All other lifecycle methods default to no-ops automatically. (`type customerSubjectHook struct { NoopCustomerSubjectHook; provisioner *SubjectProvisioner; tracer trace.Tracer }`)
**Context-key skip flag prevents re-entrant loops** — NewContextWithSkipSubjectCustomer sets a typed private context key; EnsureSubjects checks SkipSubjectCustomerFromContext before acting. Any call back into subject.Service inside a hook method must carry this context to prevent infinite recursion when a reverse hook is also registered. (`sub, err = p.subject.Create(NewContextWithSkipSubjectCustomer(ctx), subject.CreateInput{...})`)
**Config struct with Validate() for constructor injection** — All dependencies (Subject service, Logger, Tracer) are bundled in SubjectProvisionerConfig which implements models.Validator. NewCustomerSubjectHook and NewSubjectProvisioner both call config.Validate() before allocation; invalid configs return a wrapped error. (`func NewCustomerSubjectHook(config CustomerSubjectHookConfig) (CustomerSubjectHook, error) { if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid subject hook config: %w", err) } ... }`)
**OTel span per operation with deferred closure capturing err** — Every exported method opens a tracer span at entry and defers span.End() together with status-setting in a closure that captures the returned named error variable. RecordError is called only on non-nil err. (`ctx, span := p.tracer.Start(ctx, "subject_provisioner.ensure_subjects"); defer func() { if err != nil { span.SetStatus(otelcodes.Error, err.Error()); span.RecordError(err) }; span.End() }()`)
**models.IsGenericNotFoundError for idempotent upsert** — EnsureSubject calls subject.GetByIdOrKey and treats IsGenericNotFoundError (or a deleted subject) as the signal to create; all other errors are propagated. Never match errors by string comparison. (`if models.IsGenericNotFoundError(err) || sub.IsDeleted() { sub, err = p.subject.Create(NewContextWithSkipSubjectCustomer(ctx), ...) }`)
**Delegate business logic to SubjectProvisioner, not hook methods** — Hook lifecycle methods (PostCreate, PostUpdate) are thin delegators that call provisioner.EnsureSubjects and handle span status. All provisioning logic lives in SubjectProvisioner so it can be reused outside the hook interface (e.g., batch backfills). (`func (s customerSubjectHook) PostCreate(ctx context.Context, cus *customer.Customer) error { ctx, span := s.tracer.Start(...); defer span.End(); return s.provisioner.EnsureSubjects(ctx, cus) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customersubject.go` | Sole production file: defines customerSubjectHook (models.ServiceHook[customer.Customer] impl), SubjectProvisioner (provisioning worker), and their config structs. The type alias CustomerSubjectHookConfig = SubjectProvisionerConfig keeps the public API surface minimal. | Any new lifecycle method (e.g. PostDelete) must pass NewContextWithSkipSubjectCustomer(ctx) when calling back into subject.Service. Never skip config.Validate() in constructors. The EnsureSubjects nil-guard on cus must be preserved. |
| `customersubject_test.go` | Integration tests wiring concrete adapters (subjectadapter, customeradapter, customerservice, subjectservice) without importing app/common. Covers WithExistingSubject, WithoutExistingSubject, and mutual Conflict (both hooks registered simultaneously) scenarios. | Tests use t.Context() not context.Background(). testutils.InitPostgresDB + DBSchemaMigrate must be called before any DB interaction. NoopCustomerOverrideService shows how to stub billing deps without importing billing wiring. |

## Anti-Patterns

- Calling subject.Service methods inside a hook without NewContextWithSkipSubjectCustomer when a reverse hook is also registered — causes infinite recursion
- Importing app/common in tests — breaks isolation; build test deps from concrete constructors (adapter + service) directly
- Using context.Background() instead of propagating the caller ctx through hook methods
- Catching errors from subject.Service by string comparison instead of models.IsGenericNotFoundError
- Adding business logic directly in hook lifecycle methods instead of delegating to SubjectProvisioner

## Decisions

- **Separate SubjectProvisioner from customerSubjectHook** — SubjectProvisioner can be reused independently for non-hook provisioning paths (e.g. batch backfills) without carrying the models.ServiceHook interface overhead.
- **Skip flag in context rather than a boolean field on the hook** — A context key is naturally scoped to one call stack, preventing skip state from leaking across concurrent requests that share the same hook instance.
- **Type alias CustomerSubjectHookConfig = SubjectProvisionerConfig** — Avoids duplicating config fields while letting callers reference the hook-specific name; a single Validate() implementation covers both entry points.

## Example: Adding a PostDelete lifecycle that removes orphaned subjects while preventing reverse-hook loops

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
