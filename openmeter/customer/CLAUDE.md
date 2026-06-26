# customer

<!-- archie:ai-start -->

> High-fan-in (103 in-edges) customer domain. The root declares customer.Service/Adapter, the Customer + CustomerMutate model, identity types (CustomerID/CustomerKey/CustomerIDOrKey), all *Input types, lifecycle events, typed conflict errors, and a pluggable RequestValidator registry. Consumed by billing, subscription, ledger and the v3 customers handlers.

## Patterns

**Customer satisfies streaming.Customer / usage-attribution** — Customer embeds models.ManagedResource and exposes GetUsageAttribution() returning streaming.NewCustomerUsageAttribution(ID, Key, subjectKeys); a `var _ streaming.Customer = &Customer{}` assertion enforces it. (`func (c Customer) GetUsageAttribution() streaming.CustomerUsageAttribution { return streaming.NewCustomerUsageAttribution(c.ID, c.Key, subjectKeys) }`)
**Either key or subjectKeys required** — Both Customer.Validate and CustomerMutate.Validate enforce that at least one of Key or UsageAttribution.SubjectKeys is set, returning models.NewGenericValidationError. (`if !hasKey && !hasSubjectKeys { return models.NewGenericValidationError(errors.New("either key or usageAttribution.subjectKeys must be provided")) }`)
**Pluggable RequestValidator registry** — RequestValidatorRegistry fans validation to all registered RequestValidators (errors.Join over lo.Map) under an RWMutex; validators are Register'd after construction, not via Config. NoopRequestValidator is the default. (`return errors.Join(lo.Map(r.validators, func(v RequestValidator, _ int) error { return v.ValidateDeleteCustomer(ctx, input) })...)`)
**Typed conflict / precondition errors with IsX helpers** — KeyConflictError, SubjectKeyConflictError, UpdateAfterDeleteError wrap models.NewGenericConflictError; deleting a customer with active subscriptions uses the ValidationIssue ErrDeletingCustomerWithActiveSubscriptions.WithAttr. (`var ErrDeletingCustomerWithActiveSubscriptions = models.NewValidationIssue(ErrCodeDeletingCustomerWithActiveSubscriptions, "cannot delete customer with active subscriptions")`)
**Filter fields via pkg/filter wrappers** — ListCustomersInput exposes *filter.FilterString / *filter.FilterULID per filterable field, each Validate'd and collected with models.NewNillableGenericValidationError(errors.Join(...)). (`Key *filter.FilterString; BillingProfileID *filter.FilterULID`)
**Service embeds models.ServiceHooks[Customer]** — customer.Service composes CustomerService + RequestValidatorService + models.ServiceHooks[Customer], so lifecycle hooks (e.g. ledger account provisioning) register post-construction. (`type Service interface { CustomerService; RequestValidatorService; models.ServiceHooks[Customer] }`)
**Soft-delete-aware mutation guards** — Customer.IsDeleted() compares DeletedAt against clock.Now(); mutations on deleted customers must return a precondition-failed error rather than proceeding. (`func (c Customer) IsDeleted() bool { return c.DeletedAt != nil && c.DeletedAt.Before(clock.Now()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `customer.go` | Customer + CustomerMutate model, identity types, CustomerUsageAttribution, all *Input types and Validate | AsCustomerMutate drops some fields; GetFirstSubjectKey is deprecated; keep the key-or-subjectKeys invariant in BOTH Customer and CustomerMutate Validate. |
| `adapter.go` | Adapter interface (List/Create/Update/Delete/Get + usage-attribution lookups) embedding entutils.TxCreator | GetCustomerByUsageAttribution keys on customer key or subject key; reads must load subjects or mapping fails. |
| `service.go` | Service interface composing CustomerService + RequestValidatorService + ServiceHooks[Customer] | Read methods delegate straight to adapter; do not add validation/hooks/events to Get/List. |
| `requestvalidator.go` | RequestValidator interface, NoopRequestValidator, thread-safe registry | Validators run for all registered entries; any returned error aborts the mutation. |
| `errors.go` | KeyConflictError, SubjectKeyConflictError, UpdateAfterDeleteError, active-subscription deletion issue | Return these typed errors so v3 handlers surface correct conflict/precondition codes. |
| `event.go` | CustomerCreate/Update/Delete events (all v1), resource-path metadata | Delete event Validate requires DeletedAt != nil before EventMetadata dereferences it. |

## Anti-Patterns

- Putting Ent/SQL access in the root package or service instead of delegating to customer.Adapter.
- Mutating or deleting a soft-deleted customer (IsDeleted) without the precondition guard, or deleting one with active subscriptions.
- Returning bare errors for invalid input instead of models.NewGenericValidationError / the typed conflict errors.
- Bypassing the RequestValidator registry on a new mutation path.
- Breaking the key-or-subjectKeys invariant, or constructing usage attribution directly instead of via GetUsageAttribution (it carries Key too).

## Decisions

- **Request validators and lifecycle hooks register after construction, not via Config.** — Avoids import cycles with downstream domains (billing/subscription/ledger) that hook into customer mutations while still depending on customer.
- **Either a customer key or subject keys must identify a customer.** — Usage attribution must always resolve to a customer; the dual identity keeps backwards-compatible subject-key flows while supporting keyed customers.

## Example: Validating a list input by collecting all filter issues

```
func (i ListCustomersInput) Validate() error {
	var errs []error
	if i.Namespace == "" { errs = append(errs, models.NewGenericValidationError(errors.New("namespace is required"))) }
	if i.Key != nil {
		if err := i.Key.Validate(); err != nil { errs = append(errs, models.NewGenericValidationError(fmt.Errorf("invalid key filter: %w", err))) }
	}
	return models.NewNillableGenericValidationError(errors.Join(errs...))
}
```

<!-- archie:ai-end -->
