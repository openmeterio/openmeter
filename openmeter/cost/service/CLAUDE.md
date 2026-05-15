# service

<!-- archie:ai-start -->

> Thin orchestration layer implementing cost.Service: validates input via input.Validate() then delegates to cost.Adapter.QueryFeatureCost. Contains no business logic — exists solely to enforce layered architecture and provide a stable DI interface.

## Patterns

**Config struct with Validate()** — Constructor accepts a Config struct with a Validate() method; New returns (*Service, error) and rejects invalid config at wiring time rather than at runtime. (`func New(config Config) (*Service, error) { if err := config.Validate(); err != nil { return nil, err } ... }`)
**Interface compliance assertion** — var _ cost.Service = (*Service)(nil) at package level ensures the struct satisfies the interface at compile time. (`var _ cost.Service = (*Service)(nil)`)
**Input validation before delegation** — Every service method calls input.Validate() before forwarding to the adapter, keeping validation centralized at the service boundary. (`if err := input.Validate(); err != nil { return nil, err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Sole file: defines Service struct, Config, New constructor, and QueryFeatureCost — the only cost.Service method. | Do not add business logic or data access here; all computation belongs in the adapter. New methods on cost.Service must follow the same validate-then-delegate pattern. |

## Anti-Patterns

- Adding computation or data access directly in service methods — delegate to cost.Adapter.
- Skipping input.Validate() before calling the adapter.
- Accepting adapter as a bare interface parameter instead of through Config.Adapter with nil-check in Validate().

## Decisions

- **Service is a pass-through with only input validation** — cost domain has a single query operation with no cross-cutting concerns (transactions, hooks, locking); the service layer exists solely to enforce the layered pattern and provide a stable interface for DI.

<!-- archie:ai-end -->
