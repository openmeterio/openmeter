# adapter

<!-- archie:ai-start -->

> Plaintext (no-op) stub implementation of secret.Adapter — stores secrets as their raw value encoded into SecretID itself. Satisfies the Adapter interface without an external secret store; a production deployment replaces this with a real backend (Vault, AWS Secrets Manager) without changing the interface.

## Patterns

**Compile-time interface assertion at package top** — Every adapter file must declare `var _ secret.Adapter = (*adapter)(nil)` to catch interface drift at compile time. When secret.Adapter gains a new method the assertion fails here first. (`var _ secret.Adapter = (*adapter)(nil)`)
**Stateless adapter struct — state encoded in SecretID** — The adapter struct holds no fields because the plaintext value is encoded directly into the SecretID returned by Create/Update. A real backend adapter would inject its client via New() and store it on the struct. (`type adapter struct{}
func New() secret.Adapter { return &adapter{} }`)
**SecretID IS the plaintext value** — CreateAppSecret and UpdateAppSecret return secretentity.NewSecretID(input.AppID, input.Value, input.Key) — the ID field carries the raw secret. GetAppSecret reconstructs Secret.Value from input.ID. This is intentional stub behaviour; a real adapter must never do this. (`func (a adapter) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
    return secretentity.NewSecretID(input.AppID, input.Value, input.Key), nil
}`)
**No Ent dependency** — This adapter intentionally has no *entdb.Client. Only secretentity types are imported. A future real adapter sub-package would receive a client via New() but must still not import openmeter/ent/db directly in this stub package. (`import secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor New() and compile-time interface assertion. Wire calls New() to obtain a secret.Adapter. | If secret.Adapter gains new methods the assertion `var _ secret.Adapter = (*adapter)(nil)` fails here — add the method to secret.go. |
| `secret.go` | All four CRUD method implementations: CreateAppSecret, UpdateAppSecret, GetAppSecret, DeleteAppSecret. All are no-ops that encode/decode value in SecretID. | GetAppSecret sets Secret.Value = input.ID (the plaintext value was stored as the ID). DeleteAppSecret always returns nil. Both are correct for the stub but wrong for a real backend. |

## Anti-Patterns

- Storing state on the adapter struct without injecting it through New() — the stub is intentionally stateless
- Returning errors from DeleteAppSecret or CreateAppSecret in the stub — callers expect these to always succeed
- Importing openmeter/ent/db in this package — this stub has no DB dependency; Ent belongs in a future real adapter sub-package
- Adding validation logic here — validation belongs in secretservice (service/), not the adapter
- Constructing SecretID with a hashed or encrypted value instead of the raw value — the entire stub contract relies on ID == plaintext value

## Decisions

- **SecretID.ID encodes the plaintext value in the stub adapter** — Allows the full app to function without an external secret store during development; GetAppSecret can reconstruct the value from the ID without any state, keeping the adapter stateless.

## Example: Implementing a real adapter that replaces the stub

```
package vaultadapter

import (
    "context"
    secretentity "github.com/openmeterio/openmeter/openmeter/secret/entity"
    "github.com/openmeterio/openmeter/openmeter/secret"
)

var _ secret.Adapter = (*adapter)(nil)

type adapter struct{ client *vault.Client }

func New(client *vault.Client) secret.Adapter { return &adapter{client: client} }

func (a *adapter) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
// ...
```

<!-- archie:ai-end -->
