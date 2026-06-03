# secret

<!-- archie:ai-start -->

> Manages encrypted secrets (e.g. Stripe API keys) for installed marketplace apps. The root package owns the Service and Adapter interfaces; sub-packages provide the stub plaintext adapter (adapter/), pure import-cycle-free domain types (entity/), and a thin validation-only service implementation (service/).

## Patterns

**Adapter embeds SecretAdapter; Service embeds SecretService** — New storage/service operations are added to the inner SecretAdapter/SecretService interface, never directly to the wrapping Adapter/Service. (`type Adapter interface { SecretAdapter }`)
**Validate-then-delegate in service/** — service/ calls input.Validate() unconditionally before delegating to the adapter; the stub adapter does no validation. (`if err := input.Validate(); err != nil { return secretentity.SecretID{}, err }; return s.adapter.CreateAppSecret(ctx, input)`)
**Pure domain types in entity/ sub-package** — SecretID, Secret, all input structs (each with Validate()), and SecretNotFoundError live in entity/ with zero Ent/persistence imports — this breaks the openmeter/app <-> openmeter/secret import cycle. (`import secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"`)
**Stub adapter encodes plaintext value in SecretID.ID** — The plaintext adapter stores the raw secret value as SecretID.ID — a known stub convention until a real secret store is wired; CreateAppSecret/DeleteAppSecret always succeed. (`return secretentity.SecretID{ID: input.Value}, nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/secret/adapter.go` | Defines Adapter/SecretAdapter interfaces consumed by the service layer. | Adding persistence logic here — storage belongs in adapter sub-packages. |
| `openmeter/secret/service.go` | Defines Service/SecretService interfaces. | New methods must be added to SecretService first, then implemented in service/. |

## Anti-Patterns

- Adding business logic or cross-domain calls to service/ — it is validation + delegation only
- Injecting *entdb.Client into Service — persistence is the adapter's concern
- Importing openmeter/ent/db in entity/ — it must stay a pure domain-type package to preserve the cycle break
- Returning errors from the stub adapter's Create/Delete — callers expect them to succeed

## Decisions

- **Input types live in a separate entity/ sub-package** — Keeps the root package to pure interfaces and lets adapter and service import input types without circular dependencies.
- **Plaintext stub adapter stores value in SecretID.ID** — Lets the app run without a real secret store in development; a production adapter can replace it without changing the Adapter interface.

<!-- archie:ai-end -->
