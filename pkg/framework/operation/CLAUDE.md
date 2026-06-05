# operation

<!-- archie:ai-start -->

> A tiny generic abstraction for RPC-style business logic: Operation[Request, Response] is a func(ctx, Request) (Response, error), with Compose/AsNoResponseOperation combinators and a Middleware[Request,Response] chain so transport layers (HTTP handlers) stay decoupled from business logic.

## Patterns

**Operation as a generic function type** — Business logic is expressed as the function type Operation[Request, Response]; there is no interface to implement — assign a plain func to the type. AsNoResponseOperation adapts error-only (Delete-style) funcs to Operation[Request, any]. (`type Operation[Request any, Response any] func(ctx context.Context, request Request) (Response, error)`)
**Middleware chaining (outermost first)** — Middleware wraps an Operation; Chain composes middlewares so the first argument is the outermost wrapper and request flow traverses them in declaration order (implemented by reverse-applying the others before the outer). (`chain := operation.Chain[Req, Resp](mwAuth, mwLog); op := chain(handler)`)
**Operation composition** — Compose chains two operations (op1's Response becomes op2's Request), short-circuiting on op1 error with a zero Response. (`operation.Compose(getOp, updateOp)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `operation.go` | Core Operation type + AsNoResponseOperation + Compose. | Compose returns the zero value of Response on op1 error; ensure that zero value is a safe sentinel for callers. |
| `middleware.go` | Middleware type + Chain composition helper. | Chain ordering is outermost-first; reversing the argument order silently reverses middleware execution. |

## Anti-Patterns

- Leaking transport concerns (http.Request, response writers) into an Operation — keep them request/response struct only.
- Relying on Compose to surface a partial op1 result on error (it returns the zero Response).

## Decisions

- **Operation is a function type, not an interface** — Lets any plain business function be used directly and composed without boilerplate adapter structs (commented-out interface alternatives remain in operation_test.go).

## Example: Define and middleware-wrap an operation

```
import "github.com/openmeterio/openmeter/pkg/framework/operation"

var create operation.Operation[CreateReq, CreateResp] = func(ctx context.Context, r CreateReq) (CreateResp, error) {
  return svc.Create(ctx, r)
}
chained := operation.Chain[CreateReq, CreateResp](loggingMW)(create)
```

<!-- archie:ai-end -->
