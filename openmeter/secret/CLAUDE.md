# secret

<!-- archie:ai-start -->

> Manages encrypted secrets (e.g., Stripe API keys) for installed marketplace apps. The root package owns the Service and Adapter interfaces; sub-packages provide the stub plaintext adapter (adapter/), pure domain types (entity/), and the thin validation-only service implementation (service/).

## Patterns

**Interface segregation: Adapter embeds SecretAdapter** — secret.Adapter embeds secret.SecretAdapter; new storage operations must be added to SecretAdapter, not directly to Adapter. (`type Adapter interface { SecretAdapter }`)
**Validate-then-delegate in every method** — service/ calls input.Validate() unconditionally before delegating to the adapter — the stub adapter will not catch bad inputs. (`if err := input.Validate(); err != nil { return secretentity.SecretID{}, err }; return s.adapter.CreateAppSecret(ctx, input)`)
**Config struct constructor with Validate()** — service.New accepts a Config struct and calls Validate() immediately — the only nil-safety gate before storing the adapter. (`func New(cfg Config) (Service, error) { if err := cfg.Validate(); err != nil { return nil, err }; return &service{adapter: cfg.Adapter}, nil }`)
**Pure domain types in entity/ sub-package** — All input structs, SecretID, Secret, and SecretNotFoundError live in openmeter/secret/entity — no persistence logic or Ent imports. (`import secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"`)
**SecretID encodes plaintext value (stub adapter)** — The plaintext adapter stores the raw secret value as the ID field of SecretID — a known stub convention until a real secret store is wired. (`return secretentity.SecretID{ID: input.Value}, nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/secret/adapter.go` | Defines the Adapter and SecretAdapter interfaces consumed by the service layer. | Adding persistence logic here — storage belongs in adapter sub-packages, not the interface file. |
| `openmeter/secret/service.go` | Defines the Service and SecretService interfaces. | New operations must be added to SecretService first, then implemented in service/. |
| `openmeter/secret/entity/secret.go` | Pure domain types: SecretID, Secret, TaxCodeAppMapping. | Importing openmeter/ent/db or any persistence package — this file must stay dependency-free. |
| `openmeter/secret/entity/input.go` | All input structs (CreateAppSecretInput, UpdateAppSecretInput, etc.) each with a Validate() method. | New input structs added without a Validate() method — service layer calls Validate() unconditionally. |
| `openmeter/secret/entity/errors.go` | SecretNotFoundError wrapping models.GenericNotFoundError for HTTP status mapping. | Returning raw errors instead of models.NewGenericNotFoundError — breaks HTTP status code mapping. |

## Anti-Patterns

- Adding business logic or cross-domain calls to service/ — it is a thin validation+delegation layer only
- Injecting *entdb.Client directly into Service — persistence is the adapter's concern
- Importing openmeter/ent/db in entity/ — it must remain a pure domain type package
- Returning raw errors from stub adapter instead of nil — callers expect CreateAppSecret and DeleteAppSecret to succeed
- Bypassing the Config struct and constructing service{} directly — Validate() in New() is the only nil-safety gate

## Decisions

- **Input types live in a separate entity/ sub-package** — Keeps the root secret package to pure interfaces while allowing adapter and service sub-packages to import input types without circular dependencies.
- **Plaintext stub adapter stores value in SecretID.ID** — Allows the rest of the app to function without a real secret store during development; a future production adapter can replace this without changing the Adapter interface.

## Example: Implementing a new secret adapter

```
package myadapter

import (
	"context"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

type adapter struct{}

func (a *adapter) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
	if err := input.Validate(); err != nil {
		return secretentity.SecretID{}, err
	}
	// ... store and return ID
}
```

<!-- archie:ai-end -->
