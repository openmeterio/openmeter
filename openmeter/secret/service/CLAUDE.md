# service

<!-- archie:ai-start -->

> Concrete implementation of secret.Service: validates inputs then delegates all persistence to the injected secret.Adapter. This is intentionally a thin validation-only layer — no business logic, no cross-domain calls, no Ent dependency.

## Patterns

**Config struct constructor with Validate()** — New(config Config) validates config before constructing the Service. Config.Validate() checks that required fields (Adapter) are non-nil. This is the standard service constructor pattern across this repo. (`func New(config Config) (*Service, error) {
    if err := config.Validate(); err != nil { return nil, err }
    return &Service{adapter: config.Adapter}, nil
}`)
**Double compile-time interface assertion** — service.go asserts `var _ secret.Service = (*Service)(nil)` and secret.go asserts `var _ secret.SecretService = (*Service)(nil)`. Both must be kept in sync when the interfaces change. (`// service.go
var _ secret.Service = (*Service)(nil)
// secret.go
var _ secret.SecretService = (*Service)(nil)`)
**Validate-then-delegate in every method** — Each method calls input.Validate() and wraps any error in models.NewGenericValidationError before calling the adapter. No logic beyond validation should exist in this layer. (`func (s *Service) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) {
    if err := input.Validate(); err != nil {
        return secretentity.SecretID{}, models.NewGenericValidationError(fmt.Errorf("error create app secret: %w", err))
    }
    return s.adapter.CreateAppSecret(ctx, input)
}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config struct, New() constructor. The only field is adapter secret.Adapter. | Config.Validate() only checks Adapter != nil. When new required dependencies are added to Service they must be added to Config and checked in Validate(). |
| `secret.go` | All four method implementations (Create, Update, Get, Delete) with validate-then-delegate pattern. | UpdateAppSecret returns input.SecretID (not zero value) on validation error — intentional to return the original ID. Other methods return zero values on error. |

## Anti-Patterns

- Adding business logic or cross-domain calls — this is a thin validation+delegation layer only
- Calling the adapter without first calling input.Validate() — the stub adapter won't catch structurally invalid inputs
- Injecting *entdb.Client directly into Service — persistence is the adapter's concern, Service only holds secret.Adapter
- Bypassing Config struct and constructing Service{} directly — Validate() in New() is the only nil-safety gate
- Adding new service methods to this file without also adding them to the secret.SecretService interface in openmeter/secret/service.go

## Decisions

- **Service is a thin pass-through with validation only** — The secret domain has no business rules beyond ensuring inputs are structurally valid before reaching the adapter; all complexity lives in the adapter implementation (real or stub).

## Example: Adding a new secret operation following the validate-then-delegate pattern

```
// In openmeter/secret/service.go (interface)
type SecretService interface {
    // ... existing methods ...
    RotateAppSecret(ctx context.Context, input secretentity.RotateAppSecretInput) (secretentity.SecretID, error)
}

// In openmeter/secret/service/secret.go (implementation)
func (s *Service) RotateAppSecret(ctx context.Context, input secretentity.RotateAppSecretInput) (secretentity.SecretID, error) {
    if err := input.Validate(); err != nil {
        return secretentity.SecretID{}, models.NewGenericValidationError(
            fmt.Errorf("error rotate app secret: %w", err),
        )
    }
    return s.adapter.RotateAppSecret(ctx, input)
}
```

<!-- archie:ai-end -->
