# secret

<!-- archie:ai-start -->

> Defines the Adapter and Service interfaces for storing/retrieving app secrets (e.g. Stripe API keys); the package root holds only interface declarations while entity/, service/, and adapter/ hold the types, validate-and-delegate service, and the in-memory plaintext implementation respectively.

## Patterns

**Interface-only package root** — secret.go and adapter.go declare Service/Adapter interfaces (each composing a single SecretService/SecretAdapter); no implementations live at the root. (`type Service interface { SecretService }`)
**Mirrored Service/Adapter contracts** — Service and Adapter expose the identical CreateAppSecret/UpdateAppSecret/GetAppSecret/DeleteAppSecret method set — service validates then delegates to adapter. (`GetAppSecret(ctx, input secretentity.GetAppSecretInput) (secretentity.Secret, error)`)
**Input/output types live in entity/** — All method signatures take secretentity.* input structs and return secretentity.SecretID/Secret; never raw strings. (`secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Declares Adapter/SecretAdapter — the storage contract implemented by secret/adapter. | GetAppSecret in the default adapter echoes input.ID back as the value; it is not a real lookup. |
| `service.go` | Declares Service/SecretService — the validation boundary implemented by secret/service. | The service is the only layer that validates; do not skip input.Validate() there. |

## Anti-Patterns

- Putting validation or business logic in the package-root interfaces or the adapter — validation belongs only in secret/service.
- Adding a method to Service without adding the matching method to Adapter (they must stay mirrored).
- Threading raw secret strings instead of secretentity.SecretID/Secret types.

## Decisions

- **Split Service and Adapter into separate interfaces with identical method sets.** — Lets a real secret store replace the in-memory plaintext adapter without touching the service or callers.

## Example: The storage contract every secret adapter must satisfy

```
type SecretAdapter interface {
	CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error)
	UpdateAppSecret(ctx context.Context, input secretentity.UpdateAppSecretInput) (secretentity.SecretID, error)
	GetAppSecret(ctx context.Context, input secretentity.GetAppSecretInput) (secretentity.Secret, error)
	DeleteAppSecret(ctx context.Context, input secretentity.DeleteAppSecretInput) error
}
```

<!-- archie:ai-end -->
