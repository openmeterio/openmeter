# adapter

<!-- archie:ai-start -->

> Plaintext, in-memory implementation of the secret.Adapter interface used to store/retrieve app secrets (e.g. Stripe API keys). The ID and the value are identical here; production deployments are expected to swap this for a real secret store without changing the interface.

## Patterns

**Stateless struct adapter** — The adapter is an empty struct constructed via New() returning the secret.Adapter interface; it holds no DB client or state. (`func New() secret.Adapter { return &adapter{} }; type adapter struct{}`)
**Compile-time interface assertion** — A var _ secret.Adapter = (*adapter)(nil) line guarantees the struct satisfies the interface from openmeter/secret. (`var _ secret.Adapter = (*adapter)(nil)`)
**ID-equals-value plaintext encoding** — CreateAppSecret/UpdateAppSecret build the SecretID with NewSecretID(input.AppID, input.Value, input.Key) so the stored ID literally is the value; GetAppSecret reverses this by reading value := input.ID. (`return secretentity.NewSecretID(input.AppID, input.Value, input.Key), nil`)
**No validation at adapter layer** — Adapter methods do not call input.Validate(); validation is the service layer's responsibility. Adapter trusts its inputs. (`func (a adapter) CreateAppSecret(ctx, input) (...) { return secretentity.NewSecretID(...), nil }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor New() and the empty adapter struct plus the interface assertion. | New() returns the secret.Adapter interface, not *adapter; keep the var _ assertion in sync if methods change. |
| `secret.go` | Implements the four SecretAdapter methods (Create/Update/Get/Delete AppSecret). | This is a plaintext stub — Get returns the ID as the value and Delete is a no-op. Do NOT treat this as secure storage; a real secret-store impl must change the ID<->value mapping. |

## Anti-Patterns

- Putting input validation here — that belongs in openmeter/secret/service.
- Adding persistence/DB state to the struct without also threading namespacing and tx-awareness used elsewhere in the codebase.
- Assuming GetAppSecret performs a lookup — it just echoes input.ID back as the value.

## Decisions

- **ID and value are identical in this implementation.** — It is an intentional plaintext placeholder; real implementations would store the value in an external secret store and return an opaque reference ID.

## Example: Constructing the secret store adapter and creating a secret

```
import (
	"github.com/openmeterio/openmeter/openmeter/secret/adapter"
	secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
)

a := adapter.New()
id, err := a.CreateAppSecret(ctx, secretentity.CreateAppSecretInput{AppID: appID, Key: "stripe-api-key", Value: secretValue})
```

<!-- archie:ai-end -->
