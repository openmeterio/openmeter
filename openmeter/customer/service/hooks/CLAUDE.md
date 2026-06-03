# hooks

<!-- archie:ai-start -->

> Lifecycle hook implementations for the customer domain — entitlement validation pre-delete guard (entitlementvalidator.go) and subject-to-customer auto-provisioning (subjectcustomer.go). Hooks integrate with the models.ServiceHook[T] registry so the customer package never couples to its callers.

## Patterns

**ServiceHook type-alias pair + concrete struct embeds Noop** — Each hook file declares a public ServiceHook[T] alias and a Noop alias; the concrete struct embeds the Noop so only overridden lifecycle methods need implementing. (`type EntitlementValidatorHook = models.ServiceHook[customer.Customer]; type NoopEntitlementValidatorHook = models.NoopServiceHook[customer.Customer]`)
**Config struct + Validate() + constructor** — Every hook is built via a Config with a Validate() (mandatory nil-dependency checks); the constructor calls Validate() first and wraps the error. (`func NewEntitlementValidatorHook(config EntitlementValidatorHookConfig) (EntitlementValidatorHook, error) { if err := config.Validate(); err != nil { return nil, fmt.Errorf("invalid entitlement validator hook config: %w", err) }; ... }`)
**Compile-time interface assertion** — Every concrete hook declares a package-level var _ assertion to verify interface satisfaction at compile time. (`var _ models.ServiceHook[customer.Customer] = (*entitlementValidatorHook)(nil)`)
**OTel tracing in non-trivial hook methods** — Non-trivial methods start a span via the Config-injected tracer, record errors / add events, and defer span.End. (`ctx, span := s.tracer.Start(ctx, "subject_customer_hook.post_create"); defer span.End()`)
**Loop-prevention via context key** — When a hook calls back into customer.Service (Update/Create), it wraps ctx with subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx) to break re-entrant hook cycles. (`cus, err = p.customer.UpdateCustomer(subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx), customer.UpdateCustomerInput{...})`)
**models.Generic* error types for domain errors** — Domain failures return models.NewGenericValidationError / NotFoundError / ConflictError / PreConditionFailedError — never raw errors, since HTTP encoders map by type. (`return models.NewGenericValidationError(fmt.Errorf("customer has entitlements, please remove them before deleting the customer"))`)
**Tests built from package constructors (no app/common)** — Tests wire adapters and services directly (customeradapter.New, customerservice.New, subjectadapter.New) with testutils.InitPostgresDB and eventbus.NewMock; never import app/common. (`db := testutils.InitPostgresDB(t); customerService, _ := customerservice.New(customerservice.Config{Adapter: customerAdapter, Publisher: publisher})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlementvalidator.go` | PreDelete hook for customer.Customer — blocks deletion when active entitlements exist; overrides only PreDelete, rest fall through to the embedded Noop. | Uses clock.Now() for the entitlement snapshot; always pass the caller's ctx, never context.Background(). |
| `subjectcustomer.go` | PostCreate/PostUpdate/PostDelete hook for subject.Subject auto-provisioning a matching Customer; houses CustomerProvisioner (reusable create/update logic) and EnsureStripeCustomer. | getCustomerForSubject tries both usage-attribution and key lookup; EnsureCustomer handles key conflicts by omitting the key on create; CmpSubjectCustomer decides whether an update is needed — extend carefully to avoid spurious updates. |
| `subjectcustomer_test.go` | Integration tests for CustomerProvisioner against a real Postgres via testutils.InitPostgresDB (create/update/conflict scenarios). | TestEnv is local and independent of app/common; call env.DBSchemaMigrate(t) before using the DB; use t.Context(), not context.Background(). |

## Anti-Patterns

- Calling customer.Service from a hook without subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx) — causes infinite re-entrant hook invocations.
- Returning raw fmt.Errorf for domain conditions — callers and HTTP encoders depend on models.Generic* error types.
- Importing app/common in test files — breaks isolation and can introduce import cycles.
- Embedding a non-Noop base struct — hooks must embed the Noop alias so future interface method additions don't break compilation.
- Using context.Background()/context.TODO() inside hook methods — always propagate the caller's ctx.

## Decisions

- **Hooks embed models.NoopServiceHook[T] rather than implementing all methods** — The ServiceHook interface grows as new lifecycle events are added; embedding Noop provides safe defaults so existing hooks don't break on interface expansion.
- **CustomerProvisioner is a separate struct from the hook** — EnsureCustomer/EnsureStripeCustomer are reused outside the hook (e.g. app/stripe provisioning), so extracting them avoids duplication without tying logic to the subject lifecycle.
- **IgnoreErrors flag on subjectCustomerHook** — Subject create/update should not fail when customer provisioning is non-critical in some deployment modes; the flag lets callers warn-and-continue.

## Example: Adding a new PreDelete hook for the customer lifecycle

```
package hooks

import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/billing"
	"github.com/openmeterio/openmeter/openmeter/customer"
	"github.com/openmeterio/openmeter/pkg/models"
)

type (
	BillingValidatorHook     = models.ServiceHook[customer.Customer]
	NoopBillingValidatorHook = models.NoopServiceHook[customer.Customer]
)
// ...
```

<!-- archie:ai-end -->
