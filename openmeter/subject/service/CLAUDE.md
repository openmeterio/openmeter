# service

<!-- archie:ai-start -->

> Implements subject.Service (the write/read service layer for Subjects) over a subject.Adapter, wrapping every mutation in a transaction and firing service hooks at lifecycle boundaries. Subjects are usage-attribution identities kept in sync with customers; this service is the only write path and enforces input validation plus hook ordering.

## Patterns

**Service wraps Adapter, never DB directly** — Service struct holds a single subjectAdapter subject.Adapter plus a models.ServiceHookRegistry[subject.Subject]; all persistence goes through the adapter, never Ent. (`type Service struct { subjectAdapter subject.Adapter; hooks models.ServiceHookRegistry[subject.Subject] }`)
**Compile-time interface assertion** — var _ subject.Service = (*Service)(nil) at package top guarantees the struct satisfies the domain interface. (`var _ subject.Service = (*Service)(nil)`)
**Constructor validates required deps** — New(subjectAdapter) returns (*Service, error), rejects a nil adapter with fmt.Errorf, and initializes an empty ServiceHookRegistry. (`if subjectAdapter == nil { return nil, fmt.Errorf("subject adapter is required") }`)
**Validate input then wrap as generic validation error** — Mutating methods call input.Validate() first and wrap the result in models.NewGenericValidationError; GetById/GetByKey/Delete validate the identifier (id.Validate()/key.Validate()) the same way. (`if err := input.Validate(); err != nil { return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err)) }`)
**Mutations run inside transaction.Run with hooks** — Create/Update/Delete execute via transaction.Run / transaction.RunWithNoValue against s.subjectAdapter, firing PostCreate, Pre/PostUpdate, or Pre/PostDelete around the adapter call within the same tx. (`return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) { sub, err := s.subjectAdapter.Create(ctx, input); ...; err = s.hooks.PostCreate(ctx, &sub); ... })`)
**Update/Delete re-fetch before mutating** — Update and Delete first GetById the current subject so Pre-hooks observe existing state, then perform the adapter write, then fire Post-hooks. (`sub, err := s.subjectAdapter.GetById(ctx, models.NamespacedID{Namespace: input.Namespace, ID: input.ID})`)
**Read paths bypass transaction and hooks** — GetByIdOrKey, GetById, GetByKey, List delegate straight to the adapter (with identifier validation where applicable) and do not open a transaction or fire hooks. (`func (s *Service) List(ctx, orgId, params) { return s.subjectAdapter.List(ctx, orgId, params) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | The subject.Service implementation: Service struct, New constructor, RegisterHooks, and all CRUD+List methods. | Keep mutating methods inside transaction.Run/RunWithNoValue and fire the matching hook (PostCreate / Pre+PostUpdate / Pre+PostDelete). Do not add a write path that skips the hook registry or the validate+wrap step. |
| `service_test.go` | Integration test Test_SubjectService driving Create/Get/Update/Delete through subjecttestutils.NewTestEnv with real DB migrations. | Delete must NOT cascade into entitlements (they belong to the customer, not the subject); deleting a subject soft-deletes it (byID.IsDeleted()) and the key can be re-created afterward. Build deps via subjecttestutils, not app/common, to avoid import cycles. |

## Anti-Patterns

- Calling subjectAdapter directly for writes without transaction.Run / RunWithNoValue — breaks atomicity and skips hooks.
- Adding a Create/Update/Delete path that does not fire the corresponding ServiceHookRegistry method (PostCreate, Pre/PostUpdate, Pre/PostDelete).
- Skipping input.Validate() + models.NewGenericValidationError wrapping on a new mutating method.
- Hard-deleting subjects or cascading deletes into entitlements — Delete is soft (IsDeleted) and entitlements outlive the subject.
- Constructing Service via a struct literal instead of New, or passing a nil adapter.

## Decisions

- **Service depends only on subject.Adapter plus a hook registry, not on customer/entitlement services.** — Keeps the subject domain decoupled; cross-domain sync (subject<->customer) is handled by registered ServiceHooks (see the hooks child) rather than direct service calls, avoiding circular wiring.
- **All mutations are transactional and re-fetch current state before Pre-hooks.** — Ensures hooks observe a consistent snapshot and that the adapter write plus hook side effects commit or roll back together.
- **Subject deletion is soft and never touches entitlements.** — Entitlements are owned by the customer; a subject is only a usage-attribution key, so its lifecycle must not destroy customer-scoped balances (asserted in service_test.go).

## Example: Transactional mutation firing a service hook

```
import (
	"context"
	"fmt"

	"github.com/openmeterio/openmeter/openmeter/subject"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
	"github.com/openmeterio/openmeter/pkg/models"
)

func (s *Service) Create(ctx context.Context, input subject.CreateInput) (subject.Subject, error) {
	if err := input.Validate(); err != nil {
		return subject.Subject{}, fmt.Errorf("invalid input: %w", models.NewGenericValidationError(err))
	}
	return transaction.Run(ctx, s.subjectAdapter, func(ctx context.Context) (subject.Subject, error) {
		sub, err := s.subjectAdapter.Create(ctx, input)
// ...
```

<!-- archie:ai-end -->
