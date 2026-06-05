# service

<!-- archie:ai-start -->

> Thin service layer (package costservice) that validates input and delegates to cost.Adapter. Holds no business logic beyond input.Validate() — all cost computation lives in the adapter.

## Patterns

**Config-validated constructor returning (*Service, error)** — New(config Config) validates Config (adapter must be non-nil) before constructing. Service satisfies cost.Service via `var _ cost.Service = (*Service)(nil)`. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Validate input then delegate to adapter** — Each method calls input.Validate() and forwards to the wrapped cost.Adapter with no other logic. (`func (s *Service) QueryFeatureCost(ctx, input) { if err := input.Validate(); err != nil { return nil, err }; return s.adapter.QueryFeatureCost(ctx, input) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines Service, Config, Config.Validate, New, and QueryFeatureCost delegation. | Do not add cost/pricing logic here — keep it a validation-and-delegation passthrough; the package is named costservice, not adapter or cost. |

## Anti-Patterns

- Putting cost computation, meter querying, or LLM resolution in the service instead of the adapter.
- Constructing Service without going through New / Config.Validate (skips the non-nil adapter check).

## Decisions

- **Service is a validation passthrough over the adapter.** — Follows the repo's service/adapter split so the adapter owns I/O and computation while the service owns input validation and the public cost.Service contract.

<!-- archie:ai-end -->
