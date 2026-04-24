# adapter

<!-- archie:ai-start -->

> Plaintext (no-op) implementation of secret.Adapter — stores secrets as their raw value in the SecretID itself. This is a stub implementation; a production deployment would replace it with a real secret store (Vault, AWS Secrets Manager, etc.).

## Patterns

**Interface compliance assertion** — Every adapter file must declare `var _ secret.Adapter = (*adapter)(nil)` at the top to catch interface drift at compile time. (`var _ secret.Adapter = (*adapter)(nil)`)
**Stateless adapter struct** — The adapter struct holds no fields because secrets are encoded into SecretID. If a real backend is added, inject the client via New() and store it on the struct. (`type adapter struct{}`)
**No Ent dependency in this implementation** — Despite the component description mentioning Ent, this adapter uses only secretentity types — no *entdb.Client. A future real adapter would receive a client via New(). (`func New() secret.Adapter { return &adapter{} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Constructor and compile-time interface assertion. Wire calls New() to obtain a secret.Adapter. | If secret.Adapter gains new methods, the compile-time assertion will fail here first — add the method to secret.go. |
| `secret.go` | All four CRUD methods: CreateAppSecret, UpdateAppSecret, GetAppSecret, DeleteAppSecret. All are no-ops that encode the value into the SecretID. | CreateAppSecret and UpdateAppSecret return secretentity.NewSecretID(input.AppID, input.Value, input.Key) — the ID IS the plaintext value. A real implementation must never do this. |

## Anti-Patterns

- Storing state on the adapter struct without injecting a client through New()
- Returning errors from DeleteAppSecret or CreateAppSecret in the stub — callers expect these to always succeed
- Importing openmeter/ent/db in this package — this stub has no DB dependency; Ent belongs in a future real adapter sub-package
- Adding business logic here — validation belongs in secretservice, not the adapter

## Decisions

- **SecretID encodes the plaintext value as the ID field** — Simplest possible stub that satisfies the Adapter interface without an external secret store; callers reconstruct the value from GetAppSecret by reading input.ID directly.

<!-- archie:ai-end -->
