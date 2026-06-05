# hooks

<!-- archie:ai-start -->

> Lifecycle hooks that wire the customer domain into the entitlement, subject, and billing domains. Hosts the SubjectCustomerHook (provisions/syncs a Customer whenever a Subject is created/updated/deleted via CustomerProvisioner) and the EntitlementValidatorHook (blocks customer deletion while entitlements exist).

## Patterns

**ServiceHook over a domain aggregate** — Every hook is a typed models.ServiceHook[T] that embeds models.NoopServiceHook[T] so only the relevant Pre*/Post* methods are overridden. A compile-time assertion proves conformance. (`var _ models.ServiceHook[customer.Customer] = (*entitlementValidatorHook)(nil); type entitlementValidatorHook struct { NoopEntitlementValidatorHook; entitlementService entitlement.Service }`)
**Public type aliases for the hook** — Each hook exposes an exported alias (e.g. EntitlementValidatorHook = models.ServiceHook[customer.Customer]) plus a Noop alias, while the concrete struct stays unexported and is only constructed via the New* function. (`type ( EntitlementValidatorHook = models.ServiceHook[customer.Customer]; NoopEntitlementValidatorHook = models.NoopServiceHook[customer.Customer] )`)
**Config + Validate + New constructor** — Each hook has a *HookConfig struct with a Validate() error method that collects errs via errors.Join (or single fmt.Errorf), and a New* constructor that calls config.Validate() before building the struct. Never construct the struct directly outside this package (tests are the exception). (`func NewEntitlementValidatorHook(config EntitlementValidatorHookConfig) (EntitlementValidatorHook, error) { if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid entitlement validator hook config: %w", err) } ... }`)
**OTel span per hook method** — Post*/EnsureCustomer methods open a tracer span (s.tracer.Start), defer span.End(), set span status Ok/Error, record errors, and emit AddEvent for branch outcomes (customer found/created/updated/not-found). (`ctx, span := s.tracer.Start(ctx, "subject_customer_hook.post_create"); defer span.End()`)
**Re-entrancy guard via skip context** — When the provisioner mutates the customer it wraps the call in subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx) so the customer write does not re-trigger the subject->customer hook loop. (`cus, err = p.customer.CreateCustomer(subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx), customer.CreateCustomerInput{...})`)
**Idempotent provisioning via CmpSubjectCustomer** — EnsureCustomer is the convergence point: it locates a customer by usage-attribution then by key, short-circuits with CmpSubjectCustomer when already in sync, returns GenericConflictError on key conflicts, skips soft-deleted customers (DeletedAt), and tags created/updated records with annotations createdBy/subjectId/stripeCustomerId. (`if CmpSubjectCustomer(sub, cus) { return cus, nil }`)
**Typed domain errors, never bare errors to callers** — Validation failures use models.NewGenericValidationError; not-found uses models.NewGenericNotFoundError and is detected with models.IsGenericNotFoundError; conflicts use models.NewGenericConflictError; deleted-customer preconditions use models.NewGenericPreConditionFailedError. (`return models.NewGenericValidationError(fmt.Errorf("customer has entitlements, please remove them before deleting the customer"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `subjectcustomer.go` | SubjectCustomerHook + CustomerProvisioner: PostCreate/PostUpdate call provision(); PostDelete strips the subject key from the customer's UsageAttribution.SubjectKeys; EnsureCustomer/getCustomerForSubject converge a Customer onto a Subject; EnsureStripeCustomer wires the billing CustomerOverride/payment app; MetadataFromMap/toString flatten subject metadata to models.Metadata. | PostDelete and EnsureCustomer must honour DeletedAt (skip soft-deleted customers). Key-vs-usage-attribution mismatch must yield ErrCustomerKeyConflict, not a silent overwrite. Mutations must go through NewContextWithSkipSubjectCustomer to avoid hook recursion. |
| `entitlementvalidator.go` | EntitlementValidatorHook.PreDelete blocks customer deletion when entitlement.Service.GetEntitlementsOfCustomer returns any active entitlement at clock.Now(). | Only PreDelete is overridden; do not add Post* logic here. Uses clock.Now() (not time.Now) so tests can freeze time. |
| `subjectcustomer_test.go` | Integration tests + TestEnv harness (real Postgres via testutils.InitPostgresDB, real subject/customer adapters and services, mock meter adapter, eventbus.NewMock). Provides AssertSubjectCustomerEqual / AssertSubjectCustomerStrictEqual and TestMetadataFromMap table test. | TestEnv builds dependencies from package constructors (customeradapter.New, customerservice.New, subjectservice.New) — do not import app/common wiring here. Requires Postgres; uses t.Context() and a noop tracer. |

## Anti-Patterns

- Constructing entitlementValidatorHook / subjectCustomerHook / CustomerProvisioner literally instead of via the New* constructor (skips config.Validate()).
- Overriding a hook method without embedding the Noop*Hook, breaking the models.ServiceHook[T] contract for the methods you did not implement.
- Calling customer.CreateCustomer/UpdateCustomer from within a subject hook without NewContextWithSkipSubjectCustomer, causing infinite hook recursion.
- Treating a key match as a customer match without checking UsageAttribution.SubjectKeys (must return GenericConflictError on mismatch).
- Mutating or deleting a soft-deleted customer (DeletedAt != nil) instead of skipping it.

## Decisions

- **Cross-domain glue lives in hooks rather than inside customer.Service or subject.Service.** — Keeps the core customer/subject services free of billing/entitlement imports; hooks are opt-in and wired at DI time, and IgnoreErrors lets provisioning failures degrade gracefully.
- **EnsureCustomer is idempotent and convergence-based (find-by-attribution, then find-by-key, compare, then create/update).** — Subject create/update/delete events can fire repeatedly and out of order; convergence + CmpSubjectCustomer makes re-delivery safe.
- **Subject metadata is flattened to string-valued models.Metadata via MetadataFromMap/toString.** — Customer metadata is a flat string map; the reflection-based toString deterministically serialises scalars/slices/maps (sorted) and drops unsupported values.

## Example: A new lifecycle hook on a domain aggregate (Config/Validate/New + Noop embedding + typed error).

```
type EntitlementValidatorHook = models.ServiceHook[customer.Customer]
type NoopEntitlementValidatorHook = models.NoopServiceHook[customer.Customer]

var _ models.ServiceHook[customer.Customer] = (*entitlementValidatorHook)(nil)

type entitlementValidatorHook struct {
	NoopEntitlementValidatorHook
	entitlementService entitlement.Service
}

func NewEntitlementValidatorHook(config EntitlementValidatorHookConfig) (EntitlementValidatorHook, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid entitlement validator hook config: %w", err)
	}
	return &entitlementValidatorHook{entitlementService: config.EntitlementService}, nil
// ...
```

<!-- archie:ai-end -->
