# service

<!-- archie:ai-start -->

> Concrete implementation of secret.Service that validates inputs then delegates all persistence to the injected secret.Adapter. Contains no business logic beyond validation.

## Patterns

**Config struct constructor with Validate()** — New(config Config) (*Service, error) validates config before constructing. Config.Validate() checks that required fields (Adapter) are non-nil. This is the standard service constructor pattern in this repo. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err }; return &Service{adapter: config.Adapter}, nil }`)
**Double interface assertion** — service.go asserts `var _ secret.Service = (*Service)(nil)` and secret.go asserts `var _ secret.SecretService = (*Service)(nil)` — both must be kept in sync when secret.Service changes. (`var _ secret.Service = (*Service)(nil)`)
**Validate-then-delegate in every method** — Each service method calls input.Validate() and wraps any error in models.NewGenericValidationError before calling the adapter. No logic beyond validation. (`func (s *Service) CreateAppSecret(ctx context.Context, input secretentity.CreateAppSecretInput) (secretentity.SecretID, error) { if err := input.Validate(); err != nil { return secretentity.SecretID{}, models.NewGenericValidationError(fmt.Errorf("error create app secret: %w", err)) }; return s.adapter.CreateAppSecret(ctx, input) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service struct, Config struct, New() constructor. Only field is adapter secret.Adapter. | Config.Validate() only checks Adapter != nil. If new required dependencies are added to Service, add them to Config and Validate() here. |
| `secret.go` | All four method implementations (Create, Update, Get, Delete). Each wraps adapter call with validation. | UpdateAppSecret returns input.SecretID (not zero value) on validation error — intentional to return the original ID. Other methods return zero values on error. |

## Anti-Patterns

- Adding business logic or cross-domain calls to this service — it is a thin validation+delegation layer only
- Calling the adapter without first calling input.Validate() — adapter is a stub and won't catch bad inputs
- Injecting *entdb.Client directly into Service — persistence is the adapter's concern; Service only holds secret.Adapter
- Bypassing the Config struct and constructing Service{} directly — Validate() in New() is the only nil-safety gate

## Decisions

- **Service is a thin pass-through with validation only** — The secret domain has no business rules beyond ensuring inputs are structurally valid before reaching the adapter; all complexity lives in the adapter implementation (real or stub).

<!-- archie:ai-end -->
