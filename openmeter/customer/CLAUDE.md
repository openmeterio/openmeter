# customer

<!-- archie:ai-start -->

> Customer lifecycle domain (CRUD, usage attributions, soft-delete via DeletedAt) exposing two extension registries — RequestValidatorRegistry for pre-mutation cross-domain guards and ServiceHooks[Customer] for post-lifecycle callbacks — that let billing, ledger, subscription, and entitlement react to customer changes without circular imports.

## Patterns

**Root = contract; service/ orchestrates; adapter/ persists; httpdriver/ adapts HTTP** — Root files (service.go, customer.go, requestvalidator.go, adapter.go, errors.go, event.go) define the interface and domain types. service/ runs validators before the transaction and hooks inside it; adapter/ is the sole DB layer (TransactingRepo on every method); httpdriver/ translates api.* types via apimapping.go. (`service/customer.go: transaction.Run(ctx, s.adapter, func(ctx, tx customer.Adapter) (*Customer, error) { ... })`)
**Validators before the transaction, hooks inside it** — RequestValidatorRegistry and ServiceHooks[Customer] are separate registries with distinct timing: validators (errors.Join fan-out) block the mutation before any DB write; hooks fire PostCreate/PostUpdate/PostDelete inside transaction.Run so a hook failure rolls back the write. (`requestvalidator.go: requestValidatorRegistry.ValidateDeleteCustomer fans out to all registered validators`)
**Re-entrancy guard for hooks that call back into customer.Service** — Hooks (e.g. subject-customer) that invoke customer.Service must wrap ctx with NewContextWithSkipSubjectCustomer(ctx) to prevent infinite re-entrant hook invocations. Sub-package service/hooks/ holds entitlementvalidator and subjectcustomer hook implementations. (`ctx = customer.NewContextWithSkipSubjectCustomer(ctx) before re-entering the service from a hook`)
**Soft-delete everywhere, never hard-delete** — DeleteCustomer sets DeletedAt; queries default IncludeDeleted=false; Customer.IsDeleted() checks DeletedAt.Before(clock.Now()). Billing/subscription/entitlement reference customer IDs and must not be orphaned. (`func (c Customer) IsDeleted() bool { return c.DeletedAt != nil && c.DeletedAt.Before(clock.Now()) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/customer/service.go` | Composite customer.Service interface = CustomerService + RequestValidatorService + models.ServiceHooks[Customer]. | RegisterHooks and RegisterRequestValidator are separate registration points; cross-domain constraints often need both. |
| `openmeter/customer/customer.go` | Customer/CustomerMutate/CustomerID/CustomerUsageAttribution domain types and all input types with Validate(). | GetCustomerInput supports by-ID, by-Key, and by-IDOrKey lookups — always build a typed lookup struct; CustomerID is models.NamespacedID. Validate() requires either key or usageAttribution.subjectKeys. |
| `openmeter/customer/requestvalidator.go` | RequestValidator interface, NoopRequestValidator, and the RWMutex fan-out registry. | Register() is not idempotent — double-registration doubles validation calls; Wire providers must guard against it. |
| `openmeter/customer/errors.go` | Typed domain errors (KeyConflictError, SubjectKeyConflictError, UpdateAfterDeleteError, ErrDeletingCustomerWithActiveSubscriptions). | ErrDeletingCustomerWithActiveSubscriptions is a models.ValidationIssue (not a GenericError) carrying active subscription IDs in attrs. |

## Anti-Patterns

- Calling customer.Service from inside a hook without NewContextWithSkipSubjectCustomer(ctx) — causes infinite re-entrant hook invocations.
- Performing DB writes outside transaction.Run / transaction.RunWithNoValue in the service layer — partial writes are not rolled back on hook or publish failure.
- Returning raw fmt.Errorf for not-found/conflict/validation conditions — HTTP encoders depend on models.Generic* typed errors.
- Hard-deleting customer or customer_subjects rows — the domain uses soft-delete via DeletedAt everywhere.
- Importing app/common in testutils or test files — causes import cycles; build deps from customertestutils.NewTestEnv.

## Decisions

- **RequestValidatorRegistry and ServiceHooks are separate registries with distinct call sites.** — Pre-mutation validators block before any DB write; post-lifecycle hooks run after success and may have side-effects. Merging them would lose the timing guarantee.
- **Soft-delete rather than hard delete for customers and customer_subjects.** — Billing, subscription, and entitlement records reference customer IDs; soft-delete preserves referential integrity while logically removing the customer.

<!-- archie:ai-end -->
