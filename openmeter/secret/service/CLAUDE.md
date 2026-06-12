# service

<!-- archie:ai-start -->

> Service layer for the secret package: a thin pass-through that validates every input via input.Validate() before delegating to the configured secret.Adapter. It is the only layer that enforces validation.

## Patterns

**Validate-then-delegate** — Each method calls input.Validate(), wraps any failure in models.NewGenericValidationError with an operation-specific prefix, then forwards to the matching s.adapter method. (`if err := input.Validate(); err != nil { return secretentity.SecretID{}, models.NewGenericValidationError(fmt.Errorf("error create app secret: %w", err)) }; return s.adapter.CreateAppSecret(ctx, input)`)
**Config-validated constructor** — New(config Config) validates Config (adapter must be non-nil) and returns (*Service, error); the adapter is the sole dependency. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &Service{adapter: config.Adapter}, nil }`)
**Dual interface assertions** — Service asserts both secret.Service (service.go) and secret.SecretService (secret.go) at compile time, mirroring the interface composition in the parent package. (`var _ secret.Service = (*Service)(nil); var _ secret.SecretService = (*Service)(nil)`)
**Package name differs from directory** — Package is secretservice, not service; wired by app/common and the Stripe test setup. (`package secretservice`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config + Config.Validate (adapter non-nil), and New constructor. | adapter is the only field; New returns an error rather than panicking when Config is invalid. |
| `secret.go` | The four CRUD methods, each validating then delegating to the adapter. | UpdateAppSecret returns input.SecretID (not an empty id) on validation failure — keep that shape if extending. |

## Anti-Patterns

- Skipping input.Validate() in a method — the service layer is the validation boundary above the adapter.
- Adding business logic beyond validation+delegation; this service is intentionally a thin wrapper.
- Falling back to slog.Default() or a nil adapter instead of requiring Config.Adapter.

## Decisions

- **Service does nothing but validate and delegate.** — Secret storage logic lives in the swappable adapter; the service guarantees inputs are valid regardless of which adapter (plaintext or real store) is wired.

## Example: Wiring the service over an adapter

```
import (
	secretservice "github.com/openmeterio/openmeter/openmeter/secret/service"
	"github.com/openmeterio/openmeter/openmeter/secret/adapter"
)

svc, err := secretservice.New(secretservice.Config{Adapter: adapter.New()})
if err != nil { return nil, err }
```

<!-- archie:ai-end -->
