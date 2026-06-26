# hooks

<!-- archie:ai-start -->

> Provides the customer-side service hook that auto-provisions Subjects for every key in a customer's UsageAttribution.SubjectKeys. It is the bridge that keeps the subject domain in sync when customers are created or updated, registered onto the customer service via RegisterHooks.

## Patterns

**ServiceHook implementation via Noop embedding** — Hook types alias models.ServiceHook[customer.Customer] / models.NoopServiceHook[customer.Customer]. The concrete customerSubjectHook embeds NoopCustomerSubjectHook so it only overrides the lifecycle methods it cares about (PostCreate, PostUpdate), leaving the rest as no-ops. (`type customerSubjectHook struct { NoopCustomerSubjectHook; provisioner *SubjectProvisioner; ... }`)
**Config Validate() before construction** — SubjectProvisionerConfig implements models.Validator. Both NewCustomerSubjectHook and NewSubjectProvisioner call config.Validate() first and wrap failures; Validate collects errs into errors.Join (Subject, Logger, Tracer all required). (`if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid subject hook config: %w", err) }`)
**Re-entrancy guard via context flag** — EnsureSubjects short-circuits when SkipSubjectCustomerFromContext(ctx) is true. When creating a subject it passes NewContextWithSkipSubjectCustomer(ctx) into subject.Create so the reciprocal customer hook (customer/service/hooks SubjectCustomerHook) does not loop back. (`sub, err = p.subject.Create(NewContextWithSkipSubjectCustomer(ctx), subject.CreateInput{...})`)
**OTel span per hook/provisioner method** — Every exported method (PostCreate, PostUpdate, EnsureSubjects, EnsureSubject) opens a tracer span, sets otelcodes.Ok/Error status, records errors, and ends the span via defer. Span names follow dotted convention like 'subject_provisioner.ensure_subjects'. (`ctx, span := p.tracer.Start(ctx, "subject_provisioner.ensure_subjects"); defer func(){ ...; span.End() }()`)
**Get-or-create with typed not-found check** — EnsureSubject calls subject.GetByIdOrKey; treats only models.IsGenericNotFoundError(err) (or sub.IsDeleted()) as the create branch and propagates any other error wrapped with namespace/customer.id context. (`if err != nil && !models.IsGenericNotFoundError(err) { return nil, fmt.Errorf(...namespace=%s customer.id=%s...: %w, ...) }`)
**Accumulate validation errors across subject keys** — EnsureSubjects iterates all UsageAttribution.SubjectKeys collecting per-key errors into errs and returns errors.Join(errs...); a returned subject whose Key != requested subKey yields a models.NewGenericValidationError instructing callers to use key not id. (`errs = append(errs, models.NewGenericValidationError(fmt.Errorf("use subject key instead of id...")))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customersubject.go` | Defines the customer->subject provisioning hook (customerSubjectHook), the reusable SubjectProvisioner, their configs/validators, and the skip-context helpers. | CustomerSubjectHookConfig is a type alias of SubjectProvisionerConfig; do not duplicate fields. PostCreate/PostUpdate both just call provisioner.EnsureSubjects — keep them symmetric. Always thread NewContextWithSkipSubjectCustomer through subject.Create to avoid hook recursion with the customer-side SubjectCustomerHook. |
| `customersubject_test.go` | Integration test wiring a real Postgres-backed TestEnv with concrete subject + customer adapters/services; registers both this hook and the reciprocal customerservicehooks.NewSubjectCustomerHook to exercise the create/conflict paths. | TestEnv is built from underlying constructors (subjectadapter.New, subjectservice.New, customeradapter.New, customerservice.New), not app/common wiring — preserve that to avoid import cycles. NoopCustomerOverrideService is a local stub satisfying billing.CustomerOverrideService. |

## Anti-Patterns

- Calling subject.Create without NewContextWithSkipSubjectCustomer — triggers infinite recursion with the customer-side SubjectCustomerHook.
- Returning on the first failing subject key instead of accumulating into errs and errors.Join — loses errors for other keys.
- Treating any GetByIdOrKey error as not-found; only models.IsGenericNotFoundError (or IsDeleted) should branch into create.
- Constructing the hook with slog.Default() or a nil Tracer/Subject — Validate() rejects nil deps and construction must fail loudly.
- Building test dependencies from app/common wiring instead of the underlying constructors, creating test-only import cycles.

## Decisions

- **SubjectProvisioner is split out from the hook itself.** — The provisioning logic (EnsureSubjects/EnsureSubject) is reusable independent of the ServiceHook lifecycle, so it has its own constructor and config.
- **A context-key skip flag coordinates the two reciprocal hooks (subject<->customer).** — Customer-create provisions a subject and subject-create provisions a customer; the skip flag breaks the mutual-trigger cycle without disabling either hook globally.

## Example: Constructing and registering the customer subject hook

```
import subjectservicehooks "github.com/openmeterio/openmeter/openmeter/subject/service/hooks"

hook, err := subjectservicehooks.NewCustomerSubjectHook(subjectservicehooks.CustomerSubjectHookConfig{
	Subject: subjectService,
	Logger:  logger,
	Tracer:  tracer,
})
if err != nil {
	return err
}
customerService.RegisterHooks(hook) // PostCreate/PostUpdate provision subjects for UsageAttribution.SubjectKeys
```

<!-- archie:ai-end -->
