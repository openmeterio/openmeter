# hooks

<!-- archie:ai-start -->

> Lifecycle hook implementations for the customer domain — entitlement validation pre-delete guard and subject-to-customer auto-provisioning. New hooks here integrate with the models.ServiceHook[T] registry without coupling the customer package to callers.

## Patterns

**ServiceHook type alias pair** — Each hook file declares a public type alias for the interface (e.g. EntitlementValidatorHook = models.ServiceHook[customer.Customer]) and a Noop alias (NoopEntitlementValidatorHook = models.NoopServiceHook[customer.Customer]). The concrete struct embeds the Noop so only overridden methods need implementing. (`type EntitlementValidatorHook = models.ServiceHook[customer.Customer]
type NoopEntitlementValidatorHook = models.NoopServiceHook[customer.Customer]
var _ models.ServiceHook[customer.Customer] = (*entitlementValidatorHook)(nil)`)
**Config struct + Validate() + constructor** — Every hook is constructed via a Config struct with a Validate() error method. The constructor calls Validate() and returns (Hook, error). Nil-dependency checks are mandatory inside Validate. (`func NewEntitlementValidatorHook(config EntitlementValidatorHookConfig) (EntitlementValidatorHook, error) {
  if err := config.Validate(); err != nil { return nil, fmt.Errorf(...) }
  return &entitlementValidatorHook{...}, nil
}`)
**Compile-time interface assertion** — Every concrete hook struct declares a package-level var _ assertion to verify interface satisfaction at compile time. (`var _ models.ServiceHook[customer.Customer] = (*entitlementValidatorHook)(nil)`)
**OTel tracing in hook methods** — Non-trivial hook methods (PostCreate, PostUpdate, PostDelete) start a span via tracer.Start, record errors with span.RecordError + span.SetStatus, and defer span.End. The tracer is injected through the Config struct. (`func (s subjectCustomerHook) PostCreate(ctx context.Context, sub *subject.Subject) error {
  ctx, span := s.tracer.Start(ctx, "subject_customer_hook.post_create")
  defer span.End()
  ...
}`)
**Loop-prevention via context key (NewContextWithSkipSubjectCustomer)** — When a hook calls back into the customer service (UpdateCustomer/CreateCustomer), it wraps ctx with subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx) to break the re-entrant hook cycle. Any new hook that calls customer.Service must use the same guard. (`cus, err = p.customer.UpdateCustomer(
  subjectservicehooks.NewContextWithSkipSubjectCustomer(ctx),
  customer.UpdateCustomerInput{...})`)
**models.GenericError types for domain errors** — Validation failures return models.NewGenericValidationError, not-found returns models.NewGenericNotFoundError, key conflicts return models.NewGenericConflictError, precondition failures return models.NewGenericPreConditionFailedError. Never return raw errors for domain conditions. (`return models.NewGenericValidationError(fmt.Errorf("customer has entitlements, please remove them before deleting the customer"))`)
**Test environment built from package constructors (no app/common)** — Tests use NewTestEnv which wires adapters and services directly (customeradapter.New, customerservice.New, subjectadapter.New, subjectservice.New) without importing app/common. testutils.InitPostgresDB provides the Ent client; eventbus.NewMock provides the publisher. (`db := testutils.InitPostgresDB(t)
customerAdapter, _ := customeradapter.New(customeradapter.Config{Client: db.EntDriver.Client(), Logger: logger})
customerService, _ := customerservice.New(customerservice.Config{Adapter: customerAdapter, Publisher: publisher})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlementvalidator.go` | PreDelete hook for customer.Customer — blocks deletion when active entitlements exist. Minimal hook that only overrides one method; all others fall through to Noop. | Uses clock.Now() for entitlement snapshot time; always pass the caller's ctx, never context.Background(). |
| `subjectcustomer.go` | PostCreate/PostUpdate/PostDelete hook for subject.Subject — auto-provisions or updates a matching Customer. Also houses CustomerProvisioner (reusable create/update logic) and EnsureStripeCustomer (billing override sync). | getCustomerForSubject tries both usage-attribution lookup and key lookup; EnsureCustomer handles keyConflict by omitting the key on create. CmpSubjectCustomer determines whether an update is needed — extend carefully to avoid spurious updates. |
| `subjectcustomer_test.go` | Integration tests for CustomerProvisioner — exercises create, update, and conflict scenarios against a real Postgres instance provisioned by testutils.InitPostgresDB. | TestEnv is local to this package and independent from app/common. Tests call env.DBSchemaMigrate(t) before using the DB. Use t.Context(), not context.Background(). |

## Anti-Patterns

- Calling customer.Service from a hook without NewContextWithSkipSubjectCustomer(ctx) — causes infinite re-entrant hook invocations.
- Returning raw fmt.Errorf for domain conditions (not-found, conflict, validation) — callers and HTTP encoders depend on models.Generic* error types.
- Importing app/common in test files — breaks isolation and can introduce import cycles; build test deps from package constructors directly.
- Embedding a non-Noop base struct — hooks must embed the Noop alias so future interface method additions don't break compilation.
- Using context.Background() or context.TODO() inside hook methods — always propagate the caller's ctx to maintain tracing and cancellation.

## Decisions

- **Hooks embed models.NoopServiceHook[T] rather than implementing all methods.** — The ServiceHook interface grows as new lifecycle events are added; embedding Noop provides safe defaults so existing hooks don't break on interface expansion.
- **CustomerProvisioner is a separate struct from the hook.** — EnsureCustomer and EnsureStripeCustomer are reused from outside the hook (e.g. app/stripe provisioning flows), so extracting them into a standalone CustomerProvisioner avoids duplication without tying the logic to the subject lifecycle.
- **IgnoreErrors flag on subjectCustomerHook.** — Subject creation/update should not fail if customer provisioning is non-critical in some deployment modes; the flag lets the caller degrade gracefully (warn + continue) rather than surfacing the error.

## Example: Adding a new PreDelete hook that checks billing conditions before customer deletion

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
