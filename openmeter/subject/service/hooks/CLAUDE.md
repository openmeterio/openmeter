# hooks

<!-- archie:ai-start -->

> Implements the CustomerSubjectHook that bridges customer.Service lifecycle events to subject.Service provisioning, ensuring every UsageAttribution.SubjectKey has a corresponding Subject row without creating circular imports between the subject and customer domains.

## Patterns

**Embed NoopServiceHook for partial overrides** — Hook structs embed NoopCustomerSubjectHook = models.NoopServiceHook[customer.Customer] and override only the needed methods (PostCreate, PostUpdate); other lifecycle methods default to no-ops. (`type customerSubjectHook struct { NoopCustomerSubjectHook; provisioner *SubjectProvisioner; tracer trace.Tracer }`)
**Context-key skip flag prevents re-entrant loops** — NewContextWithSkipSubjectCustomer sets a typed private context key; any call back into subject.Service inside a hook must carry it to prevent infinite recursion when a reverse hook is also registered. (`sub, err = p.subject.Create(NewContextWithSkipSubjectCustomer(ctx), subject.CreateInput{...})`)
**Config struct with Validate() for constructor injection** — Dependencies (Subject service, Logger, Tracer) are bundled in SubjectProvisionerConfig (implements models.Validator); constructors call config.Validate() before allocation. (`func NewCustomerSubjectHook(config CustomerSubjectHookConfig) (CustomerSubjectHook, error) { if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid subject hook config: %w", err) } ... }`)
**OTel span per operation with deferred err capture** — Every exported method opens a tracer span at entry and defers span.End() with status-setting in a closure capturing the named error; RecordError only on non-nil err. (`ctx, span := p.tracer.Start(ctx, "subject_provisioner.ensure_subjects"); defer func() { if err != nil { span.SetStatus(otelcodes.Error, err.Error()); span.RecordError(err) }; span.End() }()`)
**models.IsGenericNotFoundError for idempotent upsert** — EnsureSubject calls GetByIdOrKey and treats IsGenericNotFoundError (or a deleted subject) as the signal to create; all other errors propagate. Never match by string comparison. (`if models.IsGenericNotFoundError(err) || sub.IsDeleted() { sub, err = p.subject.Create(NewContextWithSkipSubjectCustomer(ctx), ...) }`)
**Delegate logic to SubjectProvisioner, not hook methods** — Hook lifecycle methods are thin delegators calling provisioner.EnsureSubjects and handling span status; all provisioning logic lives in SubjectProvisioner so it can be reused outside the hook interface. (`func (s customerSubjectHook) PostCreate(ctx context.Context, cus *customer.Customer) error { ...; return s.provisioner.EnsureSubjects(ctx, cus) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customersubject.go` | Sole production file: customerSubjectHook (models.ServiceHook[customer.Customer]), SubjectProvisioner, and config structs. Type alias CustomerSubjectHookConfig = SubjectProvisionerConfig keeps the public API minimal. | Any new lifecycle method must pass NewContextWithSkipSubjectCustomer(ctx) when calling back into subject.Service. Never skip config.Validate() in constructors. Preserve the EnsureSubjects nil-guard on cus. |
| `customersubject_test.go` | Integration tests wiring concrete adapters (subjectadapter, customeradapter, customerservice, subjectservice) without app/common. Covers WithExistingSubject, WithoutExistingSubject, and mutual Conflict scenarios. | Tests use t.Context(). testutils.InitPostgresDB + DBSchemaMigrate must run before any DB interaction. NoopCustomerOverrideService stubs billing deps without importing billing wiring. |

## Anti-Patterns

- Calling subject.Service inside a hook without NewContextWithSkipSubjectCustomer when a reverse hook is registered — causes infinite recursion.
- Importing app/common in tests — breaks isolation; build test deps from concrete constructors directly.
- Using context.Background() instead of propagating the caller ctx through hook methods.
- Catching subject.Service errors by string comparison instead of models.IsGenericNotFoundError.
- Adding business logic in hook lifecycle methods instead of delegating to SubjectProvisioner.

## Decisions

- **SubjectProvisioner is separate from customerSubjectHook.** — The provisioner can be reused for non-hook paths (e.g. batch backfills) without the models.ServiceHook interface overhead.
- **Skip flag lives in context rather than as a boolean field on the hook.** — A context key is scoped to one call stack, preventing skip state from leaking across concurrent requests sharing the same hook instance.
- **Type alias CustomerSubjectHookConfig = SubjectProvisionerConfig.** — Avoids duplicating config fields while letting callers use the hook-specific name; a single Validate() covers both entry points.

## Example: Adding a PostDelete lifecycle that prevents reverse-hook loops

```
import (
	"context"
	"github.com/openmeterio/openmeter/openmeter/customer"
	subjectservicehooks "github.com/openmeterio/openmeter/openmeter/subject/service/hooks"
)

func (s customerSubjectHook) PostDelete(ctx context.Context, cus *customer.Customer) error {
	ctx, span := s.tracer.Start(ctx, "customer_subject_hook.post_delete")
	defer span.End()
	ctx = subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx)
	return s.provisioner.RemoveSubjects(ctx, cus)
}
```

<!-- archie:ai-end -->
